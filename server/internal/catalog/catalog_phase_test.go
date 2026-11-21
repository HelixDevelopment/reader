package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestLiveScaleBatchPhases attributes the cost of the batched page loader and
// compares the two ways of expressing "load these 200 child rows": a 200-value
// parameterized IN list versus a single array parameter.
//
// It also pins the measurement path itself. Every figure here is a wall-clock
// number taken by this process, and a wall-clock number attributes the whole path
// — query plan, server execution, and everything carrying the bytes between here
// and there. On this class of host the database is reached through a rootless
// container network namespace whose userspace packet forwarder has been measured
// stalling a multi-packet response by ~205 ms while the server itself reported
// 3 ms for the same statement. A ratio taken through such a path describes the
// forwarder, not the query.
//
// So each shape is timed twice: once by this client, and once by the server
// describing its own execution through EXPLAIN (ANALYZE). Where the client's
// number is far larger than the server's, the measurement is refused out loud
// rather than reported as a result.
func TestLiveScaleBatchPhases(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn, WithPool(4, 4, time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	var seeded int
	if err := repo.db.QueryRowContext(ctx, "SELECT count(*) FROM works WHERE id LIKE $1", scalePrefix+"%").Scan(&seeded); err != nil {
		t.Fatal(err)
	}
	if seeded < scaleWorks {
		t.Skipf("scale fixture not seeded (%d works); run TestLiveScaleCatalog first", seeded)
	}
	ids, err := scanStrings(ctx, repo, "SELECT id FROM works WHERE id LIKE $1 ORDER BY updated_at DESC, id LIMIT 100", scalePrefix+"%")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 100 {
		t.Fatalf("sample ids = %d, want 100", len(ids))
	}
	editionIDs, err := scanStrings(ctx, repo, "SELECT id FROM editions WHERE work_id IN ("+placeholders(len(ids))+") ORDER BY id", toArgs(ids)...)
	if err != nil {
		t.Fatal(err)
	}
	if len(editionIDs) != 200 {
		t.Fatalf("sample editions = %d, want 200", len(editionIDs))
	}

	probe := transportProbe(ctx, t, repo, editionIDs, ids)
	degraded := probe.degraded
	walls := probe.client
	_ = walls

	const iterations = 12

	// The claim the batched loader makes: fewer round trips describe the same page.
	batchWall := timed(iterations, func() {
		if _, err := repo.getWorks(ctx, ids); err != nil {
			t.Fatal(err)
		}
	})
	perWorkWall := timed(4, func() {
		for _, id := range ids {
			if _, err := repo.GetWork(ctx, id); err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Logf("LOADER batched_getWorks_100 median_ms=%.2f  perwork_GetWork_100 median_ms=%.2f", batchWall.Median, perWorkWall.Median)
	if degraded > 0 {
		t.Skipf("measurement path degraded for %d of %d shapes (client time far exceeds the server's own account of the same statement): the ratios below would describe the network path, not the queries. Re-run inside the database's own network namespace: %s", degraded, probeShapes, scaleInContainerHint)
	}
	if batchWall.Median > perWorkWall.Median {
		t.Errorf("batched loader is slower than one call per work: %.2f ms vs %.2f ms", batchWall.Median, perWorkWall.Median)
	}
}

type timing struct {
	Median float64 `json:"median_ms"`
	First  float64 `json:"first_ms"`
	Min    float64 `json:"min_ms"`
	Max    float64 `json:"max_ms"`
}

// timed runs fn n times and reports wall-clock milliseconds. Every result set is
// consumed inside the timed region: leaving rows open returns the connection to
// the pool mid-scan and makes the timings measure the harness rather than the
// query.
func timed(n int, fn func()) timing {
	times := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		fn()
		times = append(times, float64(time.Since(start).Microseconds())/1000)
	}
	sorted := append([]float64(nil), times...)
	insertionSort(sorted)
	return timing{Median: sorted[len(sorted)/2], First: times[0], Min: sorted[0], Max: sorted[len(sorted)-1]}
}

func insertionSort(values []float64) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

var execTimeRE = regexp.MustCompile(`Execution Time: ([0-9.]+) ms`)

// serverExecutionMS asks PostgreSQL how long it spent executing the statement,
// excluding everything between the server and this process.
func serverExecutionMS(ctx context.Context, t *testing.T, repo *Postgres, query string, args ...any) float64 {
	t.Helper()
	text := explainText(ctx, t, repo, query, args...)
	return execTimeMS(t, text)
}

// execTimeMS reads the server's own statement timing out of an EXPLAIN (ANALYZE)
// result.
func execTimeMS(t *testing.T, text string) float64 {
	t.Helper()
	match := execTimeRE.FindStringSubmatch(text)
	if match == nil {
		panic("no execution time reported for plan:\n" + text)
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

// explainText asks PostgreSQL to execute the statement and return its plan.
func explainText(ctx context.Context, t *testing.T, repo *Postgres, query string, args ...any) string {
	t.Helper()
	rows, err := repo.db.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+query, args...)
	if err != nil {
		t.Fatalf("explain: %v\nquery: %s", err, query)
	}
	defer rows.Close()
	lines := []string{}
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return strings.Join(lines, "\n")
}

// absent returns n identifiers the fixture cannot contain, for probes that must
// bind a parameter list without matching rows.
func absent(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("no-such-edition-%d", i)
	}
	return out
}

// probeShapes is how many statement shapes transportProbe checks; the tests that
// report a degraded path count against it.
const probeShapes = 6

const scaleInContainerHint = `build the test binary with "go test -c -o /tmp/helix-catalog.test ./internal/catalog", ` +
	`podman cp it into the database container, and run it there with ` +
	`HELIX_TEST_POSTGRES_DSN pointing at 127.0.0.1:5432`

func countRows(rows *sql.Rows) (int, error) {
	count := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		count++
	}
	return count, rows.Err()
}

func scanStrings(ctx context.Context, repo *Postgres, query string, args ...any) ([]string, error) {
	rows, err := repo.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func placeholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(parts, ",")
}

func toArgs(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

// transportProbe times the same set of statement shapes from this process and from
// the server itself, and returns the per-shape wall-clock figures along with how
// many shapes disagree. Both scale tests use it, so a payload can never be
// labelled trustworthy by a weaker check than the one that skips the comparison
// next to it.
func transportProbe(ctx context.Context, t *testing.T, repo *Postgres, editionIDs, workIDs []string) probeResult {
	t.Helper()
	out := probeResult{
		client: map[string]timing{},
		server: map[string]float64{},
	}
	// A predicate fragment is substituted rather than the whole WHERE clause being
	// written out per variant, so the three shapes below cannot drift apart in
	// their projection or ordering and be compared while differing in more than
	// the one thing under test.
	identifiers := "SELECT edition_id,scheme,value,source,confidence FROM identifiers WHERE %s ORDER BY edition_id,scheme,value,source"
	// Every shape below projects four text columns and one double precision, which is what
	// consume scans. Widening a shape has to keep that contract or the probe fails
	// loudly rather than mis-measuring.
	consume := func(label string, query string, args ...any) int {
		rows, err := repo.db.QueryContext(ctx, query, args...)
		if err != nil {
			t.Fatalf("%s: %v\nquery: %s\nargs: %d", label, err, query, len(args))
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			var a, b, c, d string
			var e float64
			if err := rows.Scan(&a, &b, &c, &d, &e); err != nil {
				t.Fatalf("%s: %v", label, err)
			}
			count++
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		return count
	}
	variants := []struct {
		name  string
		query string
		args  []any
	}{
		{"in_200_params", fmt.Sprintf(identifiers, "edition_id IN ("+placeholders(len(editionIDs))+")"), toArgs(editionIDs)},
		{"any_array_param", fmt.Sprintf(identifiers, "edition_id = ANY($1)"), []any{editionIDs}},
		{"in_20_params", fmt.Sprintf(identifiers, "edition_id IN ("+placeholders(20)+")"), toArgs(editionIDs[:20])},
		// Transport probes, separating the cost of returning many rows from the cost
		// of binding many parameters. The first touches no table, the second binds
		// the same 200 parameters and matches nothing.
		{"probe_gen_200_rows", "SELECT g::text,'scheme'::text,'value'::text,'source'::text,1.0::double precision FROM generate_series(1,200) g ORDER BY 1", nil},
		{"probe_in_200_no_rows", fmt.Sprintf(identifiers, "edition_id IN ("+placeholders(200)+")"), toArgs(absent(200))},
		// The loader shape the payload reports, checked as one statement rather than
		// inferred from the round trips around it.
		{"loader_getWorks_100", "SELECT w.id,w.canonical_title,w.series,w.subjects::text,w.series_index FROM works w WHERE w.id IN (" + placeholders(len(workIDs)) + ") ORDER BY w.updated_at DESC,w.id", toArgs(workIDs)},
	}
	const iterations = 12
	if len(variants) != probeShapes {
		t.Fatalf("transportProbe checks %d shapes, probeShapes says %d", len(variants), probeShapes)
	}
	for _, v := range variants {
		// The server times its own execution of the identical statement, with the
		// identical parameters, through the same pool.
		server := serverExecutionMS(ctx, t, repo, v.query, v.args...)
		out.server[v.name] = server
		wall := timed(iterations, func() {
			rows := consume(v.name, v.query, v.args...)
			if rows == 0 && !strings.HasPrefix(v.name, "probe_in_") {
				t.Errorf("%s returned no rows; the fixture and the probe disagree", v.name)
			}
		})
		out.client[v.name] = wall
		// Five times the server's own figure is well beyond the variance between two
		// executions of a sub-millisecond query, and far below the stall that this
		// comparison exists to catch.
		unfaithful := wall.Median > server*5+25
		if unfaithful {
			out.degraded++
		}
		transport := "clean"
		if unfaithful {
			transport = "DEGRADED"
		}
		t.Logf("VARIANT %-20s client_median_ms=%8.2f client_first_ms=%8.2f client_min_ms=%8.2f client_max_ms=%8.2f server_exec_ms=%6.2f transport=%s", v.name, wall.Median, wall.First, wall.Min, wall.Max, server, transport)
	}
	return out
}

// probeResult is one transportProbe run: per-shape client wall clocks, per-shape
// server self-reported execution times, and how many shapes disagree by more than
// the threshold.
type probeResult struct {
	client   map[string]timing
	server   map[string]float64
	degraded int
}
