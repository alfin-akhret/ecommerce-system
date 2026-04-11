CREATE TYPE payment_recon_status AS ENUM (
	'PENDING',
	'IN_PROGRESS',
	'FINISHED'
);

CREATE TABLE payment_recon (
    payment_id uuid primary key,
    status payment_recon_status not null,
    remark text,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);