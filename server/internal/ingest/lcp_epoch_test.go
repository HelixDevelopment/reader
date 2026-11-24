package ingest

// Review round 3 (T1): the epoch check, independent of the file-age check.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

// In-process: a scan leases, merges and releases X entirely while GC's re-read
// is in flight; only the epoch check prevents the unlink.
func TestGCEpochGuardsLeaseCycleDuringReRead(t *testing.T) {
	s, _, _, _, fx := gcRig(t, nil) // CacheGrace none: only lease+epoch protect
	if _, err := s.LCP.Fulfill(context.Background(), fx.License); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	ents, _ := os.ReadDir(s.LCP.CacheDir)
	for _, e := range ents {
		os.Chtimes(filepath.Join(s.LCP.CacheDir, e.Name()), old, old)
	}
	repo := &duringRefs{Repository: s.Repository}
	repo.hook = func() {
		sc := s.LCP.BeginScan()
		res, err := sc.Fulfill(context.Background(), fx.License)
		if err != nil {
			t.Fatal(err)
		}
		w := catalog.Work{ID: "w1", Editions: []catalog.Edition{{ID: "e1", Title: "x", State: catalog.Pending, Files: []catalog.File{{ID: "f1", Path: "a.lcpl", Format: "epub", SHA256: res.SHA256, Size: res.Size, DRMScheme: "lcp", LicenseHREF: "/v1/files/f1/license", ModifiedAt: time.Now()}}}}}
		if err := s.Repository.MergeWork(context.Background(), w); err != nil {
			t.Fatal(err)
		}
		sc.Release()
	}
	if _, err := s.LCP.Collect(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	refs, _ := s.Repository.ReferencedLCPArtifacts(context.Background())
	for sha := range refs {
		if _, err := os.Stat(s.LCP.ArtifactPath(sha)); err != nil {
			t.Fatalf("referenced artifact deleted: %v", err)
		}
	}
	if len(refs) == 0 {
		t.Fatal("nothing referenced")
	}
}
