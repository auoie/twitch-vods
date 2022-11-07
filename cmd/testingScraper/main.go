package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/auoie/goVods/scraper"
	"github.com/auoie/goVods/sqlvods"
	"github.com/jackc/pgx/v4/pgxpool"
)

func main() {
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		databaseUrl = "postgresql://govods:password@localhost:5432/twitch"
	}
	conn, err := pgxpool.Connect(context.Background(), databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	err = conn.Ping(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	queries := sqlvods.New(conn)
	scraper.ScrapeTwitchLiveVodsWithGqlApi(
		scraper.ScrapeTwitchLiveVodsWithGqlApiParams{
			Ctx:                        context.Background(),
			TwitchGqlFetcherDelay:      333 * time.Millisecond,
			RequestTimeLimit:           5 * time.Second,
			OldVodEvictionThreshold:    15 * time.Minute,
			MaxOldVodsQueueSize:        50000,
			NumHlsFetchers:             5,
			HlsFetcherDelay:            time.Second,
			CursorResetThreshold:       5 * time.Minute,
			LibdeflateCompressionLevel: 1,
			MinViewerCountToObserve:    5,
			MinViewerCountToRecord:     10,
			NumStreamsPerRequest:       30,
			CursorFactor:               0.8,
			Queries:                    queries,
		},
	)
}
