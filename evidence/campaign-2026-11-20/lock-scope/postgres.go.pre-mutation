package catalog

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Postgres is the durable catalog repository. Source bytes never enter the
// database; only in-place file references and metadata provenance are stored.
type Postgres struct{ db *sql.DB }

// Option tunes a repository at construction. Pool limits are deployment inputs:
// they must be raised in step with PostgreSQL capacity, never blindly.
type Option func(*poolSettings)

type poolSettings struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
}

func defaultPoolSettings() poolSettings {
	return poolSettings{maxOpenConns: 20, maxIdleConns: 5, connMaxLifetime: 30 * time.Minute}
}

// WithPool sets connection limits. A non-positive maxOpenConns, an idle count
// above the open count, or a non-positive lifetime is rejected at construction.
func WithPool(maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) Option {
	return func(s *poolSettings) {
		s.maxOpenConns, s.maxIdleConns, s.connMaxLifetime = maxOpenConns, maxIdleConns, connMaxLifetime
	}
}

// validate rejects pool settings database/sql would silently reinterpret. It is
// a method instead of inline checks because construction also fails on an
// unreachable DSN: a test asserting only that NewPostgres returned an error
// would keep passing with these checks deleted.
func (s poolSettings) validate() error {
	if s.maxOpenConns < 1 {
		return fmt.Errorf("catalog: MaxOpenConns must be at least 1")
	}
	if s.maxIdleConns < 0 || s.maxIdleConns > s.maxOpenConns {
		return fmt.Errorf("catalog: MaxIdleConns %d outside 0..MaxOpenConns %d", s.maxIdleConns, s.maxOpenConns)
	}
	if s.connMaxLifetime <= 0 {
		return fmt.Errorf("catalog: ConnMaxLifetime must be positive")
	}
	return nil
}

func NewPostgres(ctx context.Context, dsn string, options ...Option) (*Postgres, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("catalog: empty PostgreSQL DSN")
	}
	settings := defaultPoolSettings()
	for _, option := range options {
		option(&settings)
	}
	if err := settings.validate(); err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(settings.maxOpenConns)
	db.SetMaxIdleConns(settings.maxIdleConns)
	db.SetConnMaxLifetime(settings.connMaxLifetime)
	p := &Postgres{db: db}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := p.Migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return p, nil
}
func (p *Postgres) Migrate(ctx context.Context) error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(731451)"); err != nil {
		return fmt.Errorf("catalog migration lock: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS helix_schema_migrations(version INTEGER PRIMARY KEY,applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("catalog migration table: %w", err)
	}
	seen := map[int]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		version, parseErr := strconv.Atoi(prefix)
		if !ok || parseErr != nil || version <= 0 || seen[version] {
			return fmt.Errorf("catalog: invalid or duplicate migration %s", entry.Name())
		}
		seen[version] = true
		var applied bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM helix_schema_migrations WHERE version=$1)", version).Scan(&applied); err != nil {
			return fmt.Errorf("catalog migration version %d: %w", version, err)
		}
		if applied {
			continue
		}
		data, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("catalog migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO helix_schema_migrations(version) VALUES($1) ON CONFLICT (version) DO NOTHING", version); err != nil {
			return fmt.Errorf("catalog migration record %s: %w", entry.Name(), err)
		}
	}
	return tx.Commit()
}
func (p *Postgres) Close() error { return p.db.Close() }
func validateWork(w Work) error {
	if strings.TrimSpace(w.ID) == "" || len(w.Editions) == 0 {
		return fmt.Errorf("catalog: incomplete work")
	}
	for _, e := range w.Editions {
		if e.ID == "" || strings.TrimSpace(e.Title) == "" {
			return fmt.Errorf("catalog: incomplete edition")
		}
		switch e.State {
		case Pending, Identified, Partial, NeedsReview, Failed:
		default:
			return fmt.Errorf("catalog: invalid identification state %q", e.State)
		}
		for _, f := range e.Files {
			if f.ID == "" || len(f.SHA256) != 64 {
				return fmt.Errorf("catalog: incomplete file identity")
			}
		}
	}
	return nil
}
func stringsJSON(values []string) ([]byte, error) {
	if values == nil {
		values = []string{}
	}
	return json.Marshal(values)
}

type querier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// identityLockSpace is the advisory-lock class for catalog identity writes.
// It is passed as the first argument of the two-argument pg_advisory_xact_lock
// form, so these locks share neither key space with the migration (731451) or
// AI memory (731453) locks.
const identityLockSpace = 731452

// identityLockKeys lists every catalog identity one work write can touch:
// the work row, each edition row, and each source namespace whose file rows it
// claims or moves. All keys are known before the transaction begins, so taking
// them in a deterministic order gives concurrent writers a shared acquisition
// order and therefore no deadlock among themselves.
//
// The work key comes first, then the remaining keys sorted. Works and editions
// are keyed by content and files by source plus path, so two scans of different
// books in different sources have no shared row and no reason to wait for each
// other. Two scans of the same bytes from different sources collide on the work
// and edition keys and are serialized as a single global lock would have been.
func identityLockKeys(w Work) []string {
	keys := make(map[string]struct{}, 1+len(w.Editions))
	for _, e := range w.Editions {
		keys["e:"+e.ID] = struct{}{}
		for _, f := range e.Files {
			keys["s:"+f.SourceID] = struct{}{}
		}
	}
	out := make([]string, 0, len(keys)+1)
	out = append(out, "w:"+w.ID)
	rest := make([]string, 0, len(keys))
	for k := range keys {
		rest = append(rest, k)
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// lockIdentities takes the transaction-scoped advisory locks for keys. Locks
// already held by this transaction are returned immediately, so re-running the
// acquisition over a merged key set only adds the newly discovered locks.
// hashtext collisions serialize unrelated writes; distinct keys never share a
// lock.
func lockIdentities(ctx context.Context, tx *sql.Tx, keys []string) error {
	for _, key := range keys {
		if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1, hashtext($2))", identityLockSpace, key); err != nil {
			return fmt.Errorf("catalog: identity lock %q: %w", key, err)
		}
	}
	return nil
}

// transientConcurrencyCodes are the PostgreSQL failures that mean "the row you
// wanted moved under you", not "your request is wrong". Every retried operation
// is an idempotent, content-keyed write inside its own transaction, so a retry
// after rollback is safe and cheaper than surfacing a scan failure that an
// operator would otherwise have to re-run by hand.
var transientConcurrencyCodes = map[string]bool{
	"40001": true, // serialization_failure
	"40P01": true, // deadlock_detected
	"23503": true, // foreign_key_violation: a prune raced a concurrent re-home
}

func transientConcurrency(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && transientConcurrencyCodes[pgErr.Code]
}

// inTx runs fn in its own transaction and retries while it fails for a
// transient concurrency reason. fn is called again from a fresh transaction, so
// every retried operation has to be idempotent; the catalog writes are, because
// their identity comes from content and source paths rather than from a
// sequence.
func inTx(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) error {
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * 15 * time.Millisecond):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		err = oneTx(ctx, db, fn)
		if err == nil || !transientConcurrency(err) {
			return err
		}
	}
	return err
}

func oneTx(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *Postgres) UpsertWork(ctx context.Context, w Work) error {
	if err := validateWork(w); err != nil {
		return err
	}
	keys := identityLockKeys(w)
	return inTx(ctx, p.db, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockIdentities(ctx, tx, keys); err != nil {
			return err
		}
		return upsertTx(ctx, tx, w)
	})
}
func (p *Postgres) MergeWork(ctx context.Context, incoming Work) error {
	if err := validateWork(incoming); err != nil {
		return err
	}
	keys := identityLockKeys(incoming)
	return inTx(ctx, p.db, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockIdentities(ctx, tx, keys); err != nil {
			return err
		}
		existing, err := getWork(ctx, tx, incoming.ID, true)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err == nil {
			// Editions the stored work carries but this scan does not are protected
			// by the work key: upsertTx rewrites every edition of one work, and no
			// other writer can reach those rows without sharing that key or the key
			// of the source owning the file. No further locks are taken after the
			// read, so advisory locks always precede row locks in this transaction.
			incoming = MergeWork(existing, incoming)
		}
		return upsertTx(ctx, tx, incoming)
	})
}
func upsertTx(ctx context.Context, tx *sql.Tx, w Work) error {
	// A source/path file keeps its ID when its bytes change. The identity locks
	// taken by the caller serialize cross-work re-homing, so concurrent
	// content-keyed scans cannot leave duplicate file rows or orphan an
	// emptied work.
	rehomedEditions := map[string]struct{}{}
	subjects, err := stringsJSON(w.Subjects)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO works(id,canonical_title,series,series_index,subjects,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7)
	ON CONFLICT(id) DO UPDATE SET canonical_title=EXCLUDED.canonical_title,series=EXCLUDED.series,series_index=EXCLUDED.series_index,subjects=EXCLUDED.subjects,updated_at=EXCLUDED.updated_at`, w.ID, w.CanonicalTitle, w.Series, w.SeriesIndex, subjects, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM editions WHERE work_id=$1", w.ID); err != nil {
		return err
	}
	for _, e := range w.Editions {
		var owner string
		err := tx.QueryRowContext(ctx, "SELECT work_id FROM editions WHERE id=$1", e.ID).Scan(&owner)
		if err == nil && owner != w.ID {
			return fmt.Errorf("catalog: edition %s already belongs to work %s", e.ID, owner)
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		authors, err := stringsJSON(e.Authors)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO editions(id,work_id,title,subtitle,authors,publisher,published,language,description,cover_href,state,confidence,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, e.ID, w.ID, e.Title, e.Subtitle, authors, e.Publisher, e.Published, e.Language, e.Description, e.CoverHREF, e.State, e.Confidence, e.UpdatedAt); err != nil {
			return err
		}
		for _, id := range e.Identifiers {
			// Exact duplicate identifiers are provenance replays, not errors.
			if _, err := tx.ExecContext(ctx, "INSERT INTO identifiers(edition_id,scheme,value,source,confidence) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", e.ID, id.Scheme, id.Value, id.Source, id.Confidence); err != nil {
				return err
			}
		}
		for _, f := range e.Files {
			// Match both the current ID and exact source/path. The latter moves
			// catalogs written by the former case-folded ID algorithm without
			// colliding with case-distinct paths.
			rows, err := tx.QueryContext(ctx, `DELETE FROM files
				WHERE id=$1 OR (source_id=$2 AND path=$3) RETURNING edition_id`, f.ID, f.SourceID, f.Path)
			if err != nil {
				return err
			}
			for rows.Next() {
				var previousEdition string
				if err := rows.Scan(&previousEdition); err != nil {
					rows.Close()
					return err
				}
				rehomedEditions[previousEdition] = struct{}{}
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return err
			}
			if err := rows.Close(); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO files(id,edition_id,source_id,path,format,mime_type,size,sha256,protected,drm_scheme,license_href,modified_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, f.ID, e.ID, f.SourceID, f.Path, f.Format, f.MIMEType, f.Size, f.SHA256, f.Protected, f.DRMScheme, f.LicenseHREF, f.ModifiedAt); err != nil {
				return err
			}
		}
		for _, ev := range e.Evidence {
			if _, err := tx.ExecContext(ctx, "INSERT INTO metadata_evidence(edition_id,field,provider,confidence,observed_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", e.ID, ev.Field, ev.Provider, ev.Confidence, ev.ObservedAt); err != nil {
				return err
			}
		}
	}
	for editionID := range rehomedEditions {
		var workID string
		err := tx.QueryRowContext(ctx, `DELETE FROM editions e WHERE e.id=$1
			AND NOT EXISTS (SELECT 1 FROM files f WHERE f.edition_id=e.id) RETURNING work_id`, editionID).Scan(&workID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM works w WHERE w.id=$1
			AND NOT EXISTS (SELECT 1 FROM editions e WHERE e.work_id=w.id)`, workID); err != nil {
			return err
		}
	}
	return nil
}
func (p *Postgres) ReconcileSource(ctx context.Context, sourceID string, present []string) (int, error) {
	if strings.TrimSpace(sourceID) == "" {
		return 0, fmt.Errorf("catalog: source id is required for reconciliation")
	}
	if present == nil {
		present = []string{}
	}
	presentJSON, err := json.Marshal(present)
	if err != nil {
		return 0, err
	}
	removed := 0
	err = inTx(ctx, p.db, func(ctx context.Context, tx *sql.Tx) error {
		removed = 0
		// Reconciliation owns one source namespace: it removes and moves file rows
		// of that source, then prunes the editions and works they left behind. Merge
		// writes touching the same source take the same key, so a file cannot be
		// pruned mid-move. Cross-source writers share a row only through a content
		// keyed edition, and the prune is guarded by its own emptiness test plus the
		// foreign key on files.edition_id, so the worst remaining interleaving is a
		// transient error the retry turns into a second attempt.
		if err := lockIdentities(ctx, tx, []string{"s:" + sourceID}); err != nil {
			return err
		}
		count, err := reconcileTx(ctx, tx, sourceID, presentJSON)
		if err != nil {
			return err
		}
		removed = count
		return nil
	})
	if err != nil {
		return 0, err
	}
	return removed, nil
}

// reconcileTx removes the file rows of one source that a listing no longer
// reports, then prunes the editions and works those rows left empty. It is
// separated from its transaction and lock acquisition so the contention tests
// can run the identical statement sequence under different locking disciplines.
func reconcileTx(ctx context.Context, tx *sql.Tx, sourceID string, presentJSON []byte) (int, error) {
	rows, err := tx.QueryContext(ctx, `DELETE FROM files AS f
			WHERE f.source_id=$1 AND NOT EXISTS (
				SELECT 1 FROM jsonb_array_elements_text($2::jsonb) AS present(path)
				WHERE present.path=f.path
			) RETURNING edition_id`, sourceID, presentJSON)
	if err != nil {
		return 0, err
	}
	rehomedEditions := []string{}
	removed := 0
	for rows.Next() {
		var editionID string
		if err := rows.Scan(&editionID); err != nil {
			rows.Close()
			return 0, err
		}
		rehomedEditions = append(rehomedEditions, editionID)
		removed++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	// Sorted so two reconciliations that overlap on editions prune in the
	// same order instead of circling.
	sort.Strings(rehomedEditions)
	for _, editionID := range rehomedEditions {
		var workID string
		err := tx.QueryRowContext(ctx, `DELETE FROM editions e WHERE e.id=$1
			AND NOT EXISTS (SELECT 1 FROM files f WHERE f.edition_id=e.id) RETURNING work_id`, editionID).Scan(&workID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM works w WHERE w.id=$1
			AND NOT EXISTS (SELECT 1 FROM editions e WHERE e.work_id=w.id)`, workID); err != nil {
			return 0, err
		}
	}
	return removed, nil
}

func (p *Postgres) GetWork(ctx context.Context, id string) (Work, error) {
	return getWork(ctx, p.db, id, false)
}
func getWork(ctx context.Context, q querier, id string, lock bool) (Work, error) {
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	var w Work
	var subjects []byte
	err := q.QueryRowContext(ctx, "SELECT id,canonical_title,series,series_index,subjects,created_at,updated_at FROM works WHERE id=$1"+suffix, id).Scan(&w.ID, &w.CanonicalTitle, &w.Series, &w.SeriesIndex, &subjects, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Work{}, ErrNotFound
	}
	if err != nil {
		return Work{}, err
	}
	if err := json.Unmarshal(subjects, &w.Subjects); err != nil {
		return Work{}, err
	}
	rows, err := q.QueryContext(ctx, "SELECT id,title,subtitle,authors,publisher,published,language,description,cover_href,state,confidence,updated_at FROM editions WHERE work_id=$1 ORDER BY id", id)
	if err != nil {
		return Work{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var e Edition
		var authors []byte
		if err := rows.Scan(&e.ID, &e.Title, &e.Subtitle, &authors, &e.Publisher, &e.Published, &e.Language, &e.Description, &e.CoverHREF, &e.State, &e.Confidence, &e.UpdatedAt); err != nil {
			return Work{}, err
		}
		if err := json.Unmarshal(authors, &e.Authors); err != nil {
			return Work{}, err
		}
		w.Editions = append(w.Editions, e)
	}
	if err := rows.Err(); err != nil {
		return Work{}, err
	}
	for i := range w.Editions {
		if err := loadEditionChildren(ctx, q, &w.Editions[i]); err != nil {
			return Work{}, err
		}
	}
	return w, nil
}
func loadEditionChildren(ctx context.Context, q querier, e *Edition) error {
	rows, err := q.QueryContext(ctx, "SELECT scheme,value,source,confidence FROM identifiers WHERE edition_id=$1 ORDER BY scheme,value,source", e.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id Identifier
		if err := rows.Scan(&id.Scheme, &id.Value, &id.Source, &id.Confidence); err != nil {
			rows.Close()
			return err
		}
		e.Identifiers = append(e.Identifiers, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = q.QueryContext(ctx, "SELECT id,source_id,path,format,mime_type,size,sha256,protected,drm_scheme,license_href,modified_at FROM files WHERE edition_id=$1 ORDER BY id", e.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.SourceID, &f.Path, &f.Format, &f.MIMEType, &f.Size, &f.SHA256, &f.Protected, &f.DRMScheme, &f.LicenseHREF, &f.ModifiedAt); err != nil {
			rows.Close()
			return err
		}
		e.Files = append(e.Files, f)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = q.QueryContext(ctx, "SELECT field,provider,confidence,observed_at FROM metadata_evidence WHERE edition_id=$1 ORDER BY field,provider,observed_at", e.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var ev MetadataEvidence
		if err := rows.Scan(&ev.Field, &ev.Provider, &ev.Confidence, &ev.ObservedAt); err != nil {
			return err
		}
		e.Evidence = append(e.Evidence, ev)
	}
	return rows.Err()
}
func (p *Postgres) FindEdition(ctx context.Context, id string) (Work, Edition, error) {
	var workID string
	err := p.db.QueryRowContext(ctx, "SELECT work_id FROM editions WHERE id=$1", id).Scan(&workID)
	if errors.Is(err, sql.ErrNoRows) {
		return Work{}, Edition{}, ErrNotFound
	}
	if err != nil {
		return Work{}, Edition{}, err
	}
	w, err := p.GetWork(ctx, workID)
	if err != nil {
		return Work{}, Edition{}, err
	}
	for _, e := range w.Editions {
		if e.ID == id {
			return w, e, nil
		}
	}
	return Work{}, Edition{}, ErrNotFound
}
func (p *Postgres) FindFile(ctx context.Context, id string) (Work, Edition, File, error) {
	var editionID string
	var f File
	err := p.db.QueryRowContext(ctx, "SELECT id,edition_id,source_id,path,format,mime_type,size,sha256,protected,drm_scheme,license_href,modified_at FROM files WHERE id=$1", id).Scan(&f.ID, &editionID, &f.SourceID, &f.Path, &f.Format, &f.MIMEType, &f.Size, &f.SHA256, &f.Protected, &f.DRMScheme, &f.LicenseHREF, &f.ModifiedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Work{}, Edition{}, File{}, ErrNotFound
	}
	if err != nil {
		return Work{}, Edition{}, File{}, err
	}
	var workID string
	if err := p.db.QueryRowContext(ctx, "SELECT work_id FROM editions WHERE id=$1", editionID).Scan(&workID); err != nil {
		return Work{}, Edition{}, File{}, err
	}
	w, err := p.GetWork(ctx, workID)
	if err != nil {
		return Work{}, Edition{}, File{}, err
	}
	for _, e := range w.Editions {
		if e.ID == editionID {
			return w, e, f, nil
		}
	}
	return Work{}, Edition{}, File{}, ErrNotFound
}
func (p *Postgres) Search(ctx context.Context, q Query) (Page, error) {
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	where := []string{"TRUE"}
	args := []any{}
	add := func(condition string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(condition, len(args)))
	}
	if text := strings.TrimSpace(q.Text); text != "" {
		pattern := "%" + text + "%"
		add(`(w.canonical_title ILIKE $%d OR w.series ILIKE $%[1]d OR EXISTS(SELECT 1 FROM editions e WHERE e.work_id=w.id AND (e.title ILIKE $%[1]d OR e.publisher ILIKE $%[1]d OR e.description ILIKE $%[1]d OR e.authors::text ILIKE $%[1]d)) OR EXISTS(SELECT 1 FROM identifiers i JOIN editions e ON e.id=i.edition_id WHERE e.work_id=w.id AND i.value ILIKE $%[1]d))`, pattern)
	}
	if author := strings.TrimSpace(q.Author); author != "" {
		add(`EXISTS(SELECT 1 FROM editions e WHERE e.work_id=w.id AND e.authors::text ILIKE $%d)`, "%"+author+"%")
	}
	if q.State != "" {
		add(`EXISTS(SELECT 1 FROM editions e WHERE e.work_id=w.id AND e.state=$%d)`, string(q.State))
	}
	clause := strings.Join(where, " AND ")
	var total int
	if err := p.db.QueryRowContext(ctx, "SELECT count(*) FROM works w WHERE "+clause, args...).Scan(&total); err != nil {
		return Page{}, err
	}
	// The keyset predicate is written out rather than expressed as a tuple
	// comparison because the ordering mixes directions: updated_at descends while
	// id ascends, and (updated_at,id) < (x,y) would also require id to descend.
	offset := q.Offset
	if q.After != nil {
		args = append(args, q.After.UpdatedAt, q.After.ID)
		clause = fmt.Sprintf("(%s) AND (w.updated_at < $%d OR (w.updated_at = $%d AND w.id > $%d))", clause, len(args)-1, len(args)-1, len(args))
		offset = 0
	}
	// One row past the page is read so HasNext does not depend on the count: a
	// cursor client can stop paging without ever learning the total.
	queryArgs := append(append([]any{}, args...), q.Limit+1, offset)
	rows, err := p.db.QueryContext(ctx, fmt.Sprintf("SELECT w.id FROM works w WHERE %s ORDER BY w.updated_at DESC,w.id LIMIT $%d OFFSET $%d", clause, len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return Page{}, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return Page{}, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	hasMore := len(ids) > q.Limit
	if hasMore {
		ids = ids[:q.Limit]
	}
	page := Page{Total: total, Offset: q.Offset, Limit: q.Limit, HasNext: hasMore}
	works, err := p.getWorks(ctx, ids)
	if err != nil {
		return Page{}, err
	}
	page.Works = works
	return page, nil
}
func (p *Postgres) DeleteWork(ctx context.Context, id string) error {
	result, err := p.db.ExecContext(ctx, "DELETE FROM works WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
