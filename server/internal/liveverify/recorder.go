// Package liveverify holds the black-box live verifiers: Go ports of
// scripts/verify_live_server.py, scripts/verify_live_transport.py and
// scripts/verify_backup_restore.py. They drive a running helix-server process
// over its public protocols only and record one named PASS/FAIL/SKIP per
// assertion, written to results.json in the same shape the Python verifiers use.
package liveverify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Check is one named assertion outcome (the Python {"name","status"} record).
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Results is the results.json document.
type Results struct {
	Status string  `json:"status"`
	Pass   int     `json:"pass"`
	Fail   int     `json:"fail"`
	Checks []Check `json:"checks"`
	// Identity proves what was verified and what answered (11.4.289(D)); the
	// commands fill it, the in-process gate-logic tests leave it nil.
	Identity *Identity `json:"identity,omitempty"`
}

// assertion is the panic payload of a failed Require; run() recovers it the way
// the Python harness catches AssertionError, so the first failed check aborts
// the run and is recorded as the single FAIL entry.
type assertion struct{ msg string }

func (a assertion) Error() string { return a.msg }

// Recorder accumulates checks. The zero value is ready to use.
type Recorder struct{ checks []Check }

// Require records a PASS, or aborts the run with the check name when cond is false.
func (r *Recorder) Require(cond bool, name string) {
	if !cond {
		panic(assertion{name})
	}
	r.checks = append(r.checks, Check{name, "PASS"})
}

// Skip records an honest SKIP; a SKIP is never counted as a PASS.
func (r *Recorder) Skip(name string) { r.checks = append(r.checks, Check{name, "SKIP"}) }

// Fail records a FAIL entry without aborting (used by cleanup paths).
func (r *Recorder) Fail(name string) { r.checks = append(r.checks, Check{name, "FAIL"}) }

// Fatalf aborts the run with a formatted message (the Python bare raise).
func (r *Recorder) Fatalf(format string, args ...any) { panic(assertion{fmt.Sprintf(format, args...)}) }

// Checks returns a copy of the recorded checks.
func (r *Recorder) Checks() []Check { return append([]Check(nil), r.checks...) }

// Guard runs fn and converts any panic (assertion or runtime error, mirroring
// Python's catch-all except Exception) into a FAIL entry. It reports whether fn
// completed without failure.
func (r *Recorder) Guard(fn func()) (ok bool) {
	defer func() {
		if v := recover(); v != nil {
			r.Fail(panicText(v))
			ok = false
		}
	}()
	fn()
	return true
}

func panicText(v any) string {
	switch t := v.(type) {
	case assertion:
		return t.msg
	case error:
		return t.Error()
	default:
		return fmt.Sprint(t)
	}
}

// Results computes the final document: status is FAIL when any check failed or
// when the caller already marked the run failed.
func (r *Recorder) Results(failed bool) Results {
	out := Results{Status: "PASS", Checks: r.Checks()}
	if out.Checks == nil {
		out.Checks = []Check{}
	}
	for _, c := range out.Checks {
		switch c.Status {
		case "PASS":
			out.Pass++
		case "FAIL":
			out.Fail++
		}
	}
	if failed || out.Fail > 0 {
		out.Status = "FAIL"
	}
	return out
}

// WriteResults writes results.json (indent 2, trailing newline) into dir.
func WriteResults(dir string, res Results) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "results.json"), append(raw, '\n'), 0o644)
}

// Summary is the one-line stdout record the Python verifiers print.
func (res Results) Summary() string {
	raw, _ := json.Marshal(map[string]any{"status": res.Status, "pass": res.Pass, "fail": res.Fail})
	return string(raw)
}

// ExitCode maps the verdict: 0 PASS, 1 FAIL.
func (res Results) ExitCode() int {
	if res.Status == "PASS" {
		return 0
	}
	return 1
}
