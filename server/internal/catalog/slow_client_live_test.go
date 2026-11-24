package catalog

// Live slow-client tests for the catalog read/write surface
// (server/internal/catalog/postgres.go).
//
// These tests inject "slow clients" into the real PostgreSQL surface and
// assert the system holds:
//
//   - timeouts are honored (bounded): every slow-client operation returns
//     within its context deadline plus a slack window;
//   - no deadlock with concurrent fast clients: fast readers and writers keep
//     completing while slow clients hold a connection or the identity advisory
//     lock;
//   - no error storm: fast-client operations return no errors;
//   - latency for fast clients is unaffected beyond the budget;
//   - the documented pool backpressure holds: slow reads/writes hold their
//     connections without exhausting the production pool caps.
//
// Injection: a pgx stdlib driver wrapper (slowConn) returns slow connections
// to a *sql.DB wrapped by the in-package Postgres{db} seam. Slow reads stall
// inside driver.QueryContext before the query is sent (the connection is held
// while stalled — a slow-read connection holding catalog reads); slow writes
// stall inside driver.ExecContext of the first INSERT, which is after
// lockIdentities has taken the transaction-scoped advisory locks, so a real
// MergeWork on a shared work key collides with fast writers on that key while
// fast readers (plain SELECTs) must not wait on the advisory lock.
//
// Arms:
//
//	Arm A: one slow reader + one fast reader.
//	Arm B: three slow readers + three fast readers.
//
// Each arm then runs the write phase: one slow writer on a shared work key,
// fast writers on the same key, and fast readers.
//
// Evidence: evidence/campaign-2026-11-20/slow-client/catalog_slow_client_results.json
// records slow-client behavior per arm (error classes and timeouts), fast
// reader/writer p50/p95/p99 latency and every assert. Gate:
// HELIX_TEST_POSTGRES_DSN, like the other live catalog tests.

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	// slowClientStall is how long a slow client's read/write is held before the
	// ctx deadline fires. It is deliberately longer than every deadline.
	slowClientStall = 900 * time.Millisecond
	// slowClientDeadline is the per-op context deadline of a slow client. The
	// slow stall is more than twice this, so every slow op must time out.
	slowClientDeadline = 400 * time.Millisecond
	// slowDeadlineSlack is how far past the deadline an op may return before the
	// assert flags the timeout as not honored.
	slowDeadlineSlack = 250 * time.Millisecond
	// fastClientBudget is the per-op budget for fast clients: their ops are
	// milliseconds, so anything near this is already degraded.
	fastClientBudget = 2000 * time.Millisecond
	// fastRoundsPerClient bounds per-client work in each phase so the total
	// test stays in a couple of seconds.
	fastRoundsPerClient = 8
	// fastWriterRounds bounds fast writer rounds per fast writer.
	fastWriterRounds = 3
	// armBudget is the watchdog: every arm must finish well inside it.
	armBudget = 15 * time.Second
)

// ---- slow-read driver injection -------------------------------------------

// slowSpec carries the injectable slowness of the slow DB.
type slowSpec struct {
	stall        time.Duration
	readStall    atomic.Bool  // stall reads (read phase)
	writeStall   atomic.Bool  // stall the first write exec (write phase)
	writeStalled atomic.Bool  // set when a write stall begins (test signal)
	stallHits    atomic.Int64 // number of stalls that fired
}

// stallInterruptibly waits the stall, returning early with the context error
// when the deadline fires first.
func (s *slowSpec) stallInterruptibly(ctx context.Context) error {
	select {
	case <-time.After(s.stall):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type slowConnector struct {
	driver.Connector
	spec *slowSpec
}

func (c *slowConnector) Connect(ctx context.Context) (driver.Conn, error) {
	inner, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &slowConn{Conn: inner, spec: c.spec}, nil
}

type slowConn struct {
	driver.Conn
	spec *slowSpec
}

// QueryContext stalls the read while readStall is armed. The stall happens
// before the driver is called, so the connection is held while the read is
// stalled and the deadline returns ctx.Err() without sending the query.
func (c *slowConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.spec.readStall.Load() {
		if err := c.spec.stallInterruptibly(ctx); err != nil {
			c.spec.stallHits.Add(1)
			return nil, err
		}
	}
	qc, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, fmt.Errorf("slowConn: inner connection does not implement driver.QueryerContext")
	}
	return qc.QueryContext(ctx, query, args)
}

// ExecContext stalls the first INSERT while writeStall is armed. The advisory
// identity locks are already held (lockIdentities runs before the write body),
// so the slow writer is slow while holding the catalog write lock.
func (c *slowConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.spec.writeStall.Load() && strings.Contains(query, "INSERT INTO") {
		c.spec.stallHits.Add(1)
		c.spec.writeStalled.Store(true)
		if err := c.spec.stallInterruptibly(ctx); err != nil {
			return nil, err
		}
	}
	ec, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, fmt.Errorf("slowConn: inner connection does not implement driver.ExecerContext")
	}
	return ec.ExecContext(ctx, query, args)
}

// slowCatalogDB builds a Postgres whose db runs on slow driver connections.
func slowCatalogDB(t *testing.T, dsn string, spec *slowSpec) *Postgres {
	t.Helper()
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("slow-client db: parse config: %v", err)
	}
	db := sql.OpenDB(&slowConnector{Connector: stdlib.GetConnector(*cfg), spec: spec})
	settings := defaultPoolSettings()
	db.SetMaxOpenConns(settings.maxOpenConns)
	db.SetMaxIdleConns(settings.maxIdleConns)
	db.SetConnMaxLifetime(settings.connMaxLifetime)
	return &Postgres{db: db}
}

// ---- evidence types --------------------------------------------------------

type catalogSlowClientRecord struct {
	Arm       string  `json:"arm"`
	Client    string  `json:"client"`
	Op        string  `json:"op"`
	ElapsedMS float64 `json:"elapsed_ms"`
	ErrClass  string  `json:"error_class,omitempty"`
	Note      string  `json:"note,omitempty"`
}

type catalogSlowLatency struct {
	Arm       string  `json:"arm"`
	Client    string  `json:"client"`
	Op        string  `json:"op"`
	LatencyMS float64 `json:"latency_ms"`
	ErrClass  string  `json:"error_class,omitempty"`
}

type catalogSlowPhaseEvidence struct {
	Slow          []catalogSlowClientRecord `json:"slow"`
	FastReaders   []catalogSlowLatency      `json:"fast_readers"`
	FastWriters   []catalogSlowLatency      `json:"fast_writers,omitempty"`
	FastErrors    int                       `json:"fast_errors"`
	FastReaderP50 float64                   `json:"fast_reader_p50_ms"`
	FastReaderP95 float64                   `json:"fast_reader_p95_ms"`
	FastReaderP99 float64                   `json:"fast_reader_p99_ms"`
	FastWriterP50 float64                   `json:"fast_writer_p50_ms,omitempty"`
	FastWriterP95 float64                   `json:"fast_writer_p95_ms,omitempty"`
	FastWriterP99 float64                   `json:"fast_writer_p99_ms,omitempty"`
	TotalOps      int                       `json:"total_ops"`
	PhaseBudgetMS float64                   `json:"phase_budget_ms"`
}

type catalogSlowArmEvidence struct {
	Arm         string                   `json:"arm"`
	SlowClients int                      `json:"slow_clients"`
	FastClients int                      `json:"fast_clients"`
	ReadPhase   catalogSlowPhaseEvidence `json:"read_phase"`
	WritePhase  catalogSlowPhaseEvidence `json:"write_phase"`
}

type catalogSlowAssert struct {
	Name     string `json:"name"`
	Violated bool   `json:"violated"`
	Detail   string `json:"detail,omitempty"`
}

type catalogSlowClientEvidence struct {
	Test            string                   `json:"test"`
	Surface         string                   `json:"surface"`
	TestFile        string                   `json:"test_file"`
	MeasurementPath string                   `json:"measurement_path"`
	EmittedAt       string                   `json:"emitted_at"`
	Parameters      map[string]any           `json:"parameters"`
	Arms            []catalogSlowArmEvidence `json:"arms"`
	Asserts         []catalogSlowAssert      `json:"asserts"`
}

// slowClientEvidenceRoot is the repository root: the catalog test file lives
// at server/internal/catalog, four levels below it, so evidence always lands
// in the repo's evidence/ tree regardless of the test working directory.
func slowClientEvidenceRoot(t *testing.T) string {
	t.Helper()
	if _, thisFile, _, ok := runtime.Caller(0); ok {
		root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
		if _, err := os.Stat(filepath.Join(root, "evidence")); err == nil {
			return root
		}
	}
	return "."
}

func writeCatalogSlowClientEvidence(t *testing.T, ev catalogSlowClientEvidence) {
	t.Helper()
	data, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		t.Errorf("marshal slow-client evidence: %v", err)
		return
	}
	t.Logf("SLOWCLIENTJSON %s", data)
	path := filepath.Join(slowClientEvidenceRoot(t), "evidence", "campaign-2026-11-20", "slow-client", "catalog_slow_client_results.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Errorf("mkdir evidence: %v", err)
		return
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Errorf("write evidence %s: %v", path, err)
	}
}

func catalogSlowErrClass(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, sql.ErrNoRows):
		return "sql_err_no_rows"
	default:
		return fmt.Sprintf("other:%T", err)
	}
}

func catalogSlowMS(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}

// slowClientWork builds a throwaway work whose advisory identity keys are
// shared by every writer of the same tag, so writers of the same work collide.
func slowClientWork(tag string, now time.Time) Work {
	id := "slow-client-" + tag
	return Work{
		ID:             id,
		CanonicalTitle: "Slow Client " + tag,
		CreatedAt:      now,
		UpdatedAt:      now,
		Editions: []Edition{{
			ID:         id + "-ed",
			Title:      "Slow Client " + tag,
			Authors:    []string{"Slow Client"},
			State:      Identified,
			Confidence: 1,
			UpdatedAt:  now,
			Files: []File{{
				ID:         id + "-f",
				SourceID:   "slow-client-src-" + tag,
				Path:       "slow/" + tag + "/book.txt",
				Format:     "txt",
				MIMEType:   "text/plain",
				Size:       8,
				SHA256:     strings.Repeat("f", 64),
				ModifiedAt: now,
			}},
		}},
	}
}

// waitCatalogSlowArm fails the test when the arm does not finish within the
// watchdog: every arm must make progress; nothing may deadlock.
func waitCatalogSlowArm(t *testing.T, wg *sync.WaitGroup, name string) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(armBudget):
		t.Fatalf("arm %s did not complete within %v (deadlock?)", name, armBudget)
	}
}

// catalogSlowLatencyPct returns p50/p95/p99 of the latency records' latency_ms.
func catalogSlowLatencyPct(recs []catalogSlowLatency) (p50, p95, p99 float64) {
	values := make([]float64, 0, len(recs))
	for _, r := range recs {
		values = append(values, r.LatencyMS)
	}
	return percentile(values, 50), percentile(values, 95), percentile(values, 99)
}

// runCatalogSlowReadPhase: slow readers (slowRepo.GetWork with a deadline)
// plus fast readers (repo.GetWork + repo.Search with generous budgets) run
// concurrently. Slow clients must return within their deadline; fast clients
// must return with no errors and in-budget latency.
func runCatalogSlowReadPhase(t *testing.T, ctx context.Context, repo *Postgres, slowRepo *Postgres, arm string, seedID string, nSlow, nFast int) catalogSlowPhaseEvidence {
	t.Helper()
	var mu sync.Mutex
	var slowRecs []catalogSlowClientRecord
	var fastRecs []catalogSlowLatency
	var fastErrors int64

	var wg sync.WaitGroup
	for i := 0; i < nSlow; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rctx, cancel := context.WithTimeout(ctx, slowClientDeadline)
			defer cancel()
			start := time.Now()
			_, err := slowRepo.GetWork(rctx, seedID)
			elapsed := time.Since(start)
			mu.Lock()
			slowRecs = append(slowRecs, catalogSlowClientRecord{
				Arm: arm, Client: fmt.Sprintf("slow_reader_%d", i), Op: "GetWork",
				ElapsedMS: catalogSlowMS(elapsed), ErrClass: catalogSlowErrClass(err),
			})
			mu.Unlock()
		}(i)
	}
	for i := 0; i < nFast; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for round := 0; round < fastRoundsPerClient; round++ {
				fctx, cancel := context.WithTimeout(ctx, fastClientBudget)
				start := time.Now()
				w, err := repo.GetWork(fctx, seedID)
				if err == nil && w.ID != seedID {
					err = fmt.Errorf("GetWork returned work %q, want %q", w.ID, seedID)
				}
				elapsed := time.Since(start)
				if err != nil {
					atomic.AddInt64(&fastErrors, 1)
				}
				mu.Lock()
				fastRecs = append(fastRecs, catalogSlowLatency{
					Arm: arm, Client: fmt.Sprintf("fast_reader_%d", i), Op: "GetWork",
					LatencyMS: catalogSlowMS(elapsed), ErrClass: catalogSlowErrClass(err),
				})
				mu.Unlock()
				cancel()

				sctx, scancel := context.WithTimeout(ctx, fastClientBudget)
				start = time.Now()
				page, err := repo.Search(sctx, Query{Limit: 20})
				elapsed = time.Since(start)
				if err != nil || len(page.Works) == 0 {
					atomic.AddInt64(&fastErrors, 1)
					if err == nil {
						err = fmt.Errorf("Search returned %d works, want >0", len(page.Works))
					}
				}
				mu.Lock()
				fastRecs = append(fastRecs, catalogSlowLatency{
					Arm: arm, Client: fmt.Sprintf("fast_reader_%d", i), Op: "Search",
					LatencyMS: catalogSlowMS(elapsed), ErrClass: catalogSlowErrClass(err),
				})
				mu.Unlock()
				scancel()
			}
		}(i)
	}
	waitCatalogSlowArm(t, &wg, arm+"_read")

	p50, p95, p99 := catalogSlowLatencyPct(fastRecs)
	return catalogSlowPhaseEvidence{
		Slow:          slowRecs,
		FastReaders:   fastRecs,
		FastErrors:    int(atomic.LoadInt64(&fastErrors)),
		FastReaderP50: p50, FastReaderP95: p95, FastReaderP99: p99,
		TotalOps:      len(slowRecs) + len(fastRecs),
		PhaseBudgetMS: float64(armBudget.Milliseconds()),
	}
}

// runCatalogSlowWritePhase: one slow writer (real MergeWork through the slow
// DB — the stall fires inside the first INSERT, after the advisory locks are
// held) plus fast writers on the same work key plus fast readers. Fast writers
// must be released when the slow writer's deadline rolls the transaction back,
// and fast readers (plain SELECTs) must not wait on the advisory lock.
func runCatalogSlowWritePhase(t *testing.T, ctx context.Context, repo *Postgres, slowRepo *Postgres, spec *slowSpec, arm string, nFast int, work Work, seedID string) catalogSlowPhaseEvidence {
	t.Helper()
	var mu sync.Mutex
	var slowRecs []catalogSlowClientRecord
	var fastWriterRecs []catalogSlowLatency
	var fastReaderRecs []catalogSlowLatency
	var fastErrors int64

	var wg sync.WaitGroup
	spec.writeStall.Store(true)

	wg.Add(1)
	go func() {
		defer wg.Done()
		wctx, cancel := context.WithTimeout(ctx, slowClientDeadline)
		defer cancel()
		start := time.Now()
		err := slowRepo.MergeWork(wctx, work)
		elapsed := time.Since(start)
		mu.Lock()
		slowRecs = append(slowRecs, catalogSlowClientRecord{
			Arm: arm, Client: "slow_writer", Op: "MergeWork",
			ElapsedMS: catalogSlowMS(elapsed), ErrClass: catalogSlowErrClass(err),
		})
		mu.Unlock()
	}()

	// Wait until the slow writer is stalled inside its INSERT, i.e. the advisory
	// identity locks are held and the write transaction is open.
	deadline := time.Now().Add(2 * time.Second)
	for !spec.writeStalled.Load() {
		if time.Now().After(deadline) {
			spec.writeStall.Store(false)
			t.Fatalf("slow writer %s never reached its stalled INSERT", arm)
		}
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond) // locks settle before fast writers start

	for i := 0; i < nFast; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for round := 0; round < fastWriterRounds; round++ {
				fctx, cancel := context.WithTimeout(ctx, fastClientBudget)
				start := time.Now()
				err := repo.MergeWork(fctx, work)
				elapsed := time.Since(start)
				if err != nil {
					atomic.AddInt64(&fastErrors, 1)
				}
				mu.Lock()
				fastWriterRecs = append(fastWriterRecs, catalogSlowLatency{
					Arm: arm, Client: fmt.Sprintf("fast_writer_%d", i), Op: "MergeWork",
					LatencyMS: catalogSlowMS(elapsed), ErrClass: catalogSlowErrClass(err),
				})
				mu.Unlock()
				cancel()
			}
		}(i)
	}
	for i := 0; i < nFast; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for round := 0; round < fastRoundsPerClient; round++ {
				fctx, cancel := context.WithTimeout(ctx, fastClientBudget)
				start := time.Now()
				w, err := repo.GetWork(fctx, seedID)
				if err == nil && w.ID != seedID {
					err = fmt.Errorf("GetWork returned work %q, want %q", w.ID, seedID)
				}
				elapsed := time.Since(start)
				if err != nil {
					atomic.AddInt64(&fastErrors, 1)
				}
				mu.Lock()
				fastReaderRecs = append(fastReaderRecs, catalogSlowLatency{
					Arm: arm, Client: fmt.Sprintf("fast_reader_%d", i), Op: "GetWork",
					LatencyMS: catalogSlowMS(elapsed), ErrClass: catalogSlowErrClass(err),
				})
				mu.Unlock()
				cancel()
			}
		}(i)
	}
	waitCatalogSlowArm(t, &wg, arm+"_write")
	spec.writeStall.Store(false)

	wp50, wp95, wp99 := catalogSlowLatencyPct(fastWriterRecs)
	rp50, rp95, rp99 := catalogSlowLatencyPct(fastReaderRecs)
	return catalogSlowPhaseEvidence{
		Slow:          slowRecs,
		FastWriters:   fastWriterRecs,
		FastReaders:   fastReaderRecs,
		FastErrors:    int(atomic.LoadInt64(&fastErrors)),
		FastWriterP50: wp50, FastWriterP95: wp95, FastWriterP99: wp99,
		FastReaderP50: rp50, FastReaderP95: rp95, FastReaderP99: rp99,
		TotalOps:      len(slowRecs) + len(fastWriterRecs) + len(fastReaderRecs),
		PhaseBudgetMS: float64(armBudget.Milliseconds()),
	}
}

// catalogSlowViolations returns a description of every assert violation, or
// "" when the slow-client behavior holds.
func catalogSlowViolations(recs []catalogSlowClientRecord) string {
	detail := ""
	deadlineMS := float64(slowClientDeadline.Milliseconds())
	slackMS := float64(slowDeadlineSlack.Milliseconds())
	for _, r := range recs {
		inWindow := r.ElapsedMS >= deadlineMS-20 && r.ElapsedMS <= deadlineMS+slackMS
		if r.ErrClass != "context_deadline_exceeded" || !inWindow {
			detail += fmt.Sprintf(" client=%s op=%s elapsed_ms=%.0f err=%q;", r.Client, r.Op, r.ElapsedMS, r.ErrClass)
		}
	}
	return detail
}

func catalogSlowFastLatencyViolations(recs []catalogSlowLatency) string {
	detail := ""
	budgetMS := float64(fastClientBudget.Milliseconds())
	for _, r := range recs {
		if r.LatencyMS >= budgetMS || r.ErrClass != "" {
			detail += fmt.Sprintf(" client=%s op=%s latency_ms=%.0f err=%q;", r.Client, r.Op, r.LatencyMS, r.ErrClass)
		}
	}
	return detail
}

func catalogSlowAsserts(ev catalogSlowClientEvidence) []catalogSlowAssert {
	var out []catalogSlowAssert
	add := func(name, detail string) {
		out = append(out, catalogSlowAssert{Name: name, Violated: detail != "", Detail: detail})
	}
	for _, arm := range ev.Arms {
		// Read phase.
		rp := arm.ReadPhase
		add(arm.Arm+"_read_slow_client_timeout_bounded", catalogSlowViolations(rp.Slow))
		add(arm.Arm+"_read_fast_client_error_storm_absent", func() string {
			if rp.FastErrors > 0 {
				return fmt.Sprintf("fast_errors=%d;", rp.FastErrors)
			}
			return ""
		}())
		add(arm.Arm+"_read_fast_reader_latency_within_budget", catalogSlowFastLatencyViolations(rp.FastReaders))
		// Write phase.
		wp := arm.WritePhase
		add(arm.Arm+"_write_slow_client_timeout_bounded", catalogSlowViolations(wp.Slow))
		add(arm.Arm+"_write_fast_client_error_storm_absent", func() string {
			if wp.FastErrors > 0 {
				return fmt.Sprintf("fast_errors=%d;", wp.FastErrors)
			}
			return ""
		}())
		add(arm.Arm+"_write_fast_writer_latency_within_budget", catalogSlowFastLatencyViolations(wp.FastWriters))
		add(arm.Arm+"_write_fast_reader_latency_within_budget", catalogSlowFastLatencyViolations(wp.FastReaders))
	}
	return out
}

// TestLiveSlowClientCatalogSurface injects slow clients into the catalog
// read/write surface and asserts the system holds under them. See the file
// header for the arms, the injection and the recorded evidence.
func TestLiveSlowClientCatalogSurface(t *testing.T) {
	if os.Getenv("HELIX_LOAD_RUN") != "1" {
		t.Skip("HELIX_LOAD_RUN is not 1; measurement harness opted out")
	}
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	// Cleanup, not defer: the registered work deletions below run first (LIFO).
	t.Cleanup(func() { repo.Close() })

	// Self-contained: the read phase reads a work this test owns, so it neither
	// skips nor fails on an empty database and never reads another run's rows.
	seedWork := slowClientWork("seed-"+strconv.FormatInt(time.Now().UTC().UnixNano(), 36), time.Now().UTC())
	if err := repo.MergeWork(ctx, seedWork); err != nil {
		t.Fatalf("seed work: %v", err)
	}
	seedID := seedWork.ID
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		if err := repo.DeleteWork(cctx, seedID); err != nil {
			t.Logf("cleanup seed work %s: %v", seedID, err)
		}
	})

	spec := &slowSpec{stall: slowClientStall}
	slowRepo := slowCatalogDB(t, dsn, spec)
	defer slowRepo.db.Close()

	arms := []struct {
		name string
		slow int
		fast int
	}{
		{name: "a_one_slow_one_fast", slow: 1, fast: 1},
		{name: "b_three_slow_plus_fast", slow: 3, fast: 3},
	}
	ev := catalogSlowClientEvidence{
		Test:            "TestLiveSlowClientCatalogSurface",
		Surface:         "catalog read/write surface (server/internal/catalog/postgres.go)",
		TestFile:        "server/internal/catalog/slow_client_live_test.go",
		MeasurementPath: loadMeasurementPath(),
		EmittedAt:       time.Now().UTC().Format(time.RFC3339),
		Parameters: map[string]any{
			"slow_stall_ms":           slowClientStall.Milliseconds(),
			"slow_client_deadline_ms": slowClientDeadline.Milliseconds(),
			"slow_deadline_slack_ms":  slowDeadlineSlack.Milliseconds(),
			"fast_client_budget_ms":   fastClientBudget.Milliseconds(),
			"fast_rounds_per_client":  fastRoundsPerClient,
			"fast_writer_rounds":      fastWriterRounds,
			"arm_budget_s":            armBudget.Seconds(),
			"pool_max_open_conns":     defaultPoolSettings().maxOpenConns,
			"measurement_path":        loadMeasurementPath(),
		},
	}
	for _, arm := range arms {
		aev := catalogSlowArmEvidence{Arm: arm.name, SlowClients: arm.slow, FastClients: arm.fast}
		spec.readStall.Store(true)
		aev.ReadPhase = runCatalogSlowReadPhase(t, ctx, repo, slowRepo, arm.name, seedID, arm.slow, arm.fast)
		spec.readStall.Store(false)
		work := slowClientWork(arm.name+"-"+strconv.FormatInt(time.Now().UTC().UnixNano(), 36), time.Now().UTC())
		t.Cleanup(func() {
			cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer ccancel()
			_ = repo.DeleteWork(cctx, work.ID)
		})
		aev.WritePhase = runCatalogSlowWritePhase(t, ctx, repo, slowRepo, spec, arm.name, arm.fast, work, seedID)
		ev.Arms = append(ev.Arms, aev)
	}
	ev.Asserts = catalogSlowAsserts(ev)
	writeCatalogSlowClientEvidence(t, ev)

	for _, a := range ev.Asserts {
		if a.Violated {
			t.Errorf("slow-client assert %s violated: %s", a.Name, a.Detail)
		}
	}
}
