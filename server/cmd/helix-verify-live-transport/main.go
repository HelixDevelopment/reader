// Command helix-verify-live-transport is the Go port of scripts/verify_live_transport.py:
// it starts the real helix-server over TLS and verifies HTTP/1.1 (TLS 1.2),
// HTTP/2 and HTTP/3 (QUIC, TLS 1.3) plus the Alt-Svc advertisement, with the
// in-process quic-go client. With -extended (default) it also checks the TLS
// version floor, Brotli/gzip negotiation and the identity fallback; those checks
// are named "[ext]" and have no Python counterpart. Exit 0 PASS, 1 FAIL, 2 setup error.
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
	evidence := flag.String("evidence", "", "evidence directory (default <root>/evidence/live-transport-go)")
	goBin := flag.String("go", envOr("HELIX_GO", "go"), "go toolchain used to build the tree under test")
	build := flag.Bool("build", true, "build helix-server from this tree first")
	serverBin := flag.String("server-bin", "", "helix-server binary when -build=false")
	address := flag.String("address", "127.0.0.1:28443", "loopback listen address (TCP and UDP)")
	extended := flag.Bool("extended", true, "add the [ext] checks the Python verifier lacks")
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
		*evidence = filepath.Join(*root, "evidence", "live-transport-go")
	}
	if err := os.MkdirAll(*evidence, 0o755); err != nil {
		fatal(err)
	}
	work := filepath.Join(*scratch, "helix-transport-tls")
	if *build {
		*serverBin = filepath.Join(*scratch, "bin", "helix-server-live")
		if err := liveverify.Build(*goBin, *root, filepath.Join(*evidence, "build.log"), [][2]string{{"./cmd/helix-server", *serverBin}}); err != nil {
			fatal(err)
		}
	} else if *serverBin == "" {
		fatal(fmt.Errorf("-build=false needs -server-bin"))
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		fatal(err)
	}
	if err := liveverify.EnsureCert(work); err != nil {
		fatal(err)
	}
	env := liveverify.Environ(map[string]string{
		"HELIX_ADDRESS": *address, "HELIX_TLS_CERT": filepath.Join(work, "cert.pem"), "HELIX_TLS_KEY": filepath.Join(work, "key.pem"),
		"HELIX_HTTP3": "true", "HELIX_ADMIN_TOKEN": "transport-admin-token", "HELIX_OPDS_USERNAME": "transport-reader",
		"HELIX_OPDS_PASSWORD": "transport-password", "HELIX_POSTGRES_DSN": "", "HELIX_REDIS_URL": "", "HELIX_AI_MEMORY": "false", "GIN_MODE": "release",
	})
	log, err := os.Create(filepath.Join(*evidence, "server.log"))
	if err != nil {
		fatal(err)
	}
	defer log.Close()
	res := liveverify.RunTransport(liveverify.TransportConfig{
		Starter: &liveverify.ExecLauncher{Binary: *serverBin, Env: env, Log: log}, Base: "https://" + *address,
		WorkDir: work, EvidenceDir: *evidence, Extended: *extended,
	})
	res = liveverify.AttachIdentity(*evidence, res, liveverify.CollectIdentity(liveverify.IdentityConfig{
		Runner: liveverify.ExecRunner{}, RepoRoot: *root, Binaries: map[string]string{"helix-server": *serverBin}, NoDatastores: true,
	}))
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
	fmt.Fprintln(os.Stderr, "helix-verify-live-transport:", err)
	os.Exit(2)
}
