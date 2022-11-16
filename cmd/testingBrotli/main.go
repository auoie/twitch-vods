package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/andybalholm/brotli"
)

var (
	levelPointer = flag.Int("LEVEL", 2, "brotli compression level")
)

func testWriting() {
	flag.Parse()
	compressionLevel := *levelPointer
	buffer := &bytes.Buffer{}
	writer := brotli.NewWriterLevel(buffer, compressionLevel)
	inputBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	_, err = writer.Write(inputBytes)
	if err != nil {
		log.Fatal(err)
	}
	err = writer.Close()
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(buffer.Bytes())
}

func testReading() {
	inputBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	br := bytes.NewReader(inputBytes)
	decompressor := brotli.NewReader(br)
	decompressed, err := ioutil.ReadAll(decompressor)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(decompressed))
	fmt.Println("done writing")
}

func main() {
	testReading()
}
