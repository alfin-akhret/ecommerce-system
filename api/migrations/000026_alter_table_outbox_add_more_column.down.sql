ALTER TABLE outbox
DROP COLUMN retry_count,
DROP COLUMN next_attemt_at, 
DROP COLUMN locked_at, 
DROP COLUMN locker_by,
DROP COLUMN last_error;