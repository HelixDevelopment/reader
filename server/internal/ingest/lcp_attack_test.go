package ingest

// Review round 3: attack repros shipped as regression tests.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/ingest/lcpfixture"
	"github.com/helixreader/helix/server/internal/sources"
)

// A1: hostile licenses whose failure is content-defined (wrong stated hash) on a
// HEALTHY shared host put that host in backoff; a legitimate license on the same
// host is then refused in the same scan.
func TestHostBackoffIgnoresDocumentDefinedFailures(t *testing.T) {
	var legit lcpfixture.Fixture
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write(legit.Publication)
	}))
	defer srv.Close()
	legit = lcpfixture.Build(t, srv.URL+"/book.epub", nil)
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		i := i
		bad := lcpfixture.Build(t, srv.URL+"/book.epub", func(l map[string]any) {
			l["length"] = len(legit.Publication)
			l["hash"] = fmt.Sprintf("%064x", i+1) // wrong hash; host is healthy
		})
		os.WriteFile(filepath.Join(root, fmt.Sprintf("a%d.lcpl", i)), bad.License, 0o644)
	}
	os.WriteFile(filepath.Join(root, "z-legit.lcpl"), legit.License, 0o644)
	c, _ := sources.NewLocal(sources.Config{Root: root})
	s := &Service{Repository: catalog.NewMemory(), LCP: newFulfiller(t)}
	rep, err := s.ScanSource(context.Background(), "src-1", c, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("hits=%d files=%d skipped=%v", hits.Load(), rep.Files, rep.Skipped)
	if rep.Files != 1 {
		t.Fatalf("legitimate license on a healthy host refused because hostile licenses failed hash checks there")
	}
}

// A2: a stated length near MaxInt64 overflows the byte-budget sum and disables
// MaxScanBytes for the rest of the scan.
func TestScanByteBudgetSurvivesHugeStatedLength(t *testing.T) {
	var fxs []lcpfixture.Fixture
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		var i int
		fmt.Sscanf(r.URL.Path, "/p%d.epub", &i)
		w.Write(fxs[i].Publication)
	}))
	defer srv.Close()
	root := t.TempDir()
	for i := 0; i < 4; i++ {
		fxs = append(fxs, lcpfixture.Build(t, fmt.Sprintf("%s/p%d.epub", srv.URL, i), nil))
	}
	hostile := lcpfixture.Build(t, srv.URL+"/huge.epub", func(l map[string]any) { l["length"] = int64(math.MaxInt64) })
	os.WriteFile(filepath.Join(root, "b00.lcpl"), fxs[0].License, 0o644)
	os.WriteFile(filepath.Join(root, "b01.lcpl"), hostile.License, 0o644)
	os.WriteFile(filepath.Join(root, "b02.lcpl"), fxs[2].License, 0o644)
	os.WriteFile(filepath.Join(root, "b03.lcpl"), fxs[3].License, 0o644)
	c, _ := sources.NewLocal(sources.Config{Root: root})
	f := newFulfiller(t)
	f.MaxScanBytes = int64(len(fxs[0].Publication)) + 10 // room for one fetch
	s := &Service{Repository: catalog.NewMemory(), LCP: f}
	rep, err := s.ScanSource(context.Background(), "src-1", c, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("hits=%d files=%d", hits.Load(), rep.Files)
	if h := hits.Load(); h > 1 {
		t.Fatalf("byte budget allows one fetch, origin hit %d times (budget sum overflowed)", h)
	}
}

// A3: a second process sharing CacheDir (its own LCPFulfiller, so its own lease
// map) collects an old artifact that this process just got as a cache hit and is
// about to merge. Grace is the only cross-process guard and reclaim re-checks the
// catalog but not the file's age.
type duringRefs struct {
	catalog.Repository
	hook  func()
	calls int
}

func (d *duringRefs) ReferencedLCPArtifacts(ctx context.Context) (map[string]bool, error) {
	r, err := d.Repository.ReferencedLCPArtifacts(ctx)
	d.calls++
	if d.calls == 2 && d.hook != nil {
		h := d.hook
		d.hook = nil
		h()
	}
	return r, err
}

func TestCrossProcessGCSparesCacheHitBeforeMerge(t *testing.T) {
	s, c, _, _, fx := gcRig(t, func(f *LCPFulfiller) { f.CacheGrace = 10 * time.Minute })
	// Earlier fulfilment left an artifact that is older than grace and (right now)
	// unreferenced, e.g. its .lcpl was removed and is being put back.
	res, err := s.LCP.Fulfill(context.Background(), fx.License)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	os.Chtimes(s.LCP.ArtifactPath(res.SHA256), old, old)
	ents, _ := os.ReadDir(s.LCP.CacheDir)
	for _, e := range ents {
		os.Chtimes(filepath.Join(s.LCP.CacheDir, e.Name()), old, old)
	}
	procB := &LCPFulfiller{CacheDir: s.LCP.CacheDir, CacheGrace: 10 * time.Minute}
	var scanErr error
	var scanRep Report
	repoB := &duringRefs{Repository: s.Repository}
	// Process A's scan runs while process B is reading the catalog.
	repoB.hook = func() { scanRep, scanErr = s.ScanSource(context.Background(), "src-1", c, "") }
	// B's first catalog read happens before A's scan merges.
	r0, _ := s.Repository.ReferencedLCPArtifacts(context.Background())
	_ = r0
	if _, err := procB.Collect(context.Background(), repoB); err != nil {
		t.Fatal(err)
	}
	if scanErr != nil {
		t.Fatal(scanErr)
	}
	page, _ := s.Repository.Search(context.Background(), catalog.Query{})
	if len(page.Works) == 0 {
		t.Fatalf("scan merged nothing: %+v", scanRep)
	}
	sha := page.Works[0].Editions[0].Files[0].SHA256
	if _, err := os.Stat(s.LCP.ArtifactPath(sha)); err != nil {
		t.Fatalf("artifact referenced by the catalog was deleted by the other process: %v", err)
	}
}

// A2 (unit): the byte-budget comparison cannot overflow for any stated length.
func TestAdmitCannotOverflow(t *testing.T) {
	f := newFulfiller(t)
	f.MaxScanBytes = math.MaxInt64
	sc := f.BeginScan()
	if err := sc.admit(100); err != nil {
		t.Fatal(err)
	}
	if err := sc.admit(math.MaxInt64); !errors.Is(err, ErrLCPScanBudget) {
		t.Fatalf("admit(MaxInt64) after 100 bytes = %v, want budget refusal (sum overflowed)", err)
	}
}
