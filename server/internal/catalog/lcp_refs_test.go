package catalog

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func lcpRefsFixture(prefix string) []Work {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	mk := func(id, sha, drm, href string) Work {
		return Work{ID: prefix + id, CanonicalTitle: id, CreatedAt: now, UpdatedAt: now, Editions: []Edition{{ID: prefix + "e-" + id, Title: id, State: Pending, UpdatedAt: now, Files: []File{{ID: prefix + "f-" + id, SourceID: prefix + "src", Path: id + ".lcpl", Format: "epub", MIMEType: "application/epub+zip", Size: 1, SHA256: sha, DRMScheme: drm, LicenseHREF: href, ModifiedAt: now}}}}}
	}
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	return []Work{
		mk("w1", a, "lcp", "/v1/files/x/license"),
		mk("w2", a, "lcp", "/v1/files/y/license"), // same artifact twice: DISTINCT
		mk("w3", b, "lcp", "/v1/files/z/license"),
		mk("w4", strings.Repeat("c", 64), "lcp", ""),    // lcp without a license href: not fulfilled
		mk("w5", strings.Repeat("d", 64), "", ""),       // plain file
		mk("w6", strings.Repeat("e", 64), "adobe", "x"), // other DRM
	}
}

func checkLCPRefs(t *testing.T, repo Repository, prefix string) {
	t.Helper()
	ctx := context.Background()
	for _, w := range lcpRefsFixture(prefix) {
		if err := repo.UpsertWork(ctx, w); err != nil {
			t.Fatal(err)
		}
	}
	refs, err := repo.ReferencedLCPArtifacts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{strings.Repeat("a", 64), strings.Repeat("b", 64)} {
		if !refs[want] {
			t.Errorf("missing referenced artifact %s", want[:4])
		}
	}
	for _, not := range []string{"c", "d", "e"} {
		if refs[strings.Repeat(not, 64)] {
			t.Errorf("artifact %s must not be referenced", not)
		}
	}
	if err := repo.DeleteWork(ctx, prefix+"w3"); err != nil {
		t.Fatal(err)
	}
	refs, _ = repo.ReferencedLCPArtifacts(ctx)
	if refs[strings.Repeat("b", 64)] || !refs[strings.Repeat("a", 64)] {
		t.Errorf("refs after delete = %v", refs)
	}
	ctx2, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.ReferencedLCPArtifacts(ctx2); err == nil {
		t.Error("cancelled context must fail the read, never return a partial set")
	}
}

func TestMemoryReferencedLCPArtifacts(t *testing.T) { checkLCPRefs(t, NewMemory(), "m-") }

func TestLivePostgresReferencedLCPArtifacts(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	for _, id := range []string{"w1", "w2", "w3", "w4", "w5", "w6"} {
		_ = repo.DeleteWork(ctx, "pg-"+id)
	}
	defer func() {
		for _, id := range []string{"w1", "w2", "w3", "w4", "w5", "w6"} {
			_ = repo.DeleteWork(ctx, "pg-"+id)
		}
	}()
	checkLCPRefs(t, repo, "pg-")
}
