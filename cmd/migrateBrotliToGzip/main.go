package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/4kills/go-libdeflate/v2"
	"github.com/andybalholm/brotli"
	"github.com/auoie/goVods/sqlvods"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

const N = 12

type TResponse struct {
	id        uuid.UUID
	gzipBytes []byte
}

func getCompressedBytes(bytes []byte, compressor *libdeflate.Compressor) ([]byte, error) {
	compressedBytes := make([]byte, len(bytes))
	n, _, err := compressor.Compress(bytes, compressedBytes, libdeflate.ModeGzip)
	if err != nil {
		return nil, err
	}
	compressedBytes = compressedBytes[:n]
	return compressedBytes, nil
}

func main() {
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		databaseUrl = "postgresql://govods:password@localhost:5432/twitch"
	}
	ctx := context.Background()
	conn, err := pgxpool.Connect(ctx, databaseUrl)
	if err != nil {
		log.Println(fmt.Sprint("failed to connect to ", databaseUrl, ": ", err))
		log.Fatal(err)
	}
	err = conn.Ping(ctx)
	if err != nil {
		log.Println(fmt.Sprint("failed to ping ", databaseUrl, ": ", err))
		conn.Close()
		log.Fatal(err)
	}
	queries := sqlvods.New(conn)
	log.Println(fmt.Sprint("Retrieving brotli bytes from ", databaseUrl))
	rows, err := queries.GetAllBrotliBytesNotGzip(context.Background())
	log.Println(fmt.Sprint("Got ", len(rows), " brotli rows."))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Decompress and write bytes")
	requestChan := make(chan sqlvods.GetAllBrotliBytesNotGzipRow)
	responseChan := make(chan TResponse)
	numRows := len(rows)
	// numRows := 50000
	go func() {
		for i := 0; i < numRows; i++ {
			row := rows[i]
			requestChan <- row
		}
	}()
	for i := 0; i < N; i++ {
		go func() {
			compressor, err := libdeflate.NewCompressorLevel(1)
			if err != nil {
				log.Fatal(err)
			}
			for {
				request := <-requestChan
				brotliBytes := request.BrotliBytes
				if brotliBytes == nil {
					responseChan <- TResponse{id: request.ID, gzipBytes: nil}
					continue
				}
				br := bytes.NewReader(brotliBytes)
				decompressor := brotli.NewReader(br)
				decompressed, err := io.ReadAll(decompressor)
				if err != nil {
					log.Fatal(err)
				}
				gzipBytes, err := getCompressedBytes(decompressed, &compressor)
				if err != nil {
					log.Fatal(err)
				}
				responseChan <- TResponse{id: request.ID, gzipBytes: gzipBytes}
			}
		}()
	}
	idArr := []uuid.UUID{}
	gzipBytesArr := [][]byte{}
	for i := 0; i < numRows; i++ {
		if i%10000 == 0 {
			log.Println(fmt.Sprint("On row ", i))
		}
		response := <-responseChan
		idArr = append(idArr, response.id)
		gzipBytesArr = append(gzipBytesArr, response.gzipBytes)
	}
	log.Println("Got all gzip bytes")
	err = queries.SetGzipBytes(context.Background(), sqlvods.SetGzipBytesParams{IDArr: idArr, GzipBytesArr: gzipBytesArr})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("It seems like we were successful")
}
