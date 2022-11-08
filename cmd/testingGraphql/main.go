package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/auoie/goVods/twitchgql"
)

func runForever(graphqlClient graphql.Client) error {
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
	return <-done
}

func print(response any) error {
	bytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))
	return nil
}

func threeUsersInformation(graphqlClient graphql.Client, user1, user2, user3 string) error {
	response, err := twitchgql.GetThreeUsers(context.Background(), graphqlClient, user1, user2, user3)
	if err != nil {
		return err
	}
	print(response)
	user1data, _ := twitchgql.GetUserData(context.Background(), graphqlClient, response.User1.Id)
	print(user1data)
	return nil
}

func main() {
	fmt.Println("Running...")
	graphqlClient := twitchgql.NewTwitchGqlClient()
	err := threeUsersInformation(graphqlClient, "gmhikaru", "theo", "goonergooch")
	if err != nil {
		log.Fatal(err)
	}
}
