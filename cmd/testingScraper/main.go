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
	clientId, ok := os.LookupEnv("CLIENT_ID")
	if !ok {
		log.Fatal("CLIENT_ID is missing for twitch helix API")
	}
	clientSecret, ok := os.LookupEnv("CLIENT_SECRET")
	if !ok {
		log.Fatal("CLIENT_SECRET is missing for twitch helix API")
	}
	scraper.RunScraperForever(
		context.Background(),
		24*time.Hour*7,
		databaseUrl,
		2.0,
		scraper.RunScraperParams{
			TwitchHelixFetcherDelay:    333 * time.Millisecond,
			RequestTimeLimit:           30 * time.Second,
			LiveVodEvictionThreshold:   15 * time.Minute,
			WaitVodEvictionThreshold:   60 * time.Minute,
			MaxOldVodsQueueSize:        50000,
			NumHlsFetchers:             3,
			HlsFetcherDelay:            1 * time.Second,
			CursorResetThreshold:       150 * time.Second,
			LibdeflateCompressionLevel: 1,
			MinViewerCountToObserve:    5,
			MinViewerCountToRecord:     10,
			NumStreamsPerRequest:       100,
			OldVodsDelete:              time.Hour * 24 * 14,
			ClientId:                   clientId,
			ClientSecret:               clientSecret,
		},
	)
}
