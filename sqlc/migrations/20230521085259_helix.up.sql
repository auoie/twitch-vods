-- DropIndex
DROP INDEX "streams_game_id_at_start_public_sub_only_max_views_id_idx";

-- DropIndex
DROP INDEX "streams_language_at_start_public_sub_only_max_views_id_idx";

-- DropIndex
DROP INDEX "streams_public_sub_only_max_views_id_idx";

-- AlterTable
ALTER TABLE "streamers" ALTER COLUMN "profile_image_url_at_start" DROP NOT NULL;

-- AlterTable
ALTER TABLE "streams" DROP COLUMN "sub_only",
ALTER COLUMN "box_art_url_at_start" DROP NOT NULL,
ALTER COLUMN "profile_image_url_at_start" DROP NOT NULL;

-- CreateIndex
CREATE INDEX "streams_public_max_views_id_idx" ON "streams"("public", "max_views", "id");

-- CreateIndex
CREATE INDEX "streams_game_id_at_start_public_max_views_id_idx" ON "streams"("game_id_at_start", "public", "max_views", "id");

-- CreateIndex
CREATE INDEX "streams_language_at_start_public_max_views_id_idx" ON "streams"("language_at_start", "public", "max_views", "id");

-- AlterTable
ALTER TABLE "streams" DROP COLUMN "seek_previews_domain";
