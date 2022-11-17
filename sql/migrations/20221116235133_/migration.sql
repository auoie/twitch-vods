/*
  Warnings:

  - You are about to drop the column `brotli_bytes` on the `streams` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "streams" DROP COLUMN "brotli_bytes";
