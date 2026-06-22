ALTER TABLE outbox
DROP COLUMN retry_count,
DROP COLUMN next_attempt_at, 
DROP COLUMN locked_at, 
DROP COLUMN locked_by,
DROP COLUMN last_error;