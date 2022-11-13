package main

import (
	"context"
	"os"
	"time"

	"github.com/auoie/goVods/scraper"
)

func main() {
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		databaseUrl = "postgresql://govods:password@localhost:5432/twitch"
	}
	scraper.RunScraper(
		context.Background(),
		databaseUrl,
		1.1,
		scraper.RunScraperParams{
			TwitchGqlFetcherDelay:      333 * time.Millisecond,
			RequestTimeLimit:           5 * time.Second,
			OldVodEvictionThreshold:    15 * time.Minute,
			MaxOldVodsQueueSize:        50000,
			NumHlsFetchers:             4,
			HlsFetcherDelay:            time.Second,
			CursorResetThreshold:       5 * time.Minute,
			LibdeflateCompressionLevel: 1,
			MinViewerCountToObserve:    5,
			MinViewerCountToRecord:     10,
			NumStreamsPerRequest:       30,
			CursorFactor:               0.8,
		})
}
