package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMigrateFromEmptySchema proves catalog migration succeeds against a schema
// that holds no tables. The migration runner, not the .sql file, owns the
// helix_schema_migrations bookkeeping row; a version that recorded itself in
// both places aborted every cold start with a duplicate-key error while
// continuing to pass against an already-migrated database.
func TestMigrateFromEmptySchema(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	const schema = "helix_coldstart_probe"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
		t.Fatalf("prepare probe schema: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
			t.Errorf("drop probe schema: %v", err)
		}
	}()
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create probe schema (the test role needs schema creation rights): %v", err)
	}

	emptyDSN := dsn
	if strings.Contains(emptyDSN, "?") {
		emptyDSN += "&search_path=" + schema
	} else {
		emptyDSN += "?search_path=" + schema
	}

	repo, err := NewPostgres(ctx, emptyDSN)
	if err != nil {
		t.Fatalf("migration from empty schema rejected: %v", err)
	}
	defer repo.Close()

	// Repeated migration must be a no-op, never a second bookkeeping row.
	for i := 0; i < 2; i++ {
		if err := repo.Migrate(ctx); err != nil {
			t.Fatalf("repeated migration %d: %v", i+1, err)
		}
	}

	var versions []int
	rows, err := repo.db.QueryContext(ctx, "SELECT version FROM helix_schema_migrations ORDER BY version")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if len(versions) != 1 || versions[0] != 1 {
		t.Fatalf("bookkeeping rows = %v, want exactly [1]", versions)
	}

	tables, err := catalogTables(ctx, repo.db, schema)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"editions", "files", "helix_schema_migrations", "identifiers", "metadata_evidence", "works"}
	if strings.Join(tables, ",") != strings.Join(want, ",") {
		t.Fatalf("cold-start tables = %v, want %v", tables, want)
	}
}

// TestMigrationFilesDoNotRecordOwnVersion pins the ownership invariant behind
// the cold-start fix: Migrate writes helix_schema_migrations, and no embedded
// .sql file may. A file that records its own version collides with the runner
// inside the same transaction, which aborts every migration of an empty schema.
func TestMigrationFilesDoNotRecordOwnVersion(t *testing.T) {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		data, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		checked++
		normalized := strings.ToLower(strings.Join(strings.Fields(string(data)), " "))
		for _, pattern := range []string{
			"insert into helix_schema_migrations",
			"update helix_schema_migrations",
			"delete from helix_schema_migrations",
		} {
			if strings.Contains(normalized, pattern) {
				t.Errorf("%s writes the bookkeeping table (%s); Migrate owns that row", entry.Name(), pattern)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no embedded catalog migrations found; the invariant check is vacuous")
	}
}

func catalogTables(ctx context.Context, db *sql.DB, schema string) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT tablename FROM pg_tables WHERE schemaname=$1 ORDER BY tablename", schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no tables found in schema %s", schema)
	}
	return out, nil
}

func TestPoolSettingsValidation(t *testing.T) {
	for _, test := range []struct {
		name             string
		maxOpen, maxIdle int
		lifetime         time.Duration
		want             string
	}{
		{"defaults are valid", 20, 5, 30 * time.Minute, ""},
		{"zero idle is valid", 4, 0, time.Minute, ""},
		{"idle equal to open is valid", 4, 4, time.Minute, ""},
		{"zero max open", 0, 0, time.Minute, "MaxOpenConns"},
		{"negative max open", -1, 0, time.Minute, "MaxOpenConns"},
		{"idle above open", 4, 5, time.Minute, "MaxIdleConns"},
		{"negative idle", 4, -1, time.Minute, "MaxIdleConns"},
		{"zero lifetime", 4, 1, 0, "ConnMaxLifetime"},
		{"negative lifetime", 4, 1, -time.Second, "ConnMaxLifetime"},
	} {
		t.Run(test.name, func(t *testing.T) {
			settings := poolSettings{maxOpenConns: test.maxOpen, maxIdleConns: test.maxIdle, connMaxLifetime: test.lifetime}
			err := settings.validate()
			if test.want == "" {
				if err != nil {
					t.Fatalf("valid settings rejected: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("settings %+v accepted", settings)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error %q does not name %s", err, test.want)
			}
		})
	}
}

// TestNewPostgresRejectsPoolBeforeDialing pins the order the checks run in.
// Construction also fails on an unreachable DSN, so asserting only that an error
// came back proves nothing about pool validation — the earlier version of this
// test passed with the validation deleted. Asserting which error arrives is what
// gives it detection power.
func TestNewPostgresRejectsPoolBeforeDialing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, test := range []struct {
		name, want       string
		maxOpen, maxIdle int
		lifetime         time.Duration
	}{
		{"idle above open", "MaxIdleConns", 4, 9, time.Minute},
		{"zero max open", "MaxOpenConns", 0, 0, time.Minute},
		{"zero lifetime", "ConnMaxLifetime", 4, 1, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, err := NewPostgres(ctx, "postgres://unreachable.invalid:5432/none", WithPool(test.maxOpen, test.maxIdle, test.lifetime))
			if err == nil {
				repo.Close()
				t.Fatal("invalid pool accepted")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("rejected for the wrong reason: %v (want %s)", err, test.want)
			}
		})
	}
}

func TestWithPoolAppliesLimits(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn, WithPool(7, 2, 3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	if got := repo.db.Stats().MaxOpenConnections; got != 7 {
		t.Fatalf("MaxOpenConnections = %d, want 7", got)
	}
	defaulted, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer defaulted.Close()
	if got := defaulted.db.Stats().MaxOpenConnections; got != 20 {
		t.Fatalf("default MaxOpenConnections = %d, want 20", got)
	}
}
