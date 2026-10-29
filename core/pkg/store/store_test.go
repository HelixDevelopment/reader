package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type rec struct {
	BookID  string
	Chapter int
}

func TestCRUDAndPersistence(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put("progress", "b1", rec{BookID: "b1", Chapter: 3}); err != nil {
		t.Fatal(err)
	}
	var got rec
	if err := s.Get("progress", "b1", &got); err != nil || got.Chapter != 3 {
		t.Fatalf("get: %v %+v", err, got)
	}
	// reopen: data survives
	s2, _ := OpenJSON(dir)
	got = rec{}
	if err := s2.Get("progress", "b1", &got); err != nil || got.Chapter != 3 {
		t.Fatal("persistence across reopen")
	}
	if err := s2.Delete("progress", "b1"); err != nil {
		t.Fatal(err)
	}
	if err := s2.Get("progress", "b1", &got); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted key must be NotFound")
	}
}

func TestKeysSorted(t *testing.T) {
	s, _ := OpenJSON(t.TempDir())
	s.Put("books", "z", rec{})
	s.Put("books", "a", rec{})
	s.Put("books", "m", rec{})
	keys, _ := s.Keys("books")
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "m" || keys[2] != "z" {
		t.Fatalf("keys: %v", keys)
	}
}

func TestRejectsBucketTraversal(t *testing.T) {
	dir := t.TempDir()
	s, _ := OpenJSON(filepath.Join(dir, "store"))
	for _, bucket := range []string{"../escape", "/absolute", "nested/name", ""} {
		if err := s.Put(bucket, "key", "value"); err == nil {
			t.Fatalf("bucket %q must be rejected", bucket)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "escape.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("traversal created a file: %v", err)
	}
}

func TestCorruptBucketIsNeverSilentlyOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	corrupt := []byte("{not-json")
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	s, _ := OpenJSON(dir)
	if err := s.Put("progress", "new", rec{Chapter: 9}); err == nil {
		t.Fatal("corrupt bucket must fail instead of being replaced")
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != string(corrupt) {
		t.Fatalf("corrupt source was modified: %q %v", actual, err)
	}
	if _, err := s.Keys("progress"); err == nil {
		t.Fatal("corrupt bucket keys must surface decode failure")
	}
}

func TestAtomicWriteNoTmpLeft(t *testing.T) {
	dir := t.TempDir()
	s, _ := OpenJSON(dir)
	s.Put("b", "k", rec{})
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if len(matches) != 0 {
		t.Fatal("tmp file left behind")
	}
}
