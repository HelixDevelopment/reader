// Command helix-verify-live-server is the Go port of scripts/verify_live_server.py:
// it builds the tree under test, starts the real helix-server against real
// PostgreSQL/pgvector and Redis, and verifies the public protocols and a restart.
// Exit 0 = PASS, 1 = FAIL, 2 = usage/setup error. Results go to <evidence>/results.json.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/helixreader/helix/server/internal/liveverify"
)

func main() {
	wd, _ := os.Getwd()
	root := flag.String("repo-root", "", "repository root (default: found from the working directory)")
	scratch := flag.String("scratch", "", "scratch directory (default <root>/.helix-verify)")
	evidence := flag.String("evidence", "", "evidence directory (default <root>/evidence/live-server-go)")
	goBin := flag.String("go", envOr("HELIX_GO", "go"), "go toolchain used to build the tree under test")
	build := flag.Bool("build", true, "build helix-server and helix-pse-check from this tree first")
	serverBin := flag.String("server-bin", "", "helix-server binary when -build=false")
	pseBin := flag.String("pse-check-bin", "", "helix-pse-check binary when -build=false")
	address := flag.String("address", "127.0.0.1:28080", "loopback listen address of the disposable server")
	restart := flag.Bool("restart-daemons", false, "also restart the Redis and PostgreSQL daemons between server runs")
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
		*evidence = filepath.Join(*root, "evidence", "live-server-go")
	}
	if err := os.MkdirAll(*evidence, 0o755); err != nil {
		fatal(err)
	}
	binDir := filepath.Join(*scratch, "bin")
	if *build {
		*serverBin, *pseBin = filepath.Join(binDir, "helix-server-live"), filepath.Join(binDir, "helix-pse-check")
		if err := liveverify.Build(*goBin, *root, filepath.Join(*evidence, "build.log"),
			[][2]string{{"./cmd/helix-server", *serverBin}, {"./cmd/helix-pse-check", *pseBin}}); err != nil {
			fatal(err)
		}
	} else if *serverBin == "" || *pseBin == "" {
		fatal(fmt.Errorf("-build=false needs -server-bin and -pse-check-bin"))
	}

	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		name := "postgres"
		if u, err := user.Current(); err == nil {
			name = u.Username
		}
		dsn = "postgres:///helix_test?host=/var/run/postgresql&user=" + name
	}
	redisAddr := envOr("HELIX_TEST_REDIS_ADDR", "127.0.0.1:6379")
	// Disposable loopback credentials, the same fixed non-production values as the Python verifier.
	env := liveverify.Environ(map[string]string{
		"HELIX_ADDRESS": *address, "HELIX_POSTGRES_DSN": dsn, "HELIX_REDIS_URL": "redis://" + redisAddr,
		"HELIX_AI_MEMORY": "true", "HELIX_EMBEDDING_DIMENSIONS": "3", "HELIX_ADMIN_TOKEN": "live-admin-token",
		"HELIX_OPDS_USERNAME": "live-reader", "HELIX_OPDS_PASSWORD": "live-password", "HELIX_KOSYNC_REGISTRATION": "true",
		"HELIX_HTTP3": "false", "HELIX_COVER_CACHE": filepath.Join(*scratch, "live-server-covers"), "GIN_MODE": "release",
	})
	log, err := os.Create(filepath.Join(*evidence, "server.log"))
	if err != nil {
		fatal(err)
	}
	defer log.Close()
	host, port := redisAddr, "6379"
	if i := strings.LastIndex(redisAddr, ":"); i >= 0 {
		host, port = redisAddr[:i], redisAddr[i+1:]
	}
	runner := liveverify.ExecRunner{}
	res := liveverify.RunLiveServer(liveverify.LiveServerConfig{
		Launcher:    &liveverify.ExecLauncher{Binary: *serverBin, Env: env, Base: "http://" + *address, Log: log},
		EvidenceDir: *evidence,
		Library:     filepath.Join(*scratch, "live-server-library"),
		StreamLib:   filepath.Join(*scratch, "live-server-streams"),
		PSECheck:    *pseBin, PSEEnv: env, RestartDaemons: *restart,
		Daemons: liveverify.ExecDaemons{R: runner, RedisHost: host, RedisPort: port, ScratchDir: *scratch},
	})
	id := liveverify.CollectIdentity(liveverify.IdentityConfig{
		Runner: runner, RepoRoot: *root, Binaries: map[string]string{"helix-server": *serverBin, "helix-pse-check": *pseBin},
		TestDSN: os.Getenv("HELIX_TEST_POSTGRES_DSN"), RedisAddr: redisAddr,
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
	fmt.Fprintln(os.Stderr, "helix-verify-live-server:", err)
	os.Exit(2)
}
