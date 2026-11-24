// SPDX-License-Identifier: Apache-2.0
// Package main: HRD-038 web bank runner. `helixqa http` never executes playwright: steps (it reports
// "SKIP (no http: steps)" and exits 0), and the only upstream path to them is the LLM-driven
// `autonomous` pipeline. This runner drives the SAME upstream executor
// (autonomous.PlaywrightExecutor, over a CDP endpoint) from the SAME bank file, deterministically,
// with a per-step screenshot + sha256, and is strict: SKIP, BLOCKED and non-playwright steps are
// never a pass, and the process exits non-zero unless every step PASSed.
package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"digital.vasic.helixqa/pkg/autonomous"
	"digital.vasic.helixqa/pkg/testbank"
)

// Executor is the subset of autonomous.PlaywrightExecutor the runner needs.
type Executor interface {
	Execute(ctx context.Context, value string) autonomous.ActionResult
}

// Shooter captures the current page as PNG bytes.
type Shooter func(ctx context.Context) ([]byte, error)

// PageState is what the page looked like after a step: its URL, viewport and a hash of its visible text, so the
// evidence distinguishes steps even when two screenshots are byte-identical (assertion-only steps share state).
type PageState struct {
	URL     string `json:"url"`
	Width   int    `json:"viewport_width"`
	Height  int    `json:"viewport_height"`
	TextSHA string `json:"dom_text_sha256"`
}

// Options configure a run. Zero value = strict defaults except Retry (0 = no retry).
type Options struct {
	Retry          time.Duration
	Platform       string  // only cases whose platforms are empty or include it run ("" = no filter)
	Shoot          Shooter // required: a PASS needs its screenshot file
	Save           func(name string, png []byte) (path, sha string, err error)
	State          func(ctx context.Context) (PageState, error) // optional page-state probe
	ExpectViewport [2]int                                       // when non-zero, every step's viewport must equal it
	AllowUnpinned  bool                                         // false: the bank must pin expected_cases and expected_steps
}

type StepReport struct {
	Name       string     `json:"name"`
	Action     string     `json:"action"`
	Verdict    string     `json:"verdict"` // PASS | FAIL | SKIP
	Message    string     `json:"message"`
	Attempts   int        `json:"attempts"`
	Screenshot string     `json:"screenshot,omitempty"`
	SHA256     string     `json:"screenshot_sha256,omitempty"`
	Page       *PageState `json:"page_state,omitempty"`
}

type CaseReport struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Verdict string       `json:"verdict"`
	Steps   []StepReport `json:"steps"`
}

type Report struct {
	Bank          string       `json:"bank"`
	Pass          int          `json:"pass"`
	Fail          int          `json:"fail"`
	Skip          int          `json:"skip"`
	Cases         []CaseReport `json:"cases"`
	Excluded      []string     `json:"excluded_by_platform,omitempty"`
	ExpectedCases int          `json:"expected_cases,omitempty"`
	ExpectedSteps int          `json:"expected_steps,omitempty"`
	PinErrors     []string     `json:"pin_errors,omitempty"`
}

// OK is true only when at least one step PASSed and none FAILed or was skipped.
func (r *Report) OK() bool { return r.Fail == 0 && r.Skip == 0 && r.Pass > 0 && len(r.PinErrors) == 0 }

// assertVerb reports whether a playwright action value is an eventually-consistent assertion.
// upstream isVisible does not wait, and the library search is debounced (250 ms) + server-side, so
// assertions are retried until `retry` elapses; a plain navigate/click/fill is never retried.
func assertVerb(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(v, "assertvisible ") || strings.HasPrefix(v, "assertnotvisible ")
}

func runStep(ctx context.Context, ex Executor, value string, retry time.Duration) (autonomous.ActionResult, int) {
	res := ex.Execute(ctx, value)
	attempts := 1
	if !assertVerb(value) {
		return res, attempts
	}
	deadline := time.Now().Add(retry)
	for !res.Success && !res.Skipped && !res.Blocked && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		res = ex.Execute(ctx, value)
		attempts++
	}
	return res, attempts
}

func platformOK(tc *testbank.TestCase, platform string) bool {
	if platform == "" || len(tc.Platforms) == 0 {
		return true
	}
	for _, p := range tc.Platforms {
		if string(p) == platform {
			return true
		}
	}
	return false
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// safeName makes a bank id usable as a file-name component: no path separators, no "..", never empty.
func safeName(s string) string {
	s = unsafeName.ReplaceAllString(s, "_")
	s = strings.ReplaceAll(s, "..", "_")
	if s == "" || s == "." {
		s = "_"
	}
	return s
}

func intMeta(m map[string]any, k string) (int, bool) {
	switch v := m[k].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	}
	return 0, false
}

// RunBank executes the cases of bf selected by o.Platform.
func RunBank(ctx context.Context, bf *testbank.BankFile, ex Executor, o Options) *Report {
	rep := &Report{Bank: bf.Name}
	steps := 0
	seen := map[string]bool{}
	for i := range bf.TestCases {
		tc := &bf.TestCases[i]
		if seen[tc.ID] || (tc.ID != "" && seen[safeName(tc.ID)]) {
			rep.PinErrors = append(rep.PinErrors, fmt.Sprintf("duplicate case id %q", tc.ID))
			continue
		}
		seen[tc.ID], seen[safeName(tc.ID)] = true, true
		if !platformOK(tc, o.Platform) {
			rep.Excluded = append(rep.Excluded, tc.ID)
			continue
		}
		cr := CaseReport{ID: tc.ID, Name: tc.Name, Verdict: "PASS"}
		for si := range tc.Steps {
			steps++
			st := &tc.Steps[si]
			sr := StepReport{Name: st.Name, Action: st.Action}
			typ, val := st.ParseAction()
			switch {
			case st.Skip:
				sr.Verdict, sr.Message = "SKIP", "bank step marked _skip: "+st.SkipReason
			case typ != testbank.ActionTypePlaywright:
				sr.Verdict, sr.Message = "SKIP", fmt.Sprintf("action type %q is not playwright: not executable by this runner", typ)
			default:
				res, attempts := runStep(ctx, ex, val, o.Retry)
				sr.Attempts, sr.Message = attempts, res.Message
				switch {
				case res.Skipped || res.Blocked:
					sr.Verdict = "SKIP"
				case res.Success:
					sr.Verdict = "PASS"
				default:
					sr.Verdict = "FAIL"
				}
				if sr.Verdict == "PASS" {
					// A PASS needs its evidence: the screenshot must be captured AND stored.
					if o.Shoot == nil || o.Save == nil {
						sr.Verdict, sr.Message = "FAIL", "no screenshot capability configured: a PASS requires a stored screenshot"
					} else if png, err := o.Shoot(ctx); err != nil || len(png) == 0 {
						sr.Verdict, sr.Message = "FAIL", fmt.Sprintf("screenshot capture failed (err=%v, %d bytes): PASS withheld", err, len(png))
					} else if p, h, err := o.Save(fmt.Sprintf("%s-step%02d.png", safeName(tc.ID), si+1), png); err != nil || p == "" || h == "" {
						sr.Verdict, sr.Message = "FAIL", fmt.Sprintf("screenshot not stored (err=%v): PASS withheld", err)
					} else {
						sr.Screenshot, sr.SHA256 = p, h
					}
				}
				if o.State != nil {
					if ps, err := o.State(ctx); err == nil {
						sr.Page = &ps
						if sr.Verdict == "PASS" && o.ExpectViewport != [2]int{} && [2]int{ps.Width, ps.Height} != o.ExpectViewport {
							sr.Verdict, sr.Message = "FAIL", fmt.Sprintf("viewport %dx%d != expected %dx%d", ps.Width, ps.Height, o.ExpectViewport[0], o.ExpectViewport[1])
						}
					} else if sr.Verdict == "PASS" {
						sr.Verdict, sr.Message = "FAIL", "page state probe failed: "+err.Error()
					}
				}
			}
			switch sr.Verdict {
			case "PASS":
				rep.Pass++
			case "FAIL":
				rep.Fail++
				cr.Verdict = "FAIL"
			default:
				rep.Skip++
				if cr.Verdict == "PASS" {
					cr.Verdict = "SKIP"
				}
			}
			cr.Steps = append(cr.Steps, sr)
			if sr.Verdict == "FAIL" {
				break // later steps depend on earlier ones
			}
		}
		rep.Cases = append(rep.Cases, cr)
	}
	// Pinned counts: a bank that silently shrank must not stay green.
	ec, okC := intMeta(bf.Metadata, "expected_cases")
	es, okS := intMeta(bf.Metadata, "expected_steps")
	if okC && okS {
		rep.ExpectedCases, rep.ExpectedSteps = ec, es
	}
	switch {
	case !okC || !okS:
		if !o.AllowUnpinned {
			rep.PinErrors = append(rep.PinErrors, "bank metadata must pin expected_cases and expected_steps")
		}
	default:
		if len(rep.Cases) != ec {
			rep.PinErrors = append(rep.PinErrors, fmt.Sprintf("ran %d cases, bank pins %d", len(rep.Cases), ec))
		}
		if steps != es {
			rep.PinErrors = append(rep.PinErrors, fmt.Sprintf("bank has %d steps in the selected cases, bank pins %d", steps, es))
		}
		if rep.Pass != es {
			rep.PinErrors = append(rep.PinErrors, fmt.Sprintf("%d steps PASSed, bank pins %d", rep.Pass, es))
		}
	}
	return rep
}
