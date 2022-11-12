-- name: GetStreamByStreamId :one
SELECT 
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  stream_id = $1;

-- name: GetStreamsByStreamId :many
SELECT 
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  stream_id = $1;

-- name: GetStreamForEachStreamId :many
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM 
  streams
WHERE
  stream_id = ANY($1::TEXT[]);

-- name: GetStreamForEachStreamIdBatched :batchmany
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM 
  streams
WHERE
  stream_id = $1
LIMIT 1;

-- name: GetStreamForEachStreamIdUnnest :many
WITH
  ids AS (SELECT unnest(@stream_id_arr::TEXT[]) AS stream_id)
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, streams.stream_id, streamer_login_at_start
FROM 
  ids
LEFT JOIN
  streams
ON
  ids.stream_id = streams.stream_id; 

-- name: UpsertStream :exec
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start)
VALUES
  ($1, $2, $3, $4, $5, $6)
ON CONFLICT
  (stream_id)
DO
  UPDATE SET
    last_updated_at = EXCLUDED.last_updated_at,
    max_views = GREATEST(streams.max_views, EXCLUDED.max_views);

-- name: UpsertManyStreams :exec
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start, game_name_at_start, language_at_start, title_at_start)
SELECT
  unnest(@last_updated_at_arr::TIMESTAMP(3)[]) AS last_updated_at,
  unnest(@max_views_arr::BIGINT[]) AS max_views,
  unnest(@start_time_arr::TIMESTAMP(3)[]) AS start_time,
  unnest(@streamer_id_arr::TEXT[]) AS streamer_id,
  unnest(@stream_id_arr::TEXT[]) AS stream_id,
  unnest(@streamer_login_at_start_arr::TEXT[]) AS streamer_login_at_start,
  unnest(@game_name_at_start_arr::TEXT[]) AS game_name_at_start,
  unnest(@language_at_start_arr::TEXT[]) AS language_at_start,
  unnest(@title_at_start_arr::TEXT[]) AS title_at_start
ON CONFLICT
  (stream_id)
DO
  UPDATE SET
    last_updated_at = EXCLUDED.last_updated_at,
    max_views = GREATEST(streams.max_views, EXCLUDED.max_views);

-- name: AddManyStreams :copyfrom
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start)
VALUES
  ($1, $2, $3, $4, $5, $6);

-- name: GetLatestStreamFromStreamerId :one
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  streamer_id = $1
ORDER BY
  start_time DESC
LIMIT 1;

-- name: GetLatestStreamsFromStreamerId :many
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  stream_id = $1
ORDER BY
  start_time DESC
LIMIT $2;

-- name: GetLatestStreamFromStreamerLogin :one
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  streamer_login_at_start = $1
ORDER BY
  start_time DESC
LIMIT $1;

-- name: UpdateRecording :exec
UPDATE
  streams
SET
  recording_fetched_at = $2,
  hls_domain = $3,
  gzipped_bytes = $4,
  bytes_found = $5,
  seek_previews_domain = $6,
  public = $7,
  sub_only = $8
WHERE
  stream_id = $1;

-- name: GetEverything :many
SELECT
  *
FROM
  streams s;

-- name: DeleteStreams :exec
DELETE FROM streams;
