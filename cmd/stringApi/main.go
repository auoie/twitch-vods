package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/auoie/twitch-vods/sqlvods"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/klauspost/compress/zstd"
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

// It would be nice if SQLC created methods to access struct field like Genqlient
func resultsGetPopularLiveStreams(ctx context.Context, w http.ResponseWriter, p httprouter.Params, queries *sqlvods.Queries) ([]*sqlvods.GetPopularLiveStreamsRow, error, bool) {
	results, err := queries.GetPopularLiveStreams(ctx, sqlvods.GetPopularLiveStreamsParams{
		Public: sql.NullBool{Bool: p.ByName("pub-status") == "public", Valid: true},
		Limit:  50,
	})
	return results, err, false
}
func linkGetPopularLiveStreams(stream *sqlvods.GetPopularLiveStreamsRow) string {
	return fmt.Sprint("/m3u8/", stream.StreamID, "/", stream.StartTime.Unix(), "/index.m3u8")
}

func resultsGetPopularLiveStreamsByLanguage(ctx context.Context, w http.ResponseWriter, p httprouter.Params, queries *sqlvods.Queries) ([]*sqlvods.GetPopularLiveStreamsByLanguageRow, error, bool) {
	language, err := parseParam(p.ByName("language"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, nil, true
	}
	results, err := queries.GetPopularLiveStreamsByLanguage(ctx, sqlvods.GetPopularLiveStreamsByLanguageParams{
		LanguageAtStart: language,
		Public:          sql.NullBool{Bool: p.ByName("pub-status") == "public", Valid: true},
		Limit:           50,
	})
	return results, err, false
}
func linkGetPopularLiveStreamsByLanguage(stream *sqlvods.GetPopularLiveStreamsByLanguageRow) string {
	return fmt.Sprint("/m3u8/", stream.StreamID, "/", stream.StartTime.Unix(), "/index.m3u8")
}

func resultsGetPopularLiveStreamsByGameId(ctx context.Context, w http.ResponseWriter, p httprouter.Params, queries *sqlvods.Queries) ([]*sqlvods.GetPopularLiveStreamsByGameIdRow, error, bool) {
	categoryId, err := parseParam(p.ByName("game-id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, nil, true
	}
	results, err := queries.GetPopularLiveStreamsByGameId(ctx, sqlvods.GetPopularLiveStreamsByGameIdParams{
		GameIDAtStart: categoryId,
		Public:        sql.NullBool{Bool: p.ByName("pub-status") == "public", Valid: true},
		Limit:         50,
	})
	return results, err, false
}
func linkGetPopularLiveStreamsByGameId(stream *sqlvods.GetPopularLiveStreamsByGameIdRow) string {
	return fmt.Sprint("/m3u8/", stream.StreamID, "/", stream.StartTime.Unix(), "/index.m3u8")
}

func resultsGetLatestStreamsFromStreamerLogin(ctx context.Context, w http.ResponseWriter, p httprouter.Params, queries *sqlvods.Queries) ([]*sqlvods.GetLatestStreamsFromStreamerLoginRow, error, bool) {
	name, err := parseParam(p.ByName("streamer"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, nil, true
	}
	results, err := queries.GetLatestStreamsFromStreamerLogin(ctx, sqlvods.GetLatestStreamsFromStreamerLoginParams{
		StreamerLoginAtStart: name,
		Limit:                50,
	})
	return results, err, false
}
func linkGetLatestStreamsFromStreamerLogin(stream *sqlvods.GetLatestStreamsFromStreamerLoginRow) string {
	return fmt.Sprint("/m3u8/", stream.StreamID, "/", stream.StartTime.Unix(), "/index.m3u8")
}

func makeListHandler[T any](
	ctx context.Context,
	queries *sqlvods.Queries,
	getResults func(context.Context, http.ResponseWriter, httprouter.Params, *sqlvods.Queries) ([]T, error, bool),
	getLink func(T) string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		results, err, done := getResults(ctx, w, p, queries)
		if done {
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		streamResults := []TStreamResult[T]{}
		for _, stream := range results {
			streamResults = append(streamResults, TStreamResult[T]{
				Metadata: stream,
				Link:     getLink(stream),
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
		db_bytes := streams[0]
		if db_bytes == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		decompressor, err := zstd.NewReader(nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		m3u8_bytes, err := decompressor.DecodeAll(db_bytes, nil)
		defer decompressor.Close()
		if err != nil {
			w.Header().Set("Content-Length", strconv.Itoa(len(db_bytes)))
			w.Header().Set("Content-Type", "application/x-mpegURL")
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(db_bytes)
			return
		}
		gzip_buf := bytes.Buffer{}
		compressor := gzip.NewWriter(&gzip_buf)
		_, err = compressor.Write(m3u8_bytes)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = compressor.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		gzip_bytes := gzip_buf.Bytes()
		w.Header().Set("Content-Length", strconv.Itoa(len(gzip_bytes)))
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(gzip_bytes)
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
func swap[T any](vals []T, i, j int) {
	if i != j {
		vals[i], vals[j] = vals[j], vals[i]
	}
}

func makeSearchHandler(ctx context.Context, regexCheck *regexp.Regexp, queries *sqlvods.Queries) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		streamer := p.ByName("streamer")
		if !regexCheck.MatchString(streamer) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		results, err := queries.GetMatchingStreamers(ctx, sqlvods.GetMatchingStreamersParams{
			Limit:                  20,
			StreamerLoginAtStart:   streamer,
			StreamerLoginAtStart_2: fmt.Sprint("%", streamer, "%")})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if results == nil {
			results = []*sqlvods.GetMatchingStreamersRow{}
		}
		for i := 0; i < len(results); i++ {
			if streamer == results[i].StreamerLoginAtStart {
				swap(results, 0, i)
				break
			}
		}
		bytes, err := json.Marshal(results)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

type TStreamResult[T any] struct {
	Link     string
	Metadata T
}

type CustomHandler struct {
	router        *httprouter.Router
	clientUrl     string
	requestLogger *log.Logger
}

// GET registers a route and records its pattern and params for the request log.
func (ch *CustomHandler) GET(route string, handle httprouter.Handle) {
	ch.router.GET(route, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if rec, ok := w.(*responseRecorder); ok {
			rec.route = route
			rec.params = p
		}
		handle(w, r, p)
	})
}

func (ch *CustomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Access-Control-Allow-Origin", ch.clientUrl)
	rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
	ch.router.ServeHTTP(rec, r)
	ch.logRequest(r, rec, start)
}

type responseRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
	route       string
	params      httprouter.Params
}

func (rec *responseRecorder) WriteHeader(status int) {
	if !rec.wroteHeader {
		rec.status = status
		rec.wroteHeader = true
	}
	rec.ResponseWriter.WriteHeader(status)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	rec.wroteHeader = true
	n, err := rec.ResponseWriter.Write(b)
	rec.bytes += n
	return n, err
}

type requestLogEntry struct {
	Time       string            `json:"time"`
	IP         string            `json:"ip"`
	Country    string            `json:"country,omitempty"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Route      string            `json:"route,omitempty"`
	Params     map[string]string `json:"params,omitempty"`
	Status     int               `json:"status"`
	Bytes      int               `json:"bytes"`
	DurationMs float64           `json:"duration_ms"`
	Referer    string            `json:"referer,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
}

// In production, haproxy only accepts Cloudflare authenticated origin pulls, so
// CF-Connecting-IP can be trusted. Without Cloudflare (dev), fall back to the peer.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (ch *CustomHandler) logRequest(r *http.Request, rec *responseRecorder, start time.Time) {
	var params map[string]string
	if len(rec.params) > 0 {
		params = make(map[string]string, len(rec.params))
		for _, p := range rec.params {
			params[p.Key] = p.Value
		}
	}
	entry, err := json.Marshal(requestLogEntry{
		Time:       start.UTC().Format(time.RFC3339),
		IP:         clientIP(r),
		Country:    r.Header.Get("CF-IPCountry"),
		Method:     r.Method,
		Path:       r.URL.Path,
		Route:      rec.route,
		Params:     params,
		Status:     rec.status,
		Bytes:      rec.bytes,
		DurationMs: float64(time.Since(start).Microseconds()) / 1000,
		Referer:    r.Referer(),
		UserAgent:  r.UserAgent(),
	})
	if err != nil {
		log.Println(fmt.Sprint("failed to marshal request log: ", err))
		return
	}
	ch.requestLogger.Println(string(entry))
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

func main() {
	ctx := context.Background()
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
	// Request logs go to stdout as JSON lines; other logs stay on stderr.
	handler := &CustomHandler{router: router, clientUrl: clientUrl, requestLogger: log.New(os.Stdout, "", 0)}

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
	twitchUsernameRegex, err := regexp.Compile("^[a-zA-Z0-9_]{1,50}$")
	if err != nil {
		log.Fatal(fmt.Sprint("Failed to compile regex: ", err))
	}

	// pub-status: either public or private
	handler.GET("/", okHandler)
	handler.GET("/bing", bongHandler)
	handler.GET("/all/:pub-status", makeListHandler(ctx, queries, resultsGetPopularLiveStreams, linkGetPopularLiveStreams))
	handler.GET("/language/:language/all/:pub-status", makeListHandler(ctx, queries, resultsGetPopularLiveStreamsByLanguage, linkGetPopularLiveStreamsByLanguage))
	handler.GET("/category/:game-id/all/:pub-status", makeListHandler(ctx, queries, resultsGetPopularLiveStreamsByGameId, linkGetPopularLiveStreamsByGameId))
	handler.GET("/channels/:streamer", makeListHandler(ctx, queries, resultsGetLatestStreamsFromStreamerLogin, linkGetLatestStreamsFromStreamerLogin))
	handler.GET("/categories", makeCategoriesListHandler(categoriesLock))
	handler.GET("/languages", makeLanguagesListHandler(languagesLock))
	handler.GET("/search/:streamer", makeSearchHandler(ctx, twitchUsernameRegex, queries))
	handler.GET("/m3u8/:streamid/:unix/index.m3u8", makeM3U8Handler(ctx, queries))
	log.Println(fmt.Sprint("Serving on port :", port))

	http.ListenAndServe(fmt.Sprint(":", port), handler)
}
