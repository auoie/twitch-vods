package twitchgql

import (
	"net"
	"net/http"
	"time"

	"github.com/Khan/genqlient/graphql"
)

const CLIENT_ID = "kimne78kx3ncx6brgo4mv6wki5h1ko"

type VodNode *GetStreamsStreamsStreamConnectionEdgesStreamEdgeNodeStream

type authedTransport struct {
	clientID string
	wrapped  *http.Transport
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header != nil {
		req.Header.Set("Client-ID", t.clientID)
	}
	return t.wrapped.RoundTrip(req)
}

func newTwitchClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
	}
	transport := &http.Transport{DialContext: dialer.DialContext}
	return &http.Client{
		Timeout:   timeout,
		Transport: &authedTransport{clientID: CLIENT_ID, wrapped: transport},
	}
}

func NewTwitchGqlClient(timeout time.Duration) graphql.Client {
	httpClient := newTwitchClient(timeout)
	graphqlClient := graphql.NewClient("https://gql.twitch.tv/gql", httpClient)
	return graphqlClient
}

//go:generate go run github.com/Khan/genqlient genqlient.yaml
