/*
  Warnings:

  - Made the column `box_art_url_at_start` on table `streams` required. This step will fail if there are existing NULL values in that column.
  - Made the column `profile_image_url_at_start` on table `streams` required. This step will fail if there are existing NULL values in that column.

*/
-- AlterTable
ALTER TABLE "streams" ALTER COLUMN "box_art_url_at_start" SET NOT NULL,
ALTER COLUMN "box_art_url_at_start" DROP DEFAULT,
ALTER COLUMN "profile_image_url_at_start" SET NOT NULL,
ALTER COLUMN "profile_image_url_at_start" DROP DEFAULT;
