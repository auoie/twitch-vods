package scraper

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/4kills/go-libdeflate/v2"
	"github.com/Khan/genqlient/graphql"
	"github.com/auoie/goVods/sqlvods"
	"github.com/auoie/goVods/twitchgql"
	"github.com/auoie/goVods/vods"
	"github.com/grafov/m3u8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jdvr/go-again"
)

type VodDataPoint struct {
	ResponseReturnedTime time.Time
	Node                 twitchgql.VodNode
}

type LiveVod struct {
	StreamerId           string
	StreamId             string
	StartTime            time.Time
	StreamerLoginAtStart string
	MaxViews             int
	LastUpdated          time.Time

	// At the moment, I don't do anything with this. Storing it would use up a lot of space.
	// Should I serialize it with something like Protobuf and then GZIP it?
	// Should I use an OLAP database (Clickhouse or TimeScale)?
	// TimeSeries []VodDataPoint
}

func (vod *LiveVod) GetVideoData() *vods.VideoData {
	return &vods.VideoData{StreamerName: vod.StreamerLoginAtStart, VideoId: vod.StreamId, Time: vod.StartTime}
}

type VodResult struct {
	Vod                *LiveVod
	HlsBytes           []byte
	HlsBytesFound      bool
	RequestInitiated   time.Time
	HlsDomain          sql.NullString
	SeekPreviewsDomain sql.NullString
	Public             sql.NullBool
	SubOnly            sql.NullBool
}

func edgeNodesMatchingAndNonEmpty(
	a []twitchgql.GetStreamsStreamsStreamConnectionEdgesStreamEdge,
	b []twitchgql.GetStreamsStreamsStreamConnectionEdgesStreamEdge,
) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return false
	}
	for index := range a {
		if a[index].Node.Id != b[index].Node.Id {
			return false
		}
	}
	return true
}

type fetchTwitchGqlForeverParams struct {
	ctx                       context.Context
	initialLiveVodQueue       *liveVodsPriorityQueue
	client                    graphql.Client
	twitchGqlRequestTimeLimit time.Duration
	twitchGqlFetcherDelay     time.Duration
	cursorResetThreshold      time.Duration
	oldVodEvictionThreshold   time.Duration
	oldVodsCh                 chan []*LiveVod
	minViewerCountToObserve   int
	minViewerCountToRecord    int
	queries                   *sqlvods.Queries
	numStreamsPerRequest      int
	cursorFactor              float64
	done                      chan struct{}
}

func twitchGqlResponseToSqlParams(
	streams []twitchgql.GetStreamsStreamsStreamConnectionEdgesStreamEdge,
	responseReturnedTime time.Time,
) sqlvods.UpsertManyStreamsParams {
	result := sqlvods.UpsertManyStreamsParams{}
	for _, stream := range streams {
		node := stream.Node
		result.LastUpdatedAtArr = append(result.LastUpdatedAtArr, responseReturnedTime)
		result.MaxViewsArr = append(result.MaxViewsArr, int64(node.ViewersCount))
		result.StartTimeArr = append(result.StartTimeArr, node.CreatedAt)
		result.StreamIDArr = append(result.StreamIDArr, node.Id)
		result.StreamerIDArr = append(result.StreamerIDArr, node.Broadcaster.Id)
		result.StreamerLoginAtStartArr = append(result.StreamerLoginAtStartArr, node.Broadcaster.Login)
		result.GameNameAtStartArr = append(result.GameNameAtStartArr, node.Game.Name)
		result.LanguageAtStartArr = append(result.LanguageAtStartArr, string(node.Broadcaster.BroadcastSettings.Language))
		result.TitleAtStartArr = append(result.TitleAtStartArr, node.Broadcaster.BroadcastSettings.Title)
	}
	return result
}

func fetchTwitchGqlForever(params fetchTwitchGqlForeverParams) {
	log.Println("Inside fetchTwitchGqlForever...")
	log.Println(fmt.Sprint("Fetcher delay: ", params.twitchGqlFetcherDelay))
	liveVodQueue := params.initialLiveVodQueue
	twitchGqlTicker := time.NewTicker(params.twitchGqlFetcherDelay)
	defer twitchGqlTicker.Stop()
	cursor := ""
	resetCursorTimeout := time.Now().Add(params.cursorResetThreshold)
	debugIndex := -1
	resetCursor := func() {
		log.Println(fmt.Sprint("Resetting cursor on debug index: ", debugIndex))
		debugIndex = -1
		cursor = ""
		resetCursorTimeout = time.Now().Add(params.cursorResetThreshold)
	}
	log.Println("Starting twitchgql infinite for loop.")
	prevEdges := []twitchgql.GetStreamsStreamsStreamConnectionEdgesStreamEdge{}
	debugMod := 10
	for {
		select {
		case <-params.ctx.Done():
			return
		case <-twitchGqlTicker.C:
		}
		debugIndex++
		debugQueueSizeStart := liveVodQueue.Size()
		if time.Now().After(resetCursorTimeout) {
			log.Println(fmt.Sprint("Reset cursor because we've been fetching for: ", params.cursorResetThreshold))
			resetCursor()
		}
		requestCtx, requestCancel := context.WithTimeout(params.ctx, params.twitchGqlRequestTimeLimit)
		streams, err := twitchgql.GetStreams(requestCtx, params.client, params.numStreamsPerRequest, cursor)
		responseReturnedTime := time.Now()
		requestCancel()
		if err != nil {
			resetCursor()
			log.Println(fmt.Sprint("Twitch graphql client reported an error: ", err))
			continue
		}
		edges := streams.Streams.Edges
		if len(edges) == 0 {
			log.Println("edges has length 0")
			resetCursor()
		} else if !streams.Streams.PageInfo.HasNextPage {
			log.Println("streams.Streams.PageInfo does not have next page")
			resetCursor()
		} else {
			if debugIndex%debugMod == 0 {
				log.Println()
				log.Println("First and last stream")
				log.Println(edges[0])
				log.Println(edges[len(edges)-1])
				log.Println(fmt.Sprint("Live VOD queue size start: ", debugQueueSizeStart))
			}
			if edgeNodesMatchingAndNonEmpty(prevEdges, edges) {
				log.Println("prevEdges and edges node ids:")
				log.Println("prevEdges: ", prevEdges)
				log.Println("edges: ", edges)
				cursor = edges[len(edges)-1].Cursor
			} else {
				cursor = edges[int(params.cursorFactor*float64(len(edges)))].Cursor
			}
		}
		prevEdges = edges
		oldVods := []*LiveVod{}
		allVodsLessThanMinViewerCount := true
		highViewEdges := []twitchgql.GetStreamsStreamsStreamConnectionEdgesStreamEdge{}
		for _, edge := range edges {
			if edge.Node.ViewersCount < params.minViewerCountToObserve {
				continue
			}
			highViewEdges = append(highViewEdges, edge)
			allVodsLessThanMinViewerCount = false
			evictedVod, err := liveVodQueue.UpsertVod(
				responseReturnedTime,
				VodDataPoint{Node: &edge.Node, ResponseReturnedTime: responseReturnedTime},
			)
			if err != nil {
				continue
			}
			log.Println("Streamer restarted stream: ", evictedVod.StreamerLoginAtStart)
			oldVods = append(oldVods, evictedVod)
		}
		requestCtx, requestCancel = context.WithTimeout(params.ctx, params.twitchGqlRequestTimeLimit)
		err = params.queries.UpsertManyStreams(requestCtx, twitchGqlResponseToSqlParams(highViewEdges, responseReturnedTime))
		requestCancel()
		if err != nil {
			log.Println(fmt.Sprint("Upserting streams to streams table failed: ", err))
			break
		}
		if debugIndex%debugMod == 0 {
			log.Println(fmt.Sprint("Live VOD queue size after upserts: ", liveVodQueue.Size()))
		}
		debugNumRestartedStreams := len(oldVods)
		if debugNumRestartedStreams > 0 {
			log.Println(fmt.Sprint("num restarted streams: ", debugNumRestartedStreams))
		}
		if allVodsLessThanMinViewerCount {
			log.Println("All vods less than min viewer count")
			resetCursor()
		}
		oldestTimeAllowed := responseReturnedTime.Add(-params.oldVodEvictionThreshold)
		for {
			stalestVod, err := liveVodQueue.GetStalestStream()
			if err != nil {
				break
			}
			if stalestVod.LastUpdated.After(oldestTimeAllowed) {
				break
			}
			liveVodQueue.RemoveVod(stalestVod)
			if stalestVod.MaxViews >= params.minViewerCountToRecord {
				oldVods = append(oldVods, stalestVod)
			}
		}
		if debugIndex%debugMod == 0 {
			log.Println(fmt.Sprint("Live VOD queue size after removing stale VODS: ", liveVodQueue.Size()))
		}
		debugNumStaleVods := len(oldVods) - debugNumRestartedStreams
		if debugNumStaleVods > 0 {
			log.Println(fmt.Sprint("num stale vods: ", debugNumStaleVods))
		}
		select {
		case <-params.ctx.Done():
			return
		case params.oldVodsCh <- oldVods:
		}
	}
	select {
	case params.done <- struct{}{}:
	case <-params.ctx.Done():
	}
}

type processOldVodJobsParams struct {
	ctx                 context.Context
	oldVodsCh           chan []*LiveVod
	oldVodJobsCh        chan *LiveVod
	maxOldVodsQueueSize int
}

func processOldVodJobs(params processOldVodJobsParams) {
	oldVodsOrderedByViews := CreateNewOldVodQueue()
	getJobsCh := func() chan *LiveVod {
		if oldVodsOrderedByViews.Size() == 0 {
			return nil
		}
		return params.oldVodJobsCh
	}
	getNextInQueue := func() *LiveVod {
		oldVod, _ := oldVodsOrderedByViews.PopHighViewCount()
		return oldVod
	}
	debugCount := -1
	for {
		debugCount++
		if debugCount%10 == 0 {
			log.Println(fmt.Sprint("oldVodsOrderedByViews size: ", oldVodsOrderedByViews.Size()))
		}
		select {
		case <-params.ctx.Done():
			return
		case oldVods := <-params.oldVodsCh:
			for _, oldVod := range oldVods {
				oldVodsOrderedByViews.Put(oldVod)
				if oldVodsOrderedByViews.Size() > params.maxOldVodsQueueSize {
					oldVodsOrderedByViews.PopLowViewCount()
				}
			}
		case getJobsCh() <- getNextInQueue():
		}
	}
}

func getFirstValidDwpResponse(ctx context.Context, videoData *vods.VideoData) (*vods.ValidDwpResponse, error) {
	dwp, err := vods.GetFirstValidDwp(ctx, videoData.GetDomainWithPathsList(vods.DOMAINS, 1))
	if err != nil {
		return nil, err
	}
	return dwp, nil
}

func getCleanedMediaPlaylistBytes(dwp *vods.ValidDwpResponse) (*m3u8.MediaPlaylist, error) {
	mediapl, err := vods.DecodeMediaPlaylistFilterNilSegments(dwp.Body, true)
	if err != nil {
		return nil, err
	}
	vods.MuteMediaSegments(mediapl)
	dwp.Dwp.MakePathsExplicit(mediapl)
	return mediapl, nil
}

func getCompressedBytes(bytes []byte, compressor *libdeflate.Compressor) ([]byte, error) {
	compressedBytes := make([]byte, len(bytes))
	n, _, err := compressor.Compress(bytes, compressedBytes, libdeflate.ModeGzip)
	if err != nil {
		return nil, err
	}
	compressedBytes = compressedBytes[:n]
	return compressedBytes, nil
}

// Find the .m3u8 for a video and return the compressed bytes.
func getVodCompressedBytes(ctx context.Context, videoData *vods.VideoData, compressor *libdeflate.Compressor) ([]byte, *vods.DomainWithPath, error) {
	dwp, err := getFirstValidDwpResponse(ctx, videoData)
	if err != nil {
		dwp, err = getFirstValidDwpResponse(ctx, &vods.VideoData{
			StreamerName: videoData.StreamerName,
			VideoId:      videoData.VideoId,
			Time:         videoData.Time.Add(-time.Second),
		})
	}
	if err != nil {
		log.Println(fmt.Sprint("Link was not found for ", videoData.StreamerName, " because: ", err))
		return nil, nil, err
	}
	mediapl, err := getCleanedMediaPlaylistBytes(dwp)
	if err != nil {
		log.Println(fmt.Sprint("Removing nil segments for ", videoData.StreamerName, " failed because: ", err))
		log.Println(mediapl.String())
		return nil, nil, err
	}
	compressedBytes, err := getCompressedBytes(mediapl.Encode().Bytes(), compressor)
	if err != nil {
		log.Println(fmt.Sprint("Compressing failed for ", videoData.StreamerName, " because: ", err))
		return nil, nil, err
	}
	return compressedBytes, dwp.Dwp, nil
}

type videoStatus struct {
	seekPreviewsDomain sql.NullString
	public             bool
	subOnly            bool
}

func getVideoStatus(ctx context.Context, client graphql.Client, streamerId string, streamId string) (videoStatus, error) {
	response, err := twitchgql.GetUserData(ctx, client, streamerId)
	if err != nil {
		return videoStatus{}, err
	}
	videos := response.User.Videos.Edges
	public := false
	var seekPreviewsDomain sql.NullString
	for _, cur := range videos {
		seekPreviewsUrl := cur.Node.SeekPreviewsURL
		dwp, err := vods.UrlToDomainWithPath(seekPreviewsUrl)
		if err != nil {
			continue
		}
		if dwp.Path.VideoData.VideoId == streamId {
			public = true
			seekPreviewsDomain = sql.NullString{String: dwp.Domain, Valid: true}
			break
		}
	}
	subProducts := response.User.SubscriptionProducts
	subOnly := len(subProducts) > 0 && subProducts[0].HasSubonlyVideoArchive
	return videoStatus{
		public:             public,
		subOnly:            subOnly,
		seekPreviewsDomain: seekPreviewsDomain,
	}, nil
}

type hlsWorkerFetchCompressSendParams struct {
	ctx              context.Context
	client           graphql.Client
	oldVodJobsCh     chan *LiveVod
	hlsFetcherDelay  time.Duration
	compressor       *libdeflate.Compressor
	resultsCh        chan *VodResult
	requestTimeLimit time.Duration
}

func hlsWorkerFetchCompressSend(params hlsWorkerFetchCompressSendParams) {
	hlsFetcherTicker := time.NewTicker(params.hlsFetcherDelay)
	defer params.compressor.Close()
	for {
		select {
		case <-params.ctx.Done():
			return
		case <-hlsFetcherTicker.C:
		}
		var oldVod *LiveVod
		select {
		case <-params.ctx.Done():
			return
		case oldVod = <-params.oldVodJobsCh:
		}
		requestCtx, cancel := context.WithTimeout(params.ctx, params.requestTimeLimit)
		requestInitiated := time.Now()
		bytes, dwp, err := getVodCompressedBytes(requestCtx, oldVod.GetVideoData(), params.compressor)
		cancel()
		var result *VodResult
		if err != nil {
			result = &VodResult{
				Vod:              oldVod,
				HlsBytes:         nil,
				HlsBytesFound:    false,
				RequestInitiated: requestInitiated,
				HlsDomain: sql.NullString{
					String: "",
					Valid:  false,
				},
			}
		} else {
			result = &VodResult{
				Vod:              oldVod,
				HlsBytes:         bytes,
				HlsBytesFound:    true,
				RequestInitiated: requestInitiated,
				HlsDomain: sql.NullString{
					String: dwp.Domain,
					Valid:  true,
				},
			}
		}
		requestCtx, cancel = context.WithTimeout(params.ctx, params.requestTimeLimit)
		videoMeta, err := getVideoStatus(requestCtx, params.client, oldVod.StreamerId, oldVod.StreamId)
		cancel()
		if err == nil {
			result.Public = sql.NullBool{Bool: videoMeta.public, Valid: true}
			result.SeekPreviewsDomain = videoMeta.seekPreviewsDomain
			result.SubOnly = sql.NullBool{Bool: videoMeta.subOnly, Valid: true}
		}
		select {
		case <-params.ctx.Done():
			return
		case params.resultsCh <- result:
		}
	}
}

type ScrapeTwitchLiveVodsWithGqlApiParams struct {
	RunScraperParams
	// initial live vod queue fetched from database
	InitialLiveVodQueue *liveVodsPriorityQueue
	// sqlc queries instance
	Queries *sqlvods.Queries
}

// This function scares me.
// This function scrapes the Twitch Graphql API and fetches .m3u8 files for streams that finish.
// It doesn't exit if a Twitch Graphql API request fails.
// Instead, it resets the cursor and starts over.
// It stores the results in a database with concurrent updates, so you should use a queries struct that is safe for that.
// If any database query or modification returns an error, the function finishes and cleans up all resources.
func ScrapeTwitchLiveVodsWithGqlApi(ctx context.Context, params ScrapeTwitchLiveVodsWithGqlApiParams) error {
	log.Println("Starting scraping...")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	client := twitchgql.NewTwitchGqlClient()
	oldVodsCh := make(chan []*LiveVod)
	oldVodJobsCh := make(chan *LiveVod)
	resultsCh := make(chan *VodResult)
	done := make(chan struct{})
	log.Println("Made twitchgql client and channels.")
	go fetchTwitchGqlForever(
		fetchTwitchGqlForeverParams{
			ctx:                       ctx,
			client:                    client,
			initialLiveVodQueue:       params.InitialLiveVodQueue,
			twitchGqlRequestTimeLimit: params.RequestTimeLimit,
			twitchGqlFetcherDelay:     params.TwitchGqlFetcherDelay,
			cursorResetThreshold:      params.CursorResetThreshold,
			oldVodEvictionThreshold:   params.OldVodEvictionThreshold,
			oldVodsCh:                 oldVodsCh,
			minViewerCountToObserve:   params.MinViewerCountToObserve,
			minViewerCountToRecord:    params.MinViewerCountToRecord,
			queries:                   params.Queries,
			numStreamsPerRequest:      params.NumStreamsPerRequest,
			cursorFactor:              params.CursorFactor,
			done:                      done,
		},
	)
	go processOldVodJobs(processOldVodJobsParams{
		ctx:                 ctx,
		oldVodsCh:           oldVodsCh,
		oldVodJobsCh:        oldVodJobsCh,
		maxOldVodsQueueSize: params.MaxOldVodsQueueSize,
	})
	for i := 0; i < params.NumHlsFetchers; i++ {
		compressor, err := libdeflate.NewCompressorLevel(params.LibdeflateCompressionLevel)
		if err != nil {
			return err
		}
		go hlsWorkerFetchCompressSend(hlsWorkerFetchCompressSendParams{
			ctx:              ctx,
			client:           client,
			oldVodJobsCh:     oldVodJobsCh,
			hlsFetcherDelay:  params.HlsFetcherDelay,
			compressor:       &compressor,
			resultsCh:        resultsCh,
			requestTimeLimit: params.RequestTimeLimit,
		})
	}
	go func() {
		for {
			result := <-resultsCh
			log.Println("Result was logged:")
			log.Println(*result.Vod)
			log.Println(fmt.Sprint("Gzipped size: ", len(result.HlsBytes)))
			upsertRecordingParams := sqlvods.UpdateRecordingParams{
				RecordingFetchedAt: sql.NullTime{Time: result.RequestInitiated, Valid: true},
				GzippedBytes:       result.HlsBytes,
				StreamID:           result.Vod.StreamId,
				BytesFound:         sql.NullBool{Bool: result.HlsBytesFound, Valid: true},
				HlsDomain:          result.HlsDomain,
				SeekPreviewsDomain: result.SeekPreviewsDomain,
				Public:             result.Public,
				SubOnly:            result.SubOnly,
			}
			err := params.Queries.UpdateRecording(ctx, upsertRecordingParams)
			if err != nil {
				log.Println(fmt.Sprint("upserting recording failed: ", err))
				break
			}
		}
		select {
		case <-ctx.Done():
		case done <- struct{}{}:
		}
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
	return nil
}

type RunScraperParams struct {
	// In any interval of this length, the api will be called at most twice and on average once.
	TwitchGqlFetcherDelay time.Duration
	// Time limit for .m3u8 and Twitch GQL requests. If this is exceeded in the TwithGQL loop, the for-loop continues. TODO: I should fix this.
	RequestTimeLimit time.Duration
	// If a VOD in the queue of live VODs is older than this, it is moved to the old VODs queue.
	OldVodEvictionThreshold time.Duration
	// The queue of old VODs for fetching .m3u8 will never exceed this size. The VODs with the lowest view counts are evicted.
	MaxOldVodsQueueSize int
	// This is the number of goroutines fetching the .m3u8 files and compressing them.
	NumHlsFetchers int
	// In any interval of this length, at most two .m3u8 files will be processed and on average once.
	HlsFetcherDelay time.Duration
	// If this amount of time passes since the last time the cursor was reset, the cursor will be reset.
	CursorResetThreshold time.Duration
	// This is the libdeflate compression level. The highest is 1 and the lowest is 12.
	// It seems best when it's 1. The level of compression is good enough and it is fastest.
	LibdeflateCompressionLevel int
	// The queue of live VODs includes a VOD iff a VOD has at least this number of viewers.
	MinViewerCountToObserve int
	// The queue of old VODs includes a VOD iff a VOD has at least this number of viewers.
	// If a stream is observed to hvae stopped and then restarted, the stream is still recorded.
	MinViewerCountToRecord int
	// Num streams per request (must be between 1 and 30 inclusive)
	NumStreamsPerRequest int
	// Cursor at index CursorFactor * len(edges) is used. So it must satisfy 0 <= CursorFactor < 1 to not panic.
	CursorFactor float64
}

// databaseUrl is the postgres database to connect to.
// evictionRatio should be at least 1. We select live vods that were updated at most evictionRatio * oldVodEvictionThreshold ago before the newest live vod.
// params are the parameters twitch graphql scraper.
func RunScraper(ctx context.Context, databaseUrl string, evictionRatio float64, params RunScraperParams) error {
	type tInitialState struct {
		conn         *pgxpool.Pool
		queries      *sqlvods.Queries
		liveVodQueue *liveVodsPriorityQueue
	}
	getInitialState := func(ctx context.Context) (*tInitialState, error) {
		conn, err := pgxpool.Connect(ctx, databaseUrl)
		if err != nil {
			log.Println(fmt.Sprint("failed to connect to ", databaseUrl, ": ", err))
			return nil, err
		}
		err = conn.Ping(ctx)
		if err != nil {
			log.Println(fmt.Sprint("failed to ping ", databaseUrl, ": ", err))
			conn.Close()
			return nil, err
		}
		queries := sqlvods.New(conn)
		latestStreams, err := queries.GetLatestStreams(ctx, 1)
		if err != nil {
			log.Println(fmt.Sprint("failed to get latest streams from ", databaseUrl, ": ", err))
			conn.Close()
			return nil, err
		}
		liveVodQueue := CreateNewLiveVodsPriorityQueue()
		if len(latestStreams) == 0 {
			log.Println("there are 0 live vods")
			return &tInitialState{conn: conn, queries: queries, liveVodQueue: liveVodQueue}, nil
		}
		latestStream := latestStreams[0]
		lastTimeAllowed := latestStream.LastUpdatedAt.Add(-time.Duration(float64(params.OldVodEvictionThreshold) * evictionRatio))
		latestLiveStreams, err := queries.GetLatestLiveStreams(ctx, lastTimeAllowed)
		if err != nil {
			conn.Close()
			return nil, err
		}
		for _, liveStream := range latestLiveStreams {
			liveVodQueue.UpsertLiveVod(&LiveVod{
				StreamerId:           liveStream.StreamerID,
				StreamId:             liveStream.StreamID,
				StartTime:            liveStream.StartTime,
				StreamerLoginAtStart: liveStream.StreamerLoginAtStart,
				MaxViews:             int(liveStream.MaxViews),
				LastUpdated:          liveStream.LastUpdatedAt,
			})
		}
		return &tInitialState{conn: conn, queries: queries, liveVodQueue: liveVodQueue}, nil
	}
	initialState, err := again.Retry(ctx, getInitialState)
	if err != nil {
		log.Println(fmt.Sprint("failed to get initial state: ", err))
		return err
	}
	defer initialState.conn.Close()
	log.Println(fmt.Sprint("entries in liveVodsQueue: ", initialState.liveVodQueue.Size()))
	return ScrapeTwitchLiveVodsWithGqlApi(
		ctx,
		ScrapeTwitchLiveVodsWithGqlApiParams{
			RunScraperParams:    params,
			InitialLiveVodQueue: initialState.liveVodQueue,
			Queries:             initialState.queries,
		},
	)
}
