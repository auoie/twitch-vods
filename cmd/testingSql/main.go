package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/auoie/goVods/scraper"
	"github.com/auoie/goVods/sqlvods"
	"github.com/jackc/pgx/v4/pgxpool"
)

var streams0 = []scraper.LiveVod{
	{
		StreamerId:           "streamerid0",
		StreamId:             "streamid0",
		StartTime:            time.Now().Add(-60 * time.Minute),
		StreamerLoginAtStart: "0",
		MaxViews:             100,
		LastUpdated:          time.Now().Add(-1 * time.Minute),
		TimeSeries:           []scraper.VodDataPoint{},
	},
}

func getUpsertManyStreamsParams(streams []scraper.LiveVod) sqlvods.UpsertManyStreamsParams {
	result := sqlvods.UpsertManyStreamsParams{}
	for _, stream := range streams {
		result.LastUpdatedAtArr = append(result.LastUpdatedAtArr, stream.LastUpdated)
		result.MaxViewsArr = append(result.MaxViewsArr, int64(stream.MaxViews))
		result.StartTimeArr = append(result.StartTimeArr, stream.StartTime)
		result.StreamIDArr = append(result.StreamIDArr, stream.StreamId)
		result.StreamerIDArr = append(result.StreamerIDArr, stream.StreamerId)
		result.StreamerLoginAtStartArr = append(result.StreamerLoginAtStartArr, stream.StreamerLoginAtStart)
	}
	return result
}

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
		log.Fatal("unable to ping database at ", databaseUrl, ": ", err)
	}
	queries := sqlvods.New(conn)
	err = queries.UpsertManyStreams(context.Background(), getUpsertManyStreamsParams(streams0))
	if err != nil {
		log.Fatal(fmt.Sprint("failed to upsert: ", err))
	}
	err = queries.DeleteRecordings(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	err = queries.DeleteStreams(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
