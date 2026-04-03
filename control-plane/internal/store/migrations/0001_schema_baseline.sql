CREATE TABLE revisions (
    id TEXT PRIMARY KEY,
    version BIGINT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'failed')),
    checksum TEXT NOT NULL,
    bundle_path TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    validated_at TIMESTAMPTZ NULL,
    applied_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,
    rolled_back_at TIMESTAMPTZ NULL
);

CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
    site_id TEXT NULL,
    certificate_id TEXT NULL,
    requested_by_user_id TEXT NULL,
    target_revision_id TEXT NULL REFERENCES revisions(id),
    scheduled_at TIMESTAMPTZ NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    result_summary TEXT NULL,
    result_details_json JSONB NULL
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    totp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    permissions_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE audit_events (
    id TEXT PRIMARY KEY,
    actor_user_id TEXT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    site_id TEXT NULL,
    job_id TEXT NULL REFERENCES jobs(id),
    revision_id TEXT NULL REFERENCES revisions(id),
    status TEXT NOT NULL CHECK (status IN ('succeeded', 'failed')),
    occurred_at TIMESTAMPTZ NOT NULL,
    summary TEXT NOT NULL,
    details_json JSONB NULL
);

CREATE TABLE events (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    severity TEXT NOT NULL,
    site_id TEXT NULL,
    source_component TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    summary TEXT NOT NULL,
    details_json JSONB NULL,
    related_revision_id TEXT NULL REFERENCES revisions(id),
    related_job_id TEXT NULL REFERENCES jobs(id),
    related_certificate_id TEXT NULL,
    related_rule_id TEXT NULL
);
