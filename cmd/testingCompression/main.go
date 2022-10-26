package main

import (
	"flag"
	"fmt"
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
	compressor, err := libdeflate.NewCompressorLevel(compressionLevel)
	if err != nil {
		log.Fatal(err)
	}
	defer compressor.Close()
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	comp := make([]byte, len(bytes))
	n, _, err := compressor.Compress(bytes, comp, libdeflate.ModeGzip)
	if err != nil {
		log.Fatal(err)
	}
	comp = comp[n:]
	fmt.Print(comp)
}
