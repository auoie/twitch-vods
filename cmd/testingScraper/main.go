package main

import (
	"context"
	"time"

	"github.com/auoie/goVods/scraper"
)

func main() {
	scraper.ScrapeTwitchLiveVodsWithGqlApi(
		context.Background(),
		333*time.Millisecond,
		5*time.Second,
		10*time.Minute,
		100000,
		5,
		time.Second,
		5*time.Minute,
		1,
		5,
	)
}
