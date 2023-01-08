-- name: GetStreamGzippedBytes :many
SELECT
  gzipped_bytes
FROM
  streams
WHERE
  stream_id = $1 AND
  start_time = $2
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
  id, max_views, start_time, s.streamer_id, stream_id, streamer_login_at_start, game_name_at_start, language_at_start, title_at_start, is_mature_at_start, game_id_at_start, bytes_found, public, sub_only, hls_duration_seconds, box_art_url_at_start, profile_image_url_at_start
FROM
  streams s
INNER JOIN
  goal_id
ON
  s.streamer_id = goal_id.streamer_id
ORDER BY
  start_time DESC
LIMIT $2;

-- name: UpsertManyStreams :exec
INSERT INTO
  streams (last_updated_at, max_views, start_time, streamer_id, stream_id, streamer_login_at_start, game_name_at_start, language_at_start, title_at_start, profile_image_url_at_start, box_art_url_at_start, is_mature_at_start, game_id_at_start, last_updated_minus_start_time_seconds)
SELECT
  unnest(@last_updated_at_arr::TIMESTAMP(3)[]) AS last_updated_at,
  unnest(@max_views_arr::BIGINT[]) AS max_views,
  unnest(@start_time_arr::TIMESTAMP(3)[]) AS start_time,
  unnest(@streamer_id_arr::TEXT[]) AS streamer_id,
  unnest(@stream_id_arr::TEXT[]) AS stream_id,
  unnest(@streamer_login_at_start_arr::TEXT[]) AS streamer_login_at_start,
  unnest(@game_name_at_start_arr::TEXT[]) AS game_name_at_start,
  unnest(@language_at_start_arr::TEXT[]) AS language_at_start,
  unnest(@title_at_start_arr::TEXT[]) AS title_at_start,
  unnest(@profile_image_url_at_start_arr::TEXT[]) AS profile_image_url_at_start,
  unnest(@box_art_url_at_start_at_start_arr::TEXT[]) AS box_art_url_at_start,
  unnest(@is_mature_at_start_arr::BOOLEAN[]) AS is_mature_at_start,
  unnest(@game_id_at_start_arr::TEXT[]) AS game_id_at_start,
  unnest(@last_updated_minus_start_time_seconds_arr::DOUBLE PRECISION[]) AS last_updated_minus_start_time_seconds
ON CONFLICT
  (stream_id, start_time)
DO
  UPDATE SET
    last_updated_at = EXCLUDED.last_updated_at,
    last_updated_minus_start_time_seconds = EXCLUDED.last_updated_minus_start_time_seconds,
    max_views = GREATEST(streams.max_views, EXCLUDED.max_views);

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
  start_time = $2;

-- name: GetPopularLiveStreams :many
SELECT
  id, max_views, start_time, streamer_id, stream_id, streamer_login_at_start, game_name_at_start, language_at_start, title_at_start, is_mature_at_start, game_id_at_start, bytes_found, public, sub_only, hls_duration_seconds, box_art_url_at_start, profile_image_url_at_start
FROM
  streams
WHERE
  public = $1 AND sub_only = $2
ORDER BY
  max_views DESC, id DESC
LIMIT $3;

-- name: GetPopularLiveStreamsByLanguage :many
SELECT
  id, max_views, start_time, streamer_id, stream_id, streamer_login_at_start, game_name_at_start, language_at_start, title_at_start, is_mature_at_start, game_id_at_start, bytes_found, public, sub_only, hls_duration_seconds, box_art_url_at_start, profile_image_url_at_start
FROM
  streams
WHERE
  language_at_start = $1 AND public = $2 AND sub_only = $3
ORDER BY
  max_views DESC, id DESC
LIMIT $4;

-- name: GetPopularLiveStreamsByGameId :many
SELECT
  id, max_views, start_time, streamer_id, stream_id, streamer_login_at_start, game_name_at_start, language_at_start, title_at_start, is_mature_at_start, game_id_at_start, bytes_found, public, sub_only, hls_duration_seconds, box_art_url_at_start, profile_image_url_at_start
FROM
  streams
WHERE
  game_id_at_start = $1 AND public = $2 AND sub_only = $3
ORDER BY
  max_views DESC, id DESC
LIMIT $4;

-- name: GetPopularCategories :many
WITH
  categories AS
(SELECT
  COUNT(*) AS count, game_name_at_start, game_id_at_start
FROM
  streams
WHERE
  last_updated_at > NOW() - INTERVAL '1 day'
GROUP BY
  game_name_at_start, game_id_at_start)
SELECT
  *
FROM
  categories
ORDER BY
  count DESC
LIMIT
  $1;

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
