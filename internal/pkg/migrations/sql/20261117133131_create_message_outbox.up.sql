CREATE TABLE IF NOT EXISTS message_outbox (
    id BIGSERIAL PRIMARY KEY,
    aggregate_id UUID NOT NULL REFERENCES transactions(id),
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING'CHECK (status IN ('PENDING', 'PUBLISHED')),
    retry_count INT NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_message_outbox_aggregate_event UNIQUE (aggregate_id, event_type)
);

CREATE INDEX idx_message_outbox_pending ON message_outbox (created_at, id) WHERE status = 'PENDING';
CREATE INDEX idx_message_outbox_aggregate_id ON message_outbox (aggregate_id);
