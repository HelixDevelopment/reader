-- Optional AI-memory schema for the default 1536-dimensional configuration.
-- The server applies an equivalent migration when HELIX_AI_MEMORY=true.
-- If HELIX_EMBEDDING_DIMENSIONS differs, change both VECTOR values and recreate
-- the table/index in a fresh database; do not mix dimensions in this table.
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE IF NOT EXISTS ai_memories (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL,
    kind TEXT NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    model TEXT NOT NULL,
    embedding VECTOR(1536) NOT NULL,
    content_sha256 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS ai_memories_scope ON ai_memories(scope,kind);
CREATE INDEX IF NOT EXISTS ai_memories_expiry ON ai_memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS ai_memories_embedding ON ai_memories USING hnsw(embedding vector_cosine_ops);
