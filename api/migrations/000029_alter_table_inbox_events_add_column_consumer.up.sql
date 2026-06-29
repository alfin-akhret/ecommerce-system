ALTER TABLE inbox_events
ADD COLUMN consumer TEXT NOT NULL DEFAULT '';

ALTER TABLE inbox_events
DROP CONSTRAINT inbox_events_pkey;

ALTER TABLE inbox_events
ADD PRIMARY KEY (id, consumer);