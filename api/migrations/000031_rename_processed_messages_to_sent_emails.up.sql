-- rename table
ALTER TABLE processed_messages
RENAME TO sent_emails;

-- rename existing columns
ALTER TABLE sent_emails
RENAME COLUMN message_id TO event_id;

ALTER TABLE sent_emails
RENAME COLUMN created_at TO sent_at;

-- existing column should not be nullable
ALTER TABLE sent_emails
ALTER COLUMN event_id SET NOT NULL;

ALTER TABLE sent_emails
ALTER COLUMN sent_at SET NOT NULL;

-- add surrogate primary key
ALTER TABLE sent_emails
ADD COLUMN id BIGSERIAL;

-- add business column
ALTER TABLE sent_emails
ADD COLUMN recipient TEXT NOT NULL DEFAULT '',
ADD COLUMN subject TEXT NOT NULL DEFAULT '',
ADD COLUMN provider_message_id TEXT;

-- remove temporary defaults
ALTER TABLE sent_emails
ALTER COLUMN recipient DROP DEFAULT;

ALTER TABLE sent_emails
ALTER COLUMN subject DROP DEFAULT;

-- drop old primary key
ALTER TABLE sent_emails
DROP CONSTRAINT processed_messages_pkey;

-- primary key
ALTER TABLE sent_emails
ADD CONSTRAINT sent_emails_pkey
PRIMARY KEY (id);

-- business uniqueness
ALTER TABLE sent_emails
ADD CONSTRAINT sent_emails_event_id_key
UNIQUE (event_id);



