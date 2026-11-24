package ingest

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/lcp"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/ingest/lcpfixture"
	"github.com/helixreader/helix/server/internal/sources"
)

// Defect 1: a templated href (which may carry a token) must not reach reports.
func TestLCPLTemplatedHrefNeverEchoed(t *testing.T) {
	fx := lcpfixture.Build(t, "http://x.invalid/{id}?token=SECRETTOKEN", func(l map[string]any) { l["templated"] = true })
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.lcpl"), fx.License, 0o644)
	connector, _ := sources.NewLocal(sources.Config{Root: root})
	s := Service{Repository: catalog.NewMemory(), LCP: newFulfiller(t)}
	report, err := s.Scan(context.Background(), connector, "")
	if err != nil || len(report.Skipped) != 1 {
		t.Fatalf("report = %+v, %v", report, err)
	}
	if strings.Contains(report.Skipped[0], "SECRETTOKEN") {
		t.Fatalf("token echoed: %s", report.Skipped[0])
	}
	_, ferr := newFulfiller(t).Fulfill(context.Background(), fx.License)
	if !errors.Is(ferr, lcp.ErrTemplatedHref) || strings.Contains(ferr.Error(), "SECRETTOKEN") {
		t.Fatalf("Fulfill err = %v", ferr)
	}
}

// Defect 5: unverifiable link refused before any download; length enforced;
// failed fetches back off.
func TestLCPFulfillUnverifiableRefusedBeforeDownload(t *testing.T) {
	srv := newProvider(t, func() []byte { return []byte("x") })
	fx := lcpfixture.Build(t, srv.URL+"/p.epub", func(l map[string]any) { delete(l, "hash"); delete(l, "length") })
	_, err := newFulfiller(t).Fulfill(context.Background(), fx.License)
	if !errors.Is(err, lcp.ErrPublicationUnverifiable) {
		t.Fatalf("err = %v", err)
	}
	if n := srv.hits.Load(); n != 0 {
		t.Fatalf("origin hit %d times before refusal", n)
	}
}

func TestLCPFulfillContentLengthMismatchAbortsEarly(t *testing.T) {
	var fx lcpfixture.Fixture
	srv := newProvider(t, func() []byte { return append(append([]byte{}, fx.Publication...), make([]byte, 100)...) })
	fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	f := newFulfiller(t)
	if _, err := f.Fulfill(context.Background(), fx.License); !errors.Is(err, lcp.ErrPublicationMismatch) {
		t.Fatalf("err = %v", err)
	}
	assertNoArtifacts(t, f)
}

func TestLCPFulfillBacksOffAfterFailure(t *testing.T) {
	var fx lcpfixture.Fixture
	srv := newProvider(t, func() []byte { b := append([]byte{}, fx.Publication...); b[len(b)/2] ^= 0xff; return b })
	fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	f := newFulfiller(t)
	now := time.Unix(1_000_000, 0)
	f.Now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if _, err := f.Fulfill(context.Background(), fx.License); err == nil {
			t.Fatal("expected failure")
		}
		if i == 0 && srv.hits.Load() != 1 {
			t.Fatalf("hits = %d", srv.hits.Load())
		}
	}
	if h := srv.hits.Load(); h != 1 {
		t.Fatalf("origin hit %d times; failures must back off", h)
	}
	if _, err := f.Fulfill(context.Background(), fx.License); !errors.Is(err, ErrLCPBackoff) {
		t.Fatalf("err = %v", err)
	}
	now = now.Add(10 * time.Minute) // past first window; second attempt allowed
	f.Fulfill(context.Background(), fx.License)
	if h := srv.hits.Load(); h != 2 {
		t.Fatalf("hits after window = %d", h)
	}
	now = now.Add(9 * time.Minute) // second window (10m) is longer than the first (5m)
	f.Fulfill(context.Background(), fx.License)
	if h := srv.hits.Load(); h != 2 {
		t.Fatalf("backoff did not grow: hits = %d", h)
	}
}

// Defect 6: the dialed address is the validated IP, not the hostname.
type fakeResolver struct{ ips []net.IP }

func (r fakeResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	return r.ips, nil
}

func TestLCPDialsValidatedIPNotHostname(t *testing.T) {
	var fx lcpfixture.Fixture
	srv := newProvider(t, func() []byte { return fx.Publication })
	fx = lcpfixture.Build(t, "http://rebind.example:8080/p.epub", nil)
	f := newFulfiller(t)
	f.AllowPrivateNetwork = false
	f.Resolver = fakeResolver{ips: []net.IP{net.ParseIP("93.184.216.34")}}
	var dialed []string
	f.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialed = append(dialed, addr)
		return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
	}
	if _, err := f.Fulfill(context.Background(), fx.License); err != nil {
		t.Fatal(err)
	}
	if len(dialed) != 1 || dialed[0] != "93.184.216.34:8080" {
		t.Fatalf("dialed %v, want only the validated IP", dialed)
	}
}

func TestLCPResolverReturningPrivateIsBlocked(t *testing.T) {
	fx := lcpfixture.Build(t, "http://rebind.example/p.epub", nil)
	f := newFulfiller(t)
	f.AllowPrivateNetwork = false
	f.Resolver = fakeResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}}
	f.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		t.Fatal("dialed", addr)
		return nil, nil
	}
	if _, err := f.Fulfill(context.Background(), fx.License); !errors.Is(err, ErrLCPPrivateAddress) {
		t.Fatalf("err = %v", err)
	}
}

// Defect 7.
func TestRestrictedIPReviewForms(t *testing.T) {
	for _, ip := range []string{
		"64:ff9b::7f00:1", "64:ff9b::a9fe:a9fe", "64:ff9b:1::1", "2002:7f00:1::1", "2002:a9fe:a9fe::1", "0.1.2.3", "198.18.0.1", "198.19.255.255",
		"192.0.0.1", "240.0.0.1", "255.255.255.255", "fec0::1", "::7f00:1", "::ffff:169.254.169.254",
	} {
		if !restrictedIP(net.ParseIP(ip)) {
			t.Errorf("restrictedIP(%s) = false, want true", ip)
		}
	}
	for _, ip := range []string{"93.184.216.34", "8.8.8.8", "1.1.1.1", "2606:4700::1111", "2001:4860:4860::8888", "198.17.255.255", "198.20.0.1", "223.255.255.255", "2620:fe::fe"} {
		if restrictedIP(net.ParseIP(ip)) {
			t.Errorf("restrictedIP(%s) = true, want false", ip)
		}
	}
}

// Defect 4: cache GC.
func scanLCPL(t *testing.T, s *Service, c sources.Connector) Report {
	t.Helper()
	rep, err := s.ScanSource(context.Background(), "src-1", c, "")
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func gcRig(t *testing.T, mut func(*LCPFulfiller)) (*Service, sources.Connector, string, *provider, *lcpfixture.Fixture) {
	t.Helper()
	fx := new(lcpfixture.Fixture)
	srv := newProvider(t, func() []byte { return fx.Publication })
	*fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	root := t.TempDir()
	lp := filepath.Join(root, "a.lcpl")
	os.WriteFile(lp, fx.License, 0o644)
	c, _ := sources.NewLocal(sources.Config{Root: root})
	f := newFulfiller(t)
	f.CacheGrace = -1
	if mut != nil {
		mut(f)
	}
	return &Service{Repository: catalog.NewMemory(), LCP: f}, c, lp, srv, fx
}

func TestLCPCacheGCAfterSourceFileDeleted(t *testing.T) {
	s, c, lp, _, _ := gcRig(t, nil)
	scanLCPL(t, s, c)
	if n := len(cacheEntries(t, s.LCP)); n != 2 {
		t.Fatalf("cache = %v", cacheEntries(t, s.LCP))
	}
	os.Remove(lp)
	scanLCPL(t, s, c)
	if e := cacheEntries(t, s.LCP); len(e) != 0 {
		t.Fatalf("orphans left: %v", e)
	}
}

func TestLCPCacheGCOnLicenseChangeKeepsReferenced(t *testing.T) {
	s, c, lp, _, fx := gcRig(t, nil)
	scanLCPL(t, s, c)
	old := cacheEntries(t, s.LCP)
	var srv2Pub []byte
	srv2 := newProvider(t, func() []byte { return srv2Pub })
	fx2 := lcpfixture.Build(t, srv2.URL+"/p.epub", nil)
	srv2Pub = fx2.Publication
	_ = fx
	os.WriteFile(lp, fx2.License, 0o644)
	scanLCPL(t, s, c)
	now := cacheEntries(t, s.LCP)
	if len(now) != 2 {
		t.Fatalf("cache after change = %v (was %v)", now, old)
	}
	for _, o := range old {
		for _, n := range now {
			if o == n {
				t.Fatalf("stale entry %s kept", o)
			}
		}
	}
	// referenced artifact must still be served.
	page, _ := s.Repository.Search(context.Background(), catalog.Query{})
	if _, err := os.Stat(s.LCP.ArtifactPath(page.Works[0].Editions[0].Files[0].SHA256)); err != nil {
		t.Fatal(err)
	}
}

func TestLCPCacheGCRespectsGraceAndCap(t *testing.T) {
	s, c, lp, _, _ := gcRig(t, func(f *LCPFulfiller) { f.CacheGrace = time.Hour })
	scanLCPL(t, s, c)
	os.Remove(lp)
	scanLCPL(t, s, c)
	if n := len(cacheEntries(t, s.LCP)); n != 2 {
		t.Fatalf("a fresh unreferenced artifact inside the grace window was deleted: %v", cacheEntries(t, s.LCP))
	}
	s2, c2, lp2, _, _ := gcRig(t, func(f *LCPFulfiller) { f.CacheMaxBytes = 1 })
	scanLCPL(t, s2, c2)
	// Referenced artifacts are never evicted even over the cap.
	if n := len(cacheEntries(t, s2.LCP)); n != 2 {
		t.Fatalf("referenced artifact evicted: %v", cacheEntries(t, s2.LCP))
	}
	os.Remove(lp2)
	scanLCPL(t, s2, c2)
	if e := cacheEntries(t, s2.LCP); len(e) != 0 {
		t.Fatalf("over-cap orphan kept: %v", e)
	}
}
