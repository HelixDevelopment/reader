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

func NewPostgres(ctx context.Context, dsn string, dimensions int) (*Postgres, error) {
	if dimensions < 1 || dimensions > 2000 {
		return nil, fmt.Errorf("memory: dimensions must be between 1 and 2000")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(30 * time.Minute)
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
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(731452)"); err != nil {
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
	rows, err := p.db.QueryContext(ctx, `SELECT id,scope,kind,content,metadata,model,content_sha256,created_at,expires_at,1-(embedding <=> $2::vector) AS similarity FROM ai_memories WHERE scope=$1 AND (expires_at IS NULL OR expires_at>now()) ORDER BY embedding <=> $2::vector,id LIMIT $3`, scope, literal, limit)
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
