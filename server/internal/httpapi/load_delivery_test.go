package httpapi

// Load harness for the two delivery surfaces the catalog ladder does not touch:
// byte serving (/v1/files/:id/content, full and Range) and KOReader progress
// sync (PUT/GET /syncs/progress). Both are gated the same way as the catalog
// ladder — HELIX_LOAD_RUN=1 plus a live PostgreSQL, and for KOSync a live
// Redis — and emit one JSON payload per phase so the evidence file, not this
// comment, is what a claim rests on.
//
// Both tests carry an in-process concurrency proof rather than a wall-time
// argument: the delivered bytes are small enough that a "concurrent" claim based
// on latency ratios would be noise, so each level gates on all of its clients
// genuinely being in flight at once, measured with an atomic counter on a start
// barrier.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/kosync"
	"github.com/helixreader/helix/server/internal/sources"
	"github.com/redis/go-redis/v9"
)

const (
	deliverySource   = "load-delivery"
	deliveryIDPrefix = "load-dlv"
)

func deliveryEnvInt(t *testing.T, name string, fallback int) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		t.Fatalf("%s=%q is not a positive integer", name, raw)
	}
	return value
}

func deliverySizes(t *testing.T) []int64 {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("HELIX_DELIVERY_SIZES"))
	if raw == "" {
		return []int64{262144, 4194304, 33554432}
	}
	out := []int64{}
	for _, part := range strings.Split(raw, ",") {
		value, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || value < 4096 {
			t.Fatalf("HELIX_DELIVERY_SIZES contains %q; sizes below 4096 bytes cannot exercise a partial range", part)
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		t.Fatalf("HELIX_DELIVERY_SIZES=%q parsed to nothing", raw)
	}
	return out
}

// deliveryLevels defaults to 1,8,32 and refuses a single-level ladder: with no
// second level the in-flight proof below has nothing to compare against.
func deliveryLevels(t *testing.T) []int {
	t.Helper()
	out := []int{1, 8, 32}
	if raw := strings.TrimSpace(os.Getenv("HELIX_LOAD_LEVELS")); raw != "" {
		out = nil
		for _, part := range strings.Split(raw, ",") {
			value, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || value < 1 {
				t.Fatalf("HELIX_LOAD_LEVELS contains %q", part)
			}
			out = append(out, value)
		}
	}
	if len(out) < 2 {
		t.Fatalf("the delivery ladder needs at least two levels to say anything about concurrency, got %v", out)
	}
	return out
}

// inflightGate releases workers together so the peak-in-flight measurement means
// "this many requests were open at once" rather than "this many goroutines
// existed at some point during the run".
type inflightGate struct {
	clients int
	start   chan struct{}
	live    atomic.Int64
	peak    atomic.Int64
}

func (g *inflightGate) enter() {
	current := g.live.Add(1)
	for {
		high := g.peak.Load()
		if current <= high || g.peak.CompareAndSwap(high, current) {
			return
		}
	}
}

func (g *inflightGate) exit() { g.live.Add(-1) }

// seededFile is one fixture object: what is on disk, what the catalog recorded,
// and the slice the range probe expects back.
type seededFile struct {
	id       string
	name     string
	size     int64
	sha256   string
	rangeReq string
	rangeLen int64
	rangeSHA string
}

// fillRandom writes size cryptographically-random bytes. Compressible content
// would let a truncated or re-encoded body hash-match by luck on a small prefix,
// and the integrity needles below rely on the hash being specific.
func fillRandom(t *testing.T, path string, size int64) string {
	t.Helper()
	handle, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	buffer := make([]byte, 1<<16)
	digest := sha256.New()
	written := int64(0)
	for written < size {
		chunk := int64(len(buffer))
		if size-written < chunk {
			chunk = size - written
		}
		if _, err := rand.Read(buffer[:chunk]); err != nil {
			t.Fatal(err)
		}
		if _, err := handle.Write(buffer[:chunk]); err != nil {
			t.Fatal(err)
		}
		digest.Write(buffer[:chunk])
		written += chunk
	}
	if err := handle.Sync(); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

// rangeFor picks a window that is neither the head nor the tail of the object,
// because a handler that ignores the Range header and serves from offset 0 still
// returns a 206 with a plausible-looking length if the window were the prefix.
func rangeFor(size int64) (request string, length int64, start int64) {
	window := int64(65536)
	if size/4 < window {
		window = size / 4
	}
	if window < 1 {
		window = 1
	}
	start = size / 3
	return fmt.Sprintf("bytes=%d-%d", start, start+window-1), window, start
}

// windowSHA256 hashes the bytes a correct range response must contain, reading
// them from disk so the expectation never comes from the same code path as the
// response being checked.
func windowSHA256(t *testing.T, path string, start, length int64) string {
	t.Helper()
	handle, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	if _, err := handle.Seek(start, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	if _, err := io.CopyN(digest, handle, length); err != nil {
		t.Fatalf("read window: %v", err)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

// seedDelivery writes filesPerSize objects of each size into root and records
// them in the durable catalog under IDs that the cleanup removes. The catalog
// row is what the handler looks up, and the connector's live stat is what it
// revalidates against, so both have to agree or the server will refuse to serve a
// range rather than the harness having a fixture.
func seedDelivery(ctx context.Context, t *testing.T, repo *catalog.Postgres, root string, sizes []int64, filesPerSize int) []seededFile {
	t.Helper()
	out := []seededFile{}
	// Registered before the first write so a t.Fatal partway through still
	// removes the rows the catalog has already been told about.
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		clearDelivery(cleanup, repo, out)
	})
	now := time.Now().UTC().Truncate(time.Microsecond)
	for index, size := range sizes {
		for slot := 0; slot < filesPerSize; slot++ {
			sequence := len(out)
			name := fmt.Sprintf("%s-%d-%d.bin", deliveryIDPrefix, index, slot)
			path := filepath.Join(root, name)
			digest := fillRandom(t, path, size)
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			request, length, start := rangeFor(size)
			if err := os.Chtimes(path, now, now); err != nil {
				t.Fatal(err)
			}
			_ = info
			id := fmt.Sprintf("%s-f%d", deliveryIDPrefix, sequence)
			work := catalog.Work{
				ID: id + "-w", CanonicalTitle: "Load delivery " + id, CreatedAt: now, UpdatedAt: now,
				Editions: []catalog.Edition{{
					ID: id + "-e", Title: "Load delivery " + id, State: catalog.Identified, UpdatedAt: now,
					Files: []catalog.File{{
						ID: id, SourceID: deliverySource, Path: name, Format: "bin",
						MIMEType: "application/octet-stream", Size: size, SHA256: digest,
						ModifiedAt: now,
					}},
				}},
			}
			if err := repo.UpsertWork(ctx, work); err != nil {
				t.Fatalf("seed %s: %v", id, err)
			}
			out = append(out, seededFile{
				id: id, name: name, size: size, sha256: digest,
				rangeReq: request, rangeLen: length, rangeSHA: windowSHA256(t, path, start, length),
			})
		}
	}
	return out
}

func clearDelivery(ctx context.Context, repo *catalog.Postgres, seeded []seededFile) {
	for _, file := range seeded {
		if err := repo.DeleteWork(ctx, file.id+"-w"); err != nil {
			return
		}
	}
}

func TestLiveLoadContentDelivery(t *testing.T) {
	dsn := httpLoadWanted(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	repo, err := catalog.NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	root := t.TempDir()
	sizes := deliverySizes(t)
	perSize := deliveryEnvInt(t, "HELIX_DELIVERY_FILES", 3)
	seeded := seedDelivery(ctx, t, repo, root, sizes, perSize)
	if len(seeded) != len(sizes)*perSize {
		t.Fatalf("seeded %d delivery objects, want %d", len(seeded), len(sizes)*perSize)
	}

	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	srv.connectors[deliverySource] = connector
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{Transport: &http.Transport{
		MaxIdleConnsPerHost: 256, MaxConnsPerHost: 0, DisableCompression: true,
	}}

	bySize := map[int64][]seededFile{}
	for _, file := range seeded {
		bySize[file.size] = append(bySize[file.size], file)
	}

	// --- correctness before throughput ---------------------------------------
	//
	// Every needle in the ladder below assumes a full body hashes to the catalog
	// record and a range body to its window. Establishing that once, serially, is
	// what lets a ladder failure be read as a concurrency failure rather than as a
	// broken fixture.
	for _, size := range sizes {
		files := bySize[size]
		for _, file := range files {
			url := ts.URL + "/v1/files/" + file.id + "/content"
			status, header, body := doContent(t, client, http.MethodGet, url, nil)
			if status != http.StatusOK {
				t.Fatalf("full GET %s: status %d", file.id, status)
			}
			if int64(len(body)) != file.size {
				t.Fatalf("full GET %s delivered %d bytes, catalog recorded %d", file.id, len(body), file.size)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(body)); got != file.sha256 {
				t.Fatalf("full GET %s sha256 %s does not match the catalog record %s", file.id, got, file.sha256)
			}
			if header.Get("Accept-Ranges") != "bytes" {
				t.Fatalf("full GET %s did not advertise Accept-Ranges, so ranges would be served as %d-byte whole objects: %q",
					file.id, file.size, header.Get("Accept-Ranges"))
			}
			etag := header.Get("ETag")
			if !strings.Contains(etag, file.sha256) {
				t.Fatalf("full GET %s ETag %q is not the recorded digest %s", file.id, etag, file.sha256)
			}

			status, header, body = doContent(t, client, http.MethodGet, url, map[string]string{"Range": file.rangeReq})
			if status != http.StatusPartialContent {
				t.Fatalf("range GET %s (%s) returned %d, not 206: the range path is not being taken, so every client would pay for %d bytes",
					file.id, file.rangeReq, status, file.size)
			}
			if int64(len(body)) != file.rangeLen {
				t.Fatalf("range GET %s (%s) returned %d bytes, want %d", file.id, file.rangeReq, len(body), file.rangeLen)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(body)); got != file.rangeSHA {
				t.Fatalf("range GET %s returned the right length but the wrong window (sha256 %s, want %s)", file.id, got, file.rangeSHA)
			}
			want := strings.Replace(file.rangeReq, "bytes=", "bytes ", 1) + "/" + strconv.FormatInt(file.size, 10)
			if got := header.Get("Content-Range"); !strings.HasPrefix(got, want[:len(want)-len(strconv.FormatInt(file.size, 10))]) || !strings.HasSuffix(got, want[len(want)-len(strconv.FormatInt(file.size, 10)):]) {
				t.Fatalf("range GET %s Content-Range %q does not describe %q", file.id, got, want)
			}

			// A multi-range request must degrade to a full 200 rather than an
			// invalid 416 or a half-implemented 206.
			status, _, body = doContent(t, client, http.MethodGet, url, map[string]string{"Range": "bytes=0-99,200-299"})
			if status != http.StatusOK || int64(len(body)) != file.size {
				t.Fatalf("multi-range GET %s returned %d/%d bytes, want 200/%d", file.id, status, len(body), file.size)
			}

			// Unsatisfiable range.
			if status, _, _ = doContent(t, client, http.MethodGet, url, map[string]string{"Range": fmt.Sprintf("bytes=%d-", file.size+1024)}); status != http.StatusRequestedRangeNotSatisfiable {
				t.Fatalf("out-of-range GET %s returned %d, want 416", file.id, status)
			}
		}
	}

	// --- throughput ladder -----------------------------------------------------
	//
	// One full and one range job per seeded object, and the ladder runs per size
	// class: mixing a 32 MiB object into the same sample as a 256 KiB one makes the
	// per-level numbers depend on which object a level happened to reach first,
	// which is how a 1-client baseline ends up measuring only the smallest file.
	roundsFor := func(clients int) int {
		base := deliveryEnvInt(t, "HELIX_DELIVERY_ROUNDS", 0)
		if base > 0 {
			return base
		}
		// Enough requests per level that a single slow object cannot decide it.
		need := 24 / clients
		if need < 3 {
			need = 3
		}
		return need
	}
	timeout := time.Duration(deliveryEnvInt(t, "HELIX_DELIVERY_TIMEOUT_SECONDS", 60)) * time.Second
	levels := deliveryLevels(t)
	ladder := []deliveryStats{}
	for _, size := range sizes {
		jobs := []deliveryJob{}
		for _, file := range seeded {
			if file.size != size {
				continue
			}
			url := ts.URL + "/v1/files/" + file.id + "/content"
			jobs = append(jobs,
				deliveryJob{kind: "full", url: url, wantStatus: http.StatusOK, wantBytes: file.size, wantSHA: file.sha256},
				// A window that is neither head nor tail: a handler that ignores
				// Range and serves from offset 0 still returns a plausible 206
				// length if the window were the prefix.
				deliveryJob{kind: "range", url: url, headers: map[string]string{"Range": file.rangeReq}, wantStatus: http.StatusPartialContent, wantBytes: file.rangeLen, wantSHA: file.rangeSHA},
				// The compressed-encoding probe rides the range job: a middleman
				// that gzips a 206 changes both length and digest.
				deliveryJob{kind: "range-gzip", url: url, headers: map[string]string{"Range": file.rangeReq, "Accept-Encoding": "gzip, br"}, wantStatus: http.StatusPartialContent, wantBytes: file.rangeLen, wantSHA: file.rangeSHA},
			)
		}
		for _, clients := range levels {
			rounds := roundsFor(clients)
			stats := runDeliveryLadder(ctx, client, jobs, clients, rounds, timeout)
			stats.SizeBytes = size
			ladder = append(ladder, stats)
			// The concurrency proof. Wall time alone cannot separate "32 clients"
			// from "32 goroutines taking turns", and a harness that quietly
			// serialises would report a beautiful scaling curve.
			if stats.PeakInFlight < int64(clients) {
				t.Fatalf("size %d, level %d: only %d requests were open at the same instant, so the %d-client number is a serial measurement",
					size, clients, stats.PeakInFlight, clients)
			}
			if stats.Requested != clients*rounds {
				t.Fatalf("size %d, level %d: issued %d requests, want %d", size, clients, stats.Requested, clients*rounds)
			}
			for class, count := range stats.Mismatches {
				t.Fatalf("size %d, level %d: %d %s responses failed their integrity needle (%v)", size, clients, count, class, stats.Mismatches)
			}
			if stats.Completed != stats.Requested {
				t.Fatalf("size %d, level %d: %d of %d did not complete (%v)", size, clients, stats.Requested-stats.Completed, stats.Requested, stats.Errors)
			}
			t.Logf("DELIVERY LADDER size=%d clients=%d rounds=%d completed=%d wall_ms=%.1f req_per_sec=%.1f gbit_per_sec=%.3f mib_per_sec=%.1f p50_ms=%.2f p95_ms=%.2f p99_ms=%.2f max_ms=%.2f peak_inflight=%d bytes=%d",
				size, clients, rounds, stats.Completed, stats.WallMS, stats.ReqPerSec, stats.GbitPerSec,
				stats.BytesPerSec/(1<<20), stats.P50MS, stats.P95MS, stats.P99MS, stats.MaxMS, stats.PeakInFlight, stats.Bytes)
		}
	}
	// Per-size scaling: the only comparison that means anything.
	for _, size := range sizes {
		var first, last *deliveryStats
		for i := range ladder {
			if ladder[i].SizeBytes != size {
				continue
			}
			if first == nil {
				first = &ladder[i]
			}
			last = &ladder[i]
		}
		if first != nil && last != nil && first != last && first.BytesPerSec > 0 {
			t.Logf("DELIVERY SCALING size=%d: %.3f -> %.3f Gbit/s (%.2fx) from %d to %d clients",
				size, first.GbitPerSec, last.GbitPerSec, last.BytesPerSec/first.BytesPerSec, first.Clients, last.Clients)
		}
	}
	httpEmitLoad(t, map[string]any{
		"test":              "content_delivery_ladder",
		"measurement_path":  os.Getenv("HELIX_MEASUREMENT_PATH"),
		"source":            deliverySource,
		"fixture_objects":   len(seeded),
		"fixture_sizes":     sizes,
		"request_timeout_s": int(timeout.Seconds()),
		"levels":            levels,
		"results":           ladder,
	})
}

type deliveryJob struct {
	kind       string
	url        string
	headers    map[string]string
	wantStatus int
	wantBytes  int64
	wantSHA    string
}

type deliveryStats struct {
	Clients         int              `json:"clients"`
	RoundsPerClient int              `json:"rounds_per_client"`
	Requested       int              `json:"requested"`
	Completed       int              `json:"completed"`
	Errors          map[string]int   `json:"errors_by_class"`
	Mismatches      map[string]int   `json:"integrity_mismatches"`
	BytesByKind     map[string]int64 `json:"bytes_by_kind"`
	StatusByKind    map[string]int   `json:"status_by_kind"`
	Bytes           int64            `json:"total_bytes"`
	WallMS          float64          `json:"wall_ms"`
	ReqPerSec       float64          `json:"req_per_sec"`
	BytesPerSec     float64          `json:"bytes_per_sec"`
	GbitPerSec      float64          `json:"gbit_per_sec"`
	P50MS           float64          `json:"p50_ms"`
	P95MS           float64          `json:"p95_ms"`
	P99MS           float64          `json:"p99_ms"`
	MaxMS           float64          `json:"max_ms"`
	MeanMS          float64          `json:"mean_ms"`
	PeakInFlight    int64            `json:"peak_inflight"`
	SizeBytes       int64            `json:"object_size_bytes"`
}

func runDeliveryLadder(ctx context.Context, client *http.Client, jobs []deliveryJob, clients, rounds int, timeout time.Duration) deliveryStats {
	gate := &inflightGate{clients: clients, start: make(chan struct{})}
	latencies := make([]float64, 0, clients*rounds)
	perClient := make([][]float64, clients)
	errCount := map[string]int{}
	mismatch := map[string]int{}
	bytesByKind := map[string]int64{}
	statusByKind := map[string]int{}
	var mu sync.Mutex
	totalBytes := int64(0)
	var wg sync.WaitGroup
	wallStart := time.Now()
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			mine := []float64{}
			for r := 0; r < rounds; r++ {
				job := jobs[(c*rounds+r)%len(jobs)]
				<-gate.start
				gate.enter()
				rctx, cancel := context.WithTimeout(ctx, timeout)
				req, err := http.NewRequestWithContext(rctx, http.MethodGet, job.url, nil)
				if err != nil {
					gate.exit()
					cancel()
					mu.Lock()
					errCount["request_build"]++
					mu.Unlock()
					continue
				}
				for k, v := range job.headers {
					req.Header.Set(k, v)
				}
				t0 := time.Now()
				resp, err := client.Do(req)
				if err != nil {
					gate.exit()
					cancel()
					mu.Lock()
					if errors.Is(err, context.DeadlineExceeded) {
						errCount["deadline"]++
					} else {
						errCount["transport"]++
					}
					mu.Unlock()
					continue
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				elapsed := float64(time.Since(t0).Microseconds()) / 1000
				gate.exit()
				cancel()
				mu.Lock()
				if err != nil {
					errCount["body_read"]++
				}
				statusByKind[job.kind]++
				bytesByKind[job.kind] += int64(len(body))
				if resp.StatusCode != job.wantStatus {
					mismatch[job.kind+"_status"]++
				}
				if int64(len(body)) != job.wantBytes {
					mismatch[job.kind+"_length"]++
				} else if got := fmt.Sprintf("%x", sha256.Sum256(body)); got != job.wantSHA {
					mismatch[job.kind+"_digest"]++
				}
				if enc := resp.Header.Get("Content-Encoding"); enc != "" {
					mismatch[job.kind+"_recompressed"]++
				}
				mu.Unlock()
				mine = append(mine, elapsed)
				atomic.AddInt64(&totalBytes, int64(len(body)))
			}
			perClient[c] = mine
		}(c)
	}
	close(gate.start)
	wg.Wait()
	wall := float64(time.Since(wallStart).Microseconds()) / 1000
	for _, per := range perClient {
		latencies = append(latencies, per...)
	}
	errTotal := 0
	for _, n := range errCount {
		errTotal += n
	}
	stats := deliveryStats{
		Clients: clients, RoundsPerClient: rounds, Requested: clients * rounds,
		Completed: len(latencies) - errTotal, Errors: errCount, Mismatches: mismatch,
		BytesByKind: bytesByKind, StatusByKind: statusByKind, Bytes: totalBytes, WallMS: wall,
		PeakInFlight: gate.peak.Load(),
	}
	if len(latencies) > 0 && wall > 0 {
		sum := 0.0
		for _, v := range latencies {
			sum += v
		}
		stats.ReqPerSec = float64(stats.Completed) / (wall / 1000)
		stats.BytesPerSec = float64(totalBytes) / (wall / 1000)
		stats.GbitPerSec = stats.BytesPerSec * 8 / 1e9
		stats.MeanMS = sum / float64(len(latencies))
		stats.P50MS = httpPercentile(latencies, 50)
		stats.P95MS = httpPercentile(latencies, 95)
		stats.P99MS = httpPercentile(latencies, 99)
		stats.MaxMS = httpPercentile(latencies, 100)
	}
	return stats
}

// --- KOSync progress sync ----------------------------------------------------
//
// The progress store is the other surface with no harness: PUT/GET
// /syncs/progress behind bcrypt-authenticated headers, backed by Redis
// optimistic transactions. The ladder exists because two properties of that
// path are only visible under concurrency: WATCH/MULTI contention either
// converges on the monotonic winner or it silently rewinds a reader, and the
// per-request password hash is a CPU ceiling no request-count metric reveals.

func kosyncUserKey(username string) string {
	return "helix:v1:kosync:user:" + base64.RawURLEncoding.EncodeToString([]byte(username))
}

func kosyncProgressKey(username, document string) string {
	return "helix:v1:kosync:progress:" + base64.RawURLEncoding.EncodeToString([]byte(username)) +
		":" + base64.RawURLEncoding.EncodeToString([]byte(document))
}

type kosyncStats struct {
	Operation    string         `json:"operation"`
	Clients      int            `json:"clients"`
	RoundsPerCli int            `json:"rounds_per_client"`
	Requested    int            `json:"requested"`
	Completed    int            `json:"completed"`
	Errors       map[string]int `json:"errors_by_class"`
	Rejected     map[string]int `json:"rejected_responses"`
	WallMS       float64        `json:"wall_ms"`
	ReqPerSec    float64        `json:"req_per_sec"`
	P50MS        float64        `json:"p50_ms"`
	P95MS        float64        `json:"p95_ms"`
	P99MS        float64        `json:"p99_ms"`
	MaxMS        float64        `json:"max_ms"`
	MeanMS       float64        `json:"mean_ms"`
	PeakInFlight int64          `json:"peak_inflight"`
}

func runKOSyncLadder(ctx context.Context, client *http.Client, op string, clients, rounds int, timeout time.Duration, base, username, password string, documents []string) kosyncStats {
	gate := &inflightGate{clients: clients, start: make(chan struct{})}
	perClient := make([][]float64, clients)
	errCount := map[string]int{}
	rejected := map[string]int{}
	var mu sync.Mutex
	completed := 0
	var wg sync.WaitGroup
	wallStart := time.Now()
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			mine := []float64{}
			for r := 0; r < rounds; r++ {
				doc := documents[(c*rounds+r)%len(documents)]
				var req *http.Request
				var err error
				if op == "put" {
					body, marshalErr := json.Marshal(kosync.Progress{
						Document: doc, Progress: strconv.Itoa(r),
						Percentage: float64((c%97)+1) / 100, Device: "load-" + strconv.Itoa(c),
					})
					if marshalErr != nil {
						errCount["marshal"]++
						continue
					}
					req, err = http.NewRequestWithContext(ctx, http.MethodPut, base+"/syncs/progress", bytes.NewReader(body))
				} else {
					req, err = http.NewRequestWithContext(ctx, http.MethodGet, base+"/syncs/progress/"+url.PathEscape(doc), nil)
				}
				if err != nil {
					mu.Lock()
					errCount["request_build"]++
					mu.Unlock()
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("x-auth-user", username)
				req.Header.Set("x-auth-key", password)
				<-gate.start
				gate.enter()
				rctx, cancel := context.WithTimeout(ctx, timeout)
				req = req.WithContext(rctx)
				t0 := time.Now()
				resp, err := client.Do(req)
				if err != nil {
					gate.exit()
					cancel()
					mu.Lock()
					if errors.Is(err, context.DeadlineExceeded) {
						errCount["deadline"]++
					} else {
						errCount["transport"]++
					}
					mu.Unlock()
					continue
				}
				payload, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
				resp.Body.Close()
				elapsed := float64(time.Since(t0).Microseconds()) / 1000
				gate.exit()
				cancel()
				mu.Lock()
				switch {
				case readErr != nil:
					errCount["body_read"]++
				case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
					rejected["http_"+strconv.Itoa(resp.StatusCode)]++
				case resp.StatusCode != http.StatusOK:
					rejected["http_"+strconv.Itoa(resp.StatusCode)]++
				case op == "get":
					var decoded kosync.Progress
					if err := json.Unmarshal(payload, &decoded); err != nil {
						rejected["undecodable_get"]++
					} else if decoded.Document != doc {
						rejected["wrong_document"]++
					} else {
						completed++
					}
				default:
					var decoded struct {
						Document  string `json:"document"`
						Timestamp int64  `json:"timestamp"`
					}
					if err := json.Unmarshal(payload, &decoded); err != nil {
						rejected["undecodable_put"]++
					} else if decoded.Document != doc || decoded.Timestamp <= 0 {
						rejected["put_ack_mismatch"]++
					} else {
						completed++
					}
				}
				mu.Unlock()
				mine = append(mine, elapsed)
			}
			perClient[c] = mine
		}(c)
	}
	close(gate.start)
	wg.Wait()
	wall := float64(time.Since(wallStart).Microseconds()) / 1000
	all := []float64{}
	for _, per := range perClient {
		all = append(all, per...)
	}
	stats := kosyncStats{Operation: op, Clients: clients, RoundsPerCli: rounds, Requested: clients * rounds,
		Completed: completed, Errors: errCount, Rejected: rejected, WallMS: wall, PeakInFlight: gate.peak.Load()}
	if len(all) > 0 && wall > 0 {
		sum := 0.0
		for _, v := range all {
			sum += v
		}
		stats.ReqPerSec = float64(completed) / (wall / 1000)
		stats.MeanMS = sum / float64(len(all))
		stats.P50MS = httpPercentile(all, 50)
		stats.P95MS = httpPercentile(all, 95)
		stats.P99MS = httpPercentile(all, 99)
		stats.MaxMS = httpPercentile(all, 100)
	}
	return stats
}

func TestLiveLoadKOSync(t *testing.T) {
	dsn := httpLoadWanted(t)
	redisAddr := strings.TrimSpace(os.Getenv("HELIX_TEST_REDIS_ADDR"))
	if redisAddr == "" {
		t.Skip("HELIX_TEST_REDIS_ADDR is not configured; measuring the in-memory fallback store would say nothing about the Redis path that ships")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	repo, err := catalog.NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("HELIX_TEST_REDIS_ADDR=%s is opted into but not reachable: %v", redisAddr, err)
	}
	defer rdb.Close()

	username := fmt.Sprintf("loadsync%d", os.Getpid())
	password := "load-harness-password"
	documents := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		documents = append(documents, fmt.Sprintf("%s-load-doc-%d.bin", username, i))
	}
	contentionDoc := username + "-contention.bin"
	keys := []string{kosyncUserKey(username), kosyncProgressKey(username, contentionDoc)}
	for _, doc := range documents {
		keys = append(keys, kosyncProgressKey(username, doc))
	}
	// Delete before and after: registration is SET NX, so a leftover account
	// from an interrupted run would authenticate against the wrong verifier.
	rdb.Del(ctx, keys...)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rdb.Del(cleanup, keys...)
	})

	srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	srv.KOSync = kosync.NewService(kosync.NewRedis(rdb), true)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 256, MaxConnsPerHost: 0}}

	post := func(op, path, body string, auth bool) (int, []byte) {
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, op, ts.URL+path, reader)
		if err != nil {
			t.Fatal(err)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if auth {
			req.Header.Set("x-auth-user", username)
			req.Header.Set("x-auth-key", password)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return resp.StatusCode, payload
	}

	// --- account lifecycle over HTTP ------------------------------------------
	if status, body := post(http.MethodPost, "/users/create", `{"username":"`+username+`","password":"`+password+`"}`, false); status != http.StatusCreated {
		t.Fatalf("registration returned %d (%s), want 201; registration must be enabled for the sync surface to work at all", status, body)
	}
	if status, body := post(http.MethodPost, "/users/create", `{"username":"`+username+`","password":"another-password"}`, false); status != http.StatusPaymentRequired {
		t.Fatalf("duplicate registration returned %d, want 402 (KOReader reference): concurrent create must not be able to replace a verifier", status)
	} else {
		var errBody struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &errBody); err != nil || errBody.Message != "Username is already registered." {
			t.Fatalf("duplicate registration body %q, want message %q (decode err: %v)", body, "Username is already registered.", err)
		}
	}
	// Independent proof that the route writes to Redis rather than the
	// in-memory store: recompute the key the way the store does and read it
	// through a separate client.
	stored, err := rdb.Get(ctx, kosyncUserKey(username)).Result()
	if err != nil {
		t.Fatalf("account created over HTTP is absent from Redis (%s): %v — the harness is not measuring the shipping store", kosyncUserKey(username), err)
	}
	if !strings.HasPrefix(stored, "$2") {
		t.Fatalf("Redis holds something in the verifier slot that is not a bcrypt digest")
	}
	for _, probe := range []struct {
		name, user, pass string
		want             int
	}{
		{"correct", username, password, http.StatusOK},
		{"wrong-key", username, "wrong-load-password", http.StatusUnauthorized},
		{"unknown-user", "load-nobody-9999", password, http.StatusUnauthorized},
	} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/users/auth", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("x-auth-user", probe.user)
		req.Header.Set("x-auth-key", probe.pass)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != probe.want {
			t.Fatalf("auth probe %q returned %d, want %d", probe.name, resp.StatusCode, probe.want)
		}
	}

	// --- monotonic rule at the HTTP surface -----------------------------------
	if status := putProgress(ctx, t, client, ts.URL, username, password, kosync.Progress{
		Document: contentionDoc, Progress: "1/3", Percentage: 0.75, Device: "load-a"}); status != http.StatusOK {
		t.Fatalf("first PUT returned %d, want 200", status)
	}
	if status := putProgress(ctx, t, client, ts.URL, username, password, kosync.Progress{
		Document: contentionDoc, Progress: "1/9", Percentage: 0.25, Device: "load-b"}); status != http.StatusOK {
		t.Fatalf("rewind attempt returned %d, want 200 with the winner echoed back", status)
	}
	if got := getPercentage(ctx, t, client, ts.URL, username, password, contentionDoc); math.Abs(got-0.75) > 1e-9 {
		t.Fatalf("a lower percentage rewound stored progress to %v, want 0.75", got)
	}
	if status, body := post(http.MethodGet, "/syncs/progress/"+url.PathEscape(username+"-never-written.bin"), "", true); status != http.StatusOK || strings.TrimSpace(string(body)) != "{}" {
		t.Fatalf("unread document returned %d (%s), want 200 with body {} (KOReader reference)", status, body)
	}
	// --- one document, every writer at the same instant ------------------------
	//
	// Percentages are distinct so the monotonic rule has a single correct answer:
	// the stored value afterwards is the maximum any writer offered. A store that
	// accepted the last arriving write instead of the largest would pass a
	// sequential test and fail this one.
	contenders := deliveryEnvInt(t, "HELIX_KOSYNC_CONTENDERS", 32)
	if contenders < 2 {
		t.Fatalf("HELIX_KOSYNC_CONTENDERS=%d cannot contend", contenders)
	}
	best := 0.0
	rounds := deliveryEnvInt(t, "HELIX_DELIVERY_ROUNDS", 2)
	timeout := time.Duration(deliveryEnvInt(t, "HELIX_DELIVERY_TIMEOUT_SECONDS", 60)) * time.Second
	gate := &inflightGate{clients: contenders, start: make(chan struct{})}
	statuses := make([]int, contenders)
	var bodyWG sync.WaitGroup
	for i := 0; i < contenders; i++ {
		percentage := float64(i+1) / float64(contenders+1)
		if percentage > best {
			best = percentage
		}
		bodyWG.Add(1)
		go func(i int, percentage float64) {
			defer bodyWG.Done()
			<-gate.start
			gate.enter()
			defer gate.exit()
			status := putProgress(ctx, t, client, ts.URL, username, password, kosync.Progress{
				Document: contentionDoc, Progress: strconv.Itoa(i), Percentage: percentage,
				Device: fmt.Sprintf("load-contender-%d", i),
			})
			statuses[i] = status
		}(i, percentage)
	}
	close(gate.start)
	bodyWG.Wait()
	for i, status := range statuses {
		if status != http.StatusOK {
			t.Fatalf("contender %d got %d under same-document contention; a 500 here means the optimistic transaction exhausted its retry budget", i, status)
		}
	}
	if gate.peak.Load() < int64(contenders) {
		t.Fatalf("contention phase had %d writers open at once, not %d, so the result says nothing about transaction retries", gate.peak.Load(), contenders)
	}
	if got := getPercentage(ctx, t, client, ts.URL, username, password, contentionDoc); math.Abs(got-best) > 1e-9 {
		t.Fatalf("after %d concurrent writers the stored percentage is %v, want the maximum offered (%v): progress rewound or a write was lost", contenders, got, best)
	}
	if err := rdb.Get(ctx, kosyncProgressKey(username, contentionDoc)).Err(); err != nil {
		t.Fatalf("contended progress is not in Redis: %v", err)
	}

	// The acknowledged state must be the persisted state. PutProgress re-reads the
	// winner precisely so that a rejected stale write cannot ack the value it tried
	// to send; if it ever returns the caller's copy instead, a client is told its
	// position was stored when the store kept someone else's.
	ackDoc := username + "-ack-check.bin"
	keys = append(keys, kosyncProgressKey(username, ackDoc))
	if _, err := srv.KOSync.PutProgress(ctx, username, kosync.Progress{
		Document: ackDoc, Progress: "9/10", Percentage: 0.9, Device: "ack-high"}); err != nil {
		t.Fatal(err)
	}
	acked, err := srv.KOSync.PutProgress(ctx, username, kosync.Progress{
		Document: ackDoc, Progress: "1/10", Percentage: 0.1, Device: "ack-low"})
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := srv.KOSync.GetProgress(ctx, username, ackDoc)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(acked.Percentage-persisted.Percentage) > 1e-9 || acked.Device != persisted.Device {
		t.Fatalf("the acknowledged progress is not the persisted state: ack says (%.4f,%s), the store says (%.4f,%s)",
			acked.Percentage, acked.Device, persisted.Percentage, persisted.Device)
	}
	if math.Abs(persisted.Percentage-0.9) > 1e-9 {
		t.Fatalf("the winner of a rejected write is not the stored value: %v", persisted.Percentage)
	}

	// --- store-level contention, without the hash in the way ------------------
	//
	// The HTTP phase above cannot drive real transaction conflicts: every request
	// spends tens of milliseconds in bcrypt before it reaches Redis, so writers
	// arrive spread out and rarely collide. This calls the shipping store directly
	// from a barrier, which is the only way to see what the optimistic transaction
	// does when a hundred writers really do hit the same key at once.
	storeContenders := deliveryEnvInt(t, "HELIX_KOSYNC_STORE_CONTENDERS", 128)
	storeDoc := username + "-store-contention.bin"
	keys = append(keys, kosyncProgressKey(username, storeDoc))
	peakPct := 0.0
	storeGate := &inflightGate{clients: storeContenders, start: make(chan struct{})}
	storeErrs := make([]error, storeContenders)
	var storeWG sync.WaitGroup
	for i := 0; i < storeContenders; i++ {
		percentage := float64(i+1) / float64(storeContenders+1)
		if percentage > peakPct {
			peakPct = percentage
		}
		storeWG.Add(1)
		go func(i int, percentage float64) {
			defer storeWG.Done()
			<-storeGate.start
			storeGate.enter()
			defer storeGate.exit()
			_, err := srv.KOSync.PutProgress(ctx, username, kosync.Progress{
				Document: storeDoc, Progress: strconv.Itoa(i), Percentage: percentage,
				Device: fmt.Sprintf("store-contender-%d", i),
			})
			storeErrs[i] = err
		}(i, percentage)
	}
	close(storeGate.start)
	storeWG.Wait()
	storeFailures := 0
	storeErrClasses := map[string]int{}
	for _, err := range storeErrs {
		if err != nil {
			storeFailures++
			storeErrClasses[err.Error()]++
		}
	}
	// A sub-millisecond local write cannot keep 128 goroutines resident on a
	// 16-thread box, so the bar here is overlap, not simultaneity. The HTTP
	// ladder keeps the strict bar because bcrypt holds every request open long
	// enough for "all at once" to mean what it says.
	if int(storeGate.peak.Load()*4) < storeContenders*3 {
		t.Fatalf("store contention had only %d of %d writers open at once, which is not a contention measurement",
			storeGate.peak.Load(), storeContenders)
	}
	if storeFailures > 0 {
		t.Fatalf("%d of %d concurrent store writes failed outright (%v): a retried transaction should succeed, not surface an error",
			storeFailures, storeContenders, storeErrClasses)
	}
	storedProgress, err := srv.KOSync.GetProgress(ctx, username, storeDoc)
	if err != nil {
		t.Fatalf("reading contended progress: %v", err)
	}
	if math.Abs(storedProgress.Percentage-peakPct) > 1e-9 {
		t.Fatalf("after %d concurrent store writes the maximum offered percentage is %v, want %v: the optimistic transaction lost a write",
			storeContenders, storedProgress.Percentage, peakPct)
	}
	t.Logf("KOSYNC STORE CONTENTION writers=%d peak_inflight=%d stored=%.6f max_offered=%.6f",
		storeContenders, storeGate.peak.Load(), storedProgress.Percentage, peakPct)

	// --- the CPU floor under the auth header ----------------------------------
	//
	// Every authenticated request hashes the supplied key with bcrypt. The ladder
	// cannot tell a saturated pool from a saturated hash, so time the hash on its
	// own and report it next to the plateau.
	hashSamples := []float64{}
	for i := 0; i < 8; i++ {
		t0 := time.Now()
		if err := srv.KOSync.Authenticate(ctx, username, password); err != nil {
			t.Fatalf("authenticate: %v", err)
		}
		hashSamples = append(hashSamples, float64(time.Since(t0).Microseconds())/1000)
	}
	sort.Float64s(hashSamples)
	bcryptMS := hashSamples[len(hashSamples)/2]

	ladder := []kosyncStats{}
	for _, op := range []string{"put", "get"} {
		for _, clients := range deliveryLevels(t) {
			stats := runKOSyncLadder(ctx, client, op, clients, rounds, timeout, ts.URL, username, password, documents)
			ladder = append(ladder, stats)
			if stats.PeakInFlight < int64(clients) {
				t.Fatalf("%s level %d: only %d requests were open at once, so the number is serial", op, clients, stats.PeakInFlight)
			}
			if stats.Requested != clients*rounds {
				t.Fatalf("%s level %d: issued %d requests, want %d", op, clients, stats.Requested, clients*rounds)
			}
			for class, count := range stats.Rejected {
				t.Fatalf("%s level %d: %d requests were rejected as %s", op, clients, count, class)
			}
			if stats.Completed != stats.Requested {
				t.Fatalf("%s level %d: %d of %d did not complete (%v)", op, clients, stats.Requested-stats.Completed, stats.Requested, stats.Errors)
			}
			t.Logf("KOSYNC LADDER %s clients=%d completed=%d req_per_sec=%.2f p50_ms=%.2f p95_ms=%.2f p99_ms=%.2f max_ms=%.2f peak_inflight=%d",
				stats.Operation, stats.Clients, stats.Completed, stats.ReqPerSec, stats.P50MS, stats.P95MS, stats.P99MS, stats.MaxMS, stats.PeakInFlight)
		}
	}
	httpEmitLoad(t, map[string]any{
		"test":                             "kosync_ladder",
		"measurement_path":                 os.Getenv("HELIX_MEASUREMENT_PATH"),
		"redis_addr":                       redisAddr,
		"store":                            "redis",
		"contenders":                       contenders,
		"contender_peak":                   gate.peak.Load(),
		"documents":                        len(documents),
		"rounds_per_client":                rounds,
		"bcrypt_p50_ms":                    bcryptMS,
		"cpu_cores":                        runtime.NumCPU(),
		"theoretical_auth_ceiling_per_sec": float64(runtime.NumCPU()) * 1000 / bcryptMS,
		"results":                          ladder,
		"store_contention": map[string]any{
			"writers": storeContenders, "peak_inflight": storeGate.peak.Load(),
			"max_percentage": peakPct, "stored_percentage": storedProgress.Percentage, "errors": storeErrClasses,
		},
	})
}

func putProgress(ctx context.Context, t *testing.T, client *http.Client, base, username, password string, progress kosync.Progress) int {
	t.Helper()
	body, err := json.Marshal(progress)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, base+"/syncs/progress", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-auth-user", username)
	req.Header.Set("x-auth-key", password)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", progress.Document, err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20)); err != nil {
		t.Fatalf("reading PUT response: %v", err)
	}
	return resp.StatusCode
}

func getPercentage(ctx context.Context, t *testing.T, client *http.Client, base, username, password, document string) float64 {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/syncs/progress/"+url.PathEscape(document), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("x-auth-user", username)
	req.Header.Set("x-auth-key", password)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", document, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		t.Fatalf("GET %s returned %d (%s), want 200", document, resp.StatusCode, body)
	}
	var progress kosync.Progress
	if err := json.NewDecoder(resp.Body).Decode(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Document == "" {
		t.Fatalf("GET %s returned 200 with no stored progress ({}); expected the document just written", document)
	}
	return progress.Percentage
}
