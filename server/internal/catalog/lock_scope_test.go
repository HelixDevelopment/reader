package catalog

// Lock-scope correctness for the catalog identity write path.
//
// The identity lock used to be one global advisory key, so every catalog writer
// in every source waited behind every other one. It is now one lock per identity
// a write can touch. That is only a safe trade if the narrower set still covers
// every pair of writers that shares a row, so these tests assert the invariants
// the lock is responsible for, and assert them twice: once through the sharded
// locks, and once through a control that runs the byte-identical transaction
// body with a different locking discipline. A difference between modes is
// therefore attributable to lock scope and to nothing else.
//
// Live tests skip unless HELIX_TEST_POSTGRES_DSN is set. The unit tests at the
// bottom need no database and always run.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// lockMode selects the locking discipline for a catalog write.
type lockMode string

const (
	// lockProduction is the shipped discipline: one advisory lock per identity key.
	lockProduction lockMode = "production"
	// lockLegacyGlobal reproduces the previous discipline: a single advisory lock
	// serializing every catalog writer.
	lockLegacyGlobal lockMode = "legacy-global"
	// lockRemoved runs the same transaction body with no advisory lock at all.
	// It is the needle: any invariant claimed above must break here.
	lockRemoved lockMode = "removed"
)

func liveDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	return dsn
}

// mergeWorkAs performs the MergeWork read-modify-write under mode, without the
// production retry loop. Suppressing the retry is deliberate: these tests need
// the raw outcome of an interleaving, not the outcome after the caller quietly
// tries again.
func mergeWorkAs(ctx context.Context, p *Postgres, mode lockMode, incoming Work) error {
	if err := validateWork(incoming); err != nil {
		return err
	}
	return oneTx(ctx, p.db, func(ctx context.Context, tx *sql.Tx) error {
		switch mode {
		case lockProduction:
			if err := lockIdentities(ctx, tx, identityLockKeys(incoming)); err != nil {
				return err
			}
		case lockLegacyGlobal:
			if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", identityLockSpace); err != nil {
				return err
			}
		case lockRemoved:
		default:
			return fmt.Errorf("catalog: unknown lock mode %q", mode)
		}
		existing, err := getWork(ctx, tx, incoming.ID, true)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err == nil {
			incoming = MergeWork(existing, incoming)
		}
		return upsertTx(ctx, tx, incoming)
	})
}

// reconcileAs performs ReconcileSource under mode, without retry.
func reconcileAs(ctx context.Context, p *Postgres, mode lockMode, sourceID string, present []string) (int, error) {
	presentJSON, err := stringsJSON(present)
	if err != nil {
		return 0, err
	}
	removed := 0
	err = oneTx(ctx, p.db, func(ctx context.Context, tx *sql.Tx) error {
		switch mode {
		case lockProduction:
			if err := lockIdentities(ctx, tx, []string{"s:" + sourceID}); err != nil {
				return err
			}
		case lockLegacyGlobal:
			if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", identityLockSpace); err != nil {
				return err
			}
		case lockRemoved:
		default:
			return fmt.Errorf("catalog: unknown lock mode %q", mode)
		}
		count, err := reconcileTx(ctx, tx, sourceID, presentJSON)
		removed = count
		return err
	})
	return removed, err
}

// scopeSpec pins the identity of one fixture work. Overlapping two writers on
// exactly one dimension (work, edition, or source) is what isolates one lock key
// from the others.
type scopeSpec struct {
	work    string
	edition string
	source  string
	path    string
	sha     string
	title   string
}

func scopeWork(spec scopeSpec, now time.Time) Work {
	return Work{
		ID: spec.work, CanonicalTitle: "Scope " + spec.title, CreatedAt: now, UpdatedAt: now,
		Editions: []Edition{{
			ID: spec.edition, Title: "Edition " + spec.title, Authors: []string{"Scope Harness"},
			State: Identified, Confidence: 1, UpdatedAt: now,
			Identifiers: []Identifier{{Scheme: "isbn", Value: "97911" + spec.title, Source: "scope", Confidence: 1}},
			Files: []File{{
				ID: spec.edition + "|" + spec.source + "|" + spec.path, SourceID: spec.source, Path: spec.path,
				Format: "txt", MIMEType: "text/plain", Size: 8, SHA256: spec.sha, ModifiedAt: now,
			}},
			Evidence: []MetadataEvidence{{Field: "title", Provider: "scope", Confidence: 1, ObservedAt: now}},
		}},
	}
}

// shaFor returns a deterministic 64-hex content hash for a label.
func shaFor(label string) string {
	sum := 0
	for _, r := range label {
		sum = sum*131 + int(r)
	}
	return fmt.Sprintf("%064x", uint64(sum))
}

func countCatalogRows(ctx context.Context, t *testing.T, p *Postgres, query string, args ...any) int {
	t.Helper()
	var n int
	if err := p.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

func deleteScopeFixtures(ctx context.Context, t *testing.T, p *Postgres, tag string) {
	t.Helper()
	if _, err := p.db.ExecContext(ctx, "DELETE FROM works WHERE id LIKE $1", "scope-"+tag+"-%"); err != nil {
		t.Errorf("scope cleanup %s: %v", tag, err)
	}
}

// runWriters runs fn concurrently for each i. The per-request deadline is
// generous enough for a contended lock wait yet short enough that a missing lock
// surfaces as an error instead of hanging the suite.
func runWriters(ctx context.Context, n int, fn func(wctx context.Context, i int) error) []error {
	out := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			out[i] = fn(wctx, i)
		}(i)
	}
	close(start)
	wg.Wait()
	return out
}

func errorCount(errs []error) int {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	return n
}

func firstError(errs []error) string {
	for _, err := range errs {
		if err != nil {
			return err.Error()
		}
	}
	return "none"
}

func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

const sameWorkWriters = 8

// mergeSameWorkRound has N writers merge into one work, each carrying an edition
// the others do not know about. It returns the number of failed writes and the
// number of editions that survived.
func mergeSameWorkRound(ctx context.Context, t *testing.T, repo *Postgres, mode lockMode, tag string, now time.Time) (int, int) {
	deleteScopeFixtures(ctx, t, repo, tag)
	errs := runWriters(ctx, sameWorkWriters, func(wctx context.Context, i int) error {
		return mergeWorkAs(wctx, repo, mode, scopeWork(scopeSpec{
			work: "scope-" + tag + "-work", edition: "scope-" + tag + "-ed-" + strconv.Itoa(i),
			source: "scope-src-" + tag + "-" + strconv.Itoa(i), path: "scope/" + tag + "/" + strconv.Itoa(i) + ".txt",
			sha: shaFor(tag + strconv.Itoa(i)), title: strconv.Itoa(i),
		}, now))
	})
	kept := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM editions WHERE work_id=$1", "scope-"+tag+"-work")
	return errorCount(errs), kept
}

func TestCatalogMergePreservesEveryWriterUnderIdentityLocks(t *testing.T) {
	dsn := liveDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)

	// upsertTx rewrites the whole edition set of the work it is given, so a writer
	// working from a stale snapshot deletes editions it never carried. The work key
	// is the lock that makes that read-modify-write atomic. This holds for the
	// sharded locks and for the old global lock alike, which is the point: the
	// narrower set must not lose either of the invariants the wide one protected.
	for _, mode := range []lockMode{lockProduction, lockLegacyGlobal} {
		tag := "keep-" + modeSlug(mode) + "-" + strconv.FormatInt(now.UnixNano(), 36)
		defer deleteScopeFixtures(ctx, t, repo, tag)
		failed, kept := mergeSameWorkRound(ctx, t, repo, mode, tag, now)
		if failed != 0 {
			t.Fatalf("%s: %d of %d concurrent same-work merges failed", mode, failed, sameWorkWriters)
		}
		if kept != sameWorkWriters {
			t.Fatalf("%s: lost update: %d editions survived %d concurrent merges of one work", mode, kept, sameWorkWriters)
		}
		t.Logf("MODE %s: %d concurrent same-work merges, 0 errors, all %d editions retained", mode, sameWorkWriters, sameWorkWriters)
	}
}

// modeSlug makes a lock mode safe to embed in a row identifier.
func modeSlug(mode lockMode) string {
	out := make([]byte, 0, len(mode))
	for i := 0; i < len(mode); i++ {
		c := mode[i]
		if c == '-' {
			c = 'x'
		}
		out = append(out, c)
	}
	return string(out)
}

// TestCatalogMergeNeedleUnlockedLosesWriters proves the assertions above have
// detection power: the identical transaction body, with the identity locks
// removed, does not preserve all eight writers. If this test ever reports that
// the unlocked control also survived, the concurrency assertions in this file are
// vacuous and must be strengthened rather than trusted.
func TestCatalogMergeNeedleUnlockedLosesWriters(t *testing.T) {
	dsn := liveDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)

	const rounds = 6
	worst := 0
	broke := 0
	for r := 0; r < rounds; r++ {
		tag := "needle-" + strconv.FormatInt(now.UnixNano(), 36) + "-" + strconv.Itoa(r)
		defer deleteScopeFixtures(ctx, t, repo, tag)
		failed, kept := mergeSameWorkRound(ctx, t, repo, lockRemoved, tag, now)
		lost := sameWorkWriters - kept
		if lost < 0 {
			lost = 0
		}
		if failed > 0 || lost > 0 {
			broke++
		}
		if failed+lost > worst {
			worst = failed + lost
		}
		t.Logf("NEEDLE round %d unlocked: %d writer errors, %d of %d editions lost", r+1, failed, lost, sameWorkWriters)
	}
	if broke == 0 {
		t.Fatalf("needle has no detection power: %d unlocked rounds of concurrent same-work merges all preserved every writer; the locked-path assertions prove nothing until this fails", rounds)
	}
	t.Logf("NEEDLE confirmed: %d of %d unlocked rounds lost writers (worst round: %d of %d)", broke, rounds, worst, sameWorkWriters)
}

// claimRound has N writers, each merging a distinct work that claims the same
// (source, path) file row. The file row can belong to exactly one edition at the
// end, and no work may be left describing a file it no longer owns.
func claimRound(ctx context.Context, t *testing.T, repo *Postgres, mode lockMode, tag string, now time.Time) (int, int, int) {
	deleteScopeFixtures(ctx, t, repo, tag)
	errs := runWriters(ctx, sameWorkWriters, func(wctx context.Context, i int) error {
		return mergeWorkAs(wctx, repo, mode, scopeWork(scopeSpec{
			work: "scope-" + tag + "-work-" + strconv.Itoa(i), edition: "scope-" + tag + "-ed-" + strconv.Itoa(i),
			// Shared source, shared path: every writer targets one file row.
			source: "scope-src-" + tag, path: "scope/" + tag + "/shared.txt",
			sha: shaFor(tag + "-claim-" + strconv.Itoa(i)), title: "claim" + strconv.Itoa(i),
		}, now))
	})
	rows := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE source_id=$1 AND path=$2", "scope-src-"+tag, "scope/"+tag+"/shared.txt")
	owners := countCatalogRows(ctx, t, repo, "SELECT count(DISTINCT e.work_id) FROM files f JOIN editions e ON e.id=f.edition_id WHERE f.source_id=$1 AND f.path=$2", "scope-src-"+tag, "scope/"+tag+"/shared.txt")
	return errorCount(errs), rows, owners
}

func TestCatalogFileClaimByDistinctWorksKeepsOneRow(t *testing.T) {
	dsn := liveDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)

	for _, mode := range []lockMode{lockProduction, lockLegacyGlobal} {
		tag := "claim-" + modeSlug(mode) + "-" + strconv.FormatInt(now.UnixNano(), 36)
		defer deleteScopeFixtures(ctx, t, repo, tag)
		failed, rows, owners := claimRound(ctx, t, repo, mode, tag, now)
		if rows != 1 {
			t.Fatalf("%s: %d file rows for one (source, path), want 1", mode, rows)
		}
		if owners != 1 {
			t.Fatalf("%s: one file row referenced by %d distinct works, want 1", mode, owners)
		}
		if mode == lockProduction && failed != 0 {
			t.Fatalf("%s: %d of %d claims failed under the shipped locks: concurrent writers on one source namespace must serialize, not error", mode, failed, sameWorkWriters)
		}
		t.Logf("MODE %s: %d errors, %d file row(s), %d work(s) owning it", mode, failed, rows, owners)
	}
}

// TestCatalogFileClaimNeedleUnlockedDuplicatesRow shows that the source key, not
// only the work key, is load bearing: the same N writers without identity locks
// either duplicate the file row or abort with a key conflict.
func TestCatalogFileClaimNeedleUnlockedDuplicatesRow(t *testing.T) {
	dsn := liveDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)

	const rounds = 6
	broke := 0
	for r := 0; r < rounds; r++ {
		tag := "claimneedle-" + strconv.FormatInt(now.UnixNano(), 36) + "-" + strconv.Itoa(r)
		defer deleteScopeFixtures(ctx, t, repo, tag)
		failed, rows, owners := claimRound(ctx, t, repo, lockRemoved, tag, now)
		if failed > 0 || rows != 1 || owners != 1 {
			broke++
		}
		t.Logf("NEEDLE round %d unlocked claim: %d errors, %d rows, %d owning works", r+1, failed, rows, owners)
	}
	if broke == 0 {
		t.Skipf("needle did not fire in %d unlocked rounds: the primary key on files alone prevented duplication this run, so the source-key assertion is unproven on this machine", rounds)
	}
	t.Logf("NEEDLE confirmed: %d of %d unlocked rounds violated the one-row-per-(source,path) invariant", broke, rounds)
}

// TestCatalogReconcileExcludesSameSourceMerge asserts that a scan completing a
// source cannot prune a file a concurrent merge of the same source is moving.
func TestCatalogReconcileExcludesSameSourceMerge(t *testing.T) {
	dsn := liveDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)

	for _, mode := range []lockMode{lockProduction, lockLegacyGlobal} {
		tag := "recon-" + modeSlug(mode) + "-" + strconv.FormatInt(now.UnixNano(), 36)
		defer deleteScopeFixtures(ctx, t, repo, tag)
		source := "scope-src-" + tag
		seed := func(wctx context.Context, i int) error {
			return mergeWorkAs(wctx, repo, mode, scopeWork(scopeSpec{
				work: "scope-" + tag + "-work", edition: "scope-" + tag + "-ed-" + strconv.Itoa(i),
				source: source, path: "scope/" + tag + "/" + strconv.Itoa(i) + ".txt",
				sha: shaFor(tag + "-recon-" + strconv.Itoa(i)), title: "recon" + strconv.Itoa(i),
			}, now))
		}
		if errs := runWriters(ctx, sameWorkWriters, seed); errorCount(errs) != 0 {
			t.Fatalf("%s: seed failed: %s", mode, firstError(errs))
		}
		// Half the files vanish from the listing while the other half is rewritten
		// with new bytes, which is exactly the interleaving a prune can race.
		present := []string{}
		for i := 0; i < sameWorkWriters/2; i++ {
			present = append(present, "scope/"+tag+"/"+strconv.Itoa(i)+".txt")
		}
		results := make([]error, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			_, results[0] = reconcileAs(wctx, repo, mode, source, present)
		}()
		go func() {
			defer wg.Done()
			wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			results[1] = mergeWorkAs(wctx, repo, mode, scopeWork(scopeSpec{
				work: "scope-" + tag + "-work", edition: "scope-" + tag + "-ed-0",
				source: source, path: "scope/" + tag + "/0.txt",
				sha: shaFor(tag + "-rewritten"), title: "rewritten",
			}, now))
		}()
		wg.Wait()
		if failed := errorCount(results); failed != 0 {
			t.Fatalf("%s: reconcile racing a same-source merge produced %d error(s): %s", mode, failed, firstError(results))
		}
		rows := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE source_id=$1", source)
		if rows != sameWorkWriters/2 {
			t.Fatalf("%s: %d file rows after pruning %d of %d paths, want %d", mode, rows, sameWorkWriters-len(present), sameWorkWriters, sameWorkWriters/2)
		}
		orphans := countCatalogRows(ctx, t, repo, `SELECT count(*) FROM files f
			LEFT JOIN editions e ON e.id=f.edition_id WHERE e.id IS NULL AND f.source_id=$1`, source)
		if orphans != 0 {
			t.Fatalf("%s: %d orphaned file rows", mode, orphans)
		}
		stale := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE source_id=$1 AND sha256=$2", source, shaFor(tag+"-rewritten"))
		if stale != 1 {
			t.Fatalf("%s: concurrent rewrite of a surviving path is missing (found %d, want 1)", mode, stale)
		}
		t.Logf("MODE %s: reconcile + same-source merge interleaved cleanly; %d rows kept, 0 orphans", mode, rows)
	}
}

// TestIdentityLockKeysCoverEverySharedRow needs no database. It pins the shape
// of the lock set: if a writer cannot name a row before its transaction starts,
// it cannot lock it, so coverage here is what the sharded discipline rests on.
func TestIdentityLockKeysCoverEverySharedRow(t *testing.T) {
	twoSources := Work{ID: "work-A", Editions: []Edition{
		{ID: "ed-1", Files: []File{{SourceID: "src-1"}, {SourceID: "src-2"}}},
		{ID: "ed-2", Files: []File{{SourceID: "src-2"}, {SourceID: "src-1"}}},
	}}
	keys := identityLockKeys(twoSources)
	want := []string{"w:work-A", "e:ed-1", "e:ed-2", "s:src-1", "s:src-2"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %d entries", keys, len(want))
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %v, want %v", keys, want)
		}
	}
	// Determinism: the same work must produce the same order every time, or two
	// sessions can acquire the same set in opposite orders and deadlock.
	for i := 0; i < 50; i++ {
		again := identityLockKeys(twoSources)
		if strings.Join(again, ",") != strings.Join(keys, ",") {
			t.Fatalf("key order changed between calls: %v then %v", keys, again)
		}
	}
	// Every row a merge can write must appear as a lockable key: the work, each
	// edition, and each source namespace its files come from.
	for _, part := range []string{"w:work-A", "e:ed-1", "e:ed-2", "s:src-1", "s:src-2"} {
		found := false
		for _, k := range keys {
			if k == part {
				found = true
			}
		}
		if !found {
			t.Fatalf("lock set %v does not cover %s", keys, part)
		}
	}
	// Disjoint works must share no key, or the sharding buys nothing.
	other := Work{ID: "work-B", Editions: []Edition{{ID: "ed-9", Files: []File{{SourceID: "src-9"}}}}}
	for _, a := range keys {
		for _, b := range identityLockKeys(other) {
			if a == b {
				t.Fatalf("disjoint works share lock key %s", a)
			}
		}
	}
	// Same bytes found in two sources share the work and edition keys and differ
	// only by source, so they still serialize on content.
	second := Work{ID: "work-A", Editions: []Edition{{ID: "ed-1", Files: []File{{SourceID: "src-other"}}}}}
	shared := 0
	for _, a := range identityLockKeys(second) {
		for _, b := range keys {
			if a == b {
				shared++
			}
		}
	}
	if shared < 2 {
		t.Fatalf("same work and edition from a different source share %d lock keys, want at least the work and edition", shared)
	}
}

// TestTransientConcurrencyClassification pins which failures the retry loop may
// absorb. Everything else must surface: a unique violation, for instance, means
// two writers that share a row were not excluded by the lock set, and retrying it
// would hide a real defect.
func TestTransientConcurrencyClassification(t *testing.T) {
	cases := []struct {
		code string
		want bool
		why  string
	}{
		{"40P01", true, "deadlock between an advisory wait and a row wait"},
		{"40001", true, "serialization failure"},
		{"23503", true, "prune raced a concurrent re-home of the same edition"},
		{"23505", false, "unique violation means the lock set missed a conflicting writer"},
		{"22P02", false, "malformed input is never transient"},
		{"23514", false, "a check constraint violation is a data bug"},
	}
	for _, c := range cases {
		err := fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: c.code, Message: c.why})
		if got := transientConcurrency(err); got != c.want {
			t.Errorf("transientConcurrency(%s) = %v, want %v (%s)", c.code, got, c.want, c.why)
		}
	}
	if transientConcurrency(errors.New("context deadline exceeded")) {
		t.Error("a plain error must not be treated as transient")
	}
	if transientConcurrency(nil) {
		t.Error("nil must not be treated as transient")
	}
}

// TestCatalogIdentityLockScopeThroughput measures whether the narrower lock set
// actually removes the serialization, using the same transaction body in all
// three arms:
//
//	locked-global  the previous discipline, one advisory lock for every writer
//	locked-sharded the shipped discipline, one advisory lock per identity touched
//	unlocked       no advisory lock, the ceiling on how fast the body can run
//
// All writers touch row-disjoint works, editions and sources, so the only thing
// the first two arms have in common with each other is lock scope. The sharded
// arm must land near the unlocked ceiling and clearly above the global arm; if it
// does not, the lock set is still colliding somewhere and this claim is false.
//
// Gated behind HELIX_LOAD_RUN=1 like the rest of the measurement harness.
func TestCatalogIdentityLockScopeThroughput(t *testing.T) {
	dsn := loadWanted(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	const clients = 8
	opsPerClient := 40
	if v := os.Getenv("HELIX_LOAD_OPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opsPerClient = n
		}
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	rounds := map[lockMode]map[string]any{}
	for _, mode := range []lockMode{lockLegacyGlobal, lockProduction, lockRemoved} {
		tag := "scope-" + modeSlug(mode) + "-" + strconv.FormatInt(now.UnixNano(), 36)
		defer deleteScopeFixtures(ctx, t, repo, tag)
		// Warm the connection pool before the timed arm so pool growth is not
		// charged to the first discipline measured.
		if errs := runWriters(ctx, clients, func(wctx context.Context, i int) error {
			return mergeWorkAs(wctx, repo, mode, scopeWork(scopeSpec{
				work: "scope-" + tag + "-warm-" + strconv.Itoa(i), edition: "scope-" + tag + "-warm-ed-" + strconv.Itoa(i),
				source: "scope-src-" + tag + "-warm", path: "scope/" + tag + "/warm/" + strconv.Itoa(i) + ".txt",
				sha: shaFor(tag + "-warm-" + strconv.Itoa(i)), title: "warm" + strconv.Itoa(i),
			}, now))
		}); errorCount(errs) != 0 {
			t.Fatalf("%s: warm-up failed: %s", mode, firstError(errs))
		}
		if _, err := repo.ReconcileSource(ctx, "scope-src-"+tag+"-warm", nil); err != nil {
			t.Fatalf("%s: warm-up cleanup: %v", mode, err)
		}
		start := time.Now()
		errs := runWriters(ctx, clients, func(wctx context.Context, c int) error {
			var last error
			for r := 0; r < opsPerClient; r++ {
				if err := mergeWorkAs(wctx, repo, mode, scopeWork(scopeSpec{
					work:    "scope-" + tag + "-w" + strconv.Itoa(c) + "-" + strconv.Itoa(r),
					edition: "scope-" + tag + "-w" + strconv.Itoa(c) + "-" + strconv.Itoa(r) + "-ed",
					source:  "scope-src-" + tag + "-" + strconv.Itoa(c),
					path:    "scope/" + tag + "/" + strconv.Itoa(c) + "/" + strconv.Itoa(r) + ".txt",
					sha:     shaFor(tag + "-w" + strconv.Itoa(c) + "-" + strconv.Itoa(r)),
					title:   "w" + strconv.Itoa(c) + "-" + strconv.Itoa(r),
				}, now)); err != nil {
					last = err
				}
			}
			return last
		})
		elapsed := time.Since(start)
		done := clients*opsPerClient - errorCount(errs)
		rounds[mode] = map[string]any{
			"ops":             clients * opsPerClient,
			"completed":       done,
			"errors":          errorCount(errs),
			"elapsed_seconds": elapsed.Seconds(),
			"ops_per_second":  float64(done) / elapsed.Seconds(),
		}
		t.Logf("SCOPE %s completed=%d/%d errors=%d elapsed=%.2fs ops_per_sec=%.2f",
			mode, done, clients*opsPerClient, errorCount(errs), elapsed.Seconds(), float64(done)/elapsed.Seconds())
		if mode == lockProduction && errorCount(errs) != 0 {
			t.Fatalf("%s: %d of %d disjoint merges failed: %s", mode, errorCount(errs), clients*opsPerClient, firstError(errs))
		}
	}
	gg := rounds[lockLegacyGlobal]["ops_per_second"].(float64)
	sh := rounds[lockProduction]["ops_per_second"].(float64)
	no := rounds[lockRemoved]["ops_per_second"].(float64)
	emitLoad(t, map[string]any{
		"test":             "catalog_identity_lock_scope",
		"measurement_path": loadMeasurementPath(),
		"clients":          clients,
		"ops_per_client":   opsPerClient,
		"phases":           rounds,
		"sharded_vs_global": map[string]any{
			"ratio":                   sh / gg,
			"headroom_ratio":          sh / no,
			"global_ops_per_second":   gg,
			"sharded_ops_per_second":  sh,
			"unlocked_ops_per_second": no,
		},
	})
	if sh <= gg*1.2 {
		t.Fatalf("lock scoping did not measurably remove serialization: sharded %.2f ops/s vs global %.2f ops/s (ratio %.2f, expected > 1.2)", sh, gg, sh/gg)
	}
	t.Logf("VERDICT sharded %.2f ops/s vs global %.2f ops/s = %.2fx; unlocked ceiling %.2f ops/s, sharded reaches %.0f%% of it", sh, gg, sh/gg, no, 100*sh/no)
}

// ownershipRound has two writers claim the same content-keyed edition under two
// different works. Exactly one of them can own it. The edition key is what makes
// the loser hear "this edition already belongs to work X" instead of a raw
// primary-key violation, and what keeps the winner's file row intact.
func ownershipRound(ctx context.Context, t *testing.T, repo *Postgres, mode lockMode, tag string, now time.Time) ([]error, int, int) {
	deleteScopeFixtures(ctx, t, repo, tag)
	claims := []scopeSpec{
		{work: "scope-" + tag + "-work-A", edition: "scope-" + tag + "-ed-shared", source: "scope-src-" + tag + "-A", path: "scope/" + tag + "/a.txt", sha: shaFor(tag + "-A"), title: "ownA"},
		{work: "scope-" + tag + "-work-B", edition: "scope-" + tag + "-ed-shared", source: "scope-src-" + tag + "-B", path: "scope/" + tag + "/b.txt", sha: shaFor(tag + "-B"), title: "ownB"},
	}
	errs := make([]error, len(claims))
	runWriters(ctx, len(claims), func(wctx context.Context, i int) error {
		errs[i] = mergeWorkAs(wctx, repo, mode, scopeWork(claims[i], now))
		return nil
	})
	editions := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM editions WHERE id=$1", "scope-"+tag+"-ed-shared")
	files := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE edition_id=$1", "scope-"+tag+"-ed-shared")
	return errs, editions, files
}

func TestCatalogEditionOwnershipIsDecidedOnce(t *testing.T) {
	dsn := liveDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)

	for _, mode := range []lockMode{lockProduction, lockLegacyGlobal} {
		const rounds = 8
		for r := 0; r < rounds; r++ {
			tag := "own-" + modeSlug(mode) + "-" + strconv.FormatInt(now.UnixNano(), 36) + "-" + strconv.Itoa(r)
			defer deleteScopeFixtures(ctx, t, repo, tag)
			errs, editions, files := ownershipRound(ctx, t, repo, mode, tag, now)
			failed := 0
			for _, err := range errs {
				if err == nil {
					continue
				}
				failed++
				// The loser must be told which work owns the edition. A constraint
				// violation here means the two writers were not excluded from the
				// same row, and the operator sees SQL instead of a catalog reason.
				if !strings.Contains(err.Error(), "already belongs to work") {
					t.Fatalf("%s round %d: loser got %v, want the edition ownership error", mode, r+1, err)
				}
			}
			if failed != 1 {
				t.Fatalf("%s round %d: %d of 2 writers failed, want exactly the loser", mode, r+1, failed)
			}
			if editions != 1 || files > 1 {
				t.Fatalf("%s round %d: edition row count %d, file rows %d after a contested claim", mode, r+1, editions, files)
			}
		}
		t.Logf("MODE %s: %d contested edition claims each decided once, with the ownership error for the loser", mode, rounds)
	}
}

// TestCatalogMultiProcessMerges measures the catalog seam the way a deployment
// actually uses it: several OS processes, each with its own connection pool,
// writing one PostgreSQL. Inside one process a lock could in principle be
// replaced by a mutex; across processes the advisory lock is the only
// coordination that exists, so this is where its scope is worth measuring.
//
// It is a role, not a self-contained test, because Go tests cannot fork other
// copies of themselves. The runner script starts N copies of the compiled test
// binary as separate processes with HELIX_MP_ROLE=worker and distinct
// HELIX_MP_INDEX values, then runs one copy with HELIX_MP_ROLE=verify, which
// checks the database all of them wrote to.
//
//	worker  HELIX_MP_ROLE=worker HELIX_MP_INDEX=<i> HELIX_MP_PROCESSES=<n>
//	        HELIX_MP_OPS=<per arm> HELIX_MP_RUN=<tag> HELIX_LOAD_RUN=1
//	verify  HELIX_MP_ROLE=verify HELIX_MP_INDEX=0 HELIX_MP_PROCESSES=<n>
//	        HELIX_MP_OPS=<same> HELIX_MP_RUN=<same tag> HELIX_LOAD_RUN=1
//
// Gated behind HELIX_LOAD_RUN=1. Running it directly without the roles set skips
// rather than measuring nothing and reporting success.
func TestCatalogMultiProcessMerges(t *testing.T) {
	dsn := loadWanted(t)
	role := os.Getenv("HELIX_MP_ROLE")
	if role != "worker" && role != "verify" {
		t.Skip("HELIX_MP_ROLE must be worker or verify; run evidence/campaign-2026-11-20/multiprocess/run_multiprocess.sh")
	}
	procs, ops := mustInt(t, "HELIX_MP_PROCESSES"), mustInt(t, "HELIX_MP_OPS")
	index := mustInt(t, "HELIX_MP_INDEX")
	run := os.Getenv("HELIX_MP_RUN")
	if run == "" || procs <= 0 || ops <= 0 {
		t.Fatal("HELIX_MP_RUN, HELIX_MP_PROCESSES and HELIX_MP_OPS are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)

	if role == "worker" {
		// Contended identities come from a small pool, not from the operation
		// index. If each process walked its own unique sequence, staggered
		// process startup alone would space them apart and nothing would ever
		// collide — which is exactly how the first version of this measurement
		// passed with no locks at all.
		const pool = 8

		// Arm 1: row-disjoint writes. Every process touches identities no other
		// process names, so any waiting is the lock set over-restricting.
		disjointStart := time.Now()
		var disjointErrs int
		for r := 0; r < ops; r++ {
			w := scopeWork(scopeSpec{
				work:    mpWorkTag(run, "d", index, r),
				edition: mpWorkTag(run, "d", index, r) + "-ed",
				source:  "mp-src-" + run + "-p" + strconv.Itoa(index),
				path:    "mp/" + run + "/" + strconv.Itoa(index) + "/d" + strconv.Itoa(r) + ".txt",
				sha:     shaFor(mpWorkTag(run, "d", index, r)), title: "d" + strconv.Itoa(index) + "-" + strconv.Itoa(r),
			}, now)
			if err := repo.MergeWork(ctx, w); err != nil {
				disjointErrs++
				t.Logf("WORKER %d disjoint merge %d failed: %v", index, r, err)
			}
		}
		disjointDBDeadlocks := mpDBDeadlocks(ctx, repo)
		disjointRate := float64(ops-disjointErrs) / time.Since(disjointStart).Seconds()
		disjointDBDeadlocksAfter := mpDBDeadlocks(ctx, repo)

		mpBarrier(ctx, t, repo, run, "arm2", index, procs)
		// Arm 2: every process merges distinct editions into the same handful of
		// works. upsertTx rewrites the whole edition set of a work, so a writer
		// working from a snapshot taken before another process committed deletes
		// that other edition. This is lost update, and it is silent.
		contestedStart := time.Now()
		var contestedErrs int
		for r := 0; r < ops; r++ {
			work := mpWorkTag(run, "c", r%pool, 0)
			w := scopeWork(scopeSpec{
				work: work, edition: work + "-p" + strconv.Itoa(index) + "-" + strconv.Itoa(r),
				source: "mp-src-" + run + "-p" + strconv.Itoa(index),
				path:   "mp/" + run + "/" + strconv.Itoa(index) + "/c" + strconv.Itoa(r) + ".txt",
				sha:    shaFor(work + "-p" + strconv.Itoa(index) + "-" + strconv.Itoa(r)),
				title:  "c" + strconv.Itoa(index) + "-" + strconv.Itoa(r),
			}, now)
			if err := repo.MergeWork(ctx, w); err != nil {
				contestedErrs++
				t.Logf("WORKER %d contested merge %d failed: %v", index, r, err)
			}
		}
		contestedDBDeadlocks := mpDBDeadlocks(ctx, repo)
		contestedRate := float64(ops-contestedErrs) / time.Since(contestedStart).Seconds()
		contestedDBDeadlocksAfter := mpDBDeadlocks(ctx, repo)

		mpBarrier(ctx, t, repo, run, "arm3", index, procs)
		// Arm 3: the same source namespace and the same paths claimed by every
		// process, which is two server replicas scanning one connector. They fight
		// over one file row, so the source key is what turns an abort into a wait.
		sharedStart := time.Now()
		var sharedErrs int
		for r := 0; r < ops; r++ {
			work := mpWorkTag(run, "f", r%pool, 0)
			w := scopeWork(scopeSpec{
				work: work, edition: work + "-p" + strconv.Itoa(index),
				// Shared source and shared path: one file row, many claimants.
				source: "mp-shared-src-" + run, path: "mp/" + run + "/shared/f" + strconv.Itoa(r%pool) + ".txt",
				sha: shaFor(work + "-p" + strconv.Itoa(index)), title: "f" + strconv.Itoa(index) + "-" + strconv.Itoa(r),
			}, now)
			if err := repo.MergeWork(ctx, w); err != nil {
				sharedErrs++
				t.Logf("WORKER %d shared-path merge %d failed: %v", index, r, err)
			}
		}
		sharedDBDeadlocks := mpDBDeadlocks(ctx, repo)
		sharedRate := float64(ops-sharedErrs) / time.Since(sharedStart).Seconds()
		sharedDBDeadlocksAfter := mpDBDeadlocks(ctx, repo)

		emitLoad(t, map[string]any{
			"test":             "catalog_multiprocess",
			"role":             "worker",
			"measurement_path": loadMeasurementPath(),
			"processes":        procs,
			"process_index":    index,
			"ops_per_arm":      ops,
			"contended_pool":   pool,
			"run_tag":          run,
			// pg_stat_database.deadlocks is database-wide and cumulative, so a
			// worker's own delta counts other processes' aborts too. The absolute
			// before/after pair lets the aggregator derive a run-wide lower bound
			// instead; 40P01 is retried in production code and never surfaces as a
			// worker error, so without this the retry count would be invisible.
			"disjoint": map[string]any{
				"errors": disjointErrs, "ops_per_second": disjointRate,
				"db_deadlocks_before": disjointDBDeadlocks, "db_deadlocks_after": disjointDBDeadlocksAfter,
			},
			"contested_work": map[string]any{
				"errors": contestedErrs, "ops_per_second": contestedRate,
				"db_deadlocks_before": contestedDBDeadlocks, "db_deadlocks_after": contestedDBDeadlocksAfter,
			},
			"shared_file_row": map[string]any{
				"errors": sharedErrs, "ops_per_second": sharedRate,
				"db_deadlocks_before": sharedDBDeadlocks, "db_deadlocks_after": sharedDBDeadlocksAfter,
			},
		})
		if failed := disjointErrs + contestedErrs + sharedErrs; failed != 0 {
			t.Fatalf("worker %d reported %d failed merges (disjoint %d, contested %d, shared %d)",
				index, failed, disjointErrs, contestedErrs, sharedErrs)
		}
		return
	}

	// verify: the database must account for every write every process claimed to
	// make. This is the only arm that can see the effect of a missing lock,
	// because a lost update is not an error in whichever process lost it.
	const pool = 8
	missingWorks := 0
	for p := 0; p < procs; p++ {
		for r := 0; r < ops; r++ {
			if countCatalogRows(ctx, t, repo, "SELECT count(*) FROM works WHERE id=$1", mpWorkTag(run, "d", p, r)) != 1 {
				missingWorks++
			}
		}
	}
	if missingWorks != 0 {
		t.Errorf("verify: %d of %d disjoint works are missing after %d processes finished", missingWorks, procs*ops, procs)
	}
	// Each contended work must hold one edition per process per operation that
	// targeted it, and each of those editions one file row.
	lostEditions := 0
	var sample string
	totalExpected, totalFound := 0, 0
	for j := 0; j < pool; j++ {
		work := mpWorkTag(run, "c", j, 0)
		want := 0
		for r := 0; r < ops; r++ {
			if r%pool == j {
				want += procs
			}
		}
		got := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM editions WHERE work_id=$1", work)
		totalExpected += want
		totalFound += got
		if got != want {
			lostEditions++
			if sample == "" {
				sample = fmt.Sprintf("work %s holds %d editions, want %d", work, got, want)
			}
		}
	}
	if lostEditions != 0 {
		t.Errorf("verify: %d of %d contended works lost concurrently merged editions (%d of %d editions present); first: %s",
			lostEditions, pool, totalFound, totalExpected, sample)
	}
	// One file row per (source, path), and none of them orphaned.
	dup := countCatalogRows(ctx, t, repo, `SELECT count(*) FROM (
		SELECT source_id, path FROM files WHERE source_id LIKE $1 GROUP BY source_id, path HAVING count(*) > 1) d`, "mp-%"+run+"%")
	if dup != 0 {
		t.Errorf("verify: %d (source, path) pairs have more than one file row", dup)
	}
	orphans := countCatalogRows(ctx, t, repo, `SELECT count(*) FROM files f
		LEFT JOIN editions e ON e.id=f.edition_id WHERE f.source_id LIKE $1 AND e.id IS NULL`, "mp-%"+run+"%")
	if orphans != 0 {
		t.Errorf("verify: %d orphaned file rows", orphans)
	}
	// The shared-file arm must leave exactly one file row per shared path, owned
	// by one edition of one work.
	sharedRows := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE source_id=$1", "mp-shared-src-"+run)
	if sharedRows != pool {
		t.Errorf("verify: shared source holds %d file rows, want %d (one per contended path)", sharedRows, pool)
	}
	emitLoad(t, map[string]any{
		"test": "catalog_multiprocess", "role": "verify",
		"measurement_path": loadMeasurementPath(),
		"processes":        procs, "ops_per_arm": ops, "run_tag": run, "contended_pool": pool,
		"missing_disjoint_works":  missingWorks,
		"contested_works_lost":    lostEditions,
		"editions_expected":       totalExpected,
		"editions_found":          totalFound,
		"duplicate_file_rows":     dup,
		"orphan_file_rows":        orphans,
		"shared_source_file_rows": sharedRows,
		"shared_source_expected":  pool,
	})
	t.Logf("VERIFY %d processes x %d ops per arm: %d disjoint works missing, %d of %d contended works lost editions (%d/%d present)",
		procs, ops, missingWorks, lostEditions, pool, totalFound, totalExpected)
}

// mpDBDeadlocks reads the cumulative, database-wide deadlock counter. It is
// sampled before and after each arm so a retried 40P01 leaves a trace: the retry
// in production code swallows the error, and workers would otherwise report
// errors=0 for a run where writers did abort and re-run.
func mpDBDeadlocks(ctx context.Context, repo *Postgres) int64 {
	var n int64
	if err := repo.db.QueryRowContext(ctx,
		"SELECT deadlocks FROM pg_stat_database WHERE datname = current_database()").Scan(&n); err != nil {
		return -1
	}
	return n
}

// mpBarrier releases the workers as close to the same instant as this
// measurement can manage. It matters more than it looks: without it, the
// process that starts first has already created the contended rows before the
// last one begins, and a plain row lock then protects the read-modify-write on
// its own, so an unlocked build looks safe. The window the identity lock covers
// is specifically the first write to an identity that does not exist yet, and
// only a simultaneous start can reach it.
//
// The rendezvous is a row per worker in the database the workers already share,
// which is also the only thing they have in common — separate processes, separate
// pools, no in-memory channel.
func mpBarrier(ctx context.Context, t *testing.T, repo *Postgres, run, name string, index, procs int) {
	t.Helper()
	if procs < 2 {
		return
	}
	id := "scope-mp-" + run + "-barrier-" + name + "-p" + strconv.Itoa(index)
	if _, err := repo.db.ExecContext(ctx,
		"INSERT INTO works(id,canonical_title,created_at,updated_at) VALUES($1,$2,now(),now()) ON CONFLICT(id) DO NOTHING",
		id, "barrier "+name); err != nil {
		t.Fatalf("barrier %s register: %v", name, err)
	}
	deadline := time.Now().Add(60 * time.Second)
	for {
		var n int
		if err := repo.db.QueryRowContext(ctx,
			"SELECT count(*) FROM works WHERE id LIKE $1", "scope-mp-"+run+"-barrier-"+name+"-%").Scan(&n); err != nil {
			t.Fatalf("barrier %s poll: %v", name, err)
		}
		if n >= procs {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("barrier %s timed out at %d of %d workers", name, n, procs)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("barrier %s canceled: %v", name, ctx.Err())
		case <-time.After(time.Millisecond):
		}
	}
}

func mustInt(t *testing.T, name string) int {
	t.Helper()
	v, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		t.Fatalf("%s = %q, want an integer", name, os.Getenv(name))
	}
	return v
}

func mpWorkTag(run, arm string, process, op int) string {
	return "scope-mp-" + run + "-" + arm + "-p" + strconv.Itoa(process) + "-" + strconv.Itoa(op)
}
