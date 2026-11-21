package catalog

// Concurrency, contention and resource-envelope measurements for ATM-027.
// These tests are measurement harnesses: they add no production behavior and
// skip cleanly unless HELIX_TEST_POSTGRES_DSN is set AND HELIX_LOAD_RUN=1, so
// an ordinary `go test ./...` never pays for them.
//
// The numbers only mean what their measurement path allows. Run inside the
// database container for authoritative figures (see scaleInContainerHint);
// from the host through podman's pasta forwarder multi-packet responses stall
// and every latency figure describes that path instead of the catalog. Each
// payload therefore carries measurement_path as recorded by the runner.
//
// Harness usage:
//
//	HELIX_TEST_POSTGRES_DSN=... HELIX_LOAD_RUN=1 HELIX_MEASUREMENT_PATH=in_container_clean \
//	  go test ./internal/catalog -run 'TestLiveLoad' -v -timeout 45m

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// loadWanted gates every measurement test: database configured, run opted in.
func loadWanted(t *testing.T) string {
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

func loadMeasurementPath() string {
	if v := os.Getenv("HELIX_MEASUREMENT_PATH"); v != "" {
		return v
	}
	return "unspecified"
}

// loadLevels is the concurrency ladder. HELIX_LOAD_LEVELS lets a rerun shrink
// the ladder while debugging without recompiling.
func loadLevels() []int {
	if v := os.Getenv("HELIX_LOAD_LEVELS"); v != "" {
		out := []int{}
		for _, part := range strings.Split(v, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || n < 1 {
				panic(fmt.Errorf("HELIX_LOAD_LEVELS: %q", part))
			}
			out = append(out, n)
		}
		return out
	}
	return []int{1, 8, 32, 64}
}

func loadRoundsPerClient() int {
	if v := os.Getenv("HELIX_LOAD_ROUNDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			panic(fmt.Errorf("HELIX_LOAD_ROUNDS: %q", v))
		}
		return n
	}
	return 8
}

func loadRequestTimeout() time.Duration {
	if v := os.Getenv("HELIX_LOAD_TIMEOUT_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 30 * time.Second
}

// emitLoad prints the payload as one LOADJSON line and appends it to
// $HELIX_LOAD_OUTPUT when set, so a runner can persist results incrementally
// and nothing is held only in a test log.
func emitLoad(t *testing.T, payload map[string]any) {
	t.Helper()
	payload["emitted_at"] = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
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
			t.Errorf("append %s: %v", path, err)
		}
	}
}

// percentile takes p in [0,100] over a sorted-able slice (copies internally).
func percentile(values []float64, p float64) float64 {
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

func errClass(err error) string {
	switch {
	case err == nil:
		return ""
	case strings.Contains(err.Error(), "context deadline exceeded"):
		return "deadline"
	case strings.Contains(err.Error(), "context canceled"):
		return "canceled"
	case strings.Contains(err.Error(), "too many connections"), strings.Contains(err.Error(), "FATAL"):
		return "db_connections"
	case strings.Contains(err.Error(), "connection refused"):
		return "db_unreachable"
	default:
		return "other"
	}
}

// ---- resource and contention samplers ----

// loadSampler owns a private two-connection pool so its queries do not steal
// from the pool under measurement, and samples PostgreSQL views plus this
// process's own resource counters. The sampled process is the harness: for
// in-process measurements it is the same Go code, pool configuration and
// query shapes the server binary runs, but it is not a separately deployed
// container; the payload says so.
type loadSampler struct {
	db       *sql.DB
	repo     *Postgres
	interval time.Duration
	cgroup   bool

	stop chan struct{}
	done chan struct{}
	mu   sync.Mutex
	rows []map[string]any
}

func newLoadSampler(ctx context.Context, dsn string, repo *Postgres, interval time.Duration) (*loadSampler, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	s := &loadSampler{db: db, repo: repo, interval: interval, cgroup: os.Getenv("HELIX_LOAD_CGROUP") == "1", stop: make(chan struct{}), done: make(chan struct{})}
	go s.loop()
	return s, nil
}

func (s *loadSampler) Close() {
	close(s.stop)
	<-s.done
	s.db.Close()
}

func (s *loadSampler) loop() {
	defer close(s.done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			sample := s.sampleOnce(ctx)
			cancel()
			s.mu.Lock()
			s.rows = append(s.rows, sample)
			s.mu.Unlock()
		}
	}
}

// snapshot returns and forgets the samples collected so far; call between
// ladder phases so each phase reports its own contention picture.
func (s *loadSampler) snapshot() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.rows
	s.rows = nil
	return out
}

func (s *loadSampler) sampleOnce(ctx context.Context) map[string]any {
	sample := map[string]any{"at": time.Now().UTC().Format(time.RFC3339Nano)}
	var ungranted int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM pg_locks WHERE NOT granted").Scan(&ungranted); err == nil {
		sample["ungranted_locks"] = ungranted
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
		sample["wait_events"] = waits
	}
	var backends, maxConns int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity").Scan(&backends); err == nil {
		sample["pg_backends"] = backends
	}
	if err := s.db.QueryRowContext(ctx, "SELECT current_setting('max_connections')::int").Scan(&maxConns); err == nil {
		sample["max_connections"] = maxConns
	}
	var deadlocks, xactCommit, tempFiles int64
	var tempBytes int64
	if err := s.db.QueryRowContext(ctx, `SELECT deadlocks, xact_commit, temp_files, temp_bytes FROM pg_stat_database WHERE datname = current_database()`).Scan(&deadlocks, &xactCommit, &tempFiles, &tempBytes); err == nil {
		sample["db_deadlocks"] = deadlocks
		sample["db_xact_commit"] = xactCommit
		sample["db_temp_files"] = tempFiles
		sample["db_temp_bytes"] = tempBytes
	}
	if st := s.repo.db.Stats(); true {
		sample["pool_open"] = st.OpenConnections
		sample["pool_in_use"] = st.InUse
		sample["pool_wait_count"] = st.WaitCount
		sample["pool_wait_ms_total"] = st.WaitDuration.Milliseconds()
	}
	sample["goroutines"] = runtime.NumGoroutine()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	sample["go_heap_inuse_bytes"] = m.HeapInuse
	sample["go_sys_bytes"] = m.Sys
	if rss, ok := procSelfRSSKB(); ok {
		sample["self_rss_kb"] = rss
	}
	if u, s2, ok := procSelfCPUTicks(); ok {
		sample["self_cpu_utime_ticks"] = u
		sample["self_cpu_stime_ticks"] = s2
	}
	if s.cgroup {
		if v, ok := readInt64File("/sys/fs/cgroup/memory.current"); ok {
			sample["cgroup_memory_current_bytes"] = v
		}
		if v, ok := cgroupCPUUsec(ctx, s); ok {
			sample["cgroup_cpu_usage_usec"] = v
		}
	}
	return sample
}

// cgroupCPUUsec reads cumulative CPU for the whole container when readable.
// The sampler cannot run psql against its own cgroup file, so it asks the
// database only when the harness process itself cannot read it.
func cgroupCPUUsec(ctx context.Context, s *loadSampler) (int64, bool) {
	data, err := os.ReadFile("/sys/fs/cgroup/cpu.stat")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if rest, ok := strings.CutPrefix(line, "usage_usec "); ok {
				v, err := strconv.ParseInt(strings.TrimSpace(rest), 10, 64)
				return v, err == nil
			}
		}
	}
	return 0, false
}

func readInt64File(path string) (int64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	return v, err == nil
}

func procSelfRSSKB() (int64, bool) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if rest, ok := strings.CutPrefix(sc.Text(), "VmRSS:"); ok {
			fields := strings.Fields(rest)
			if len(fields) > 0 {
				v, err := strconv.ParseInt(fields[0], 10, 64)
				return v, err == nil
			}
		}
	}
	return 0, false
}

// procSelfCPUTicks returns utime+stime clock ticks for this process
// (USER_HZ is 100 on Linux; the payload keeps raw ticks and the summarizer
// converts using the sample interval).
func procSelfCPUTicks() (int64, int64, bool) {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0, 0, false
	}
	text := string(data)
	// The comm field can contain spaces and parentheses; fields start after
	// the last ')'.
	idx := strings.LastIndex(text, ")")
	if idx < 0 {
		return 0, 0, false
	}
	fields := strings.Fields(text[idx+1:])
	// After comm: field 3 of stat is state, so utime is index 11 and stime 12
	// in this sliced view.
	if len(fields) < 13 {
		return 0, 0, false
	}
	u, err1 := strconv.ParseInt(fields[11], 10, 64)
	s, err2 := strconv.ParseInt(fields[12], 10, 64)
	return u, s, err1 == nil && err2 == nil
}

// ---- concurrency ladder ----

type ladderResult struct {
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
}

// runLadder fires clients goroutines that each issue rounds sequential
// requests behind one barrier, so the concurrency level is real from the
// first request. Wall time spans barrier release to the last completion.
func runLadder(ctx context.Context, name string, clients, rounds int, timeout time.Duration, fn func(ctx context.Context, client, round int) error) ladderResult {
	start := make(chan struct{})
	var wg sync.WaitGroup
	latencies := make([][]float64, clients)
	errCount := map[string]int{}
	var emu sync.Mutex
	wallStart := time.Now()
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			mine := make([]float64, 0, rounds)
			for r := 0; r < rounds; r++ {
				<-start
				rctx, cancel := context.WithTimeout(ctx, timeout)
				t0 := time.Now()
				err := fn(rctx, c, r)
				mine = append(mine, float64(time.Since(t0).Microseconds())/1000)
				cancel()
				if err != nil {
					emu.Lock()
					errCount[errClass(err)]++
					emu.Unlock()
				}
			}
			latencies[c] = mine
		}(c)
	}
	close(start)
	wg.Wait()
	wall := float64(time.Since(wallStart).Microseconds()) / 1000
	all := []float64{}
	completed := 0
	for _, per := range latencies {
		for _, v := range per {
			all = append(all, v)
		}
	}
	errTotal := 0
	for _, n := range errCount {
		errTotal += n
	}
	completed = len(all) - errTotal
	res := ladderResult{Workload: name, Clients: clients, RoundsPerCli: rounds, Completed: completed, Errors: errCount, WallMS: wall}
	if len(all) > 0 && wall > 0 {
		res.ReqPerSec = float64(completed) / (wall / 1000)
		sum := 0.0
		for _, v := range all {
			sum += v
		}
		res.MeanMS = sum / float64(len(all))
		res.P50MS = percentile(all, 50)
		res.P95MS = percentile(all, 95)
		res.P99MS = percentile(all, 99)
		res.MaxMS = percentile(all, 100)
	}
	return res
}

// summarizeSamples condenses one phase's sampler rows: peak values and the
// fraction of samples in which each signal was present. Averages of raw
// counters are meaningless; deltas and presence fractions are not.
func summarizeSamples(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return map[string]any{"samples": 0}
	}
	out := map[string]any{"samples": len(rows)}
	peak := func(key string) {
		max := math.Inf(-1)
		first, last := math.NaN(), math.NaN()
		present := 0
		for i, r := range rows {
			v, ok := numeric(r[key])
			if !ok {
				continue
			}
			present++
			if i == 0 {
				first = v
			}
			last = v
			if v > max {
				max = v
			}
		}
		if present > 0 {
			out["peak_"+key] = max
			if last >= first {
				out["delta_"+key] = last - first
			}
		}
	}
	for _, key := range []string{"ungranted_locks", "pg_backends", "goroutines", "self_rss_kb", "pool_in_use", "pool_open", "db_deadlocks", "db_xact_commit", "db_temp_files", "db_temp_bytes", "pool_wait_count", "pool_wait_ms_total", "cgroup_memory_current_bytes", "cgroup_cpu_usage_usec"} {
		peak(key)
	}
	waitSeen := map[string]int{}
	advisorySamples := 0
	for _, r := range rows {
		list, _ := r["wait_events"].([]string)
		sawAdvisory := false
		for _, w := range list {
			waitSeen[w]++
			if strings.Contains(w, "advisory") {
				sawAdvisory = true
			}
		}
		if sawAdvisory {
			advisorySamples++
		}
	}
	out["wait_event_observations"] = waitSeen
	out["samples_with_advisory_waiters"] = advisorySamples
	return out
}

func numeric(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// TestLiveLoadConcurrencyLadder runs representative catalog read workloads at
// 1, 8, 32 and 64 simultaneous clients against the seeded scale fixture and
// records completed requests, error classes, throughput and latency
// percentiles per level, with database-side contention and process resource
// samples for every phase. The per-request workload is the catalog seam the
// HTTP handlers call (Search with the same Query shapes /v1/works builds).
func TestLiveLoadConcurrencyLadder(t *testing.T) {
	dsn := loadWanted(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	var seeded int
	if err := repo.db.QueryRowContext(ctx, "SELECT count(*) FROM works WHERE id LIKE $1", scalePrefix+"%").Scan(&seeded); err != nil {
		t.Fatal(err)
	}
	if seeded < scaleWorks {
		t.Skipf("scale fixture not seeded (%d works < %d); run TestLiveScaleCatalog first", seeded, scaleWorks)
	}
	deep, err := repo.Search(ctx, Query{Offset: 40000, Limit: 20})
	if err != nil || len(deep.Works) != 20 {
		t.Fatalf("deep seed page: %v (%d works)", err, len(deep.Works))
	}
	deepCursor := CursorAfter(deep.Works)
	transport := "clean"
	if degraded := transportProbeDegraded(ctx, t, repo, deep); degraded > 0 {
		transport = fmt.Sprintf("degraded_%d_of_%d_probe_shapes", degraded, probeShapes)
	}
	label := loadMeasurementPath()
	if transport != "clean" && !strings.Contains(label, "degraded") {
		label = label + "+" + transport
	}

	sampler, err := newLoadSampler(ctx, dsn, repo, 500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer sampler.Close()
	// Warm the pool and the plan caches once; not part of any level.
	warm := runLadder(ctx, "warmup", 4, 2, loadRequestTimeout(), func(cctx context.Context, _, _ int) error {
		_, err := repo.Search(cctx, Query{Offset: 0, Limit: 20})
		return err
	})
	_ = warm
	sampler.snapshot()

	workloads := []struct {
		name   string
		rounds int
		fn     func(cctx context.Context, client, round int) error
	}{
		{"browse_offset_page", loadRoundsPerClient(), func(cctx context.Context, _, _ int) error {
			page, err := repo.Search(cctx, Query{Offset: 0, Limit: 20})
			if err == nil && len(page.Works) != 20 {
				return fmt.Errorf("short page %d", len(page.Works))
			}
			return err
		}},
		{"browse_keyset_deep_page", loadRoundsPerClient(), func(cctx context.Context, _, _ int) error {
			page, err := repo.Search(cctx, Query{After: deepCursor, Limit: 20})
			if err == nil && len(page.Works) != 20 {
				return fmt.Errorf("short page %d", len(page.Works))
			}
			return err
		}},
		{"search_text_contains", loadRoundsPerClient() / 4, func(cctx context.Context, _, _ int) error {
			page, err := repo.Search(cctx, Query{Text: "Scale Volume 42", Limit: 20})
			if err == nil && len(page.Works) == 0 {
				return fmt.Errorf("text search matched nothing")
			}
			return err
		}},
	}
	results := []ladderResult{}
	contention := map[string]any{}
	for _, w := range workloads {
		if w.rounds < 1 {
			w.rounds = 1
		}
		for _, clients := range loadLevels() {
			res := runLadder(ctx, w.name, clients, w.rounds, loadRequestTimeout(), w.fn)
			results = append(results, res)
			contention[w.name+"@clients"+strconv.Itoa(clients)] = summarizeSamples(sampler.snapshot())
			t.Logf("LADDER %s clients=%d completed=%d errors=%v req_per_sec=%.2f p50=%.2f p95=%.2f p99=%.2f max=%.2f wall_ms=%.0f",
				res.Workload, res.Clients, res.Completed, res.Errors, res.ReqPerSec, res.P50MS, res.P95MS, res.P99MS, res.MaxMS, res.WallMS)
		}
	}
	// Attribution probe: same browse workload at the top ladder level with the
	// pool raised to the client count. If throughput moves, the default pool
	// was the limiter at that level; if it does not, the database was.
	levels := loadLevels()
	top := levels[len(levels)-1]
	if _, err := repo.db.ExecContext(ctx, "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	probeRepo, err := NewPostgres(ctx, dsn, WithPool(top, top, time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	defer probeRepo.Close()
	probe := runLadder(ctx, "browse_offset_page_pool_probe", top, loadRoundsPerClient(), loadRequestTimeout(), func(cctx context.Context, _, _ int) error {
		page, err := probeRepo.Search(cctx, Query{Offset: 0, Limit: 20})
		if err == nil && len(page.Works) != 20 {
			return fmt.Errorf("short page %d", len(page.Works))
		}
		return err
	})
	results = append(results, probe)
	contention["browse_offset_page@pool_probe_clients"+strconv.Itoa(top)] = summarizeSamples(sampler.snapshot())
	t.Logf("POOL PROBE clients=%d pool=%d req_per_sec=%.2f p95=%.2f", top, top, probe.ReqPerSec, probe.P95MS)

	emitLoad(t, map[string]any{
		"test":                  "catalog_ladder",
		"measurement_path":      label,
		"transport_probe":       transport,
		"fixture_works":         seeded,
		"pool_max_open_default": defaultPoolSettings().maxOpenConns,
		"request_timeout_s":     int(loadRequestTimeout().Seconds()),
		"results":               results,
		"contention":            contention,
	})
}

// transportProbeDegraded runs the shared six-shape transport probe and returns
// how many shapes disagree with the server's own timing. A deep page supplies
// representative ids.
func transportProbeDegraded(ctx context.Context, t *testing.T, repo *Postgres, deep Page) int {
	t.Helper()
	ids := []string{}
	for _, w := range deep.Works {
		ids = append(ids, w.ID)
	}
	if len(ids) == 0 {
		return 0
	}
	editionIDs, err := scanStrings(ctx, repo, "SELECT id FROM editions WHERE work_id IN ("+placeholders(len(ids))+") LIMIT 200", toArgs(ids)...)
	if err != nil {
		t.Fatalf("probe editions: %v", err)
	}
	probe := transportProbe(ctx, t, repo, editionIDs, ids)
	return probe.degraded
}

// ---- database contention on the write path ----

// contentionWork builds one throwaway work for the lock-contention phases.
// Every work, edition and file id carries the phase tag so concurrent writers
// touch row-disjoint data: any serialization observed cannot be a row lock.
func contentionWork(tag string, i int, now time.Time) Work {
	id := fmt.Sprintf("load-contend-%s-%d", tag, i)
	sha := fmt.Sprintf("%064x", i)
	return Work{
		ID: id, CanonicalTitle: "Contention " + tag + " " + strconv.Itoa(i),
		CreatedAt: now, UpdatedAt: now,
		Editions: []Edition{{
			ID: id + "-ed", Title: "Contention " + tag, Authors: []string{"Load Harness"},
			State: Identified, Confidence: 1, UpdatedAt: now,
			Identifiers: []Identifier{{Scheme: "isbn", Value: "9790000" + strconv.Itoa(i), Source: "load", Confidence: 1}},
			Files: []File{{
				ID: id + "-f", SourceID: "load-contend-src-" + tag + "-" + strconv.Itoa(i),
				Path: "load/" + tag + "/" + strconv.Itoa(i) + ".txt", Format: "txt", MIMEType: "text/plain",
				Size: 8, SHA256: sha, ModifiedAt: now,
			}},
			Evidence: []MetadataEvidence{{Field: "title", Provider: "load", Confidence: 1, ObservedAt: now}},
		}},
	}
}

func deleteContentionWorks(ctx context.Context, t *testing.T, repo *Postgres, tag string) {
	t.Helper()
	if _, err := repo.db.ExecContext(ctx, "DELETE FROM works WHERE id LIKE $1", "load-contend-"+tag+"-%"); err != nil {
		t.Errorf("contention cleanup %s: %v", tag, err)
	}
}

// TestLiveLoadWriteContention determines by measurement, not by reading the
// source, whether the catalog write path serializes concurrent writers.
//
// Four phases over row-disjoint data (works/editions/files all distinct per
// writer, so no row could contend):
//
//	A  sequential MergeWork            — the serial baseline
//	B  concurrent MergeWork (N = 8)    — the production write path
//	C  concurrent same-transaction body with the identity locks omitted
//	     (in-test control)             — isolates locking from the writes
//	D  concurrent ReconcileSource      — the other holder of the source key
//
// If B's aggregate throughput does not improve over A while C's does, some lock
// is still the serializer, and D shows the same signature if it shares it. B
// versus C is the headroom left on the table: C is what the write body costs
// with no exclusion at all, which is not a safe state to ship (see
// TestCatalogMergeNeedleUnlockedLosesWriters).
func TestLiveLoadWriteContention(t *testing.T) {
	dsn := loadWanted(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	sampler, err := newLoadSampler(ctx, dsn, repo, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer sampler.Close()
	concurrency := 8
	opsPerWriter := 24
	if v := os.Getenv("HELIX_LOAD_OPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opsPerWriter = n
		}
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	phase := func(name string, clients, rounds int, fn func(cctx context.Context, client, round int) error) ladderResult {
		res := runLadder(ctx, name, clients, rounds, loadRequestTimeout(), fn)
		summary := summarizeSamples(sampler.snapshot())
		t.Logf("CONTENTION %s completed=%d errors=%v ops_per_sec=%.2f p50_ms=%.2f p95_ms=%.2f advisory_wait_samples=%v peak_ungranted=%v",
			name, res.Completed, res.Errors, res.ReqPerSec, res.P50MS, res.P95MS, summary["samples_with_advisory_waiters"], summary["peak_ungranted_locks"])
		phaseSummaries[name] = map[string]any{"result": res, "contention": summary}
		return res
	}
	phaseSummaries = map[string]any{}

	// A: sequential baseline through the production path.
	aTag := "a" + strconv.FormatInt(now.UnixNano(), 36)
	defer deleteContentionWorks(ctx, t, repo, aTag)
	seqA := phase("A_sequential_mergework", 1, opsPerWriter, func(cctx context.Context, _, r int) error {
		return repo.MergeWork(cctx, contentionWork(aTag, r, now))
	})
	// B: concurrent writers, production path (takes advisory lock per tx).
	bTag := "b" + strconv.FormatInt(now.UnixNano(), 36)
	defer deleteContentionWorks(ctx, t, repo, bTag)
	concB := phase("B_concurrent_mergework", concurrency, opsPerWriter, func(cctx context.Context, c, r int) error {
		return repo.MergeWork(cctx, contentionWork(bTag, c*10000+r, now))
	})
	// C: identical transaction body without the advisory lock, concurrent.
	cTag := "c" + strconv.FormatInt(now.UnixNano(), 36)
	defer deleteContentionWorks(ctx, t, repo, cTag)
	unlocked := func(cctx context.Context, w Work) error {
		tx, err := repo.db.BeginTx(cctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err := upsertTx(cctx, tx, w); err != nil {
			return err
		}
		return tx.Commit()
	}
	concC := phase("C_concurrent_upserttx_without_lock", concurrency, opsPerWriter, func(cctx context.Context, c, r int) error {
		return unlocked(cctx, contentionWork(cTag, c*10000+r, now))
	})
	// D: concurrent reconciles; each source holds disjoint files seeded via C's
	// unlocked path first, so seeding itself does not contaminate the sample.
	dTag := "d" + strconv.FormatInt(now.UnixNano(), 36)
	defer deleteContentionWorks(ctx, t, repo, dTag)
	sources := concurrency
	for c := 0; c < sources; c++ {
		w := contentionWork(dTag, c*10000, now)
		w.Editions[0].Files = nil
		for f := 0; f < 5; f++ {
			file := w.Editions[0].ID + "-f" + strconv.Itoa(f)
			w.Editions[0].Files = append(w.Editions[0].Files, File{ID: file, SourceID: "load-contend-src-" + dTag + "-" + strconv.Itoa(c), Path: "load/" + dTag + "/" + strconv.Itoa(c) + "/" + strconv.Itoa(f) + ".txt", Format: "txt", Size: 1, SHA256: fmt.Sprintf("%064x", c*100+f), ModifiedAt: now})
		}
		if err := unlocked(ctx, w); err != nil {
			t.Fatalf("seed D fixture: %v", err)
		}
	}
	sampler.snapshot()
	concD := phase("D_concurrent_reconcile_source", concurrency, 3, func(cctx context.Context, c, _ int) error {
		_, err := repo.ReconcileSource(cctx, "load-contend-src-"+dTag+"-"+strconv.Itoa(c), []string{"load/" + dTag + "/" + strconv.Itoa(c) + "/0.txt"})
		return err
	})
	seqDTag := "d2" + strconv.FormatInt(now.UnixNano(), 36)
	defer deleteContentionWorks(ctx, t, repo, seqDTag)
	for c := 0; c < sources; c++ {
		w := contentionWork(seqDTag, c*10000, now)
		w.Editions[0].Files = nil
		for f := 0; f < 5; f++ {
			w.Editions[0].Files = append(w.Editions[0].Files, File{ID: w.Editions[0].ID + "-f" + strconv.Itoa(f), SourceID: "load-contend-src-" + seqDTag + "-" + strconv.Itoa(c), Path: "load/" + seqDTag + "/" + strconv.Itoa(c) + "/" + strconv.Itoa(f) + ".txt", Format: "txt", Size: 1, SHA256: fmt.Sprintf("%064x", c*100+f), ModifiedAt: now})
		}
		if err := unlocked(ctx, w); err != nil {
			t.Fatalf("seed D2 fixture: %v", err)
		}
	}
	sampler.snapshot()
	seqD := phase("D2_sequential_reconcile_source", 1, sources*3, func(cctx context.Context, _, r int) error {
		c := r % sources
		_, err := repo.ReconcileSource(cctx, "load-contend-src-"+seqDTag+"-"+strconv.Itoa(c), []string{"load/" + seqDTag + "/" + strconv.Itoa(c) + "/0.txt"})
		return err
	})

	ratioB := concB.ReqPerSec / seqA.ReqPerSec
	ratioC := concC.ReqPerSec / seqA.ReqPerSec
	verdict := "undetermined"
	switch {
	case ratioB >= 1.5 && ratioC >= 1.5:
		verdict = "parallel: sharded identity locks let concurrent disjoint MergeWork scale past the sequential baseline, and the unlocked control confirms the write body itself was never the limit"
	case ratioB < 1.3 && ratioC > 2:
		verdict = "serialized: concurrent MergeWork matches the sequential baseline while the same transaction body without any identity lock scales; a lock is still the serializer"
	default:
		verdict = "inconclusive at this concurrency: neither the lock-attribution ratio nor its negation held"
	}
	emitLoad(t, map[string]any{
		"test":             "catalog_write_contention",
		"measurement_path": loadMeasurementPath(),
		"concurrency":      concurrency,
		"ops_per_writer":   opsPerWriter,
		"phases":           phaseSummaries,
		"mergework_serial_scaling": map[string]any{
			"sequential_ops_per_sec":           seqA.ReqPerSec,
			"locked_concurrent_ops_per_sec":    concB.ReqPerSec,
			"unlocked_concurrent_ops_per_sec":  concC.ReqPerSec,
			"ratio_locked_vs_sequential":       ratioB,
			"ratio_unlocked_vs_sequential":     ratioC,
			"reconcile_concurrent_ops_per_sec": concD.ReqPerSec,
			"reconcile_sequential_ops_per_sec": seqD.ReqPerSec,
		},
		"verdict": verdict,
	})
	t.Logf("VERDICT %s", verdict)
}

var phaseSummaries map[string]any
