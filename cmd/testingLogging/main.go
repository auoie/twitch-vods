package main

import (
	"fmt"
	"log"
)

func main() {
	i := 3
	log.Println(i)
	log.Println(&i)
	log.Println(fmt.Sprint(&i))
}
