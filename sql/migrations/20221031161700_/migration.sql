-- CreateTable
CREATE TABLE "streams" (
    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "streamer_id" TEXT NOT NULL,
    "stream_id" TEXT NOT NULL,
    "start_time" TIMESTAMP(3) NOT NULL,
    "max_views" BIGINT NOT NULL,
    "last_updated_at" TIMESTAMP(3) NOT NULL,
    "streamer_login_at_start" TEXT NOT NULL,

    CONSTRAINT "streams_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "recordings" (
    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "fetched_at" TIMESTAMP(3) NOT NULL,
    "gzipped_bytes" BYTEA NOT NULL,
    "streams_id" UUID NOT NULL,

    CONSTRAINT "recordings_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "streams_stream_id_key" ON "streams"("stream_id");

-- CreateIndex
CREATE INDEX "streams_streamer_id_start_time_idx" ON "streams"("streamer_id", "start_time" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "recordings_streams_id_key" ON "recordings"("streams_id");

-- AddForeignKey
ALTER TABLE "recordings" ADD CONSTRAINT "recordings_streams_id_fkey" FOREIGN KEY ("streams_id") REFERENCES "streams"("id") ON DELETE CASCADE ON UPDATE CASCADE;
