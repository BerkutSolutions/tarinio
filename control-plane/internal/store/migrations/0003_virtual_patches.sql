CREATE TABLE IF NOT EXISTS virtual_patches (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    pattern TEXT NOT NULL,
    target TEXT NOT NULL CHECK (target IN ('uri', 'body', 'header')),
    action TEXT NOT NULL CHECK (action IN ('block', 'monitor')),
    expires_at TIMESTAMPTZ NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_virtual_patches_site_id ON virtual_patches (site_id);
CREATE INDEX IF NOT EXISTS idx_virtual_patches_expires_at ON virtual_patches (expires_at) WHERE expires_at IS NOT NULL;
