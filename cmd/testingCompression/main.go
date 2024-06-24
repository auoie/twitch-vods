package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/4kills/go-libdeflate/v2"
	"github.com/klauspost/compress/zstd"
)

var (
	levelPointer = flag.Int("LEVEL", 6, "libdeflate compression level")
)

func gzip_compress() {
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
	// os.Stdout.Write(comp)
	fmt.Println(len(comp))
}

func getCompressedBytes(bytes []byte, compressor *zstd.Encoder) []byte {
	return compressor.EncodeAll(bytes, nil)
}

func zstd_decompress() {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		log.Fatal(err)
	}
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	result, err := decoder.DecodeAll(bytes, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(result))
}

func zstd_compress() {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		log.Fatal(err)
	}
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	result := getCompressedBytes(bytes, encoder)
	fmt.Println(len(result))
}

func decomp_zstd_compress_gzip() {
	db_bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	decompressor, err := zstd.NewReader(nil)
	if err != nil {
		log.Fatal(err)
		return
	}
	m3u8_bytes, err := decompressor.DecodeAll(db_bytes, nil)
	defer decompressor.Close()
	if err != nil {
		fmt.Println("decompression error")
		fmt.Println(err)
		fmt.Println()
		fmt.Println(string(db_bytes))
		return
	}
	gzip_buf := bytes.Buffer{}
	compressor := gzip.NewWriter(&gzip_buf)
	_, err = compressor.Write(m3u8_bytes)
	if err != nil {
		log.Fatal(err)
		return
	}
	err = compressor.Close()
	if err != nil {
		log.Fatal(err)
		return
	}
	gzip_bytes := gzip_buf.Bytes()
	os.Stdout.Write(gzip_bytes)
}

func main() {
	// zstd_compress()
	// gzip_compress()
	// zstd_decompress()
	decomp_zstd_compress_gzip()
}
