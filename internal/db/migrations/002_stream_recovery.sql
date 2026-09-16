-- 002_stream_recovery.sql
-- Adds auto-recovery configuration per stream for automated health remediation

ALTER TABLE streams ADD COLUMN auto_recover INTEGER DEFAULT 1;
ALTER TABLE streams ADD COLUMN recover_timeout_sec INTEGER DEFAULT 30;
