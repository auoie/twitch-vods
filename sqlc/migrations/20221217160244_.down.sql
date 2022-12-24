-- DropIndex
DROP INDEX "streams_game_id_at_start_public_sub_only_max_views_id_idx";

-- CreateIndex
CREATE INDEX "streams_game_name_at_start_public_sub_only_max_views_id_idx" ON "streams"("game_name_at_start" ASC, "public" ASC, "sub_only" ASC, "max_views" ASC, "id" ASC);

