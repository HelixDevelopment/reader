// Package mutation is the Go port of scripts/verify_go_mutation.py: it proves
// that targeted Go tests reject deliberate contract-breaking mutations.
//
// Each mutant runs in an isolated copy of the repository's Go modules. The
// original files are hashed before and after the campaign; a mutant that
// passes, a failed mutation edit, or any change to the original tree fails the
// campaign. Logs contain command output only.
//
// A mutant that fails for an unrelated reason proves nothing about the
// assertion it targets, so every mutant carries Expect: a substring the test
// output must contain for the mutant to count as rejected.
package mutation

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// Mutant is one entry of the mutation catalogue (same fields and semantics as
// the Python MUTANTS list).
type Mutant struct {
	ID      string `json:"id"`
	Module  string `json:"module"`
	File    string `json:"file"`
	Old     string `json:"old"`
	New     string `json:"new"`
	Package string `json:"package"`
	Test    string `json:"test"`
	Expect  string `json:"expect"`
}

//go:embed catalogue.json
var defaultCatalogue []byte

// DefaultCatalogue returns the embedded product catalogue (ported verbatim
// from scripts/verify_go_mutation.py).
func DefaultCatalogue() ([]Mutant, error) { return ParseCatalogue(defaultCatalogue) }

// ParseCatalogue decodes a catalogue and rejects duplicate ids.
func ParseCatalogue(b []byte) ([]Mutant, error) {
	var ms []Mutant
	if err := json.Unmarshal(b, &ms); err != nil {
		return nil, fmt.Errorf("catalogue: %w", err)
	}
	seen := map[string]bool{}
	for _, m := range ms {
		if seen[m.ID] {
			return nil, fmt.Errorf("catalogue: duplicate mutant id %q", m.ID)
		}
		seen[m.ID] = true
	}
	return ms, nil
}

// Config selects where a campaign runs. Zero values give the Python defaults.
type Config struct {
	Root     string        // repository root holding the Go modules (default ".")
	Work     string        // scratch tree (default Root/.helix-verify/go-mutation)
	Evidence string        // logs and results.json (default Root/evidence/go-mutation)
	Go       string        // go binary (default $HELIX_GO or "go")
	Wall     time.Duration // per-test wall budget (default 300s)
	Only     []string      // restrict the campaign to these mutant ids
	Progress io.Writer     // one line per mutant (default discarded)
}

// Result is one row of results.json.
type Result struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Expected string `json:"expected,omitempty"`
	Test     string `json:"test,omitempty"`
	Exit     *int   `json:"exit,omitempty"`
	Log      string `json:"log,omitempty"`
}

// Summary is results.json.
type Summary struct {
	Status                string   `json:"status"`
	Pass                  int      `json:"pass"`
	Fail                  int      `json:"fail"`
	Mutants               int      `json:"mutants,omitempty"`
	Controls              int      `json:"controls,omitempty"`
	WithPredicted         int      `json:"with_predicted_assertion,omitempty"`
	OriginalTreeUnchanged bool     `json:"original_tree_unchanged"`
	Reason                string   `json:"reason,omitempty"`
	IDs                   []string `json:"ids,omitempty"`
	Results               []Result `json:"results,omitempty"`
}

// Only the Go toolchain prints these; a test asserting its own error text
// cannot produce them, so their presence means the mutant never ran.
var buildFailures = []string{
	"[build failed]",
	"[setup failed]",
	"replacement directory",
	"no required module provides package",
	"cannot find module",
	"missing go.sum entry",
}

const timeoutExit = 124

func (c *Config) defaults() {
	if c.Root == "" {
		c.Root = "."
	}
	c.Root, _ = filepath.Abs(c.Root)
	if c.Work == "" {
		c.Work = filepath.Join(c.Root, ".helix-verify", "go-mutation")
	}
	c.Work, _ = filepath.Abs(c.Work)
	if c.Evidence == "" {
		c.Evidence = filepath.Join(c.Root, "evidence", "go-mutation")
	}
	if c.Go == "" {
		c.Go = os.Getenv("HELIX_GO")
	}
	if c.Go == "" {
		c.Go = "go"
	}
	if c.Wall == 0 {
		c.Wall = 300 * time.Second
	}
	if c.Progress == nil {
		c.Progress = io.Discard
	}
}

// goModules lists the top-level directories holding a go.mod. Mutants are
// edited into a copy that mirrors the repo layout, because a module may reach
// its sibling through a relative replace directive; copying only the target
// module would leave that replace dangling and the resulting build failure
// would otherwise be counted as a rejected mutant.
func goModules(root string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(root, "*", "go.mod"))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, p := range paths {
		names = append(names, filepath.Base(filepath.Dir(p)))
	}
	sort.Strings(names)
	return names, nil
}

func digest(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// copyTree copies src to dst like shutil.copytree(symlinks=False): symlinks
// (directories included) are followed and their contents copied. A symlink
// cycle is cut after 32 levels.
func copyTree(src, dst string) error { return copyTreeDepth(src, dst, 0) }

func copyTreeDepth(src, dst string, depth int) error {
	if depth > 32 {
		return fmt.Errorf("copy %s: too many levels of symbolic links", src)
	}
	info, err := os.Stat(src) // follows symlinks
	if err != nil {
		return err
	}
	if !info.IsDir() {
		b, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, info.Mode().Perm())
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if err := copyTreeDepth(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()), depth+1); err != nil {
			return err
		}
	}
	return nil
}

const (
	scratchDirName = ".helix-verify"
	scratchMarker  = ".helix-verify-scratch"
	scratchMagic   = "helix-verify scratch directory created by gomutate; safe to remove\n"
)

// protectedDirs are never scratch, whatever their location (mirrors the
// system and HOME-child lists of submodules/constitution/scripts/lib/safe_rm.sh).
var protectedDirs = []string{"/", "/bin", "/boot", "/dev", "/etc", "/home", "/lib", "/lib32", "/lib64", "/media", "/mnt", "/opt", "/proc", "/root", "/run", "/sbin", "/snap", "/srv", "/sys", "/tmp", "/usr", "/var"}

var protectedHomeChildren = []string{"Projects", "scripts", "Documents", "Desktop", "Downloads", "Pictures", "Music", "Videos", "bin", "go", "opt", ".ssh", ".config", ".local", ".cache", ".gnupg", ".aws", ".kube", ".docker", ".git"}

// scratchPath validates work and returns the resolved absolute path. It
// accepts only a STRICT descendant of a directory named ".helix-verify"
// (after resolving every symlink in the existing prefix), never HOME, an
// ancestor of HOME, a system directory or a protected HOME child.
func scratchPath(work string) (string, error) {
	refuse := func(why string) (string, error) {
		return "", fmt.Errorf("refusing scratch path %q: %s (constitution 11.4.287)", work, why)
	}
	if work == "" || !filepath.IsAbs(work) {
		return refuse("not an absolute path")
	}
	abs := filepath.Clean(work)
	if fi, err := os.Lstat(abs); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return refuse("is a symlink")
	}
	// resolve the deepest existing ancestor so a symlinked parent cannot hide the real target
	existing, rest := abs, ""
	for {
		if _, err := os.Lstat(existing); err == nil {
			break
		}
		rest = filepath.Join(filepath.Base(existing), rest)
		parent := filepath.Dir(existing)
		if parent == existing {
			break
		}
		existing = parent
	}
	real, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return refuse("cannot resolve: " + err.Error())
	}
	resolved := filepath.Join(real, rest)
	for _, p := range protectedDirs {
		if resolved == p {
			return refuse("is a system directory")
		}
	}
	home := os.Getenv("HOME")
	if home != "" {
		if h, err := filepath.EvalSymlinks(home); err == nil {
			home = h
		}
		home = filepath.Clean(home)
		if resolved == home || strings.HasPrefix(home+string(filepath.Separator), resolved+string(filepath.Separator)) {
			return refuse("is HOME or an ancestor of HOME")
		}
		for _, c := range protectedHomeChildren {
			if resolved == filepath.Join(home, c) {
				return refuse("is a protected child of HOME")
			}
		}
	}
	// strict descendant of a directory named .helix-verify
	underScratch := false
	for d := filepath.Dir(resolved); ; d = filepath.Dir(d) {
		if filepath.Base(d) == scratchDirName {
			underScratch = true
			break
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	if !underScratch {
		return refuse("is not a descendant of a " + scratchDirName + " directory")
	}
	return resolved, nil
}

// prepareScratch validates work, clears a previous scratch of ours and
// creates it with the marker file removeScratch later requires.
func prepareScratch(work string) error {
	if err := removeScratch(work); err != nil {
		return err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(work, scratchMarker), []byte(scratchMagic), 0o644)
}

// removeScratch deletes a scratch tree this tool created. It refuses anything
// else: the path must pass scratchPath, and an existing directory must carry
// the marker file with the exact magic (or be empty). Nothing is deleted on
// refusal (constitution 11.4.287).
func removeScratch(work string) error {
	p, err := scratchPath(work)
	if err != nil {
		return err
	}
	fi, err := os.Lstat(p)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("refusing scratch path %q: not a directory", work)
	}
	if b, err := os.ReadFile(filepath.Join(p, scratchMarker)); err != nil || string(b) != scratchMagic {
		ents, derr := os.ReadDir(p)
		if derr != nil || len(ents) != 0 {
			return fmt.Errorf("refusing scratch path %q: no %s marker and not empty (not created by this tool)", work, scratchMarker)
		}
	}
	return os.RemoveAll(p)
}

func goTest(ctx context.Context, goBin, moduleDir, pkg, test string, wall time.Duration) (command []string, output string, code int) {
	inner := int(wall.Seconds()) - 60
	if inner < 30 {
		inner = 30
	}
	command = []string{goBin, "test", "-trimpath", pkg, "-run", "^" + test + "$", "-count=1", "-timeout", fmt.Sprintf("%ds", inner)}
	ctx, cancel := context.WithTimeout(ctx, wall)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = moduleDir
	cmd.Env = append(os.Environ(), "GOPROXY=off")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Kill the whole group the test run created (go test and its test binary), never pgid <= 1.
		if pid := cmd.Process.Pid; pid > 1 {
			return syscall.Kill(-pid, syscall.SIGKILL)
		}
		return cmd.Process.Kill()
	}
	cmd.WaitDelay = 5 * time.Second
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	output = strings.ToValidUTF8(buf.String(), "�")
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return command, output + fmt.Sprintf("\nkilled: no verdict after %ds\n", int(wall.Seconds())), timeoutExit
	}
	if err == nil {
		return command, output, 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() != 0 {
		return command, output, ee.ExitCode()
	}
	// The go binary could not even start: that is a setup failure, never a verdict about the mutant.
	return command, output + "\n[setup failed] " + err.Error() + "\n", 127
}

func writeSummary(cfg Config, s Summary) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfg.Evidence, "results.json"), append(b, '\n'), 0o644)
}

func ptr(i int) *int { return &i }

// Run executes the campaign and returns its summary. A FAIL summary is a
// normal return value; err is only for harness problems (bad config, I/O).
func Run(cfg Config, catalogue []Mutant) (Summary, error) {
	cfg.defaults()
	mutants := catalogue
	if len(cfg.Only) > 0 {
		want := map[string]bool{}
		for _, id := range cfg.Only {
			want[id] = true
		}
		mutants = nil
		for _, m := range catalogue {
			if want[m.ID] {
				mutants = append(mutants, m)
				delete(want, m.ID)
			}
		}
		if len(want) > 0 {
			var missing []string
			for id := range want {
				missing = append(missing, id)
			}
			sort.Strings(missing)
			return Summary{}, fmt.Errorf("unknown mutant ids: %v", missing)
		}
	}
	if err := os.MkdirAll(cfg.Evidence, 0o755); err != nil {
		return Summary{}, err
	}
	// Every mutant must name the assertion that proves its contract was caught.
	// A mutant without one could be "killed" by any unrelated failure.
	var unpredicted []string
	for _, m := range mutants {
		if m.Expect == "" {
			unpredicted = append(unpredicted, m.ID)
		}
	}
	if len(unpredicted) > 0 {
		s := Summary{Status: "FAIL", Fail: len(unpredicted), Reason: "mutants with no predicted assertion", IDs: unpredicted}
		return s, writeSummary(cfg, s)
	}
	modules, err := goModules(cfg.Root)
	if err != nil {
		return Summary{}, err
	}
	if len(modules) == 0 {
		return Summary{}, fmt.Errorf("no Go modules (*/go.mod) under %s", cfg.Root)
	}
	if err := prepareScratch(cfg.Work); err != nil {
		return Summary{}, err
	}
	type key struct{ module, file string }
	originals := map[key]string{}
	for _, m := range mutants {
		d, err := digest(filepath.Join(cfg.Root, m.Module, m.File))
		if err != nil {
			return Summary{}, fmt.Errorf("mutant %s: %w", m.ID, err)
		}
		originals[key{m.Module, m.File}] = d
	}
	ctx := context.Background()
	var results []Result

	// Positive control: a targeted test that already fails on unmutated code
	// would "reject" every mutant thrown at it. Each distinct test is run once
	// against a pristine copy first.
	controlCase := filepath.Join(cfg.Work, "_control")
	for _, name := range modules {
		if err := copyTree(filepath.Join(cfg.Root, name), filepath.Join(controlCase, name)); err != nil {
			return Summary{}, err
		}
	}
	type ctl struct{ module, pkg, test string }
	seen := map[ctl]bool{}
	var controls []ctl
	for _, m := range mutants {
		c := ctl{m.Module, m.Package, m.Test}
		if !seen[c] {
			seen[c] = true
			controls = append(controls, c)
		}
	}
	sort.Slice(controls, func(i, j int) bool {
		a, b := controls[i], controls[j]
		if a.module != b.module {
			return a.module < b.module
		}
		if a.pkg != b.pkg {
			return a.pkg < b.pkg
		}
		return a.test < b.test
	})
	for _, c := range controls {
		command, output, code := goTest(ctx, cfg.Go, filepath.Join(controlCase, c.module), c.pkg, c.test, cfg.Wall)
		logName := fmt.Sprintf("control-%s-%s.log", c.module, c.test)
		if err := os.WriteFile(filepath.Join(cfg.Evidence, logName), []byte("$ "+strings.Join(command, " ")+"\n"+output+fmt.Sprintf("\nexit=%d\n", code)), 0o644); err != nil {
			return Summary{}, err
		}
		if code != 0 {
			results = append(results, Result{ID: fmt.Sprintf("control-%s-%s", c.module, c.test), Status: "FAIL",
				Reason: "targeted test does not pass on unmutated code, so it cannot reject anything"})
		}
	}

	for _, m := range mutants {
		r := runMutant(ctx, cfg, modules, m)
		results = append(results, r)
		fmt.Fprintf(cfg.Progress, "%s: %s %s\n", m.ID, r.Status, r.Reason)
	}

	unchanged := true
	for k, before := range originals {
		now, err := digest(filepath.Join(cfg.Root, k.module, k.file))
		if err != nil || now != before {
			unchanged = false
		}
	}
	if !unchanged {
		results = append(results, Result{ID: "original-tree-unchanged", Status: "FAIL"})
	}
	pass, fail := 0, 0
	for _, r := range results {
		if r.Status == "PASS" {
			pass++
		} else if r.Status == "FAIL" {
			fail++
		}
	}
	status := "FAIL"
	if len(results) == len(mutants) && fail == 0 && unchanged {
		status = "PASS"
	}
	predicted := 0
	for _, m := range mutants {
		if m.Expect != "" {
			predicted++
		}
	}
	s := Summary{Status: status, Pass: pass, Fail: fail, Mutants: len(mutants), Controls: len(controls),
		WithPredicted: predicted, OriginalTreeUnchanged: unchanged, Results: results}
	if err := writeSummary(cfg, s); err != nil {
		return s, err
	}
	if err := removeScratch(cfg.Work); err != nil {
		return s, err
	}
	return s, nil
}

func runMutant(ctx context.Context, cfg Config, modules []string, m Mutant) Result {
	fail := func(reason string) Result { return Result{ID: m.ID, Status: "FAIL", Reason: reason} }
	caseDir := filepath.Join(cfg.Work, m.ID)
	for _, name := range modules {
		if err := copyTree(filepath.Join(cfg.Root, name), filepath.Join(caseDir, name)); err != nil {
			return fail("setup: " + err.Error())
		}
	}
	moduleCopy := filepath.Join(caseDir, m.Module)
	target := filepath.Join(moduleCopy, m.File)
	text, err := os.ReadFile(target)
	if err != nil {
		return fail("setup: " + err.Error())
	}
	if n := strings.Count(string(text), m.Old); n != 1 {
		return fail(fmt.Sprintf("mutation anchor count %d", n))
	}
	if err := os.WriteFile(target, []byte(strings.Replace(string(text), m.Old, m.New, 1)), 0o644); err != nil {
		return fail("setup: " + err.Error())
	}
	command, output, code := goTest(ctx, cfg.Go, moduleCopy, m.Package, m.Test, cfg.Wall)
	logName := m.ID + ".log"
	_ = os.WriteFile(filepath.Join(cfg.Evidence, logName), []byte("$ "+strings.Join(command, " ")+"\n"+output+fmt.Sprintf("\nexit=%d\n", code)), 0o644)
	setup := ""
	for _, mark := range buildFailures {
		if strings.Contains(output, mark) {
			setup = mark
			break
		}
	}
	killed := code != 0 && code != timeoutExit
	reason := ""
	if !killed {
		reason = "targeted test accepted the mutant"
	}
	switch {
	case code == timeoutExit:
		killed = false
		reason = "no verdict: the targeted test did not finish, which proves nothing about the mutant"
	case setup != "":
		killed = false
		reason = fmt.Sprintf("mutant never ran: the harness could not build the package (%s)", setup)
	case killed && m.Expect != "" && !strings.Contains(output, m.Expect):
		killed = false
		reason = fmt.Sprintf("test failed, but not with the predicted assertion %s", pyRepr(m.Expect))
	case killed && m.Expect != "":
		reason = "rejected with the predicted assertion"
	}
	status := "FAIL"
	if killed {
		status = "PASS"
	}
	return Result{ID: m.ID, Status: status, Reason: reason, Expected: m.Expect, Test: m.Test, Exit: ptr(code), Log: logName}
}

// pyRepr approximates Python's repr() of a str for the reason text.
func pyRepr(s string) string {
	q := "'"
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		q = `"`
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	if q == "'" {
		s = strings.ReplaceAll(s, "'", `\'`)
	}
	s = strings.ReplaceAll(s, "\n", `\n`)
	return q + s + q
}
