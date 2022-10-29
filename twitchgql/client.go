package twitchgql

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

const CLIENT_ID = "kimne78kx3ncx6brgo4mv6wki5h1ko"

type VodNode *GetStreamsStreamsStreamConnectionEdgesStreamEdgeNodeStream

type authedTransport struct {
	clientID string
	wrapped  http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Client-ID", t.clientID)
	return t.wrapped.RoundTrip(req)
}

func NewTwitchClient() *http.Client {
	return &http.Client{
		Transport: &authedTransport{
			clientID: CLIENT_ID,
			wrapped:  http.DefaultTransport,
		},
	}
}

func NewTwitchGqlClient() graphql.Client {
	httpClient := NewTwitchClient()
	graphqlClient := graphql.NewClient("https://gql.twitch.tv/gql", httpClient)
	return graphqlClient
}

//go:generate go run github.com/Khan/genqlient genqlient.yaml
