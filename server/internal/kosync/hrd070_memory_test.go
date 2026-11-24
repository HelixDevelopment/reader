package kosync

import (
	"context"
	"errors"
	"testing"
)

// TestHRD070MemoryStore pins every branch of the Memory store directly (the
// service-level tests reach it only through Service). Oracle: the documented
// Store contract (duplicate users rejected, keep-the-higher-percentage with a
// tie replacing, typed not-found errors), decided by hand.
func TestHRD070MemoryStore(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	if _, err := m.UserVerifier(ctx, "ann"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unknown user: %v", err)
	}
	if err := m.CreateUser(ctx, "ann", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := m.CreateUser(ctx, "ann", "v2"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("duplicate user: %v", err)
	}
	if v, err := m.UserVerifier(ctx, "ann"); err != nil || v != "v1" {
		t.Fatalf("verifier replaced by a duplicate create: %q %v", v, err)
	}
	if _, err := m.GetProgress(ctx, "ann", "d"); !errors.Is(err, ErrProgressNotFound) {
		t.Fatalf("no progress yet: %v", err)
	}
	put := func(user, doc string, pct float64, marker string) {
		t.Helper()
		if err := m.PutProgress(ctx, user, Progress{Document: doc, Percentage: pct, Progress: marker}); err != nil {
			t.Fatal(err)
		}
	}
	want := func(user, doc string, pct float64, marker string) {
		t.Helper()
		got, err := m.GetProgress(ctx, user, doc)
		if err != nil || got.Percentage != pct || got.Progress != marker {
			t.Fatalf("%s/%s: got %+v %v, want %v %q", user, doc, got, err, pct, marker)
		}
	}
	put("ann", "d", 0.5, "first")
	want("ann", "d", 0.5, "first")
	put("ann", "d", 0.25, "lower")
	want("ann", "d", 0.5, "first") // lower never regresses
	put("ann", "d", 0.5, "tie")
	want("ann", "d", 0.5, "tie") // a tie replaces (>=)
	put("ann", "d", 0.75, "higher")
	want("ann", "d", 0.75, "higher")
	// first write wins even below zero: nothing existing to compare with
	put("ann", "neg", -1, "neg")
	want("ann", "neg", -1, "neg")
	// documents and users are independent
	put("ann", "other", 0.1, "o")
	want("ann", "other", 0.1, "o")
	want("ann", "d", 0.75, "higher")
	put("bob", "d", 0.2, "bob")
	want("bob", "d", 0.2, "bob")
	want("ann", "d", 0.75, "higher")
	if _, err := m.GetProgress(ctx, "bob", "other"); !errors.Is(err, ErrProgressNotFound) {
		t.Fatalf("bob has no 'other': %v", err)
	}
	if _, err := m.GetProgress(ctx, "nobody", "d"); !errors.Is(err, ErrProgressNotFound) {
		t.Fatalf("unknown user progress: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}
