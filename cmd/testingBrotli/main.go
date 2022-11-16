package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"os"

	"github.com/andybalholm/brotli"
)

var (
	levelPointer = flag.Int("LEVEL", 2, "brotli compression level")
)

func main() {
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
