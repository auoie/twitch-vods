-- CreateTable
CREATE TABLE "streams" (
    "id" TEXT NOT NULL,
    "streamer_id" TEXT NOT NULL,
    "stream_id" TEXT NOT NULL,
    "start_time" TIMESTAMP(3) NOT NULL,
    "max_views" INTEGER NOT NULL,
    "last_updated_at" TIMESTAMP(3) NOT NULL,
    "time_series" BYTEA NOT NULL,

    CONSTRAINT "streams_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "streams_stream_id_key" ON "streams"("stream_id");
