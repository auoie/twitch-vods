package scraper

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/4kills/go-libdeflate/v2"
	"github.com/auoie/goVods/vods"
	"github.com/auoie/twitch-vods/sqlvods"
	"github.com/grafov/m3u8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jdvr/go-again"
	"github.com/nicklaw5/helix"
)

type VodDataPoint struct {
	ResponseReturnedTimeUnix int64
	Node                     *helix.Stream
}

type LiveVod struct {
	StreamerId           string
	StreamId             string
	StartTimeUnix        int64
	StreamerLoginAtStart string
	GameIdAtStart        string
	MaxViews             int
	LastUpdatedUnix      int64 // time twitchgql request completed
	LastInteractionUnix  int64 // last time interacted with (e.g. time twitchgql request completed or time sql fetch completed)

	// At the moment, I don't do anything with this. Storing it would use up a lot of space.
	// Should I serialize it with something like Protobuf and then GZIP it?
	// Should I use an OLAP database (Clickhouse or TimeScale)?
	// TimeSeries []VodDataPoint
}

func (vod *LiveVod) GetVideoData() *vods.VideoData {
	return &vods.VideoData{StreamerName: vod.StreamerLoginAtStart, VideoId: vod.StreamId, Time: time.Unix(vod.StartTimeUnix, 0).UTC()}
}

type VodResult struct {
	Vod                *LiveVod
	HlsBytes           []byte
	HlsBytesFound      bool
	RequestInitiated   time.Time
	HlsDomain          sql.NullString
	Public             sql.NullBool
	ProfileImageUrl    sql.NullString
	BoxArtUrl          sql.NullString
	HlsDurationSeconds sql.NullFloat64
}

func edgeNodesMatchingAndNonEmpty(
	a []helix.Stream,
	b []helix.Stream,
) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return false
	}
	for index := range a {
		if a[index].ID != b[index].ID {
			return false
		}
	}
	return true
}

type fetchTwitchHelixForeverParams struct {
	ctx                      context.Context
	initialWaitVodQueue      *waitVodsPriorityQueue
	twitchHelixClient        *helix.Client
	sqlRequestTimeLimit      time.Duration
	twitchHelixFetcherDelay  time.Duration
	cursorResetThreshold     time.Duration
	liveVodEvictionThreshold time.Duration
	waitVodEvictionThreshold time.Duration
	oldVodsCh                chan []*LiveVod
	minViewerCountToObserve  int
	minViewerCountToRecord   int
	queries                  *sqlvods.Queries
	numStreamsPerRequest     int
	oldVodsDelete            time.Duration
	done                     chan struct{}
}

func twitchGqlResponseUpsertStreamsParams(
	streams []*helix.Stream,
	responseReturnedTime time.Time,
) sqlvods.UpsertManyStreamsParams {
	result := sqlvods.UpsertManyStreamsParams{}
	for _, node := range streams {
		result.LastUpdatedAtArr = append(result.LastUpdatedAtArr, responseReturnedTime)
		result.MaxViewsArr = append(result.MaxViewsArr, int64(node.ViewerCount))
		result.StartTimeArr = append(result.StartTimeArr, node.StartedAt.UTC())
		result.StreamIDArr = append(result.StreamIDArr, node.ID)
		result.StreamerIDArr = append(result.StreamerIDArr, node.UserID)
		result.StreamerLoginAtStartArr = append(result.StreamerLoginAtStartArr, node.UserLogin)
		result.GameNameAtStartArr = append(result.GameNameAtStartArr, node.GameName)
		result.LanguageAtStartArr = append(result.LanguageAtStartArr, string(node.Language))
		result.TitleAtStartArr = append(result.TitleAtStartArr, node.Title)
		result.GameIDAtStartArr = append(result.GameIDAtStartArr, node.GameID)
		result.IsMatureAtStartArr = append(result.IsMatureAtStartArr, node.IsMature)
		result.LastUpdatedMinusStartTimeSecondsArr = append(result.LastUpdatedMinusStartTimeSecondsArr, responseReturnedTime.Sub(node.StartedAt).Seconds())
	}
	return result
}

func resetAppAccessToken(client *helix.Client) error {
	appAccessToken, err := client.RequestAppAccessToken([]string{})
	if err != nil {
		return err
	}
	client.SetAppAccessToken(appAccessToken.Data.AccessToken)
	return nil
}

func twitchGqlResponseUpsertStreamersParams(
	streams []*helix.Stream,
) sqlvods.UpsertManyStreamersParams {
	result := sqlvods.UpsertManyStreamersParams{}
	for _, node := range streams {
		result.StartTimeArr = append(result.StartTimeArr, node.StartedAt.UTC())
		result.StreamerIDArr = append(result.StreamerIDArr, node.UserID)
		result.StreamerLoginAtStartArr = append(result.StreamerLoginAtStartArr, node.UserLogin)
	}
	return result
}

func retryOnError[T any](doer func() (T, error)) (T, error) {
	res, err := doer()
	if err != nil {
		log.Println(fmt.Sprint("retrying on error: ", err))
		return doer()
	}
	return res, err
}

func fetchTwitchHelixForever(params fetchTwitchHelixForeverParams) {
	log.Println("Inside fetchTwitchGqlForever...")
	log.Println(fmt.Sprint("Fetcher delay: ", params.twitchHelixFetcherDelay))
	liveVodQueue := CreateNewLiveVodsPriorityQueue()
	waitVodQueue := params.initialWaitVodQueue
	twitchGqlTicker := time.NewTicker(params.twitchHelixFetcherDelay)
	defer twitchGqlTicker.Stop()
	cursor := ""
	resetCursorTimeout := time.Now().UTC().Add(params.cursorResetThreshold)
	debugIndex := -1
	resetCursor := func() {
		log.Println(fmt.Sprint("Resetting cursor on debug index: ", debugIndex))
		debugIndex = -1
		cursor = ""
		resetCursorTimeout = time.Now().UTC().Add(params.cursorResetThreshold)
		_, err := retryOnError(func() (struct{}, error) {
			return struct{}{}, resetAppAccessToken(params.twitchHelixClient)
		})
		if err != nil {
			log.Println(err)
		}
	}
	log.Println("Starting twitchgql infinite for loop.")
	prevEdges := []helix.Stream{}
	debugMod := 10
	for {
		// Wait until done are next ticker
		select {
		case <-params.ctx.Done():
			return
		case <-twitchGqlTicker.C:
		}
		debugIndex++
		debugQueueSizeStart := liveVodQueue.Size()
		// Reset cursor if fetching for long time
		if time.Now().UTC().After(resetCursorTimeout) {
			log.Println(fmt.Sprint("Reset cursor because we've been fetching for: ", params.cursorResetThreshold))
			resetCursor()
		}
		// Request live streams
		streams, err := retryOnError(func() (*helix.StreamsResponse, error) {
			return params.twitchHelixClient.GetStreams(&helix.StreamsParams{
				After: cursor,
				First: params.numStreamsPerRequest,
			})
		})
		responseReturnedTime := time.Now().UTC()
		responseReturnedTimeUnix := responseReturnedTime.Unix()
		// If request failed, reset cursor
		if err != nil {
			resetCursor()
			log.Println(fmt.Sprint("Twitch graphql client reported an error: ", err))
			continue
		}
		// Get the next cursor and if there are no streams or no next page, reset cursor
		edges := streams.Data.Streams
		if len(edges) == 0 {
			log.Println("edges has length 0")
			resetCursor()
		} else if streams.Data.Pagination.Cursor == "" {
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
				cursor = streams.Data.Pagination.Cursor
			} else {
				cursor = streams.Data.Pagination.Cursor
			}
		}
		prevEdges = edges
		// Remove repeats from wait queue, upsert the min observe view count vods, and insert evicted vods into wait queue
		numRemoved := 0
		oldVods := []*LiveVod{}
		allVodsLessThanMinViewerCount := true
		highViewNodes := []*helix.Stream{}
		for _, edge := range edges {
			nodeClone := edge
			node := &nodeClone
			if node.ViewerCount < params.minViewerCountToObserve {
				continue
			}
			highViewNodes = append(highViewNodes, node)
			allVodsLessThanMinViewerCount = false
			waitVod, err := waitVodQueue.GetByStreamIdStartTime(node.ID, node.StartedAt.UTC().Unix())
			if err == nil {
				log.Println(fmt.Sprint("Removing vod from wait queue: ", *waitVod))
				waitVodQueue.RemoveVod(waitVod)
				liveVodQueue.UpsertLiveVod(waitVod)
			}
			evictedVod, err := liveVodQueue.UpsertVod(
				VodDataPoint{Node: node, ResponseReturnedTimeUnix: responseReturnedTimeUnix},
			)
			if err != nil {
				continue
			}
			log.Println("Streamer restarted stream: ", evictedVod.StreamerLoginAtStart)
			evictedVod.LastInteractionUnix = responseReturnedTimeUnix
			waitVodQueue.Put(evictedVod)
			numRemoved++
		}
		if debugIndex%debugMod == 0 {
			log.Println(fmt.Sprint("Live VOD queue size after upserts: ", liveVodQueue.Size()))
		}
		debugNumRestartedStreams := numRemoved
		if debugNumRestartedStreams > 0 {
			log.Println(fmt.Sprint("num restarted streams: ", debugNumRestartedStreams))
		}
		// If all vods are below minimum observe view count, reset cursor
		if allVodsLessThanMinViewerCount {
			log.Println("All vods less than min viewer count")
			resetCursor()
		}
		// Evict vods with old last updated time and add vods to wait queue
		oldestUpdateTimeAllowedUnix := responseReturnedTime.Add(-params.liveVodEvictionThreshold).Unix()
		for {
			stalestVod, err := liveVodQueue.GetStalestStream()
			if err != nil {
				break
			}
			if stalestVod.LastUpdatedUnix > oldestUpdateTimeAllowedUnix {
				break
			}
			liveVodQueue.RemoveVod(stalestVod)
			stalestVod.LastInteractionUnix = responseReturnedTimeUnix
			waitVodQueue.Put(stalestVod)
			numRemoved++
		}
		if debugIndex%debugMod == 0 {
			log.Println(fmt.Sprint("Live VOD queue size after removing stale VODS: ", liveVodQueue.Size()))
			log.Println(fmt.Sprint("Wait VOD queue size: ", waitVodQueue.Size()))
		}
		debugNumStaleVods := numRemoved - debugNumRestartedStreams
		if debugNumStaleVods > 0 {
			log.Println(fmt.Sprint("num stale vods: ", debugNumStaleVods))
		}
		// The queries to delete the old streams and upsert the new streams should be combined into a single transaction
		requestCtx, requestCancel := context.WithTimeout(params.ctx, params.sqlRequestTimeLimit)
		err = params.queries.DeleteOldStreams(requestCtx, responseReturnedTime.Add(-params.oldVodsDelete))
		requestCancel()
		if err != nil {
			log.Println(fmt.Sprint("deleting old streams failed: ", err))
			break
		}
		requestCtx, requestCancel = context.WithTimeout(params.ctx, params.sqlRequestTimeLimit)
		err = params.queries.UpsertManyStreams(requestCtx, twitchGqlResponseUpsertStreamsParams(highViewNodes, responseReturnedTime))
		requestCancel()
		if err != nil {
			log.Println(fmt.Sprint("Upserting streams to streams table failed: ", err))
			break
		}
		requestCtx, requestCancel = context.WithTimeout(params.ctx, params.sqlRequestTimeLimit)
		err = params.queries.DeleteOldStreamers(requestCtx, responseReturnedTime.Add(-params.oldVodsDelete))
		requestCancel()
		if err != nil {
			log.Println(fmt.Sprint("deleting old streamers failed: ", err))
			break
		}
		requestCtx, requestCancel = context.WithTimeout(params.ctx, params.sqlRequestTimeLimit)
		err = params.queries.UpsertManyStreamers(requestCtx, twitchGqlResponseUpsertStreamersParams(highViewNodes))
		requestCancel()
		if err != nil {
			log.Println(fmt.Sprint("Upserting streamers to streamers table failed: ", err))
			break
		}
		// Evict vods with old last interaction time from wait vods queue and record iff at least record view count
		oldestInteractionTimeAllowedUnix := responseReturnedTime.Add(-params.waitVodEvictionThreshold).Unix()
		for {
			stalestVod, err := waitVodQueue.GetStalestStream()
			if err != nil {
				break
			}
			if stalestVod.LastInteractionUnix > oldestInteractionTimeAllowedUnix {
				break
			}
			waitVodQueue.RemoveVod(stalestVod)
			if stalestVod.MaxViews >= params.minViewerCountToRecord {
				oldVods = append(oldVods, stalestVod)
			}
		}
		// Add old vods to old vods queue
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
		oldVod, _ := oldVodsOrderedByViews.GetLowViewCount()
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
			oldVodsOrderedByViews.PopLowViewCount()
		}
	}
}

func getFirstValidDwpResponse(ctx context.Context, videoData *vods.VideoData, toUnix bool, client *http.Client) (*vods.ValidDwpResponse, error) {
	dwp, err := vods.GetFirstValidDwp(ctx, videoData.GetDomainWithPathsList(vods.DOMAINS, 1, toUnix), client)
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
type vodCompressedBytesResult struct {
	compressedBytes []byte
	dwp             *vods.DomainWithPath
	duration        time.Duration
}

func getValidDwp(ctx context.Context, videoData *vods.VideoData, client *http.Client) (*vods.ValidDwpResponse, error) {
	dwp, err := getFirstValidDwpResponse(ctx, videoData, true, client)
	if err == nil {
		return dwp, nil
	}
	dwp, err = getFirstValidDwpResponse(ctx, &vods.VideoData{
		StreamerName: videoData.StreamerName,
		VideoId:      videoData.VideoId,
		Time:         videoData.Time.Add(-time.Second),
	}, true, client)
	if err == nil {
		log.Println(fmt.Sprint("minus 1 success for ", *videoData))
		return dwp, nil
	}
	dwp, err = getFirstValidDwpResponse(ctx, videoData, false, client)
	if err == nil {
		log.Println(fmt.Sprint("non-unix success for ", *videoData))
		return dwp, nil
	}
	return dwp, err
}

func getVodCompressedBytes(ctx context.Context, videoData *vods.VideoData, compressor *libdeflate.Compressor, client *http.Client) (*vodCompressedBytesResult, error) {
	dwp, err := getValidDwp(ctx, videoData, client)
	if err != nil {
		log.Println(fmt.Sprint("Link was not found for ", videoData.StreamerName, " because: ", err))
		return nil, err
	}
	mediapl, err := getCleanedMediaPlaylistBytes(dwp)
	if err != nil {
		log.Println(fmt.Sprint("Removing nil segments for ", videoData.StreamerName, " failed because: ", err))
		log.Println(mediapl.String())
		return nil, err
	}
	duration := vods.GetMediaPlaylistDuration(mediapl)
	compressedBytes, err := getCompressedBytes(mediapl.Encode().Bytes(), compressor)
	if err != nil {
		log.Println(fmt.Sprint("Compressing failed for ", videoData.StreamerName, " because: ", err))
		return nil, err
	}
	return &vodCompressedBytesResult{compressedBytes: compressedBytes, dwp: dwp.Dwp, duration: duration}, nil
}

type videoStatus struct {
	public          sql.NullBool
	boxArtUrl       sql.NullString
	profileImageUrl sql.NullString
}

func SetBoxArtWidthHeight(boxArtUrl string, width int, height int) string {
	return strings.Replace(boxArtUrl, "-{width}x{height}", fmt.Sprint("-", width, "x", height), 1)
}

func SetProfileImageWidth(profileImageUrl string, width int) string {
	return strings.Replace(profileImageUrl, "-300x300.png", fmt.Sprint("-", width, "x", width, ".png"), 1)
}

func getVideoStatus(ctx context.Context, client *helix.Client, streamerId string, streamId string, gameId string) videoStatus {
	var public sql.NullBool
	{
		response, err := retryOnError(func() (*helix.VideosResponse, error) {
			return client.GetVideos(&helix.VideosParams{UserID: streamerId})
		})
		if err != nil {
			log.Println(fmt.Sprint("error getting user data for (", streamerId, ", ", streamId, "): ", err))
		} else {
			public = sql.NullBool{Valid: true, Bool: false}
			videos := response.Data.Videos
			for _, cur := range videos {
				if cur.StreamID == streamId {
					public = sql.NullBool{Valid: true, Bool: true}
					break
				}
			}
		}

	}
	var profileImageUrl sql.NullString
	{
		usersResponse, _ := retryOnError(func() (*helix.UsersResponse, error) {
			return client.GetUsers(&helix.UsersParams{IDs: []string{streamerId}})
		})
		if len(usersResponse.Data.Users) > 0 {
			profileImageUrlStr := usersResponse.Data.Users[0].ProfileImageURL
			profileImageUrlStr = SetProfileImageWidth(profileImageUrlStr, 50)
			profileImageUrl = sql.NullString{Valid: true, String: profileImageUrlStr}
		}
	}
	var boxArtUrl sql.NullString
	{
		gamesResponse, _ := retryOnError(func() (*helix.GamesResponse, error) {
			return client.GetGames(&helix.GamesParams{IDs: []string{gameId}})
		})
		if len(gamesResponse.Data.Games) > 0 {
			boxArtUrlStr := gamesResponse.Data.Games[0].BoxArtURL
			boxArtUrlStr = SetBoxArtWidthHeight(boxArtUrlStr, 40, 56)
			boxArtUrl = sql.NullString{Valid: true, String: boxArtUrlStr}
		}
	}
	return videoStatus{
		public:          public,
		profileImageUrl: profileImageUrl,
		boxArtUrl:       boxArtUrl,
	}
}

type hlsWorkerFetchCompressSendParams struct {
	ctx               context.Context
	twitchHelixClient *helix.Client
	httpClient        *http.Client
	oldVodJobsCh      chan *LiveVod
	hlsFetcherDelay   time.Duration
	compressor        *libdeflate.Compressor
	resultsCh         chan *VodResult
	requestTimeLimit  time.Duration
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
		requestInitiated := time.Now().UTC()
		compressedBytesResult, err := getVodCompressedBytes(params.ctx, oldVod.GetVideoData(), params.compressor, params.httpClient)
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
				HlsDurationSeconds: sql.NullFloat64{
					Float64: 0.0,
					Valid:   false,
				},
			}
		} else {
			result = &VodResult{
				Vod:              oldVod,
				HlsBytes:         compressedBytesResult.compressedBytes,
				HlsBytesFound:    true,
				RequestInitiated: requestInitiated,
				HlsDomain: sql.NullString{
					String: compressedBytesResult.dwp.Domain,
					Valid:  true,
				},
				HlsDurationSeconds: sql.NullFloat64{
					Float64: compressedBytesResult.duration.Seconds(),
					Valid:   true,
				},
			}
		}
		videoMeta := getVideoStatus(params.ctx, params.twitchHelixClient, oldVod.StreamerId, oldVod.StreamId, oldVod.GameIdAtStart)
		result.Public = videoMeta.public
		result.BoxArtUrl = videoMeta.boxArtUrl
		result.ProfileImageUrl = videoMeta.profileImageUrl
		select {
		case <-params.ctx.Done():
			return
		case params.resultsCh <- result:
		}
	}
}

func processVodResults(ctx context.Context, resultsCh chan *VodResult, done chan struct{}, queries *sqlvods.Queries) {
	for {
		var result *VodResult
		select {
		case result = <-resultsCh:
		case <-ctx.Done():
		}
		if result == nil {
			log.Println("ctx is done meaning result is nil, so breaking")
			break
		}
		log.Println("Result was logged:")
		log.Println(*result.Vod)
		log.Println(fmt.Sprint("Gzipped size: ", len(result.HlsBytes)))
		upsertRecordingParams := sqlvods.UpdateRecordingParams{
			RecordingFetchedAt:     sql.NullTime{Time: result.RequestInitiated, Valid: true},
			GzippedBytes:           result.HlsBytes,
			StreamID:               result.Vod.StreamId,
			BytesFound:             sql.NullBool{Bool: result.HlsBytesFound, Valid: true},
			HlsDomain:              result.HlsDomain,
			Public:                 result.Public,
			HlsDurationSeconds:     result.HlsDurationSeconds,
			ProfileImageUrlAtStart: result.ProfileImageUrl,
			BoxArtUrlAtStart:       result.BoxArtUrl,
			StartTime:              time.Unix(result.Vod.StartTimeUnix, 0).UTC(),
		}
		err := queries.UpdateRecording(ctx, upsertRecordingParams)
		if err != nil {
			log.Println(fmt.Sprint("upserting recording failed: ", err))
			break
		}
		updateStreamerParams := sqlvods.UpdateStreamerParams{
			StreamerLoginAtStart:   result.Vod.StreamerLoginAtStart,
			ProfileImageUrlAtStart: result.ProfileImageUrl,
		}
		err = queries.UpdateStreamer(ctx, updateStreamerParams)
		if err != nil {
			log.Println(fmt.Sprint("updating streamer failed: ", err))
			break
		}
	}
	select {
	case <-ctx.Done():
	case done <- struct{}{}:
	}
}

type ScrapeTwitchLiveVodsWithGqlApiParams struct {
	RunScraperParams
	// initial live vod queue fetched from database
	InitialWaitVodQueue *waitVodsPriorityQueue
	// sqlc queries instance
	Queries *sqlvods.Queries
}

func makeRobustHttpClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
	}
	transport := &http.Transport{DialContext: dialer.DialContext}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
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
	httpClient := makeRobustHttpClient(params.RequestTimeLimit)
	twitchHelixClient, err := helix.NewClient(&helix.Options{
		ClientID:     params.ClientId,
		ClientSecret: params.ClientSecret,
		HTTPClient:   httpClient,
	})
	if err != nil {
		return err
	}
	_, err = retryOnError(func() (struct{}, error) {
		return struct{}{}, resetAppAccessToken(twitchHelixClient)
	})
	if err != nil {
		return err
	}
	oldVodsCh := make(chan []*LiveVod)
	oldVodJobsCh := make(chan *LiveVod)
	resultsCh := make(chan *VodResult)
	done := make(chan struct{})
	log.Println("Made twitchgql client and channels.")
	go fetchTwitchHelixForever(
		fetchTwitchHelixForeverParams{
			ctx:                      ctx,
			twitchHelixClient:        twitchHelixClient,
			initialWaitVodQueue:      params.InitialWaitVodQueue,
			sqlRequestTimeLimit:      params.RequestTimeLimit,
			twitchHelixFetcherDelay:  params.TwitchHelixFetcherDelay,
			cursorResetThreshold:     params.CursorResetThreshold,
			liveVodEvictionThreshold: params.LiveVodEvictionThreshold,
			waitVodEvictionThreshold: params.WaitVodEvictionThreshold,
			oldVodsCh:                oldVodsCh,
			minViewerCountToObserve:  params.MinViewerCountToObserve,
			minViewerCountToRecord:   params.MinViewerCountToRecord,
			queries:                  params.Queries,
			numStreamsPerRequest:     params.NumStreamsPerRequest,
			oldVodsDelete:            params.OldVodsDelete,
			done:                     done,
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
			ctx:               ctx,
			twitchHelixClient: twitchHelixClient,
			httpClient:        httpClient,
			oldVodJobsCh:      oldVodJobsCh,
			hlsFetcherDelay:   params.HlsFetcherDelay,
			compressor:        &compressor,
			resultsCh:         resultsCh,
			requestTimeLimit:  params.RequestTimeLimit,
		})
	}
	go processVodResults(ctx, resultsCh, done, params.Queries)
	select {
	case <-done:
	case <-ctx.Done():
	}
	return nil
}

type RunScraperParams struct {
	// In any interval of this length, the api will be called at most twice and on average once.
	TwitchHelixFetcherDelay time.Duration
	// Time limit for .m3u8 and Twitch Helix requests. If this is exceeded in the TwitchGQL loop, the for-loop continues. TODO: I should fix this.
	RequestTimeLimit time.Duration
	// If a VOD in the queue of live VODs has a last updated time older than this, it is moved out of the live VODs queue.
	LiveVodEvictionThreshold time.Duration
	// If a VOD in the queue of wait VODs has a last interaction time older than this, it is moved out of the wait VODs queue.
	WaitVodEvictionThreshold time.Duration
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
	// Vods older than the current time minus this duration will be deleted
	OldVodsDelete time.Duration
	// Twitch helix client ID
	ClientId string
	// Twitch helix client secret
	ClientSecret string
}

// databaseUrl is the postgres database to connect to.
// evictionRatio should be at least 1.
// We select live vods that were updated at most evictionRatio * (liveVodEvictionThreshold + waitVodEvictionThreshold) ago before the newest live vod.
// params are the parameters twitch graphql scraper.
func RunScraper(ctx context.Context, databaseUrl string, evictionRatio float64, params RunScraperParams) error {
	type tInitialState struct {
		conn         *pgxpool.Pool
		queries      *sqlvods.Queries
		waitVodQueue *waitVodsPriorityQueue
	}
	getInitialState := func(ctx context.Context) (*tInitialState, error) {
		compressor, err := libdeflate.NewCompressorLevel(params.LibdeflateCompressionLevel)
		if err != nil {
			log.Println(fmt.Sprint("failed to create compressor: ", err))
			return nil, err
		}
		_, err = getCompressedBytes([]byte("Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet."), &compressor)
		compressor.Close()
		if err != nil {
			log.Println(fmt.Sprint("failed to compress: ", err))
			return nil, err
		}
		testClient := makeRobustHttpClient(params.RequestTimeLimit)
		resp, err := retryOnError(func() (*http.Response, error) {
			return testClient.Get(vods.DOMAINS[0])
		})
		if err != nil {
			log.Println(fmt.Sprint("failed to establish test connection to domain: ", err))
			return nil, err
		}
		resp.Body.Close()
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
		waitVodQueue := CreateNewWaitVodsPriorityQueue()
		if len(latestStreams) == 0 {
			log.Println("there are 0 live vods")
			return &tInitialState{conn: conn, queries: queries, waitVodQueue: waitVodQueue}, nil
		}
		latestStream := latestStreams[0]
		lastTimeAllowed := latestStream.LastUpdatedAt.UTC().Add(-time.Duration(float64(params.LiveVodEvictionThreshold+params.WaitVodEvictionThreshold) * evictionRatio))
		latestLiveStreams, err := queries.GetLatestLiveStreams(ctx, lastTimeAllowed)
		if err != nil {
			log.Println("There are no latestLiveStreams")
			conn.Close()
			return nil, err
		}
		lastInteraction := time.Now().UTC()
		for _, liveStream := range latestLiveStreams {
			waitVodQueue.Put(&LiveVod{
				StreamerId:           liveStream.StreamerID,
				StreamId:             liveStream.StreamID,
				StartTimeUnix:        liveStream.StartTime.UTC().Unix(),
				StreamerLoginAtStart: liveStream.StreamerLoginAtStart,
				GameIdAtStart:        liveStream.GameIDAtStart,
				MaxViews:             int(liveStream.MaxViews),
				LastUpdatedUnix:      liveStream.LastUpdatedAt.UTC().Unix(),
				LastInteractionUnix:  lastInteraction.Unix(),
			})
		}
		return &tInitialState{conn: conn, queries: queries, waitVodQueue: waitVodQueue}, nil
	}
	initialState, err := again.Retry(ctx, getInitialState)
	if err != nil {
		log.Println(fmt.Sprint("failed to get initial state: ", err))
		return err
	}
	defer initialState.conn.Close()
	log.Println(fmt.Sprint("entries in waitVodsQueue: ", initialState.waitVodQueue.Size()))
	return ScrapeTwitchLiveVodsWithGqlApi(
		ctx,
		ScrapeTwitchLiveVodsWithGqlApiParams{
			RunScraperParams:    params,
			InitialWaitVodQueue: initialState.waitVodQueue,
			Queries:             initialState.queries,
		},
	)
}

func RunScraperForever(ctx context.Context, scraperDuration time.Duration, databaseUrl string, evictionRatio float64, params RunScraperParams) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			scraperCtx, cancelScraper := context.WithTimeout(ctx, scraperDuration)
			RunScraper(scraperCtx, databaseUrl, evictionRatio, params)
			cancelScraper()
		}
	}
}
