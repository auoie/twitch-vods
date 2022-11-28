-- AlterTable
ALTER TABLE "streams" ADD COLUMN     "game_id_at_start" TEXT NOT NULL DEFAULT '',
ADD COLUMN     "is_mature_at_start" BOOLEAN NOT NULL DEFAULT false;
