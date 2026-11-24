package ingest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/lcp"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/ingest/lcpfixture"
	"github.com/helixreader/helix/server/internal/sources"
)

// provider serves fx.Publication (or body when set) and counts requests.
type provider struct {
	*httptest.Server
	hits atomic.Int64
}

func newProvider(t *testing.T, body func() []byte) *provider {
	t.Helper()
	p := &provider{}
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.hits.Add(1)
		w.Header().Set("Content-Type", "application/epub+zip")
		w.Write(body())
	}))
	t.Cleanup(p.Close)
	return p
}

func newFulfiller(t *testing.T) *LCPFulfiller {
	return &LCPFulfiller{CacheDir: filepath.Join(t.TempDir(), "lcp"), AllowPrivateNetwork: true}
}

func cacheEntries(t *testing.T, f *LCPFulfiller) []string {
	t.Helper()
	ents, _ := os.ReadDir(f.CacheDir)
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	return out
}

func TestLCPLScanFulfillsStoresAndStaysIdempotent(t *testing.T) {
	var fx lcpfixture.Fixture
	srv := newProvider(t, func() []byte { return fx.Publication })
	fx = lcpfixture.Build(t, srv.URL+"/pub.epub", nil)

	root := t.TempDir()
	licPath := filepath.Join(root, "Sealed Book.lcpl")
	if err := os.WriteFile(licPath, fx.License, 0o644); err != nil {
		t.Fatal(err)
	}
	before := dirDigest(t, root)
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	ful := newFulfiller(t)
	repo := catalog.NewMemory()
	s := Service{Repository: repo, LCP: ful}

	report, err := s.ScanSource(context.Background(), "src-1", connector, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Files != 1 || report.Works != 1 || len(report.Skipped) != 0 {
		t.Fatalf("report = %+v", report)
	}
	page, _ := repo.Search(context.Background(), catalog.Query{Text: "Sealed"})
	if len(page.Works) != 1 {
		t.Fatalf("catalog works = %d", len(page.Works))
	}
	file := page.Works[0].Editions[0].Files[0]
	if !file.Protected || file.DRMScheme != "lcp" || file.Format != "epub" || file.MIMEType != "application/epub+zip" || file.Path != "Sealed Book.lcpl" {
		t.Fatalf("file = %+v", file)
	}
	if file.LicenseHREF != "/v1/files/"+file.ID+"/license" {
		t.Fatalf("license href = %q", file.LicenseHREF)
	}
	if !IsFulfilledLCP(file) {
		t.Fatal("IsFulfilledLCP false for a fulfilled file")
	}
	art, err := os.ReadFile(ful.ArtifactPath(file.SHA256))
	if err != nil {
		t.Fatalf("artifact not in cache: %v", err)
	}
	if int64(len(art)) != file.Size {
		t.Fatalf("size %d != catalog %d", len(art), file.Size)
	}
	sum := sha256.Sum256(art)
	if hex.EncodeToString(sum[:]) != file.SHA256 {
		t.Fatal("catalog hash is not the artifact's")
	}
	// The stored artifact is a valid LCP-protected EPUB: it carries the license
	// byte-identically and unlocks with the passphrase to the original chapter.
	c, err := lcp.OpenContainer(bytes.NewReader(art), int64(len(art)))
	if err != nil {
		t.Fatal(err)
	}
	if c.License == nil || !bytes.Equal(c.License.Raw, fx.License) {
		t.Fatal("embedded license is not the received one")
	}
	if _, err := c.Unlock(lcpfixture.Passphrase, lcp.CertificatePolicy{Roots: rootPool(fx)}); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	got, err := c.Resource("OEBPS/chapter1.xhtml", 1<<20)
	if err != nil || string(got) != lcpfixture.Chapter {
		t.Fatalf("chapter after unlock = %q, %v", got, err)
	}
	// Source bytes untouched (HRD-004).
	if after := dirDigest(t, root); after != before {
		t.Fatalf("source directory changed:\n%s\n---\n%s", before, after)
	}
	if bytes.Contains(art, []byte(lcpfixture.Passphrase)) {
		t.Fatal("passphrase leaked into the artifact")
	}

	// Idempotent rescan: same IDs, no second download.
	report2, err := s.ScanSource(context.Background(), "src-1", connector, "")
	if err != nil || report2.Files != 1 || len(report2.Skipped) != 0 {
		t.Fatalf("rescan = %+v, %v", report2, err)
	}
	page2, _ := repo.Search(context.Background(), catalog.Query{Text: "Sealed"})
	if len(page2.Works) != 1 || page2.Works[0].ID != page.Works[0].ID || page2.Works[0].Editions[0].Files[0].ID != file.ID {
		t.Fatalf("rescan changed identity: %+v", page2.Works)
	}
	if n := srv.hits.Load(); n != 1 {
		t.Fatalf("publication fetched %d times; the rescan must hit the cache", n)
	}
}

func TestLCPLWithoutFulfillerIsStillSkipped(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.lcpl"), []byte(`{}`), 0o644)
	connector, _ := sources.NewLocal(sources.Config{Root: root})
	s := Service{Repository: catalog.NewMemory()}
	report, err := s.Scan(context.Background(), connector, "")
	if err != nil || report.Files != 0 || len(report.Skipped) != 1 {
		t.Fatalf("report = %+v, %v", report, err)
	}
}

func TestLCPLFailurePathsRefuseAndLeaveNoArtifact(t *testing.T) {
	ctx := context.Background()
	t.Run("hash mismatch", func(t *testing.T) {
		var fx lcpfixture.Fixture
		srv := newProvider(t, func() []byte { b := append([]byte{}, fx.Publication...); b[len(b)/2] ^= 0xff; return b })
		fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
		f := newFulfiller(t)
		_, err := f.Fulfill(ctx, fx.License)
		if !errors.Is(err, lcp.ErrPublicationMismatch) {
			t.Fatalf("err = %v", err)
		}
		assertNoArtifacts(t, f)
	})
	t.Run("length mismatch", func(t *testing.T) {
		var fx lcpfixture.Fixture
		srv := newProvider(t, func() []byte { return append(append([]byte{}, fx.Publication...), 0) })
		fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
		f := newFulfiller(t)
		if _, err := f.Fulfill(ctx, fx.License); !errors.Is(err, lcp.ErrPublicationMismatch) {
			t.Fatalf("err = %v", err)
		}
		assertNoArtifacts(t, f)
	})
	t.Run("unreachable host", func(t *testing.T) {
		l, _ := net.Listen("tcp", "127.0.0.1:0")
		addr := l.Addr().String()
		l.Close() // nothing listens here any more
		fx := lcpfixture.Build(t, "http://"+addr+"/p.epub", nil)
		f := newFulfiller(t)
		_, err := f.Fulfill(ctx, fx.License)
		if !errors.Is(err, ErrLCPFetch) {
			t.Fatalf("err = %v", err)
		}
		assertNoArtifacts(t, f)
	})
	t.Run("private address blocked by default and never contacted", func(t *testing.T) {
		var fx lcpfixture.Fixture
		srv := newProvider(t, func() []byte { return fx.Publication })
		fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
		f := newFulfiller(t)
		f.AllowPrivateNetwork = false
		_, err := f.Fulfill(ctx, fx.License)
		if !errors.Is(err, ErrLCPPrivateAddress) {
			t.Fatalf("err = %v", err)
		}
		if srv.hits.Load() != 0 {
			t.Fatal("the private host was contacted")
		}
		assertNoArtifacts(t, f)
	})
	t.Run("redirect to a private address is blocked", func(t *testing.T) {
		target := newProvider(t, func() []byte { return nil })
		front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL+"/secret", http.StatusFound)
		}))
		defer front.Close()
		fx := lcpfixture.Build(t, "http://93.184.216.34/p.epub", nil) // a public-looking literal
		f := newFulfiller(t)
		f.AllowPrivateNetwork = false
		// Route only the public-looking address to the front server; every
		// later dial (the redirect to 127.0.0.1) hits the real policy check.
		f.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, front.Listener.Addr().String())
		}
		_, err := f.Fulfill(ctx, fx.License)
		if !errors.Is(err, ErrLCPPrivateAddress) {
			t.Fatalf("err = %v", err)
		}
		if target.hits.Load() != 0 {
			t.Fatal("redirect target was contacted")
		}
	})
	t.Run("oversized body", func(t *testing.T) {
		var fx lcpfixture.Fixture
		srv := newProvider(t, func() []byte { return fx.Publication })
		fx = lcpfixture.Build(t, srv.URL+"/p.epub", nil)
		f := newFulfiller(t)
		f.MaxPublicationBytes = int64(len(fx.Publication)) - 1
		if _, err := f.Fulfill(ctx, fx.License); !errors.Is(err, ErrLCPTooLarge) {
			t.Fatalf("err = %v", err)
		}
		assertNoArtifacts(t, f)
	})
	t.Run("oversized chunked body without Content-Length", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fl := w.(http.Flusher)
			for i := 0; i < 64; i++ {
				w.Write(bytes.Repeat([]byte("x"), 1024))
				fl.Flush()
			}
		}))
		defer srv.Close()
		// No stated length: the configured cap alone must stop the download.
		fx := lcpfixture.Build(t, srv.URL+"/p.epub", func(l map[string]any) { delete(l, "hash"); delete(l, "length") })
		f := newFulfiller(t)
		f.AllowUnverified = true
		f.MaxPublicationBytes = 4096
		if _, err := f.Fulfill(ctx, fx.License); !errors.Is(err, ErrLCPTooLarge) {
			t.Fatalf("err = %v", err)
		}
		assertNoArtifacts(t, f)
	})
	t.Run("wrong type is explicitly unsupported and not fetched", func(t *testing.T) {
		srv := newProvider(t, func() []byte { return []byte("%PDF-") })
		fx := lcpfixture.Build(t, srv.URL+"/p.pdf", func(l map[string]any) { l["type"] = "application/pdf" })
		f := newFulfiller(t)
		if _, err := f.Fulfill(ctx, fx.License); !errors.Is(err, lcp.ErrUnsupportedFormat) {
			t.Fatalf("err = %v", err)
		}
		if srv.hits.Load() != 0 {
			t.Fatal("an unsupported publication was downloaded")
		}
	})
	t.Run("templated href", func(t *testing.T) {
		fx := lcpfixture.Build(t, "http://x.invalid/{id}", func(l map[string]any) { l["templated"] = true })
		if _, err := newFulfiller(t).Fulfill(ctx, fx.License); !errors.Is(err, lcp.ErrTemplatedHref) {
			t.Fatalf("err = %v", err)
		}
	})
	for _, href := range []string{"file:///etc/passwd", "ftp://example.org/p.epub", "http://user:pw@example.org/p.epub", "//example.org/p.epub"} {
		t.Run("bad href "+href, func(t *testing.T) {
			fx := lcpfixture.Build(t, href, nil)
			if _, err := newFulfiller(t).Fulfill(ctx, fx.License); !errors.Is(err, ErrLCPBadURL) {
				t.Fatalf("err = %v", err)
			}
		})
	}
	t.Run("slow provider times out", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(2 * time.Second) }))
		defer srv.Close()
		fx := lcpfixture.Build(t, srv.URL+"/p.epub", nil)
		f := newFulfiller(t)
		f.Timeout = 200 * time.Millisecond
		start := time.Now()
		_, err := f.Fulfill(ctx, fx.License)
		if !errors.Is(err, ErrLCPFetch) || time.Since(start) > time.Second {
			t.Fatalf("err = %v after %v", err, time.Since(start))
		}
	})
	t.Run("http error status", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		defer srv.Close()
		fx := lcpfixture.Build(t, srv.URL+"/p.epub", nil)
		if _, err := newFulfiller(t).Fulfill(ctx, fx.License); !errors.Is(err, ErrLCPFetch) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("unverifiable link fails closed unless opted in", func(t *testing.T) {
		var fx lcpfixture.Fixture
		srv := newProvider(t, func() []byte { return fx.Publication })
		fx = lcpfixture.Build(t, srv.URL+"/p.epub", func(l map[string]any) { delete(l, "hash"); delete(l, "length") })
		f := newFulfiller(t)
		if _, err := f.Fulfill(ctx, fx.License); !errors.Is(err, lcp.ErrPublicationUnverifiable) {
			t.Fatalf("err = %v", err)
		}
		assertNoArtifacts(t, f)
		f.AllowUnverified = true
		if _, err := f.Fulfill(ctx, fx.License); err != nil {
			t.Fatalf("opted-in err = %v", err)
		}
	})
	t.Run("not a license", func(t *testing.T) {
		if _, err := newFulfiller(t).Fulfill(ctx, []byte("<html/>")); err == nil {
			t.Fatal("garbage accepted")
		}
	})
}

func TestLCPLScanReportsFailureWithoutLeakingURLs(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := l.Addr().String()
	l.Close()
	fx := lcpfixture.Build(t, "http://"+addr+"/p.epub?token=SECRETTOKEN", nil)
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.lcpl"), fx.License, 0o644)
	connector, _ := sources.NewLocal(sources.Config{Root: root})
	s := Service{Repository: catalog.NewMemory(), LCP: newFulfiller(t)}
	report, err := s.Scan(context.Background(), connector, "")
	if err != nil || report.Files != 0 || len(report.Skipped) != 1 {
		t.Fatalf("report = %+v, %v", report, err)
	}
	if strings.Contains(report.Skipped[0], "SECRETTOKEN") {
		t.Fatalf("token echoed: %s", report.Skipped[0])
	}
	if !strings.Contains(report.Skipped[0], "LCP fulfillment") {
		t.Fatalf("skip reason unclear: %s", report.Skipped[0])
	}
}

func TestLCPArtifactPathRejectsNonHashes(t *testing.T) {
	f := &LCPFulfiller{CacheDir: "/cache"}
	for _, bad := range []string{"", "../x", strings.Repeat("A", 64), strings.Repeat("a", 63), "/etc/passwd"} {
		if p := f.ArtifactPath(bad); p != "" {
			t.Errorf("ArtifactPath(%q) = %q", bad, p)
		}
	}
	if p := f.ArtifactPath(strings.Repeat("a", 64)); !strings.HasPrefix(p, "/cache/") {
		t.Errorf("good hash path = %q", p)
	}
}

func TestRestrictedIP(t *testing.T) {
	for ip, want := range map[string]bool{
		"127.0.0.1": true, "10.1.2.3": true, "192.168.0.1": true, "172.16.0.1": true, "169.254.169.254": true,
		"100.64.0.1": true, "0.0.0.0": true, "::1": true, "fe80::1": true, "fd00::1": true, "::ffff:127.0.0.1": true,
		"224.0.0.1": true, "93.184.216.34": false, "8.8.8.8": false, "2606:4700::1111": false,
	} {
		if got := restrictedIP(net.ParseIP(ip)); got != want {
			t.Errorf("restrictedIP(%s) = %v, want %v", ip, got, want)
		}
	}
}

func assertNoArtifacts(t *testing.T, f *LCPFulfiller) {
	t.Helper()
	for _, n := range cacheEntries(t, f) {
		if strings.HasSuffix(n, ".epub") || strings.HasSuffix(n, ".idx") || strings.HasSuffix(n, ".tmp") {
			t.Errorf("failed fulfillment left %s in the cache", n)
		}
	}
}

func dirDigest(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, _ := os.ReadFile(p)
		sum := sha256.Sum256(data)
		b.WriteString(p + " " + hex.EncodeToString(sum[:]) + " " + info.ModTime().String() + "\n")
		return nil
	})
	return b.String()
}

func rootPool(fx lcpfixture.Fixture) []*x509.Certificate { return []*x509.Certificate{fx.Root} }
