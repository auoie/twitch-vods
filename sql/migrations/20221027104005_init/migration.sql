-- CreateTable
CREATE TABLE "Stream" (
    "id" TEXT NOT NULL,
    "streamerId" TEXT NOT NULL,
    "streamId" TEXT NOT NULL,
    "startTime" TIMESTAMP(3) NOT NULL,
    "maxViews" INTEGER NOT NULL,
    "lastUpdatedAt" TIMESTAMP(3) NOT NULL,
    "timeSeries" BYTEA NOT NULL,

    CONSTRAINT "Stream_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "Stream_streamId_key" ON "Stream"("streamId");
