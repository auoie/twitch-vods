package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/auoie/goVods/twitchgql"
)

func main() {
	fmt.Println("Running...")
	graphqlClient := twitchgql.NewTwitchGqlClient()
	done := make(chan error)
	count := 0
	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		goForever := func() error {
			cursor := ""
			for {
				response, err := twitchgql.GetStreams(context.Background(), graphqlClient, 30, cursor)
				if err != nil {
					return err
				}
				streams := response.Streams
				edges := streams.Edges
				if len(edges) == 0 {
					return err
				}
				bytes, err := json.MarshalIndent(streams, "", "  ")
				if err != nil {
					return err
				}
				<-ticker.C
				fmt.Println(string(bytes))
				fmt.Println("Count:", count)
				fmt.Println(time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"))
				count++
				lastCursor := edges[len(edges)-1].Cursor
				if lastCursor == "" {
					return errors.New("last cursor is empty")
				}
				cursor = lastCursor
			}
		}
		done <- goForever()
	}()
	err := <-done
	fmt.Print(err)
}
