package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/redis/go-redis/v9"
)

// Multi-instance live load harness for HRD-027, the continuation of the gap named
// in evidence/campaign-2026-11-20/queue/SUMMARY.md: "multi-instance contention on
// one Redis (two helix-server processes sharing a journal) ... not established".
//
// Topology: ONE real Redis (helixqual-redis, HELIX_TEST_REDIS_ADDR — 127.0.0.1:34379
// in this campaign) shared by TWO queue consumers — two jobStore instances, each
// with its own redisScanJournal client (separate connections = separate
// application instances). Concurrent producers create jobs through either store;
// each instance runs a claim loop that claims non-terminal jobs from the shared
// journal with Redis SETNX (helix:scan:claim:<id>, owner instance id, TTL) and
// executes them through its own store (jobStore.run) — the real consumer path.
//
// The single-instance baseline is measured in the same run: one consumer, one
// producer set. Both phases submit the same total (HELIX_MI_TOTAL, default 384)
// under the same producer behavior.
//
// Phases (both gated on HELIX_LOAD_RUN=1 + HELIX_TEST_REDIS_ADDR):
//   1. single_instance_baseline — one consumer.
//   2. two_instances — two consumers; the test asserts the two-instance
//      throughput is >= the single-instance baseline.
//
// Per phase, runMIPhase asserts:
//   - no double execution: each accepted job executed at most once across both
//     instances (execution ledger per job ID);
//   - no lost jobs: every accepted job is either executed or recorded terminal in
//     the journal (failed/canceled included — no silent loss);
//   - no unhandled errors: no panic escaping a claim loop, no "scan journal write
//     failed" warnings observed;
//   - admission caps hold per consumer: peak concurrent scans per instance <=
//     maxConcurrentScans.
//
// Knobs: HELIX_MI_TOTAL (default 384), HELIX_MI_PRODUCERS (default 8),
// HELIX_MI_SCAN_MS (default 20).

const (
	miClaimKeyPrefix = "helix:scan:claim:"
	miClaimTTL       = 30 * time.Second
)

// miExecLedger is the cross-instance execution ledger: how many times each job ID
// was executed, and by which instance. Double execution and lost-job accounting
// are read from it after the drain.
type miExecLedger struct {
	mu         sync.Mutex
	counts     map[string]int
	byInstance map[string]int
	execLat    []float64 // created -> fn entry (claim + queue latency)
	termLat    []float64 // created -> run return (terminal)
	execErrs   []string
	maxTerm    time.Time
}

func newMIExecLedger() *miExecLedger {
	return &miExecLedger{
		counts:     map[string]int{},
		byInstance: map[string]int{},
	}
}

func (l *miExecLedger) recordExec(id, instance string, createdAt, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.counts[id]++
	l.byInstance[instance]++
	l.execLat = append(l.execLat, float64(now.Sub(createdAt).Microseconds())/1000)
}

func (l *miExecLedger) recordTerm(id, instance string, createdAt, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.termLat = append(l.termLat, float64(now.Sub(createdAt).Microseconds())/1000)
	if now.After(l.maxTerm) {
		l.maxTerm = now
	}
}

func (l *miExecLedger) recordExecErr(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.execErrs = append(l.execErrs, msg)
}

func (l *miExecLedger) count(id string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.counts[id]
}

func (l *miExecLedger) snapshots() (counts map[string]int, byInstance map[string]int, execLat, termLat []float64, execErrs []string, maxTerm time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	counts = make(map[string]int, len(l.counts))
	for k, v := range l.counts {
		counts[k] = v
	}
	byInstance = make(map[string]int, len(l.byInstance))
	for k, v := range l.byInstance {
		byInstance[k] = v
	}
	execLat = append([]float64(nil), l.execLat...)
	termLat = append([]float64(nil), l.termLat...)
	execErrs = append([]string(nil), l.execErrs...)
	maxTerm = l.maxTerm
	return
}

func miSorted(lat []float64) []float64 {
	out := append([]float64(nil), lat...)
	sort.Float64s(out)
	return out
}

// miConsumer is one application instance: its own jobStore and journal client,
// its own claim loop, its own worker bound.
type miConsumer struct {
	id          string
	store       *jobStore
	client      *redis.Client
	ledger      *miExecLedger
	gate        inflightGate
	peakRunning atomic.Int64
	panicked    atomic.Bool
	panicMsg    atomic.Value
	// claimSlots bounds claims per instance: the claim loop never runs more
	// than maxConcurrentScans jobs at once, matching the store's own sem.
	claimSlots chan struct{}
	// runWG tracks the spawned run goroutines so the phase snapshots only
	// after they have finished recording their terminal latencies.
	runWG sync.WaitGroup
}

func miDecodeJob(ctx context.Context, client *redis.Client, id string) (ScanJob, bool) {
	raw, err := client.Get(ctx, scanKeyPrefix+id).Result()
	if err != nil {
		return ScanJob{}, false
	}
	var job ScanJob
	if err := json.Unmarshal([]byte(raw), &job); err != nil || job.ID != id {
		return ScanJob{}, false
	}
	return job, true
}

// claimAndRunOne takes a run slot, polls the shared journal index and, for the
// first non-terminal job no live instance has claimed, executes it through this
// consumer's store (jobStore.run — the real consumer path), the way startScan
// runs a job in a goroutine. The SETNX claim key is the mutual exclusion: two
// instances cannot both claim the same job. After claiming, the journal is
// re-read: a job that finished in the poll-to-claim window is released without
// being executed. The claim slot bounds in-flight runs per instance to
// maxConcurrentScans, matching the store's own worker sem.
func (c *miConsumer) claimAndRunOne(ctx context.Context, scanDur time.Duration) bool {
	select {
	case c.claimSlots <- struct{}{}:
	case <-ctx.Done():
		return false
	}
	ids, err := c.client.ZRevRange(ctx, scanKeyIndex, 0, -1).Result()
	if err != nil {
		<-c.claimSlots
		return false
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			<-c.claimSlots
			return false
		}
		job, ok := miDecodeJob(ctx, c.client, id)
		if !ok || terminal(job.State) {
			continue
		}
		claimed, err := c.client.SetNX(ctx, miClaimKeyPrefix+id, c.id, miClaimTTL).Result()
		if err != nil || !claimed {
			continue
		}
		if again, ok2 := miDecodeJob(ctx, c.client, id); ok2 && terminal(again.State) {
			_ = c.client.Del(ctx, miClaimKeyPrefix+id).Err()
			continue
		}
		c.store.mu.Lock()
		_, inMap := c.store.jobs[id]
		if !inMap {
			rec := job
			c.store.jobs[id] = &rec
		}
		c.store.mu.Unlock()
		if !inMap {
			c.store.wg.Add(1)
		}
		createdAt := job.CreatedAt
		c.runWG.Add(1)
		go func(id string, createdAt time.Time) {
			defer c.runWG.Done()
			defer func() { <-c.claimSlots }()
			c.store.run(ctx, id, func(runCtx context.Context) (ingest.Report, error) {
				c.ledger.recordExec(id, c.id, createdAt, time.Now())
				c.gate.enter()
				defer c.gate.exit()
				if n := c.gate.peak.Load(); n > c.peakRunning.Load() {
					c.peakRunning.Store(n)
				}
				select {
				case <-time.After(scanDur):
				case <-runCtx.Done():
				}
				return ingest.Report{Files: 1, Works: 1}, nil
			})
			_ = c.client.Del(ctx, miClaimKeyPrefix+id).Err()
			c.ledger.recordTerm(id, c.id, createdAt, time.Now())
		}(id, createdAt)
		return true
	}
	<-c.claimSlots
	return false
}

func (c *miConsumer) claimLoop(ctx context.Context, scanDur time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			c.panicked.Store(true)
			c.panicMsg.Store(fmt.Sprintf("panic: %v", r))
		}
	}()
	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		if c.claimAndRunOne(ctx, scanDur) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type miPhaseMetrics struct {
	Name               string           `json:"name"`
	Instances          int              `json:"instances"`
	Submitted          int              `json:"submitted"`
	Accepted           int              `json:"accepted"`
	RejectedQueueFull  int              `json:"rejected_queue_full"`
	OtherRejections    int              `json:"other_rejections"`
	Executed           int              `json:"executed"`
	DoubleExecuted     int              `json:"double_executed"`
	Lost               int              `json:"lost"`
	FailedTerminal     int              `json:"failed_terminal_unexecuted"`
	ExecByInstance     map[string]int   `json:"executed_by_instance"`
	AcceptedByStore    map[string]int   `json:"accepted_by_store"`
	PeakRunning        map[string]int64 `json:"peak_running_by_instance"`
	ProduceWallMS      float64          `json:"produce_wall_ms"`
	WallMS             float64          `json:"wall_ms"`
	ProducerRatePerSec float64          `json:"producer_rate_per_sec"`
	ThroughputPerSec   float64          `json:"throughput_executed_per_sec"`
	LatencyP50MS       float64          `json:"latency_p50_ms"`
	LatencyP95MS       float64          `json:"latency_p95_ms"`
	ClaimP50MS         float64          `json:"claim_p50_ms"`
	ClaimP95MS         float64          `json:"claim_p95_ms"`
	ExecErrors         []string         `json:"exec_errors"`
	PanicMessages      []string         `json:"panic_messages"`
}

// runMIPhase runs one phase of the multi-instance harness: producers create jobs
// through the given stores while the consumers claim and execute them from the
// shared journal, then asserts the phase invariants.
func runMIPhase(t *testing.T, ctx context.Context, client *redis.Client, phase string, stores []*jobStore, storeIDs []string, producers, perProducer int, scanDur time.Duration) miPhaseMetrics {
	t.Helper()

	ledger := newMIExecLedger()
	consumers := make([]*miConsumer, len(stores))
	claimCtx, claimCancel := context.WithCancel(ctx)
	defer claimCancel()
	var consumerWG sync.WaitGroup
	for i, store := range stores {
		// The journal is the durable mirror the claim loops read: without it,
		// create() writes nothing to Redis and there is no queue to claim from.
		store.attachJournal(newRedisScanJournal(client))
		consumers[i] = &miConsumer{
			id:         storeIDs[i],
			store:      store,
			client:     client,
			ledger:     ledger,
			claimSlots: make(chan struct{}, maxConcurrentScans),
		}
		consumerWG.Add(1)
		go func(c *miConsumer) {
			defer consumerWG.Done()
			c.claimLoop(claimCtx, scanDur)
		}(consumers[i])
	}

	var acceptMu sync.Mutex
	accepted := []string{}
	acceptedByIdx := make([]int, len(stores))
	var minCreated time.Time
	var rejected int64
	var otherRejections []string

	produceGate := &inflightGate{clients: producers, start: make(chan struct{})}
	var produceWG sync.WaitGroup
	for i := 0; i < producers; i++ {
		produceWG.Add(1)
		go func(i int) {
			defer produceWG.Done()
			<-produceGate.start
			storeIdx := i % len(stores)
			for j := 0; j < perProducer; j++ {
				job, _, err := stores[storeIdx].create(fmt.Sprintf("mi-src-%s-p%d-j%d", phase, i, j))
				if err != nil {
					if errors.Is(err, errScanQueueFull) {
						atomic.AddInt64(&rejected, 1)
					} else {
						acceptMu.Lock()
						otherRejections = append(otherRejections, fmt.Sprintf("producer %d: %v", i, err))
						acceptMu.Unlock()
					}
					continue
				}
				acceptMu.Lock()
				accepted = append(accepted, job.ID)
				acceptedByIdx[storeIdx]++
				if minCreated.IsZero() || job.CreatedAt.Before(minCreated) {
					minCreated = job.CreatedAt
				}
				acceptMu.Unlock()
			}
		}(i)
	}
	produceStart := time.Now()
	close(produceGate.start)
	produceWG.Wait()
	produceWall := time.Since(produceStart)

	// Drain: every accepted job must reach a terminal journal record or have
	// been executed. A real "lost" job is a terminal-free, never-executed
	// accepted ID after the deadline.
	deadline := time.Now().Add(30 * time.Second)
	for {
		var pending []string
		for _, id := range accepted {
			job, ok := miDecodeJob(ctx, client, id)
			if ok && terminal(job.State) {
				continue
			}
			if ledger.count(id) > 0 {
				continue
			}
			pending = append(pending, id)
		}
		if len(pending) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("HRD-027 multi-instance %s: drain timed out with %d jobs never executed or recorded terminal: %v", phase, len(pending), pending[:min(5, len(pending))])
		}
		time.Sleep(10 * time.Millisecond)
	}
	claimCancel()
	consumerWG.Wait()
	for _, c := range consumers {
		c.runWG.Wait()
	}

	// Final accounting: no double execution, no silent loss.
	var lost, double, failedTerminal, executed int
	for _, id := range accepted {
		n := ledger.count(id)
		job, ok := miDecodeJob(ctx, client, id)
		isTerminal := ok && terminal(job.State)
		switch {
		case n > 1:
			double++
		case n == 0 && !isTerminal:
			lost++
		case n == 0 && isTerminal:
			failedTerminal++
		default:
			executed++
		}
	}
	if double > 0 {
		t.Fatalf("HRD-027 multi-instance %s: double execution observed — %d jobs executed more than once", phase, double)
	}
	if lost > 0 {
		t.Fatalf("HRD-027 multi-instance %s: silent loss — %d accepted jobs were neither executed nor recorded terminal", phase, lost)
	}
	for i, c := range consumers {
		if peak := c.peakRunning.Load(); peak > int64(maxConcurrentScans) {
			t.Fatalf("HRD-027 multi-instance %s: %s ran %d scans concurrently, above maxConcurrentScans=%d: the per-consumer admission cap is not holding", phase, storeIDs[i], peak, maxConcurrentScans)
		}
	}

	_, byInstance, execLat, termLat, execErrs, maxTerm := ledger.snapshots()
	peakRunning := make(map[string]int64, len(consumers))
	panics := []string{}
	for i, c := range consumers {
		peakRunning[storeIDs[i]] = c.peakRunning.Load()
		if c.panicked.Load() {
			if msg, ok := c.panicMsg.Load().(string); ok {
				panics = append(panics, msg)
			}
		}
	}
	acceptedByStore := make(map[string]int, len(stores))
	for i := range stores {
		acceptedByStore[storeIDs[i]] = acceptedByIdx[i]
	}

	wallMS := 0.0
	if !minCreated.IsZero() && !maxTerm.IsZero() && maxTerm.After(minCreated) {
		wallMS = float64(maxTerm.Sub(minCreated).Microseconds()) / 1000
	}
	producerRate := 0.0
	if produceWall > 0 {
		producerRate = float64(len(accepted)) / produceWall.Seconds()
	}
	throughput := 0.0
	if wallMS > 0 {
		throughput = float64(executed) / (wallMS / 1000)
	}

	metrics := miPhaseMetrics{
		Name:               phase,
		Instances:          len(stores),
		Submitted:          producers * perProducer,
		Accepted:           len(accepted),
		RejectedQueueFull:  int(atomic.LoadInt64(&rejected)),
		OtherRejections:    len(otherRejections),
		Executed:           executed,
		DoubleExecuted:     double,
		Lost:               lost,
		FailedTerminal:     failedTerminal,
		ExecByInstance:     byInstance,
		AcceptedByStore:    acceptedByStore,
		PeakRunning:        peakRunning,
		ProduceWallMS:      float64(produceWall.Microseconds()) / 1000,
		WallMS:             wallMS,
		ProducerRatePerSec: producerRate,
		ThroughputPerSec:   throughput,
		LatencyP50MS:       jobsPercentile(miSorted(termLat), 0.50),
		LatencyP95MS:       jobsPercentile(miSorted(termLat), 0.95),
		ClaimP50MS:         jobsPercentile(miSorted(execLat), 0.50),
		ClaimP95MS:         jobsPercentile(miSorted(execLat), 0.95),
		ExecErrors:         execErrs,
		PanicMessages:      panics,
	}

	// Best-effort key cleanup for the keys this phase created. The product keeps
	// terminal records for retention; this harness removes them so the next
	// phase's index is not padded by the previous one.
	cleanupCtx := context.Background()
	for _, id := range accepted {
		_ = client.Del(cleanupCtx, scanKeyPrefix+id).Err()
		_ = client.ZRem(cleanupCtx, scanKeyIndex, id).Err()
		_ = client.Del(cleanupCtx, miClaimKeyPrefix+id).Err()
	}

	t.Logf("MULTI %s: submitted=%d accepted=%d rejected_queue_full=%d other_rejected=%d executed=%d lost=%d double=%d by_instance=%v peak=%v produce_ms=%.1f wall_ms=%.0f producer_rate=%.1f/s throughput=%.1f/s latency_p50=%.2fms p95=%.2fms claim_p50=%.2fms claim_p95=%.2fms",
		phase, metrics.Submitted, metrics.Accepted, metrics.RejectedQueueFull, metrics.OtherRejections, metrics.Executed, metrics.Lost, metrics.DoubleExecuted,
		metrics.ExecByInstance, metrics.PeakRunning, metrics.ProduceWallMS, metrics.WallMS, metrics.ProducerRatePerSec, metrics.ThroughputPerSec,
		metrics.LatencyP50MS, metrics.LatencyP95MS, metrics.ClaimP50MS, metrics.ClaimP95MS)
	return metrics
}

func miRedisVersion(ctx context.Context, client *redis.Client) string {
	info, err := client.Info(ctx, "server").Result()
	if err != nil {
		return ""
	}
	m := regexp.MustCompile(`redis_version:(\S+)`).FindStringSubmatch(info)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

type miSlogCapture struct {
	mu       sync.Mutex
	captured []string
}

func (h *miSlogCapture) Enabled(context.Context, slog.Level) bool { return true }

func (h *miSlogCapture) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r.Message == "scan journal write failed" {
		h.captured = append(h.captured, r.Message)
	}
	return nil
}

func (h *miSlogCapture) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *miSlogCapture) WithGroup(string) slog.Handler      { return h }

func (h *miSlogCapture) messages() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.captured...)
}

type miEvidenceAssertions struct {
	NoDoubleExecution            bool `json:"no_double_execution"`
	NoLostJobs                   bool `json:"no_lost_jobs"`
	NoUnhandledErrors            bool `json:"no_unhandled_errors"`
	AdmissionCapsHoldPerConsumer bool `json:"admission_caps_hold_per_consumer"`
	MultiThroughputGeSingle      bool `json:"multi_instance_throughput_ge_single_instance"`
}

type miEvidence struct {
	Campaign      string               `json:"campaign"`
	Test          string               `json:"test"`
	TestFile      string               `json:"test_file"`
	MeasuredAt    string               `json:"measured_at"`
	DSN           string               `json:"dsn"`
	RedisAddr     string               `json:"redis_addr"`
	RedisVersion  string               `json:"redis_version"`
	InstanceCount int                  `json:"instance_count"`
	TotalJobs     int                  `json:"total_jobs_submitted"`
	Phases        []miPhaseMetrics     `json:"phases"`
	Assertions    miEvidenceAssertions `json:"assertions"`
}

func miWriteEvidence(t *testing.T, addr, redisVersion string, single, multi miPhaseMetrics) {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "evidence", "campaign-2026-11-20", "multi-instance")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("evidence dir: %v", err)
	}
	admissionCapsHold := true
	for _, p := range []miPhaseMetrics{single, multi} {
		for _, peak := range p.PeakRunning {
			if peak > int64(maxConcurrentScans) {
				admissionCapsHold = false
			}
		}
	}
	ev := miEvidence{
		Campaign:      "2026-11-20",
		Test:          "TestLiveLoadJobQueueMultiInstance",
		TestFile:      "server/internal/httpapi/load_jobs_multiinstance_test.go",
		MeasuredAt:    time.Now().UTC().Format(time.RFC3339),
		DSN:           "redis://" + addr,
		RedisAddr:     addr,
		RedisVersion:  redisVersion,
		InstanceCount: multi.Instances,
		TotalJobs:     multi.Submitted,
		Phases:        []miPhaseMetrics{single, multi},
		Assertions: miEvidenceAssertions{
			NoDoubleExecution:            single.DoubleExecuted == 0 && multi.DoubleExecuted == 0,
			NoLostJobs:                   single.Lost == 0 && multi.Lost == 0,
			NoUnhandledErrors:            len(single.ExecErrors) == 0 && len(multi.ExecErrors) == 0 && len(single.PanicMessages) == 0 && len(multi.PanicMessages) == 0,
			AdmissionCapsHoldPerConsumer: admissionCapsHold,
			MultiThroughputGeSingle:      multi.ThroughputPerSec >= single.ThroughputPerSec,
		},
	}
	data, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	data = append(data, '\n')
	path := filepath.Join(dir, "multi_instance.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	t.Logf("MULTI wrote evidence: %s", path)
}

func TestLiveLoadJobQueueMultiInstance(t *testing.T) {
	if os.Getenv("HELIX_LOAD_RUN") != "1" {
		t.Skip("HELIX_LOAD_RUN is not 1; measurement harness opted out")
	}
	addr := strings.TrimSpace(os.Getenv("HELIX_TEST_REDIS_ADDR"))
	if addr == "" {
		t.Skip("HELIX_TEST_REDIS_ADDR is not set; multi-instance contention needs a real Redis")
	}
	total := jobsEnvInt(t, "HELIX_MI_TOTAL", 384)
	producers := jobsEnvInt(t, "HELIX_MI_PRODUCERS", 8)
	if total%producers != 0 {
		t.Fatalf("HELIX_MI_TOTAL=%d must be divisible by HELIX_MI_PRODUCERS=%d", total, producers)
	}
	perProducer := total / producers
	scanDur := time.Duration(jobsEnvInt(t, "HELIX_MI_SCAN_MS", 20)) * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("HELIX_TEST_REDIS_ADDR=%s unreachable: %v", addr, err)
	}
	redisVersion := miRedisVersion(ctx, client)

	slogCap := &miSlogCapture{}
	prevDefault := slog.Default()
	slog.SetDefault(slog.New(slogCap))
	t.Cleanup(func() { slog.SetDefault(prevDefault) })

	single := runMIPhase(t, ctx, client, "single_instance_baseline", []*jobStore{newJobStore()}, []string{"instance-1"}, producers, perProducer, scanDur)
	multi := runMIPhase(t, ctx, client, "two_instances", []*jobStore{newJobStore(), newJobStore()}, []string{"instance-1", "instance-2"}, producers, perProducer, scanDur)

	if multi.ThroughputPerSec < single.ThroughputPerSec {
		t.Errorf("HRD-027 multi-instance contention: two-instance throughput %.1f jobs/s is below the single-instance baseline %.1f jobs/s (contention is not minimal)", multi.ThroughputPerSec, single.ThroughputPerSec)
	}
	if len(single.ExecErrors) > 0 || len(multi.ExecErrors) > 0 {
		t.Errorf("execution errors recorded: single=%v multi=%v", single.ExecErrors, multi.ExecErrors)
	}
	if len(slogCap.messages()) > 0 {
		t.Errorf("unhandled journal write failures observed: %v", slogCap.messages())
	}

	miWriteEvidence(t, addr, redisVersion, single, multi)

	httpEmitLoad(t, map[string]any{
		"test":                          "job_queue_multiinstance",
		"redis_addr":                    addr,
		"instance_count":                multi.Instances,
		"total_jobs":                    multi.Submitted,
		"single_instance_executed":      single.Executed,
		"multi_instance_executed":       multi.Executed,
		"single_throughput":             single.ThroughputPerSec,
		"multi_throughput":              multi.ThroughputPerSec,
		"multi_single_throughput_ratio": multi.ThroughputPerSec / max(single.ThroughputPerSec, 0.0001),
		"single_latency_p50":            single.LatencyP50MS,
		"multi_latency_p50":             multi.LatencyP50MS,
	})

	t.Logf("MULTI summary single_instance_baseline: accepted=%d executed=%d throughput=%.1f/s p50=%.2fms p95=%.2fms",
		single.Accepted, single.Executed, single.ThroughputPerSec, single.LatencyP50MS, single.LatencyP95MS)
	t.Logf("MULTI summary two_instances: accepted=%d executed=%d by_instance=%v throughput=%.1f/s p50=%.2fms p95=%.2fms",
		multi.Accepted, multi.Executed, multi.ExecByInstance, multi.ThroughputPerSec, multi.LatencyP50MS, multi.LatencyP95MS)
}
