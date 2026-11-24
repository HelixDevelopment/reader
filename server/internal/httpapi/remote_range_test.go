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
	"io"
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

// remoteErrOrigin is a WebDAV origin whose PROPFIND and GET answers are
// scriptable, to drive every error path of content().
type remoteErrOrigin struct {
	propfind  int  // PROPFIND status (207 = valid answer for the 1000-byte file)
	get       int  // GET status; 0 = serve normally honouring Range
	ignoreRng bool // answer GET with 200 + whole body even for a Range request
}

func remoteErrServer(t *testing.T, o *remoteErrOrigin, payload []byte, mod time.Time) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PROPFIND" {
			if o.propfind != 207 {
				w.WriteHeader(o.propfind)
				return
			}
			w.WriteHeader(207)
			fmt.Fprintf(w, `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/b.bin</D:href><D:propstat><D:prop><D:getcontentlength>%d</D:getcontentlength><D:getlastmodified>%s</D:getlastmodified><D:resourcetype/></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response></D:multistatus>`, len(payload), mod.Format(http.TimeFormat))
			return
		}
		if o.get != 0 {
			w.WriteHeader(o.get)
			return
		}
		if o.ignoreRng {
			_, _ = w.Write(payload)
			return
		}
		http.ServeContent(w, r, "b.bin", mod, bytes.NewReader(payload))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func remoteRouteFor(t *testing.T, conn sources.Connector, payload []byte, mod time.Time) *httptest.Server {
	t.Helper()
	repo := catalog.NewMemory()
	now := time.Now().UTC()
	if err := repo.UpsertWork(context.Background(), catalog.Work{ID: "w", CanonicalTitle: "w", CreatedAt: now, UpdatedAt: now,
		Editions: []catalog.Edition{{ID: "e", Title: "w", State: catalog.Identified, UpdatedAt: now,
			Files: []catalog.File{{ID: "f", SourceID: "src", Path: "b.bin", Format: "bin", MIMEType: "application/octet-stream", Size: int64(len(payload)), SHA256: fmt.Sprintf("%x", sha256.Sum256(payload)), ModifiedAt: mod}}}}}); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	srv.connectors["src"] = conn
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return ts
}

// seekFailConnector reports a current file through OpenRevalidated but its
// reader fails on Seek (the local-source range path).
type seekFailConnector struct {
	sources.Connector
	size int64
	mod  time.Time
}

type seekFailReader struct{}

func (seekFailReader) Read([]byte) (int, error)       { return 0, fmt.Errorf("read failed") }
func (seekFailReader) Seek(int64, int) (int64, error) { return 0, fmt.Errorf("seek failed") }
func (seekFailReader) Close() error                   { return nil }

func (c seekFailConnector) OpenRevalidated(context.Context, string) (io.ReadSeekCloser, int64, time.Time, error) {
	return seekFailReader{}, c.size, c.mod, nil
}

// Class: no response produced by an error path of content() may carry headers
// prepared for the success/partial response (a stale Content-Length truncates
// the JSON error body; Content-Range/ETag/Accept-Ranges/attachment name label
// bytes that were never sent). doContent fails the test on a short body.
func TestContentErrorPathsCarryNoSuccessHeaders(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789"), 100)
	mod := time.Date(2026, 11, 23, 21, 47, 33, 0, time.UTC)
	cases := []struct {
		name   string
		o      remoteErrOrigin
		method string
		rng    string
		want   int
	}{
		{"full GET, origin GET 500 after PROPFIND", remoteErrOrigin{propfind: 207, get: 500}, "GET", "", 502},
		{"full GET, origin GET 404 after PROPFIND", remoteErrOrigin{propfind: 207, get: 404}, "GET", "", 404},
		{"range GET, origin GET 500", remoteErrOrigin{propfind: 207, get: 500}, "GET", "bytes=10-19", 502},
		{"range GET, origin ignores Range at offset", remoteErrOrigin{propfind: 207, ignoreRng: true}, "GET", "bytes=10-19", 502},
		{"full GET, PROPFIND 500", remoteErrOrigin{propfind: 500}, "GET", "", 502},
		{"range GET, PROPFIND 500", remoteErrOrigin{propfind: 500}, "GET", "bytes=10-19", 502},
		{"range GET, PROPFIND 404", remoteErrOrigin{propfind: 404}, "GET", "bytes=10-19", 404},
		{"HEAD, PROPFIND 500", remoteErrOrigin{propfind: 500}, "HEAD", "", 502},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			origin := remoteErrServer(t, &c.o, payload, mod)
			conn, err := sources.NewWebDAV(sources.Config{Protocol: "webdav", Endpoint: origin.URL + "/"})
			if err != nil {
				t.Fatal(err)
			}
			ts := remoteRouteFor(t, conn, payload, mod)
			client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
			h := map[string]string{}
			if c.rng != "" {
				h["Range"] = c.rng
			}
			st, hdr, b := doContent(t, client, c.method, ts.URL+"/v1/files/f/content", h)
			assertErrorHeaders(t, st, c.want, hdr, b, c.method)
		})
	}
	t.Run("local-style source: Seek fails on a range request", func(t *testing.T) {
		origin := remoteErrServer(t, &remoteErrOrigin{propfind: 207}, payload, mod)
		inner, err := sources.NewWebDAV(sources.Config{Protocol: "webdav", Endpoint: origin.URL + "/"})
		if err != nil {
			t.Fatal(err)
		}
		ts := remoteRouteFor(t, seekFailConnector{Connector: inner, size: int64(len(payload)), mod: mod}, payload, mod)
		client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
		st, hdr, b := doContent(t, client, "GET", ts.URL+"/v1/files/f/content", map[string]string{"Range": "bytes=10-19"})
		assertErrorHeaders(t, st, 502, hdr, b, "GET")
	})
}

func assertErrorHeaders(t *testing.T, got, want int, hdr http.Header, body []byte, method string) {
	t.Helper()
	if got != want {
		t.Fatalf("status %d, want %d", got, want)
	}
	for _, k := range []string{"Content-Range", "ETag", "Accept-Ranges", "Content-Disposition"} {
		if hdr.Get(k) != "" {
			t.Fatalf("error response carries %s=%q", k, hdr.Get(k))
		}
	}
	if cl := hdr.Get("Content-Length"); method != "HEAD" && cl != "" && cl != fmt.Sprint(len(body)) {
		t.Fatalf("Content-Length %q does not match the %d-byte error body", cl, len(body))
	}
}

// RFC 9110 13.1.5: If-Range uses the strong comparison; a weak validator never
// matches, so the route answers a full 200 and not a 206 window.
func TestIfRangeWeakValidatorGetsFull200(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789"), 100)
	mod := time.Date(2026, 11, 23, 21, 47, 33, 0, time.UTC)
	origin := remoteErrServer(t, &remoteErrOrigin{propfind: 207}, payload, mod)
	conn, err := sources.NewWebDAV(sources.Config{Protocol: "webdav", Endpoint: origin.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	ts := remoteRouteFor(t, conn, payload, mod)
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	_, h, _ := doContent(t, client, "GET", ts.URL+"/v1/files/f/content", nil)
	etag := h.Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on a current file")
	}
	st, _, b := doContent(t, client, "GET", ts.URL+"/v1/files/f/content", map[string]string{"Range": "bytes=10-19", "If-Range": "W/" + etag})
	if st != 200 || len(b) != len(payload) {
		t.Fatalf("weak If-Range: status %d len %d, want 200 full", st, len(b))
	}
	st, _, b = doContent(t, client, "GET", ts.URL+"/v1/files/f/content", map[string]string{"Range": "bytes=10-19", "If-Range": etag})
	if st != 206 || len(b) != 10 {
		t.Fatalf("strong If-Range: status %d len %d, want 206 window", st, len(b))
	}
}
