package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/julienschmidt/httprouter"
)

func bad() {
	router := httprouter.New()
	count := 0
	router.GET("/get", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Write([]byte(strconv.Itoa(count)))
	})
	router.GET("/incr", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		count++
		w.Write([]byte(strconv.Itoa(count)))
	})
	fmt.Println("running at http://localhost:4000")
	http.ListenAndServe(":4000", router)
}

type tRequest struct {
	method     int // 0 is get, 1 is incr
	responseCh chan int
}

func channel(ctx context.Context) {
	router := httprouter.New()
	requestCh := make(chan tRequest)
	go func() {
		count := 0
		for {
			select {
			case request := <-requestCh:
				if request.method == 0 {
					select {
					case <-ctx.Done():
						return
					case request.responseCh <- count:
					}
				} else {
					count++
					select {
					case <-ctx.Done():
					case request.responseCh <- count:
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	router.GET("/get", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		responseCh := make(chan int)
		select {
		case <-ctx.Done():
			w.WriteHeader(http.StatusInternalServerError)
			return
		case requestCh <- tRequest{method: 0, responseCh: responseCh}:
		}
		select {
		case <-ctx.Done():
			w.WriteHeader(http.StatusInternalServerError)
			return
		case val := <-responseCh:
			w.Write([]byte(strconv.Itoa(val)))
		}
	})
	router.GET("/incr", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		responseCh := make(chan int)
		select {
		case <-ctx.Done():
			w.WriteHeader(http.StatusInternalServerError)
			return
		case requestCh <- tRequest{method: 1, responseCh: responseCh}:
		}
		select {
		case <-ctx.Done():
			w.WriteHeader(http.StatusInternalServerError)
			return
		case val := <-responseCh:
			w.Write([]byte(strconv.Itoa(val)))
		}
	})
	fmt.Println("running at http://localhost:4001")
	http.ListenAndServe(":4001", router)
}

func lock() {
	router := httprouter.New()
	count := 0
	countLock := sync.RWMutex{}
	getCount := func() int {
		countLock.RLock()
		defer countLock.RUnlock()
		return count
	}
	incrCount := func() int {
		countLock.Lock()
		defer countLock.Unlock()
		count++
		return count
	}
	router.GET("/get", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Write([]byte(strconv.Itoa(getCount())))
	})
	router.GET("/incr", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Write([]byte(strconv.Itoa(incrCount())))
	})
	fmt.Println("running at http://localhost:4002")
	http.ListenAndServe(":4002", router)
}

func main() {
	go func() {
		bad()
	}()
	go func() {
		lock()
	}()
	channel(context.Background())
}
