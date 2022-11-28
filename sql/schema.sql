-- CreateTable
CREATE TABLE "streams" (
    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "streamer_id" TEXT NOT NULL,
    "stream_id" TEXT NOT NULL,
    "start_time" TIMESTAMP(3) NOT NULL,
    "max_views" BIGINT NOT NULL,
    "last_updated_at" TIMESTAMP(3) NOT NULL,
    "streamer_login_at_start" TEXT NOT NULL,
    "language_at_start" TEXT NOT NULL,
    "title_at_start" TEXT NOT NULL,
    "game_name_at_start" TEXT NOT NULL,
    "game_id_at_start" TEXT,
    "is_mature_at_start" BOOLEAN,
    "recording_fetched_at" TIMESTAMP(3),
    "gzipped_bytes" BYTEA,
    "hls_domain" TEXT,
    "bytes_found" BOOLEAN,
    "public" BOOLEAN,
    "sub_only" BOOLEAN,
    "seek_previews_domain" TEXT,
    "hls_duration_seconds" DOUBLE PRECISION,

    CONSTRAINT "streams_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "streams_stream_id_key" ON "streams"("stream_id");

-- CreateIndex
CREATE INDEX "streams_streamer_id_start_time_idx" ON "streams"("streamer_id", "start_time" DESC);

-- CreateIndex
CREATE INDEX "streams_start_time_idx" ON "streams"("start_time" DESC);

-- CreateIndex
CREATE INDEX "streams_max_views_idx" ON "streams"("max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_bytes_found_max_views_idx" ON "streams"("bytes_found", "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_sub_only_max_views_idx" ON "streams"("sub_only", "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_language_at_start_max_views_idx" ON "streams"("language_at_start", "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_bytes_found_language_at_start_max_views_idx" ON "streams"("bytes_found", "language_at_start", "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_sub_only_bytes_found_max_views_idx" ON "streams"("sub_only", "bytes_found", "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_sub_only_language_at_start_max_views_idx" ON "streams"("sub_only", "language_at_start", "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_sub_only_bytes_found_language_at_start_max_views_idx" ON "streams"("sub_only", "bytes_found", "language_at_start", "max_views" DESC);

-- CreateIndex
CREATE INDEX "streams_bytes_found_recording_fetched_at_id_idx" ON "streams"("bytes_found", "recording_fetched_at", "id");
