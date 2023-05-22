-- DropIndex
DROP INDEX "streams_game_id_at_start_public_max_views_id_idx";

-- DropIndex
DROP INDEX "streams_language_at_start_public_max_views_id_idx";

-- DropIndex
DROP INDEX "streams_public_max_views_id_idx";

-- AlterTable
ALTER TABLE "streamers" ALTER COLUMN "profile_image_url_at_start" SET NOT NULL;

-- AlterTable
ALTER TABLE "streams" ADD COLUMN     "seek_previews_domain" TEXT,
ADD COLUMN     "sub_only" BOOLEAN,
ALTER COLUMN "box_art_url_at_start" SET NOT NULL,
ALTER COLUMN "profile_image_url_at_start" SET NOT NULL;

-- CreateIndex
CREATE INDEX "streams_game_id_at_start_public_sub_only_max_views_id_idx" ON "streams"("game_id_at_start" ASC, "public" ASC, "sub_only" ASC, "max_views" ASC, "id" ASC);

-- CreateIndex
CREATE INDEX "streams_language_at_start_public_sub_only_max_views_id_idx" ON "streams"("language_at_start" ASC, "public" ASC, "sub_only" ASC, "max_views" ASC, "id" ASC);

-- CreateIndex
CREATE INDEX "streams_public_sub_only_max_views_id_idx" ON "streams"("public" ASC, "sub_only" ASC, "max_views" ASC, "id" ASC);
