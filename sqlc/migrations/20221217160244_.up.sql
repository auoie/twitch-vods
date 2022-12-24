-- DropIndex
DROP INDEX "streams_game_name_at_start_public_sub_only_max_views_id_idx";

-- CreateIndex
CREATE INDEX "streams_game_id_at_start_public_sub_only_max_views_id_idx" ON "streams"("game_id_at_start", "public", "sub_only", "max_views", "id");
