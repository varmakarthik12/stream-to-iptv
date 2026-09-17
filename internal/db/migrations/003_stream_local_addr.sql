-- 003_stream_local_addr.sql
-- Add local_addr column for UDP multicast network interface binding

ALTER TABLE streams ADD COLUMN local_addr TEXT DEFAULT '';
