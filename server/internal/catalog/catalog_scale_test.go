package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// Large-catalog qualification for ATM-027. This is synthetic data seeded with a
// fixed number of bulk statements; it establishes what the query shapes cost at
// volume, not what a particular user's library costs. Results are printed as one
// JSON line prefixed SCALE so a runner can capture them without reformatting Go
// test output.
//
// Run with:
//
//	HELIX_TEST_POSTGRES_DSN=... go test ./internal/catalog -run TestLiveScale -v
const (
	scaleWorks    = 50000
	scaleEditions = 2
	scaleFiles    = 2
	scalePrefix   = "scale-work-"
	// scaleBase anchors every seeded timestamp so the fixture is identical on
	// every run; a random component here would make latency figures irreproducible.
	scaleBase = "2026-01-01T00:00:00Z"
)

type scaleRow struct {
	Name       string  `json:"name"`
	Limit      int     `json:"limit"`
	Iterations int     `json:"iterations"`
	MedianMS   float64 `json:"median_ms"`
	MinMS      float64 `json:"min_ms"`
	MaxMS      float64 `json:"max_ms"`
	Rows       int     `json:"rows"`
	Error      string  `json:"error,omitempty"`
}

func scaleAvailable(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	return dsn
}

func TestLiveScaleCatalog(t *testing.T) {
	dsn := scaleAvailable(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	if _, err := repo.db.ExecContext(ctx, "DELETE FROM works WHERE id LIKE $1", scalePrefix+"%"); err != nil {
		t.Fatal(err)
	}
	seedStart := time.Now()
	statements := []string{
		`INSERT INTO works(id,canonical_title,created_at,updated_at)
		 SELECT $5||g, 'Scale Volume '||g, t0, t0 + (g * interval '1 second')
		 FROM generate_series(1,$1) g, (SELECT t0::timestamptz t0 FROM (SELECT ($2::timestamptz + $3::interval) t0) s) c`,
	}
	_ = statements
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO works(id,canonical_title,created_at,updated_at)
		SELECT $3||g, 'Scale Volume '||g, $2::timestamptz, $2::timestamptz + (g * interval '1 second')
		FROM generate_series(1,$1) g`, scaleWorks, scaleBase, scalePrefix); err != nil {
		t.Fatalf("seed works: %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO editions(id,work_id,title,authors,publisher,language,description,state,confidence,updated_at)
		SELECT 'scale-edition-'||g||'-'||e, $3||g, 'Scale Volume '||g||' edition '||e, '["Scale Author"]'::jsonb, 'Helix Press', 'en', '', CASE WHEN g % 5 = 0 THEN 'pending' ELSE 'identified' END, 0.9,
		       $2::timestamptz + (g * interval '1 second')
		FROM generate_series(1,$1) g, generate_series(1,$4) e`, scaleWorks, scaleBase, scalePrefix, scaleEditions); err != nil {
		t.Fatalf("seed editions: %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO identifiers(edition_id,scheme,value,source,confidence)
		SELECT 'scale-edition-'||g||'-'||e, 'isbn', '9780000'||lpad((g*10+e)::text,8,'0'), 'scale', 1
		FROM generate_series(1,$1) g, generate_series(1,$2) e`, scaleWorks, scaleEditions); err != nil {
		t.Fatalf("seed identifiers: %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO files(id,edition_id,source_id,path,format,mime_type,size,sha256,protected,modified_at)
		SELECT 'scale-file-'||g||'-'||e||'-'||f, 'scale-edition-'||g||'-'||e, 'scale-source', 'scale/'||g||'/'||e||'/'||f||'.epub', 'epub', 'application/epub+zip', 1048576,
		       right(md5(g::text)||md5(e::text), 64), false, $1::timestamptz + (g * interval '1 second')
		FROM generate_series(1,$2) g, generate_series(1,$3) e, generate_series(1,$4) f`, scaleBase, scaleWorks, scaleEditions, scaleFiles); err != nil {
		t.Fatalf("seed files: %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO metadata_evidence(edition_id,field,provider,confidence,observed_at)
		SELECT 'scale-edition-'||g||'-'||e, 'title', 'scale', 0.95, $1::timestamptz + (g * interval '1 second')
		FROM generate_series(1,$2) g, generate_series(1,$3) e`, scaleBase, scaleWorks, scaleEditions); err != nil {
		t.Fatalf("seed evidence: %v", err)
	}
	if _, err := repo.db.ExecContext(ctx, "ANALYZE works, editions, identifiers, files, metadata_evidence"); err != nil {
		t.Fatal(err)
	}

	// An index-only scan over works_updated_id has to know which heap pages are
	// visible to it, and a table that was just bulk-loaded has no all-visible pages
	// at all. The planner prices every page as a heap fetch and prefers a sequential
	// scan plus an external merge sort over the index that exists for exactly this
	// ORDER BY. VACUUM is what makes those pages visible, so it belongs between a
	// bulk write and a query-performance claim — and the plan from before it is
	// recorded rather than discarded, because it is what a client sees immediately
	// after a large ingest.
	pageQuery := `SELECT id FROM works w ORDER BY w.updated_at DESC,w.id LIMIT 20 OFFSET 40000`
	preVacuum := explainText(ctx, t, repo, pageQuery)
	if _, err := repo.db.ExecContext(ctx, "VACUUM (ANALYZE) works"); err != nil {
		t.Fatal(err)
	}
	postVacuum := explainText(ctx, t, repo, pageQuery)
	t.Logf("PLANNER pre_vacuum  %s", strings.Join(strings.Fields(strings.SplitN(preVacuum, "\n", 2)[0]), " "))
	t.Logf("PLANNER post_vacuum %s", strings.Join(strings.Fields(strings.SplitN(postVacuum, "\n", 2)[0]), " "))
	if strings.Contains(postVacuum, "Seq Scan on works") {
		t.Errorf("page query planned a sequential scan after VACUUM, so works_updated_id does not serve it:\n%s", postVacuum)
	}
	if !strings.Contains(postVacuum, "works_updated_id") {
		t.Errorf("page query did not use works_updated_id after VACUUM:\n%s", postVacuum)
	}

	seedElapsed := time.Since(seedStart)
	t.Logf("SCALE seed_elapsed_ms=%d works=%d vacuum=done", seedElapsed.Milliseconds(), scaleWorks)
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if os.Getenv("HELIX_SCALE_KEEP") == "1" {
			t.Log("scale fixture retained for phase attribution")
			_ = cleanup
		} else if _, err := repo.db.ExecContext(cleanup, "DELETE FROM works WHERE id LIKE $1", scalePrefix+"%"); err != nil {
			t.Errorf("scale cleanup: %v", err)
		}
	}()

	var rows []scaleRow
	measure := func(name string, iterations int, limit int, fn func() (int, error)) {
		times := []time.Duration{}
		last := 0
		for i := 0; i < iterations; i++ {
			start := time.Now()
			got, err := fn()
			if err != nil {
				rows = append(rows, scaleRow{Name: name, Limit: limit, Error: err.Error()})
				return
			}
			times = append(times, time.Since(start))
			last = got
		}
		sort.Slice(times, func(i, k int) bool { return times[i] < times[k] })
		rows = append(rows, scaleRow{
			Name: name, Iterations: iterations, Rows: last, Limit: limit,
			MedianMS: float64(times[len(times)/2].Microseconds()) / 1000,
			MinMS:    float64(times[0].Microseconds()) / 1000,
			MaxMS:    float64(times[len(times)-1].Microseconds()) / 1000,
		})
	}

	// Offset sweep: this is the shape a paginated client actually issues.
	for _, offset := range []int{0, 1000, 10000, 40000, 49000} {
		o := offset
		measure(fmt.Sprintf("search_browse_offset_%d_limit_20", o), 7, 20, func() (int, error) {
			page, err := repo.Search(ctx, Query{Offset: o, Limit: 20})
			return len(page.Works), err
		})
	}
	// The cursor shape at the same depth: a client that has already walked to
	// offset 40,000 asks for what follows the last row it saw, so the database never
	// counts and discards 40,000 rows to answer.
	deep, err := repo.Search(ctx, Query{Offset: 40000, Limit: 20})
	if err != nil {
		t.Fatalf("deep offset page: %v", err)
	}
	if len(deep.Works) != 20 {
		t.Fatalf("deep offset page returned %d works, want 20", len(deep.Works))
	}
	deepCursor := CursorAfter(deep.Works)
	if deepCursor == nil {
		t.Fatal("no cursor after a full deep page")
	}
	measure("search_keyset_after_offset_40000_limit_20", 7, 20, func() (int, error) {
		page, err := repo.Search(ctx, Query{After: deepCursor, Limit: 20})
		return len(page.Works), err
	})
	// The equivalence of the two shapes is pinned by
	// TestLivePostgresCursorWalkMatchesOffset; this measures the cost difference.

	// The two shapes above are nearly the same cost, because the offset page is not
	// mostly the OFFSET: Search pays a count(*) of the whole filtered set on every
	// call, and the batched child load follows for both. These three measurements
	// separate the parts so the documentable claim is about the term the cursor
	// actually removes, not about the request as a whole.
	measure("part_count_star_only", 7, 1, func() (int, error) {
		var n int
		err := repo.db.QueryRowContext(ctx, `SELECT count(*) FROM works w WHERE w.id LIKE $1`, scalePrefix+"%").Scan(&n)
		return n, err
	})
	measure("part_ids_at_offset_40000", 7, 20, func() (int, error) {
		rows, err := repo.db.QueryContext(ctx, `SELECT w.id FROM works w WHERE w.id LIKE $1 ORDER BY w.updated_at DESC,w.id LIMIT 20 OFFSET 40000`, scalePrefix+"%")
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		return countRows(rows)
	})
	measure("part_ids_at_keyset", 7, 20, func() (int, error) {
		rows, err := repo.db.QueryContext(ctx, `SELECT w.id FROM works w WHERE w.id LIKE $1 AND (w.updated_at < $2 OR (w.updated_at = $3 AND w.id > $4)) ORDER BY w.updated_at DESC,w.id LIMIT 20`,
			scalePrefix+"%", deepCursor.UpdatedAt, deepCursor.UpdatedAt, deepCursor.ID)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		return countRows(rows)
	})

	// Both shapes above pay the same, because the fixture-scoping LIKE cannot be
	// served by works_updated_id: PostgreSQL walks the ordering index and filters, so
	// there is no start point to seek to. The real browse query has no such filter,
	// and there the difference appears — measured separately rather than assumed.
	measure("part_ids_unfiltered_offset_40000", 7, 20, func() (int, error) {
		rows, err := repo.db.QueryContext(ctx, `SELECT w.id FROM works w ORDER BY w.updated_at DESC,w.id LIMIT 20 OFFSET 40000`)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		return countRows(rows)
	})
	measure("part_ids_unfiltered_keyset", 7, 20, func() (int, error) {
		rows, err := repo.db.QueryContext(ctx, `SELECT w.id FROM works w WHERE w.updated_at < $1 OR (w.updated_at = $2 AND w.id > $3) ORDER BY w.updated_at DESC,w.id LIMIT 20`,
			deepCursor.UpdatedAt, deepCursor.UpdatedAt, deepCursor.ID)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		return countRows(rows)
	})

	// Filtered shapes: a state predicate and an unanchored text match, which
	// cannot use a btree index by construction.
	measure("search_state_identified_limit_20", 7, 20, func() (int, error) {
		page, err := repo.Search(ctx, Query{State: Identified, Offset: 40000, Limit: 20})
		return len(page.Works), err
	})
	measure("search_text_prefix_like_limit_20", 7, 20, func() (int, error) {
		page, err := repo.Search(ctx, Query{Text: "Scale Volume 42", Limit: 20})
		return len(page.Works), err
	})

	// The N+1 comparison: the previous page loader versus the batched one, over
	// the same identifier set.
	ids, err := func() ([]string, error) {
		rowsX, err := repo.db.QueryContext(ctx, "SELECT id FROM works WHERE id LIKE $1 ORDER BY updated_at DESC, id LIMIT 100", scalePrefix+"%")
		if err != nil {
			return nil, err
		}
		defer rowsX.Close()
		out := []string{}
		for rowsX.Next() {
			var id string
			if err := rowsX.Scan(&id); err != nil {
				return nil, err
			}
			out = append(out, id)
		}
		return out, rowsX.Err()
	}()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 100 {
		t.Fatalf("sample ids = %d, want 100", len(ids))
	}
	measure("page_load_per_work_100", 5, 100, func() (int, error) {
		total := 0
		for _, id := range ids {
			w, err := repo.GetWork(ctx, id)
			if err != nil {
				return 0, err
			}
			total += len(w.Editions)
		}
		return total, nil
	})
	measure("page_load_batched_100", 5, 100, func() (int, error) {
		works, err := repo.getWorks(ctx, ids)
		if err != nil {
			return 0, err
		}
		total := 0
		for _, w := range works {
			total += len(w.Editions)
		}
		return total, nil
	})

	// Query-plan evidence for the batched loader's own shapes, with the same
	// 100-work IN list Search would build.
	buildList := func(count int, offset int) string {
		parts := []string{}
		for i := offset; i < offset+count && i < len(ids); i++ {
			parts = append(parts, fmt.Sprintf("$%d", i-offset+1))
		}
		return strings.Join(parts, ",")
	}
	one := ids[0]
	batch100 := buildList(100, 0)
	editionArgs := make([]any, 0, 100)
	for _, id := range ids {
		editionArgs = append(editionArgs, id)
	}
	childArgsSample := make([]any, 0, 200)
	for i := 0; i < 200 && i < len(ids)*2; i++ {
		childArgsSample = append(childArgsSample, fmt.Sprintf("scale-edition-%d-%d", i/2+1, i%2+1))
	}
	sampleChildList := strings.TrimSuffix(strings.Repeat("?,", len(childArgsSample)), ",")
	_ = sampleChildList
	placeholders := make([]string, len(childArgsSample))
	for i := range childArgsSample {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	childListSQL := strings.Join(placeholders, ",")
	dynamicPlans := map[string]struct {
		query string
		args  []any
	}{
		"batched_editions_in_100":    {"SELECT work_id,id,title,subtitle,authors,publisher,published,language,description,cover_href,state,confidence,updated_at FROM editions WHERE work_id IN (" + batch100 + ") ORDER BY work_id,id", editionArgs},
		"batched_identifiers_in_200": {"SELECT edition_id,scheme,value,source,confidence FROM identifiers WHERE edition_id IN (" + childListSQL + ") ORDER BY edition_id,scheme,value,source", childArgsSample},
		"batched_files_in_200":       {"SELECT edition_id,id,source_id,path,format,mime_type,size,sha256,protected,drm_scheme,license_href,modified_at FROM files WHERE edition_id IN (" + childListSQL + ") ORDER BY edition_id,id", childArgsSample},
		"single_editions_by_work":    {"SELECT id,title,subtitle,authors,publisher,published,language,description,cover_href,state,confidence,updated_at FROM editions WHERE work_id = $1 ORDER BY id", []any{one}},
	}
	_ = buildList

	// Query-plan evidence for the remaining page and count shapes.
	plans := map[string]string{
		"page_offset_40000_pre_vacuum":  preVacuum,
		"page_offset_40000_post_vacuum": postVacuum,
	}
	for name, query := range map[string]string{
		"count_all":        `SELECT count(*) FROM works w WHERE TRUE`,
		"editions_by_work": `SELECT id FROM editions WHERE work_id = (SELECT min(id) FROM works WHERE id LIKE 'scale-work-%')`,
	} {
		plans[name] = explainText(ctx, t, repo, query)
	}

	for name, spec := range dynamicPlans {
		plans[name] = explainText(ctx, t, repo, spec.query, spec.args...)
	}
	// The client-side figures above are wall-clock numbers from this process. If the
	// server's own account of the same statement is far smaller, the path between
	// them is carrying something the query did not cost, and every measurement in
	// this payload describes that path instead of the catalog. The check is the same
	// probe the batch-phase test uses — including its widest shapes — so a payload
	// cannot be labelled trustworthy by a weaker test than the numbers it ships with.
	editionSample := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		editionSample = append(editionSample, fmt.Sprintf("scale-edition-%d-%d", i/2+1, i%2+1))
	}
	probe := transportProbe(ctx, t, repo, editionSample, ids)
	degradedShapes := probe.degraded
	path := "clean"
	if degradedShapes > 0 {
		path = fmt.Sprintf("degraded_%d_of_%d_shapes", degradedShapes, probeShapes)
		t.Logf("MEASUREMENT PATH DEGRADED for %d shapes: those numbers describe the network path, not the catalog. Re-run where the database is: %s", degradedShapes, scaleInContainerHint)
	}

	payload, _ := json.Marshal(map[string]any{
		"measurements":          rows,
		"plans":                 plans,
		"measurement_path":      path,
		"probe_client_ms":       probe.client,
		"probe_server_ms":       probe.server,
		"measurement_path_note": "degraded means re-run inside the database container: " + scaleInContainerHint,
		"fixture_works":         scaleWorks,
		"seed_elapsed_ms":       seedElapsed.Milliseconds(),
	})
	t.Logf("SCALE %s", payload)
}
