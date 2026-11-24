package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/ingest/lcpfixture"
	"github.com/helixreader/helix/server/internal/sources"
)

type lcpRig struct {
	srv      *Server
	fx       lcpfixture.Fixture
	file     catalog.File
	editionI string
	cacheDir string
	root     string
}

func newLCPRig(t *testing.T, withCreds bool) *lcpRig {
	t.Helper()
	var fx lcpfixture.Fixture
	pub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(fx.Publication) }))
	t.Cleanup(pub.Close)
	fx = lcpfixture.Build(t, pub.URL+"/pub.epub", nil)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Sealed Book.lcpl"), fx.License, 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(t.TempDir(), "lcp")
	repo := catalog.NewMemory()
	cfg := config.Config{AdminToken: "secret", BrotliLevel: 4}
	if withCreds {
		cfg.OPDSUsername, cfg.OPDSPassword = "reader", "pw"
	}
	s := New(cfg, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo, LCP: &ingest.LCPFulfiller{CacheDir: cache, AllowPrivateNetwork: true}})
	t.Cleanup(func() { s.Close() })
	s.connectors["src-1"] = connector
	report, err := s.Ingest.ScanSource(context.Background(), "src-1", connector, "")
	if err != nil || report.Files != 1 {
		t.Fatalf("scan = %+v, %v", report, err)
	}
	page, _ := repo.Search(context.Background(), catalog.Query{})
	e := page.Works[0].Editions[0]
	return &lcpRig{srv: s, fx: fx, file: e.Files[0], editionI: e.ID, cacheDir: cache, root: root}
}

func (r *lcpRig) get(t *testing.T, method, path string, hdr map[string]string, auth bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if auth {
		req.SetBasicAuth("reader", "pw")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.srv.Router().ServeHTTP(w, req)
	return w
}

func TestLCPFulfilledContentIsServedByteExactAndRanged(t *testing.T) {
	r := newLCPRig(t, false)
	want, err := os.ReadFile(r.srv.Ingest.LCP.ArtifactPath(r.file.SHA256))
	if err != nil {
		t.Fatal(err)
	}
	w := r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/content", nil, false)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), want) {
		t.Fatalf("status %d, %d bytes (want %d)", w.Code, w.Body.Len(), len(want))
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/epub+zip" {
		t.Errorf("content-type = %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); cd != `attachment; filename="Sealed Book.epub"` {
		t.Errorf("content-disposition = %q", cd)
	}
	if et := w.Header().Get("ETag"); et != `"sha256-`+r.file.SHA256+`"` {
		t.Errorf("etag = %q", et)
	}
	w = r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/content", map[string]string{"Range": "bytes=4-9"}, false)
	if w.Code != http.StatusPartialContent || !bytes.Equal(w.Body.Bytes(), want[4:10]) {
		t.Fatalf("range: %d %q", w.Code, w.Body.Bytes())
	}
	w = r.get(t, http.MethodHead, "/v1/files/"+r.file.ID+"/content", nil, false)
	if w.Code != 200 || w.Body.Len() != 0 || w.Header().Get("Content-Length") == "" {
		t.Fatalf("head: %d len=%d", w.Code, w.Body.Len())
	}
}

func TestLCPLicenseRouteServesTheLicenseOnly(t *testing.T) {
	r := newLCPRig(t, false)
	w := r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/license", nil, false)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), r.fx.License) {
		t.Fatalf("license: %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != ingest.LCPLicenseMediaType {
		t.Errorf("content-type = %q", ct)
	}
	if bytes.Contains(w.Body.Bytes(), []byte(lcpfixture.Passphrase)) {
		t.Fatal("passphrase in license response")
	}
	if w := r.get(t, http.MethodGet, "/v1/files/nope/license", nil, false); w.Code != 404 {
		t.Errorf("unknown file: %d", w.Code)
	}
}

func TestLCPLicenseRouteRefusesNonLCPFiles(t *testing.T) {
	s, _, _ := testServer(t) // plain book.txt, id f1
	defer s.Close()
	if w := request(t, s.Router(), http.MethodGet, "/v1/files/f1/license", "", nil, nil); w.Code != 404 {
		t.Fatalf("plain file license: %d", w.Code)
	}
}

func TestLCPRoutesRequireOPDSCredentialsWhenConfigured(t *testing.T) {
	r := newLCPRig(t, true)
	for _, p := range []string{"/v1/files/" + r.file.ID + "/content", "/v1/files/" + r.file.ID + "/license", "/opds/1.2/publications", "/opds/2.0/publications"} {
		if w := r.get(t, http.MethodGet, p, nil, false); w.Code != http.StatusUnauthorized {
			t.Errorf("%s without credentials: %d", p, w.Code)
		}
		if w := r.get(t, http.MethodGet, p, nil, true); w.Code != 200 {
			t.Errorf("%s with credentials: %d", p, w.Code)
		}
	}
}

func TestLCPOPDSFeedsAdvertiseFulfilledEPUBAndLicense(t *testing.T) {
	r := newLCPRig(t, false)
	content := "/v1/files/" + r.file.ID + "/content"
	license := "/v1/files/" + r.file.ID + "/license"
	for _, p := range []string{"/opds/1.2/publications", "/opds/1.2/editions/" + r.editionI} {
		body := r.get(t, http.MethodGet, p, nil, false).Body.String()
		if strings.Contains(body, `href="`+content+`"`) {
			t.Errorf("%s offers the protected EPUB as a plain acquisition:\n%s", p, body)
		}
		if !strings.Contains(body, `rel="http://opds-spec.org/acquisition" href="`+license+`" type="application/vnd.readium.lcp.license.v1.0+json"`) || !strings.Contains(body, `<opds:indirectAcquisition type="application/epub+zip">`) {
			t.Errorf("%s lacks the license acquisition link with indirectAcquisition:\n%s", p, body)
		}
	}
	for _, p := range []string{"/opds/2.0/publications", "/opds/2.0/editions/" + r.editionI} {
		body := r.get(t, http.MethodGet, p, nil, false).Body.String()
		var anyJSON any
		if err := json.Unmarshal([]byte(body), &anyJSON); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if strings.Contains(body, `"href": "`+content+`"`) || !strings.Contains(body, `"href": "`+license+`"`) || !strings.Contains(body, ingest.LCPLicenseMediaType) || !strings.Contains(body, `"indirectAcquisition"`) {
			t.Errorf("%s lacks links:\n%s", p, body)
		}
	}
}

func TestLCPContentFailsClosedWhenCacheUnavailable(t *testing.T) {
	r := newLCPRig(t, false)
	path := r.srv.Ingest.LCP.ArtifactPath(r.file.SHA256)
	if err := os.WriteFile(path, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if w := r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/content", nil, false); w.Code != http.StatusConflict {
		t.Errorf("size-mismatched artifact: %d", w.Code)
	}
	os.Remove(path)
	if w := r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/content", nil, false); w.Code != http.StatusConflict {
		t.Errorf("missing artifact: %d", w.Code)
	}
	r.srv.Ingest.LCP = nil
	if w := r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/content", nil, false); w.Code != http.StatusConflict {
		t.Errorf("fulfillment disabled: %d", w.Code)
	}
}

func TestLCPDisabledByDefaultFromEnv(t *testing.T) {
	t.Setenv("HELIX_LCP_CACHE_DIR", "")
	repo := catalog.NewMemory()
	if s := New(config.Config{}, repo, nil, &ingest.Service{Repository: repo}); s.Ingest.LCP != nil {
		t.Fatal("LCP enabled without HELIX_LCP_CACHE_DIR")
	}
	t.Setenv("HELIX_LCP_CACHE_DIR", t.TempDir())
	t.Setenv("HELIX_LCP_ALLOW_PRIVATE_NETWORK", "1")
	if s := New(config.Config{}, repo, nil, &ingest.Service{Repository: repo}); s.Ingest.LCP == nil || !s.Ingest.LCP.AllowPrivateNetwork {
		t.Fatalf("env not honored: %+v", s.Ingest.LCP)
	}
}
