-- name: GetStreamGzippedBytes :many
SELECT
  gzipped_bytes
FROM
  streams
WHERE
  stream_id = $1
LIMIT 1;

-- name: GetStreamForEachStreamIdBatched :batchmany
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM 
  streams
WHERE
  stream_id = $1
LIMIT 1;

-- name: GetLatestStreamsFromStreamerLogin :many
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
  id, last_updated_at, max_views, start_time, s.streamer_id, stream_id, streamer_login_at_start
FROM
  streams s
INNER JOIN
  goal_id
ON
  s.streamer_id = goal_id.streamer_id
ORDER BY
  start_time DESC
LIMIT $2;


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

-- name: GetLatestStreamsFromStreamerId :many
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start
FROM
  streams
WHERE
  streamer_id = $1
ORDER BY
  start_time DESC
LIMIT $2;

-- name: GetLatestStreams :many
SELECT
  id, stream_id, streamer_id, start_time, max_views, last_updated_at
FROM
  streams
ORDER BY
  last_updated_at DESC
LIMIT $1;

-- name: GetLatestLiveStreams :many
SELECT
  id, stream_id, streamer_id, streamer_login_at_start, start_time, max_views, last_updated_at
FROM
  streams
WHERE
  last_updated_at >= $1 AND
  bytes_found IS NULL;

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

-- name: GetHighestViewedLiveStreams :many
SELECT
  streamer_login_at_start, title_at_start, max_views, start_time, stream_id
FROM
  streams
WHERE
  bytes_found = $1 AND public = $2 AND language_at_start = $3
ORDER BY
  max_views DESC
LIMIT $4;

-- name: DeleteOldStreams :exec
DELETE FROM streams
WHERE 
  start_time < $1;

-- name: GetEverything :many
SELECT
  *
FROM
  streams s;

-- name: DeleteStreams :exec
DELETE FROM streams;
