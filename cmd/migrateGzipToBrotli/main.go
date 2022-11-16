package main

import (
	"bytes"
	"context"
	"fmt"
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
	id          uuid.UUID
	brotliBytes []byte
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
	log.Println(fmt.Sprint("Retrieving bytes from ", databaseUrl))
	rows, err := queries.GetAllGzippedBytes(context.Background())
	log.Println(fmt.Sprint("Got ", len(rows), " rows."))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Decompress and write bytes")
	requestChan := make(chan sqlvods.GetAllGzippedBytesRow)
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
			dc, err := libdeflate.NewDecompressor()
			if err != nil {
				log.Fatal(err)
			}
			for {
				request := <-requestChan
				gzipBytes := request.GzippedBytes
				if gzipBytes == nil {
					responseChan <- TResponse{id: request.ID, brotliBytes: nil}
					continue
				}
				_, fullBytes, err := dc.Decompress(gzipBytes, nil, libdeflate.ModeGzip)
				if err != nil {
					log.Fatal(err)
				}
				buffer := &bytes.Buffer{}
				writer := brotli.NewWriterLevel(buffer, 2)
				_, err = writer.Write(fullBytes)
				if err != nil {
					log.Fatal(err)
				}
				writer.Close()
				brotliBytes := buffer.Bytes()
				responseChan <- TResponse{id: request.ID, brotliBytes: brotliBytes}
			}
		}()
	}
	idArr := []uuid.UUID{}
	brotliBytesArr := [][]byte{}
	for i := 0; i < numRows; i++ {
		if i%10000 == 0 {
			log.Println(fmt.Sprint("On row ", i))
		}
		response := <-responseChan
		idArr = append(idArr, response.id)
		brotliBytesArr = append(brotliBytesArr, response.brotliBytes)
	}
	log.Println("Got all brotli bytes")
	err = queries.SetBrotliBytes(context.Background(), sqlvods.SetBrotliBytesParams{IDArr: idArr, BrotliBytesArr: brotliBytesArr})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("It seems like we were successful")
}
