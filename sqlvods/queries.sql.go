// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: queries.sql

package sqlvods

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const deleteOldStreams = `-- name: DeleteOldStreams :exec
DELETE FROM streams
WHERE 
  start_time < $1
`

func (q *Queries) DeleteOldStreams(ctx context.Context, startTime time.Time) error {
	_, err := q.db.Exec(ctx, deleteOldStreams, startTime)
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
  id, streamer_id, stream_id, start_time, max_views, last_updated_at, streamer_login_at_start, language_at_start, title_at_start, game_name_at_start, game_id_at_start, is_mature_at_start, last_updated_minus_start_time_seconds, recording_fetched_at, gzipped_bytes, hls_domain, hls_duration_seconds, bytes_found, public, sub_only, seek_previews_domain
FROM
  streams s
`

func (q *Queries) GetEverything(ctx context.Context) ([]Stream, error) {
	rows, err := q.db.Query(ctx, getEverything)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Stream
	for rows.Next() {
		var i Stream
		if err := rows.Scan(
			&i.ID,
			&i.StreamerID,
			&i.StreamID,
			&i.StartTime,
			&i.MaxViews,
			&i.LastUpdatedAt,
			&i.StreamerLoginAtStart,
			&i.LanguageAtStart,
			&i.TitleAtStart,
			&i.GameNameAtStart,
			&i.GameIDAtStart,
			&i.IsMatureAtStart,
			&i.LastUpdatedMinusStartTimeSeconds,
			&i.RecordingFetchedAt,
			&i.GzippedBytes,
			&i.HlsDomain,
			&i.HlsDurationSeconds,
			&i.BytesFound,
			&i.Public,
			&i.SubOnly,
			&i.SeekPreviewsDomain,
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

const getHighestViewedLiveStreams = `-- name: GetHighestViewedLiveStreams :many
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start, game_name_at_start, language_at_start, title_at_start, is_mature_at_start, game_id_at_start, last_updated_minus_start_time_seconds, recording_fetched_at, hls_domain, bytes_found, seek_previews_domain, public, sub_only, hls_duration_seconds
FROM
  streams
WHERE
  public = $1 AND sub_only = $2
ORDER BY
  max_views DESC
LIMIT $3
`

type GetHighestViewedLiveStreamsParams struct {
	Public  sql.NullBool
	SubOnly sql.NullBool
	Limit   int32
}

type GetHighestViewedLiveStreamsRow struct {
	ID                               uuid.UUID
	LastUpdatedAt                    time.Time
	MaxViews                         int64
	StartTime                        time.Time
	StreamerID                       string
	StreamID                         string
	StreamerLoginAtStart             string
	GameNameAtStart                  string
	LanguageAtStart                  string
	TitleAtStart                     string
	IsMatureAtStart                  bool
	GameIDAtStart                    string
	LastUpdatedMinusStartTimeSeconds float64
	RecordingFetchedAt               sql.NullTime
	HlsDomain                        sql.NullString
	BytesFound                       sql.NullBool
	SeekPreviewsDomain               sql.NullString
	Public                           sql.NullBool
	SubOnly                          sql.NullBool
	HlsDurationSeconds               sql.NullFloat64
}

func (q *Queries) GetHighestViewedLiveStreams(ctx context.Context, arg GetHighestViewedLiveStreamsParams) ([]GetHighestViewedLiveStreamsRow, error) {
	rows, err := q.db.Query(ctx, getHighestViewedLiveStreams, arg.Public, arg.SubOnly, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetHighestViewedLiveStreamsRow
	for rows.Next() {
		var i GetHighestViewedLiveStreamsRow
		if err := rows.Scan(
			&i.ID,
			&i.LastUpdatedAt,
			&i.MaxViews,
			&i.StartTime,
			&i.StreamerID,
			&i.StreamID,
			&i.StreamerLoginAtStart,
			&i.GameNameAtStart,
			&i.LanguageAtStart,
			&i.TitleAtStart,
			&i.IsMatureAtStart,
			&i.GameIDAtStart,
			&i.LastUpdatedMinusStartTimeSeconds,
			&i.RecordingFetchedAt,
			&i.HlsDomain,
			&i.BytesFound,
			&i.SeekPreviewsDomain,
			&i.Public,
			&i.SubOnly,
			&i.HlsDurationSeconds,
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

const getLatestLiveStreams = `-- name: GetLatestLiveStreams :many
SELECT
  id, stream_id, streamer_id, streamer_login_at_start, start_time, max_views, last_updated_at
FROM
  streams
WHERE
  last_updated_at >= $1 AND
  bytes_found IS NULL
`

type GetLatestLiveStreamsRow struct {
	ID                   uuid.UUID
	StreamID             string
	StreamerID           string
	StreamerLoginAtStart string
	StartTime            time.Time
	MaxViews             int64
	LastUpdatedAt        time.Time
}

func (q *Queries) GetLatestLiveStreams(ctx context.Context, lastUpdatedAt time.Time) ([]GetLatestLiveStreamsRow, error) {
	rows, err := q.db.Query(ctx, getLatestLiveStreams, lastUpdatedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetLatestLiveStreamsRow
	for rows.Next() {
		var i GetLatestLiveStreamsRow
		if err := rows.Scan(
			&i.ID,
			&i.StreamID,
			&i.StreamerID,
			&i.StreamerLoginAtStart,
			&i.StartTime,
			&i.MaxViews,
			&i.LastUpdatedAt,
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

const getLatestStreams = `-- name: GetLatestStreams :many
SELECT
  id, stream_id, streamer_id, start_time, max_views, last_updated_at
FROM
  streams
ORDER BY
  last_updated_at DESC
LIMIT $1
`

type GetLatestStreamsRow struct {
	ID            uuid.UUID
	StreamID      string
	StreamerID    string
	StartTime     time.Time
	MaxViews      int64
	LastUpdatedAt time.Time
}

func (q *Queries) GetLatestStreams(ctx context.Context, limit int32) ([]GetLatestStreamsRow, error) {
	rows, err := q.db.Query(ctx, getLatestStreams, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetLatestStreamsRow
	for rows.Next() {
		var i GetLatestStreamsRow
		if err := rows.Scan(
			&i.ID,
			&i.StreamID,
			&i.StreamerID,
			&i.StartTime,
			&i.MaxViews,
			&i.LastUpdatedAt,
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

const getLatestStreamsFromStreamerLogin = `-- name: GetLatestStreamsFromStreamerLogin :many
WITH
  goal_id AS
(SELECT
  streamer_id
FROM
  streams
WHERE
  streams.streamer_login_at_start = $1
ORDER BY
  start_time DESC
LIMIT 1)
SELECT
  id, last_updated_at, max_views, start_time, s.streamer_id, stream_id, streamer_login_at_start, game_name_at_start, language_at_start, title_at_start, is_mature_at_start, game_id_at_start, last_updated_minus_start_time_seconds, recording_fetched_at, hls_domain, bytes_found, seek_previews_domain, public, sub_only, hls_duration_seconds
FROM
  streams s
INNER JOIN
  goal_id
ON
  s.streamer_id = goal_id.streamer_id
ORDER BY
  start_time DESC
LIMIT $2
`

type GetLatestStreamsFromStreamerLoginParams struct {
	StreamerLoginAtStart string
	Limit                int32
}

type GetLatestStreamsFromStreamerLoginRow struct {
	ID                               uuid.UUID
	LastUpdatedAt                    time.Time
	MaxViews                         int64
	StartTime                        time.Time
	StreamerID                       string
	StreamID                         string
	StreamerLoginAtStart             string
	GameNameAtStart                  string
	LanguageAtStart                  string
	TitleAtStart                     string
	IsMatureAtStart                  bool
	GameIDAtStart                    string
	LastUpdatedMinusStartTimeSeconds float64
	RecordingFetchedAt               sql.NullTime
	HlsDomain                        sql.NullString
	BytesFound                       sql.NullBool
	SeekPreviewsDomain               sql.NullString
	Public                           sql.NullBool
	SubOnly                          sql.NullBool
	HlsDurationSeconds               sql.NullFloat64
}

func (q *Queries) GetLatestStreamsFromStreamerLogin(ctx context.Context, arg GetLatestStreamsFromStreamerLoginParams) ([]GetLatestStreamsFromStreamerLoginRow, error) {
	rows, err := q.db.Query(ctx, getLatestStreamsFromStreamerLogin, arg.StreamerLoginAtStart, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetLatestStreamsFromStreamerLoginRow
	for rows.Next() {
		var i GetLatestStreamsFromStreamerLoginRow
		if err := rows.Scan(
			&i.ID,
			&i.LastUpdatedAt,
			&i.MaxViews,
			&i.StartTime,
			&i.StreamerID,
			&i.StreamID,
			&i.StreamerLoginAtStart,
			&i.GameNameAtStart,
			&i.LanguageAtStart,
			&i.TitleAtStart,
			&i.IsMatureAtStart,
			&i.GameIDAtStart,
			&i.LastUpdatedMinusStartTimeSeconds,
			&i.RecordingFetchedAt,
			&i.HlsDomain,
			&i.BytesFound,
			&i.SeekPreviewsDomain,
			&i.Public,
			&i.SubOnly,
			&i.HlsDurationSeconds,
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

const getStreamGzippedBytes = `-- name: GetStreamGzippedBytes :many
SELECT
  gzipped_bytes
FROM
  streams
WHERE
  stream_id = $1 AND
  start_time = $2
LIMIT 1
`

type GetStreamGzippedBytesParams struct {
	StreamID  string
	StartTime time.Time
}

func (q *Queries) GetStreamGzippedBytes(ctx context.Context, arg GetStreamGzippedBytesParams) ([][]byte, error) {
	rows, err := q.db.Query(ctx, getStreamGzippedBytes, arg.StreamID, arg.StartTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items [][]byte
	for rows.Next() {
		var gzipped_bytes []byte
		if err := rows.Scan(&gzipped_bytes); err != nil {
			return nil, err
		}
		items = append(items, gzipped_bytes)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateRecording = `-- name: UpdateRecording :exec
UPDATE
  streams
SET
  recording_fetched_at = $3,
  hls_domain = $4,
  gzipped_bytes = $5,
  bytes_found = $6,
  seek_previews_domain = $7,
  public = $8,
  sub_only = $9,
  hls_duration_seconds = $10
WHERE
  stream_id = $1 AND
  start_time = $2
`

type UpdateRecordingParams struct {
	StreamID           string
	StartTime          time.Time
	RecordingFetchedAt sql.NullTime
	HlsDomain          sql.NullString
	GzippedBytes       []byte
	BytesFound         sql.NullBool
	SeekPreviewsDomain sql.NullString
	Public             sql.NullBool
	SubOnly            sql.NullBool
	HlsDurationSeconds sql.NullFloat64
}

func (q *Queries) UpdateRecording(ctx context.Context, arg UpdateRecordingParams) error {
	_, err := q.db.Exec(ctx, updateRecording,
		arg.StreamID,
		arg.StartTime,
		arg.RecordingFetchedAt,
		arg.HlsDomain,
		arg.GzippedBytes,
		arg.BytesFound,
		arg.SeekPreviewsDomain,
		arg.Public,
		arg.SubOnly,
		arg.HlsDurationSeconds,
	)
	return err
}

const upsertManyStreams = `-- name: UpsertManyStreams :exec
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start, game_name_at_start, language_at_start, title_at_start, is_mature_at_start, game_id_at_start, last_updated_minus_start_time_seconds)
SELECT
  unnest($1::TIMESTAMP(3)[]) AS last_updated_at,
  unnest($2::BIGINT[]) AS max_views,
  unnest($3::TIMESTAMP(3)[]) AS start_time,
  unnest($4::TEXT[]) AS streamer_id,
  unnest($5::TEXT[]) AS stream_id,
  unnest($6::TEXT[]) AS streamer_login_at_start,
  unnest($7::TEXT[]) AS game_name_at_start,
  unnest($8::TEXT[]) AS language_at_start,
  unnest($9::TEXT[]) AS title_at_start,
  unnest($10::BOOLEAN[]) AS is_mature_at_start,
  unnest($11::TEXT[]) AS game_id_at_start,
  unnest($12::DOUBLE PRECISION[]) AS last_updated_minus_start_time_seconds
ON CONFLICT
  (stream_id, start_time)
DO
  UPDATE SET
    last_updated_at = EXCLUDED.last_updated_at,
    last_updated_minus_start_time_seconds = EXCLUDED.last_updated_minus_start_time_seconds,
    max_views = GREATEST(streams.max_views, EXCLUDED.max_views)
`

type UpsertManyStreamsParams struct {
	LastUpdatedAtArr                    []time.Time
	MaxViewsArr                         []int64
	StartTimeArr                        []time.Time
	StreamerIDArr                       []string
	StreamIDArr                         []string
	StreamerLoginAtStartArr             []string
	GameNameAtStartArr                  []string
	LanguageAtStartArr                  []string
	TitleAtStartArr                     []string
	IsMatureAtStartArr                  []bool
	GameIDAtStartArr                    []string
	LastUpdatedMinusStartTimeSecondsArr []float64
}

func (q *Queries) UpsertManyStreams(ctx context.Context, arg UpsertManyStreamsParams) error {
	_, err := q.db.Exec(ctx, upsertManyStreams,
		arg.LastUpdatedAtArr,
		arg.MaxViewsArr,
		arg.StartTimeArr,
		arg.StreamerIDArr,
		arg.StreamIDArr,
		arg.StreamerLoginAtStartArr,
		arg.GameNameAtStartArr,
		arg.LanguageAtStartArr,
		arg.TitleAtStartArr,
		arg.IsMatureAtStartArr,
		arg.GameIDAtStartArr,
		arg.LastUpdatedMinusStartTimeSecondsArr,
	)
	return err
}
