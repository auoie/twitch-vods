package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/auoie/twitchVods/scraper"
	"github.com/auoie/twitchVods/sqlvods"
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

func isNonZero[T comparable](input T) bool {
	var zeroValue T
	return input != zeroValue
}

func createLiveVodWithViews(id int, views int) scraper.LiveVod {
	return scraper.LiveVod{
		StreamerId:           fmt.Sprint("streamerid", id),
		StreamId:             fmt.Sprint("streamid", id),
		StartTime:            time.Now().Add(-60 * time.Minute),
		StreamerLoginAtStart: fmt.Sprint(id),
		MaxViews:             views,
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
	createLiveVodWithViews(1, 50),
	createLiveVod(2),
}

var streams2 = []scraper.LiveVod{
	createLiveVod(0),
	createLiveVodWithViews(1, 150),
	createLiveVod(2),
}

var streams3 = []scraper.LiveVod{
	createLiveVod(0),
	createLiveVod(1),
	createLiveVod(3),
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
		result.LanguageAtStartArr = append(result.LanguageAtStartArr, "EN")
		result.TitleAtStartArr = append(result.TitleAtStartArr, "title")
		result.GameNameAtStartArr = append(result.GameNameAtStartArr, "game")
	}
	return result
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

	// logFatalOnError(queries.UpsertManyStreams(context.Background(), getUpsertManyStreamsParams(streams0)))
	// everything, err := queries.GetEverything(context.Background())
	// logFatalOnError(err)
	// log.Println(everything)
	// log.Println(len(everything))
	// log.Print()

	// logFatalOnError(queries.UpsertManyStreams(context.Background(), getUpsertManyStreamsParams(streams1)))
	// everything, err = queries.GetEverything(context.Background())
	// logFatalOnError(err)
	// log.Println(everything)
	// log.Println(len(everything))
	// log.Print()

	// logFatalOnError(queries.UpsertManyStreams(context.Background(), getUpsertManyStreamsParams(streams2)))
	// everything, err = queries.GetEverything(context.Background())
	// logFatalOnError(err)
	// log.Println(everything)
	// log.Println(len(everything))
	// log.Print()

	// logFatalOnError(queries.UpsertManyStreams(context.Background(), getUpsertManyStreamsParams(streams3)))
	// everything, err = queries.GetEverything(context.Background())
	// logFatalOnError(err)
	// log.Println("upserted many streams")
	// log.Println(everything)
	// log.Println(len(everything))
	// log.Print()

	// batchedStreams := queries.GetStreamForEachStreamIdBatched(context.Background(), []string{"hmm", "streamid0", "doesn't exist"})
	// streams := []sqlvods.GetStreamForEachStreamIdBatchedRow{}
	// batchedStreams.Query(func(i int, gsfesibr []sqlvods.GetStreamForEachStreamIdBatchedRow, err error) {
	// 	if err != nil {
	// 		return
	// 	}
	// 	if len(gsfesibr) != 1 {
	// 		return
	// 	}
	// 	streams = append(streams, gsfesibr[0])
	// })
	// logFatalOnError(err)
	// log.Println(streams)
	// log.Println(len(streams))
	// log.Print()

	// err = queries.UpdateRecording(context.Background(), sqlvods.UpdateRecordingParams{
	// 	GzippedBytes:       []byte{'a', 'b', 'c'},
	// 	StreamID:           "lskdjfslkjf",
	// 	BytesFound:         sql.NullBool{Bool: true, Valid: true},
	// 	RecordingFetchedAt: sql.NullTime{Time: time.Now(), Valid: true}})
	// log.Println("upserted recording with invalid stream id")
	// log.Println(err)
	// log.Print()

	// log.Println(streams[0].StreamID)
	// logFatalOnError(queries.UpdateRecording(context.Background(), sqlvods.UpdateRecordingParams{
	// 	RecordingFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
	// 	GzippedBytes:       []byte{'a', 'b', 'c'},
	// 	StreamID:           streams[0].StreamID,
	// 	BytesFound:         sql.NullBool{Bool: true, Valid: true}}))
	// everything, err = queries.GetEverything(context.Background())
	// logFatalOnError(err)
	// log.Println(everything)
	// log.Println(len(everything))
	// log.Print()

	// logFatalOnError(queries.UpdateRecording(context.Background(), sqlvods.UpdateRecordingParams{
	// 	RecordingFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
	// 	GzippedBytes:       nil,
	// 	StreamID:           streams[0].StreamID,
	// 	BytesFound:         sql.NullBool{Bool: false, Valid: true},
	// 	Public:             sql.NullBool{Bool: true, Valid: false},
	// 	SubOnly:            sql.NullBool{Bool: false, Valid: false},
	// }))
	// everything, err = queries.GetEverything(context.Background())
	// logFatalOnError(err)
	// log.Println(streams[0].StreamID)
	// log.Println("hello")
	// log.Println(everything)
	// log.Println(len(everything))
	// log.Print()

	// results := queries.GetStreamForEachStreamIdBatched(context.Background(), []string{"hmm", "streamid0", "doesn't exist"})
	// results.Query(func(i int, gsfesibr []sqlvods.GetStreamForEachStreamIdBatchedRow, err error) {
	// 	if err != nil {
	// 		log.Println(err)
	// 	} else {
	// 		log.Println(gsfesibr)
	// 		log.Println(gsfesibr == nil)
	// 	}
	// })
	// log.Print()

	// streamsunnest, err := queries.GetStreamForEachStreamIdUnnest(context.Background(), []string{"hmm", "streamid0", "doesn't exist", "streamid0", "streamid1"})
	// logFatalOnError(err)
	// log.Println(streamsunnest)
	// log.Println(len(streamsunnest))
	// for _, elem := range streamsunnest {
	// 	log.Println(isNonZero(elem))
	// 	log.Println(elem.ID.Valid)
	// }
	// log.Print()

	logFatalOnError(queries.DeleteStreams(context.Background()))
}
