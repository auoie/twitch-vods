package scraper

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/4kills/go-libdeflate/v2"
	"github.com/Khan/genqlient/graphql"
	"github.com/auoie/goVods/twitchgql"
	"github.com/auoie/goVods/vods"
)

type LiveVod struct {
	StreamerId           string
	StreamId             string
	StartTime            time.Time
	StreamerLoginAtStart string
	MaxViews             int
	LastUpdated          time.Time
	TimeSeries           []twitchgql.VodDataPoint
}

type VodResult struct {
	Vod      *LiveVod
	HlsBytes []byte
}

func (vod *LiveVod) GetVideoData() *vods.VideoData {
	return &vods.VideoData{StreamerName: vod.StreamerLoginAtStart, VideoId: vod.StreamId, Time: vod.StartTime}
}

func fetchTwitchGqlForever(
	ctx context.Context,
	client graphql.Client,
	twitchGqlRequestTimeLimit time.Duration,
	twitchGqlFetcherDelay time.Duration,
	cursorResetThreshold time.Duration,
	oldVodEvictionThreshold time.Duration,
	oldVodsCh chan []*LiveVod) {
	liveVodQueue := CreateNewLiveVodsPriorityQueue()
	twitchGqlTicker := time.NewTicker(twitchGqlFetcherDelay)
	defer twitchGqlTicker.Stop()
	cursor := ""
	resetCursorTimeout := time.Now().Add(cursorResetThreshold)
	resetCursor := func() {
		cursor = ""
		resetCursorTimeout = time.Now().Add(cursorResetThreshold)
	}
	for {
		if time.Now().After(resetCursorTimeout) {
			resetCursor()
		}
		ctx, cancel := context.WithTimeout(ctx, twitchGqlRequestTimeLimit)
		streams, err := twitchgql.GetStreams(ctx, client, 30, cursor)
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-twitchGqlTicker.C:
		}
		if err != nil {
			log.Println(fmt.Sprint("Twitch graphql client reported an error: ", err))
		}
		edges := streams.Streams.Edges
		if len(edges) == 0 || !streams.Streams.PageInfo.HasNextPage {
			resetCursor()
		} else {
			cursor = edges[len(edges)-1].Cursor
		}
		oldVods := []*LiveVod{}
		allVodsAtMostOneView := true
		curTime := time.Now()
		for _, edge := range edges {
			if edge.Node.ViewersCount <= 1 {
				continue
			}
			allVodsAtMostOneView = false
			evictedVod, err := liveVodQueue.UpsertVod(
				curTime,
				twitchgql.VodDataPoint(&edge.Node),
			)
			if err != nil {
				continue
			}
			oldVods = append(oldVods, evictedVod)
		}
		if allVodsAtMostOneView {
			resetCursor()
		}
		oldestTimeAllowed := curTime.Add(-oldVodEvictionThreshold)
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
		select {
		case <-ctx.Done():
			return
		case oldVodsCh <- oldVods:
		}
	}
}

func processOldVodJobs(
	ctx context.Context,
	oldVodsCh chan []*LiveVod,
	oldVodJobsCh chan *LiveVod,
	maxOldVodsQueueSize int) {
	oldVodsOrderedByViews := CreateNewOldVodQueue()
	getJobsCh := func() chan *LiveVod {
		if oldVodsOrderedByViews.Size() == 0 {
			return nil
		}
		return oldVodJobsCh
	}
	getNextInQueue := func() *LiveVod {
		oldVod, _ := oldVodsOrderedByViews.PopHighViewCount()
		return oldVod
	}
	for {
		select {
		case <-ctx.Done():
			return
		case oldVods := <-oldVodsCh:
			for _, oldVod := range oldVods {
				oldVodsOrderedByViews.Put(oldVod)
				if oldVodsOrderedByViews.Size() > maxOldVodsQueueSize {
					oldVodsOrderedByViews.PopLowViewCount()
				}
			}
		case getJobsCh() <- getNextInQueue():
		}
	}
}

func hlsWorkerFetchCompressSend(
	ctx context.Context,
	oldVodJobsCh chan *LiveVod,
	hlsFetcherDelay time.Duration,
	compressor *libdeflate.Compressor,
	resultsCh chan *VodResult) {
	hlsFetcherTicker := time.NewTicker(hlsFetcherDelay)
	defer compressor.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case <-hlsFetcherTicker.C:
		}
		oldVod := <-oldVodJobsCh
		dwp, err := vods.GetFirstValidDwp(ctx, oldVod.GetVideoData().GetDomainWithPathsList(vods.DOMAINS, 1))
		if err != nil {
			continue
		}
		mediapl, err := vods.DecodeMediaPlaylistFilterNilSegments(dwp.Body, true)
		if err != nil {
			continue
		}
		vods.MuteMediaSegments(mediapl)
		dwp.Dwp.MakePathsExplicit(mediapl)
		mediaplBytes := mediapl.Encode().Bytes()
		compressedBytes := make([]byte, len(mediaplBytes))
		n, _, err := compressor.Compress(mediaplBytes, compressedBytes, libdeflate.ModeGzip)
		if err != nil {
			continue
		}
		compressedBytes = compressedBytes[:n]
		result := &VodResult{Vod: oldVod, HlsBytes: compressedBytes}
		select {
		case <-ctx.Done():
			return
		case resultsCh <- result:
		}
	}
}

// Params
// Context to cancel the scraping operation
// timelimit for each of the graphql requests
// This function scares me.
func ScrapeTwitchLiveVodsWithGqlApi(
	ctx context.Context,
	twitchGqlFetcherDelay time.Duration,
	twitchGqlRequestTimeLimit time.Duration,
	oldVodEvictionThreshold time.Duration,
	maxOldVodsQueueSize int,
	numHlsFetchers int,
	hlsFetcherDelay time.Duration,
	cursorResetThreshold time.Duration,
	libdeflateCompressionLevel int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	client := twitchgql.NewTwitchGqlClient()
	oldVodsCh := make(chan []*LiveVod)
	oldVodJobsCh := make(chan *LiveVod)
	resultsCh := make(chan *VodResult)
	go fetchTwitchGqlForever(ctx, client, twitchGqlRequestTimeLimit, twitchGqlFetcherDelay, cursorResetThreshold, oldVodEvictionThreshold, oldVodsCh)
	go processOldVodJobs(ctx, oldVodsCh, oldVodJobsCh, maxOldVodsQueueSize)
	for i := 0; i < numHlsFetchers; i++ {
		compressor, err := libdeflate.NewCompressorLevel(libdeflateCompressionLevel)
		if err != nil {
			return err
		}
		go hlsWorkerFetchCompressSend(ctx, oldVodJobsCh, hlsFetcherDelay, &compressor, resultsCh)
	}
	go func() {
		for {
			result := <-resultsCh
			fmt.Println(result)
			// send result to a database
		}
	}()
	return nil
}
