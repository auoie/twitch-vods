-- DropIndex
DROP INDEX "streams_game_name_at_start_public_sub_only_bytes_found_max__idx";

-- DropIndex
DROP INDEX "streams_game_name_at_start_public_sub_only_bytes_found_star_idx";

-- DropIndex
DROP INDEX "streams_language_at_start_public_sub_only_bytes_found_max_v_idx";

-- DropIndex
DROP INDEX "streams_language_at_start_public_sub_only_bytes_found_start_idx";

-- DropIndex
DROP INDEX "streams_last_updated_at_idx";

-- DropIndex
DROP INDEX "streams_public_sub_only_max_views_id_idx";

-- DropIndex
DROP INDEX "streams_public_sub_only_start_time_id_idx";

-- DropIndex
DROP INDEX "streams_start_time_idx";

-- DropIndex
DROP INDEX "streams_streamer_id_start_time_idx";

-- DropIndex
DROP INDEX "streams_streamer_login_at_start_start_time_idx";

-- CreateIndex
CREATE INDEX "streams_bytes_found_language_at_start_max_views_idx" ON "streams"("bytes_found" ASC, "language_at_start" ASC, "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_bytes_found_max_views_idx" ON "streams"("bytes_found" ASC, "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_bytes_found_recording_fetched_at_id_idx" ON "streams"("bytes_found" ASC, "recording_fetched_at" ASC, "id" ASC);

-- CreateIndex
CREATE INDEX "streams_language_at_start_max_views_idx" ON "streams"("language_at_start" ASC, "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_max_views_idx" ON "streams"("max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_start_time_idx" ON "streams"("start_time" DESC);

-- CreateIndex
CREATE INDEX "streams_streamer_id_start_time_idx" ON "streams"("streamer_id" ASC, "start_time" DESC);

-- CreateIndex
CREATE INDEX "streams_streamer_login_at_start_start_time_idx" ON "streams"("streamer_login_at_start" ASC, "start_time" DESC);

-- CreateIndex
CREATE INDEX "streams_sub_only_bytes_found_language_at_start_max_views_idx" ON "streams"("sub_only" ASC, "bytes_found" ASC, "language_at_start" ASC, "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_sub_only_bytes_found_max_views_idx" ON "streams"("sub_only" ASC, "bytes_found" ASC, "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_sub_only_language_at_start_max_views_idx" ON "streams"("sub_only" ASC, "language_at_start" ASC, "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_sub_only_max_views_idx" ON "streams"("sub_only" ASC, "max_views" DESC);

