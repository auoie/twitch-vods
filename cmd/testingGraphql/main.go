package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/auoie/twitch-vods/twitchgql"
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
					log.Println(err)
					continue
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
func retryOnError[T any](doer func() (T, error)) (T, error) {
	res, err := doer()
	if err != nil {
		log.Println(fmt.Sprint("retrying on error: ", err))
		return doer()
	}
	return res, err
}
func threeUsersInformation(graphqlClient graphql.Client, user1, user2, user3 string) error {
	response, err := retryOnError(func() (*twitchgql.GetThreeUsersResponse, error) {
		return twitchgql.GetThreeUsers(context.Background(), graphqlClient, user1, user2, user3)
	})
	if err != nil {
		return err
	}
	startTime := response.User2.Videos.Edges[0].Node.CreatedAt
	fmt.Println(startTime)
	fmt.Println(startTime.Location())
	fmt.Println(startTime == startTime.UTC())
	fmt.Println(time.Now().UTC().Location())
	print(response)
	user1data, _ := twitchgql.GetUserData(context.Background(), graphqlClient, response.User1.Id)
	print(user1data)
	return nil
}

func main() {
	fmt.Println("Running...")
	graphqlClient := twitchgql.NewTwitchGqlClient(5 * time.Second)
	err := threeUsersInformation(graphqlClient, "gmhikaru", "stoopzz", "duke")
	if err != nil {
		log.Fatal(err)
	}
	runForever(graphqlClient)
}
