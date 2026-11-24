package httpapi

// Live slow-client tests for the scan-job surface (server/internal/httpapi/jobs.go).
//
// These tests inject "slow clients" into the real scan-job surface and assert
// the system holds:
//
//   - timeouts are honored (bounded): a slow scan with a context deadline
//     reaches its documented terminal state within the deadline; slow-client
//     job reads are bounded by their per-read budgets;
//   - no deadlock with concurrent fast clients: fast scans and fast reads keep
//     completing while slow clients occupy the job store;
//   - no error storm: fast-client job reads return no errors;
//   - latency for fast clients is unaffected beyond the budget;
//   - any documented backpressure behavior holds: while the two-slot scan
//     semaphore is held by slow scans, the documented admission bound
//     (errScanQueueFull at maxPendingScans, running <= maxConcurrentScans)
//     still fires, and shutdown still rejects new admissions
//     (errScanStoreClosed).
//
// Slow clients: a slow scan occupies one of the two scan slots while a trickle
// of slow job reads (store.get/store.list at intervals) holds the surface.
// Fast clients: fast scans plus fast reads of terminal job records.
//
// Arms:
//
//	Arm A: one slow client + one fast client.
//	Arm B: three slow clients + three fast clients.
//
// Evidence: evidence/campaign-2026-11-20/slow-client/jobs_slow_client_results.json
// records slow-client behavior per arm, fast-client p50/p95/p99 latency, the
// backpressure observations and every assert. Gate: HELIX_LOAD_RUN=1 plus a
// reachable HELIX_TEST_REDIS_ADDR, like TestLiveLoadJobQueueDurability.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/redis/go-redis/v9"
)

// The documented admission bounds, written out independently of the
// production constants (maxPendingScans = 64, maxConcurrentScans = 2): an
// assertion that read the production constant would follow a mutated or
// changed value and could never fail.
const (
	docPendingBound = 64
	docRunningBound = 2
)

const (
	// jobsSlowScanDuration is how long a slow client's scan runs before it
	// completes. It is deliberately slower than every fast-client budget.
	jobsSlowScanDuration = 600 * time.Millisecond
	// jobsTrickleInterval is the spacing between the slow client's job reads.
	jobsTrickleInterval = 20 * time.Millisecond
	// jobsFastReadBudget is the per-read budget for fast clients.
	jobsFastReadBudget = 400 * time.Millisecond
	// jobsFastReads is how many fast reads each fast client performs after its
	// scan is terminal.
	jobsFastReads = 10
	// jobsWhileFullReads is how many fast reads are sampled while the
	// documented admission bound has fired.
	jobsWhileFullReads = 10
	// jobsSlowScanBudget bounds the time a slow scan takes to reach its
	// terminal state (arm B serializes its third slow scan behind the
	// two-slot semaphore, so the budget carries that margin).
	jobsSlowScanBudget = 2 * time.Second
	// jobsArmMargin ends the slow trickle loop this much after the slow scan
	// started, so the loop cannot outlive the scan by much.
	jobsArmMargin = time.Second
	// jobsBurstBudget bounds the admission burst: it must end when the
	// documented queue-full rejection fires or when the budget lapses.
	jobsBurstBudget = 5 * time.Second
	// jobsBurstPollInterval is the sampling interval of the running/pending
	// peak recorder during the arm and the burst.
	jobsBurstPollInterval = 25 * time.Millisecond
	// jobsDeadlineScanBudget is the context deadline of the slow scan with a
	// timeout; its terminal state must arrive by then.
	jobsDeadlineScanBudget = 500 * time.Millisecond
	// jobsDeadlineScanDuration is how long the deadline arm's scan would run if
	// it ignored its context: five times the budget, far above budget+slack, so
	// a scan that does not honor its deadline cannot pass the elapsed check.
	jobsDeadlineScanDuration = 5 * jobsDeadlineScanBudget
	// jobsDeadlineSlackMS is the allowed overshoot of the deadline arm.
	jobsDeadlineSlackMS = 300
	// jobsShutdownBudget bounds the shutdown drain and the admission check
	// after it.
	jobsShutdownBudget = 5 * time.Second
	// jobsArmWatchdog fails the arm when it does not complete in time.
	jobsArmWatchdog = 20 * time.Second
)

type jobsArmSpec struct {
	name string
	slow int
	fast int
}

type jobsSlowScanRecord struct {
	Client           string  `json:"client"`
	State            string  `json:"state"`
	TimeToTerminalMS float64 `json:"time_to_terminal_ms"`
}

type jobsLatencyRecord struct {
	Client    string  `json:"client"`
	Op        string  `json:"op"`
	LatencyMS float64 `json:"latency_ms"`
	ErrClass  string  `json:"error_class,omitempty"`
}

type jobsSlowArmEvidence struct {
	Arm                string               `json:"arm"`
	SlowClients        int                  `json:"slow_clients"`
	FastClients        int                  `json:"fast_clients"`
	SlowScans          []jobsSlowScanRecord `json:"slow_scans"`
	SlowTrickleReads   []jobsLatencyRecord  `json:"slow_trickle_reads"`
	FastScans          []jobsLatencyRecord  `json:"fast_scans"`
	FastReads          []jobsLatencyRecord  `json:"fast_reads"`
	FastReadsWhileFull []jobsLatencyRecord  `json:"fast_reads_while_queue_full"`
	FastReadErrors     int                  `json:"fast_read_errors"`
	FastReadP50MS      float64              `json:"fast_read_p50_ms"`
	FastReadP95MS      float64              `json:"fast_read_p95_ms"`
	FastReadP99MS      float64              `json:"fast_read_p99_ms"`
	WhileFullReadP50MS float64              `json:"while_full_read_p50_ms"`
	WhileFullReadP95MS float64              `json:"while_full_read_p95_ms"`
	WhileFullReadP99MS float64              `json:"while_full_read_p99_ms"`
	Backpressure       map[string]any       `json:"backpressure"`
}

type jobsSlowDeadlineRecord struct {
	State     string  `json:"state"`
	ElapsedMS float64 `json:"elapsed_ms"`
	BudgetMS  float64 `json:"budget_ms"`
}

type jobsShutdownRecord struct {
	ElapsedMS float64 `json:"elapsed_ms"`
	BudgetMS  float64 `json:"budget_ms"`
	ErrClass  string  `json:"error_class,omitempty"`
}

type jobsCreateAfterShutdown struct {
	Rejected bool   `json:"rejected"`
	ErrClass string `json:"error_class,omitempty"`
}

type jobsSlowAssert struct {
	Name     string `json:"name"`
	Violated bool   `json:"violated"`
	Detail   string `json:"detail,omitempty"`
}

type jobsSlowClientEvidence struct {
	Test                string                  `json:"test"`
	Surface             string                  `json:"surface"`
	TestFile            string                  `json:"test_file"`
	EmittedAt           string                  `json:"emitted_at"`
	Parameters          map[string]any          `json:"parameters"`
	Arms                []jobsSlowArmEvidence   `json:"arms"`
	SlowWithDeadline    jobsSlowDeadlineRecord  `json:"slow_scan_with_deadline"`
	Shutdown            jobsShutdownRecord      `json:"shutdown"`
	CreateAfterShutdown jobsCreateAfterShutdown `json:"create_after_shutdown"`
	Asserts             []jobsSlowAssert        `json:"asserts"`
}

func jobsErrClass(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, errScanQueueFull):
		return "scan_queue_full"
	case errors.Is(err, errScanStoreClosed):
		return "scan_store_closed"
	default:
		return fmt.Sprintf("other:%T", err)
	}
}

func jobsLatencyMS(t0 time.Time) float64 {
	return float64(time.Since(t0).Microseconds()) / 1000
}

func jobsLatencyPct(recs []jobsLatencyRecord) (p50, p95, p99 float64) {
	sorted := make([]float64, 0, len(recs))
	for _, r := range recs {
		sorted = append(sorted, r.LatencyMS)
	}
	sort.Float64s(sorted)
	pick := func(p float64) float64 {
		if len(sorted) == 0 {
			return 0
		}
		i := int(p * float64(len(sorted)))
		if i >= len(sorted) {
			i = len(sorted) - 1
		}
		if i < 0 {
			i = 0
		}
		return sorted[i]
	}
	return pick(50), pick(95), pick(99)
}

func jobsSlowScanFn(runCtx context.Context) (ingest.Report, error) {
	select {
	case <-time.After(jobsSlowScanDuration):
		return ingest.Report{Files: 1, Works: 1}, nil
	case <-runCtx.Done():
		return ingest.Report{}, runCtx.Err()
	}
}

func jobsDeadlineScanFn(runCtx context.Context) (ingest.Report, error) {
	select {
	case <-time.After(jobsDeadlineScanDuration):
		return ingest.Report{Files: 1, Works: 1}, nil
	case <-runCtx.Done():
		return ingest.Report{}, runCtx.Err()
	}
}

func jobsFastScanFn(runCtx context.Context) (ingest.Report, error) {
	return ingest.Report{Files: 2, Works: 2}, nil
}

func waitJobsArm(t *testing.T, wg *sync.WaitGroup, name string) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(jobsArmWatchdog):
		t.Fatalf("arm %s did not complete within %v (deadlock?)", name, jobsArmWatchdog)
	}
}

// runJobsSlowArm runs one arm: slow clients (slow scans + trickle reads) and
// fast clients (fast scans + fast reads) plus the documented-backpressure
// burst while the slow scans hold the two-slot semaphore.
func runJobsSlowArm(t *testing.T, store *jobStore, arm jobsArmSpec, recordCreated func(string)) jobsSlowArmEvidence {
	t.Helper()
	ev := jobsSlowArmEvidence{Arm: arm.name, SlowClients: arm.slow, FastClients: arm.fast}
	sourceID := "slow-client-" + arm.name

	var mu sync.Mutex
	var fastErrs int64
	// armCreated tracks every job admitted this arm so the burst jobs can be
	// drained before the next arm; the documented queue bound counts all
	// non-terminal jobs, not per source.
	var armCreated []string
	record := func(id string) {
		armCreated = append(armCreated, id)
		recordCreated(id)
	}
	recordSlowScan := func(r jobsSlowScanRecord) { mu.Lock(); ev.SlowScans = append(ev.SlowScans, r); mu.Unlock() }
	recordSlowRead := func(r jobsLatencyRecord) {
		mu.Lock()
		ev.SlowTrickleReads = append(ev.SlowTrickleReads, r)
		mu.Unlock()
	}
	recordFastScan := func(r jobsLatencyRecord) { mu.Lock(); ev.FastScans = append(ev.FastScans, r); mu.Unlock() }
	recordFastRead := func(r jobsLatencyRecord) { mu.Lock(); ev.FastReads = append(ev.FastReads, r); mu.Unlock() }
	recordWhileFull := func(r jobsLatencyRecord) {
		mu.Lock()
		ev.FastReadsWhileFull = append(ev.FastReadsWhileFull, r)
		mu.Unlock()
	}
	recordErr := func() { atomic.AddInt64(&fastErrs, 1) }

	var wg sync.WaitGroup
	slowJobIDs := []string{}

	// ---- running/pending sampler: runs for the WHOLE arm (a sampler that only
	// ran during the burst could report 0 and pass vacuously) ----
	var maxRunning, maxNonTerminal, samples int64
	updateMax := func(ptr *int64, v int) {
		for {
			cur := atomic.LoadInt64(ptr)
			if int64(v) <= cur || atomic.CompareAndSwapInt64(ptr, cur, int64(v)) {
				return
			}
		}
	}
	pollDone := make(chan struct{})
	stopPoll := make(chan struct{})
	go func() {
		defer close(pollDone)
		ticker := time.NewTicker(jobsBurstPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				running, nonTerminal := 0, 0
				for _, j := range store.list() {
					if terminal(j.State) {
						continue
					}
					nonTerminal++
					if j.State == "running" {
						running++
					}
				}
				updateMax(&maxRunning, running)
				updateMax(&maxNonTerminal, nonTerminal)
				atomic.AddInt64(&samples, 1)
			case <-stopPoll:
				return
			}
		}
	}()

	// ---- slow clients: slow scan + trickle reads while the scan runs ----
	for i := 0; i < arm.slow; i++ {
		job, jobCtx, err := store.create(sourceID)
		if err != nil {
			t.Fatalf("arm %s: create slow client job: %v", arm.name, err)
		}
		record(job.ID)
		slowJobIDs = append(slowJobIDs, job.ID)
		wg.Add(1)
		go func(i int, job ScanJob, jobCtx context.Context) {
			defer wg.Done()
			start := time.Now()
			stopRead := make(chan struct{})
			readDone := make(chan struct{})
			go func() {
				defer close(readDone)
				ticker := time.NewTicker(jobsTrickleInterval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						if time.Since(start) > jobsSlowScanDuration+jobsArmMargin {
							return
						}
						t0 := time.Now()
						_, _ = store.get(job.ID)
						_ = store.list()
						recordSlowRead(jobsLatencyRecord{
							Client:    fmt.Sprintf("slow_reader_%d", i),
							Op:        "get_and_list",
							LatencyMS: jobsLatencyMS(t0),
						})
					case <-stopRead:
						return
					}
				}
			}()
			store.run(jobCtx, job.ID, jobsSlowScanFn)
			close(stopRead)
			<-readDone
			cur, _ := store.get(job.ID)
			recordSlowScan(jobsSlowScanRecord{
				Client: fmt.Sprintf("slow_reader_%d", i), State: cur.State,
				TimeToTerminalMS: jobsLatencyMS(start),
			})
		}(i, job, jobCtx)
	}

	// ---- fast clients: fast scan + fast reads after terminal ----
	for i := 0; i < arm.fast; i++ {
		job, jobCtx, err := store.create(sourceID)
		if err != nil {
			t.Fatalf("arm %s: create fast client job: %v", arm.name, err)
		}
		record(job.ID)
		wg.Add(1)
		go func(i int, job ScanJob, jobCtx context.Context) {
			defer wg.Done()
			t0 := time.Now()
			store.run(jobCtx, job.ID, jobsFastScanFn)
			lat := jobsLatencyMS(t0)
			cur, _ := store.get(job.ID)
			if cur.State != "succeeded" {
				recordErr()
			}
			recordFastScan(jobsLatencyRecord{Client: fmt.Sprintf("fast_scan_%d", i), Op: "run", LatencyMS: lat})
			for r := 0; r < jobsFastReads; r++ {
				t1 := time.Now()
				got, ok := store.get(job.ID)
				_ = store.list()
				lat := jobsLatencyMS(t1)
				if !ok || got.State != "succeeded" {
					recordErr()
				}
				recordFastRead(jobsLatencyRecord{Client: fmt.Sprintf("fast_scan_%d", i), Op: "get_and_list", LatencyMS: lat})
			}
		}(i, job, jobCtx)
	}

	// ---- documented backpressure burst while the slow scans hold the sem ----
	burstStart := time.Now()
	burstAccepted, burstFull, burstOther, nonTermAtRejection := 0, 0, 0, 0
	burstErrClasses := map[string]int{}
	var burstMu sync.Mutex
burstLoop:
	for time.Since(burstStart) < jobsBurstBudget {
		burstJob, _, err := store.create(sourceID)
		switch {
		case err == nil:
			burstAccepted++
			record(burstJob.ID)
		case errors.Is(err, errScanQueueFull):
			burstFull++
			for _, j := range store.list() {
				if !terminal(j.State) {
					nonTermAtRejection++
				}
			}
			burstMu.Lock()
			burstErrClasses["scan_queue_full"]++
			burstMu.Unlock()
			break burstLoop
		case err != nil:
			burstOther++
			burstMu.Lock()
			burstErrClasses[jobsErrClass(err)]++
			burstMu.Unlock()
		}
	}

	// ---- fast reads while the documented queue-full bound has fired ----
	for r := 0; r < jobsWhileFullReads; r++ {
		t0 := time.Now()
		_, _ = store.get(slowJobIDs[0])
		_ = store.list()
		recordWhileFull(jobsLatencyRecord{Client: "fast_reader_while_full", Op: "get_and_list", LatencyMS: jobsLatencyMS(t0)})
	}

	waitJobsArm(t, &wg, arm.name)
	close(stopPoll)
	<-pollDone

	// Drain the admission-bound burst jobs so they do not count against the
	// documented queue bound of the next arm.
	for _, id := range armCreated {
		if job, ok := store.get(id); ok && !terminal(job.State) {
			store.run(context.Background(), id, jobsFastScanFn)
		}
	}

	ev.FastReadErrors = int(atomic.LoadInt64(&fastErrs))
	ev.FastReadP50MS, ev.FastReadP95MS, ev.FastReadP99MS = jobsLatencyPct(ev.FastReads)
	ev.WhileFullReadP50MS, ev.WhileFullReadP95MS, ev.WhileFullReadP99MS = jobsLatencyPct(ev.FastReadsWhileFull)
	ev.Backpressure = map[string]any{
		"queue_full_count":         burstFull,
		"other_errors":             burstOther,
		"error_classes":            burstErrClasses,
		"burst_accepted":           burstAccepted,
		"max_running":              atomic.LoadInt64(&maxRunning),
		"max_nonterminal":          atomic.LoadInt64(&maxNonTerminal),
		"samples":                  atomic.LoadInt64(&samples),
		"nonterminal_at_rejection": nonTermAtRejection,
		"burst_budget_ms":          jobsBurstBudget.Milliseconds(),
		"documented_bounds":        map[string]int{"running": docRunningBound, "pending": docPendingBound},
	}
	return ev
}

func jobsSlowAsserts(ev jobsSlowClientEvidence) []jobsSlowAssert {
	var out []jobsSlowAssert
	add := func(name, detail string) {
		out = append(out, jobsSlowAssert{Name: name, Violated: detail != "", Detail: detail})
	}
	boundedLatency := func(recs []jobsLatencyRecord) string {
		detail := ""
		for _, r := range recs {
			if r.LatencyMS >= float64(jobsFastReadBudget.Milliseconds()) || r.ErrClass != "" {
				detail += fmt.Sprintf(" client=%s op=%s latency_ms=%.0f err=%q;", r.Client, r.Op, r.LatencyMS, r.ErrClass)
			}
		}
		return detail
	}
	for _, arm := range ev.Arms {
		// Slow scans: terminal, succeeded, within the slow-scan budget.
		detail := ""
		for _, s := range arm.SlowScans {
			if s.State != "succeeded" || s.TimeToTerminalMS > float64(jobsSlowScanBudget.Milliseconds()) {
				detail += fmt.Sprintf(" client=%s state=%s time_to_terminal_ms=%.0f;", s.Client, s.State, s.TimeToTerminalMS)
			}
		}
		add(arm.Arm+"_slow_scans_terminal_within_budget", detail)

		// Slow trickle reads: bounded by the per-read budget.
		add(arm.Arm+"_slow_trickle_reads_bounded", boundedLatency(arm.SlowTrickleReads))
		// Fast reads: no error storm, within budget.
		add(arm.Arm+"_fast_read_latency_within_budget", boundedLatency(arm.FastReads))
		// Fast reads while the queue is full: within budget.
		add(arm.Arm+"_fast_reads_unaffected_while_queue_full", boundedLatency(arm.FastReadsWhileFull))
		// Documented backpressure: the queue-full bound fired while the
		// documented running bound was never exceeded.
		bp := arm.Backpressure
		full, _ := bp["queue_full_count"].(int)
		other, _ := bp["other_errors"].(int)
		maxRunning, _ := bp["max_running"].(int64)
		samples, _ := bp["samples"].(int64)
		atRej, _ := bp["nonterminal_at_rejection"].(int)
		detail = ""
		if full < 1 {
			detail += "no queue_full rejection observed;"
		}
		if other != 0 {
			detail += fmt.Sprintf("other_errors=%d;", other)
		}
		// Pending bound: the rejection must fire at exactly docPendingBound
		// non-terminal jobs (tolerance: slow scans that finished between the
		// rejection and the count), NOT at the 256-job retention cap.
		if full >= 1 && (atRej > docPendingBound || atRej < docPendingBound-arm.SlowClients) {
			detail += fmt.Sprintf("queue_full fired at %d non-terminal jobs, want %d (+-%d);", atRej, docPendingBound, arm.SlowClients)
		}
		// Running bound: the sampler must have produced data (control needle:
		// a 0 with no samples proves nothing), the bound must never be exceeded
		// and, with slow clients present, must actually be reached.
		if samples < 5 {
			detail += fmt.Sprintf("only %d running samples (sampler blind);", samples)
		}
		if maxRunning > int64(docRunningBound) {
			detail += fmt.Sprintf("max_running=%d exceeds documented %d;", maxRunning, docRunningBound)
		}
		wantRunning := int64(arm.SlowClients)
		if wantRunning > int64(docRunningBound) {
			wantRunning = int64(docRunningBound)
		}
		if arm.SlowClients > 0 && maxRunning < wantRunning {
			detail += fmt.Sprintf("max_running=%d, want %d (bound never exercised);", maxRunning, wantRunning)
		}
		// Serialization: with more slow scans than slots the last one waits for
		// a slot, so the slowest reaches terminal no earlier than ~2 scan times.
		if arm.SlowClients > docRunningBound {
			slowest := 0.0
			for _, sc := range arm.SlowScans {
				if sc.TimeToTerminalMS > slowest {
					slowest = sc.TimeToTerminalMS
				}
			}
			if slowest < 1.5*float64(jobsSlowScanDuration.Milliseconds()) {
				detail += fmt.Sprintf("slowest slow scan took %.0f ms, want >= %.0f (semaphore not serializing);", slowest, 1.5*float64(jobsSlowScanDuration.Milliseconds()))
			}
		}
		add(arm.Arm+"_documented_backpressure_holds", detail)
	}
	add("slow_scan_with_deadline_timeout_honored", func() string {
		if ev.SlowWithDeadline.State != "canceled" {
			return fmt.Sprintf("state=%q, want canceled", ev.SlowWithDeadline.State)
		}
		if ev.SlowWithDeadline.ElapsedMS > ev.SlowWithDeadline.BudgetMS+jobsDeadlineSlackMS {
			return fmt.Sprintf("elapsed_ms=%.0f exceeds budget_ms=%.0f by slack;", ev.SlowWithDeadline.ElapsedMS, ev.SlowWithDeadline.BudgetMS)
		}
		return ""
	}())
	add("shutdown_bounded", func() string {
		if ev.Shutdown.ErrClass != "" {
			return fmt.Sprintf("err=%q;", ev.Shutdown.ErrClass)
		}
		if ev.Shutdown.ElapsedMS >= float64(jobsShutdownBudget.Milliseconds())+500 {
			return fmt.Sprintf("elapsed_ms=%.0f not bounded by budget_ms=%.0f;", ev.Shutdown.ElapsedMS, ev.Shutdown.BudgetMS)
		}
		return ""
	}())
	add("create_after_shutdown_rejected", func() string {
		if !ev.CreateAfterShutdown.Rejected || ev.CreateAfterShutdown.ErrClass != "scan_store_closed" {
			return fmt.Sprintf("rejected=%v err=%q, want rejected with scan_store_closed;", ev.CreateAfterShutdown.Rejected, ev.CreateAfterShutdown.ErrClass)
		}
		return ""
	}())
	return out
}

// slowClientOutDir returns a fresh directory for this run's evidence. A live
// run never writes into a tracked path (recorded evidence is frozen):
// HELIX_SLOW_CLIENT_OUT names the directory (it must be missing or empty);
// otherwise a fresh os.MkdirTemp directory outside the repository is used.
func slowClientOutDir() (string, error) {
	dir := os.Getenv("HELIX_SLOW_CLIENT_OUT")
	if dir == "" {
		return os.MkdirTemp("", "helix-slow-client-")
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return "", fmt.Errorf("HELIX_SLOW_CLIENT_OUT %s is not empty; refusing to overwrite recorded evidence", dir)
	}
	return dir, os.MkdirAll(dir, 0o755)
}

func writeJobsSlowClientEvidence(t *testing.T, ev jobsSlowClientEvidence) {
	t.Helper()
	data, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		t.Errorf("marshal slow-client evidence: %v", err)
		return
	}
	t.Logf("SLOWCLIENTJSON %s", data)
	dir, err := slowClientOutDir()
	if err != nil {
		t.Errorf("slow-client evidence dir: %v", err)
		return
	}
	path := filepath.Join(dir, "jobs_slow_client_results.json")
	t.Logf("slow-client evidence written to %s", path)
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Errorf("write evidence %s: %v", path, err)
	}
}

// TestLiveSlowClientJobSurface injects slow clients into the scan-job surface
// (server/internal/httpapi/jobs.go) and asserts the documented behavior holds.
func TestLiveSlowClientJobSurface(t *testing.T) {
	if os.Getenv("HELIX_LOAD_RUN") != "1" {
		t.Skip("feature_disabled_by_config: HELIX_LOAD_RUN is not 1; measurement harness opted out")
	}
	addr := os.Getenv("HELIX_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("topology_unsupported: HELIX_TEST_REDIS_ADDR is not configured; a live journal needs Redis")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("HELIX_TEST_REDIS_ADDR=%s unreachable: %v", addr, err)
	}

	store := newJobStore()
	store.attachJournal(newRedisScanJournal(client))

	var createdMu sync.Mutex
	var created []string
	t.Cleanup(func() {
		createdMu.Lock()
		ids := append([]string(nil), created...)
		createdMu.Unlock()
		for _, id := range ids {
			_ = client.ZRem(context.Background(), scanKeyIndex, id).Err()
			_ = client.Del(context.Background(), scanKeyPrefix+id).Err()
		}
	})
	recordCreated := func(id string) {
		createdMu.Lock()
		defer createdMu.Unlock()
		created = append(created, id)
	}

	ev := runJobsSlowClient(t, store, recordCreated, "TestLiveSlowClientJobSurface")
	writeJobsSlowClientEvidence(t, ev)
	for _, a := range ev.Asserts {
		if a.Violated {
			t.Errorf("slow-client assert %s violated: %s", a.Name, a.Detail)
		}
	}
}

// TestSlowClientJobSurfaceHermetic runs the same harness against a real
// in-process jobStore with no journal (no Redis needed). It exercises the
// production jobStore (create/run/shutdown), not a double; only the Redis
// journal leg is absent, so this is a hermetic check of the assertions and of
// the store's documented bounds.
func TestSlowClientJobSurfaceHermetic(t *testing.T) {
	store := newJobStore()
	ev := runJobsSlowClient(t, store, func(string) {}, "TestSlowClientJobSurfaceHermetic")
	for _, a := range ev.Asserts {
		if a.Violated {
			t.Errorf("slow-client assert %s violated: %s", a.Name, a.Detail)
		}
	}
	if len(ev.Asserts) < 10 {
		t.Fatalf("only %d asserts evaluated", len(ev.Asserts))
	}
}

// runJobsSlowClient drives both arms, the deadline arm and the shutdown arm on
// store and returns the evidence with the asserts evaluated.
func runJobsSlowClient(t *testing.T, store *jobStore, recordCreated func(string), testName string) jobsSlowClientEvidence {
	t.Helper()
	ev := jobsSlowClientEvidence{
		Test:      testName,
		Surface:   "scan-job surface (server/internal/httpapi/jobs.go)",
		TestFile:  "server/internal/httpapi/slow_client_live_test.go",
		EmittedAt: time.Now().UTC().Format(time.RFC3339),
		Parameters: map[string]any{
			"slow_scan_duration_ms":    jobsSlowScanDuration.Milliseconds(),
			"slow_trickle_interval_ms": jobsTrickleInterval.Milliseconds(),
			"fast_read_budget_ms":      jobsFastReadBudget.Milliseconds(),
			"fast_reads_per_client":    jobsFastReads,
			"fast_reads_while_full":    jobsWhileFullReads,
			"burst_budget_ms":          jobsBurstBudget.Milliseconds(),
			"burst_poll_interval_ms":   jobsBurstPollInterval.Milliseconds(),
			"arm_margin_ms":            jobsArmMargin.Milliseconds(),
			"deadline_scan_budget_ms":  jobsDeadlineScanBudget.Milliseconds(),
			"shutdown_budget_ms":       jobsShutdownBudget.Milliseconds(),
			"arm_watchdog_s":           jobsArmWatchdog.Seconds(),
			"documented_bounds":        map[string]int{"running": docRunningBound, "pending": docPendingBound},
		},
	}
	for _, arm := range []jobsArmSpec{
		{name: "a_one_slow_one_fast", slow: 1, fast: 1},
		{name: "b_three_slow_plus_fast", slow: 3, fast: 3},
	} {
		ev.Arms = append(ev.Arms, runJobsSlowArm(t, store, arm, recordCreated))
	}

	// A slow client whose context deadline applies: the terminal state must be
	// "canceled" and bounded by the deadline.
	dlJob, _, err := store.create("slow-client-deadline")
	if err != nil {
		t.Fatal(err)
	}
	recordCreated(dlJob.ID)
	dlStart := time.Now()
	dlRunCtx, dlRunCancel := context.WithTimeout(context.Background(), jobsDeadlineScanBudget)
	defer dlRunCancel()
	store.run(dlRunCtx, dlJob.ID, jobsDeadlineScanFn)
	ev.SlowWithDeadline = jobsSlowDeadlineRecord{
		State:     mustGetState(store, dlJob.ID),
		ElapsedMS: jobsLatencyMS(dlStart),
		BudgetMS:  float64(jobsDeadlineScanBudget.Milliseconds()),
	}

	// Shutdown: bounded, then documented admission rejection.
	shutCtx, shutCancel := context.WithTimeout(context.Background(), jobsShutdownBudget)
	defer shutCancel()
	shutdownStart := time.Now()
	shutErr := store.shutdown(shutCtx)
	ev.Shutdown = jobsShutdownRecord{
		ElapsedMS: jobsLatencyMS(shutdownStart),
		BudgetMS:  float64(jobsShutdownBudget.Milliseconds()),
		ErrClass:  jobsErrClass(shutErr),
	}
	_, _, createErr := store.create("slow-client-after-shutdown")
	ev.CreateAfterShutdown = jobsCreateAfterShutdown{
		Rejected: errors.Is(createErr, errScanStoreClosed),
		ErrClass: jobsErrClass(createErr),
	}

	ev.Asserts = jobsSlowAsserts(ev)
	return ev
}

func mustGetState(store *jobStore, id string) string {
	job, ok := store.get(id)
	if !ok {
		return ""
	}
	return job.State
}
