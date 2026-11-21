package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Scan jobs are queued, driven, and capped in process memory. What must survive
// a restart is the record an operator reads afterwards: which scan ran, what it
// reported, and — for work the process died in the middle of — that it was
// interrupted rather than silently missing.
const (
	scanKeyPrefix = "helix:scan:job:"
	scanKeyIndex  = "helix:scan:index"
)

// scanJournal is the durable mirror of jobStore. Every method is best-effort
// from the caller's point of view: a scan still runs when the journal is
// unreachable, and the failure is logged rather than turning a working
// ingestion into an error response.
type scanJournal interface {
	save(context.Context, ScanJob) error
	recent(context.Context, int) ([]ScanJob, error)
}

type redisScanJournal struct {
	client *redis.Client
}

func newRedisScanJournal(client *redis.Client) *redisScanJournal {
	return &redisScanJournal{client: client}
}

// save writes one record and refreshes its position in the index. The key TTL
// equals the in-memory retention window so the journal and the store agree on
// how long a finished scan stays visible; an operator never sees a job reappear
// after it had already been pruned.
func (j *redisScanJournal) save(ctx context.Context, job ScanJob) error {
	if job.ID == "" {
		return errors.New("scan journal: record without an ID")
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	score := float64(job.CreatedAt.UTC().UnixNano())
	if err := j.client.Set(ctx, scanKeyPrefix+job.ID, payload, scanRetention).Err(); err != nil {
		return err
	}
	if err := j.client.ZAdd(ctx, scanKeyIndex, redis.Z{Score: score, Member: job.ID}).Err(); err != nil {
		return err
	}
	// The index is bounded so evicted records cannot outlive their keys
	// indefinitely, and stale members left by an expired key are dropped in the
	// same pass that reads them.
	if err := j.client.ZRemRangeByRank(ctx, scanKeyIndex, 0, -(maxRetainedScans + 1)).Err(); err != nil {
		return err
	}
	return nil
}

// recent returns the newest records by creation time. Identifiers the index
// still lists after their key expired, records this server cannot decode, and
// records outside the retention window are dropped from both structures as they
// are encountered, so a damaged or foreign entry cannot persist.
func (j *redisScanJournal) recent(ctx context.Context, limit int) ([]ScanJob, error) {
	if limit <= 0 {
		limit = maxRetainedScans
	}
	ids, err := j.client.ZRevRange(ctx, scanKeyIndex, 0, int64(limit)-1).Result()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]ScanJob, 0, len(ids))
	for _, id := range ids {
		raw, err := j.client.Get(ctx, scanKeyPrefix+id).Result()
		if errors.Is(err, redis.Nil) {
			if err := j.client.ZRem(ctx, scanKeyIndex, id).Err(); err != nil {
				return nil, err
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		var job ScanJob
		if err := json.Unmarshal([]byte(raw), &job); err != nil || job.ID != id {
			if err := j.client.Del(ctx, scanKeyPrefix+id).Err(); err != nil {
				return nil, err
			}
			if err := j.client.ZRem(ctx, scanKeyIndex, id).Err(); err != nil {
				return nil, err
			}
			continue
		}
		// Memory prunes on this same rule, so the journal must not hand back a
		// record the store would have already forgotten.
		if job.FinishedAt != nil && now.Sub(*job.FinishedAt) > scanRetention {
			if err := j.client.Del(ctx, scanKeyPrefix+id).Err(); err != nil {
				return nil, err
			}
			if err := j.client.ZRem(ctx, scanKeyIndex, id).Err(); err != nil {
				return nil, err
			}
			continue
		}
		out = append(out, job)
	}
	return out, nil
}

// UseScanJournal mirrors scan records in Redis so they stay readable across a
// server restart. The caller keeps ownership of the client and closes it. When
// no journal is attached the store is process-local, which is the documented
// behavior for a deployment without Redis.
func (s *Server) UseScanJournal(client *redis.Client) {
	s.jobs.attachJournal(newRedisScanJournal(client))
}

// RestoreScans repopulates scan records at startup. A scan that was queued or
// running when the previous process stopped is reported as failed with an
// interruption reason: its worker is gone, and an honest terminal record beats
// a permanent "running" entry or a missing identifier. Restore never re-runs
// work. Source registrations are still process-local, so a resumed deployment
// re-registers its sources and scans them again explicitly.
func (s *Server) RestoreScans(ctx context.Context) error {
	interrupted, err := s.jobs.restore(ctx)
	if err != nil {
		return err
	}
	if interrupted > 0 {
		slog.Warn("scan records restored with interrupted work", "interrupted", interrupted)
	}
	return nil
}

// restore repopulates the store from the journal and returns the records that
// were still in flight when the previous process stopped. Those become failed
// jobs: their worker no longer exists, and reporting them as interrupted beats
// either a permanent "running" entry or a missing identifier.
func (s *jobStore) restore(ctx context.Context) (int, error) {
	s.mu.RLock()
	journal := s.journal
	s.mu.RUnlock()
	if journal == nil {
		return 0, nil
	}
	jobs, err := journal.recent(ctx, maxRetainedScans)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	restored := 0
	s.mu.Lock()
	for _, job := range jobs {
		if existing := s.jobs[job.ID]; existing != nil {
			continue
		}
		record := job
		record.cancel = nil
		if !terminal(record.State) {
			record.State = "failed"
			record.Error = "scan interrupted by server restart"
			finished := now
			record.FinishedAt = &finished
			record.Report = nil
			restored++
		}
		s.jobs[record.ID] = &record
	}
	s.mu.Unlock()
	return restored, nil
}
