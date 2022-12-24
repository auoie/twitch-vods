-- AlterTable
ALTER TABLE "streams" ALTER COLUMN "box_art_url_at_start" DROP NOT NULL,
ALTER COLUMN "box_art_url_at_start" SET DEFAULT '',
ALTER COLUMN "profile_image_url_at_start" DROP NOT NULL,
ALTER COLUMN "profile_image_url_at_start" SET DEFAULT '';

