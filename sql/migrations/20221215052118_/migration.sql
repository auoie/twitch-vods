-- DropIndex
DROP INDEX "streams_bytes_found_language_at_start_max_views_idx";

-- DropIndex
DROP INDEX "streams_bytes_found_max_views_idx";

-- DropIndex
DROP INDEX "streams_bytes_found_recording_fetched_at_id_idx";

-- DropIndex
DROP INDEX "streams_language_at_start_max_views_idx";

-- DropIndex
DROP INDEX "streams_max_views_idx";

-- DropIndex
DROP INDEX "streams_start_time_idx";

-- DropIndex
DROP INDEX "streams_streamer_id_start_time_idx";

-- DropIndex
DROP INDEX "streams_streamer_login_at_start_start_time_idx";

-- DropIndex
DROP INDEX "streams_sub_only_bytes_found_language_at_start_max_views_idx";

-- DropIndex
DROP INDEX "streams_sub_only_bytes_found_max_views_idx";

-- DropIndex
DROP INDEX "streams_sub_only_language_at_start_max_views_idx";

-- DropIndex
DROP INDEX "streams_sub_only_max_views_idx";

-- CreateIndex
CREATE INDEX "streams_streamer_id_start_time_idx" ON "streams"("streamer_id", "start_time");

-- CreateIndex
CREATE INDEX "streams_streamer_login_at_start_start_time_idx" ON "streams"("streamer_login_at_start", "start_time");

-- CreateIndex
CREATE INDEX "streams_last_updated_at_idx" ON "streams"("last_updated_at");

-- CreateIndex
CREATE INDEX "streams_start_time_idx" ON "streams"("start_time");

-- CreateIndex
CREATE INDEX "streams_public_sub_only_start_time_id_idx" ON "streams"("public", "sub_only", "start_time", "id");

-- CreateIndex
CREATE INDEX "streams_public_sub_only_max_views_id_idx" ON "streams"("public", "sub_only", "max_views", "id");

-- CreateIndex
CREATE INDEX "streams_game_name_at_start_public_sub_only_bytes_found_star_idx" ON "streams"("game_name_at_start", "public", "sub_only", "bytes_found", "start_time", "id");

-- CreateIndex
CREATE INDEX "streams_language_at_start_public_sub_only_bytes_found_start_idx" ON "streams"("language_at_start", "public", "sub_only", "bytes_found", "start_time", "id");

-- CreateIndex
CREATE INDEX "streams_game_name_at_start_public_sub_only_bytes_found_max__idx" ON "streams"("game_name_at_start", "public", "sub_only", "bytes_found", "max_views", "id");

-- CreateIndex
CREATE INDEX "streams_language_at_start_public_sub_only_bytes_found_max_v_idx" ON "streams"("language_at_start", "public", "sub_only", "bytes_found", "max_views", "id");
