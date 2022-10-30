package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/auoie/goVods/scraper"
	"github.com/auoie/goVods/sqlvods"
	"github.com/jackc/pgx/v4/pgxpool"
)

func createLiveVod(id int) scraper.LiveVod {
	return scraper.LiveVod{
		StreamerId:           fmt.Sprint("streamerid", id),
		StreamId:             fmt.Sprint("streamid", id),
		StartTime:            time.Now().Add(-60 * time.Minute),
		StreamerLoginAtStart: fmt.Sprint(id),
		MaxViews:             100 + id,
		LastUpdated:          time.Now().Add(-1 * time.Minute),
	}
}

var streams0 = []scraper.LiveVod{
	createLiveVod(0),
	createLiveVod(1),
	createLiveVod(2),
}

var streams1 = []scraper.LiveVod{
	createLiveVod(0),
	createLiveVod(1),
	createLiveVod(2),
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

func prettyPrint(value any) {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(bytes))
}

func logFatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
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
	logFatalOnError(conn.Ping(context.Background()))
	queries := sqlvods.New(conn)
	logFatalOnError(queries.UpsertManyStreams(context.Background(), getUpsertManyStreamsParams(streams0)))
	everything, err := queries.GetEverything(context.Background())
	logFatalOnError(err)
	prettyPrint(everything)
	logFatalOnError(queries.UpsertManyStreams(context.Background(), getUpsertManyStreamsParams(streams1)))
	everything, err = queries.GetEverything(context.Background())
	logFatalOnError(err)
	prettyPrint(everything)
	helloStream, err := queries.GetStreamByStreamId(context.Background(), "hello")
	log.Println(err)
	prettyPrint(helloStream)
	helloStreams, err := queries.GetStreamsByStreamId(context.Background(), "hello")
	logFatalOnError(err)
	prettyPrint(len(helloStreams))
	logFatalOnError(queries.DeleteRecordings(context.Background()))
	logFatalOnError(queries.DeleteStreams(context.Background()))
}
