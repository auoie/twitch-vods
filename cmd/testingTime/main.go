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
	thing := time.Now()
	thing2 := thing.UTC()
	fmt.Println(thing == thing2)
	fmt.Println(thing)
	fmt.Println(thing2)
	thing3 := time.Unix(0, 0).UTC()
	fmt.Println(thing3)
}
