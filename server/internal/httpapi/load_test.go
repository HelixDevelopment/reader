package httpapi

// HTTP-level concurrency ladder for ATM-027. This exercises the routes a
// client actually hits — /v1/works offset browse, /v1/works deep keyset page,
// /v1/works text search and the OPDS 2.0 acquisition feed — through the real
// Gin router and a real TCP listener, with the PostgreSQL catalog seam behind
// them. Because the listener and the client run in one process, the HTTP path
// itself never crosses podman's pasta forwarder; only the database connection
// does, which is why the authoritative run is the one made from inside the
// database container with HELIX_MEASUREMENT_PATH=in_container_clean.
//
// Skips cleanly unless HELIX_TEST_POSTGRES_DSN is set and HELIX_LOAD_RUN=1.
//
//	HELIX_TEST_POSTGRES_DSN=... HELIX_LOAD_RUN=1 go test ./internal/httpapi -run TestLiveLoadHTTPLadder -v -timeout 45m

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/sources"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func httpLoadWanted(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	if os.Getenv("HELIX_LOAD_RUN") != "1" {
		t.Skip("HELIX_LOAD_RUN is not 1; measurement harness opted out")
	}
	return dsn
}

func httpEmitLoad(t *testing.T, payload map[string]any) {
	t.Helper()
	payload["emitted_at"] = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	t.Logf("LOADJSON %s", data)
	if path := os.Getenv("HELIX_LOAD_OUTPUT"); path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			t.Errorf("open %s: %v", path, err)
			return
		}
		defer f.Close()
		if _, err := f.Write(append(data, '\n')); err != nil {
			t.Errorf("append: %v", err)
		}
	}
}

type httpLadderResult struct {
	Workload     string         `json:"workload"`
	Clients      int            `json:"clients"`
	RoundsPerCli int            `json:"rounds_per_client"`
	Completed    int            `json:"completed"`
	Errors       map[string]int `json:"errors_by_class"`
	WallMS       float64        `json:"wall_ms"`
	ReqPerSec    float64        `json:"req_per_sec"`
	P50MS        float64        `json:"p50_ms"`
	P95MS        float64        `json:"p95_ms"`
	P99MS        float64        `json:"p99_ms"`
	MaxMS        float64        `json:"max_ms"`
	MeanMS       float64        `json:"mean_ms"`
	MeanBytes    float64        `json:"mean_response_bytes"`
}

// hostCPUTicks reads the aggregate cpu line of /proc/stat. The CPU subsystem is
// not namespaced, so this reports the whole machine even when the harness runs
// inside the database container — which is exactly the distinction the sampler
// above cannot make. A cell measured while an unrelated build occupies the cores
// is not a slower catalog; it is a slower machine.
//
// Only the first line may be read: the per-core cpuN lines repeat the columns and
// the intr line's counters would poison the totals. guest and guest_nice are
// excluded because the kernel already counts them inside user.
func hostCPUTicks() (busy uint64, total uint64, err error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, 0, fmt.Errorf("/proc/stat unreadable: %v", scanner.Err())
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 9 || fields[0] != "cpu" {
		return 0, 0, fmt.Errorf("unexpected /proc/stat first line %q", strings.Join(fields, " "))
	}
	var idle uint64
	for i, raw := range fields[1:9] {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("field %d of %q: %v", i, strings.Join(fields, " "), err)
		}
		total += v
		// idle and iowait are the only two not attributed to runnable work.
		if i == 3 || i == 4 {
			idle += v
		}
	}
	return total - idle, total, nil
}

func hostLoadAvg1m() float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return -1
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return -1
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return -1
	}
	return v
}

// hostContention brackets one cell and reports what fraction of host CPU time
// went to runnable work during it, normalised by core count so 1.0 means every
// core busy. loadavg is reported too because it counts runnable *and* blocked,
// and a value far above the busy fraction points at contention the CPU figure
// alone would call idle.
func hostContention(busyBefore, totalBefore uint64, wallMS float64) map[string]any {
	busy, total, err := hostCPUTicks()
	if err != nil {
		return map[string]any{"error": err.Error(), "window_ms": wallMS}
	}
	db, dt := busy-busyBefore, total-totalBefore
	cores := runtime.NumCPU()
	out := map[string]any{
		"cores":          cores,
		"loadavg_1m":     hostLoadAvg1m(),
		"window_ms":      wallMS,
		"cpu_busy_ratio": 0.0,
		"cpu_busy_cores": 0.0,
	}
	if dt > 0 {
		ratio := float64(db) / float64(dt)
		out["cpu_busy_ratio"] = math.Round(ratio*10000) / 10000
		out["cpu_busy_cores"] = math.Round(ratio*float64(cores)*100) / 100
	}
	return out
}

// hostIdleProbe samples the machine for a short window while the harness itself
// is doing nothing. cpu_busy_ratio during a cell cannot be attributed on its own
// — a parallel sequential scan is busy too — but the entry value is load that
// existed before the cell released any client, which is what makes a co-tenant
// build visible instead of looking like a slow query.
func hostIdleProbe(window time.Duration) (coresBusy float64, loadavg float64, err error) {
	busy0, total0, err := hostCPUTicks()
	if err != nil {
		return 0, 0, err
	}
	time.Sleep(window)
	out := hostContention(busy0, total0, float64(window.Microseconds())/1000)
	if e, ok := out["error"]; ok {
		return 0, 0, fmt.Errorf("host cpu probe: %v", e)
	}
	return out["cpu_busy_cores"].(float64), out["loadavg_1m"].(float64), nil
}

func httpPercentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func runHTTPLadder(ctx context.Context, client *http.Client, name string, clients, rounds int, timeout time.Duration, url string) httpLadderResult {
	start := make(chan struct{})
	var wg sync.WaitGroup
	latencies := make([][]float64, clients)
	bytesPer := make([][]int64, clients)
	errCount := map[string]int{}
	var emu sync.Mutex
	wallStart := time.Now()
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			mine := make([]float64, 0, rounds)
			size := make([]int64, 0, rounds)
			for r := 0; r < rounds; r++ {
				<-start
				rctx, cancel := context.WithTimeout(ctx, timeout)
				req, err := http.NewRequestWithContext(rctx, http.MethodGet, url, nil)
				if err != nil {
					emu.Lock()
					errCount["request_build"]++
					emu.Unlock()
					cancel()
					continue
				}
				t0 := time.Now()
				resp, err := client.Do(req)
				if err != nil {
					cls := "other"
					if strings.Contains(err.Error(), "context deadline exceeded") {
						cls = "deadline"
					}
					emu.Lock()
					errCount[cls]++
					emu.Unlock()
					cancel()
					continue
				}
				n, err := io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				mine = append(mine, float64(time.Since(t0).Microseconds())/1000)
				size = append(size, n)
				if err != nil {
					emu.Lock()
					errCount["body_read"]++
					emu.Unlock()
				} else if resp.StatusCode != http.StatusOK {
					emu.Lock()
					errCount["http_"+strconv.Itoa(resp.StatusCode)]++
					emu.Unlock()
				}
				cancel()
			}
			latencies[c] = mine
			bytesPer[c] = size
		}(c)
	}
	close(start)
	wg.Wait()
	wall := float64(time.Since(wallStart).Microseconds()) / 1000
	all := []float64{}
	totalBytes := int64(0)
	counted := 0
	for _, per := range latencies {
		all = append(all, per...)
	}
	for _, per := range bytesPer {
		for _, n := range per {
			totalBytes += n
			counted++
		}
	}
	errTotal := 0
	for _, n := range errCount {
		errTotal += n
	}
	res := httpLadderResult{Workload: name, Clients: clients, RoundsPerCli: rounds, Completed: len(all) - errTotal, Errors: errCount, WallMS: wall}
	if len(all) > 0 && wall > 0 {
		sum := 0.0
		for _, v := range all {
			sum += v
		}
		res.ReqPerSec = float64(res.Completed) / (wall / 1000)
		res.MeanMS = sum / float64(len(all))
		res.P50MS = httpPercentile(all, 50)
		res.P95MS = httpPercentile(all, 95)
		res.P99MS = httpPercentile(all, 99)
		res.MaxMS = httpPercentile(all, 100)
	}
	if counted > 0 {
		res.MeanBytes = float64(totalBytes) / float64(counted)
	}
	return res
}

// httpResourceSample polls PostgreSQL views and this process; the process
// here contains the server itself (router, Gin, pool), so RSS and CPU
// describe the server envelope, labelled as such.
type httpResourceSampler struct {
	db       *sql.DB
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	rows     []map[string]any
}

func (s *httpResourceSampler) loop() {
	defer close(s.done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			row := map[string]any{"at": time.Now().UTC().Format(time.RFC3339Nano)}
			var ungranted int
			if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM pg_locks WHERE NOT granted").Scan(&ungranted); err == nil {
				row["ungranted_locks"] = ungranted
			}
			var backends int
			if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity").Scan(&backends); err == nil {
				row["pg_backends"] = backends
			}
			var deadlocks int64
			if err := s.db.QueryRowContext(ctx, "SELECT deadlocks FROM pg_stat_database WHERE datname = current_database()").Scan(&deadlocks); err == nil {
				row["db_deadlocks"] = deadlocks
			}
			rows, err := s.db.QueryContext(ctx, `SELECT coalesce(wait_event_type,'cpu'), coalesce(wait_event,'(none)'), count(*) FROM pg_stat_activity WHERE state='active' AND pid <> pg_backend_pid() GROUP BY 1,2`)
			if err == nil {
				waits := []string{}
				for rows.Next() {
					var t, e string
					var n int
					if rows.Scan(&t, &e, &n) == nil {
						waits = append(waits, fmt.Sprintf("%s:%s=%d", t, e, n))
					}
				}
				rows.Close()
				row["waits"] = waits
			}
			cancel()
			row["goroutines"] = runtime.NumGoroutine()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			row["go_heap_inuse_bytes"] = m.HeapInuse
			row["go_sys_bytes"] = m.Sys
			if f, err := os.Open("/proc/self/status"); err == nil {
				sc := bufio.NewScanner(f)
				for sc.Scan() {
					if rest, ok := strings.CutPrefix(sc.Text(), "VmRSS:"); ok {
						var v int64
						if _, err := fmt.Sscanf(rest, "%d kB", &v); err == nil {
							row["self_rss_kb"] = v
						}
					}
				}
				f.Close()
			}
			if data, err := os.ReadFile("/proc/self/stat"); err == nil {
				text := string(data)
				if idx := strings.LastIndex(text, ")"); idx >= 0 {
					fields := strings.Fields(text[idx+1:])
					if len(fields) >= 14 {
						var u, s2 int64
						u, _ = strconv.ParseInt(fields[11], 10, 64)
						s2, _ = strconv.ParseInt(fields[12], 10, 64)
						row["self_cpu_ticks_total"] = u + s2
					}
				}
			}
			if v, ok := httpLoadReadCgroup(); ok {
				row["cgroup_memory_current_bytes"] = v
			}
			s.mu.Lock()
			s.rows = append(s.rows, row)
			s.mu.Unlock()
		}
	}
}

func httpLoadReadCgroup() (int64, bool) {
	if os.Getenv("HELIX_LOAD_CGROUP") != "1" {
		return 0, false
	}
	data, err := os.ReadFile("/sys/fs/cgroup/memory.current")
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	return v, err == nil
}

func (s *httpResourceSampler) Close() {
	close(s.stop)
	<-s.done
}

func (s *httpResourceSampler) snapshot() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.rows
	s.rows = nil
	return out
}

func httpSummarize(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return map[string]any{"samples": 0}
	}
	out := map[string]any{"samples": len(rows)}
	for _, key := range []string{"ungranted_locks", "pg_backends", "db_deadlocks", "goroutines", "self_rss_kb", "go_heap_inuse_bytes", "go_sys_bytes", "self_cpu_ticks_total", "cgroup_memory_current_bytes"} {
		peak := math.Inf(-1)
		var first, last float64
		seen := 0
		for i, r := range rows {
			v, ok := r[key].(int64)
			if !ok {
				if vi, ok2 := r[key].(int); ok2 {
					v = int64(vi)
				} else {
					continue
				}
			}
			if seen == 0 {
				first = float64(v)
			}
			last = float64(v)
			seen++
			if float64(v) > peak {
				peak = float64(v)
			}
			_ = i
		}
		if seen > 0 {
			out["peak_"+key] = peak
			if last >= first {
				out["delta_"+key] = last - first
			}
		}
	}
	waits := map[string]int{}
	for _, r := range rows {
		list, _ := r["waits"].([]string)
		for _, w := range list {
			waits[w]++
		}
	}
	out["wait_event_observations"] = waits
	return out
}

func TestLiveLoadHTTPLadder(t *testing.T) {
	dsn := httpLoadWanted(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	repo, err := catalog.NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	srv := New(config.Config{Address: "127.0.0.1:0", ShutdownTimeout: 15 * time.Second, BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	probeDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer probeDB.Close()
	probeDB.SetMaxOpenConns(2)

	var fixtureWorks int
	if err := probeDB.QueryRowContext(ctx, "SELECT count(*) FROM works").Scan(&fixtureWorks); err != nil {
		t.Fatal(err)
	}
	if fixtureWorks < 50000 {
		t.Skipf("catalog holds %d works; the scale fixture is not seeded, run TestLiveScaleCatalog first", fixtureWorks)
	}

	// The deep cursor is derived through the API itself, the way a paging
	// client would obtain it: walk an offset page and use the last row.
	deepPage, err := httpGETJSON(ctx, ts.URL+"/v1/works?offset=40000&limit=20")
	if err != nil {
		t.Fatalf("deep page: %v", err)
	}
	works, _ := deepPage["works"].([]any)
	if len(works) != 20 {
		t.Fatalf("deep page works = %d, want 20", len(works))
	}
	last := works[19].(map[string]any)
	rawUpdatedAt, _ := last["updated_at"].(string)
	updatedAt, err := time.Parse(time.RFC3339Nano, rawUpdatedAt)
	if err != nil {
		t.Fatalf("parse updated_at %q: %v", rawUpdatedAt, err)
	}
	cursor := catalog.Cursor{UpdatedAt: updatedAt.UTC().Truncate(time.Microsecond), ID: last["id"].(string)}
	deepURL := ts.URL + "/v1/works?after=" + cursor.Encode() + "&limit=20"

	sampler := &httpResourceSampler{db: probeDB, interval: 500 * time.Millisecond, stop: make(chan struct{}), done: make(chan struct{})}
	go sampler.loop()
	defer sampler.Close()

	client := &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 256, MaxConnsPerHost: 0}}
	rounds := 8
	if v := os.Getenv("HELIX_LOAD_ROUNDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rounds = n
		}
	}
	timeout := 30 * time.Second
	if v := os.Getenv("HELIX_LOAD_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}
	levels := []int{1, 8, 32, 64}
	if v := os.Getenv("HELIX_LOAD_LEVELS"); v != "" {
		levels = nil
		for _, part := range strings.Split(v, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || n < 1 {
				t.Fatalf("HELIX_LOAD_LEVELS %q: %v", v, err)
			}
			levels = append(levels, n)
		}
	}
	// Warmup, not reported.
	wu := runHTTPLadder(ctx, client, "warmup", 4, 2, timeout, ts.URL+"/v1/works?offset=0&limit=20")
	_ = wu
	sampler.snapshot()

	routes := []struct {
		name   string
		url    string
		rounds int
	}{
		{"GET /v1/works offset browse", ts.URL + "/v1/works?offset=0&limit=20", rounds},
		{"GET /v1/works deep keyset page", deepURL, rounds},
		{"GET /v1/works text search", ts.URL + "/v1/works?q=" + strings.ReplaceAll("Scale Volume 42", " ", "+") + "&limit=20", maxInt(1, rounds/4)},
		{"GET /opds/2.0/publications feed build", ts.URL + "/opds/2.0/publications?limit=20", rounds},
	}
	results := []httpLadderResult{}
	contention := map[string]any{}
	host := map[string]any{}
	for _, r := range routes {
		for _, c := range levels {
			entryCores, entryLoad, err := hostIdleProbe(150 * time.Millisecond)
			if err != nil {
				t.Fatalf("host entry probe: %v", err)
			}
			busyBefore, totalBefore, err := hostCPUTicks()
			if err != nil {
				t.Fatalf("host cpu baseline: %v", err)
			}
			res := runHTTPLadder(ctx, client, r.name, c, r.rounds, timeout, r.url)
			// The entry probe alone can be fooled by a co-tenant that starts during
			// the run, and a ladder run is only tens of seconds long.
			exitCores, _, err := hostIdleProbe(150 * time.Millisecond)
			if err != nil {
				t.Fatalf("host exit probe: %v", err)
			}
			results = append(results, res)
			contention[r.name+"@clients"+strconv.Itoa(c)] = httpSummarize(sampler.snapshot())
			key := r.name + "@clients" + strconv.Itoa(c)
			hc := hostContention(busyBefore, totalBefore, res.WallMS)
			hc["entry_busy_cores"] = entryCores
			hc["entry_loadavg_1m"] = entryLoad
			hc["exit_busy_cores"] = exitCores
			host[key] = hc
			t.Logf("HTTP LADDER %q clients=%d completed=%d errors=%v req_per_sec=%.2f p50=%.2f p95=%.2f p99=%.2f max=%.2f mean_bytes=%.0f host_cores at_entry=%.2f during=%.2f after=%.2f loadavg=%.2f",
				res.Workload, res.Clients, res.Completed, res.Errors, res.ReqPerSec, res.P50MS, res.P95MS, res.P99MS, res.MaxMS, res.MeanBytes,
				entryCores, hc["cpu_busy_cores"], exitCores, entryLoad)
		}
	}
	// A cell entered while the machine already has half its cores busy measures a
	// saturated host, and its latency is not a property of the catalog. The clean
	// network path was never sufficient for that claim, so the payload states both
	// conditions instead of leaving a reader to infer them.
	cores := float64(runtime.NumCPU())
	maxExternal, contendedCells := 0.0, 0
	for _, v := range host {
		cell := v.(map[string]any)
		external := math.Max(cell["entry_busy_cores"].(float64), cell["exit_busy_cores"].(float64))
		if external > maxExternal {
			maxExternal = external
		}
		if external > cores/2 {
			contendedCells++
		}
	}
	hostQuiet := contendedCells == 0
	if !hostQuiet && os.Getenv("HELIX_MEASUREMENT_PATH") == "in_container_clean" {
		t.Logf("HOST NOT QUIET: %d of %d cells had >%.0f%% of cores busy by other processes just before or after them (peak %.1f/%d). The network path is clean but these latencies describe a saturated machine; do not record this run as authoritative.",
			contendedCells, len(host), 50.0, maxExternal, runtime.NumCPU())
	}
	httpEmitLoad(t, map[string]any{
		"test":                   "http_ladder",
		"measurement_path":       os.Getenv("HELIX_MEASUREMENT_PATH"),
		"host_quiet":             hostQuiet,
		"host_contended_cells":   contendedCells,
		"host_cells_total":       len(host),
		"host_cores":             runtime.NumCPU(),
		"host_external_busy_max": maxExternal,
		"fixture_works_total":    fixtureWorks,
		"pool_max_open_default":  20,
		"request_timeout_s":      int(timeout.Seconds()),
		"results":                results,
		"contention":             contention,
		"host_contention":        host,
		"host_contention_note":   "entry_busy_cores is host-wide CPU busy in a 150ms window before the cell released any client: load from another project on this machine. cpu_busy_cores covers the cell itself and includes this harness's own parallel scans, so only the entry figure indicts a co-tenant",
	})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func httpGETJSON(ctx context.Context, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d body %s", resp.StatusCode, bytes.TrimSpace(data))
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
