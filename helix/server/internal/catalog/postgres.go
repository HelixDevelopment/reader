package catalog

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Postgres is the durable catalog repository. Source bytes never enter the
// database; only in-place file references and metadata provenance are stored.
type Postgres struct{ db *sql.DB }

func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("catalog: empty PostgreSQL DSN")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
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
		if _, err := tx.ExecContext(ctx, "INSERT INTO helix_schema_migrations(version) VALUES($1)", version); err != nil {
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

func (p *Postgres) UpsertWork(ctx context.Context, w Work) error {
	if err := validateWork(w); err != nil {
		return err
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(731452)"); err != nil {
		return err
	}
	if err := upsertTx(ctx, tx, w); err != nil {
		return err
	}
	return tx.Commit()
}
func (p *Postgres) MergeWork(ctx context.Context, incoming Work) error {
	if err := validateWork(incoming); err != nil {
		return err
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(731452)"); err != nil {
		return err
	}
	existing, err := getWork(ctx, tx, incoming.ID, true)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if err == nil {
		incoming = MergeWork(existing, incoming)
	}
	if err := upsertTx(ctx, tx, incoming); err != nil {
		return err
	}
	return tx.Commit()
}
func upsertTx(ctx context.Context, tx *sql.Tx, w Work) error {
	// A source/path file keeps its ID when its bytes change. The write lock
	// serializes cross-work re-homing so concurrent content-keyed scans cannot
	// leave duplicate file rows or orphan an emptied work.
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
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	// Reconciliation shares the catalog identity lock with merge/re-homing so
	// deletion and file movement cannot interleave inside PostgreSQL.
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(731452)"); err != nil {
		return 0, err
	}
	rows, err := tx.QueryContext(ctx, `DELETE FROM files AS f
		WHERE f.source_id=$1 AND NOT EXISTS (
			SELECT 1 FROM jsonb_array_elements_text($2::jsonb) AS present(path)
			WHERE present.path=f.path
		) RETURNING edition_id`, sourceID, presentJSON)
	if err != nil {
		return 0, err
	}
	rehomedEditions := map[string]struct{}{}
	removed := 0
	for rows.Next() {
		var editionID string
		if err := rows.Scan(&editionID); err != nil {
			rows.Close()
			return 0, err
		}
		rehomedEditions[editionID] = struct{}{}
		removed++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	for editionID := range rehomedEditions {
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
	if err := tx.Commit(); err != nil {
		return 0, err
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
	queryArgs := append(append([]any{}, args...), q.Limit, q.Offset)
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
	page := Page{Works: []Work{}, Total: total, Offset: q.Offset, Limit: q.Limit}
	for _, id := range ids {
		w, err := p.GetWork(ctx, id)
		if err != nil {
			return Page{}, err
		}
		page.Works = append(page.Works, w)
	}
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
