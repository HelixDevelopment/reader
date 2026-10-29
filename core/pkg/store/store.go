// Package store is the persistence contract for the core (plan P-2 #12).
// Interim implementation: atomic JSON-file store — stdlib-only, usable in
// tests and WASM. Production swap: modernc.org/sqlite (cgo-free) with goose
// migrations porting sql/helix_schema*.sql; the Store interface is the
// stable seam.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var ErrNotFound = errors.New("store: not found")

// Store is the persistence contract.
type Store interface {
	Put(bucket, key string, value interface{}) error
	Get(bucket, key string, out interface{}) error
	Delete(bucket, key string) error
	Keys(bucket string) ([]string, error) // sorted
	Close() error
}

// JSONStore persists each bucket as one JSON file. Writes use a same-directory
// temporary file, file fsync, atomic rename and directory fsync.
type JSONStore struct {
	mu   sync.Mutex
	dir  string
	data map[string]map[string]json.RawMessage
}

func OpenJSON(dir string) (*JSONStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &JSONStore{dir: dir, data: map[string]map[string]json.RawMessage{}}, nil
}

func validBucket(bucket string) error {
	if bucket == "" || len(bucket) > 64 {
		return fmt.Errorf("store: invalid bucket name")
	}
	for _, r := range bucket {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return fmt.Errorf("store: invalid bucket name")
		}
	}
	return nil
}

func (s *JSONStore) file(bucket string) string {
	return filepath.Join(s.dir, bucket+".json")
}

func (s *JSONStore) load(bucket string) (map[string]json.RawMessage, error) {
	if err := validBucket(bucket); err != nil {
		return nil, err
	}
	if b, ok := s.data[bucket]; ok {
		return b, nil
	}
	b := map[string]json.RawMessage{}
	raw, err := os.ReadFile(s.file(bucket))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("store: load bucket %s: %w", bucket, err)
	}
	if err == nil {
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("store: decode bucket %s: %w", bucket, err)
		}
		if b == nil {
			return nil, fmt.Errorf("store: decode bucket %s: expected JSON object", bucket)
		}
	}
	s.data[bucket] = b
	return b, nil
}

func (s *JSONStore) flush(bucket string) (err error) {
	if err := validBucket(bucket); err != nil {
		return err
	}
	b := s.data[bucket]
	raw, err := json.Marshal(b) // Go maps marshal with sorted keys — deterministic
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, "."+bucket+"-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		_ = os.Remove(tmp)
	}()
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		closed = true
		return err
	}
	closed = true
	if err := os.Rename(tmp, s.file(bucket)); err != nil {
		return err
	}
	dir, err := os.Open(s.dir)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (s *JSONStore) Put(bucket, key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	b, err := s.load(bucket)
	if err != nil {
		return err
	}
	previous, existed := b[key]
	b[key] = raw
	if err := s.flush(bucket); err != nil {
		if existed {
			b[key] = previous
		} else {
			delete(b, key)
		}
		return err
	}
	return nil
}

func (s *JSONStore) Get(bucket, key string, out interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.load(bucket)
	if err != nil {
		return err
	}
	raw, ok := b[key]
	if !ok {
		return fmt.Errorf("%w: %s/%s", ErrNotFound, bucket, key)
	}
	return json.Unmarshal(raw, out)
}

func (s *JSONStore) Delete(bucket, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.load(bucket)
	if err != nil {
		return err
	}
	previous, ok := b[key]
	if !ok {
		return fmt.Errorf("%w: %s/%s", ErrNotFound, bucket, key)
	}
	delete(b, key)
	if err := s.flush(bucket); err != nil {
		b[key] = previous
		return err
	}
	return nil
}

func (s *JSONStore) Keys(bucket string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.load(bucket)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(b))
	for k := range b {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

func (s *JSONStore) Close() error { return nil }
