-- CreateTable
CREATE TABLE "streamers" (
    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "start_time" TIMESTAMP(3) NOT NULL,
    "streamer_login_at_start" TEXT NOT NULL,
    "streamer_id" TEXT NOT NULL,
    "profile_image_url_at_start" TEXT NOT NULL,

    CONSTRAINT "streamers_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "streamers_start_time_idx" ON "streamers"("start_time");

-- CreateIndex
CREATE UNIQUE INDEX "streamers_streamer_login_at_start_key" ON "streamers"("streamer_login_at_start");
