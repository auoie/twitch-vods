package scraper

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/4kills/go-libdeflate/v2"
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

// Params
// Context to cancel the scraping operation
// timelimit for each of the graphql requests
func ScrapeTwitchLiveVodsWithGqlApi(
	ctx context.Context,
	twitchGqlFetcherDelay time.Duration,
	twitchGqlRequestTimeLimit time.Duration,
	oldVodEvictionThreshold time.Duration,
	maxOldVodsQueueSize int,
	numHlsFetchers int,
	hlsFetcherDelay time.Duration,
	compressor *libdeflate.Compressor) {
	client := twitchgql.NewTwitchGqlClient()
	liveVodQueue := CreateNewLiveVodsPriorityQueue()
	oldVodsCh := make(chan []*LiveVod)
	oldVodJobsCh := make(chan *LiveVod)
	resultsCh := make(chan *VodResult)
	go func() {
		twitchGqlTicker := time.NewTicker(twitchGqlFetcherDelay)
		defer twitchGqlTicker.Stop()
		cursor := ""
		for {
			ctx, cancel := context.WithTimeout(ctx, twitchGqlRequestTimeLimit)
			curTime := time.Now()
			streams, err := twitchgql.GetStreams(ctx, client, 30, cursor)
			<-twitchGqlTicker.C
			if err != nil {
				log.Println(fmt.Sprint("Twitch graphql client reported an error: ", err))
			}
			cancel()
			edges := streams.Streams.Edges
			if len(edges) == 0 || !streams.Streams.PageInfo.HasNextPage {
				cursor = ""
			} else {
				cursor = edges[len(edges)-1].Cursor
			}
			oldVods := []*LiveVod{}
			allVodsAtMostOneView := true
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
				cursor = ""
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
			oldVodsCh <- oldVods
		}
	}()
	go func() {
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
	}()
	for i := 0; i < numHlsFetchers; i++ {
		go func() {
			hlsFetcherTicker := time.NewTicker(hlsFetcherDelay)
			for {
				<-hlsFetcherTicker.C
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
				resultsCh <- result
			}
		}()
	}
	go func() {
		for {
			result := <-resultsCh
			fmt.Println(result)
			// send result to a database
		}
	}()
}
