-- AlterTable
ALTER TABLE "streams" ADD COLUMN     "last_updated_minus_start_time_seconds" DOUBLE PRECISION NOT NULL DEFAULT 0;
