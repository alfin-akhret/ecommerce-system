-- drop business uniqueness
ALTER TABLE sent_emails
DROP CONSTRAINT sent_emails_event_id_key;

-- drop primary key
ALTER TABLE sent_emails
DROP CONSTRAINT sent_emails_pkey;

-- drop business columns
ALTER TABLE sent_emails
DROP COLUMN provider_message_id,
DROP COLUMN subject,
DROP COLUMN recipient,
DROP COLUMN id;

-- allow existing columns to be nullable again
ALTER TABLE sent_emails
ALTER COLUMN event_id DROP NOT NULL;

ALTER TABLE sent_emails
ALTER COLUMN sent_at DROP NOT NULL;

-- rename existing columns back
ALTER TABLE sent_emails
RENAME COLUMN event_id TO message_id;

ALTER TABLE sent_emails
RENAME COLUMN sent_at TO processed_at;

-- rename table back
ALTER TABLE sent_emails
RENAME TO processed_messages;