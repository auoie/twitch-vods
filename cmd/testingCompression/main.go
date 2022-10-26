package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/4kills/go-libdeflate/v2"
)

var (
	levelPointer = flag.Int("LEVEL", 6, "libdeflate compression level")
)

func main() {
	flag.Parse()
	compressionLevel := *levelPointer
	c, err := libdeflate.NewCompressorLevel(compressionLevel)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	comp := make([]byte, len(bytes))
	n, _, err := c.Compress(bytes, comp, libdeflate.ModeGzip)
	if err != nil {
		log.Fatal(err)
	}
	comp = comp[:n]
	os.Stdout.Write(comp)
}
