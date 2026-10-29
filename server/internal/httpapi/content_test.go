package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/covers"
	"github.com/helixreader/helix/server/internal/sources"
)

// sizeBlindConnector hides any live-size revalidation method so tests can
// exercise the unknown-size fallback path with a local filesystem behind it.
type sizeBlindConnector struct{ sources.Connector }

type handleTrackingConnector struct {
	sources.Connector
	handleOpens int
	pathOpens   int
	revalidates int
}

func (c *handleTrackingConnector) OpenRevalidated(ctx context.Context, path string) (io.ReadSeekCloser, int64, time.Time, error) {
	c.handleOpens++
	return c.Connector.(interface {
		OpenRevalidated(context.Context, string) (io.ReadSeekCloser, int64, time.Time, error)
	}).OpenRevalidated(ctx, path)
}
func (c *handleTrackingConnector) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	c.pathOpens++
	return c.Connector.Open(ctx, path)
}
func (c *handleTrackingConnector) Revalidate(ctx context.Context, path string) (int64, time.Time, error) {
	c.revalidates++
	return c.Connector.(interface {
		Revalidate(context.Context, string) (int64, time.Time, error)
	}).Revalidate(ctx, path)
}

func contentClient(t *testing.T, s *Server) (*httptest.Server, *http.Client) {
	t.Helper()
	server := httptest.NewServer(s.Router())
	t.Cleanup(server.Close)
	return server, &http.Client{Transport: &http.Transport{DisableCompression: true}}
}

func doContent(t *testing.T, client *http.Client, method, url string, headers map[string]string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, resp.Header, body
}

func writeBook(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "book.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The catalog records Size:10 from scan time; the live file changes after.
func TestContentRevalidatesSizeAfterGrowthAndShrink(t *testing.T) {
	s, _, root := testServer(t)
	defer s.Close()
	server, client := contentClient(t, s)
	url := server.URL + "/v1/files/f1/content"

	grown := strings.Repeat("0123456789", 5) // 50 bytes, catalog still says 10
	writeBook(t, root, grown)
	status, header, body := doContent(t, client, http.MethodGet, url, nil)
	if status != http.StatusOK || string(body) != grown {
		t.Fatalf("growth: status=%d body=%q", status, body)
	}
	if got := header.Get("Content-Length"); got != "50" {
		t.Fatalf("growth served stale Content-Length %q (truncation risk)", got)
	}

	shrunk := "abcd"
	writeBook(t, root, shrunk)
	status, header, body = doContent(t, client, http.MethodGet, url, nil)
	if status != http.StatusOK || string(body) != shrunk {
		t.Fatalf("shrink: status=%d body=%q", status, body)
	}
	if got := header.Get("Content-Length"); got != "4" {
		t.Fatalf("shrink served stale Content-Length %q", got)
	}
	// Once live metadata differs from the scan, ranges and the old strong ETag
	// are no longer safe. The complete current representation is returned.
	status, header, body = doContent(t, client, http.MethodGet, url, map[string]string{"Range": "bytes=0-9"})
	if status != http.StatusOK || string(body) != shrunk {
		t.Fatalf("changed-file range fallback: status=%d body=%q", status, body)
	}
	if header.Get("Content-Range") != "" || header.Get("ETag") != "" {
		t.Fatalf("changed-file range leaked stale validators: %+v", header)
	}
}

func TestContentIfRangeHonorsETag(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	server, client := contentClient(t, s)
	url := server.URL + "/v1/files/f1/content"

	status, header, _ := doContent(t, client, http.MethodGet, url, nil)
	etag := header.Get("ETag")
	if status != http.StatusOK || etag != `"sha256-`+testSHA256+`"` {
		t.Fatalf("etag: %d %q", status, etag)
	}

	status, header, body := doContent(t, client, http.MethodGet, url, map[string]string{
		"Range": "bytes=2-5", "If-Range": etag,
	})
	if status != http.StatusPartialContent || string(body) != "2345" || header.Get("Content-Range") != "bytes 2-5/10" {
		t.Fatalf("matching If-Range must serve 206: %d %q %q", status, body, header.Get("Content-Range"))
	}

	for _, stale := range []string{`"sha256-stale"`, "Wed, 21 Oct 2015 07:28:00 GMT"} {
		status, header, body = doContent(t, client, http.MethodGet, url, map[string]string{
			"Range": "bytes=2-5", "If-Range": stale,
		})
		if status != http.StatusOK || string(body) != "0123456789" || header.Get("Content-Range") != "" {
			t.Fatalf("stale If-Range %q must force full 200: %d %q %q", stale, status, body, header.Get("Content-Range"))
		}
	}
}

func TestContentStaleETagCannotSpliceChangedEqualSizeFile(t *testing.T) {
	s, _, root := testServer(t)
	defer s.Close()
	server, client := contentClient(t, s)
	url := server.URL + "/v1/files/f1/content"

	_, header, _ := doContent(t, client, http.MethodGet, url, nil)
	stale := header.Get("ETag")
	if stale == "" {
		t.Fatal("unchanged file did not expose its strong ETag")
	}
	writeBook(t, root, "abcdefghij") // same size, different bytes
	path := filepath.Join(root, "book.txt")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := info.ModTime().Add(2 * time.Second)
	if err := os.Chtimes(path, changed, changed); err != nil {
		t.Fatal(err)
	}
	status, header, body := doContent(t, client, http.MethodGet, url, map[string]string{
		"Range": "bytes=0-3", "If-Range": stale,
	})
	if status != http.StatusOK || string(body) != "abcdefghij" {
		t.Fatalf("stale validator produced unsafe response: %d %q", status, body)
	}
	if header.Get("ETag") != "" || header.Get("Content-Range") != "" {
		t.Fatalf("changed bytes retained stale range metadata: %+v", header)
	}
}

func TestContentMultiRangeFallsBackToFull200(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	server, client := contentClient(t, s)
	status, header, body := doContent(t, client, http.MethodGet, server.URL+"/v1/files/f1/content", map[string]string{
		"Range": "bytes=0-1,4-5",
	})
	if status != http.StatusOK || string(body) != "0123456789" {
		t.Fatalf("multi-range must be ignored with a full 200: %d %q", status, body)
	}
	if header.Get("Content-Range") != "" || header.Get("Content-Length") != "10" {
		t.Fatalf("multi-range headers: %q %q", header.Get("Content-Range"), header.Get("Content-Length"))
	}
}

func TestContentUnknownSizeSafeFallback(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	local, ok := s.connectors["source-test"]
	if !ok {
		t.Fatal("missing test connector")
	}
	s.connectors["source-test"] = sizeBlindConnector{local}
	server, client := contentClient(t, s)
	url := server.URL + "/v1/files/f1/content"

	status, header, body := doContent(t, client, http.MethodGet, url, nil)
	if status != http.StatusOK || string(body) != "0123456789" {
		t.Fatalf("unknown-size full response: %d %q", status, body)
	}
	if got := header.Get("Content-Length"); got != "" {
		t.Fatalf("unknown size must omit Content-Length, got %q", got)
	}

	status, header, body = doContent(t, client, http.MethodGet, url, map[string]string{"Range": "bytes=2-5"})
	if status != http.StatusOK || string(body) != "0123456789" {
		t.Fatalf("unknown-size range must fall back to full 200: %d %q", status, body)
	}
	if header.Get("Content-Range") != "" || header.Get("Content-Length") != "" {
		t.Fatalf("unknown-size range leaked stale headers: %q %q", header.Get("Content-Range"), header.Get("Content-Length"))
	}
}

func TestContentAndCoverHEAD(t *testing.T) {
	s, repo, _ := testServer(t)
	defer s.Close()
	server, client := contentClient(t, s)

	status, header, body := doContent(t, client, http.MethodHead, server.URL+"/v1/files/f1/content", nil)
	if status != http.StatusOK || len(body) != 0 {
		t.Fatalf("content HEAD: %d body=%q", status, body)
	}
	if header.Get("Content-Length") != "10" || header.Get("ETag") != `"sha256-`+testSHA256+`"` {
		t.Fatalf("content HEAD headers: %+v", header)
	}
	status, header, body = doContent(t, client, http.MethodHead, server.URL+"/v1/files/f1/content", map[string]string{"Range": "bytes=2-5"})
	if status != http.StatusPartialContent || len(body) != 0 {
		t.Fatalf("content range HEAD: %d body=%q", status, body)
	}
	if header.Get("Content-Range") != "bytes 2-5/10" || header.Get("Content-Length") != "4" {
		t.Fatalf("content range HEAD headers: %+v", header)
	}

	// Cover route HEAD must serve headers without a body and without a
	// network fetch once the cover is cached.
	coverDir := t.TempDir()
	s.Covers = covers.New(coverDir)
	const coverURL = "https://covers.example/c.png"
	key := sha256.Sum256([]byte("e1\x00" + coverURL))
	png := []byte("\x89PNG\r\n\x1a\n-cover-bytes")
	if err := os.WriteFile(filepath.Join(coverDir, hex.EncodeToString(key[:])+".png"), png, 0o640); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	work := catalog.Work{ID: "w1", CanonicalTitle: "Book", CreatedAt: now, UpdatedAt: now, Editions: []catalog.Edition{{ID: "e1", Title: "Book", State: catalog.Identified, UpdatedAt: now, CoverHREF: coverURL, Files: []catalog.File{{ID: "f1", SourceID: "source-test", Path: "book.txt", Format: "txt", MIMEType: "text/plain", Size: 10, SHA256: testSHA256}}}}}
	if err := repo.UpsertWork(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	status, header, body = doContent(t, client, http.MethodHead, server.URL+"/v1/covers/e1", nil)
	if status != http.StatusOK || len(body) != 0 {
		t.Fatalf("cover HEAD: %d body=%q", status, body)
	}
	if header.Get("Content-Type") != "image/png" {
		t.Fatalf("cover HEAD content type: %q", header.Get("Content-Type"))
	}
}

// Binary downloads must not be recompressed: both 206 and full 200 content
// responses stay identity even when the client offers gzip/brotli.
func TestContentStaysIdentityUnderAcceptEncoding(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	server, client := contentClient(t, s)
	url := server.URL + "/v1/files/f1/content"

	status, header, body := doContent(t, client, http.MethodGet, url, map[string]string{
		"Range": "bytes=2-5", "Accept-Encoding": "br, gzip",
	})
	if status != http.StatusPartialContent || string(body) != "2345" {
		t.Fatalf("206 identity: %d %q", status, body)
	}
	if enc := header.Get("Content-Encoding"); enc != "" {
		t.Fatalf("206 recompressed as %q", enc)
	}
	if header.Get("Content-Length") != "4" {
		t.Fatalf("206 Content-Length: %q", header.Get("Content-Length"))
	}

	status, header, body = doContent(t, client, http.MethodGet, url, map[string]string{
		"Accept-Encoding": "br, gzip",
	})
	if status != http.StatusOK || string(body) != "0123456789" {
		t.Fatalf("200 identity: %d %q", status, body)
	}
	if enc := header.Get("Content-Encoding"); enc != "" {
		t.Fatalf("binary 200 recompressed as %q", enc)
	}
	if header.Get("Content-Length") != "10" {
		t.Fatalf("200 Content-Length: %q", header.Get("Content-Length"))
	}
}

func TestContentDeletedLocalFileReturns404(t *testing.T) {
	s, _, root := testServer(t)
	defer s.Close()
	if err := os.Remove(filepath.Join(root, "book.txt")); err != nil {
		t.Fatal(err)
	}
	server, client := contentClient(t, s)
	status, _, _ := doContent(t, client, http.MethodGet, server.URL+"/v1/files/f1/content", nil)
	if status != http.StatusNotFound {
		t.Fatalf("deleted local file returned %d, want 404", status)
	}
}

func TestContentGETValidatesAndReadsSameOpenedHandle(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	tracker := &handleTrackingConnector{Connector: s.connectors["source-test"]}
	s.connectors["source-test"] = tracker
	server, client := contentClient(t, s)
	url := server.URL + "/v1/files/f1/content"
	status, _, body := doContent(t, client, http.MethodGet, url, nil)
	if status != http.StatusOK || string(body) != "0123456789" {
		t.Fatalf("full response: %d %q", status, body)
	}
	status, _, body = doContent(t, client, http.MethodGet, url, map[string]string{"Range": "bytes=2-5"})
	if status != http.StatusPartialContent || string(body) != "2345" {
		t.Fatalf("range response: %d %q", status, body)
	}
	if tracker.handleOpens != 2 || tracker.pathOpens != 0 || tracker.revalidates != 0 {
		t.Fatalf("GET did not use one opened handle per response: %+v", tracker)
	}
}
