package ingest

// Review round 4: adversarial repros shipped as regression tests.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/ingest/lcpfixture"
	"github.com/helixreader/helix/server/internal/sources"
)

func rootWith(t *testing.T, lic map[string][]byte) sources.Connector {
	root := t.TempDir()
	for n, b := range lic {
		os.WriteFile(filepath.Join(root, n), b, 0o644)
	}
	c, _ := sources.NewLocal(sources.Config{Root: root})
	return c
}

func TestHost4xxDocumentDefinedDoesNotLockHost(t *testing.T) {
	var legit lcpfixture.Fixture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/book.epub" {
			w.Write(legit.Publication)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	legit = lcpfixture.Build(t, srv.URL+"/book.epub", nil)
	m := map[string][]byte{"z-legit.lcpl": legit.License}
	for i := 0; i < 3; i++ {
		b := lcpfixture.Build(t, fmt.Sprintf("%s/nope%d.epub", srv.URL, i), func(l map[string]any) {
			l["length"] = 1000
			l["hash"] = fmt.Sprintf("%064x", i+1)
		})
		m[fmt.Sprintf("a%d.lcpl", i)] = b.License
	}
	s := &Service{Repository: catalog.NewMemory(), LCP: newFulfiller(t)}
	rep, _ := s.ScanSource(context.Background(), "s", rootWith(t, m), "")
	if rep.Files != 1 {
		t.Fatalf("SURVIVOR: legit refused after 3 hostile 404s on same host: files=%d skipped=%v", rep.Files, rep.Skipped)
	}
}

func TestOperatorCancelRecordsNoFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	f := newFulfiller(t)
	for i := 0; i < 3; i++ {
		fx := lcpfixture.Build(t, fmt.Sprintf("%s/p%d.epub", srv.URL, i), nil)
		ctx, cancel := context.WithCancel(context.Background())
		go func() { time.Sleep(100 * time.Millisecond); cancel() }()
		f.Fulfill(ctx, fx.License)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.hostFails) > 0 || len(f.fails) > 0 {
		t.Fatalf("SURVIVOR: operator cancel recorded failures: %+v %+v", f.hostFails, f.fails)
	}
}

func TestFailedFetchChargesOnlyBytesRead(t *testing.T) {
	var fx lcpfixture.Fixture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/book.epub" {
			w.Write(fx.Publication)
			return
		}
		w.WriteHeader(500)
	}))
	defer srv.Close()
	fx = lcpfixture.Build(t, srv.URL+"/book.epub", nil)
	m := map[string][]byte{"z.lcpl": fx.License}
	for i := 0; i < 9; i++ {
		b := lcpfixture.Build(t, fmt.Sprintf("http://127.0.0.%d:1/d.epub", i+2), func(l map[string]any) {
			l["length"] = 256 << 20
			l["hash"] = fmt.Sprintf("%064x", i+1)
		})
		m[fmt.Sprintf("a%d.lcpl", i)] = b.License
	}
	f := newFulfiller(t)
	s := &Service{Repository: catalog.NewMemory(), LCP: f}
	c := rootWith(t, m)
	rep, _ := s.ScanSource(context.Background(), "s", c, "")
	t.Logf("scan1 files=%d warnings=%v", rep.Files, rep.Warnings)
	if rep.Files != 1 {
		t.Errorf("FINDING: legit license starved in scan1 by 9 dead hostile licenses stating 256MiB each (charged bytes without transferring any)")
	}
	rep, _ = s.ScanSource(context.Background(), "s", c, "")
	t.Logf("scan2 files=%d", rep.Files)
	if rep.Files != 1 {
		t.Errorf("legit still starved in scan2")
	}
}

func TestConcurrentScansDownloadOnce(t *testing.T) {
	var fx lcpfixture.Fixture
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(200 * time.Millisecond)
		w.Write(fx.Publication)
	}))
	defer srv.Close()
	fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	f := newFulfiller(t)
	var wg sync.WaitGroup
	shas := make(chan string, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc := f.BeginScan()
			defer sc.Close()
			r, err := sc.Fulfill(context.Background(), fx.License)
			if err != nil {
				t.Errorf("err %v", err)
				return
			}
			shas <- r.SHA256
		}()
	}
	wg.Wait()
	close(shas)
	seen := map[string]bool{}
	for s := range shas {
		seen[s] = true
	}
	t.Logf("origin hits=%d distinct artifacts=%d entries=%v", hits.Load(), len(seen), cacheEntries(t, f))
	if len(seen) != 1 {
		t.Fatal("divergent artifacts")
	}
	if hits.Load() != 1 {
		t.Fatalf("30 concurrent scans of one license made %d origin downloads, want 1", hits.Load())
	}
}

func TestCacheHitVsCrossProcessGCStress(t *testing.T) {
	s, _, _, _, fx := gcRig(t, func(f *LCPFulfiller) { f.CacheGrace = 10 * time.Minute })
	res, err := s.LCP.Fulfill(context.Background(), fx.License)
	if err != nil {
		t.Fatal(err)
	}
	other := &LCPFulfiller{CacheDir: s.LCP.CacheDir, CacheGrace: 10 * time.Minute}
	p := s.LCP.ArtifactPath(res.SHA256)
	old := time.Now().Add(-time.Hour)
	var lost int
	for i := 0; i < 300; i++ {
		os.Chtimes(p, old, old)
		ents, _ := os.ReadDir(s.LCP.CacheDir)
		for _, e := range ents {
			os.Chtimes(filepath.Join(s.LCP.CacheDir, e.Name()), old, old)
		}
		var wg sync.WaitGroup
		wg.Add(2)
		var got Fulfilled
		var ferr error
		go func() { defer wg.Done(); other.Collect(context.Background(), s.Repository) }()
		go func() { defer wg.Done(); got, ferr = s.LCP.Fulfill(context.Background(), fx.License) }()
		wg.Wait()
		if ferr == nil {
			if _, err := os.Stat(s.LCP.ArtifactPath(got.SHA256)); err != nil {
				lost++
				// refetch for next round
				s.LCP.Fulfill(context.Background(), fx.License)
			}
		}
	}
	t.Logf("fulfil returned success but artifact gone: %d/300", lost)
	if lost > 0 {
		t.Errorf("SURVIVOR: cache hit handed out an artifact another process unlinked (%d/300)", lost)
	}
}

// Item 5 (deterministic): a cache hit makes the file visibly young BEFORE it
// opens or hashes it, so another process's age re-check spares it.
func TestCacheHitTouchesBeforeHashing(t *testing.T) {
	s, _, _, _, fx := gcRig(t, nil)
	res, err := s.LCP.Fulfill(context.Background(), fx.License)
	if err != nil {
		t.Fatal(err)
	}
	p := s.LCP.ArtifactPath(res.SHA256)
	old := time.Now().Add(-time.Hour)
	os.Chtimes(p, old, old)
	var mtimeAtHash time.Time
	lcpAfterHashHook = func(path string) { st, _ := os.Stat(path); mtimeAtHash = st.ModTime() }
	defer func() { lcpAfterHashHook = nil }()
	lsum := sha256.Sum256(fx.License)
	if _, ok := s.LCP.cached(hex.EncodeToString(lsum[:]), nil); !ok {
		t.Fatal("cache hit expected")
	}
	if time.Since(mtimeAtHash) > time.Minute {
		t.Fatalf("file still old (%v) when the hit finished hashing; it must be touched before open", mtimeAtHash)
	}
}

// Item 5 (deterministic): if the artifact vanishes (another process unlinked
// it) between the hit's hash and its return, the hit is refused and the caller
// re-fetches instead of being handed a missing file.
func TestCacheHitRefusedWhenArtifactVanishesAfterHash(t *testing.T) {
	s, _, _, _, fx := gcRig(t, nil)
	res, err := s.LCP.Fulfill(context.Background(), fx.License)
	if err != nil {
		t.Fatal(err)
	}
	lcpAfterHashHook = func(path string) { os.Remove(path) }
	defer func() { lcpAfterHashHook = nil }()
	lsum := sha256.Sum256(fx.License)
	if _, ok := s.LCP.cached(hex.EncodeToString(lsum[:]), nil); ok {
		t.Fatalf("hit returned for an artifact that no longer exists (%s)", res.SHA256[:8])
	}
}
