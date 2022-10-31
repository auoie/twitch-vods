// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0
// source: queries.sql

package sqlvods

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type AddManyStreamsParams struct {
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

const addStream = `-- name: AddStream :exec
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start)
VALUES
  ($1, $2, $3, $4, $5, $6)
`

type AddStreamParams struct {
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

func (q *Queries) AddStream(ctx context.Context, arg AddStreamParams) error {
	_, err := q.db.Exec(ctx, addStream,
		arg.LastUpdatedAt,
		arg.MaxViews,
		arg.StartTime,
		arg.StreamerID,
		arg.StreamID,
		arg.StreamerLoginAtStart,
	)
	return err
}

const deleteRecordings = `-- name: DeleteRecordings :exec
DELETE FROM recordings
`

func (q *Queries) DeleteRecordings(ctx context.Context) error {
	_, err := q.db.Exec(ctx, deleteRecordings)
	return err
}

const deleteStreams = `-- name: DeleteStreams :exec
DELETE FROM streams
`

func (q *Queries) DeleteStreams(ctx context.Context) error {
	_, err := q.db.Exec(ctx, deleteStreams)
	return err
}

const getEverything = `-- name: GetEverything :many
SELECT
  s.id, streamer_id, stream_id, start_time, max_views, last_updated_at, streamer_login_at_start, r.id, fetched_at, gzipped_bytes, streams_id
FROM
  streams s
LEFT JOIN
  recordings r
ON
  s.id = r.streams_id
`

type GetEverythingRow struct {
	ID                   uuid.UUID
	StreamerID           string
	StreamID             string
	StartTime            time.Time
	MaxViews             int64
	LastUpdatedAt        time.Time
	StreamerLoginAtStart string
	ID_2                 uuid.NullUUID
	FetchedAt            sql.NullTime
	GzippedBytes         []byte
	StreamsID            uuid.NullUUID
}

func (q *Queries) GetEverything(ctx context.Context) ([]GetEverythingRow, error) {
	rows, err := q.db.Query(ctx, getEverything)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetEverythingRow
	for rows.Next() {
		var i GetEverythingRow
		if err := rows.Scan(
			&i.ID,
			&i.StreamerID,
			&i.StreamID,
			&i.StartTime,
			&i.MaxViews,
			&i.LastUpdatedAt,
			&i.StreamerLoginAtStart,
			&i.ID_2,
			&i.FetchedAt,
			&i.GzippedBytes,
			&i.StreamsID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getLatestStreamAndRecordingFromStreamId = `-- name: GetLatestStreamAndRecordingFromStreamId :one
SELECT
  s.id, s.last_updated_at, s.max_views, s.start_time, s.streamer_id, s.stream_id, s.streamer_login_at_start, r.id, r.fetched_at, r.gzipped_bytes
FROM 
  streams s
LEFT JOIN
  recordings r
ON
  s.id = r.streams_id
WHERE
  s.stream_id = $1
`

type GetLatestStreamAndRecordingFromStreamIdRow struct {
	ID                   uuid.UUID
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
	ID_2                 uuid.NullUUID
	FetchedAt            sql.NullTime
	GzippedBytes         []byte
}

func (q *Queries) GetLatestStreamAndRecordingFromStreamId(ctx context.Context, streamID string) (GetLatestStreamAndRecordingFromStreamIdRow, error) {
	row := q.db.QueryRow(ctx, getLatestStreamAndRecordingFromStreamId, streamID)
	var i GetLatestStreamAndRecordingFromStreamIdRow
	err := row.Scan(
		&i.ID,
		&i.LastUpdatedAt,
		&i.MaxViews,
		&i.StartTime,
		&i.StreamerID,
		&i.StreamID,
		&i.StreamerLoginAtStart,
		&i.ID_2,
		&i.FetchedAt,
		&i.GzippedBytes,
	)
	return i, err
}

const getLatestStreamFromStreamerId = `-- name: GetLatestStreamFromStreamerId :one
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  streamer_id = $1
ORDER BY
  start_time DESC
LIMIT 1
`

type GetLatestStreamFromStreamerIdRow struct {
	ID                   uuid.UUID
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

func (q *Queries) GetLatestStreamFromStreamerId(ctx context.Context, streamerID string) (GetLatestStreamFromStreamerIdRow, error) {
	row := q.db.QueryRow(ctx, getLatestStreamFromStreamerId, streamerID)
	var i GetLatestStreamFromStreamerIdRow
	err := row.Scan(
		&i.ID,
		&i.LastUpdatedAt,
		&i.MaxViews,
		&i.StartTime,
		&i.StreamerID,
		&i.StreamID,
		&i.StreamerLoginAtStart,
	)
	return i, err
}

const getLatestStreamFromStreamerLogin = `-- name: GetLatestStreamFromStreamerLogin :one
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  streamer_login_at_start = $1
ORDER BY
  start_time DESC
LIMIT $1
`

type GetLatestStreamFromStreamerLoginRow struct {
	ID                   uuid.UUID
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

func (q *Queries) GetLatestStreamFromStreamerLogin(ctx context.Context, limit int32) (GetLatestStreamFromStreamerLoginRow, error) {
	row := q.db.QueryRow(ctx, getLatestStreamFromStreamerLogin, limit)
	var i GetLatestStreamFromStreamerLoginRow
	err := row.Scan(
		&i.ID,
		&i.LastUpdatedAt,
		&i.MaxViews,
		&i.StartTime,
		&i.StreamerID,
		&i.StreamID,
		&i.StreamerLoginAtStart,
	)
	return i, err
}

const getLatestStreamsFromStreamerId = `-- name: GetLatestStreamsFromStreamerId :many
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  stream_id = $1
ORDER BY
  start_time DESC
LIMIT $2
`

type GetLatestStreamsFromStreamerIdParams struct {
	StreamID string
	Limit    int32
}

type GetLatestStreamsFromStreamerIdRow struct {
	ID                   uuid.UUID
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

func (q *Queries) GetLatestStreamsFromStreamerId(ctx context.Context, arg GetLatestStreamsFromStreamerIdParams) ([]GetLatestStreamsFromStreamerIdRow, error) {
	rows, err := q.db.Query(ctx, getLatestStreamsFromStreamerId, arg.StreamID, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetLatestStreamsFromStreamerIdRow
	for rows.Next() {
		var i GetLatestStreamsFromStreamerIdRow
		if err := rows.Scan(
			&i.ID,
			&i.LastUpdatedAt,
			&i.MaxViews,
			&i.StartTime,
			&i.StreamerID,
			&i.StreamID,
			&i.StreamerLoginAtStart,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getStreamByStreamId = `-- name: GetStreamByStreamId :one
SELECT 
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  stream_id = $1
`

type GetStreamByStreamIdRow struct {
	ID                   uuid.UUID
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

func (q *Queries) GetStreamByStreamId(ctx context.Context, streamID string) (GetStreamByStreamIdRow, error) {
	row := q.db.QueryRow(ctx, getStreamByStreamId, streamID)
	var i GetStreamByStreamIdRow
	err := row.Scan(
		&i.ID,
		&i.LastUpdatedAt,
		&i.MaxViews,
		&i.StartTime,
		&i.StreamerID,
		&i.StreamID,
		&i.StreamerLoginAtStart,
	)
	return i, err
}

const getStreamForEachStreamId = `-- name: GetStreamForEachStreamId :many
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM 
  streams
WHERE
  stream_id = ANY($1::TEXT[])
`

type GetStreamForEachStreamIdRow struct {
	ID                   uuid.UUID
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

func (q *Queries) GetStreamForEachStreamId(ctx context.Context, dollar_1 []string) ([]GetStreamForEachStreamIdRow, error) {
	rows, err := q.db.Query(ctx, getStreamForEachStreamId, dollar_1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetStreamForEachStreamIdRow
	for rows.Next() {
		var i GetStreamForEachStreamIdRow
		if err := rows.Scan(
			&i.ID,
			&i.LastUpdatedAt,
			&i.MaxViews,
			&i.StartTime,
			&i.StreamerID,
			&i.StreamID,
			&i.StreamerLoginAtStart,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getStreamForEachStreamIdUnnest = `-- name: GetStreamForEachStreamIdUnnest :many
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, streams.stream_id, streamer_login_at_start
FROM 
  streams
RIGHT JOIN
  (SELECT unnest($1::TEXT[]) AS stream_id) AS ids
ON
  streams.stream_id = ids.stream_id
`

type GetStreamForEachStreamIdUnnestRow struct {
	ID                   uuid.NullUUID
	LastUpdatedAt        sql.NullTime
	MaxViews             sql.NullInt64
	StartTime            sql.NullTime
	StreamerID           sql.NullString
	StreamID             sql.NullString
	StreamerLoginAtStart sql.NullString
}

func (q *Queries) GetStreamForEachStreamIdUnnest(ctx context.Context, streamIDArr []string) ([]GetStreamForEachStreamIdUnnestRow, error) {
	rows, err := q.db.Query(ctx, getStreamForEachStreamIdUnnest, streamIDArr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetStreamForEachStreamIdUnnestRow
	for rows.Next() {
		var i GetStreamForEachStreamIdUnnestRow
		if err := rows.Scan(
			&i.ID,
			&i.LastUpdatedAt,
			&i.MaxViews,
			&i.StartTime,
			&i.StreamerID,
			&i.StreamID,
			&i.StreamerLoginAtStart,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getStreamsByStreamId = `-- name: GetStreamsByStreamId :many
SELECT 
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  stream_id = $1
`

type GetStreamsByStreamIdRow struct {
	ID                   uuid.UUID
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

func (q *Queries) GetStreamsByStreamId(ctx context.Context, streamID string) ([]GetStreamsByStreamIdRow, error) {
	rows, err := q.db.Query(ctx, getStreamsByStreamId, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetStreamsByStreamIdRow
	for rows.Next() {
		var i GetStreamsByStreamIdRow
		if err := rows.Scan(
			&i.ID,
			&i.LastUpdatedAt,
			&i.MaxViews,
			&i.StartTime,
			&i.StreamerID,
			&i.StreamID,
			&i.StreamerLoginAtStart,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateStream = `-- name: UpdateStream :exec
UPDATE
  streams 
SET
  last_updated_at = $1, max_views = $2
WHERE
  stream_id = $1
`

type UpdateStreamParams struct {
	LastUpdatedAt time.Time
	MaxViews      int64
}

func (q *Queries) UpdateStream(ctx context.Context, arg UpdateStreamParams) error {
	_, err := q.db.Exec(ctx, updateStream, arg.LastUpdatedAt, arg.MaxViews)
	return err
}

const upsertManyStreams = `-- name: UpsertManyStreams :exec
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start)
SELECT
  unnest($1::TIMESTAMP(3)[]) AS last_updated_at,
  unnest($2::BIGINT[]) AS max_views,
  unnest($3::TIMESTAMP(3)[]) AS start_time,
  unnest($4::TEXT[]) AS streamer_id,
  unnest($5::TEXT[]) AS stream_id,
  unnest($6::TEXT[]) AS streamer_login_at_start
ON CONFLICT
  (stream_id)
DO
  UPDATE SET
    last_updated_at = EXCLUDED.last_updated_at,
    max_views = GREATEST(streams.max_views, EXCLUDED.max_views)
`

type UpsertManyStreamsParams struct {
	LastUpdatedAtArr        []time.Time
	MaxViewsArr             []int64
	StartTimeArr            []time.Time
	StreamerIDArr           []string
	StreamIDArr             []string
	StreamerLoginAtStartArr []string
}

func (q *Queries) UpsertManyStreams(ctx context.Context, arg UpsertManyStreamsParams) error {
	_, err := q.db.Exec(ctx, upsertManyStreams,
		arg.LastUpdatedAtArr,
		arg.MaxViewsArr,
		arg.StartTimeArr,
		arg.StreamerIDArr,
		arg.StreamIDArr,
		arg.StreamerLoginAtStartArr,
	)
	return err
}

const upsertRecording = `-- name: UpsertRecording :exec
INSERT INTO
  recordings (fetched_at, gzipped_bytes, streams_id)
VALUES
  ($1, $2, $3)
ON CONFLICT
  (streams_id)
DO
  UPDATE SET
    fetched_at = EXCLUDED.fetched_at,
    gzipped_bytes = EXCLUDED.gzipped_bytes
`

type UpsertRecordingParams struct {
	FetchedAt    time.Time
	GzippedBytes []byte
	StreamsID    uuid.UUID
}

func (q *Queries) UpsertRecording(ctx context.Context, arg UpsertRecordingParams) error {
	_, err := q.db.Exec(ctx, upsertRecording, arg.FetchedAt, arg.GzippedBytes, arg.StreamsID)
	return err
}

const upsertStream = `-- name: UpsertStream :exec
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start)
VALUES
  ($1, $2, $3, $4, $5, $6)
ON CONFLICT
  (stream_id)
DO
  UPDATE SET
    last_updated_at = EXCLUDED.last_updated_at,
    max_views = GREATEST(streams.max_views, EXCLUDED.max_views)
`

type UpsertStreamParams struct {
	LastUpdatedAt        time.Time
	MaxViews             int64
	StartTime            time.Time
	StreamerID           string
	StreamID             string
	StreamerLoginAtStart string
}

func (q *Queries) UpsertStream(ctx context.Context, arg UpsertStreamParams) error {
	_, err := q.db.Exec(ctx, upsertStream,
		arg.LastUpdatedAt,
		arg.MaxViews,
		arg.StartTime,
		arg.StreamerID,
		arg.StreamID,
		arg.StreamerLoginAtStart,
	)
	return err
}
