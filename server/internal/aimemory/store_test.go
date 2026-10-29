package aimemory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"testing"
)

func TestPrepareDeterministicIdentityAndHash(t *testing.T) {
	record := Record{Scope: "library", Kind: "preference", Content: "Use serif fonts", Model: "embedding-test", Metadata: json.RawMessage(`{"source":"test"}`)}
	a, err := Prepare(record, 3)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Prepare(record, 3)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID || a.ContentSHA256 == "" {
		t.Fatalf("identity not deterministic: %+v %+v", a, b)
	}
}
func TestPrepareRejectsInvalidMetadata(t *testing.T) {
	if _, err := Prepare(Record{Scope: "s", Kind: "k", Content: "c", Model: "m", Metadata: json.RawMessage(`{`)}, 3); err == nil {
		t.Fatal("accepted invalid metadata")
	}
}
func TestPrepareRecomputesContentHashWithContent(t *testing.T) {
	// With a caller-fixed ID, changing content must always advance
	// content_sha256 alongside it; the pair may never diverge.
	first, err := Prepare(Record{ID: "fixed", Scope: "s", Kind: "k", Model: "m", Content: "alpha"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Prepare(Record{ID: "fixed", Scope: "s", Kind: "k", Model: "m", Content: "beta"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("beta"))
	if second.ContentSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("content_sha256 not recomputed with content: %+v", second)
	}
	if first.ContentSHA256 == second.ContentSHA256 {
		t.Fatal("distinct content produced identical content hashes")
	}
	if len(second.ContentSHA256) != 64 {
		t.Fatalf("content hash must be 64 hex chars: %q", second.ContentSHA256)
	}
}
func TestVectorValidation(t *testing.T) {
	if err := ValidateVector([]float32{1, 0, 0}, 3); err != nil {
		t.Fatal(err)
	}
	for _, vector := range [][]float32{{1, 0}, {0, 0, 0}, {float32(math.NaN()), 0, 0}} {
		if err := ValidateVector(vector, 3); err == nil {
			t.Fatalf("accepted %v", vector)
		}
	}
	if got := VectorLiteral([]float32{1, 0.5, -2}); got != "[1,0.5,-2]" {
		t.Fatalf("bad vector literal: %s", got)
	}
}
