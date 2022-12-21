package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

var ErrParse = errors.New("must contain @")

func parseParam(param string) (string, error) {
	if len(param) == 0 || param[0] != '@' {
		return "", ErrParse
	}
	return param[1:], nil
}

func makeAddCorsMiddleare(clientUrl string) func(httprouter.Handle) httprouter.Handle {
	return func(f httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			w.Header().Set("Access-Control-Allow-Origin", clientUrl)
			f(w, r, p)
		}
	}
}

func okHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func makeMostViewedHandler(ctx context.Context, queries *sqlvods.Queries) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		results, err := queries.GetPopularLiveStreams(ctx, sqlvods.GetPopularLiveStreamsParams{
			Public:  sql.NullBool{Bool: p.ByName("pub-status") == "public", Valid: true},
			SubOnly: sql.NullBool{Bool: p.ByName("sub-status") == "sub", Valid: true},
			Limit:   50,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(results) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		streamResults := []TStreamResult{}
		for _, stream := range results {
			streamResults = append(streamResults, TStreamResult{
				Metadata: (*sqlvods.GetLatestStreamsFromStreamerLoginRow)(stream),
				Link: fmt.Sprint("/m3u8/", stream.StreamID, "/",
					stream.StartTime.Unix(), "/index.m3u8"),
			})
		}
		bytes, err := json.Marshal(streamResults)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

func makeAllLanguageHandler(ctx context.Context, queries *sqlvods.Queries) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		language, err := parseParam(p.ByName("language"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		results, err := queries.GetPopularLiveStreamsByLanguage(ctx, sqlvods.GetPopularLiveStreamsByLanguageParams{
			LanguageAtStart: language,
			Public:          sql.NullBool{Bool: p.ByName("pub-status") == "public", Valid: true},
			SubOnly:         sql.NullBool{Bool: p.ByName("sub-status") == "sub", Valid: true},
			Limit:           50,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(results) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		streamResults := []TStreamResult{}
		for _, stream := range results {
			streamResults = append(streamResults, TStreamResult{
				Metadata: (*sqlvods.GetLatestStreamsFromStreamerLoginRow)(stream),
				Link: fmt.Sprint("/m3u8/", stream.StreamID, "/",
					stream.StartTime.Unix(), "/index.m3u8"),
			})
		}
		bytes, err := json.Marshal(streamResults)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

func makeAllCategoryHandler(ctx context.Context, queries *sqlvods.Queries) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		categoryId, err := parseParam(p.ByName("game-id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		results, err := queries.GetPopularLiveStreamsByGameId(ctx, sqlvods.GetPopularLiveStreamsByGameIdParams{
			GameIDAtStart: categoryId,
			Public:        sql.NullBool{Bool: p.ByName("pub-status") == "public", Valid: true},
			SubOnly:       sql.NullBool{Bool: p.ByName("sub-status") == "sub", Valid: true},
			Limit:         50,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(results) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		streamResults := []TStreamResult{}
		for _, stream := range results {
			streamResults = append(streamResults, TStreamResult{
				Metadata: (*sqlvods.GetLatestStreamsFromStreamerLoginRow)(stream),
				Link: fmt.Sprint("/m3u8/", stream.StreamID, "/",
					stream.StartTime.Unix(), "/index.m3u8"),
			})
		}
		bytes, err := json.Marshal(streamResults)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

func makeStreamerHandler(ctx context.Context, queries *sqlvods.Queries) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		name, err := parseParam(p.ByName("streamer"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		streams, err := queries.GetLatestStreamsFromStreamerLogin(ctx, sqlvods.GetLatestStreamsFromStreamerLoginParams{StreamerLoginAtStart: name, Limit: 50})
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
			streamResults = append(streamResults, TStreamResult{
				Metadata: stream,
				Link:     fmt.Sprint("/m3u8/", stream.StreamID, "/", stream.StartTime.Unix(), "/index.m3u8"),
			})
		}
		bytes, err := json.Marshal(streamResults)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

func makeM3U8Handler(ctx context.Context, queries *sqlvods.Queries) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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
		bytes := streams[0]
		if bytes == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(bytes)
	}
}

type TStreamResult struct {
	Link     string
	Metadata *sqlvods.GetLatestStreamsFromStreamerLoginRow
}

func main() {
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		databaseUrl = "postgresql://govods:password@localhost:5432/twitch"
	}
	clientUrl, ok := os.LookupEnv("CLIENT_URL")
	if !ok {
		log.Fatal("CLIENT_URL is missing for CORS")
	}
	addCors := makeAddCorsMiddleare(clientUrl)
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

	// pub-status: either public or private
	// sub-status: either sub or free
	router.GET("/", addCors(okHandler))
	router.GET("/all/:pub-status/:sub-status", addCors(makeMostViewedHandler(ctx, queries)))
	router.GET("/channels/:streamer", addCors(makeStreamerHandler(ctx, queries)))
	router.GET("/m3u8/:streamid/:unix/index.m3u8", addCors(makeM3U8Handler(ctx, queries)))
	router.GET("/language/:language/all/:pub-status/:sub-status", addCors(makeAllLanguageHandler(ctx, queries)))
	router.GET("/category/:game-id/all/:pub-status/:sub-status", addCors(makeAllCategoryHandler(ctx, queries)))
	http.ListenAndServe(":3000", router)
}
