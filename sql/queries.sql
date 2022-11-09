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

-- name: AddStream :exec
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start)
VALUES
  ($1, $2, $3, $4, $5, $6);

-- name: UpdateStream :exec
UPDATE
  streams 
SET
  last_updated_at = $1, max_views = $2
WHERE
  stream_id = $1;

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

-- name: GetLatestStreamAndRecordingFromStreamId :one
SELECT
  s.id, s.last_updated_at, s.max_views, s.start_time, s.streamer_id, s.stream_id, s.streamer_login_at_start, r.id, r.fetched_at, r.gzipped_bytes
FROM 
  streams s
LEFT JOIN
  recordings r
ON
  s.stream_id = r.stream_id
WHERE
  s.stream_id = $1;

-- name: UpsertRecording :exec
INSERT INTO
  recordings (stream_id, fetched_at, hls_domain, gzipped_bytes, bytes_found, seek_previews_domain, public, sub_only)
VALUES
  ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT
  (stream_id)
DO
  UPDATE SET
    fetched_at = EXCLUDED.fetched_at,
    hls_domain = EXCLUDED.hls_domain,
    gzipped_bytes = EXCLUDED.gzipped_bytes,
    bytes_found = EXCLUDED.bytes_found,
    seek_previews_domain = EXCLUDED.seek_previews_domain,
    public = EXCLUDED.public,
    sub_only = EXCLUDED.sub_only;

-- name: GetEverything :many
SELECT
  *
FROM
  streams s
LEFT JOIN
  recordings r
ON
  s.stream_id = r.stream_id;

-- name: DeleteStreams :exec
DELETE FROM streams;
