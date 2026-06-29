ALTER TABLE inbox_events
DROP CONSTRAINT inbox_events_pkey;

ALTER TABLE inbox_events
ADD PRIMARY KEY (id);

ALTER TABLE inbox_events
DROP COLUMN consumer;