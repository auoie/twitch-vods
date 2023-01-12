CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX streamer_login_at_start_gin_trgm_ops_idx ON "streamers" USING gin ("streamer_login_at_start" gin_trgm_ops);
