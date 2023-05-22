package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/nicklaw5/helix"
)

func makeRobustHttpClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
	}
	transport := &http.Transport{DialContext: dialer.DialContext}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

type RateLimitHeaders struct {
	RateLimitRemaining string
	RateLimitLimit     string
	RateLimitReset     string
}

func prettyPrint(v any) {
	bytes, _ := json.MarshalIndent(v, "", "    ")
	fmt.Println(string(bytes))
}
func getRateLimitHeaders(headers http.Header) RateLimitHeaders {
	return RateLimitHeaders{
		RateLimitRemaining: headers.Get("RateLimit-Remaining"),
		RateLimitLimit:     headers.Get("RateLimit-Limit"),
		RateLimitReset:     headers.Get("RateLimit-Reset"),
	}
}
func resetAppAccessToken(client *helix.Client) error {
	appAccessToken, err := client.RequestAppAccessToken([]string{})
	if err != nil {
		return err
	}
	client.SetAppAccessToken(appAccessToken.Data.AccessToken)
	return nil
}
func min[T int64 | int](x, y T) T {
	if x < y {
		return x
	} else {
		return y
	}
}
func safeSlice[T any](arr []T, max int) []T {
	return arr[:min(len(arr), max)]
}
func main() {
	clientId, ok := os.LookupEnv("CLIENT_ID")
	if !ok {
		log.Fatal("CLIENT_ID is missing for twitch helix API")
	}
	clientSecret, ok := os.LookupEnv("CLIENT_SECRET")
	if !ok {
		log.Fatal("CLIENT_SECRET is missing for twitch helix API")
	}
	client, err := helix.NewClient(&helix.Options{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		HTTPClient:   makeRobustHttpClient(time.Second * 10),
	})
	if err != nil {
		log.Fatal(err)
	}
	if err = resetAppAccessToken(client); err != nil {
		log.Fatal(err)
	}
	timer := time.NewTicker(300 * time.Millisecond)
	pagination := ""
	cursorResetTicker := time.NewTicker(2 * 2 * time.Second)
	for {
		select {
		case <-cursorResetTicker.C:
			fmt.Println("resetting cursor")
			resetAppAccessToken(client)
			resetAppAccessToken(client)
			pagination = ""
		case <-timer.C:
			resp, err := client.GetStreams(&helix.StreamsParams{
				First: 100,
				After: pagination,
			})
			if err != nil {
				log.Fatal(err)
			}
			pagination = resp.Data.Pagination.Cursor
			headers := getRateLimitHeaders(resp.Header)
			fmt.Println(headers)
			if len(resp.Data.Streams) == 0 {
				continue
			}
			stream := resp.Data.Streams[0]
			prettyPrint(stream)
			usersResponse, err := client.GetUsers(&helix.UsersParams{IDs: []string{stream.UserID}})
			if err != nil {
				log.Fatal(err)
			}
			prettyPrint(safeSlice(usersResponse.Data.Users, 3))
			videoResponse, err := client.GetVideos(&helix.VideosParams{UserID: stream.UserID})
			if err != nil {
				log.Fatal(err)
			}
			prettyPrint(safeSlice(videoResponse.Data.Videos, 3))
			gamesResponse, err := client.GetGames(&helix.GamesParams{IDs: []string{stream.GameID}})
			if err != nil {
				log.Fatal(err)
			}
			prettyPrint(safeSlice(gamesResponse.Data.Games, 3))
		}
	}
}
