package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/auoie/twitchVods/sqlvods"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/julienschmidt/httprouter"
)

type TStreamResult struct {
	Link     string
	Metadata sqlvods.GetLatestStreamsFromStreamerLoginRow
}

func main() {
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		databaseUrl = "postgresql://govods:password@localhost:5432/twitch"
	}
	ctx := context.Background()
	conn, err := pgxpool.Connect(ctx, databaseUrl)
	if err != nil {
		log.Println(fmt.Sprint("failed to connect to ", databaseUrl, ": ", err))
		log.Fatal(err)
	}
	err = conn.Ping(ctx)
	if err != nil {
		log.Println(fmt.Sprint("failed to ping ", databaseUrl, ": ", err))
		conn.Close()
		log.Fatal(err)
	}
	queries := sqlvods.New(conn)
	router := httprouter.New()
	router.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3001")
		w.WriteHeader(http.StatusOK)
	})
	router.GET("/highest_viewed_private_available", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3001")
		results, err := queries.GetHighestViewedLiveStreams(ctx, sqlvods.GetHighestViewedLiveStreamsParams{
			BytesFound:      sql.NullBool{Bool: true, Valid: true},
			Public:          sql.NullBool{Bool: false, Valid: true},
			LanguageAtStart: "EN",
			Limit:           100,
		})
		if err != nil {
			w.WriteHeader(500)
			return
		}
		bytes, err := json.Marshal(results)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	})
	router.GET("/channels/:streamer", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3001")
		name := p.ByName("streamer")
		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		streams, err := queries.GetLatestStreamsFromStreamerLogin(ctx, sqlvods.GetLatestStreamsFromStreamerLoginParams{StreamerLoginAtStart: name, Limit: 100})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(streams) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		streamResults := []TStreamResult{}
		for _, stream := range streams {
			streamResults = append(streamResults, TStreamResult{Metadata: stream, Link: fmt.Sprint("/m3u8/", stream.StreamID, "/", stream.StartTime.Unix(), "/index.m3u8")})
		}
		bytes, err := json.Marshal(streamResults)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	})
	router.GET("/m3u8/:streamid/:unix/index.m3u8", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		streamid := p.ByName("streamid")
		if streamid == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		unix := p.ByName("unix")
		if unix == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		unix_int, err := strconv.ParseInt(unix, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		streams, err := queries.GetStreamGzippedBytes(ctx, sqlvods.GetStreamGzippedBytesParams{
			StreamID:  streamid,
			StartTime: time.Unix(unix_int, 0).UTC(),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(streams) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		streamBytes := streams[0]
		if streamBytes == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(streamBytes)
	})
	http.ListenAndServe(":3000", router)
}
