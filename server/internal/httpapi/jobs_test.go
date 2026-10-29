package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/ingest"
)

func waitForJobState(t *testing.T, store *jobStore, id, want string) ScanJob {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := store.get(id)
		if ok && job.State == want {
			return job
		}
		time.Sleep(time.Millisecond)
	}
	job, _ := store.get(id)
	t.Fatalf("job %s did not reach %s: %+v", id, want, job)
	return ScanJob{}
}

func TestJobStoreBoundsConcurrencyAndCancelsQueuedWork(t *testing.T) {
	store := newJobStore()
	var active atomic.Int32
	var maximum atomic.Int32
	started := make(chan string, 3)
	release := make(chan struct{})
	run := func(ctx context.Context) (ingest.Report, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for {
			old := maximum.Load()
			if n <= old || maximum.CompareAndSwap(old, n) {
				break
			}
		}
		started <- "started"
		select {
		case <-release:
			return ingest.Report{Files: 1}, nil
		case <-ctx.Done():
			return ingest.Report{}, ctx.Err()
		}
	}
	var ids []string
	for i := 0; i < 3; i++ {
		job, ctx, err := store.create("source")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, job.ID)
		go store.run(ctx, job.ID, run)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Fatal("initial scans did not start")
		}
	}
	select {
	case <-started:
		t.Fatal("third scan exceeded concurrency bound")
	case <-time.After(30 * time.Millisecond):
	}
	if _, ok := store.cancelJob(ids[2]); !ok {
		t.Fatal("queued job missing")
	}
	close(release)
	for _, id := range ids[:2] {
		job := waitForJobState(t, store, id, "succeeded")
		if job.Report == nil || job.Report.Files != 1 {
			t.Fatalf("missing report: %+v", job)
		}
	}
	waitForJobState(t, store, ids[2], "canceled")
	if got := maximum.Load(); got != maxConcurrentScans {
		t.Fatalf("maximum concurrency=%d, want %d", got, maxConcurrentScans)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := store.shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestJobStoreRecoversPanicAndReportsFailure(t *testing.T) {
	store := newJobStore()
	job, ctx, err := store.create("source")
	if err != nil {
		t.Fatal(err)
	}
	store.run(ctx, job.ID, func(context.Context) (ingest.Report, error) {
		panic("sensitive implementation detail")
	})
	finished := waitForJobState(t, store, job.ID, "failed")
	if finished.Error != "scan worker panic" || finished.FinishedAt == nil {
		t.Fatalf("panic not contained safely: %+v", finished)
	}

	job, ctx, err = store.create("source")
	if err != nil {
		t.Fatal(err)
	}
	store.run(ctx, job.ID, func(context.Context) (ingest.Report, error) {
		return ingest.Report{}, errors.New("read failed")
	})
	finished = waitForJobState(t, store, job.ID, "failed")
	if finished.Error != "read failed" || finished.Report != nil {
		t.Fatalf("failure not recorded: %+v", finished)
	}
}

func TestJobStoreSanitizesScanFailureCredentials(t *testing.T) {
	store := newJobStore()
	job, ctx, err := store.create("source")
	if err != nil {
		t.Fatal(err)
	}
	// A failing connector.List surfaces a *url.Error whose request URL can
	// carry credentials; the admin-visible job error must never retain them.
	store.run(ctx, job.ID, func(context.Context) (ingest.Report, error) {
		return ingest.Report{}, &url.Error{
			Op:  "Get",
			URL: "https://user:s3cr3t-pw@example.com/books?token=abc123#frag",
			Err: errors.New("connection refused"),
		}
	})
	finished := waitForJobState(t, store, job.ID, "failed")
	for _, secret := range []string{"s3cr3t-pw", "abc123", "user@", "#frag", "token="} {
		if strings.Contains(finished.Error, secret) {
			t.Fatalf("job error leaked %q: %q", secret, finished.Error)
		}
	}
	if !strings.Contains(finished.Error, "https://example.com/books") {
		t.Fatalf("job error lost sanitized context: %q", finished.Error)
	}
}

func TestJobStoreQueueCapAndShutdownDrain(t *testing.T) {
	store := newJobStore()
	var workers sync.WaitGroup
	for i := 0; i < maxPendingScans; i++ {
		job, ctx, err := store.create("source")
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		workers.Add(1)
		go func() {
			defer workers.Done()
			store.run(ctx, job.ID, func(ctx context.Context) (ingest.Report, error) {
				<-ctx.Done()
				return ingest.Report{}, ctx.Err()
			})
		}()
	}
	if _, _, err := store.create("source"); !errors.Is(err, errScanQueueFull) {
		t.Fatalf("queue overflow error=%v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := store.shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	workers.Wait()
	if _, _, err := store.create("source"); !errors.Is(err, errScanStoreClosed) {
		t.Fatalf("create after shutdown error=%v", err)
	}
	for _, job := range store.list() {
		if job.State != "canceled" {
			t.Fatalf("job not canceled during shutdown: %+v", job)
		}
	}
}

func TestJobStoreRetentionAndSnapshotIsolation(t *testing.T) {
	store := newJobStore()
	now := time.Now().UTC()
	for i := 0; i < maxRetainedScans; i++ {
		finished := now.Add(-time.Duration(i+1) * time.Minute)
		id := fmt.Sprintf("old-%03d", i)
		store.jobs[id] = &ScanJob{ID: id, State: "succeeded", CreatedAt: finished, FinishedAt: &finished}
	}
	job, ctx, err := store.create("source")
	if err != nil {
		t.Fatal(err)
	}
	if len(store.jobs) != maxRetainedScans {
		t.Fatalf("retained jobs=%d, want %d", len(store.jobs), maxRetainedScans)
	}
	store.run(ctx, job.ID, func(context.Context) (ingest.Report, error) {
		return ingest.Report{Files: 2, Skipped: []string{"original"}}, nil
	})
	first := waitForJobState(t, store, job.ID, "succeeded")
	first.Report.Skipped[0] = "mutated"
	second, _ := store.get(job.ID)
	if second.Report.Skipped[0] != "original" {
		t.Fatal("job snapshot aliases stored report")
	}
}

func TestJobStoreShutdownTimeout(t *testing.T) {
	store := newJobStore()
	job, ctx, err := store.create("source")
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	go store.run(ctx, job.ID, func(context.Context) (ingest.Report, error) {
		<-release // deliberately ignores cancellation to exercise the bound
		return ingest.Report{}, nil
	})
	waitForJobState(t, store, job.ID, "running")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := store.shutdown(shutdownCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown error=%v", err)
	}
	close(release)
	waitForJobState(t, store, job.ID, "canceled")
}
