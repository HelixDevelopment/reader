CREATE TABLE IF NOT EXISTS helix_schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS works (
    id TEXT PRIMARY KEY,
    canonical_title TEXT NOT NULL,
    series TEXT NOT NULL DEFAULT '',
    series_index DOUBLE PRECISION NOT NULL DEFAULT 0,
    subjects JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS editions (
    id TEXT PRIMARY KEY,
    work_id TEXT NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    subtitle TEXT NOT NULL DEFAULT '',
    authors JSONB NOT NULL DEFAULT '[]'::jsonb,
    publisher TEXT NOT NULL DEFAULT '',
    published TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    cover_href TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL CHECK (state IN ('pending','identified','partial','needs_review','failed')),
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (confidence >= 0 AND confidence <= 1),
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS editions_work_id ON editions(work_id);
CREATE INDEX IF NOT EXISTS editions_state ON editions(state);
CREATE TABLE IF NOT EXISTS identifiers (
    edition_id TEXT NOT NULL REFERENCES editions(id) ON DELETE CASCADE,
    scheme TEXT NOT NULL,
    value TEXT NOT NULL,
    source TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    PRIMARY KEY (edition_id, scheme, value, source)
);
CREATE INDEX IF NOT EXISTS identifiers_value ON identifiers(value);
CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY,
    edition_id TEXT NOT NULL REFERENCES editions(id) ON DELETE CASCADE,
    source_id TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL,
    format TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT '',
    size BIGINT NOT NULL CHECK (size >= 0),
    sha256 TEXT NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    protected BOOLEAN NOT NULL DEFAULT false,
    drm_scheme TEXT NOT NULL DEFAULT '',
    license_href TEXT NOT NULL DEFAULT '',
    modified_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS files_source_path ON files(source_id, path);
CREATE INDEX IF NOT EXISTS files_edition_id ON files(edition_id);
CREATE TABLE IF NOT EXISTS metadata_evidence (
    edition_id TEXT NOT NULL REFERENCES editions(id) ON DELETE CASCADE,
    field TEXT NOT NULL,
    provider TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    observed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (edition_id, field, provider, observed_at)
);
CREATE INDEX IF NOT EXISTS works_updated_id ON works(updated_at DESC, id);
INSERT INTO helix_schema_migrations(version) VALUES (1) ON CONFLICT DO NOTHING;
