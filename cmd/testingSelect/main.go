package main

import "fmt"

func main() {
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})
	f1 := func() chan struct{} {
		fmt.Println("f1")
		return ch1
	}
	f2 := func() chan struct{} {
		fmt.Println("f2")
		return ch2
	}
	go func() {
		<-ch1
	}()
	go func() {
		<-ch2
	}()
	select {
	case f1() <- struct{}{}:
		fmt.Println("Case f1")
	case f2() <- struct{}{}:
		fmt.Println("Case f2")
	}
}
