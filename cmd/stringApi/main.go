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
	"sync"
	"time"

	"github.com/auoie/twitch-vods/sqlvods"
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

func okHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
}

func bongHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.Write([]byte("bong"))
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
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(bytes)
	}
}

func makeCategoriesListHandler(categoriesLock *LockValue[[]*sqlvods.GetPopularCategoriesRow]) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		popularCategories := categoriesLock.Get()
		bytes, err := json.Marshal(popularCategories)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

func makeLanguagesListHandler(languagesLock *LockValue[[]*sqlvods.GetLanguagesRow]) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		popularLanguages := languagesLock.Get()
		bytes, err := json.Marshal(popularLanguages)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

type TStreamResult struct {
	Link     string
	Metadata *sqlvods.GetLatestStreamsFromStreamerLoginRow
}

type CustomHandler struct {
	router    *httprouter.Router
	clientUrl string
}

func (ch *CustomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", ch.clientUrl)
	ch.router.ServeHTTP(w, r)
}

type LockValue[T any] struct {
	value T
	sync.RWMutex
}

func (lv *LockValue[T]) Get() T {
	lv.RLock()
	defer lv.RUnlock()
	return lv.value
}

func (lv *LockValue[T]) Set(value T) {
	lv.Lock()
	defer lv.Unlock()
	lv.value = value
}

func initApp(ctx context.Context) (string, *sqlvods.Queries, *httprouter.Router, *CustomHandler) {
	// init app
	port, ok := os.LookupEnv("PORT")
	if !ok {
		log.Fatal("PORT is missing to listen on")
	}
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		log.Fatal("DATABASE_URL is missing for db connection string")
	}
	clientUrl, ok := os.LookupEnv("CLIENT_URL")
	if !ok {
		log.Fatal("CLIENT_URL is missing for CORS")
	}
	conn, err := pgxpool.Connect(ctx, databaseUrl)
	if err != nil {
		log.Println(fmt.Sprint("failed to connect to ", databaseUrl, ": ", err))
		log.Fatal(err)
	}
	defer conn.Close()
	err = conn.Ping(ctx)
	if err != nil {
		log.Println(fmt.Sprint("failed to ping ", databaseUrl, ": ", err))
		log.Fatal(err)
	}
	queries := sqlvods.New(conn)
	router := httprouter.New()
	handler := &CustomHandler{router: router, clientUrl: clientUrl}
	return port, queries, router, handler
}

func main() {
	ctx := context.Background()
	port, queries, router, handler := initApp(ctx)

	// should use rabbitmq or apache kafka instead of polling every hour
	categoriesLock := &LockValue[[]*sqlvods.GetPopularCategoriesRow]{}
	setPopularCategories := func() {
		log.Println("Fetching categories")
		categories, err := queries.GetPopularCategories(ctx, 200)
		if err == nil {
			categoriesLock.Set(categories)
			log.Println("Set categories")
		} else {
			log.Println("Failed to set categories")
		}
	}
	go func(ctx context.Context) {
		interval := time.NewTicker(1 * time.Hour)
		setPopularCategories()
		for {
			select {
			case <-interval.C:
				setPopularCategories()
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	languagesLock := &LockValue[[]*sqlvods.GetLanguagesRow]{}
	setLanguages := func() {
		log.Println("Fetching languages")
		languages, err := queries.GetLanguages(ctx)
		if err == nil {
			languagesLock.Set(languages)
			log.Println("Set languages")
		} else {
			log.Println("Failed to set languages")
		}
	}
	go func(ctx context.Context) {
		interval := time.NewTicker(1 * time.Hour)
		setLanguages()
		for {
			select {
			case <-interval.C:
				setLanguages()
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	// pub-status: either public or private
	// sub-status: either sub or free
	router.GET("/", okHandler)
	router.GET("/bing", bongHandler)
	router.GET("/all/:pub-status/:sub-status", makeMostViewedHandler(ctx, queries))
	router.GET("/channels/:streamer", makeStreamerHandler(ctx, queries))
	router.GET("/m3u8/:streamid/:unix/index.m3u8", makeM3U8Handler(ctx, queries))
	router.GET("/language/:language/all/:pub-status/:sub-status", makeAllLanguageHandler(ctx, queries))
	router.GET("/category/:game-id/all/:pub-status/:sub-status", makeAllCategoryHandler(ctx, queries))
	router.GET("/categories", makeCategoriesListHandler(categoriesLock))
	router.GET("/languages", makeLanguagesListHandler(languagesLock))
	log.Println(fmt.Sprint("Serving on port :", port))

	http.ListenAndServe(fmt.Sprint(":", port), handler)
}
