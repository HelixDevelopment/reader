package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/helixreader/helix/server/internal/ingest"
)

const (
	maxConcurrentScans = 2
	maxPendingScans    = 64
	maxRetainedScans   = 256
	scanRetention      = 24 * time.Hour
)

var (
	errScanStoreClosed = errors.New("scan store is shutting down")
	errScanQueueFull   = errors.New("scan queue is full")
)

type ScanJob struct {
	ID         string         `json:"id"`
	SourceID   string         `json:"source_id"`
	State      string         `json:"state"`
	CreatedAt  time.Time      `json:"created_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Report     *ingest.Report `json:"report,omitempty"`
	Error      string         `json:"error,omitempty"`
	cancel     context.CancelFunc
}

type jobStore struct {
	mu     sync.RWMutex
	jobs   map[string]*ScanJob
	sem    chan struct{}
	wg     sync.WaitGroup
	closed bool
}

func newJobStore() *jobStore {
	return &jobStore{jobs: map[string]*ScanJob{}, sem: make(chan struct{}, maxConcurrentScans)}
}

func randomID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(b[:]), nil
}

func terminal(state string) bool {
	return state == "succeeded" || state == "failed" || state == "canceled"
}

// pruneLocked bounds memory without deleting work that is still queued or
// running. Recently finished jobs are retained for at least scanRetention;
// older terminal jobs are removed first, then oldest terminal jobs if the hard
// cap is reached.
func (s *jobStore) pruneLocked(now time.Time) {
	for id, j := range s.jobs {
		if terminal(j.State) && j.FinishedAt != nil && now.Sub(*j.FinishedAt) > scanRetention {
			delete(s.jobs, id)
		}
	}
	for len(s.jobs) >= maxRetainedScans {
		oldestID := ""
		var oldest time.Time
		for id, j := range s.jobs {
			if !terminal(j.State) || j.FinishedAt == nil {
				continue
			}
			if oldestID == "" || j.FinishedAt.Before(oldest) || (j.FinishedAt.Equal(oldest) && id < oldestID) {
				oldestID, oldest = id, *j.FinishedAt
			}
		}
		if oldestID == "" {
			return
		}
		delete(s.jobs, oldestID)
	}
}

func (s *jobStore) create(sourceID string) (ScanJob, context.Context, error) {
	id, err := randomID("scan")
	if err != nil {
		return ScanJob{}, nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now().UTC()
	j := &ScanJob{ID: id, SourceID: sourceID, State: "queued", CreatedAt: now, cancel: cancel}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		cancel()
		return ScanJob{}, nil, errScanStoreClosed
	}
	s.pruneLocked(now)
	pending := 0
	for _, existing := range s.jobs {
		if !terminal(existing.State) {
			pending++
		}
	}
	if pending >= maxPendingScans || len(s.jobs) >= maxRetainedScans {
		cancel()
		return ScanJob{}, nil, errScanQueueFull
	}
	s.jobs[id] = j
	s.wg.Add(1)
	return cloneJob(j), ctx, nil
}

func cloneReport(report *ingest.Report) *ingest.Report {
	if report == nil {
		return nil
	}
	out := *report
	out.Skipped = append([]string(nil), report.Skipped...)
	out.Warnings = append([]string(nil), report.Warnings...)
	return &out
}

func cloneJob(j *ScanJob) ScanJob {
	out := *j
	out.cancel = nil
	out.Report = cloneReport(j.Report)
	if j.FinishedAt != nil {
		finished := *j.FinishedAt
		out.FinishedAt = &finished
	}
	return out
}

func (s *jobStore) finish(id string, report ingest.Report, err error, canceled bool) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.jobs[id]
	if j == nil {
		return
	}
	j.FinishedAt = &now
	j.cancel = nil
	if canceled {
		j.State = "canceled"
		j.Report = nil
		j.Error = ""
		return
	}
	if err != nil {
		j.State = "failed"
		// Scan failures surface connector errors (for example *url.Error from
		// HTTP sources); never persist credentials in the admin-visible record.
		j.Error = ingest.SanitizeError(err)
		j.Report = nil
		return
	}
	j.State = "succeeded"
	j.Error = ""
	j.Report = cloneReport(&report)
}

func executeScan(ctx context.Context, fn func(context.Context) (ingest.Report, error)) (report ingest.Report, err error) {
	defer func() {
		if recover() != nil {
			report = ingest.Report{}
			err = fmt.Errorf("scan worker panic")
		}
	}()
	return fn(ctx)
}

// run owns exactly one WaitGroup count acquired by create. It bounds active
// scans, observes cancellation while queued, contains panics, and records a
// terminal state on every exit path.
func (s *jobStore) run(ctx context.Context, id string, fn func(context.Context) (ingest.Report, error)) {
	defer s.wg.Done()
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		s.finish(id, ingest.Report{}, ctx.Err(), true)
		return
	}
	if ctx.Err() != nil {
		s.finish(id, ingest.Report{}, ctx.Err(), true)
		return
	}
	s.mu.Lock()
	j := s.jobs[id]
	if j == nil || j.State == "cancel_requested" {
		s.mu.Unlock()
		s.finish(id, ingest.Report{}, context.Canceled, true)
		return
	}
	j.State = "running"
	s.mu.Unlock()
	report, err := executeScan(ctx, fn)
	s.finish(id, report, err, ctx.Err() != nil)
}

func (s *jobStore) get(id string) (ScanJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return ScanJob{}, false
	}
	return cloneJob(j), true
}

func (s *jobStore) list() []ScanJob {
	s.mu.Lock()
	s.pruneLocked(time.Now().UTC())
	out := make([]ScanJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, cloneJob(j))
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, k int) bool {
		if !out[i].CreatedAt.Equal(out[k].CreatedAt) {
			return out[i].CreatedAt.After(out[k].CreatedAt)
		}
		return out[i].ID < out[k].ID
	})
	return out
}

func (s *jobStore) sourceBusy(sourceID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, j := range s.jobs {
		if j.SourceID == sourceID && !terminal(j.State) {
			return true
		}
	}
	return false
}

func (s *jobStore) cancelJob(id string) (ScanJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return ScanJob{}, false
	}
	if j.State == "queued" || j.State == "running" {
		j.State = "cancel_requested"
		if j.cancel != nil {
			j.cancel()
		}
	}
	return cloneJob(j), true
}

// shutdown rejects new scans, cancels queued and active work, and drains every
// worker. On timeout the caller must not close connectors still used by scans.
func (s *jobStore) shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		for _, j := range s.jobs {
			if !terminal(j.State) && j.cancel != nil {
				j.cancel()
			}
		}
	}
	s.mu.Unlock()
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
