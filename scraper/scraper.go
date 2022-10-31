package scraper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/4kills/go-libdeflate/v2"
	"github.com/Khan/genqlient/graphql"
	"github.com/auoie/goVods/twitchgql"
	"github.com/auoie/goVods/vods"
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
	Vod           *LiveVod
	HlsBytes      []byte
	HlsBytesFound bool
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
	client                    graphql.Client
	twitchGqlRequestTimeLimit time.Duration
	twitchGqlFetcherDelay     time.Duration
	cursorResetThreshold      time.Duration
	oldVodEvictionThreshold   time.Duration
	oldVodsCh                 chan []*LiveVod
	minViewerCountToObserve   int
}

func fetchTwitchGqlForever(params fetchTwitchGqlForeverParams) {
	log.Println("Inside fetchTwitchGqlForever...")
	log.Println(fmt.Sprint("Fetcher delay: ", params.twitchGqlFetcherDelay))
	liveVodQueue := CreateNewLiveVodsPriorityQueue()
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
		debugIndex++
		debugQueueSizeStart := liveVodQueue.Size()
		if time.Now().After(resetCursorTimeout) {
			log.Println(fmt.Sprint("Reset cursor because we've been fetching for: ", params.cursorResetThreshold))
			resetCursor()
		}
		requestCtx, requestCancel := context.WithTimeout(params.ctx, params.twitchGqlRequestTimeLimit)
		streams, err := twitchgql.GetStreams(requestCtx, params.client, 30, cursor)
		responseReturnedTime := time.Now()
		requestCancel()
		select {
		case <-params.ctx.Done():
			return
		case <-twitchGqlTicker.C:
		}
		if err != nil {
			log.Println(fmt.Sprint("Twitch graphql client reported an error: ", err))
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
				cursor = edges[2*len(edges)/3].Cursor
			}
		}
		prevEdges = edges
		oldVods := []*LiveVod{}
		allVodsLessThanMinViewerCount := true
		for _, edge := range edges {
			if edge.Node.ViewersCount < params.minViewerCountToObserve {
				continue
			}
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
			oldVods = append(oldVods, stalestVod)
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

type hlsWorkerFetchCompressSendParams struct {
	ctx             context.Context
	oldVodJobsCh    chan *LiveVod
	hlsFetcherDelay time.Duration
	compressor      *libdeflate.Compressor
	resultsCh       chan *VodResult
}

// Find the .m3u8 for a video and return the compressed bytes.
func getVodCompressedBytes(ctx context.Context, videoData *vods.VideoData, compressor *libdeflate.Compressor) ([]byte, error) {
	dwp, err := vods.GetFirstValidDwp(ctx, videoData.GetDomainWithPathsList(vods.DOMAINS, 1))
	if err != nil {
		log.Println(fmt.Sprint("Link was not found for ", videoData.StreamerName, " because: ", err))
		return nil, err
	}
	mediapl, err := vods.DecodeMediaPlaylistFilterNilSegments(dwp.Body, true)
	if err != nil {
		log.Println(fmt.Sprint("Removing nil segments for ", videoData.StreamerName, " failed because: ", err))
		log.Println(mediapl.String())
		return nil, err
	}
	vods.MuteMediaSegments(mediapl)
	dwp.Dwp.MakePathsExplicit(mediapl)
	mediaplBytes := mediapl.Encode().Bytes()
	compressedBytes := make([]byte, len(mediaplBytes))
	n, _, err := compressor.Compress(mediaplBytes, compressedBytes, libdeflate.ModeGzip)
	if err != nil {
		log.Println(fmt.Sprint("Compressing failed for ", videoData.StreamerName, " because: ", err))
		return nil, err
	}
	compressedBytes = compressedBytes[:n]
	return compressedBytes, nil
}

func hlsWorkerFetchCompressSend(params hlsWorkerFetchCompressSendParams) {
	hlsFetcherTicker := time.NewTicker(params.hlsFetcherDelay)
	defer params.compressor.Close()
	getOldVodJob := func() (*LiveVod, error) {
		select {
		case <-params.ctx.Done():
			return nil, errors.New("context done")
		case oldVod := <-params.oldVodJobsCh:
			return oldVod, nil
		}
	}
	for {
		select {
		case <-params.ctx.Done():
			return
		case <-hlsFetcherTicker.C:
		}
		oldVod, err := getOldVodJob()
		if err != nil {
			return
		}
		bytes, err := getVodCompressedBytes(params.ctx, oldVod.GetVideoData(), params.compressor)
		var result *VodResult
		if err != nil {
			result = &VodResult{Vod: oldVod, HlsBytes: nil, HlsBytesFound: false}
		} else {
			result = &VodResult{Vod: oldVod, HlsBytes: bytes, HlsBytesFound: true}
		}
		select {
		case <-params.ctx.Done():
			return
		case params.resultsCh <- result:
		}
	}
}

type ScrapeTwitchLiveVodsWithGqlApiParams struct {
	// Context to cancel the scraping operation.
	Ctx context.Context
	// In any interval of this length, the api will be called at most twice and on average once.
	TwitchGqlFetcherDelay time.Duration
	// If this is exceeded, the for loop continues. TODO: I should fix this.
	TwitchGqlRequestTimeLimit time.Duration
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
}

// This function scares me.
func ScrapeTwitchLiveVodsWithGqlApi(params ScrapeTwitchLiveVodsWithGqlApiParams) error {
	log.Println("Starting scraping...")
	ctx, cancel := context.WithCancel(params.Ctx)
	defer cancel()
	client := twitchgql.NewTwitchGqlClient()
	oldVodsCh := make(chan []*LiveVod)
	oldVodJobsCh := make(chan *LiveVod)
	resultsCh := make(chan *VodResult)
	log.Println("Made twitchgql client and channels.")
	go fetchTwitchGqlForever(
		fetchTwitchGqlForeverParams{
			ctx:                       ctx,
			client:                    client,
			twitchGqlRequestTimeLimit: params.TwitchGqlRequestTimeLimit,
			twitchGqlFetcherDelay:     params.TwitchGqlFetcherDelay,
			cursorResetThreshold:      params.CursorResetThreshold,
			oldVodEvictionThreshold:   params.OldVodEvictionThreshold,
			oldVodsCh:                 oldVodsCh,
			minViewerCountToObserve:   params.MinViewerCountToObserve,
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
			ctx:             ctx,
			oldVodJobsCh:    oldVodJobsCh,
			hlsFetcherDelay: params.HlsFetcherDelay,
			compressor:      &compressor,
			resultsCh:       resultsCh,
		})
	}
	go func() {
		for {
			result := <-resultsCh
			log.Println("Result was logged:")
			log.Println(*result.Vod)
			log.Println(fmt.Sprint("Gzipped size: ", len(result.HlsBytes)))
			// send result to a database
		}
	}()
	<-ctx.Done()
	return nil
}
