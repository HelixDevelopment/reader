package liveverify

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	backupMainDB      = "helix_test"
	backupRestoreDB   = "helix_restore_test"
	RestoreRedisPort  = 26379
	backupAdminToken  = "backup-admin-token"
	backupOPDSUser    = "backup-reader"
	backupOPDSPass    = "backup-password"
	backupMemoryID    = "backup-restore-memory"
	backupProgressUsr = "backup-restore-user"
	backupProgressBk  = "backup-restore-book"
)

// Runner executes external commands (psql, pg_dump, redis-cli, ...). The real one
// shells out; the verifier's own logic tests use a recording one.
type Runner interface {
	Run(env []string, timeout time.Duration, stdin string, name string, args ...string) error
	Output(timeout time.Duration, name string, args ...string) (string, error)
	OutputEnv(env []string, timeout time.Duration, name string, args ...string) (string, error)
	LookPath(name string) bool
}

type ExecRunner struct{}

func (ExecRunner) Run(env []string, timeout time.Duration, stdin string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if env != nil {
		cmd.Env = env
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := runWithTimeout(cmd, timeout); err != nil {
		if tail := strings.TrimSpace(stderr.String()); tail != "" {
			if len(tail) > 300 {
				tail = tail[len(tail)-300:]
			}
			return fmt.Errorf("%s: %w: %s", name, err, tail)
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
func (r ExecRunner) Output(timeout time.Duration, name string, args ...string) (string, error) {
	return r.OutputEnv(nil, timeout, name, args...)
}
func (ExecRunner) OutputEnv(env []string, timeout time.Duration, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if env != nil {
		cmd.Env = env
	}
	var out strings.Builder
	cmd.Stdout = &out
	if err := runWithTimeout(cmd, timeout); err != nil {
		return out.String(), fmt.Errorf("%s: %w", name, err)
	}
	return out.String(), nil
}
func (ExecRunner) LookPath(name string) bool { _, err := exec.LookPath(name); return err == nil }

// BackupConfig configures the backup/restore rehearsal.
type BackupConfig struct {
	Runner      Runner
	EvidenceDir string
	WorkDir     string // scratch; wiped and recreated
	// TestDSN is HELIX_TEST_POSTGRES_DSN ("" = local peer-auth socket default).
	TestDSN string
	// RedisAddr is HELIX_TEST_REDIS_ADDR (host:port) of the source daemon.
	RedisAddr string
	// RedisDataDir is the host-visible directory holding the source Redis dump.rdb
	// (HELIX_TEST_REDIS_DATA_DIR); needed when Redis runs in a container and reports
	// its in-container dir. Empty = ask the running daemon (CONFIG GET dir).
	RedisDataDir string
	// StartServer starts a helix-server on port against database/redisURL, logging to logName.
	StartServer func(port int, database, redisURL, logName string) (base string, p Process, err error)
	// User is the OS user for the peer-auth defaults (default: current user).
	User string
}

func (c *BackupConfig) defaults() {
	if c.RedisAddr == "" {
		c.RedisAddr = "127.0.0.1:6379"
	}
	if c.User == "" {
		if u, err := user.Current(); err == nil {
			c.User = u.Username
		}
	}
}

func (c BackupConfig) redisHostPort() (string, string) {
	i := strings.LastIndex(c.RedisAddr, ":")
	return c.RedisAddr[:i], c.RedisAddr[i+1:]
}

// DSN derives the connection string for database from the test DSN by swapping
// the database name, or builds the peer-auth socket default.
func (c BackupConfig) DSN(database string) string {
	if c.TestDSN != "" {
		u, err := url.Parse(c.TestDSN)
		if err == nil {
			u.Path = "/" + database
			u.RawPath = ""
			return u.String()
		}
	}
	return fmt.Sprintf("postgres:///%s?host=/var/run/postgresql&user=%s", database, c.User)
}

// PGEnvironment is the environment for psql/pg_dump/createdb/dropdb/pg_restore so
// maintenance commands reach the same endpoint selected by the test DSN. Credentials
// travel via environment only and are never written to evidence.
func (c BackupConfig) PGEnvironment(database string) []string {
	over := map[string]string{"PGDATABASE": database}
	if c.TestDSN != "" {
		if u, err := url.Parse(c.TestDSN); err == nil {
			if u.User != nil {
				if name := u.User.Username(); name != "" {
					over["PGUSER"] = name
				}
				if pw, ok := u.User.Password(); ok && pw != "" {
					over["PGPASSWORD"] = pw
				}
			}
			if h := u.Hostname(); h != "" {
				over["PGHOST"] = h
			}
			if p := u.Port(); p != "" {
				over["PGPORT"] = p
			}
			for key, vals := range u.Query() {
				if len(vals) == 0 {
					continue
				}
				v := vals[len(vals)-1]
				switch key {
				case "host":
					over["PGHOST"] = v
				case "port":
					over["PGPORT"] = v
				case "user":
					over["PGUSER"] = v
				case "password":
					over["PGPASSWORD"] = v
				}
			}
		}
	} else {
		over["PGHOST"] = "/var/run/postgresql"
		over["PGUSER"] = c.User
	}
	return Environ(over)
}

func (c BackupConfig) redisCLI(port string) []string {
	if port == "" {
		h, p := c.redisHostPort()
		return []string{"-h", h, "-p", p}
	}
	return []string{"-h", "127.0.0.1", "-p", port}
}

// ProgressKey is the Redis key the server uses for one user's book progress.
func ProgressKey(userID, bookID string) string {
	part := func(v string) string { return base64.RawURLEncoding.EncodeToString([]byte(v)) }
	return "helix:v1:progress:" + part(userID) + ":" + part(bookID)
}

type backup struct {
	cfg    BackupConfig
	rec    *Recorder
	hc     *Client
	admin  map[string]string
	basic  *[2]string
	procs  []Process
	workID string
}

func (b *backup) redisReady(port string) bool {
	args := append(b.cfg.redisCLI(port), "ping")
	return b.cfg.Runner.Run(nil, 2*time.Second, "", "redis-cli", args...) == nil
}

func (b *backup) stopRestoreRedis() {
	p := strconv.Itoa(RestoreRedisPort)
	if b.redisReady(p) {
		_ = b.cfg.Runner.Run(nil, 10*time.Second, "", "redis-cli", append(b.cfg.redisCLI(p), "SHUTDOWN", "NOSAVE")...)
	}
}

func (b *backup) dropRestoreDB() {
	_ = b.cfg.Runner.Run(b.cfg.PGEnvironment("postgres"), 30*time.Second, "", "dropdb", "--if-exists", backupRestoreDB)
}

func (b *backup) cleanupMainData() error {
	env := b.cfg.PGEnvironment(backupMainDB)
	// SQL goes through stdin so psql interpolates :'work_id'; psql -c does not
	// interpolate variables, which would make the delete a silent syntax error.
	if b.workID != "" {
		if err := b.cfg.Runner.Run(env, 15*time.Second, "DELETE FROM works WHERE id=:'work_id';\n", "psql", "-d", backupMainDB, "-v", "ON_ERROR_STOP=1", "-v", "work_id="+b.workID); err != nil {
			return fmt.Errorf("delete work: %w", err)
		}
	}
	if err := b.cfg.Runner.Run(env, 15*time.Second, "", "psql", "-d", backupMainDB, "-v", "ON_ERROR_STOP=1", "-c", "DELETE FROM ai_memories WHERE id='"+backupMemoryID+"'"); err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return b.cfg.Runner.Run(nil, 10*time.Second, "", "redis-cli", append(b.cfg.redisCLI(""), "DEL", ProgressKey(backupProgressUsr, backupProgressBk))...)
}

// RunBackupRestore runs the rehearsal: fixtures through a source server, a
// PostgreSQL dump restored into an isolated database, a Redis snapshot loaded
// into an isolated daemon, and a fresh server over the restored pair.
func RunBackupRestore(cfg BackupConfig) Results {
	cfg.defaults()
	rec := &Recorder{}
	b := &backup{cfg: cfg, rec: rec, hc: NewClient(), admin: map[string]string{"Authorization": "Bearer " + backupAdminToken}, basic: &[2]string{backupOPDSUser, backupOPDSPass}}
	defer b.hc.Close()
	failed := false
	finish := func() Results {
		res := rec.Results(failed)
		if err := WriteResults(cfg.EvidenceDir, res); err != nil {
			res.Status = "FAIL"
		}
		return res
	}
	if err := os.MkdirAll(cfg.EvidenceDir, 0o755); err != nil {
		rec.Fail("evidence directory: " + err.Error())
		failed = true
		return finish()
	}
	library := filepath.Join(cfg.WorkDir, "library")
	failed = !rec.Guard(func() {
		must(removeTree(cfg.WorkDir))
		must(os.MkdirAll(library, 0o755))
		must(os.WriteFile(filepath.Join(library, "backup-restore-book.txt"), []byte("backup-restore-book\n\nDeterministic recovery fixture.\n"), 0o644))
		b.body(library)
	})
	for _, p := range b.procs {
		if err := p.Stop(); err != nil {
			failed = true
			rec.Fail("server cleanup: " + err.Error())
		}
	}
	func() {
		defer func() {
			if v := recover(); v != nil {
				failed = true
				rec.Fail("test-data cleanup: " + panicText(v))
			}
		}()
		b.stopRestoreRedis()
		b.dropRestoreDB()
		if err := b.cleanupMainData(); err != nil {
			failed = true
			rec.Fail("test-data cleanup: " + err.Error())
		}
	}()
	return finish()
}

// removeTree deletes dir only when checkRemovable (the constitution 11.4.287(B)
// protected-path semantics) allows it.
func removeTree(dir string) error {
	if err := checkRemovable(dir); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Clean(dir))
}
