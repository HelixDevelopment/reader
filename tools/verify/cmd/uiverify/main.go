// Command uiverify is the Go replacement for scripts/verify_ui.py: deterministic
// browser checks of the static Helix product prototype at four viewport sizes.
//
//	uiverify [--serve DIR] [--chrome PATH] [BASE_URL [OUT_DIR]]
//	uiverify --fixture [--chrome PATH] [OUT_DIR]   (validates the locator/visibility engine against known content)
//
// It launches Chromium, so it REFUSES to run anywhere but inside the container
// that tools/isolation/run_container.sh starts (rootless, no network, read-only
// launch.sh marker, HRDISO_ISOLATED=1; see qa/isolation/run_uiverify.sh).
// There is no override. Exit: 0 all PASS,
// 1 a check FAILed, 2 harness error, 3 not inside the isolation container.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/helixreader/helix/tools/verify/internal/uiverify"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	fs := flag.NewFlagSet("uiverify", flag.ContinueOnError)
	fixture := fs.Bool("fixture", false, "serve the embedded known-content fixture page and validate the locator/visibility engine (golden-good and golden-bad cases)")
	serve := fs.String("serve", "", "serve this built directory on 127.0.0.1:4173 for the duration of the run")
	chrome := fs.String("chrome", firstNonEmpty(os.Getenv("HELIX_CHROME"), "/usr/bin/chromium"), "Chromium executable")
	timeout := fs.Duration("timeout", 15*time.Second, "per-action wait budget")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := uiverify.RequireContainer(uiverify.DefaultIsolationProbe()); err != nil {
		fmt.Fprintln(os.Stderr, "uiverify:", err)
		return 3
	}
	if *fixture {
		return runFixture(*chrome, *timeout, fs.Arg(0))
	}
	base := "http://127.0.0.1:4173/"
	out := "evidence/ui"
	if fs.NArg() > 0 {
		base = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		out = fs.Arg(1)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "uiverify:", err)
		return 2
	}
	if *serve != "" {
		ln, err := net.Listen("tcp", "127.0.0.1:4173")
		if err != nil {
			fmt.Fprintln(os.Stderr, "uiverify: serve:", err)
			return 2
		}
		srv := &http.Server{Handler: http.FileServer(http.Dir(*serve))}
		go func() {
			if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				fmt.Fprintln(os.Stderr, "uiverify: serve:", err)
			}
		}()
		defer func() {
			ctx, c := context.WithTimeout(context.Background(), 2*time.Second)
			defer c()
			_ = srv.Shutdown(ctx)
		}()
	}
	rep := uiverify.RunAll(&uiverify.ChromeBrowser{ExecPath: *chrome, Timeout: *timeout}, base, out)
	b, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "ui-results.json"), append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "uiverify:", err)
		return 2
	}
	s, _ := json.Marshal(rep.Summary)
	fmt.Println(string(s))
	if rep.Summary.Fail > 0 {
		return 1
	}
	return 0
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

func runFixture(chrome string, timeout time.Duration, out string) int {
	if out == "" {
		out = "evidence/ui-fixture"
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "uiverify:", err)
		return 2
	}
	ln, err := net.Listen("tcp", "127.0.0.1:4173")
	if err != nil {
		fmt.Fprintln(os.Stderr, "uiverify: fixture:", err)
		return 2
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(uiverify.FixtureHTML))
	})}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()
	checks, err := uiverify.RunFixture(&uiverify.ChromeBrowser{ExecPath: chrome, Timeout: timeout}, "http://127.0.0.1:4173/")
	if err != nil {
		fmt.Fprintln(os.Stderr, "uiverify: fixture:", err)
		return 2
	}
	pass, fail := 0, 0
	for _, c := range checks {
		if c.Status == "PASS" {
			pass++
		} else {
			fail++
		}
	}
	rep := map[string]any{"schema_version": 1, "kind": "hv-fixture", "checks": checks, "summary": map[string]int{"pass": pass, "fail": fail}}
	b, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "fixture-results.json"), append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "uiverify:", err)
		return 2
	}
	s, _ := json.Marshal(rep["summary"])
	fmt.Println(string(s))
	if fail > 0 {
		return 1
	}
	return 0
}
