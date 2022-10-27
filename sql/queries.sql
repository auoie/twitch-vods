-- name: GetStreamByStreamId :one
SELECT 
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, time_series
FROM
  streams
WHERE
  id = $1 LIMIT 1;

-- name: GetStreamForEachStreamId :many
SELECT
  id, last_updated_at, max_views, start_time, streamer_id, stream_id, time_series
FROM 
  streams
WHERE
  id = ANY($1::string[]); 

-- name: AddStream :one
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, time_series)
VALUES
  ($1, $2, $3, $4, $5, $6)
RETURNING
  *;

-- name: AddManyStreams :copyfrom
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, time_series)
VALUES
  ($1, $2, $3, $4, $5, $6);
