package liveverify

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FindRepoRoot walks up from dir until it finds server/go.mod.
func FindRepoRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "server", "go.mod")); err == nil {
			return d, nil
		}
		if filepath.Dir(d) == d {
			return "", fmt.Errorf("no server/go.mod above %s; pass -repo-root", abs)
		}
	}
}

// Build compiles the exact tree under test into the given outputs (package -> file),
// never reusing a stale binary; the compiler output goes to logPath.
func Build(goBin, repoRoot, logPath string, outputs [][2]string) error {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	log, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer log.Close()
	for _, o := range outputs {
		if err := os.MkdirAll(filepath.Dir(o[1]), 0o755); err != nil {
			return err
		}
		cmd := exec.Command(goBin, "build", "-trimpath", "-o", o[1], o[0])
		cmd.Dir = filepath.Join(repoRoot, "server")
		cmd.Stdout, cmd.Stderr = log, log
		if err := runWithTimeout(cmd, 180*time.Second); err != nil {
			return fmt.Errorf("go build %s: %w", o[0], err)
		}
	}
	return nil
}

// ExecDaemons restarts the real Redis and PostgreSQL daemons (the --restart-daemons leg).
type ExecDaemons struct {
	R         ExecRunnerLike
	RedisHost string
	RedisPort string
	// PGCluster is "<version> <name>" for pg_ctlcluster (default "15 main").
	PGCluster  string
	ScratchDir string
}

// ErrRedisNotInstalled means no redis-server binary exists on this host: the
// daemon cannot be started or restarted here. It is a SKIP condition, never a pass.
var ErrRedisNotInstalled = errors.New("redis-server not installed; set HELIX_TEST_REDIS_ADDR to an externally managed loopback Redis")

// ExecRunnerLike is the subset of Runner the daemons need.
type ExecRunnerLike interface {
	Run(env []string, timeout time.Duration, stdin string, name string, args ...string) error
	LookPath(name string) bool
}

func (d ExecDaemons) cli(args ...string) error {
	return d.R.Run(nil, 15*time.Second, "", "redis-cli", append([]string{"-h", d.RedisHost, "-p", d.RedisPort}, args...)...)
}

func (d ExecDaemons) RedisReady() bool {
	return d.R.Run(nil, 2*time.Second, "", "redis-cli", "-h", d.RedisHost, "-p", d.RedisPort, "ping") == nil
}

// StartRedis ensures a Redis is up. An externally managed daemon is never
// started here; with none installed that is an honest error, never a simulated pass.
func (d ExecDaemons) StartRedis() error {
	if d.RedisReady() {
		return nil
	}
	if os.Getenv("HELIX_TEST_REDIS_ADDR") != "" {
		return fmt.Errorf("Redis not reachable at HELIX_TEST_REDIS_ADDR=%s:%s", d.RedisHost, d.RedisPort)
	}
	if !d.R.LookPath("redis-server") {
		return ErrRedisNotInstalled
	}
	if err := os.MkdirAll(d.ScratchDir, 0o755); err != nil {
		return err
	}
	if err := d.R.Run(nil, 10*time.Second, "", "redis-server", "--daemonize", "yes", "--dir", d.ScratchDir, "--dbfilename", "dump.rdb", "--appendonly", "no", "--save", ""); err != nil {
		return err
	}
	for i := 0; i < 100; i++ {
		if d.RedisReady() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("Redis readiness after daemon restart")
}

// preflight proves the restart can complete BEFORE anything is shut down, so a
// refused restart never leaves a shared daemon down.
func (d ExecDaemons) preflight() error {
	if os.Getenv("HELIX_TEST_REDIS_ADDR") != "" {
		return fmt.Errorf("refusing --restart-daemons: Redis is externally managed (HELIX_TEST_REDIS_ADDR); it cannot be restarted here")
	}
	if !d.R.LookPath("redis-server") {
		return fmt.Errorf("refusing --restart-daemons before shutdown: %w", ErrRedisNotInstalled)
	}
	if !d.R.LookPath("pg_ctlcluster") {
		return fmt.Errorf("refusing --restart-daemons before shutdown: pg_ctlcluster not installed (PostgreSQL is not a host-managed cluster)")
	}
	return nil
}

func (d ExecDaemons) Restart() error {
	if err := d.preflight(); err != nil {
		return err
	}
	if err := d.cli("SAVE"); err != nil {
		return err
	}
	if err := d.cli("SHUTDOWN", "SAVE"); err != nil {
		return err
	}
	for i := 0; i < 100; i++ {
		if !d.RedisReady() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := d.StartRedis(); err != nil {
		return err
	}
	cluster := d.PGCluster
	if cluster == "" {
		cluster = "15 main"
	}
	if err := d.R.Run(nil, 45*time.Second, "", "pg_ctlcluster", append(strings.Fields(cluster), "restart")...); err != nil {
		return err
	}
	for i := 0; i < 150; i++ {
		if d.R.Run(nil, 2*time.Second, "", "pg_isready") == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("PostgreSQL readiness after daemon restart")
}
