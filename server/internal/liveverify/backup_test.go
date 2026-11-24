package liveverify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/aimemory"
	"github.com/helixreader/helix/server/internal/catalog"
)

// fakeRunner stands in for psql/pg_dump/redis-cli/... ONLY to test the verifier's
// own sequencing and failure handling (gate-logic test, 11.4.289(C)(2)); it is not
// evidence that a backup of a real PostgreSQL/Redis restores.
type fakeRunner struct {
	mu       sync.Mutex
	calls    []string
	envs     map[string][]string
	rdbDir   string
	failOn   string // command name that fails
	noRedis  bool   // redis-server missing
	restored bool
}

func (f *fakeRunner) record(name string, args []string, env []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if f.envs == nil {
		f.envs = map[string][]string{}
	}
	f.envs[name] = env
	if name == f.failOn {
		return fmt.Errorf("%s: exit status 1", name)
	}
	return nil
}

func (f *fakeRunner) Run(env []string, _ time.Duration, _ string, name string, args ...string) error {
	if err := f.record(name, args, env); err != nil {
		return err
	}
	switch name {
	case "pg_dump":
		for _, a := range args {
			if strings.HasPrefix(a, "--file=") {
				return os.WriteFile(strings.TrimPrefix(a, "--file="), []byte("PGDMP"), 0o644)
			}
		}
	case "redis-server":
		f.restored = true
	case "redis-cli":
		// ping of the restore daemon port succeeds only once it was started
		if len(args) >= 4 && args[3] == "26379" && f.restored == false && args[len(args)-1] == "ping" {
			return os.ErrInvalid
		}
		if len(args) >= 4 && args[3] != "26379" && args[len(args)-1] == "ping" {
			return nil
		}
	}
	return nil
}

func (f *fakeRunner) Output(_ time.Duration, name string, args ...string) (string, error) {
	if err := f.record(name, args, nil); err != nil {
		return "", err
	}
	if len(args) > 0 && args[len(args)-1] == "dir" {
		return "dir\n" + f.rdbDir + "\n", nil
	}
	return "", nil
}

func (f *fakeRunner) OutputEnv(_ []string, t time.Duration, name string, args ...string) (string, error) {
	return f.Output(t, name, args...)
}

func (f *fakeRunner) LookPath(name string) bool { return !(name == "redis-server" && f.noRedis) }

// backupFixture wires a fake runner and two in-process servers; the restored
// server shares state with the source unless a mutant resets part of it.
func backupFixture(t *testing.T, mutate func(restored *inproc)) (BackupConfig, *fakeRunner) {
	dir := t.TempDir()
	rdbDir := filepath.Join(dir, "redisdata")
	os.MkdirAll(rdbDir, 0o755)
	os.WriteFile(filepath.Join(rdbDir, "dump.rdb"), []byte("REDIS0011"), 0o644)
	run := &fakeRunner{rdbDir: rdbDir}
	src := newInproc(t, nil)
	src.creds = [3]string{backupAdminToken, backupOPDSUser, backupOPDSPass}
	n := 0
	cfg := BackupConfig{
		Runner: run, EvidenceDir: filepath.Join(dir, "evidence"), WorkDir: filepath.Join(dir, "work", "backup-restore"), RedisAddr: "127.0.0.1:6379", User: "tester",
		StartServer: func(port int, db, redisURL, logName string) (string, Process, error) {
			n++
			l := src
			if n == 2 {
				l = &inproc{t: t, repo: src.repo, redis: src.redis, memory: src.memory, covers: t.TempDir(), creds: src.creds}
				if mutate != nil {
					mutate(l)
				}
			}
			return l.Start()
		},
	}
	return cfg, run
}

func TestBackupRestorePassesWithRealInProcessServers(t *testing.T) {
	cfg, run := backupFixture(t, nil)
	res := RunBackupRestore(cfg)
	if res.Status != "PASS" {
		t.Fatalf("expected PASS: %s %v", failedName(res), res.Checks)
	}
	want := []string{"source fixture creation", "source fixture scan accepted", "source fixture ingestion", "source catalog fixture", "source vector fixture", "source progress fixture",
		"PostgreSQL backup created", "PostgreSQL restore into isolated database", "Redis backup created", "isolated Redis restore started",
		"restored catalog verified through fresh server", "restored vector memory verified", "restored reading progress verified"}
	if len(res.Checks) != len(want) {
		t.Fatalf("%d checks, want %d: %v", len(res.Checks), len(want), res.Checks)
	}
	for i, w := range want {
		if res.Checks[i].Name != w {
			t.Fatalf("check %d = %q want %q", i, res.Checks[i].Name, w)
		}
	}
	// ordering: dump, createdb, restore happen in that order and cleanup drops the restore DB
	joined := strings.Join(run.calls, "\n")
	for _, seq := range [][2]string{{"pg_dump", "createdb"}, {"createdb", "pg_restore"}, {"redis-cli -h 127.0.0.1 -p 6379 SAVE", "redis-server"}} {
		if strings.Index(joined, seq[0]) < 0 || strings.Index(joined, seq[0]) > strings.Index(joined, seq[1]) {
			t.Fatalf("expected %q before %q in:\n%s", seq[0], seq[1], joined)
		}
	}
	if !strings.Contains(joined, "dropdb --if-exists helix_restore_test") {
		t.Fatal("restore database is not dropped on cleanup")
	}
}

func TestBackupRestoreMutantsAreEachRejected(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*inproc)
		fake   func(*fakeRunner)
		want   string
	}{
		{"restored-catalog-empty", func(l *inproc) { l.repo = catalog.NewMemory() }, nil, "restored catalog verified through fresh server"},
		{"restored-memory-lost", func(l *inproc) { l.memory = &memStore{recs: map[string]aimemory.Record{}} }, nil, "restored vector memory verified"},
		{"restored-progress-lost", func(l *inproc) { l.redis = miniredisFresh(t) }, nil, "restored reading progress verified"},
		{"pg-restore-fails", nil, func(f *fakeRunner) { f.failOn = "pg_restore" }, "pg_restore: exit status"},
		{"redis-server-missing", nil, func(f *fakeRunner) { f.noRedis = true }, "redis-server binary unavailable"},
		{"empty-redis-snapshot", nil, func(f *fakeRunner) { os.WriteFile(filepath.Join(f.rdbDir, "dump.rdb"), nil, 0o644) }, "Redis backup created"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			cfg, run := backupFixture(t, c.mutate)
			if c.fake != nil {
				c.fake(run)
			}
			res := RunBackupRestore(cfg)
			if res.Status != "FAIL" || res.ExitCode() != 1 {
				t.Fatalf("mutant %s not rejected: %s", c.name, res.Status)
			}
			got := failedName(res)
			if !strings.Contains(got, c.want) {
				t.Fatalf("mutant %s failed on %q, want %q", c.name, got, c.want)
			}
		})
	}
}

func TestBackupDSNAndEnvironmentDerivation(t *testing.T) {
	c := BackupConfig{User: "tester", TestDSN: "postgres://alice:p%40ss@db.example:5433/helix_test?sslmode=disable&host=/sock&user=bob"}
	if got := c.DSN("helix_restore_test"); !strings.Contains(got, "/helix_restore_test?") || !strings.Contains(got, "alice") || strings.Contains(got, "helix_test?") {
		t.Fatalf("DSN swap wrong: %s", got)
	}
	env := strings.Join(c.PGEnvironment("postgres"), "\n")
	for _, want := range []string{"PGDATABASE=postgres", "PGUSER=bob", "PGPASSWORD=p@ss", "PGHOST=/sock", "PGPORT=5433"} {
		if !strings.Contains(env, want) {
			t.Fatalf("missing %s in env", want)
		}
	}
	d := BackupConfig{User: "tester"}
	if d.DSN("x") != "postgres:///x?host=/var/run/postgresql&user=tester" {
		t.Fatalf("peer-auth default: %s", d.DSN("x"))
	}
	if env := strings.Join(d.PGEnvironment("x"), "\n"); !strings.Contains(env, "PGHOST=/var/run/postgresql") || !strings.Contains(env, "PGUSER=tester") {
		t.Fatalf("peer env: %s", env)
	}
}

func TestProgressKeyEncodingMatchesServerStore(t *testing.T) {
	if got := ProgressKey("backup-restore-user", "backup-restore-book"); got != "helix:v1:progress:YmFja3VwLXJlc3RvcmUtdXNlcg:YmFja3VwLXJlc3RvcmUtYm9vaw" {
		t.Fatalf("key %s", got)
	}
}

func TestRemoveTreeRefusesShallowAndRelativePaths(t *testing.T) {
	for _, p := range []string{"", "/", "/home", "/home/x", "relative/dir", os.Getenv("HOME")} {
		if removeTree(p) == nil {
			t.Fatalf("removeTree(%q) must refuse", p)
		}
	}
}

// An externally managed (containerised) Redis reports its in-container dir; the
// host-visible snapshot directory is then given explicitly.
func TestBackupRestoreUsesExplicitRedisDataDir(t *testing.T) {
	cfg, run := backupFixture(t, nil)
	cfg.RedisDataDir = run.rdbDir
	run.rdbDir = "/data-not-on-host"
	res := RunBackupRestore(cfg)
	if res.Status != "PASS" {
		t.Fatalf("explicit RedisDataDir must be used: %s %v", failedName(res), res.Checks)
	}
}
