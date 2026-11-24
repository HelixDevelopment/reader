package httpapi

// HRD-036 / GAP-009: remote (WebDAV, rclone) sources must honour Range.
//
// Both connectors are exercised against a real `rclone serve webdav` target.
// The catalog row is built from the connector's own List entry, exactly as a
// scan does, so the live revalidation inside the content route compares
// like with like. No mocks: a skipped run is NOT a pass (the evidence runner
// fails on any SKIP).

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/sources"
)

func TestRemoteSourcesHonourRange(t *testing.T) {
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone not installed")
	}
	root := t.TempDir()
	body := make([]byte, 300_000)
	if _, err := rand.Read(body); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "books"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"r.bin", "s.bin"} {
		if err := os.WriteFile(filepath.Join(root, "books", n), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(body))
	cfgPath := filepath.Join(t.TempDir(), "rclone.conf")
	if err := os.WriteFile(cfgPath, []byte("[wfdav]\ntype = local\nnolink = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	addr := remoteTestFreeAddr(t)
	serve := exec.Command("rclone", "--config", cfgPath, "serve", "webdav", "wfdav:", "--addr", addr, "--dir-cache-time", "1ms")
	serve.Dir = root
	logs := &remoteTestLogBuffer{}
	serve.Stdout, serve.Stderr = logs, logs
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = serve.Process.Kill(); _ = serve.Wait() }()
	remoteTestWaitWebDAV(t, "http://"+addr, "books/r.bin", digest, logs)

	webdav, err := sources.NewWebDAV(sources.Config{Protocol: "webdav", Endpoint: "http://" + addr + "/"})
	if err != nil {
		t.Fatal(err)
	}
	rc, err := sources.NewRclone(sources.Config{Protocol: "sftp", Endpoint: "wfdav:" + root, RcloneConfig: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	for name, conn := range map[string]sources.Connector{"webdav": webdav, "rclone": rc} {
		conn := conn
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			entries, err := conn.List(ctx, "")
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			live := map[string]sources.Entry{}
			for _, en := range entries {
				if en.Size > 0 { // WebDAV listings can carry the directory row itself
					live[en.Path] = en
				}
			}
			if len(live) != 2 {
				t.Fatalf("list returned %d file rows: %v", len(live), entries)
			}
			e, se := live["books/r.bin"], live["books/s.bin"]
			repo := catalog.NewMemory()
			now := time.Now().UTC()
			mk := func(id, path string, size int64, mod time.Time) {
				err := repo.UpsertWork(ctx, catalog.Work{ID: id + "-w", CanonicalTitle: id, CreatedAt: now, UpdatedAt: now,
					Editions: []catalog.Edition{{ID: id + "-e", Title: id, State: catalog.Identified, UpdatedAt: now,
						Files: []catalog.File{{ID: id, SourceID: "src", Path: path, Format: "bin",
							MIMEType: "application/octet-stream", Size: size, SHA256: digest, ModifiedAt: mod}}}}})
				if err != nil {
					t.Fatal(err)
				}
			}
			mk("fresh", e.Path, e.Size, e.Modified)
			mk("stale", se.Path, se.Size+1, se.Modified) // catalog says a different size than the live source
			srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
			srv.connectors["src"] = conn
			ts := httptest.NewServer(srv.Router())
			defer ts.Close()
			client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
			get := func(id string, h map[string]string) (int, http.Header, []byte) {
				return doContent(t, client, http.MethodGet, ts.URL+"/v1/files/"+id+"/content", h)
			}

			st, h, b := get("fresh", map[string]string{"Range": "bytes=1000-2047"})
			if st != http.StatusPartialContent {
				t.Fatalf("Range GET status %d, want 206 (Accept-Ranges=%q body=%.120q)", st, h.Get("Accept-Ranges"), b)
			}
			if want := fmt.Sprintf("bytes 1000-2047/%d", len(body)); h.Get("Content-Range") != want {
				t.Fatalf("Content-Range %q want %q", h.Get("Content-Range"), want)
			}
			if !bytes.Equal(b, body[1000:2048]) {
				t.Fatal("206 body differs from source bytes")
			}
			if h.Get("Accept-Ranges") != "bytes" {
				t.Fatalf("Accept-Ranges %q", h.Get("Accept-Ranges"))
			}
			st, _, b = get("fresh", map[string]string{"Range": "bytes=-500"})
			if st != 206 || !bytes.Equal(b, body[len(body)-500:]) {
				t.Fatalf("suffix range status %d len %d", st, len(b))
			}
			st, h, b = get("fresh", nil)
			if st != 200 || !bytes.Equal(b, body) || h.Get("Accept-Ranges") != "bytes" {
				t.Fatalf("full GET status %d identical=%v accept=%q", st, bytes.Equal(b, body), h.Get("Accept-Ranges"))
			}
			st, _, _ = get("fresh", map[string]string{"Range": fmt.Sprintf("bytes=%d-", len(body)+10)})
			if st != http.StatusRequestedRangeNotSatisfiable {
				t.Fatalf("unsatisfiable range status %d, want 416", st)
			}
			// Stale catalog: never label live bytes with a stale validator or range.
			st, h, b = get("stale", map[string]string{"Range": "bytes=0-9"})
			if st != 200 || h.Get("ETag") != "" || h.Get("Accept-Ranges") != "" || !bytes.Equal(b, body) {
				t.Fatalf("stale catalog: status %d etag=%q accept=%q identical=%v", st, h.Get("ETag"), h.Get("Accept-Ranges"), bytes.Equal(b, body))
			}
			// Missing live file: 404, not a 5xx or a hang.
			if err := os.Remove(filepath.Join(root, "books", "r.bin")); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.WriteFile(filepath.Join(root, "books", "r.bin"), body, 0o644) }()
			st, _, _ = get("fresh", map[string]string{"Range": "bytes=0-9"})
			if st != http.StatusNotFound {
				t.Fatalf("deleted source file status %d, want 404", st)
			}
		})
	}
}

// Origin that never reports getcontentlength: the route must still deliver the
// real body (plain 200 or a correct range), never an empty 200.
func TestWebDAVWithoutContentLengthNeverServesEmpty(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789"), 100) // 1000 bytes
	mod := time.Date(2026, 11, 23, 21, 47, 33, 0, time.UTC)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PROPFIND" {
			w.WriteHeader(207)
			fmt.Fprintf(w, `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/b.bin</D:href><D:propstat><D:prop><D:getlastmodified>%s</D:getlastmodified><D:resourcetype/></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response></D:multistatus>`, mod.Format(http.TimeFormat))
			return
		}
		http.ServeContent(w, r, "b.bin", mod, bytes.NewReader(payload))
	}))
	defer origin.Close()
	conn, err := sources.NewWebDAV(sources.Config{Protocol: "webdav", Endpoint: origin.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	now := time.Now().UTC()
	if err := repo.UpsertWork(context.Background(), catalog.Work{ID: "w", CanonicalTitle: "w", CreatedAt: now, UpdatedAt: now,
		Editions: []catalog.Edition{{ID: "e", Title: "w", State: catalog.Identified, UpdatedAt: now,
			Files: []catalog.File{{ID: "f", SourceID: "src", Path: "b.bin", Format: "bin", MIMEType: "application/octet-stream", Size: 1000, SHA256: fmt.Sprintf("%x", sha256.Sum256(payload)), ModifiedAt: mod}}}}}); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	srv.connectors["src"] = conn
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	st, _, b := doContent(t, client, http.MethodGet, ts.URL+"/v1/files/f/content", nil)
	if st != 200 || !bytes.Equal(b, payload) {
		t.Fatalf("full GET: status %d, %d bytes (want 200 and 1000)", st, len(b))
	}
	st, h, b := doContent(t, client, http.MethodGet, ts.URL+"/v1/files/f/content", map[string]string{"Range": "bytes=10-19"})
	switch {
	case st == 206 && bytes.Equal(b, payload[10:20]):
	case st == 200 && bytes.Equal(b, payload) && h.Get("Content-Range") == "":
	default:
		t.Fatalf("range GET: status %d, %d bytes, Content-Range %q", st, len(b), h.Get("Content-Range"))
	}
}

// An origin that ignores Range at a non-zero offset makes OpenRange fail. The
// resulting 502 must not carry the 206-style Content-Range / Content-Length
// headers that were prepared for the success path.
func TestOpenRangeFailureDoesNotLeakPartialHeaders(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789"), 100)
	mod := time.Date(2026, 11, 23, 21, 47, 33, 0, time.UTC)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PROPFIND" {
			w.WriteHeader(207)
			fmt.Fprintf(w, `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/b.bin</D:href><D:propstat><D:prop><D:getcontentlength>1000</D:getcontentlength><D:getlastmodified>%s</D:getlastmodified><D:resourcetype/></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response></D:multistatus>`, mod.Format(http.TimeFormat))
			return
		}
		_, _ = w.Write(payload) // ignores Range: always 200 with the whole body
	}))
	defer origin.Close()
	conn, err := sources.NewWebDAV(sources.Config{Protocol: "webdav", Endpoint: origin.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	now := time.Now().UTC()
	if err := repo.UpsertWork(context.Background(), catalog.Work{ID: "w", CanonicalTitle: "w", CreatedAt: now, UpdatedAt: now,
		Editions: []catalog.Edition{{ID: "e", Title: "w", State: catalog.Identified, UpdatedAt: now,
			Files: []catalog.File{{ID: "f", SourceID: "src", Path: "b.bin", Format: "bin", MIMEType: "application/octet-stream", Size: 1000, SHA256: fmt.Sprintf("%x", sha256.Sum256(payload)), ModifiedAt: mod}}}}}); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	srv.connectors["src"] = conn
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	st, h, _ := doContent(t, client, http.MethodGet, ts.URL+"/v1/files/f/content", map[string]string{"Range": "bytes=10-19"})
	if st != http.StatusBadGateway {
		t.Fatalf("status %d, want 502", st)
	}
	if h.Get("Content-Range") != "" || h.Get("Content-Length") == "10" {
		t.Fatalf("error response leaks partial-content headers: Content-Range=%q Content-Length=%q", h.Get("Content-Range"), h.Get("Content-Length"))
	}
}
