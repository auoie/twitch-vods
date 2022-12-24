-- DropIndex
DROP INDEX "streams_game_name_at_start_public_sub_only_max_views_id_idx";

-- DropIndex
DROP INDEX "streams_language_at_start_public_sub_only_max_views_id_idx";

-- CreateIndex
CREATE INDEX "streams_game_name_at_start_public_sub_only_bytes_found_max__idx" ON "streams"("game_name_at_start" ASC, "public" ASC, "sub_only" ASC, "bytes_found" ASC, "max_views" ASC, "id" ASC);

-- CreateIndex
CREATE INDEX "streams_game_name_at_start_public_sub_only_bytes_found_star_idx" ON "streams"("game_name_at_start" ASC, "public" ASC, "sub_only" ASC, "bytes_found" ASC, "start_time" ASC, "id" ASC);

-- CreateIndex
CREATE INDEX "streams_language_at_start_public_sub_only_bytes_found_max_v_idx" ON "streams"("language_at_start" ASC, "public" ASC, "sub_only" ASC, "bytes_found" ASC, "max_views" ASC, "id" ASC);

-- CreateIndex
CREATE INDEX "streams_language_at_start_public_sub_only_bytes_found_start_idx" ON "streams"("language_at_start" ASC, "public" ASC, "sub_only" ASC, "bytes_found" ASC, "start_time" ASC, "id" ASC);

-- CreateIndex
CREATE INDEX "streams_public_sub_only_start_time_id_idx" ON "streams"("public" ASC, "sub_only" ASC, "start_time" ASC, "id" ASC);

