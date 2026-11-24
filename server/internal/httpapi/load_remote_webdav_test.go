package httpapi

// WebDAV load run (HRD-027 "WebDAV/rclone load runs").
//
// The tree has WebDAV support: server/internal/sources/webdav.go (NewWebDAV —
// PROPFIND discovery, GET Open), server/internal/sources/webdav_test.go, and
// source.go's DefaultRegistry ("webdav"/"webdavs" → NewWebDAV).
//
// No rclone/WebDAV target is pre-configured in this environment (`rclone
// listremotes` is empty), so this run stands up a real WebDAV target with
// `rclone serve webdav` backed by a `local` backend remote ("wfdav") whose root
// is the fixture directory, and puts a counting reverse proxy in front of it so
// origin requests are measured. Run style mirrors TestLiveLoadRemoteSource:
// client → route → sources.NewWebDAV → counting proxy → rclone serve webdav →
// local fixture files. If rclone is not installed the test skips with the gap
// recorded in the emitted evidence.
//
// Gated on HELIX_LOAD_RUN=1 like the rest of the load suite.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/sources"
)

const webdavSourceID = "load-remote-webdav"

// remoteLoadRounds mirrors TestLiveLoadRemoteSource's ladder shape: 12 rounds
// at level 1, 2 rounds at higher levels.
func remoteLoadRounds(clients int) int {
	need := 12 / clients
	if need < 2 {
		need = 2
	}
	return need
}

type webdavFile struct {
	id, name string
	size     int64
	sha256   string
	rangeReq string
	rangeLen int64
	rangeSHA string
}

type webdavRow struct {
	deliveryStats
	OriginRequests int64 `json:"origin_requests"`
	OriginBytes    int64 `json:"origin_bytes"`
}

func TestLiveLoadRemoteSourceWebDAV(t *testing.T) {
	if os.Getenv("HELIX_LOAD_RUN") != "1" {
		t.Skip("HELIX_LOAD_RUN is not 1; measurement harness opted out")
	}
	setRemoteLoadOutput(t, "evidence/campaign-2026-11-23/remote/remote_webdav_ladder.jsonl")
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone is not installed: no WebDAV target available in this environment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Fixture: random bytes, digests are what the route must deliver.
	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	repo := catalog.NewMemory()
	files := []webdavFile{}
	for index, size := range remoteSizes(t) {
		name := fmt.Sprintf("books/rv%d.bin", index)
		abs := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		body := make([]byte, size)
		if _, err := rand.Read(body); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		digest := hex.EncodeToString(sum[:])
		if err := os.WriteFile(abs, body, 0o644); err != nil {
			t.Fatal(err)
		}
		id := fmt.Sprintf("load-webdav-f%d", index)
		work := catalog.Work{
			ID: id + "-w", CanonicalTitle: "WebDAV load " + id, CreatedAt: now, UpdatedAt: now,
			Editions: []catalog.Edition{{
				ID: id + "-e", Title: "WebDAV load " + id, State: catalog.Identified, UpdatedAt: now,
				Files: []catalog.File{{
					ID: id, SourceID: webdavSourceID, Path: name, Format: "bin",
					MIMEType: "application/octet-stream", Size: size, SHA256: digest, ModifiedAt: now,
				}},
			}},
		}
		if err := repo.UpsertWork(ctx, work); err != nil {
			t.Fatal(err)
		}
		request, window, start := rangeFor(size)
		files = append(files, webdavFile{id: id, name: name, size: size, sha256: digest, rangeReq: request, rangeLen: window, rangeSHA: windowSHA256(t, abs, start, window)})
	}

	// rclone config: the `local` backend remote "wfdav" — its root is the
	// fixture directory because the serve process runs with Dir=root.
	cfgPath := filepath.Join(t.TempDir(), "rclone.conf")
	if err := os.WriteFile(cfgPath, []byte("[wfdav]\ntype = local\nnolink = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Start the real WebDAV target: `rclone serve webdav wfdav:`.
	addr := freeTCPAddr(t)
	serveCmd := exec.Command("rclone", "--config", cfgPath, "serve", "webdav", "wfdav:", "--addr", addr)
	serveCmd.Dir = root
	serveLog := &serveLogBuffer{}
	serveCmd.Stdout, serveCmd.Stderr = serveLog, serveLog
	if err := serveCmd.Start(); err != nil {
		t.Fatalf("start rclone serve webdav: %v", err)
	}
	defer func() {
		serveCmd.Process.Kill()
		serveCmd.Wait()
	}()
	waitForWebDAVPath(t, "http://"+addr, files[0].name, files[0].sha256, serveLog)

	// Counting reverse proxy in front of the WebDAV target.
	var originRequests, originBytes atomic.Int64
	target, err := url.Parse("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originRequests.Add(1)
		rec := &countingWriter{ResponseWriter: w}
		httputil.NewSingleHostReverseProxy(target).ServeHTTP(rec, r)
		originBytes.Add(int64(rec.written))
	}))
	defer proxy.Close()

	connector, err := sources.NewWebDAV(sources.Config{Protocol: "webdav", Endpoint: proxy.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	srv.connectors[webdavSourceID] = connector
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 256, MaxConnsPerHost: 0, DisableCompression: true}}

	// Documented semantics, per file: the route serves the whole object with no
	// validators (docs/api.md: non-revalidating sources never advertise ETag or
	// ranges, and no Content-Length from a stale catalog size).
	for _, f := range files {
		url := ts.URL + "/v1/files/" + f.id + "/content"
		status, header, body := doContent(t, client, http.MethodGet, url, nil)
		if status != http.StatusOK {
			t.Fatalf("webdav full GET %s: status %d", f.id, status)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(body)); got != f.sha256 {
			t.Fatalf("webdav full GET %s delivered sha256 %s, catalog recorded %s", f.id, got, f.sha256)
		}
		// HRD-036: WebDAV now revalidates live size/mtime (PROPFIND Depth 0), so
		// a matching file carries the strong ETag, byte ranges and Content-Length.
		if got := header.Get("ETag"); got == "" {
			t.Fatalf("webdav %s revalidated but advertised no strong validator", f.id)
		}
		if got := header.Get("Accept-Ranges"); got != "bytes" {
			t.Fatalf("webdav %s Accept-Ranges %q, want bytes", f.id, got)
		}
		if got := header.Get("Content-Length"); got != fmt.Sprint(f.size) {
			t.Fatalf("webdav %s Content-Length %q, want %d", f.id, got, f.size)
		}
	}

	// Ladder, same shape as TestLiveLoadRemoteSource.
	levels := deliveryLevels(t)
	timeout := time.Duration(deliveryEnvInt(t, "HELIX_DELIVERY_TIMEOUT_SECONDS", 60)) * time.Second
	rows := []webdavRow{}
	for _, f := range files {
		url := ts.URL + "/v1/files/" + f.id + "/content"
		jobs := []deliveryJob{
			{kind: "full", url: url, wantStatus: http.StatusOK, wantBytes: f.size, wantSHA: f.sha256},
			{kind: "range", url: url, headers: map[string]string{"Range": f.rangeReq}, wantStatus: http.StatusPartialContent, wantBytes: f.rangeLen, wantSHA: f.rangeSHA},
		}
		for _, clients := range levels {
			requestsBefore := originRequests.Load()
			bytesBefore := originBytes.Load()
			stats := runDeliveryLadder(ctx, client, jobs, clients, remoteLoadRounds(clients), timeout)
			stats.SizeBytes = f.size
			originReqs := originRequests.Load() - requestsBefore
			originBytesTransferred := originBytes.Load() - bytesBefore
			rows = append(rows, webdavRow{deliveryStats: stats, OriginRequests: originReqs, OriginBytes: originBytesTransferred})
			if stats.PeakInFlight < int64(clients) {
				t.Fatalf("webdav size %d level %d: only %d requests were open at once", f.size, clients, stats.PeakInFlight)
			}
			for class, count := range stats.Mismatches {
				t.Fatalf("webdav size %d level %d: %d %s responses failed their integrity needle (%v)", f.size, clients, count, class, stats.Mismatches)
			}
			if stats.Completed != stats.Requested {
				t.Fatalf("webdav size %d level %d: %d of %d did not complete (%v)", f.size, clients, stats.Requested-stats.Completed, stats.Requested, stats.Errors)
			}
			if stats.StatusByKind["full"]+stats.StatusByKind["range"] != stats.Requested {
				t.Fatalf("webdav size %d level %d: status by kind %v, want the two job kinds to total requested %d", f.size, clients, stats.StatusByKind, stats.Requested)
			}
			if originReqs != 2*int64(stats.Requested) {
				t.Fatalf("webdav size %d level %d: %d client requests caused %d origin requests; want exactly PROPFIND+GET per request", f.size, clients, stats.Requested, originReqs)
			}
			if originBytesTransferred < stats.Bytes {
				t.Fatalf("webdav size %d level %d: the route delivered %d bytes but the origin only produced %d", f.size, clients, stats.Bytes, originBytesTransferred)
			}
			t.Logf("WEBDAV LADDER size=%d clients=%d completed=%d req_per_sec=%.1f mib_per_sec=%.1f p50_ms=%.2f p95_ms=%.2f peak_inflight=%d client_bytes=%d origin_bytes=%d origin_requests=%d",
				f.size, clients, stats.Completed, stats.ReqPerSec, stats.BytesPerSec/(1<<20), stats.P50MS, stats.P95MS,
				stats.PeakInFlight, stats.Bytes, originBytesTransferred, originReqs)
		}
	}

	rcloneVersion := rcloneVersionLine(t)
	httpEmitLoad(t, map[string]any{
		"test":             "remote_webdav_ladder",
		"connector":        "webdav",
		"target":           "rclone serve webdav (rclone " + rcloneVersion + ", local backend remote \"wfdav\")",
		"measurement_path": "in_process_loopback",
		"note":             "route revalidates via PROPFIND then serves ETag/ranges (HRD-036); each client request costs one PROPFIND plus one GET at the origin; rclone target stood up by the run, not pre-existing in the environment",
		"origin_requests":  originRequests.Load(),
		"origin_bytes":     originBytes.Load(),
		"levels":           levels,
		"results":          rows,
	})
}

// freeTCPAddr returns an address that is free right now. Race-y in theory; the
// rclone serve process is started on it immediately and we poll readiness.
func freeTCPAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

// waitForWebDAVPath polls the real WebDAV target until GET name returns 200
// with the fixture digest, so readiness implies a working path, not just a
// process listening.
func waitForWebDAVPath(t *testing.T, base, name, wantSHA string, log *serveLogBuffer) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(base + "/" + name)
		if err != nil {
			// Not listening yet: retry, the process needs a moment.
			time.Sleep(100 * time.Millisecond)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			t.Fatalf("webdav readiness GET %s: read body: %v", name, readErr)
		}
		if resp.StatusCode == http.StatusOK {
			if got := fmt.Sprintf("%x", sha256.Sum256(body)); got == wantSHA {
				return
			}
			t.Fatalf("webdav readiness GET %s: digest mismatch", name)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("webdav target at %s never served %s: rclone serve webdav log:\n%s", base, name, log.String())
}

// serveLogBuffer accumulates the rclone serve process output so a failure can
// include it in the error message.
type serveLogBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *serveLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *serveLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func rcloneVersionLine(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("rclone", "version").Output()
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "rclone v") {
			return strings.TrimSpace(strings.TrimPrefix(line, "rclone "))
		}
	}
	return "unknown"
}
