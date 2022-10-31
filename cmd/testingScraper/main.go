package main

import (
	"context"
	"time"

	"github.com/auoie/goVods/scraper"
)

func main() {
	scraper.ScrapeTwitchLiveVodsWithGqlApi(
		scraper.ScrapeTwitchLiveVodsWithGqlApiParams{
			Ctx:                        context.Background(),
			TwitchGqlFetcherDelay:      333 * time.Millisecond,
			RequestTimeLimit:           5 * time.Second,
			OldVodEvictionThreshold:    10 * time.Minute,
			MaxOldVodsQueueSize:        20000,
			NumHlsFetchers:             5,
			HlsFetcherDelay:            time.Second,
			CursorResetThreshold:       5 * time.Minute,
			LibdeflateCompressionLevel: 1,
			MinViewerCountToObserve:    5,
		},
	)
}
