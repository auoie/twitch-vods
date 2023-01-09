package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/auoie/twitch-vods/scraper"
)

func main() {
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		log.Fatal("DATABASE_URL is missing for db connection string")
	}
	scraper.RunScraperForever(
		context.Background(),
		24*time.Hour*7,
		databaseUrl,
		2.0,
		scraper.RunScraperParams{
			TwitchGqlFetcherDelay:      333 * time.Millisecond,
			RequestTimeLimit:           30 * time.Second,
			LiveVodEvictionThreshold:   15 * time.Minute,
			WaitVodEvictionThreshold:   60 * time.Minute,
			MaxOldVodsQueueSize:        50000,
			NumHlsFetchers:             5,
			HlsFetcherDelay:            1 * time.Second,
			CursorResetThreshold:       5 * time.Minute,
			LibdeflateCompressionLevel: 1,
			MinViewerCountToObserve:    5,
			MinViewerCountToRecord:     10,
			NumStreamsPerRequest:       30,
			OldVodsDelete:              time.Hour * 24 * 14,
			CursorFactor:               0.8,
		},
	)
}
