// Package aimemory implements scoped, vector-searchable AI memory. Memory is
// advisory context, never a source of verified catalog metadata.
package aimemory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	ID            string          `json:"id"`
	Scope         string          `json:"scope"`
	Kind          string          `json:"kind"`
	Content       string          `json:"content"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	Model         string          `json:"model"`
	ContentSHA256 string          `json:"content_sha256"`
	CreatedAt     time.Time       `json:"created_at"`
	ExpiresAt     *time.Time      `json:"expires_at,omitempty"`
}
type Result struct {
	Record     Record  `json:"record"`
	Similarity float64 `json:"similarity"`
}
type Store interface {
	Put(context.Context, Record, []float32) (Record, error)
	Search(context.Context, string, []float32, int) ([]Result, error)
	Delete(context.Context, string, string) error
	Close() error
}

func Prepare(record Record, dimensions int) (Record, error) {
	record.Scope = strings.TrimSpace(record.Scope)
	record.Kind = strings.TrimSpace(record.Kind)
	record.Model = strings.TrimSpace(record.Model)
	if record.Scope == "" || len(record.Scope) > 256 || record.Kind == "" || len(record.Kind) > 64 {
		return record, fmt.Errorf("memory: invalid scope or kind")
	}
	if record.Content == "" || len(record.Content) > 65536 {
		return record, fmt.Errorf("memory: content size invalid")
	}
	if record.Model == "" || len(record.Model) > 256 {
		return record, fmt.Errorf("memory: embedding model required")
	}
	if len(record.Metadata) > 16384 {
		return record, fmt.Errorf("memory: metadata too large")
	}
	if len(record.Metadata) > 0 && !json.Valid(record.Metadata) {
		return record, fmt.Errorf("memory: invalid metadata JSON")
	}
	hash := sha256.Sum256([]byte(record.Content))
	record.ContentSHA256 = hex.EncodeToString(hash[:])
	if record.ID == "" {
		id := sha256.Sum256([]byte(record.Scope + "\x00" + record.Kind + "\x00" + record.Model + "\x00" + record.ContentSHA256))
		record.ID = hex.EncodeToString(id[:])
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	return record, nil
}
func ValidateVector(vector []float32, dimensions int) error {
	if dimensions < 1 || dimensions > 2000 {
		return fmt.Errorf("memory: dimensions must be between 1 and 2000")
	}
	if len(vector) != dimensions {
		return fmt.Errorf("memory: embedding dimension mismatch")
	}
	norm := false
	for _, v := range vector {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return fmt.Errorf("memory: non-finite embedding")
		}
		if v != 0 {
			norm = true
		}
	}
	if !norm {
		return fmt.Errorf("memory: cosine embedding must be nonzero")
	}
	return nil
}
func VectorLiteral(vector []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, v := range vector {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(v), 'g', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
