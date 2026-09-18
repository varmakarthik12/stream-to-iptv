-- 004_stream_probe_settings.sql
-- Add analyze_duration and probe_size columns for configurable FFmpeg probing

ALTER TABLE streams ADD COLUMN analyze_duration TEXT DEFAULT '';
ALTER TABLE streams ADD COLUMN probe_size TEXT DEFAULT '';
