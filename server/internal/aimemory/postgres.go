package aimemory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Postgres struct {
	db         *sql.DB
	Dimensions int
}

// Option tunes the memory store at construction.
type Option func(*poolSettings)

type poolSettings struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
}

func defaultPoolSettings() poolSettings {
	return poolSettings{maxOpenConns: 10, maxIdleConns: 3, connMaxLifetime: 30 * time.Minute}
}

// WithPool sets connection limits, validated against PostgreSQL capacity.
func WithPool(maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) Option {
	return func(s *poolSettings) {
		s.maxOpenConns, s.maxIdleConns, s.connMaxLifetime = maxOpenConns, maxIdleConns, connMaxLifetime
	}
}

// validate rejects pool settings database/sql would silently reinterpret. The
// checks live here rather than inline because NewPostgres also fails on an
// unreachable DSN, which makes a construction-level assertion blind to them.
func (s poolSettings) validate() error {
	if s.maxOpenConns < 1 {
		return fmt.Errorf("memory: MaxOpenConns must be at least 1")
	}
	if s.maxIdleConns < 0 || s.maxIdleConns > s.maxOpenConns {
		return fmt.Errorf("memory: MaxIdleConns %d outside 0..MaxOpenConns %d", s.maxIdleConns, s.maxOpenConns)
	}
	if s.connMaxLifetime <= 0 {
		return fmt.Errorf("memory: ConnMaxLifetime must be positive")
	}
	return nil
}

func NewPostgres(ctx context.Context, dsn string, dimensions int, options ...Option) (*Postgres, error) {
	if dimensions < 1 || dimensions > 2000 {
		return nil, fmt.Errorf("memory: dimensions must be between 1 and 2000")
	}
	settings := defaultPoolSettings()
	for _, option := range options {
		option(&settings)
	}
	if err := settings.validate(); err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(settings.maxOpenConns)
	db.SetMaxIdleConns(settings.maxIdleConns)
	db.SetConnMaxLifetime(settings.connMaxLifetime)
	p := &Postgres{db: db, Dimensions: dimensions}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := p.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return p, nil
}
func (p *Postgres) migrate(ctx context.Context) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Distinct from the catalog keys: AI-memory migrations must not serialize
	// behind catalog identity writes.
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(731453)"); err != nil {
		return fmt.Errorf("memory migration lock: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS helix_ai_memory_migrations(version INTEGER PRIMARY KEY,dimensions INTEGER NOT NULL,applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("memory migration table: %w", err)
	}
	var recordedDimensions int
	err = tx.QueryRowContext(ctx, "SELECT dimensions FROM helix_ai_memory_migrations WHERE version=1").Scan(&recordedDimensions)
	applied := err == nil
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("memory migration version: %w", err)
	}
	if applied && recordedDimensions != p.Dimensions {
		return fmt.Errorf("memory: configured dimensions %d do not match migrated dimensions %d", p.Dimensions, recordedDimensions)
	}
	if !applied {
		statements := []string{"CREATE EXTENSION IF NOT EXISTS vector", `CREATE TABLE IF NOT EXISTS ai_memories(id TEXT PRIMARY KEY,scope TEXT NOT NULL,kind TEXT NOT NULL,content TEXT NOT NULL,metadata JSONB NOT NULL DEFAULT '{}'::jsonb,model TEXT NOT NULL,embedding VECTOR(` + fmt.Sprint(p.Dimensions) + `) NOT NULL,content_sha256 TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL,expires_at TIMESTAMPTZ)`, `CREATE INDEX IF NOT EXISTS ai_memories_scope ON ai_memories(scope,kind)`, `CREATE INDEX IF NOT EXISTS ai_memories_expiry ON ai_memories(expires_at) WHERE expires_at IS NOT NULL`, `CREATE INDEX IF NOT EXISTS ai_memories_embedding ON ai_memories USING hnsw(embedding vector_cosine_ops)`}
		for _, statement := range statements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("memory migration: %w", err)
			}
		}
	}
	var actual string
	if err := tx.QueryRowContext(ctx, "SELECT format_type(atttypid,atttypmod) FROM pg_attribute WHERE attrelid='ai_memories'::regclass AND attname='embedding' AND NOT attisdropped").Scan(&actual); err != nil {
		return fmt.Errorf("memory embedding schema: %w", err)
	}
	expected := fmt.Sprintf("vector(%d)", p.Dimensions)
	if actual != expected {
		return fmt.Errorf("memory: configured dimensions %s do not match existing %s", expected, actual)
	}
	if !applied {
		if _, err := tx.ExecContext(ctx, "INSERT INTO helix_ai_memory_migrations(version,dimensions) VALUES(1,$1)", p.Dimensions); err != nil {
			return fmt.Errorf("memory migration record: %w", err)
		}
	}
	return tx.Commit()
}
func (p *Postgres) Close() error { return p.db.Close() }
func (p *Postgres) Put(ctx context.Context, record Record, embedding []float32) (Record, error) {
	record, err := Prepare(record, p.Dimensions)
	if err != nil {
		return record, err
	}
	if err := ValidateVector(embedding, p.Dimensions); err != nil {
		return record, err
	}
	metadata := record.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	// On conflict, content and its SHA-256 must advance together with the
	// embedding so a re-embedded record can never pair stale content with a
	// fresh vector or hash.
	_, err = p.db.ExecContext(ctx, `INSERT INTO ai_memories(id,scope,kind,content,metadata,model,embedding,content_sha256,created_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7::vector,$8,$9,$10) ON CONFLICT(id) DO UPDATE SET content=EXCLUDED.content,metadata=EXCLUDED.metadata,embedding=EXCLUDED.embedding,content_sha256=EXCLUDED.content_sha256,expires_at=EXCLUDED.expires_at`, record.ID, record.Scope, record.Kind, record.Content, metadata, record.Model, VectorLiteral(embedding), record.ContentSHA256, record.CreatedAt, record.ExpiresAt)
	return record, err
}

// searchSQL is the entirety of Search's query, exported to the package as a
// constant so the load benchmark can run this exact text instead of a copy that
// may drift from it.
//
// The tiebreak lives OUTSIDE the subquery on purpose. pgvector can satisfy an
// ordering of exactly `embedding <=> $2`; the `,id` that used to sit in that
// clause made the ordering unsatisfiable by the index, so every Search was a
// sequential scan of every stored embedding plus a full sort, while the HNSW
// index was still maintained on each insert. Measured at 50,000 rows x 384
// dimensions: 28.3 ms p50 for the released statement against 1.86 ms p50 for
// this one (run 20261120T220611Z in evidence/campaign-2026-11-20/vector/,
// recorded as DEFECT-005 in docs/scalability-review.md). `OFFSET 0` stops the
// planner merging the subquery away and pushing the outer tiebreak back into it.
//
// What changed semantically: the returned *set* is now the index's approximate
// top-k rather than the exact top-k, and rows tied on distance at the k boundary
// may be chosen arbitrarily by the index. The returned *order* is unchanged.
const searchSQL = `SELECT id,scope,kind,content,metadata,model,content_sha256,created_at,expires_at,1-distance AS similarity ` +
	`FROM (SELECT id,scope,kind,content,metadata,model,content_sha256,created_at,expires_at,embedding <=> $2::vector AS distance ` +
	`FROM ai_memories WHERE scope=$1 AND (expires_at IS NULL OR expires_at>now()) ` +
	`ORDER BY embedding <=> $2::vector LIMIT $3 OFFSET 0) ranked ORDER BY distance,id`

func (p *Postgres) Search(ctx context.Context, scope string, embedding []float32, limit int) ([]Result, error) {
	if err := ValidateVector(embedding, p.Dimensions); err != nil {
		return nil, err
	}
	if scope == "" {
		return nil, fmt.Errorf("memory: scope required")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	literal := VectorLiteral(embedding)
	rows, err := p.db.QueryContext(ctx, searchSQL, scope, literal, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Result{}
	for rows.Next() {
		var result Result
		var record Record
		if err := rows.Scan(&record.ID, &record.Scope, &record.Kind, &record.Content, &record.Metadata, &record.Model, &record.ContentSHA256, &record.CreatedAt, &record.ExpiresAt, &result.Similarity); err != nil {
			return nil, err
		}
		result.Record = record
		out = append(out, result)
	}
	return out, rows.Err()
}
func (p *Postgres) Delete(ctx context.Context, scope, id string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM ai_memories WHERE scope=$1 AND id=$2", scope, id)
	return err
}
