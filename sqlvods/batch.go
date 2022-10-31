// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0
// source: batch.go

package sqlvods

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

const getStreamForEachStreamIdBatched = `-- name: GetStreamForEachStreamIdBatched :batchmany
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM 
  streams
WHERE
  stream_id = $1
LIMIT 1
`

type GetStreamForEachStreamIdBatchedBatchResults struct {
	br     pgx.BatchResults
	tot    int
	closed bool
}

type GetStreamForEachStreamIdBatchedRow struct {
	ID                   uuid.UUID
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

func (q *Queries) GetStreamForEachStreamIdBatched(ctx context.Context, streamID []string) *GetStreamForEachStreamIdBatchedBatchResults {
	batch := &pgx.Batch{}
	for _, a := range streamID {
		vals := []interface{}{
			a,
		}
		batch.Queue(getStreamForEachStreamIdBatched, vals...)
	}
	br := q.db.SendBatch(ctx, batch)
	return &GetStreamForEachStreamIdBatchedBatchResults{br, len(streamID), false}
}

func (b *GetStreamForEachStreamIdBatchedBatchResults) Query(f func(int, []GetStreamForEachStreamIdBatchedRow, error)) {
	defer b.br.Close()
	for t := 0; t < b.tot; t++ {
		var items []GetStreamForEachStreamIdBatchedRow
		if b.closed {
			if f != nil {
				f(t, items, errors.New("batch already closed"))
			}
			continue
		}
		err := func() error {
			rows, err := b.br.Query()
			defer rows.Close()
			if err != nil {
				return err
			}
			for rows.Next() {
				var i GetStreamForEachStreamIdBatchedRow
				if err := rows.Scan(
					&i.ID,
					&i.LastUpdatedAt,
					&i.MaxViews,
					&i.StartTime,
					&i.StreamerID,
					&i.StreamID,
					&i.StreamerLoginAtStart,
				); err != nil {
					return err
				}
				items = append(items, i)
			}
			return rows.Err()
		}()
		if f != nil {
			f(t, items, err)
		}
	}
}

func (b *GetStreamForEachStreamIdBatchedBatchResults) Close() error {
	b.closed = true
	return b.br.Close()
}
