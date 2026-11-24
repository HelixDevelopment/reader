package ingest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/ingest/lcpfixture"
	"github.com/helixreader/helix/server/internal/sources"
)

// F1: an in-process lease protects an artifact from fulfilment until after its
// MergeWork, independent of the grace window.
func TestGCSkipsLeasedArtifactUntilScanCloses(t *testing.T) {
	s, _, _, _, fx := gcRig(t, nil) // grace none: the artifact counts as old
	sc := s.LCP.BeginScan()
	res, err := sc.Fulfill(context.Background(), fx.License)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LCP.GC(context.Background(), s.Repository); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.LCP.ArtifactPath(res.SHA256)); err != nil {
		t.Fatalf("leased artifact collected: %v", err)
	}
	sc.Close()
	if _, err := s.LCP.GC(context.Background(), s.Repository); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.LCP.ArtifactPath(res.SHA256)); err == nil {
		t.Fatal("artifact still present after the lease was released and nothing references it")
	}
}

// F1: over the cap with nothing evictable: warn, delete nothing referenced.
func TestGCOverCapWarnsAndKeepsReferenced(t *testing.T) {
	s, c, _, _, _ := gcRig(t, func(f *LCPFulfiller) { f.CacheMaxBytes = 1 })
	rep := scanLCPL(t, s, c)
	if n := len(cacheEntries(t, s.LCP)); n != 2 {
		t.Fatalf("referenced artifact evicted: %v", cacheEntries(t, s.LCP))
	}
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "exceeds") && strings.Contains(w, "cap") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no over-cap warning in report: %v", rep.Warnings)
	}
}

func lcplFiles(t *testing.T, n int, href func(i int) string) (sources.Connector, []lcpfixture.Fixture) {
	t.Helper()
	root := t.TempDir()
	var fxs []lcpfixture.Fixture
	for i := 0; i < n; i++ {
		fx := lcpfixture.Build(t, href(i), nil)
		fxs = append(fxs, fx)
		os.WriteFile(filepath.Join(root, fmt.Sprintf("b%02d.lcpl", i)), fx.License, 0o644)
	}
	c, _ := sources.NewLocal(sources.Config{Root: root})
	return c, fxs
}

// F5: a scan stops fetching after MaxScanFetches distinct publications.
func TestScanBudgetLimitsDistinctFetches(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); http.NotFound(w, r) }))
	defer srv.Close()
	c, _ := lcplFiles(t, 6, func(i int) string { return fmt.Sprintf("%s/p%d.epub", srv.URL, i) })
	f := newFulfiller(t)
	f.MaxScanFetches = 2
	f.HostFailureThreshold = -1 // isolate the budget from host backoff
	s := &Service{Repository: catalog.NewMemory(), LCP: f}
	rep, err := s.ScanSource(context.Background(), "src-1", c, "")
	if err != nil {
		t.Fatal(err)
	}
	if h := hits.Load(); h != 2 {
		t.Fatalf("origin hit %d times, budget is 2", h)
	}
	warned := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "scan budget") {
			warned = true
		}
	}
	if !warned || len(rep.Skipped) != 6 {
		t.Fatalf("warnings = %v skipped = %d", rep.Warnings, len(rep.Skipped))
	}
}

// F5: the byte budget refuses a fetch whose stated size no longer fits.
func TestScanBudgetLimitsTotalBytes(t *testing.T) {
	var fxs []lcpfixture.Fixture
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		var i int
		fmt.Sscanf(r.URL.Path, "/p%d.epub", &i)
		w.Write(fxs[i].Publication)
	}))
	defer srv.Close()
	c, fx := lcplFiles(t, 4, func(i int) string { return fmt.Sprintf("%s/p%d.epub", srv.URL, i) })
	fxs = fx
	f := newFulfiller(t)
	f.MaxScanBytes = int64(fx[0].Link["length"].(int))*2 + 10
	s := &Service{Repository: catalog.NewMemory(), LCP: f}
	rep, err := s.ScanSource(context.Background(), "src-1", c, "")
	if err != nil {
		t.Fatal(err)
	}
	if h := hits.Load(); h != 2 || rep.Files != 2 {
		t.Fatalf("hits = %d files = %d, want 2 and 2", h, rep.Files)
	}
}

// F5: failures back off per publication host as well as per license, so a flood
// of distinct licenses pointing at one dead host costs a bounded number of hits.
func TestHostBackoffAcrossDistinctLicenses(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(http.StatusBadGateway) }))
	defer srv.Close()
	f := newFulfiller(t)
	now := time.Unix(2_000_000, 0)
	f.Now = func() time.Time { return now }
	var lastErr error
	for i := 0; i < 12; i++ {
		fx := lcpfixture.Build(t, fmt.Sprintf("%s/p%d.epub", srv.URL, i), nil)
		_, lastErr = f.Fulfill(context.Background(), fx.License)
	}
	if h := hits.Load(); h > 3 {
		t.Fatalf("origin hit %d times for 12 distinct licenses on one failing host", h)
	}
	if !errors.Is(lastErr, ErrLCPBackoff) {
		t.Fatalf("err = %v", lastErr)
	}
	now = now.Add(2 * time.Hour)
	fx := lcpfixture.Build(t, srv.URL+"/again.epub", nil)
	before := hits.Load()
	f.Fulfill(context.Background(), fx.License)
	if hits.Load() != before+1 {
		t.Fatal("host backoff never expires")
	}
}

// F5: eviction of tracked failures is deterministic (oldest first) and does not
// wipe backoff state for other licenses or hosts.
func TestFailureMapEvictsOldestOnly(t *testing.T) {
	f := newFulfiller(t)
	now := time.Unix(3_000_000, 0)
	f.Now = func() time.Time { return now }
	for i := 0; i < lcpMaxTrackedFailures; i++ {
		now = now.Add(time.Second)
		f.recordFailure(fmt.Sprintf("lic-%05d", i), "")
	}
	now = now.Add(time.Second)
	f.recordFailure("lic-new", "")
	f.mu.Lock()
	_, oldest := f.fails["lic-00000"]
	n := len(f.fails)
	var wiped []string
	for _, k := range []string{"lic-00001", "lic-02048", fmt.Sprintf("lic-%05d", lcpMaxTrackedFailures-1), "lic-new"} {
		if _, ok := f.fails[k]; !ok {
			wiped = append(wiped, k)
		}
	}
	f.mu.Unlock()
	if oldest {
		t.Fatal("oldest entry should have been evicted")
	}
	if len(wiped) > 0 {
		t.Fatalf("backoff state wiped for %v", wiped)
	}
	if !f.backingOff("lic-new") {
		t.Fatal("newest failure must be backing off")
	}
	if n != lcpMaxTrackedFailures {
		t.Fatalf("tracked = %d", n)
	}
}
