package aimemory

// Cardinality, latency and recall benchmark for the vector search path. The
// surfaces it measures were named as unmeasured in docs/scalability-review.md
// ("no vector-cardinality/recall/latency benchmark").
//
// Why a benchmark table instead of ai_memories: the production store pins the
// embedding dimension into the table definition and refuses to open when the
// configured dimension differs from the migrated one (migrate() in
// postgres.go). The shared qualification database is migrated at the tiny
// dimension the functional live test uses, where recall is meaninglessly 1.0.
// So this file rebuilds that table in a schema of its own -- same columns, same
// types, same index set, same index names, verified by comparing the catalogs of
// the two tables rather than by asserting the similarity in a comment -- and runs
// production's own statement text against it, unmodified, by putting the bench
// schema first on search_path. What differs is the dimension and the row count,
// and the payload says so.
//
// Plan choice is a measured property here, not an assumption: every figure is
// labelled with the access path EXPLAIN actually reported, the plan is sampled as
// the table grows, and the classifier is proved to discriminate by a pair of
// profiles that differ only in planner settings. That discipline is what found
// DEFECT-005 -- the released Search ordered by `distance,id`, an ordering no index
// can supply, so the maintained HNSW index was never read and every Search scanned
// the whole table (14.58 ms against 0.183 ms at 20k rows on identical data).

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const benchSchema = "helix_bench"

// benchHNSWIndex is production's own index name (ai_memories_embedding), kept
// identical on purpose: the plan classifier below recognizes the access path by
// that name, so a renamed production index breaks the benchmark loudly instead of
// letting it report some other index as the ANN path.
const benchHNSWIndex = "ai_memories_embedding"
const benchTable = "ai_memories"

// legacySearchSQL is the Search statement as it stood before 2026-11-20, kept here
// only so the benchmark can measure what the fix changed. The `,id` inside the
// index-ordered clause is DEFECT-005: pgvector can order by `embedding <=> $2` and
// nothing else, so adding the tiebreak there made the HNSW index unusable and
// turned every Search into a sequential scan of every stored embedding plus a full
// sort. Production now uses searchSQL; this text must never return to postgres.go.
const legacySearchSQL = `SELECT id,scope,kind,content,metadata,model,content_sha256,created_at,expires_at,1-(embedding <=> $2::vector) AS similarity ` +
	`FROM ai_memories WHERE scope=$1 AND (expires_at IS NULL OR expires_at>now()) ORDER BY embedding <=> $2::vector,id LIMIT $3`

func benchEnabled() (string, bool) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	return dsn, dsn != "" && os.Getenv("HELIX_LOAD_RUN") == "1"
}

func benchEnvInt(name string, fallback int) int {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func benchMeasurementPath() string {
	if v := os.Getenv("HELIX_MEASUREMENT_PATH"); v != "" {
		return v
	}
	return "unspecified"
}

func benchEmit(t *testing.T, payload map[string]any) {
	t.Helper()
	payload["emitted_at"] = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	t.Logf("LOADJSON %s", data)
	if path := os.Getenv("HELIX_LOAD_OUTPUT"); path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			t.Errorf("open %s: %v", path, err)
			return
		}
		defer f.Close()
		if _, err := f.Write(append(data, '\n')); err != nil {
			t.Errorf("append %s: %v", path, err)
		}
	}
}

// benchMatrix returns n deterministically seeded, L2-normalized unit vectors of
// dim floats. Re-running with the same knobs reproduces the same dataset, which
// is the only way a recall figure means anything across machines.
func benchMatrix(n, dim int, seed uint64) [][]float32 {
	out := make([][]float32, 0, n)
	state := seed
	next := func() float64 {
		// xorshift64*
		state ^= state >> 12
		state ^= state << 25
		state ^= state >> 27
		// 53-bit mantissa conversion: float64(x)/2^64 would round away the low
		// bits and overflow at 1<<64.
		return float64(state>>11) / float64(uint64(1)<<53)
	}
	for i := 0; i < n; i++ {
		v := make([]float32, dim)
		var norm float64
		for d := 0; d < dim; d++ {
			x := next()*2 - 1
			v[d] = float32(x)
			norm += x * x
		}
		norm = math.Sqrt(norm)
		if norm == 0 {
			norm = 1
		}
		for d := range v {
			v[d] = float32(float64(v[d]) / norm)
		}
		out = append(out, v)
	}
	return out
}

func benchPercentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return -1
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	rank := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
}

// benchRun drives count requests through fn at the given client count and
// returns per-request latencies in milliseconds plus an error class tally.
func benchRun(ctx context.Context, clients, count int, fn func(context.Context, int) error) ([]float64, map[string]int, error) {
	if clients < 1 {
		clients = 1
	}
	latencies := make([]float64, 0, count)
	errs := map[string]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	next := make(chan int, count)
	for i := 0; i < count; i++ {
		next <- i
	}
	close(next)
	start := time.Now()
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				single := time.Now()
				if err := fn(ctx, i); err != nil {
					mu.Lock()
					errs[err.Error()]++
					mu.Unlock()
				}
				mu.Lock()
				latencies = append(latencies, float64(time.Since(single).Microseconds())/1000.0)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	return latencies, errs, nil
}

func benchStats(latencies []float64, errs map[string]int, seconds float64) map[string]any {
	completed := len(latencies) - sumErrs(errs)
	return map[string]any{
		"completed":       completed,
		"errors_by_class": errs,
		"wall_ms":         math.Round(seconds*100000) / 100,
		"req_per_sec":     math.Round(float64(completed)/seconds*100) / 100,
		"p50_ms":          math.Round(benchPercentile(latencies, 50)*1000) / 1000,
		"p95_ms":          math.Round(benchPercentile(latencies, 95)*1000) / 1000,
		"p99_ms":          math.Round(benchPercentile(latencies, 99)*1000) / 1000,
	}
}

func sumErrs(errs map[string]int) int {
	total := 0
	for _, n := range errs {
		total += n
	}
	return total
}

func benchLevels() []int {
	if v := strings.TrimSpace(os.Getenv("HELIX_LOAD_LEVELS")); v != "" {
		out := []int{}
		for _, part := range strings.Split(v, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && n > 0 {
				out = append(out, n)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []int{1, 8, 32}
}

// TestLiveLoadVectorSearch measures ANN latency, exact-scan latency and recall@k
// at a configurable vector cardinality, and asserts from the plan that the HNSW
// index is what was actually measured.
func TestLiveLoadVectorSearch(t *testing.T) {
	dsn, ok := benchEnabled()
	if !ok {
		t.Skip("HELIX_TEST_POSTGRES_DSN and HELIX_LOAD_RUN=1 are both required for the vector load benchmark")
	}
	dim := benchEnvInt("HELIX_VECTOR_DIMS", 384)
	rows := benchEnvInt("HELIX_VECTOR_ROWS", 5000)
	queries := benchEnvInt("HELIX_VECTOR_QUERIES", 100)
	k := benchEnvInt("HELIX_VECTOR_K", 10)
	levels := benchLevels()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(benchEnvInt("HELIX_LOAD_TIMEOUT_SECONDS", 1800))*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Defers run LIFO: the schema drop is registered after the close so it runs
	// first. Putting it in t.Cleanup instead drops the schema against a handle
	// that is already closed, and the leftover bench schema then poisons the next
	// run's cardinality assertion.
	defer db.Close()
	db.SetMaxOpenConns(levels[len(levels)-1] + 4)
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		t.Skipf("pgvector unavailable in this database: %v", err)
	}
	if _, err := db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+benchSchema+" CASCADE"); err != nil {
		t.Fatalf("drop stale bench schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+benchSchema); err != nil {
		t.Fatalf("create bench schema: %v", err)
	}
	defer func() {
		if _, err := db.Exec("DROP SCHEMA IF EXISTS " + benchSchema + " CASCADE"); err != nil {
			t.Errorf("cleanup bench schema: %v", err)
		}
	}()
	// The bench table is created with production's column list, types, index set and
	// index names, differing only in schema and embedding dimension. The unqualified
	// production statement then resolves against it through search_path, so no
	// character of the measured SQL has to be rewritten for the benchmark; the
	// equivalence of the two tables is asserted below rather than assumed.
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s.%s(
		id TEXT PRIMARY KEY,scope TEXT NOT NULL,kind TEXT NOT NULL,content TEXT NOT NULL,
		metadata JSONB NOT NULL DEFAULT '{}'::jsonb,model TEXT NOT NULL,
		embedding VECTOR(%d) NOT NULL,content_sha256 TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL,expires_at TIMESTAMPTZ)`, benchSchema, benchTable, dim)); err != nil {
		t.Fatalf("create bench table: %v", err)
	}
	indexStart := time.Now()
	for _, statement := range []string{
		fmt.Sprintf("CREATE INDEX ai_memories_scope ON %s.%s(scope,kind)", benchSchema, benchTable),
		fmt.Sprintf("CREATE INDEX ai_memories_expiry ON %s.%s(expires_at) WHERE expires_at IS NOT NULL", benchSchema, benchTable),
		fmt.Sprintf("CREATE INDEX %s ON %s.%s USING hnsw(embedding vector_cosine_ops)", benchHNSWIndex, benchSchema, benchTable),
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("create bench index: %v (%s)", err, statement)
		}
	}
	indexSeconds := time.Since(indexStart).Seconds()

	// --- statements, planner profiles, helpers -------------------------------

	scope := "bench"

	// Two statements are measured against the same data. "production" is searchSQL
	// read straight from postgres.go, so the benchmark cannot drift from the code it
	// claims to measure; "released_tiebreak" is the text that shipped until
	// 2026-11-20 and is kept only to put a number on DEFECT-005.
	type shape struct {
		name  string
		query string
		note  string
	}
	shapes := []shape{
		{"production", searchSQL, "live Search statement: inner ORDER BY the index can supply, tiebreak resolved outside it"},
		{"released_tiebreak", legacySearchSQL, "statement as released before 2026-11-20: ORDER BY distance,id, which no index can supply"},
	}

	// The bench schema is first on search_path for every connection here, which is
	// what allows the unqualified production statement to run unchanged against the
	// bench table. It is not a planner setting and does not affect plan choice.
	// Settings are written as name=value pairs because they travel to the server in
	// the connection startup packet (-c name=value), not as SET statements.
	pathGUC := fmt.Sprintf("search_path=%s,public,pg_catalog", benchSchema)

	type queryExecer interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	}
	// planText returns EXPLAIN text for one statement shape. analyze=true also runs
	// the query, which is the only way to see actual row counts; the cardinality
	// sweep uses analyze=false because it runs while the table is still growing.
	planText := func(ctx context.Context, q queryExecer, sh shape, analyze bool, vec []float32) string {
		t.Helper()
		statement := "EXPLAIN (VERBOSE, FORMAT TEXT) " + sh.query
		if analyze {
			statement = "EXPLAIN (ANALYZE, VERBOSE, FORMAT TEXT) " + sh.query
		}
		rows, err := q.QueryContext(ctx, statement, scope, VectorLiteral(vec), k)
		if err != nil {
			t.Fatalf("explain: %v", err)
		}
		defer rows.Close()
		var b strings.Builder
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				t.Fatalf("scan plan: %v", err)
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("plan rows: %v", err)
		}
		return b.String()
	}
	// classifyPlan names the access path a plan describes. It is deliberately
	// conservative: an unrecognized plan reports "other" and fails the run rather
	// than being counted as an index scan. benchHNSWIndex is production's own HNSW
	// index name, which appears in a plan only if that index is really read.
	classifyPlan := func(plan string) string {
		switch {
		case strings.Contains(plan, benchHNSWIndex):
			return "ann_index"
		case strings.Contains(plan, "Seq Scan") && strings.Contains(plan, "Sort"):
			return "seq_scan_topn"
		case strings.Contains(plan, "Scan") && strings.Contains(plan, "Sort"):
			return "other_index_then_sort"
		default:
			return "other"
		}
	}
	// planHead keeps payloads readable: the scan/sort node lines, not the
	// multi-kilobyte vector literal VERBOSE repeats for every node.
	planHead := func(plan string) string {
		lines := strings.Split(strings.TrimRight(plan, "\n"), "\n")
		head := make([]string, 0, 4)
		for _, line := range lines {
			if strings.Contains(line, "Scan") || strings.Contains(line, "Sort") || strings.Contains(line, "Limit") {
				if len(line) > 200 {
					line = line[:200] + "…"
				}
				head = append(head, strings.TrimSpace(line))
			}
			if len(head) == 4 {
				break
			}
		}
		return strings.Join(head, " | ")
	}
	// searchIDs runs one statement shape and returns the ids it selected, in the order
	// the statement returned them. Every column production's Search selects is
	// transferred and decoded, not just the id, because that transfer is part of what
	// a Search costs; the first column is the id in both statement shapes.
	searchIDs := func(ctx context.Context, q queryExecer, sh shape, vec []float32) ([]string, error) {
		rows, err := q.QueryContext(ctx, sh.query, scope, VectorLiteral(vec), k)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		columns, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		ids := []string{}
		for rows.Next() {
			if err := rows.Scan(pointers...); err != nil {
				return nil, err
			}
			switch v := values[0].(type) {
			case string:
				ids = append(ids, v)
			case []byte:
				ids = append(ids, string(v))
			default:
				return nil, fmt.Errorf("id column came back as %T, not text", values[0])
			}
		}
		return ids, rows.Err()
	}
	// attestSettings reads the settings back off the connections the profile is about
	// to run on and fails if any differs from what that profile's label claims.
	// Inferring a profile's identity from the plan class it produced was the design
	// here until the needle gate killed it: at a cardinality where the planner prefers
	// the index on its own, a mutation that stripped the forced profile's settings
	// survived, because ann_index looks the same either way. Checking only what the
	// connection string asked for would not catch that either, so each profile states
	// its requirement independently here, and the requirement, the transport and the
	// live session must all three agree.
	attestSettings := func(pdb *sql.DB, gucs []string, requires map[string]string, label string) map[string]string {
		t.Helper()
		normalize := func(value string) string {
			parts := strings.Split(strings.ToLower(value), ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			return strings.Join(parts, ",")
		}
		transport := map[string]string{}
		for _, setting := range gucs {
			name, value, ok := strings.Cut(setting, "=")
			if !ok {
				t.Fatalf("%s: connection setting %q is not name=value", label, setting)
			}
			transport[name] = value
		}
		for name, want := range transport {
			if claimed, isRequirement := requires[name]; isRequirement && normalize(claimed) != normalize(want) {
				t.Fatalf("%s: the connection asks for %s=%s while the profile label requires %s=%s", label, name, want, name, claimed)
			}
		}
		// Every connection is held at once, which is what forces the pool to actually
		// open that many. Borrowing and returning one connection in a loop would attest
		// a single session and assume the rest of the ladder matches it.
		count := levels[len(levels)-1]
		conns := make([]*sql.Conn, 0, count)
		defer func() {
			for _, conn := range conns {
				conn.Close()
			}
		}()
		wantPath := normalize(strings.TrimPrefix(pathGUC, "search_path="))
		observed := map[string]string{}
		for i := 0; i < count; i++ {
			conn, err := pdb.Conn(ctx)
			if err != nil {
				t.Fatalf("%s: reserve connection %d: %v", label, i, err)
			}
			conns = append(conns, conn)
			var indexScan, seqScan, bitmapScan, path string
			if err := conn.QueryRowContext(ctx, `SELECT current_setting('enable_indexscan'), current_setting('enable_seqscan'),
				current_setting('enable_bitmapscan'), current_setting('search_path')`).Scan(&indexScan, &seqScan, &bitmapScan, &path); err != nil {
				t.Fatalf("%s: read session settings: %v", label, err)
			}
			observed = map[string]string{"enable_indexscan": indexScan, "enable_seqscan": seqScan,
				"enable_bitmapscan": bitmapScan, "search_path": path}
			for name, want := range requires {
				got, known := observed[name]
				if !known {
					t.Fatalf("%s: required setting %s has no attestation column, so this harness would silently not check it", label, name)
				}
				if normalize(got) != normalize(want) {
					t.Fatalf("%s: connection %d reports %s=%q but the profile requires %q, so its figures would describe a different access path than the one they are labelled with",
						label, i, name, got, want)
				}
			}
			if normalize(path) != wantPath {
				t.Fatalf("%s: connection %d searches %q instead of %q, so it is measuring production's table rather than this fixture",
					label, i, path, wantPath)
			}
		}
		observed["connections_checked"] = fmt.Sprint(len(conns))
		return observed
	}
	ownProfileDB := func(gucs []string) *sql.DB {
		t.Helper()
		// "options" is applied by the server when the connection starts, so the
		// settings reach every connection the pool opens during a ladder and not
		// only the first one. attestSettings checks that they really did, rather than
		// trusting the plan class to imply it.
		arguments := make([]string, 0, len(gucs))
		for _, setting := range gucs {
			arguments = append(arguments, "-c "+setting)
		}
		profileDSN := dsn
		if len(arguments) > 0 {
			sep := "&"
			if !strings.Contains(dsn, "?") {
				sep = "?"
			}
			// Each -c needs its own flag, and the separator must be %20 rather than +:
			// url.QueryEscape produces +, which pgx hands to the server verbatim, and
			// PostgreSQL then reports "unrecognized configuration parameter
			// +search_path".
			profileDSN = fmt.Sprintf("%s%soptions=%s", dsn, sep, strings.ReplaceAll(url.QueryEscape(strings.Join(arguments, " ")), "+", "%20"))
		}
		pdb, err := sql.Open("pgx", profileDSN)
		if err != nil {
			t.Fatalf("open profile pool: %v", err)
		}
		pdb.SetMaxOpenConns(levels[len(levels)-1] + 4)
		if err := pdb.PingContext(ctx); err != nil {
			t.Fatalf("ping profile pool (%s): %v", profileDSN, err)
		}
		return pdb
	}

	// --- is this table really production's shape? ----------------------------

	// The benchmark is evidence about production only if the tables match. Compare
	// them instead of asserting it in a comment.
	schemaEquivalence := "no public." + benchTable + " in this database to compare against"
	var productionTable string
	if err := db.QueryRowContext(ctx, "SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename=$1", benchTable).Scan(&productionTable); err == nil {
		columns := func(schema string) []string {
			rows, err := db.QueryContext(ctx, `SELECT column_name||':'||data_type FROM information_schema.columns
				WHERE table_schema=$1 AND table_name=$2 ORDER BY ordinal_position`, schema, benchTable)
			if err != nil {
				t.Fatalf("columns of %s: %v", schema, err)
			}
			defer rows.Close()
			out := []string{}
			for rows.Next() {
				var v string
				if err := rows.Scan(&v); err != nil {
					t.Fatalf("scan column: %v", err)
				}
				// The embedding dimension is the one intentional difference: migrate()
				// pins it to the configured model, and this run measures another.
				if strings.HasPrefix(v, "embedding:") {
					v = "embedding:USER-DEFINED"
				}
				out = append(out, v)
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("column rows: %v", err)
			}
			return out
		}
		indexes := func(schema string) []string {
			rows, err := db.QueryContext(ctx, `SELECT indexdef FROM pg_indexes
				WHERE schemaname=$1 AND tablename=$2`, schema, benchTable)
			if err != nil {
				t.Fatalf("indexes of %s: %v", schema, err)
			}
			defer rows.Close()
			out := []string{}
			for rows.Next() {
				var def string
				if err := rows.Scan(&def); err != nil {
					t.Fatalf("scan indexdef: %v", err)
				}
				// Normalize away the index name and the schema-qualified relation, which
				// differ by construction; what must match is the access method, the
				// opclass, the column list and any predicate.
				if i := strings.Index(def, " ON "); i >= 0 {
					rest := def[i+4:]
					if j := strings.Index(rest, " "); j >= 0 {
						def = strings.TrimSpace(rest[j:])
					}
				}
				out = append(out, def)
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("index rows: %v", err)
			}
			sort.Strings(out)
			return out
		}
		benchColumns, productionColumns := columns(benchSchema), columns("public")
		benchIndexes, productionIndexes := indexes(benchSchema), indexes("public")
		if !reflect.DeepEqual(benchColumns, productionColumns) {
			t.Fatalf("bench table has drifted from production: bench %v, production %v", benchColumns, productionColumns)
		}
		// Needle with real teeth: if the bench HNSW index were missing, renamed, or
		// built with another opclass, every plan class reported below would describe
		// a table production does not have.
		if !reflect.DeepEqual(benchIndexes, productionIndexes) {
			t.Fatalf("bench indexes have drifted from production: bench %v, production %v", benchIndexes, productionIndexes)
		}
		schemaEquivalence = fmt.Sprintf("columns and indexes identical to public.%s (%d columns, %d indexes; embedding dimension deliberately different)",
			benchTable, len(benchColumns), len(benchIndexes))
	} else if err != sql.ErrNoRows {
		t.Fatalf("look for production table: %v", err)
	}

	// --- seeding, with the plan recorded as the table grows ------------------

	seedStart := time.Now()
	corpus := benchMatrix(rows, dim, 20261120)
	probes := benchMatrix(queries, dim, 777)
	const batch = 100
	// Plan choice between an ordered index scan and "sequential scan plus top-N
	// heapsort" turns on the row estimate, so a single plan text recorded at one
	// fixed cardinality cannot describe the surface. Each checkpoint analyzes and
	// then explains both statements while the table still holds exactly that many
	// rows.
	checkpoints := []int{}
	for _, c := range []int{1000, 5000, 20000, 50000, 100000, 250000} {
		if c < rows {
			checkpoints = append(checkpoints, c)
		}
	}
	checkpoints = append(checkpoints, rows)
	nextCheckpoint := 0
	planAtCardinality := func(ctx context.Context, cardinality int) {
		t.Helper()
		// The catalog review records that plan choice on a freshly loaded table turns
		// on visibility-map freshness, so each checkpoint is analyzed before its plan
		// is read; otherwise this sweep would report the cold-load plan.
		if _, err := db.ExecContext(ctx, "VACUUM (ANALYZE) "+benchSchema+"."+benchTable); err != nil {
			t.Fatalf("vacuum analyze at %d rows: %v", cardinality, err)
		}
		for _, sh := range shapes {
			plan := planText(ctx, db, sh, false, probes[0])
			class := classifyPlan(plan)
			if class == "other" {
				t.Fatalf("unclassified plan for %s at %d rows:\n%s", sh.name, cardinality, plan)
			}
			benchEmit(t, map[string]any{
				"test": "vector_search", "role": "plan_at_cardinality",
				"measurement_path": benchMeasurementPath(),
				"cardinality":      cardinality, "dimensions": dim, "k": k,
				"statement": sh.name, "plan_class": class, "plan_head": planHead(plan),
			})
		}
	}
	for start := 0; start < rows; start += batch {
		end := start + batch
		if end > rows {
			end = rows
		}
		values := make([]string, 0, end-start)
		args := make([]any, 0, (end-start)*3)
		for i := start; i < end; i++ {
			args = append(args, fmt.Sprintf("bench-%d", i), scope, VectorLiteral(corpus[i]))
			values = append(values, fmt.Sprintf("($%d,$%d,'note','content','model',$%d::vector,'sha',now())", (i-start)*3+1, (i-start)*3+2, (i-start)*3+3))
		}
		statement := fmt.Sprintf("INSERT INTO %s.%s(id,scope,kind,content,model,embedding,content_sha256,created_at) VALUES %s",
			benchSchema, benchTable, strings.Join(values, ","))
		if _, err := db.ExecContext(ctx, statement, args...); err != nil {
			t.Fatalf("seed batch at %d: %v", start, err)
		}
		// Production creates the index before the rows exist, so every insert above
		// maintains it; the checkpoint plans therefore describe a growing live table.
		for nextCheckpoint < len(checkpoints) && end >= checkpoints[nextCheckpoint] {
			planAtCardinality(ctx, checkpoints[nextCheckpoint])
			nextCheckpoint++
		}
	}
	seedSeconds := time.Since(seedStart).Seconds()
	var rowCount int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+benchSchema+"."+benchTable).Scan(&rowCount); err != nil {
		t.Fatalf("count: %v", err)
	}
	// Needle: without this the run could report ladder numbers for a table holding a
	// fraction of the requested cardinality after any partial seeding failure.
	if rowCount != rows {
		t.Fatalf("seeded %d rows, want %d", rowCount, rows)
	}
	// --- the measurement matrix ---------------------------------------------

	// Each profile gets its own pool. A planner setting is a session setting: putting
	// it on one borrowed connection would mix two access paths into one throughput
	// number.
	type profile struct {
		name     string
		gucs     []string
		requires map[string]string
		ladder   bool
		note     string
	}
	profiles := []profile{
		{name: "exact", gucs: []string{pathGUC, "enable_indexscan=off"},
			requires: map[string]string{"enable_indexscan": "off"},
			note:     "exhaustive scan: the recall ground truth, and the cost of the same statement without any index"},
		{name: "default", gucs: []string{pathGUC}, ladder: true,
			// "No override" is only a real profile if the server's own defaults are
			// these; a database configured otherwise would make this column describe
			// something else than its label.
			requires: map[string]string{"enable_indexscan": "on", "enable_seqscan": "on", "enable_bitmapscan": "on"},
			note:     "no planner override: whichever path the server picks for an ordinary production connection"},
		{name: "index_forced", gucs: []string{pathGUC, "enable_seqscan=off", "enable_bitmapscan=off"}, ladder: true,
			requires: map[string]string{"enable_seqscan": "off", "enable_bitmapscan": "off"},
			note:     "scan-and-sort denied, so this is the HNSW path wherever a statement is able to take it"},
	}
	results := map[string]map[string]any{}
	truths := map[string][][]string{}
	for _, sh := range shapes {
		for _, p := range profiles {
			pdb := ownProfileDB(p.gucs)
			fail := func(format string, args ...any) {
				pdb.Close()
				t.Fatalf(format, args...)
			}
			// The GUCs must be visible on the connections themselves before a single
			// figure is attributed to this profile.
			session := attestSettings(pdb, p.gucs, p.requires, fmt.Sprintf("%s/%s", sh.name, p.name))
			plan := planText(ctx, pdb, sh, true, probes[0])
			class := classifyPlan(plan)
			if class == "other" {
				fail("statement %s under profile %s produced an unclassified plan, so the harness cannot report which path it measured:\n%s", sh.name, p.name, plan)
			}
			// Needles on the GUC mechanism itself: a profile that asked for one plan
			// and got another means the setting never reached the connection, and every
			// figure attached to it would be a duplicate of the default profile wearing
			// a different label.
			switch {
			case p.name == "exact" && class != "seq_scan_topn":
				fail("statement %s under the exact profile is not an exhaustive scan (classified %s), so it cannot serve as recall ground truth:\n%s", sh.name, class, plan)
			case p.name == "index_forced" && sh.name == "production" && class != "ann_index":
				fail("the live Search statement did not read the HNSW index even with sequential and bitmap scans denied (classified %s), so the fix this file exists to measure is not in effect:\n%s", class, plan)
			case p.name == "index_forced" && sh.name == "released_tiebreak" && class == "ann_index":
				fail("the released tiebreak-inside-ORDER-BY statement did use the HNSW index after all, which refutes DEFECT-005 and means the fix was made for the wrong reason:\n%s", plan)
			}

			latents := make([]float64, 0, queries)
			ids := make([][]string, 0, queries)
			for i := 0; i < queries; i++ {
				start := time.Now()
				got, err := searchIDs(ctx, pdb, sh, probes[i])
				latents = append(latents, float64(time.Since(start).Microseconds())/1000.0)
				if err != nil {
					fail("statement %s profile %s query %d: %v", sh.name, p.name, i, err)
				}
				// Needle: exactly k rows, every time. A profile silently returning
				// fewer would report perfect recall on an empty result set.
				if len(got) != k {
					fail("statement %s profile %s query %d returned %d rows, want %d (cardinality %d, k %d)", sh.name, p.name, i, len(got), k, rows, k)
				}
				ids = append(ids, got)
			}
			recall := 1.0
			if p.name == "exact" {
				truths[sh.name] = ids
			} else {
				if len(truths[sh.name]) != queries {
					fail("statement %s profile %s measured before its ground truth existed", sh.name, p.name)
				}
				matched := 0
				for i, got := range ids {
					inTruth := map[string]bool{}
					for _, id := range truths[sh.name][i] {
						inTruth[id] = true
					}
					for _, id := range got {
						if inTruth[id] {
							matched++
						}
					}
				}
				recall = float64(matched) / float64(queries*k)
				// Needle, calibrated against chance rather than against an opinion about
				// how good the index should be: k matches out of `rows` candidates is
				// what a broken comparison still gets by luck. The approximation quality
				// itself is reported, and the ef_search sweep below is what distinguishes
				// "the index is lossy here" from "the harness compares nothing to nothing"
				// — at 50k rows x 384 dimensions of uniform vectors this fixture measured
				// recall@10 of 0.124 with pgvector's compiled-in search width, which is a
				// property of the data and the default, not of the harness.
				if recall < 0.02 {
					fail("statement %s profile %s recall@%d is %.4f over %d queries at %d rows: at or near the k/rows chance rate, so the fixture or the comparison is broken",
						sh.name, p.name, k, recall, queries, rows)
				}
			}

			// Self-recall: probe with rows that are in the table.
			selfProbes := 20
			if rows < selfProbes {
				selfProbes = rows
			}
			step := 1
			if selfProbes > 0 {
				step = rows / selfProbes
			}
			selfHits := 0
			for s := 0; s < selfProbes; s++ {
				idx := s * step
				if idx >= rows {
					continue
				}
				got, err := searchIDs(ctx, pdb, sh, corpus[idx])
				if err != nil {
					fail("statement %s profile %s self probe %d: %v", sh.name, p.name, s, err)
				}
				if len(got) > 0 && got[0] == fmt.Sprintf("bench-%d", idx) {
					selfHits++
				}
			}
			// A row is its own nearest neighbour by construction, so the index's top-1
			// rate on self probes is the most direct read of whether it is searching the
			// data this file loaded. The bar is set where a wrong table or a collapsed
			// fixture falls, not where a well-behaved index in high dimensions is
			// expected to land: at 384 dimensions of uniform vectors distance
			// concentration makes top-1 a genuinely harder question than it is in 64, and
			// guessing that bar would replace a measurement with an assumption.
			if p.name != "exact" && selfProbes > 0 && selfHits*4 < selfProbes {
				fail("statement %s profile %s returned the probe row itself as the top hit only %d/%d times, so it is not searching the rows this fixture loaded",
					sh.name, p.name, selfHits, selfProbes)
			}

			entry := map[string]any{
				"statement":        sh.name,
				"profile":          p.name,
				"gucs":             p.gucs,
				"note":             p.note,
				"plan_class":       class,
				"session_settings": session,
				"plan_head":        planHead(plan),
				"recall_at_k":      math.Round(recall*10000) / 10000,
				"self_top1":        fmt.Sprintf("%d/%d", selfHits, selfProbes),
				"sequential":       benchStats(latents, map[string]int{}, sumMillis(latents)/1000.0),
				"ladder":           []map[string]any{},
				"errors":           0,
				"order_stable":     "",
			}
			if p.ladder {
				ladder := []map[string]any{}
				throughput := map[int]float64{}
				walls := map[int]float64{}
				p50, _ := entry["sequential"].(map[string]any)["p50_ms"].(float64)
				for _, clients := range levels {
					// Size each level by duration, not by a fixed op count: an index path
					// that answers in a fifth of a millisecond needs hundreds of thousands
					// of queries before a "concurrent" claim means anything, and a
					// sequential scan needs only a few thousand. Same wall time per level,
					// so the levels are comparable.
					ops := ladderOps(ladderTargetSeconds, clients, p50, queries)
					sampler := ownProfileDB([]string{pathGUC})
					sampler.SetMaxOpenConns(2)
					stop := make(chan struct{})
					peak := make(chan int, 1)
					go func() {
						peak <- benchPeakBackends(ctx, sampler, stop)
					}()
					start := time.Now()
					lats, errs, err := benchRun(ctx, clients, ops, func(ctx context.Context, i int) error {
						_, err := searchIDs(ctx, pdb, sh, probes[i%queries])
						return err
					})
					elapsed := time.Since(start).Seconds()
					close(stop)
					peakActive := <-peak
					sampler.Close()
					if err != nil {
						fail("statement %s profile %s ladder %d clients: %v", sh.name, p.name, clients, err)
					}
					// Needle: a ladder reporting throughput while queries error out would
					// read as a capacity result for work that never completed.
					if sumErrs(errs) != 0 {
						fail("statement %s profile %s ladder %d clients reported errors: %v", sh.name, p.name, clients, errs)
					}
					stats := benchStats(lats, errs, elapsed)
					stats["clients"] = clients
					stats["ops"] = ops
					stats["peak_active_backends"] = peakActive
					ladder = append(ladder, stats)
					walls[clients] = elapsed
					throughput[clients] = float64(len(lats)) / elapsed
					// Needle: this one does not depend on how fast the workload is, only on
					// whether the database ever saw two of these queries running at once.
					if clients >= 8 && peakActive < 2 {
						fail("statement %s profile %s ladder %d clients never had a second backend active at the same time (peak %d), so the ladder measured serialization and not concurrency",
							sh.name, p.name, clients, peakActive)
					}
				}
				entry["ladder"] = ladder
				if top := levels[len(levels)-1]; top >= 4 && len(levels) > 1 {
					gain := throughput[top] / throughput[levels[0]]
					// Needle: levels are duration-scaled, so concurrency shows up as more
					// operations per second, not shorter wall time. A harness that quietly
					// serialized -- one shared connection, a locked pool, a single-threaded
					// server -- would report ~1.0x here.
					if gain < 1.5 {
						fail("statement %s profile %s throughput needle: %d clients reached %.1f req/s against %.1f req/s for 1 client (%.2fx), so nothing here demonstrates concurrent capacity",
							sh.name, p.name, top, throughput[top], throughput[levels[0]], gain)
					}
					entry["throughput_gain_at_top_x"] = math.Round(gain*100) / 100
				}
			}
			if sh.name == "production" {
				// Determinism is the property the outer sort exists to preserve: the
				// index chooses the set approximately, but a given query must see the
				// same order every time or clients cannot paginate or diff results.
				first, err := searchIDs(ctx, pdb, sh, probes[1])
				if err != nil {
					fail("determinism probe: %v", err)
				}
				stable := true
				for r := 1; r < 5; r++ {
					again, err := searchIDs(ctx, pdb, sh, probes[1])
					if err != nil {
						fail("determinism probe repeat %d: %v", r, err)
					}
					if strings.Join(again, ",") != strings.Join(first, ",") {
						stable = false
					}
				}
				entry["order_stable"] = stable
				if !stable {
					fail("the live Search statement returned a different order for the same query across 5 runs, so the deterministic tiebreak is not doing its job")
				}
			}
			results[sh.name+"/"+p.name] = entry
			pdb.Close()
		}
	}

	// Third layer, and the one that depends on nothing this file asserts about
	// itself: a plain range query whose plan must differ between the default and
	// index_forced settings at any cardinality, because one of those sessions has
	// sequential scans denied. If the settings ever stopped reaching the server, this
	// fails even on a one-row table, where the vector statement's plan class is not
	// discriminating enough to notice.
	findProfile := func(name string) profile {
		for _, candidate := range profiles {
			if candidate.name == name {
				return candidate
			}
		}
		t.Fatalf("profile %s is not defined, so the measurements labelled with it have no source", name)
		return profile{}
	}
	plainPlan := func(p profile) string {
		pool := ownProfileDB(p.gucs)
		defer pool.Close()
		rs, err := pool.QueryContext(ctx, "EXPLAIN (VERBOSE, FORMAT TEXT) SELECT id FROM ai_memories WHERE scope=$1 LIMIT 5", scope)
		if err != nil {
			t.Fatalf("profile %s: explain control query: %v", p.name, err)
		}
		defer rs.Close()
		var b strings.Builder
		for rs.Next() {
			var line string
			if err := rs.Scan(&line); err != nil {
				t.Fatalf("scan control plan: %v", err)
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
		if err := rs.Err(); err != nil {
			t.Fatalf("control plan rows: %v", err)
		}
		return b.String()
	}
	controlDefault := plainPlan(findProfile("default"))
	controlForced := plainPlan(findProfile("index_forced"))
	if !strings.Contains(controlDefault, "Seq Scan") {
		t.Fatalf("the control query does not sequential-scan under the default profile at %d rows, so it cannot prove anything about the settings:\n%s", rows, controlDefault)
	}
	if strings.Contains(controlForced, "Seq Scan") || !strings.Contains(controlForced, "Index") {
		t.Fatalf("the control query still sequential-scans under the index_forced profile, so those planner settings are not reaching the session and every figure labelled index_forced measures the default path:\n%s", controlForced)
	}

	// What the recall figures mean, as distinct from what they say. pgvector widens
	// HNSW search with one knob — ef_search, compile-time default 40 — that nothing in
	// this product exposes, so a low recall has exactly two candidate explanations: the
	// harness is comparing nothing against nothing, or the default width is lossy on
	// this data. Only the second makes recall move when the width does, which is what
	// this distinguishes. The assertion is directional rather than an absolute bar, so
	// it holds at 64 dimensions where recall is already near 1 as well as at 384 where
	// it is not.
	widths := []int{40, 100, 400}
	truth := truths["production"]
	widthProbes := queries
	if widthProbes > 40 {
		widthProbes = 40
	}
	if len(truth) < widthProbes {
		widthProbes = len(truth)
	}
	widthResults := []map[string]any{}
	previous := -1.0
	first, last := 0.0, 0.0
	for _, width := range widths {
		if widthProbes == 0 {
			break
		}
		gucs := append(append([]string{}, findProfile("index_forced").gucs...), fmt.Sprintf("hnsw.ef_search=%d", width))
		pool := ownProfileDB(gucs)
		var reported string
		err := pool.QueryRowContext(ctx, "SHOW hnsw.ef_search").Scan(&reported)
		if err == nil && reported != fmt.Sprint(width) {
			err = fmt.Errorf("session reports hnsw.ef_search=%s, wanted %d", reported, width)
		}
		if err != nil {
			pool.Close()
			t.Fatalf("the search width the sweep is labelled with is not the one the session used: %v", err)
		}
		matched := 0
		for i := 0; i < widthProbes; i++ {
			got, serr := searchIDs(ctx, pool, shapes[0], probes[i])
			if serr != nil {
				pool.Close()
				t.Fatalf("ef_search %d probe %d: %v", width, i, serr)
			}
			inTruth := map[string]bool{}
			for _, id := range truth[i] {
				inTruth[id] = true
			}
			for _, id := range got {
				if inTruth[id] {
					matched++
				}
			}
		}
		pool.Close()
		recall := float64(matched) / float64(widthProbes*k)
		widthResults = append(widthResults, map[string]any{
			"ef_search":     width,
			"recall_at_k":   math.Round(recall*10000) / 10000,
			"probes":        widthProbes,
			"reported_back": reported,
		})
		if previous >= 0 && recall+0.02 < previous {
			t.Fatalf("recall at hnsw.ef_search=%d (%.4f) is below the narrower width's %.4f, so the sweep is not measuring search width at all",
				width, recall, previous)
		}
		previous = recall
		if width == widths[0] {
			first = recall
		}
		last = recall
	}
	if widthProbes > 0 && first < 0.9 && last <= first+0.05 {
		t.Fatalf("widening hnsw.ef_search from %d to %d moved recall@%d from %.4f to %.4f on a fixture the index should be able to search, so either the setting is not reaching the search or the recall figures describe something other than this fixture",
			widths[0], widths[len(widths)-1], k, first, last)
	}

	// Positive control on the classifier: the profiles that differ only in planner
	// settings must land in different classes. If they ever agree, "this benchmark
	// measured the index path" stops being falsifiable — which is exactly how the
	// first draft of this file reported a sequential scan as an ANN result.
	if results["production/exact"]["plan_class"] == results["production/index_forced"]["plan_class"] {
		t.Fatalf("plan classifier has no detection power: both exact and index_forced classified as %v", results["production/exact"]["plan_class"])
	}
	// The claim the fix makes, measured rather than asserted: with enough rows that
	// the sequential alternative is expensive, the live statement's index path must
	// beat the released statement's plan. Below 20k rows the difference is noise, so
	// the assertion is gated on the cardinality that makes it meaningful.
	if rows >= 20000 {
		fixed, _ := results["production/index_forced"]["sequential"].(map[string]any)
		broken, _ := results["released_tiebreak/default"]["sequential"].(map[string]any)
		fixedP50, _ := fixed["p50_ms"].(float64)
		brokenP50, _ := broken["p50_ms"].(float64)
		if fixedP50 <= 0 || brokenP50 <= 0 || fixedP50 >= brokenP50 {
			t.Fatalf("the fixed statement is not faster than the released one at %d rows (index path p50 %v ms, released p50 %v ms), so DEFECT-005's diagnosis or the fix built on it is wrong",
				rows, fixedP50, brokenP50)
		}
	}
	for _, key := range []string{"production/default", "production/index_forced", "released_tiebreak/default"} {
		if r, ok := results[key]; ok {
			if recall, _ := r["recall_at_k"].(float64); recall > 0 && recall < 0.95 {
				t.Logf("FINDING: recall@%d is %.4f for %s against an exhaustive scan of the same fixture", k, recall, key)
			}
		}
	}
	if class, _ := results["production/default"]["plan_class"].(string); class != "ann_index" {
		t.Logf("FINDING: at %d rows x %d dims the server serves the live statement as %s rather than from the HNSW index; the index_forced rows describe the ANN path's cost", rows, dim, class)
	}
	// Relative cost, which is the ratio a deployment decision actually turns on.
	for _, sh := range shapes {
		base, _ := results[sh.name+"/exact"]["sequential"].(map[string]any)
		baseP99, _ := base["p99_ms"].(float64)
		for _, p := range profiles {
			if p.name == "exact" || baseP99 <= 0 {
				continue
			}
			s, _ := results[sh.name+"/"+p.name]["sequential"].(map[string]any)
			if p99, _ := s["p99_ms"].(float64); p99 > 0 {
				results[sh.name+"/"+p.name]["p99_over_exact"] = math.Round(p99/baseP99*1000) / 1000
			}
		}
	}
	// The production store cannot be opened at this dimension if the shared table
	// is migrated at another (migrate() rejects it), which is itself the property
	// the review records as a constraint. Verify the guard rather than assume it,
	// and measure the production path at whatever dimension the table really has.
	productionPath := "no-production-migration"
	productionRows := -1
	var recorded int
	switch err := db.QueryRowContext(ctx, "SELECT dimensions FROM helix_ai_memory_migrations WHERE version=1").Scan(&recorded); err {
	case sql.ErrNoRows:
		// Nothing to check against; leave the labels as they are.
	case nil:
		if _, err := NewPostgres(ctx, dsn, recorded+1); err == nil || !strings.Contains(err.Error(), "dimensions") {
			t.Fatalf("production store accepted a mismatched embedding dimension without a dimension error: %v", err)
		}
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM ai_memories").Scan(&productionRows); err != nil {
			t.Fatalf("count production rows: %v", err)
		}
		store, err := NewPostgres(ctx, dsn, recorded)
		if err != nil {
			t.Fatalf("open production store at its migrated dimension %d: %v", recorded, err)
		}
		defer store.Close()
		probe := make([]float32, recorded)
		for i := range probe {
			probe[i] = float32(i+1) / float32(recorded*10)
		}
		prodStart := time.Now()
		for i := 0; i < 20; i++ {
			if _, err := store.Search(ctx, "live-scope", probe, 10); err != nil {
				t.Fatalf("production Search: %v", err)
			}
		}
		productionPath = "measured-small-fixture"
		benchEmit(t, map[string]any{
			"test": "vector_search", "role": "production_path",
			"measurement_path": benchMeasurementPath(),
			"dimensions":       recorded, "rows_in_table": productionRows,
			"queries": 20, "k": 10,
			"wall_ms": math.Round(time.Since(prodStart).Seconds()*100000) / 100,
			"note":    "Production Search() at the dimension the shared table is migrated to, over the small functional fixture. Not a cardinality result.",
		})
	default:
		t.Fatalf("read migrated dimensions: %v", err)
	}

	out := make([]any, 0, len(shapes)*len(profiles))
	for _, sh := range shapes {
		for _, p := range profiles {
			out = append(out, results[sh.name+"/"+p.name])
		}
	}
	benchEmit(t, map[string]any{
		"test": "vector_search", "role": "summary",
		"measurement_path":    benchMeasurementPath(),
		"cardinality":         rows,
		"dimensions":          dim,
		"k":                   k,
		"queries":             queries,
		"levels":              levels,
		"seed_seconds":        math.Round(seedSeconds*1000) / 1000,
		"index_build_seconds": math.Round(indexSeconds*1000) / 1000,
		"vacuum_analyze":      true,
		"schema_equivalence":  schemaEquivalence,
		// The heads of the two control plans are kept so a reader can see for
		// themselves that the profile settings changed what the server did.
		"recall_vs_ef_search":      widthResults,
		"guc_control_default_plan": planHead(controlDefault),
		"guc_control_forced_plan":  planHead(controlForced),
		// Measured per statement, not assumed: the first draft of this file asserted
		// the index was used, and the planner had chosen a sequential scan.
		"production_plan_default": results["production/default"]["plan_class"],
		"production_plan_index":   results["production/index_forced"]["plan_class"],
		"released_plan_default":   results["released_tiebreak/default"]["plan_class"],
		"released_plan_index":     results["released_tiebreak/index_forced"]["plan_class"],
		"hnsw_index_used":         results["production/default"]["plan_class"] == "ann_index",
		"statements":              out,
		"production_path":         productionPath,
		"production_rows":         productionRows,
		"table":                   "public.ai_memories' column list, types, index set and index names, rebuilt in schema " + benchSchema + " at this run's dimension; the unqualified production statement resolves to it through search_path",
		"limits": "synthetic unit vectors uniform on the sphere with a fixed seed, one scope matching every row, no deletes, no concurrent writers during the ladder, " +
			"no real embedding model in the loop; recall measured against an exhaustive scan of the same fixture, so it bounds the index's approximation error on separable data, not on production embeddings. " +
			"HNSW search parameters (ef_search, ef_construction, m) are the compiled-in defaults because none is configurable in this product.",
	})
	t.Logf("VECTOR %d rows x %d dims, k=%d | production plan by default: %v | index path p50 %v ms vs released-statement p50 %v ms | index recall@k %v",
		rows, dim, k, results["production/default"]["plan_class"],
		results["production/index_forced"]["sequential"].(map[string]any)["p50_ms"],
		results["released_tiebreak/default"]["sequential"].(map[string]any)["p50_ms"],
		results["production/index_forced"]["recall_at_k"])
}

// ladderTargetSeconds is the wall time each ladder level is sized to. Long enough
// that per-query latency and client setup do not dominate the measurement, short
// enough that the whole matrix stays inside the test timeout at every cardinality.
const ladderTargetSeconds = 1.5

// ladderOps returns the op count that keeps one ladder level about
// targetSeconds long at the given client count and per-query latency, never below
// the floor and never above the ceiling that bounds the run.
func ladderOps(targetSeconds float64, clients int, p50Milliseconds float64, floor int) int {
	if p50Milliseconds <= 0 {
		p50Milliseconds = 1
	}
	want := int(math.Ceil(targetSeconds * float64(clients) * 1000.0 / p50Milliseconds))
	if want < floor {
		want = floor
	}
	const ceiling = 400000
	if want > ceiling {
		want = ceiling
	}
	return want
}

// benchPeakBackends samples how many other active connections this database had
// while the caller ran its load, until stop closes. It answers the question a
// wall-time ratio cannot: whether the server ever saw two queries in flight.
func benchPeakBackends(ctx context.Context, db *sql.DB, stop <-chan struct{}) int {
	peak := 0
	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return peak
		case <-ctx.Done():
			return peak
		case <-ticker.C:
			var active int
			if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity
				WHERE datname = current_database() AND pid <> pg_backend_pid() AND state = 'active'`).Scan(&active); err != nil {
				return peak
			}
			if active > peak {
				peak = active
			}
		}
	}
}

func sumMillis(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}
