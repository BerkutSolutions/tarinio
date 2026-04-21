CREATE TABLE IF NOT EXISTS state_documents (
    key TEXT PRIMARY KEY,
    content BYTEA NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS state_blobs (
    key TEXT PRIMARY KEY,
    content BYTEA NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_state_blobs_key_prefix ON state_blobs (key);
