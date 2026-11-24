package ingest

// Review round 5: singleflight edge cases, redirect/429 classification and the
// reclaim ctime guard.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
)

func licSHA(doc []byte) string { s := sha256.Sum256(doc); return hex.EncodeToString(s[:]) }

// A leader that ends without a result (backoff, panic, evicted artifact) must
// never hand a follower (empty, nil).
func TestFollowerOfSilentLeaderNeverGetsEmptyNil(t *testing.T) {
	var fx lcpfixture.Fixture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(fx.Publication) }))
	defer srv.Close()
	fx = lcpfixture.Build(t, srv.URL+"/x.epub", nil)
	f := newFulfiller(t)
	fl, leader := f.joinFlight(licSHA(fx.License))
	if !leader {
		t.Fatal("not leader")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	var res Fulfilled
	var err error
	go func() { defer wg.Done(); res, err = f.Fulfill(context.Background(), fx.License) }()
	time.Sleep(200 * time.Millisecond)
	f.endFlight(licSHA(fx.License), fl) // leader vanished without a result and without an error
	wg.Wait()
	if err == nil && res.SHA256 == "" {
		t.Fatalf("follower got (empty, nil)")
	}
	if err != nil {
		t.Fatalf("follower should have retried as leader and succeeded: %v", err)
	}
}

func TestBackoffLeaderNeverYieldsEmptyNilUnderConcurrency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	fx := lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	f := newFulfiller(t)
	f.Fulfill(context.Background(), fx.License) // records failure: license in backoff
	var bad atomic.Int32
	for round := 0; round < 100; round++ {
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				if r, err := f.Fulfill(context.Background(), fx.License); err == nil {
					_ = r
					bad.Add(1)
				}
			}()
		}
		close(start)
		wg.Wait()
	}
	if bad.Load() > 0 {
		t.Fatalf("%d callers got a nil error with no artifact", bad.Load())
	}
}

func TestEmptyResultNeverReachesCatalog(t *testing.T) {
	fx := lcpfixture.Build(t, "http://127.0.0.1:1/x.epub", nil)
	f := newFulfiller(t)
	s := &Service{Repository: catalog.NewMemory(), LCP: f}
	fl, _ := f.joinFlight(licSHA(fx.License))
	go func() { time.Sleep(300 * time.Millisecond); f.endFlight(licSHA(fx.License), fl) }()
	rep, _ := s.ScanSource(context.Background(), "s", rootWith(t, map[string][]byte{"a.lcpl": fx.License}), "")
	if rep.Files != 0 {
		t.Fatalf("scan counted an LCP file with an empty artifact hash")
	}
	refs, _ := s.Repository.ReferencedLCPArtifacts(context.Background())
	if refs[""] {
		t.Fatal("catalog references an empty artifact hash")
	}
}

// A leader panic must not strand followers; the leader's panic still propagates.
func TestLeaderPanicDoesNotStrandFollowers(t *testing.T) {
	var fx lcpfixture.Fixture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(fx.Publication) }))
	defer srv.Close()
	fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	f := newFulfiller(t)
	var calls atomic.Int32
	gate := make(chan struct{})
	lcpLeaderHook = func() {
		if calls.Add(1) == 1 {
			<-gate
			panic("leader exploded")
		}
	}
	defer func() { lcpLeaderHook = nil }()
	leaderDone := make(chan any, 1)
	go func() {
		defer func() { leaderDone <- recover() }()
		f.Fulfill(context.Background(), fx.License)
	}()
	time.Sleep(150 * time.Millisecond)
	var res Fulfilled
	var err error
	fdone := make(chan struct{})
	go func() { res, err = f.Fulfill(context.Background(), fx.License); close(fdone) }()
	time.Sleep(150 * time.Millisecond)
	close(gate)
	if r := <-leaderDone; r == nil {
		t.Fatal("leader panic was swallowed")
	}
	select {
	case <-fdone:
	case <-time.After(10 * time.Second):
		t.Fatal("follower stranded by leader panic")
	}
	if err != nil || res.SHA256 == "" {
		t.Fatalf("follower = %+v, %v; want a retried success", res, err)
	}
	f.mu.Lock()
	n := len(f.flights)
	f.mu.Unlock()
	if n != 0 {
		t.Fatalf("flights leaked: %d", n)
	}
}

// A live follower must not inherit the leader's operator cancellation.
func TestFollowerDoesNotInheritLeaderCancel(t *testing.T) {
	var fx lcpfixture.Fixture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(1500 * time.Millisecond):
			w.Write(fx.Publication)
		}
	}))
	defer srv.Close()
	fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	f := newFulfiller(t)
	lctx, cancel := context.WithCancel(context.Background())
	go f.Fulfill(lctx, fx.License)
	time.Sleep(150 * time.Millisecond)
	go func() { time.Sleep(200 * time.Millisecond); cancel() }()
	if _, err := f.Fulfill(context.Background(), fx.License); err != nil {
		t.Fatalf("live follower failed because the leader was cancelled: %v", err)
	}
}

// A follower with budget must not inherit another scan's budget refusal, and a
// follower without budget raises the warning on its own scan.
func TestFollowerDoesNotInheritLeaderBudgetRefusal(t *testing.T) {
	var fx lcpfixture.Fixture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(fx.Publication) }))
	defer srv.Close()
	fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	f := newFulfiller(t)
	fl, _ := f.joinFlight(licSHA(fx.License))
	sc := f.BeginScan()
	var res Fulfilled
	var err error
	done := make(chan struct{})
	go func() { res, err = sc.Fulfill(context.Background(), fx.License); close(done) }()
	time.Sleep(150 * time.Millisecond)
	fl.err = ErrLCPScanBudget // another scan, out of budget, was the leader
	f.endFlight(licSHA(fx.License), fl)
	<-done
	if err != nil || res.SHA256 == "" {
		t.Fatalf("follower = %+v, %v; want its own fetch", res, err)
	}
	// and a follower that is itself over budget gets a budget error + warning
	f2 := newFulfiller(t)
	f2.MaxScanFetches = 1
	sc2 := f2.BeginScan()
	sc2.fetches = 1
	fl2, _ := f2.joinFlight(licSHA(fx.License))
	d2 := make(chan error, 1)
	go func() { _, e := sc2.Fulfill(context.Background(), fx.License); d2 <- e }()
	time.Sleep(150 * time.Millisecond)
	fl2.err = ErrLCPScanBudget
	f2.endFlight(licSHA(fx.License), fl2)
	if e := <-d2; !errors.Is(e, ErrLCPScanBudget) || len(sc2.Warnings()) != 1 {
		t.Fatalf("own-budget follower: err=%v warnings=%v", e, sc2.Warnings())
	}
}

// Item 4: a newcomer that becomes leader after the previous leader finished
// re-checks the cache instead of downloading again.
func TestNewLeaderRechecksCacheBeforeDownloading(t *testing.T) {
	var fx lcpfixture.Fixture
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.Write(fx.Publication) }))
	defer srv.Close()
	fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	f := newFulfiller(t)
	var once sync.Once
	lcpBeforeJoinHook = func() {
		once.Do(func() {
			lcpBeforeJoinHook = nil
			if _, err := f.Fulfill(context.Background(), fx.License); err != nil { // the "previous leader"
				t.Error(err)
			}
		})
	}
	defer func() { lcpBeforeJoinHook = nil }()
	if _, err := f.Fulfill(context.Background(), fx.License); err != nil {
		t.Fatal(err)
	}
	if h := hits.Load(); h != 1 {
		t.Fatalf("origin downloads = %d, want 1", h)
	}
}

// Item 2: redirect loops are license-chosen (license-only backoff); 429 is the
// host asking us to slow down (host-attributed).
func TestRedirectLoopDoesNotLockHost(t *testing.T) {
	var legit lcpfixture.Fixture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/book.epub" {
			w.Write(legit.Publication)
			return
		}
		http.Redirect(w, r, r.URL.Path+"x", 302)
	}))
	defer srv.Close()
	legit = lcpfixture.Build(t, srv.URL+"/book.epub", nil)
	m := map[string][]byte{"z-legit.lcpl": legit.License}
	for i := 0; i < 3; i++ {
		b := lcpfixture.Build(t, srv.URL+"/loop"+string(rune('a'+i)), func(l map[string]any) { l["length"] = 1000; l["hash"] = sha256hex(i) })
		m["a"+string(rune('0'+i))+".lcpl"] = b.License
	}
	s := &Service{Repository: catalog.NewMemory(), LCP: newFulfiller(t)}
	rep, _ := s.ScanSource(context.Background(), "s", rootWith(t, m), "")
	if rep.Files != 1 {
		t.Fatalf("redirect-loop licenses locked a healthy host: files=%d skipped=%v", rep.Files, rep.Skipped)
	}
}

func sha256hex(i int) string { s := sha256.Sum256([]byte{byte(i)}); return hex.EncodeToString(s[:]) }

func TestHost429CountsAgainstHost(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(http.StatusTooManyRequests) }))
	defer srv.Close()
	f := newFulfiller(t)
	for i := 0; i < 8; i++ {
		fx := lcpfixture.Build(t, srv.URL+"/p"+string(rune('a'+i))+".epub", nil)
		f.Fulfill(context.Background(), fx.License)
	}
	if h := hits.Load(); h > 3 {
		t.Fatalf("a host answering 429 was hit %d times across distinct licenses", h)
	}
}

// Item 5: mtime reset by a third party (or backdated) but ctime fresh: GC must
// spare the file inside the grace window.
func TestGCSparesBackdatedMtimeWithFreshCtime(t *testing.T) {
	s, _, _, _, fx := gcRig(t, func(f *LCPFulfiller) { f.CacheGrace = 10 * time.Minute })
	res, err := s.LCP.Fulfill(context.Background(), fx.License)
	if err != nil {
		t.Fatal(err)
	}
	p := s.LCP.ArtifactPath(res.SHA256)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(p, old, old); err != nil { // mtime old, ctime = now
		t.Fatal(err)
	}
	if _, err := s.LCP.Collect(context.Background(), s.Repository); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("artifact with old mtime but fresh ctime was collected: %v", err)
	}
	_ = filepath.Join
}
