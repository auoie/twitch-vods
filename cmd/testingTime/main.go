package main

import (
	"fmt"
	"time"
)

func mul(dur time.Duration, val float64) time.Duration {
	return time.Duration(float64(dur) * val)
}

func main() {
	timeThing := 100 * time.Millisecond
	fmt.Println(mul(timeThing, 1.1))
}
