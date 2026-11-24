// Command helix-verify-backup-restore is the Go port of scripts/verify_backup_restore.py:
// a live backup/restore rehearsal with an isolated restored PostgreSQL database
// and Redis daemon. It needs psql, pg_dump, pg_restore, createdb, dropdb,
// redis-cli and redis-server on PATH. Exit 0 PASS, 1 FAIL, 2 setup error.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helixreader/helix/server/internal/liveverify"
)

func main() {
	wd, _ := os.Getwd()
	root := flag.String("repo-root", "", "repository root (default: found from the working directory)")
	scratch := flag.String("scratch", "", "scratch directory (default <root>/.helix-verify)")
	evidence := flag.String("evidence", "", "evidence directory (default <root>/evidence/backup-restore-go)")
	goBin := flag.String("go", envOr("HELIX_GO", "go"), "go toolchain used to build the tree under test")
	build := flag.Bool("build", true, "build helix-server from this tree first")
	serverBin := flag.String("server-bin", "", "helix-server binary when -build=false")
	flag.Parse()
	if *root == "" {
		r, err := liveverify.FindRepoRoot(wd)
		if err != nil {
			fatal(err)
		}
		*root = r
	}
	if *scratch == "" {
		*scratch = filepath.Join(*root, ".helix-verify")
	}
	if *evidence == "" {
		*evidence = filepath.Join(*root, "evidence", "backup-restore-go")
	}
	if err := os.MkdirAll(*evidence, 0o755); err != nil {
		fatal(err)
	}
	if *build {
		*serverBin = filepath.Join(*scratch, "bin", "helix-server-live")
		if err := liveverify.Build(*goBin, *root, filepath.Join(*evidence, "build.log"), [][2]string{{"./cmd/helix-server", *serverBin}}); err != nil {
			fatal(err)
		}
	} else if *serverBin == "" {
		fatal(fmt.Errorf("-build=false needs -server-bin"))
	}
	work := filepath.Join(*scratch, "backup-restore")
	cfg := liveverify.BackupConfig{
		Runner: liveverify.ExecRunner{}, EvidenceDir: *evidence, WorkDir: work,
		TestDSN: os.Getenv("HELIX_TEST_POSTGRES_DSN"), RedisAddr: os.Getenv("HELIX_TEST_REDIS_ADDR"), RedisDataDir: os.Getenv("HELIX_TEST_REDIS_DATA_DIR"),
	}
	cfg.StartServer = func(port int, database, redisURL, logName string) (string, liveverify.Process, error) {
		log, err := os.Create(filepath.Join(*evidence, logName))
		if err != nil {
			return "", nil, err
		}
		env := liveverify.Environ(map[string]string{
			"HELIX_ADDRESS": fmt.Sprintf("127.0.0.1:%d", port), "HELIX_POSTGRES_DSN": cfg.DSN(database), "HELIX_REDIS_URL": redisURL,
			"HELIX_AI_MEMORY": "true", "HELIX_EMBEDDING_DIMENSIONS": "3", "HELIX_ADMIN_TOKEN": "backup-admin-token",
			"HELIX_OPDS_USERNAME": "backup-reader", "HELIX_OPDS_PASSWORD": "backup-password", "HELIX_KOSYNC_REGISTRATION": "false",
			"HELIX_HTTP3": "false", "HELIX_COVER_CACHE": filepath.Join(work, fmt.Sprintf("covers-%d", port)), "GIN_MODE": "release",
		})
		l := &liveverify.ExecLauncher{Binary: *serverBin, Env: env, Base: fmt.Sprintf("http://127.0.0.1:%d", port), Log: log}
		return l.Start()
	}
	res := liveverify.RunBackupRestore(cfg)
	redisAddr := cfg.RedisAddr
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	id := liveverify.CollectIdentity(liveverify.IdentityConfig{
		Runner: cfg.Runner, RepoRoot: *root, Binaries: map[string]string{"helix-server": *serverBin},
		TestDSN: cfg.TestDSN, RedisAddr: redisAddr,
	})
	res = liveverify.AttachIdentity(*evidence, res, id)
	fmt.Println(res.Summary())
	os.Exit(res.ExitCode())
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "helix-verify-backup-restore:", err)
	os.Exit(2)
}
