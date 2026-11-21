package httpapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/redis/go-redis/v9"
)

// Durable-queue load harness for ATM-027.
//
// jobs_journal_test.go proves the journal reconciles one job at a time. The
// acceptance clause is about load: a burst of submissions must be bounded, and
// the bound must hold because of the admission check rather than because
// submissions happened to arrive slowly. That needs simultaneous starts, the same
// lesson DEFECT-004 and the delivery ladder each had to learn separately.
//
// Gated on HELIX_LOAD_RUN=1 plus HELIX_TEST_REDIS_ADDR. Against miniredis the
// journal writes never cross a network, so a latency figure taken there would
// describe an in-process library instead of the deployment behaviour claimed.
// The store is always the real jobStore; only its journal is Redis.
//
// Knobs: HELIX_JOBS_BURSTS (default 3), HELIX_JOBS_PER_BURST (default 96, must
// exceed maxPendingScans or the bound is never contested), HELIX_JOBS_SCAN_MS
// (default 20).
//
// Keys are cleaned up individually. Never FLUSHDB: helixqual-redis is shared
// with the KOSync harness running in the same package binary.

func jobsEnvInt(t *testing.T, name string, fallback int) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		t.Fatalf("%s=%q", name, raw)
	}
	return v
}

func jobsPercentile(samples []float64, q float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	i := int(q * float64(len(samples)))
	if i >= len(samples) {
		i = len(samples) - 1
	}
	return samples[i]
}

// countingJournal wraps the Redis journal to measure what the queue costs the
// durable mirror, and to record which states each write carried. A lost terminal
// write is invisible to a later read if the key is simply never rewritten, so
// the per-job write history is what the kept-pace claim rests on.
type countingJournal struct {
	inner   scanJournal
	saves   atomic.Int64
	mu      sync.Mutex
	states  map[string][]string
	latency []float64
}

func (j *countingJournal) save(ctx context.Context, job ScanJob) error {
	start := time.Now()
	err := j.inner.save(ctx, job)
	elapsed := float64(time.Since(start).Microseconds()) / 1000
	if err != nil {
		return err
	}
	j.saves.Add(1)
	j.mu.Lock()
	j.states[job.ID] = append(j.states[job.ID], job.State)
	j.latency = append(j.latency, elapsed)
	j.mu.Unlock()
	return nil
}

func (j *countingJournal) recent(ctx context.Context, limit int) ([]ScanJob, error) {
	return j.inner.recent(ctx, limit)
}

func (j *countingJournal) statesFor(id string) []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.states[id]...)
}

func (j *countingJournal) sortedLatency() []float64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := append([]float64(nil), j.latency...)
	sort.Float64s(out)
	return out
}

type burstRow struct {
	Burst         int     `json:"burst"`
	Submitted     int     `json:"submitted"`
	QueueFull     int     `json:"queue_full_rejections"`
	OtherRejected int     `json:"other_rejections"`
	PeakPending   int64   `json:"peak_pending"`
	SubmissionMS  float64 `json:"submission_wall_ms"`
	DrainMS       float64 `json:"drain_ms"`
	DrainTimedOut bool    `json:"drain_timed_out"`
}

func TestLiveLoadJobQueueDurability(t *testing.T) {
	if os.Getenv("HELIX_LOAD_RUN") != "1" {
		t.Skip("HELIX_LOAD_RUN is not 1; measurement harness opted out")
	}
	addr := os.Getenv("HELIX_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("HELIX_TEST_REDIS_ADDR is not set; durability needs a real Redis")
	}
	bursts := jobsEnvInt(t, "HELIX_JOBS_BURSTS", 3)
	perBurst := jobsEnvInt(t, "HELIX_JOBS_PER_BURST", 96)
	if perBurst <= maxPendingScans {
		t.Fatalf("HELIX_JOBS_PER_BURST=%d does not exceed maxPendingScans=%d: the admission bound is never contested, so this run could not detect its removal", perBurst, maxPendingScans)
	}
	scanDur := time.Duration(jobsEnvInt(t, "HELIX_JOBS_SCAN_MS", 20)) * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("HELIX_TEST_REDIS_ADDR=%s unreachable: %v", addr, err)
	}

	// helixqual-redis is shared and holds unreachable journal keys left by earlier
	// aborted runs (the index trims them, the 24h TTL drops them). Every index
	// figure below is reported against this baseline as well as absolutely.
	indexBefore := client.ZCard(ctx, scanKeyIndex).Val()
	inner := newRedisScanJournal(client)
	journal := &countingJournal{inner: inner, states: map[string][]string{}}
	store := newJobStore()
	store.attachJournal(journal)

	var accepted []string
	var acceptMu sync.Mutex
	var otherRejections []string
	allIDs := []string{}
	t.Cleanup(func() {
		clean := context.Background()
		for _, id := range append([]string(nil), allIDs...) {
			_ = client.Del(clean, scanKeyPrefix+id).Err()
			_ = client.ZRem(clean, scanKeyIndex, id).Err()
		}
	})

	// runningGate brackets the scan body, which run() only reaches while holding
	// a semaphore slot. Its peak is therefore the worker bound, measured.
	var runningGate inflightGate
	runningGate.start = make(chan struct{})
	close(runningGate.start)
	var peakRunning int64

	// --- burst ladder ------------------------------------------------------
	burstRows := []burstRow{}
	for b := 0; b < bursts; b++ {
		gate := &inflightGate{clients: perBurst, start: make(chan struct{})}
		pending := &inflightGate{clients: perBurst, start: make(chan struct{})}
		var wg sync.WaitGroup
		var full int64
		started := time.Now()
		for i := 0; i < perBurst; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-gate.start
				sourceID := "load-src-" + strconv.Itoa(b) + "-" + strconv.Itoa(i)
				job, jobCtx, err := store.create(sourceID)
				switch {
				case err == nil:
					// enter() brackets "accepted and not yet terminal", which is the
					// quantity the admission check counts. exit() runs after run()
					// returns, i.e. after the terminal write, so the bracket can only
					// over-count — never hide a breach.
					pending.enter()
					acceptMu.Lock()
					accepted = append(accepted, job.ID)
					acceptMu.Unlock()
					go func(id string) {
						store.run(jobCtx, id, func(runCtx context.Context) (ingest.Report, error) {
							runningGate.enter()
							defer runningGate.exit()
							if n := runningGate.peak.Load(); n > atomic.LoadInt64(&peakRunning) {
								atomic.StoreInt64(&peakRunning, n)
							}
							select {
							case <-time.After(scanDur):
							case <-runCtx.Done():
							}
							return ingest.Report{Files: 1, Works: 1}, nil
						})
						pending.exit()
					}(job.ID)
				case errors.Is(err, errScanQueueFull):
					atomic.AddInt64(&full, 1)
				default:
					acceptMu.Lock()
					otherRejections = append(otherRejections, sourceID+": "+err.Error())
					acceptMu.Unlock()
				}
			}(i)
		}
		close(gate.start)
		wg.Wait()
		peak := pending.peak.Load()
		row := burstRow{
			Burst: b, Submitted: perBurst,
			QueueFull:    int(atomic.LoadInt64(&full)),
			PeakPending:  peak,
			SubmissionMS: float64(time.Since(started).Microseconds()) / 1000,
		}
		row.OtherRejected = len(otherRejections)
		// Drain before the next burst so bursts contest the bound on their own
		// traffic instead of inheriting a queue that never emptied.
		drainStart := time.Now()
		deadline := drainStart.Add(30 * time.Second)
		for {
			nonTerminal := 0
			for _, j := range store.list() {
				if !terminal(j.State) {
					nonTerminal++
				}
			}
			if nonTerminal == 0 {
				break
			}
			if time.Now().After(deadline) {
				row.DrainTimedOut = true
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
		row.DrainMS = float64(time.Since(drainStart).Microseconds()) / 1000
		// The store flips to terminal before the journal write for that state
		// completes, so a settled store is not a settled queue. Wait for every
		// run() call to return; pending brackets admission through the terminal
		// write, so live == 0 is the point where the journal can be read back.
		liveDeadline := time.Now().Add(10 * time.Second)
		for pending.live.Load() > 0 {
			if time.Now().After(liveDeadline) {
				t.Fatalf("burst %d: %d accepted scans never returned from run() after the store drained", b, pending.live.Load())
			}
			time.Sleep(time.Millisecond)
		}
		burstRows = append(burstRows, row)
		t.Logf("JOBS BURST #%d submitted=%d queue_full=%d peak_pending=%d submit_ms=%.1f drain_ms=%.1f timed_out=%v",
			row.Burst, row.Submitted, row.QueueFull, row.PeakPending, row.SubmissionMS, row.DrainMS, row.DrainTimedOut)

		// The bound is a property of the queue, not of arrival timing.
		if peak > int64(maxPendingScans) {
			t.Fatalf("burst %d held %d non-terminal jobs, above maxPendingScans=%d: admission is not bounded", b, peak, maxPendingScans)
		}
		if b == 0 && row.QueueFull == 0 {
			t.Fatalf("burst 0 submitted %d simultaneously and nothing was rejected: either the bound is unreachable or the barrier released serially, and this harness proves neither", perBurst)
		}
	}
	if len(otherRejections) > 0 {
		t.Fatalf("unexpected non-queue-full rejections: %v", otherRejections)
	}
	if peak := atomic.LoadInt64(&peakRunning); peak > maxConcurrentScans {
		t.Fatalf("%d scans ran concurrently, above maxConcurrentScans=%d: the worker semaphore is not holding", peak, maxConcurrentScans)
	}

	acceptMu.Lock()
	acceptedIDs := append([]string(nil), accepted...)
	acceptMu.Unlock()
	allIDs = append(allIDs, acceptedIDs...)

	// --- every accepted job reached a terminal state -----------------------
	// The store is deliberately bounded: pruneLocked drops terminal records once
	// it holds maxRetainedScans. With enough bursts, an accepted job is therefore
	// legitimately absent — evicted after finishing, not lost. Only the journal's
	// write history distinguishes those two cases, so absence is resolved against
	// it rather than treated as failure.
	stillOpen := []string{}
	prunedAfterFinish := 0
	for _, id := range acceptedIDs {
		job, ok := store.get(id)
		if !ok {
			states := journal.statesFor(id)
			if len(states) > 0 && terminal(states[len(states)-1]) {
				prunedAfterFinish++
				continue
			}
			stillOpen = append(stillOpen, id+" (absent, journal ends at "+strconv.Itoa(len(states))+" writes)")
			continue
		}
		if !terminal(job.State) {
			stillOpen = append(stillOpen, id+" ("+job.State+")")
		}
	}
	if len(stillOpen) > 0 {
		t.Fatalf("%d of %d accepted scans never reached a terminal state: %v", len(stillOpen), len(acceptedIDs), stillOpen[:min(5, len(stillOpen))])
	}

	// Every accepted job must have been written at least three times: queued on
	// admission, running on acquire, terminal on finish.
	shortWrites := []string{}
	for _, id := range acceptedIDs {
		states := journal.statesFor(id)
		if n := len(states); n < 3 {
			final, _ := store.get(id)
			shortWrites = append(shortWrites, fmt.Sprintf("%s journal=%v store=%q", id, states, final.State))
		}
	}
	if len(shortWrites) > 0 {
		t.Fatalf("%d accepted scans were journaled fewer than 3 times (queued/running/terminal); the mirror is not a record of the queue: %v", len(shortWrites), shortWrites[:min(5, len(shortWrites))])
	}

	// --- reconstruction ------------------------------------------------------
	// A fresh store against the same Redis must reproduce the drained queue.
	// Retention bounds the index at maxRetainedScans, so an accepted job can
	// legitimately fall off the index under this write volume; that is measured
	// and reported, not failed on. What must never happen is a job the index
	// still lists coming back absent or in the wrong state.
	fresh := newJobStore()
	fresh.attachJournal(newRedisScanJournal(client))
	rewritten, err := fresh.restore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	indexed := map[string]bool{}
	ids, err := client.ZRevRange(ctx, scanKeyIndex, 0, -1).Result()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		indexed[id] = true
	}
	allIDs = append(allIDs, ids...)
	misrepresented, evicted := []string{}, 0
	for _, id := range acceptedIDs {
		job, ok := fresh.get(id)
		if indexed[id] {
			if !ok {
				misrepresented = append(misrepresented, id+" (indexed but not restored)")
			} else if job.State != "succeeded" {
				misrepresented = append(misrepresented, id+" state="+job.State+" err="+job.Error)
			}
			continue
		}
		if ok {
			misrepresented = append(misrepresented, id+" (restored though not indexed)")
		}
		evicted++
	}
	if len(misrepresented) > 0 {
		t.Fatalf("journal lost or misrepresented %d accepted scans that the index still lists: %v", len(misrepresented), misrepresented[:min(5, len(misrepresented))])
	}
	if rewritten != 0 {
		t.Fatalf("restore reclassified %d drained jobs as interrupted; a completed scan must come back as it finished", rewritten)
	}
	indexLen := client.ZCard(ctx, scanKeyIndex).Val()
	if indexLen > maxRetainedScans {
		t.Fatalf("journal index holds %d members, above maxRetainedScans=%d: eviction is not bounded under write load", indexLen, maxRetainedScans)
	}
	t.Logf("JOBS RECONSTRUCT accepted=%d evicted_by_retention=%d index=%d cap=%d", len(acceptedIDs), evicted, indexLen, maxRetainedScans)

	// --- interrupted work at queue scale ------------------------------------
	// Records left non-terminal in Redis stand in for a process that died
	// mid-scan. The unit test checks this for one job; at scale it checks the
	// reconciliation does not misreport the drained records sitting beside them.
	victims := []string{}
	runToken := strconv.FormatInt(time.Now().UnixNano(), 36)
	for i := 0; i < maxPendingScans; i++ {
		id := "load-interrupted-" + runToken + "-" + strconv.Itoa(i)
		allIDs = append(allIDs, id)
		state := "running"
		if i%3 == 0 {
			state = "queued"
		}
		if err := inner.save(ctx, ScanJob{
			ID: id, SourceID: "load-src-dead", State: state,
			CreatedAt: time.Now().UTC().Add(-time.Duration(i) * time.Millisecond),
		}); err != nil {
			t.Fatal(err)
		}
		victims = append(victims, id)
	}
	restarted := newJobStore()
	restarted.attachJournal(inner)
	interrupted, err := restarted.restore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if interrupted < len(victims) {
		t.Fatalf("restore reported %d interrupted, want at least %d written as in flight", interrupted, len(victims))
	}
	for _, id := range victims {
		job, ok := restarted.get(id)
		if !ok {
			t.Fatalf("interrupted scan %s vanished instead of being reported", id)
		}
		if job.State != "failed" || job.Error == "" {
			t.Fatalf("interrupted scan %s came back as %q (error %q): in-flight work must be reported as interrupted, not left running or silently dropped", id, job.State, job.Error)
		}
	}
	// The drained records must survive the same restore — the ones retention has
	// not displaced. Writing the interrupted records evicts the oldest members,
	// so the reachable set is re-read here rather than reusing the earlier
	// snapshot; assuming it still holds would fail for a reason that is correct
	// behavior.
	recentIDs, err := client.ZRevRange(ctx, scanKeyIndex, 0, -1).Result()
	if err != nil {
		t.Fatal(err)
	}
	reach := map[string]bool{}
	for _, id := range recentIDs {
		reach[id] = true
	}
	displaced := 0
	for _, id := range acceptedIDs {
		if !reach[id] {
			displaced++
			continue
		}
		if job, ok := restarted.get(id); ok {
			if job.State != "succeeded" {
				t.Fatalf("restore rewrote drained scan %s to %q", id, job.State)
			}
		} else {
			t.Fatalf("drained scan %s missing after restore", id)
		}
	}
	t.Logf("JOBS SECOND RESTORE displaced_by_retention=%d reachable=%d", displaced, len(reach))

	// --- journal write cost --------------------------------------------------
	latency := journal.sortedLatency()
	path := os.Getenv("HELIX_MEASUREMENT_PATH")
	if path == "" {
		path = "unspecified"
	}
	httpEmitLoad(t, map[string]any{
		"test":                 "job_queue_durability",
		"measurement_path":     path,
		"redis_addr":           addr,
		"bursts":               bursts,
		"per_burst":            perBurst,
		"scan_ms":              int(scanDur / time.Millisecond),
		"max_concurrent":       maxConcurrentScans,
		"max_pending":          maxPendingScans,
		"max_retained":         maxRetainedScans,
		"peak_running":         atomic.LoadInt64(&peakRunning),
		"accepted_total":       len(acceptedIDs),
		"pruned_after_finish":  prunedAfterFinish,
		"evicted_by_retention": evicted,
		"journal_saves":        journal.saves.Load(),
		"journal_p50_ms":       jobsPercentile(latency, 0.50),
		"journal_p95_ms":       jobsPercentile(latency, 0.95),
		"journal_p99_ms":       jobsPercentile(latency, 0.99),
		"journal_index_before": indexBefore,
		"journal_index_size":   indexLen,
		"interrupted_rows":     len(victims),
		"bursts_detail":        burstRows,
	})
	t.Logf("JOURNAL saves=%d p50=%.2fms p95=%.2fms accepted=%d peak_running=%d",
		journal.saves.Load(), jobsPercentile(latency, 0.50), jobsPercentile(latency, 0.95), len(acceptedIDs), atomic.LoadInt64(&peakRunning))
}
