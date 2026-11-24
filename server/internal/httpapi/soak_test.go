package httpapi

// GAP-013 / HRD-048: memory and goroutine soak. A fixed request mix runs for N
// iterations against a real listening server; live heap (post-GC) and goroutine
// counts are sampled and a least-squares slope is computed over the steady
// state. The detector is itself verified (§11.4.273): a deliberately leaking
// workload must be flagged, otherwise a PASS here would mean nothing.
//
// HELIX_SOAK_ITERS (default 6000) sets the length; HELIX_SOAK_REPORT, when set,
// receives the machine-readable result as JSON.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/sources"
	"io"
	"math"
	mrand "math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type soakResult struct {
	Iterations     int       `json:"iterations"`
	Samples        int       `json:"samples"`
	HeapBytes      []uint64  `json:"heap_alloc_bytes_post_gc"`
	Goroutines     []int     `json:"goroutines"`
	SlopeBytesIter float64   `json:"heap_slope_bytes_per_iteration"`
	R2             float64   `json:"heap_fit_r2"`
	GoroutineDelta int       `json:"goroutine_delta_first_to_last"`
	SteadyGrowthB  int64     `json:"steady_state_growth_bytes"`
	Leaked         bool      `json:"leak_detected"`
	Reasons        []string  `json:"reasons,omitempty"`
	GoVersion      string    `json:"go_version"`
	FinishedAt     time.Time `json:"finished_at"`
}

// Detection thresholds (documented in evidence/.../hrd048/README.txt).
// A leak is flagged only when ALL hold over the steady state: a regression slope
// above soakSlopeLimit bytes/iteration, a linear fit with R^2 above soakMinR2
// (growth that is steady, not a one-off step or noise), and absolute growth
// above soakGrowthLimit. Clean workloads measure |slope| of about 1 B/iter and
// growth of a few KiB, far below these limits.
const (
	soakSlopeLimit  = 32.0
	soakMinR2       = 0.80
	soakGrowthLimit = 64 << 10
	// A capacity-doubling leak (one slice appended to every iteration) grows in
	// steps and fits a line with R^2 of only ~0.77-0.80; when the growth is
	// large the R^2 gate relaxes to soakStepR2.
	soakStepGrowth    = 256 << 10
	soakStepR2        = 0.50
	soakGoroutineSlop = 3
	// soakMaxIterations caps HELIX_SOAK_ITERS.
	soakMaxIterations = 200000
)

// fit returns the least-squares slope per sample and R^2.
func fit(ys []uint64) (slope, r2 float64) {
	n := float64(len(ys))
	var sx, sy, sxx, sxy, syy float64
	for i, y := range ys {
		x, v := float64(i), float64(y)
		sx += x
		sy += v
		sxx += x * x
		sxy += x * v
		syy += v * v
	}
	den := n*sxx - sx*sx
	vden := n*syy - sy*sy
	if den == 0 || vden == 0 {
		return 0, 0
	}
	slope = (n*sxy - sx*sy) / den
	r := (n*sxy - sx*sy) / math.Sqrt(den*vden)
	return slope, r * r
}

// runSoak runs workload(i) iterations times, sampling 20 checkpoints.
func runSoak(iterations int, workload func(i int)) soakResult {
	const checkpoints = 20
	res := soakResult{Iterations: iterations, GoVersion: runtime.Version()}
	step := iterations / checkpoints
	if step < 1 {
		step = 1
	}
	sample := func() {
		// Let short-lived goroutines (bursts, in-flight requests) finish: wait
		// until the count stops falling, bounded at 300 ms.
		prev := runtime.NumGoroutine()
		for k := 0; k < 30; k++ {
			time.Sleep(10 * time.Millisecond)
			cur := runtime.NumGoroutine()
			if cur >= prev {
				break
			}
			prev = cur
		}
		runtime.GC()
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		res.HeapBytes = append(res.HeapBytes, m.HeapAlloc)
		res.Goroutines = append(res.Goroutines, runtime.NumGoroutine())
	}
	for i := 0; i < iterations; i++ {
		workload(i)
		if (i+1)%step == 0 {
			sample()
		}
	}
	analyzeSoak(&res, step)
	res.FinishedAt = time.Now().UTC()
	return res
}

// analyzeSoak applies the leak gates to the sampled series in res (HeapBytes,
// Goroutines) taken every step iterations. It is separate from runSoak so each
// gate can be tested against synthetic series.
func analyzeSoak(res *soakResult, step int) {
	res.Samples = len(res.HeapBytes)
	// Steady state: discard the first 30% (caches, pools, lazy init).
	skip := res.Samples * 3 / 10
	steady := res.HeapBytes[skip:]
	perSample, r2 := fit(steady)
	res.SlopeBytesIter = perSample / float64(step)
	res.R2 = r2
	res.SteadyGrowthB = int64(steady[len(steady)-1]) - int64(steady[0])
	res.GoroutineDelta = res.Goroutines[len(res.Goroutines)-1] - res.Goroutines[skip]
	if res.SlopeBytesIter > soakSlopeLimit && res.SteadyGrowthB > soakGrowthLimit &&
		(r2 > soakMinR2 || (r2 > soakStepR2 && res.SteadyGrowthB > soakStepGrowth)) {
		res.Leaked = true
		res.Reasons = append(res.Reasons, fmt.Sprintf("heap slope %.1f B/iter, R2 %.3f, growth %d B", res.SlopeBytesIter, r2, res.SteadyGrowthB))
	}
	if res.GoroutineDelta > soakGoroutineSlop {
		res.Leaked = true
		res.Reasons = append(res.Reasons, fmt.Sprintf("goroutines grew by %d", res.GoroutineDelta))
	}
}

// runSoakConfirmed runs the soak and, only if it flags a leak, runs it AGAIN on
// the same (now warm) system. A real leak persists into the second pass; a
// cache or pool that was still filling during the first one does not. This is
// the noise control for the no-false-positive guarantee.
func runSoakConfirmed(iterations int, workload func(i int)) soakResult {
	first := runSoak(iterations, workload)
	if !first.Leaked {
		return first
	}
	second := runSoak(iterations, workload)
	second.Reasons = append([]string{"confirmed on second pass; first pass: " + strings.Join(first.Reasons, "; ")}, second.Reasons...)
	return second
}

func soakIterations() int {
	// Bounded both ways: below 200 the detector has too few samples; above
	// soakMaxIterations a typo could occupy the host for hours (§12).
	if v, err := strconv.Atoi(os.Getenv("HELIX_SOAK_ITERS")); err == nil && v >= 200 {
		if v > soakMaxIterations {
			return soakMaxIterations
		}
		return v
	}
	return 6000
}

// Detector self-test (the RED, §11.4.273): leaks AT THE CLAIMED FLOOR must be
// reported at the default length, and clean workloads must never be.
func TestSoakDetectorCatchesLeaksAtTheClaimedFloor(t *testing.T) {
	n := soakIterations()
	for _, perIter := range []int{256, 512, 4096} {
		var sink [][]byte
		res := runSoak(n, func(i int) { sink = append(sink, make([]byte, perIter)) })
		runtime.KeepAlive(sink)
		if !res.Leaked {
			t.Fatalf("heap leak of %d B/iteration not detected: slope=%.1f r2=%.3f growth=%d", perIter, res.SlopeBytesIter, res.R2, res.SteadyGrowthB)
		}
	}
	// 1 goroutine per 200 iterations.
	stop := make(chan struct{})
	gor := runSoak(n, func(i int) {
		if i%200 == 0 {
			go func() { <-stop }()
		}
	})
	close(stop)
	if !gor.Leaked || gor.GoroutineDelta < 10 {
		t.Fatalf("goroutine leak (1 per 200) not detected: delta=%d", gor.GoroutineDelta)
	}
}

func TestSoakDetectorNoFalsePositiveOnCleanWorkload(t *testing.T) {
	for run := 1; run <= 5; run++ {
		var keep [][]byte
		res := runSoakConfirmed(soakIterations(), func(i int) {
			b := make([]byte, 4096)
			b[0] = byte(i)
			if i%20 == 0 { // bounded working set that fills early, then churns
				keep = append(keep, b)
				if len(keep) > 50 {
					keep = keep[1:]
				}
			}
		})
		runtime.KeepAlive(keep)
		if res.Leaked {
			t.Fatalf("run %d: false positive on a bounded workload: %v", run, res.Reasons)
		}
	}
}

func TestSoakServerRequestMixDoesNotLeak(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	server, _ := contentClient(t, s)
	tr := &http.Transport{DisableCompression: true}
	client := &http.Client{Transport: tr}
	defer tr.CloseIdleConnections()
	// Every request of the mix asserts its expected status and a body needle:
	// a server answering 404 for everything must FAIL here, not "not leak".
	const fixtureBody = "0123456789"
	get := func(path, rng, method string, wantStatus int, check func(body []byte) string) {
		req, _ := http.NewRequest(method, server.URL+path, nil)
		if rng != "" {
			req.Header.Set("Range", rng)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("%s %s: read body: %v", method, path, err)
		}
		if resp.StatusCode != wantStatus {
			t.Fatalf("%s %s: status %d, want %d (body %.80q)", method, path, resp.StatusCode, wantStatus, body)
		}
		if msg := check(body); msg != "" {
			t.Fatalf("%s %s: %s (body %.80q)", method, path, msg, body)
		}
	}
	contains := func(needle string) func([]byte) string {
		return func(b []byte) string {
			if !strings.Contains(string(b), needle) {
				return "body lacks " + strconv.Quote(needle)
			}
			return ""
		}
	}
	equals := func(want string) func([]byte) string {
		return func(b []byte) string {
			if string(b) != want {
				return "body differs from " + strconv.Quote(want)
			}
			return ""
		}
	}
	n := soakIterations()
	res := runSoakConfirmed(n, func(i int) {
		switch i % 6 {
		case 0:
			get("/v1/works", "", http.MethodGet, http.StatusOK, contains(`"w1"`))
		case 1:
			get("/v1/works/w1", "", http.MethodGet, http.StatusOK, contains(`"w1"`))
		case 2:
			get("/v1/files/f1/content", "", http.MethodGet, http.StatusOK, equals(fixtureBody))
		case 3:
			get("/v1/files/f1/content", "bytes=2-5", http.MethodGet, http.StatusPartialContent, equals(fixtureBody[2:6]))
		case 4:
			get("/v1/files/f1/content", "", http.MethodHead, http.StatusOK, equals(""))
		case 5:
			get("/opds/1.2/publications", "", http.MethodGet, http.StatusOK, contains("Book"))
		}
	})
	t.Logf("SOAK iterations=%d samples=%d slope=%.2f B/iter steady_growth=%d B goroutine_delta=%d heap_first=%d heap_last=%d",
		res.Iterations, res.Samples, res.SlopeBytesIter, res.SteadyGrowthB, res.GoroutineDelta, res.HeapBytes[0], res.HeapBytes[len(res.HeapBytes)-1])
	if path := os.Getenv("HELIX_SOAK_REPORT"); path != "" {
		data, _ := json.MarshalIndent(res, "", "  ")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if res.Leaked {
		t.Fatalf("leak detected: %v", res.Reasons)
	}
}

// ---- remote-source variant: WebDAV connector and rclone connector ----------

// rcloneWorkers counts rclone children of this test process other than the
// long-lived `serve` target: the short-lived `cat` / `lsjson` helpers the
// rclone connector spawns. Linux /proc only.
func rcloneWorkers() int {
	me := strconv.Itoa(os.Getpid())
	dirs, _ := os.ReadDir("/proc")
	n := 0
	for _, d := range dirs {
		if _, err := strconv.Atoi(d.Name()); err != nil {
			continue
		}
		status, err := os.ReadFile("/proc/" + d.Name() + "/status")
		if err != nil || !strings.Contains(string(status), "\nPPid:\t"+me+"\n") {
			continue
		}
		// A zombie (exited, never waited for) has an empty cmdline; recognise
		// it by state Z and its command name, so a missing cmd.Wait() counts.
		if st, err := os.ReadFile("/proc/" + d.Name() + "/stat"); err == nil {
			str := string(st)
			if open, cl := strings.IndexByte(str, '('), strings.LastIndexByte(str, ')'); open >= 0 && cl > open && cl+2 < len(str) {
				if str[cl+2] == 'Z' && strings.HasPrefix(str[open+1:cl], "rclone") {
					n++
					continue
				}
			}
		}
		cmd, _ := os.ReadFile("/proc/" + d.Name() + "/cmdline")
		c := strings.ReplaceAll(string(cmd), "\x00", " ")
		if strings.Contains(c, "rclone") && !strings.Contains(c, " serve ") {
			n++
		}
	}
	return n
}

type remoteFixture struct {
	ts   *httptest.Server
	body []byte
}

func newRemoteFixture(t *testing.T, kind string) remoteFixture {
	t.Helper()
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone not installed")
	}
	root := t.TempDir()
	body := make([]byte, 200_000)
	if _, err := rand.Read(body); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "books"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "books", "r.bin"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(t.TempDir(), "rclone.conf")
	if err := os.WriteFile(cfg, []byte("[wfdav]\ntype = local\nnolink = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var conn sources.Connector
	var err error
	if kind == "webdav" {
		addr := freeTCPAddr(t)
		serve := exec.Command("rclone", "--config", cfg, "serve", "webdav", "wfdav:", "--addr", addr)
		serve.Dir = root
		logs := &serveLogBuffer{}
		serve.Stdout, serve.Stderr = logs, logs
		if err := serve.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = serve.Process.Kill(); _ = serve.Wait() })
		waitForWebDAVPath(t, "http://"+addr, "books/r.bin", fmt.Sprintf("%x", sha256.Sum256(body)), logs)
		conn, err = sources.NewWebDAV(sources.Config{Protocol: "webdav", Endpoint: "http://" + addr + "/"})
	} else {
		conn, err = sources.NewRclone(sources.Config{Protocol: "sftp", Endpoint: "wfdav:" + root, RcloneConfig: cfg})
	}
	if err != nil {
		t.Fatal(err)
	}
	entries, err := conn.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	now := time.Now().UTC()
	for _, e := range entries {
		if e.Path != "books/r.bin" {
			continue
		}
		if err := repo.UpsertWork(context.Background(), catalog.Work{ID: "w", CanonicalTitle: "w", CreatedAt: now, UpdatedAt: now,
			Editions: []catalog.Edition{{ID: "e", Title: "w", State: catalog.Identified, UpdatedAt: now,
				Files: []catalog.File{{ID: "f", SourceID: "src", Path: e.Path, Format: "bin", MIMEType: "application/octet-stream",
					Size: e.Size, SHA256: fmt.Sprintf("%x", sha256.Sum256(body)), ModifiedAt: e.Modified}}}}}); err != nil {
			t.Fatal(err)
		}
	}
	srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	srv.connectors["src"] = conn
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return remoteFixture{ts: ts, body: body}
}

func TestSoakRemoteSourcesDoNotLeakGoroutinesHeapOrProcesses(t *testing.T) {
	for kind, iters := range map[string]int{"webdav": 3000, "rclone": 300} {
		kind, iters := kind, iters
		t.Run(kind, func(t *testing.T) {
			fx := newRemoteFixture(t, kind)
			tr := &http.Transport{DisableCompression: true}
			client := &http.Client{Transport: tr}
			defer tr.CloseIdleConnections()
			res := runSoakConfirmed(iters, func(i int) {
				req, _ := http.NewRequest(http.MethodGet, fx.ts.URL+"/v1/files/f/content", nil)
				want := fx.body
				if i%2 == 0 {
					req.Header.Set("Range", "bytes=1000-4999")
					want = fx.body[1000:5000]
				}
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				b, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil || !bytes.Equal(b, want) {
					t.Fatalf("iteration %d: err=%v got %d bytes want %d", i, err, len(b), len(want))
				}
			})
			// rclone helpers exit when the request ends; give the reaper a moment.
			deadline := time.Now().Add(5 * time.Second)
			for rcloneWorkers() > 0 && time.Now().Before(deadline) {
				time.Sleep(50 * time.Millisecond)
			}
			workers := rcloneWorkers()
			t.Logf("REMOTE SOAK %s iterations=%d slope=%.2f B/iter r2=%.2f goroutine_delta=%d lingering_rclone_workers=%d",
				kind, res.Iterations, res.SlopeBytesIter, res.R2, res.GoroutineDelta, workers)
			if path := os.Getenv("HELIX_SOAK_REPORT"); path != "" {
				data, _ := json.MarshalIndent(res, "", "  ")
				_ = os.WriteFile(strings.TrimSuffix(path, ".json")+"_remote_"+kind+".json", data, 0o644)
			}
			if res.Leaked {
				t.Fatalf("leak detected: %v", res.Reasons)
			}
			if workers != 0 {
				t.Fatalf("%d rclone helper processes outlived their requests", workers)
			}
		})
	}
}

// UNCONFIRMED question from review: is there a cap on concurrent rclone
// helpers? Fire 50 concurrent ranged requests at an rclone source, sample the
// peak helper count, require every body correct and zero stragglers.
func TestRcloneConcurrentRangePeakProcesses(t *testing.T) {
	// Spawns up to 50 rclone helper processes: opt-in only (§12.12), never on a
	// plain `go test`.
	if os.Getenv("HELIX_SOAK_RCLONE_PROBE") != "1" {
		t.Skip("feature_disabled_by_config: HELIX_SOAK_RCLONE_PROBE is not 1; concurrency probe opted out")
	}
	fx := newRemoteFixture(t, "rclone")
	tr := &http.Transport{DisableCompression: true, MaxIdleConnsPerHost: 64}
	client := &http.Client{Transport: tr}
	defer tr.CloseIdleConnections()
	var peak atomic.Int64
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				if n := int64(rcloneWorkers()); n > peak.Load() {
					peak.Store(n)
				}
				time.Sleep(2 * time.Millisecond)
			}
		}
	}()
	var wg sync.WaitGroup
	errs := make(chan error, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := 1000 + i*100
			req, _ := http.NewRequest(http.MethodGet, fx.ts.URL+"/v1/files/f/content", nil)
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, start+4095))
			resp, err := client.Do(req)
			if err != nil {
				errs <- err
				return
			}
			b, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || resp.StatusCode != 206 || !bytes.Equal(b, fx.body[start:start+4096]) {
				errs <- fmt.Errorf("request %d: status %d err %v len %d", i, resp.StatusCode, err, len(b))
			}
		}(i)
	}
	wg.Wait()
	close(stop)
	<-done
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	t.Logf("RCLONE CONCURRENCY requests=50 peak_rclone_helper_processes=%d lingering=%d", peak.Load(), rcloneWorkers())
	if rcloneWorkers() != 0 {
		t.Fatalf("%d helpers lingered", rcloneWorkers())
	}
}

// ---- detector shape coverage (round-2 review) ------------------------------

type soakShape struct {
	name string
	mk   func() func(i int)
}

// staircase leaks append to ONE growing slice, so the heap grows in capacity
// steps rather than linearly; they defeated an R^2-only gate.
func staircaseLeak(perIter int) soakShape {
	return soakShape{fmt.Sprintf("staircase_%dB_per_iter", perIter), func() func(int) {
		var buf []byte
		chunk := make([]byte, perIter)
		return func(i int) { buf = append(buf, chunk...); runtime.KeepAlive(buf) }
	}}
}

func cleanShapes() []soakShape {
	return []soakShape{
		{"gc_heavy_bursty", func() func(int) {
			var keep [][]byte
			return func(i int) {
				k := 1
				if i%50 == 0 {
					k = 200
				}
				for j := 0; j < k; j++ {
					_ = make([]byte, 8192)
				}
				if i%97 == 0 {
					keep = append(keep, make([]byte, 1024))
					if len(keep) > 100 {
						keep = keep[1:]
					}
				}
			}
		}},
		{"random_evict_cache", func() func(int) {
			m := map[int][]byte{}
			r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
			return func(i int) {
				m[r.Intn(500)] = make([]byte, 2048+r.Intn(4096))
				if len(m) > 400 {
					for k := range m {
						delete(m, k)
						break
					}
				}
			}
		}},
		{"cache_filling_over_two_thirds_of_run", func() func(int) {
			m := map[int][]byte{}
			return func(i int) {
				if i < 4000 && i%2 == 0 {
					m[i] = make([]byte, 256)
				}
			}
		}},
		{"goroutine_bursts", func() func(int) {
			return func(i int) {
				if i%100 == 0 {
					for j := 0; j < 20; j++ {
						go func() { time.Sleep(time.Duration(mrand.Intn(3)) * time.Millisecond) }()
					}
				}
			}
		}},
		{"sawtooth_release_every_1500", func() func(int) {
			var s [][]byte
			return func(i int) {
				s = append(s, make([]byte, 512))
				if i%1500 == 1499 {
					s = nil
				}
			}
		}},
	}
}

func TestSoakDetectorCatchesStaircaseLeaks(t *testing.T) {
	for _, perIter := range []int{100, 300, 512} {
		sh := staircaseLeak(perIter)
		for run := 0; run < 3; run++ {
			res := runSoakConfirmed(soakIterations(), sh.mk())
			if !res.Leaked {
				t.Fatalf("%s run %d not detected: slope=%.1f r2=%.3f growth=%d", sh.name, run, res.SlopeBytesIter, res.R2, res.SteadyGrowthB)
			}
		}
	}
}

func TestSoakDetectorNoFalsePositiveOnBurstyShapes(t *testing.T) {
	for _, sh := range cleanShapes() {
		for run := 1; run <= 10; run++ {
			res := runSoakConfirmed(soakIterations(), sh.mk())
			if res.Leaked {
				t.Fatalf("%s run %d: confirmed false positive: %v", sh.name, run, res.Reasons)
			}
		}
	}
}

// The helper counter must see zombies too: a child that exited and was never
// waited for has an empty cmdline, which hid a missing cmd.Wait().
func TestRcloneWorkersCounterSeesZombiesAndLiveHelpers(t *testing.T) {
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone not installed")
	}
	base := rcloneWorkers()
	z := exec.Command("rclone", "version")
	if err := z.Start(); err != nil {
		t.Fatal(err)
	}
	defer z.Wait()
	deadline := time.Now().Add(10 * time.Second)
	for rcloneWorkers() < base+1 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(1500 * time.Millisecond) // exited by now, never waited: zombie
	if got := rcloneWorkers(); got != base+1 {
		t.Fatalf("zombie rclone child not counted: base %d, now %d", base, got)
	}
	_ = z.Wait()
	deadline = time.Now().Add(5 * time.Second)
	for rcloneWorkers() != base && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if got := rcloneWorkers(); got != base {
		t.Fatalf("after Wait the count should return to %d, got %d", base, got)
	}
}
