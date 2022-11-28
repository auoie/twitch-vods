/*
  Warnings:

  - You are about to drop the column `hls_duration` on the `streams` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "streams" DROP COLUMN "hls_duration",
ADD COLUMN     "hls_duration_seconds" DOUBLE PRECISION;
