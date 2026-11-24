// Command gomutate is the Go replacement for scripts/verify_go_mutation.py.
//
//	gomutate                      run the product catalogue against the repository
//	gomutate --list               print the catalogue ids
//	gomutate --only id[,id...]    run a subset
//	gomutate --selftest           run the runner's own paired mutants (tools/verify/mutation/selftest_catalogue.json)
//
// Exit status: 0 PASS, 1 FAIL, 2 harness error.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helixreader/helix/tools/verify/internal/mutation"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	fs := flag.NewFlagSet("gomutate", flag.ContinueOnError)
	root := fs.String("root", "", "repository root holding the Go modules (default: the git top level, else .)")
	catalogue := fs.String("catalogue", "", "catalogue JSON (default: the embedded product catalogue)")
	only := fs.String("only", "", "comma-separated mutant ids")
	list := fs.Bool("list", false, "print the catalogue ids and exit")
	selftest := fs.Bool("selftest", false, "run the runner's own mutants against tools/")
	evidence := fs.String("evidence", "", "evidence directory (default <root>/evidence/go-mutation)")
	work := fs.String("work", "", "scratch directory (default <root>/.helix-verify/go-mutation)")
	wall := fs.Duration("wall", 300*time.Second, "wall budget per targeted test")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := *root
	if r == "" {
		r = repoRoot()
	}
	var ms []mutation.Mutant
	var err error
	switch {
	case *selftest:
		r = filepath.Join(r, "tools")
		if *catalogue == "" {
			*catalogue = filepath.Join(r, "verify", "mutation", "selftest_catalogue.json")
		}
		if *evidence == "" {
			*evidence = filepath.Join(r, "..", "evidence", "go-mutation-selftest")
		}
		if *work == "" {
			*work = filepath.Join(r, "..", ".helix-verify", "go-mutation-selftest")
		}
		fallthrough
	case *catalogue != "":
		var b []byte
		if b, err = os.ReadFile(*catalogue); err == nil {
			ms, err = mutation.ParseCatalogue(b)
		}
	default:
		ms, err = mutation.DefaultCatalogue()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gomutate:", err)
		return 2
	}
	if *list {
		for _, m := range ms {
			fmt.Println(m.ID)
		}
		return 0
	}
	cfg := mutation.Config{Root: r, Work: *work, Evidence: *evidence, Wall: *wall, Progress: os.Stderr}
	if *only != "" {
		cfg.Only = strings.Split(*only, ",")
	}
	sum, err := mutation.Run(cfg, ms)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gomutate:", err)
		return 2
	}
	brief := map[string]any{"status": sum.Status, "pass": sum.Pass, "fail": sum.Fail}
	if sum.Reason != "" {
		brief["reason"], brief["ids"] = sum.Reason, sum.IDs
	} else {
		brief["mutants"], brief["controls"], brief["with_predicted_assertion"] = sum.Mutants, sum.Controls, sum.WithPredicted
	}
	b, _ := json.Marshal(brief)
	fmt.Println(string(b))
	if sum.Status != "PASS" {
		return 1
	}
	return 0
}

func repoRoot() string {
	dir, _ := os.Getwd()
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "helix-deps.yaml")); err == nil {
			return d
		}
		if filepath.Dir(d) == d {
			return dir
		}
	}
}
