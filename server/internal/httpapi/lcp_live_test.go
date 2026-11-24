package httpapi

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/lcp"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/ingest/lcpfixture"
)

// TestLCPLiveServerQualification drives a freshly built helix-server process.
// It runs only when HELIX_LCP_LIVE_BIN names that binary and
// HELIX_LCP_LIVE_EVIDENCE names the directory for the machine evidence; see
// evidence/campaign-2026-11-23/opds-lcp/run.sh.
func TestLCPLiveServerQualification(t *testing.T) {
	bin, evDir := os.Getenv("HELIX_LCP_LIVE_BIN"), os.Getenv("HELIX_LCP_LIVE_EVIDENCE")
	if bin == "" || evDir == "" {
		t.Skip("set HELIX_LCP_LIVE_BIN and HELIX_LCP_LIVE_EVIDENCE to run the live qualification")
	}
	var checks []map[string]any
	check := func(name string, ok bool, detail string) {
		t.Helper()
		status := "PASS"
		if !ok {
			status = "FAIL"
			t.Errorf("%s: %s", name, detail)
		}
		checks = append(checks, map[string]any{"name": name, "status": status, "detail": detail})
	}

	// Provider on loopback serving the unlicensed encrypted EPUB.
	var fx lcpfixture.Fixture
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/epub+zip")
		w.Write(fx.Publication)
	}))
	defer provider.Close()
	fx = lcpfixture.Build(t, provider.URL+"/books/sealed.epub", nil)

	// The expected artifact is produced independently, in this process, by the
	// core library the server also uses.
	expected, err := lcp.Fulfill(fx.License, bytes.NewReader(fx.Publication), int64(len(fx.Publication)))
	if err != nil {
		t.Fatal(err)
	}
	expSum := sha256.Sum256(expected)

	work := t.TempDir()
	library := filepath.Join(work, "library")
	cache := filepath.Join(work, "lcp-cache")
	os.MkdirAll(library, 0o755)
	licPath := filepath.Join(library, "Sealed Book.lcpl")
	os.WriteFile(licPath, fx.License, 0o644)
	licBefore := sha256.Sum256(fx.License)

	l, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := l.Addr().String()
	l.Close()
	base := "http://" + addr
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "HELIX_ADDRESS="+addr, "HELIX_HTTP3=false", "HELIX_ADMIN_TOKEN=live-admin-token",
		"HELIX_OPDS_USERNAME=reader", "HELIX_OPDS_PASSWORD=live-reader-pw", "HELIX_POSTGRES_DSN=", "HELIX_REDIS_URL=",
		"HELIX_LCP_CACHE_DIR="+cache, "HELIX_LCP_ALLOW_PRIVATE_NETWORK=1", "HELIX_COVER_CACHE="+filepath.Join(work, "covers"))
	var serverLog bytes.Buffer
	cmd.Stdout, cmd.Stderr = &serverLog, &serverLog
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	client := &http.Client{Timeout: 10 * time.Second}
	do := func(method, path string, body []byte, hdr map[string]string, auth string) (int, http.Header, []byte) {
		req, _ := http.NewRequest(method, base+path, bytes.NewReader(body))
		switch auth {
		case "admin":
			req.Header.Set("Authorization", "Bearer live-admin-token")
		case "reader":
			req.SetBasicAuth("reader", "live-reader-pw")
		}
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header, b
	}
	for i := 0; ; i++ {
		if c, _, _ := func() (int, http.Header, []byte) {
			resp, err := client.Get(base + "/healthz")
			if err != nil {
				return 0, nil, nil
			}
			resp.Body.Close()
			return resp.StatusCode, nil, nil
		}(); c == 200 {
			break
		}
		if i > 100 {
			t.Fatalf("server did not start:\n%s", serverLog.String())
		}
		time.Sleep(100 * time.Millisecond)
	}

	src, _ := json.Marshal(map[string]string{"id": "live-lcp", "protocol": "local", "root": library})
	code, _, body := do("POST", "/v1/sources", src, map[string]string{"Content-Type": "application/json"}, "admin")
	check("source created", code == 201 || code == 200, fmt.Sprintf("status %d %s", code, body))
	scan := func() map[string]any {
		sb, _ := json.Marshal(map[string]string{"source_id": "live-lcp", "prefix": ""})
		code, _, body := do("POST", "/v1/scans", sb, map[string]string{"Content-Type": "application/json"}, "admin")
		if code != 202 && code != 200 && code != 201 {
			t.Fatalf("scan start: %d %s", code, body)
		}
		var job map[string]any
		json.Unmarshal(body, &job)
		for i := 0; i < 200; i++ {
			_, _, jb := do("GET", "/v1/scans/"+job["id"].(string), nil, nil, "admin")
			json.Unmarshal(jb, &job)
			if st, _ := job["state"].(string); st == "succeeded" || st == "failed" || st == "canceled" {
				return job
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("scan did not finish: %v", job)
		return nil
	}
	job := scan()
	jb, _ := json.Marshal(job)
	check("scan finished without skipping the license", strings.Contains(string(jb), `"files":1`) && !strings.Contains(string(jb), "LCP fulfillment:"), string(jb))

	// OPDS 1.2 feed with the acquisition link.
	code, _, feed := do("GET", "/opds/1.2/publications", nil, nil, "reader")
	check("opds feed requires credentials", func() bool { c, _, _ := do("GET", "/opds/1.2/publications", nil, nil, ""); return c == 401 }(), "")
	var parsed struct {
		Entries []struct {
			Title string `xml:"title"`
			Links []struct {
				Rel  string `xml:"rel,attr"`
				Href string `xml:"href,attr"`
				Type string `xml:"type,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(feed, &parsed); err != nil || code != 200 {
		t.Fatalf("feed: %d %v\n%s", code, err, feed)
	}
	var acq, lic string
	for _, e := range parsed.Entries {
		for _, ln := range e.Links {
			if ln.Rel == "http://opds-spec.org/acquisition" && ln.Type == "application/epub+zip" {
				t.Errorf("protected EPUB advertised as a plain acquisition: %s", ln.Href)
			}
			if ln.Rel == "http://opds-spec.org/acquisition" && ln.Type == ingest.LCPLicenseMediaType {
				lic = ln.Href
			}
		}
	}
	// The fulfilled artifact stays downloadable by id even though the feed only
	// advertises the license acquisition.
	if lic != "" {
		acq = strings.TrimSuffix(lic, "/license") + "/content"
	}
	check("feed carries the license acquisition link (rel acquisition, license type)", lic != "" && strings.Contains(string(feed), "indirectAcquisition"), fmt.Sprintf("license=%q", lic))
	if acq == "" || lic == "" {
		return
	}

	// Download through the link; byte-compare with the independently expected file.
	code, hdr, got := do("GET", acq, nil, nil, "reader")
	gotSum := sha256.Sum256(got)
	check("download is byte-identical to the independently fulfilled EPUB", code == 200 && bytes.Equal(got, expected), fmt.Sprintf("status %d bytes %d sha256 %x", code, len(got), gotSum))
	check("download disposition and type", hdr.Get("Content-Type") == "application/epub+zip" && hdr.Get("Content-Disposition") == `attachment; filename="Sealed Book.epub"`, hdr.Get("Content-Disposition"))
	code, _, part := do("GET", acq, nil, map[string]string{"Range": "bytes=10-29"}, "reader")
	check("range request", code == 206 && bytes.Equal(part, expected[10:30]), fmt.Sprintf("status %d", code))
	code, _, _ = do("GET", acq, nil, nil, "")
	check("download requires credentials", code == 401, fmt.Sprintf("status %d", code))

	code, _, lbody := do("GET", lic, nil, nil, "reader")
	check("license served byte-identical to the source .lcpl", code == 200 && bytes.Equal(lbody, fx.License), fmt.Sprintf("status %d", code))

	// Unlock round trip on the DOWNLOADED bytes: proves a valid LCP-protected EPUB.
	c, err := lcp.OpenContainer(bytes.NewReader(got), int64(len(got)))
	if err != nil {
		t.Fatal(err)
	}
	_, uerr := c.Unlock(lcpfixture.Passphrase, lcp.CertificatePolicy{Roots: []*x509.Certificate{fx.Root}})
	chapter, rerr := c.Resource("OEBPS/chapter1.xhtml", 1<<20)
	check("unlock round trip with the test passphrase", uerr == nil && rerr == nil && string(chapter) == lcpfixture.Chapter, fmt.Sprintf("unlock=%v resource=%v", uerr, rerr))
	_, werr := func() (any, error) {
		c2, _ := lcp.OpenContainer(bytes.NewReader(got), int64(len(got)))
		return c2.Unlock("wrong passphrase", lcp.CertificatePolicy{Roots: []*x509.Certificate{fx.Root}})
	}()
	check("wrong passphrase is refused", werr != nil, fmt.Sprint(werr))

	// Rescan: idempotent, no second download, identity unchanged.
	job2 := scan()
	jb2, _ := json.Marshal(job2)
	_, _, feed2 := do("GET", "/opds/1.2/publications", nil, nil, "reader")
	check("rescan keeps the same license acquisition link", bytes.Contains(feed2, []byte(lic)) && strings.Contains(string(jb2), `"files":1`), string(jb2))

	// Source untouched; no secrets in the server log.
	after, _ := os.ReadFile(licPath)
	afterSum := sha256.Sum256(after)
	ents, _ := os.ReadDir(library)
	check("source directory untouched (HRD-004)", afterSum == licBefore && len(ents) == 1, fmt.Sprintf("%d entries", len(ents)))
	check("server log holds no passphrase", !strings.Contains(serverLog.String(), lcpfixture.Passphrase) && !strings.Contains(serverLog.String(), "live-reader-pw"), "")
	cached, _ := os.ReadDir(cache)
	var names []string
	for _, e := range cached {
		names = append(names, e.Name())
	}
	check("cache holds exactly the artifact and its license index", len(cached) == 2, strings.Join(names, ","))

	out := map[string]any{
		"campaign": "2026-11-23", "item": "HRD-031", "scope": "LCP license -> fulfilled EPUB over OPDS, in-memory catalog",
		"expected_artifact_sha256": hex.EncodeToString(expSum[:]), "expected_artifact_bytes": len(expected),
		"downloaded_sha256": hex.EncodeToString(gotSum[:]), "downloaded_bytes": len(got),
		"acquisition_href": acq, "license_href": lic, "checks": checks,
		"not_verified": []string{"real LCP provider or Readium LCP server", "third-party LCP reader unlock", "PostgreSQL-backed catalog for this flow", "license signature trust chain on the server (by design; readers verify)"},
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	os.MkdirAll(evDir, 0o755)
	os.WriteFile(filepath.Join(evDir, "results.json"), data, 0o644)
	os.WriteFile(filepath.Join(evDir, "feed-1.2.xml"), feed, 0o644)
}
