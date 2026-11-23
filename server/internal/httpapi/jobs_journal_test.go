package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/redis/go-redis/v9"
)

func journalStore(t *testing.T, addr string) (*jobStore, *redis.Client) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { client.Close() })
	store := newJobStore()
	store.attachJournal(newRedisScanJournal(client))
	return store, client
}

// waitFinished runs a scan to completion and returns its terminal record.
func waitFinished(t *testing.T, store *jobStore, sourceID string, report ingest.Report, err error) ScanJob {
	t.Helper()
	job, ctx, failure := store.create(sourceID)
	if failure != nil {
		t.Fatal(failure)
	}
	done := make(chan struct{})
	go func() {
		store.run(ctx, job.ID, func(context.Context) (ingest.Report, error) { return report, err })
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scan did not finish")
	}
	terminal, ok := store.get(job.ID)
	if !ok {
		t.Fatal("terminal job missing from store")
	}
	return terminal
}

// TestJournalSurvivesStoreReconstruction is the HRD-027 durability requirement
// expressed without a process boundary: the records a finished store wrote must
// be readable by a fresh store built against the same Redis.
func TestJournalSurvivesStoreReconstruction(t *testing.T) {
	server := miniredis.RunT(t)
	store, _ := journalStore(t, server.Addr())
	want := waitFinished(t, store, "journal-source", ingest.Report{Files: 3, Works: 4}, nil)
	if want.State != "succeeded" {
		t.Fatalf("state = %q, want succeeded", want.State)
	}

	restarted, _ := journalStore(t, server.Addr())
	if _, err := restarted.restore(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, ok := restarted.get(want.ID)
	if !ok {
		t.Fatal("succeeded scan is not readable after store reconstruction")
	}
	if got.SourceID != want.SourceID || got.State != want.State {
		t.Fatalf("restored job = %+v, want %+v", got, want)
	}
	if got.Report == nil || got.Report.Files != 3 || got.Report.Works != 4 {
		t.Fatalf("restored report = %+v, want files=3 works=4", got.Report)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("restored created_at = %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if len(restarted.list()) != 1 {
		t.Fatalf("restored list = %d jobs, want 1", len(restarted.list()))
	}
}

// TestRestoreMarksInFlightScansInterrupted covers the crash path: a record left
// non-terminal belongs to a worker that no longer exists.
func TestRestoreMarksInFlightScansInterrupted(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	queued := ScanJob{ID: "scan-queued", SourceID: "s", State: "queued", CreatedAt: time.Now().UTC()}
	running := ScanJob{ID: "scan-running", SourceID: "s", State: "running", CreatedAt: time.Now().UTC().Add(time.Second)}
	for _, job := range []ScanJob{queued, running} {
		payload, err := json.Marshal(job)
		if err != nil {
			t.Fatal(err)
		}
		server.Set(scanKeyPrefix+job.ID, string(payload))
		server.ZAdd(scanKeyIndex, float64(job.CreatedAt.UnixNano()), job.ID)
	}

	store, _ := journalStore(t, server.Addr())
	interrupted, err := store.restore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if interrupted != 2 {
		t.Fatalf("interrupted = %d, want 2", interrupted)
	}
	for _, id := range []string{"scan-queued", "scan-running"} {
		job, ok := store.get(id)
		if !ok {
			t.Fatalf("%s not restored", id)
		}
		if job.State != "failed" || job.Error != "scan interrupted by server restart" {
			t.Fatalf("%s = %q/%q, want failed/interrupted", id, job.State, job.Error)
		}
		if job.FinishedAt == nil {
			t.Fatalf("%s has no finish time", id)
		}
	}
	// An interrupted scan must not hold its source busy forever.
	if store.sourceBusy("s") {
		t.Fatal("restored interrupted scan still marks its source busy")
	}
	// The journal must not report the interrupted record as still running either.
	var persisted ScanJob
	raw, err := store.journal.(*redisScanJournal).client.Get(ctx, scanKeyPrefix+"scan-running").Result()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw), &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.State != "running" {
		t.Fatalf("journal record rewritten to %q; restore is read-only over Redis", persisted.State)
	}
}

func TestRestoreSkipsUnreadableAndStaleRecords(t *testing.T) {
	server := miniredis.RunT(t)
	server.Set(scanKeyPrefix+"scan-corrupt", "{not json")
	server.ZAdd(scanKeyIndex, float64(time.Now().UnixNano()), "scan-corrupt")
	expired := time.Now().UTC().Add(-2 * scanRetention)
	stale := ScanJob{ID: "scan-expired", SourceID: "s", State: "succeeded", CreatedAt: expired, FinishedAt: &expired}
	payload, err := json.Marshal(stale)
	if err != nil {
		t.Fatal(err)
	}
	server.Set(scanKeyPrefix+"scan-expired", string(payload))
	server.ZAdd(scanKeyIndex, float64(expired.UnixNano()), "scan-expired")
	// An index member whose key already expired must be dropped, not fatal.
	server.ZAdd(scanKeyIndex, float64(time.Now().UnixNano()), "scan-gone")

	store, _ := journalStore(t, server.Addr())
	if _, err := store.restore(context.Background()); err != nil {
		t.Fatal(err)
	}
	if jobs := store.list(); len(jobs) != 0 {
		t.Fatalf("restore adopted unusable records: %+v", jobs)
	}
	if size, err := store.journal.(*redisScanJournal).client.ZCard(context.Background(), scanKeyIndex).Result(); err != nil {
		t.Fatal(err)
	} else if size != 0 {
		t.Fatalf("index still lists %d unusable records", size)
	}
}

// TestRestorePreservesLiveRecords asserts that startup reconciliation cannot
// overwrite a job this process is actively running.
func TestRestorePreservesLiveRecords(t *testing.T) {
	server := miniredis.RunT(t)
	stale := ScanJob{ID: "scan-live", SourceID: "old", State: "running", CreatedAt: time.Now().UTC()}
	payload, err := json.Marshal(stale)
	if err != nil {
		t.Fatal(err)
	}
	server.Set(scanKeyPrefix+"scan-live", string(payload))
	server.ZAdd(scanKeyIndex, float64(stale.CreatedAt.UnixNano()), "scan-live")

	store, _ := journalStore(t, server.Addr())
	store.mu.Lock()
	store.jobs["scan-live"] = &ScanJob{ID: "scan-live", SourceID: "current", State: "running", CreatedAt: time.Now().UTC()}
	store.mu.Unlock()
	if _, err := store.restore(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, ok := store.get("scan-live")
	if !ok {
		t.Fatal("live record lost")
	}
	if got.SourceID != "current" || got.State != "running" {
		t.Fatalf("restore overwrote a live record: %+v", got)
	}
}

// TestJournalFailureDoesNotFailScans pins the deliberate best-effort boundary:
// Redis being unreachable is not a reason to lose an otherwise healthy scan.
func TestJournalFailureDoesNotFailScans(t *testing.T) {
	server := miniredis.RunT(t)
	// MaxRetries 0 keeps this honest about the same failure a real deployment
	// sees, without spending the test in dial backoff.
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: 0, DialTimeout: 100 * time.Millisecond})
	t.Cleanup(func() { client.Close() })
	store := newJobStore()
	store.attachJournal(newRedisScanJournal(client))
	job, ctx, err := store.create("degraded-source")
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	store.run(ctx, job.ID, func(context.Context) (ingest.Report, error) {
		return ingest.Report{Files: 1}, nil
	})
	got, ok := store.get(job.ID)
	if !ok {
		t.Fatal("job vanished when the journal failed")
	}
	if got.State != "succeeded" || got.Report == nil || got.Report.Files != 1 {
		t.Fatalf("job = %+v, want succeeded with a report", got)
	}
}

// TestJournalIndexIsBounded proves the index cannot grow without limit while
// records keep arriving.
func TestJournalIndexIsBounded(t *testing.T) {
	server := miniredis.RunT(t)
	journal := newRedisScanJournal(redis.NewClient(&redis.Options{Addr: server.Addr()}))
	ctx := context.Background()
	base := time.Now().UTC()
	for i := 0; i < maxRetainedScans+20; i++ {
		job := ScanJob{
			ID:        fmt.Sprintf("scan-%03d", i),
			SourceID:  "s",
			State:     "succeeded",
			CreatedAt: base.Add(time.Duration(i) * time.Millisecond),
		}
		if err := journal.save(ctx, job); err != nil {
			t.Fatal(err)
		}
	}
	recent, err := journal.recent(ctx, maxRetainedScans)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != maxRetainedScans {
		t.Fatalf("journal returned %d records, want %d", len(recent), maxRetainedScans)
	}
	// Newest first, so the list keeps the most recent history on both paths.
	if recent[0].CreatedAt.Before(recent[len(recent)-1].CreatedAt) {
		t.Fatal("journal is not ordered newest first")
	}
}

func TestJournalRejectsRecordWithoutIdentity(t *testing.T) {
	server := miniredis.RunT(t)
	journal := newRedisScanJournal(redis.NewClient(&redis.Options{Addr: server.Addr()}))
	if err := journal.save(context.Background(), ScanJob{}); err == nil {
		t.Fatal("anonymous scan record accepted")
	}
}
