// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"digital.vasic.helixqa/pkg/autonomous"
	"digital.vasic.helixqa/pkg/config"
	"digital.vasic.helixqa/pkg/testbank"
)

type fakeEx struct {
	calls   []string
	results map[string][]autonomous.ActionResult // per value, consumed in order; last repeats
}

func (f *fakeEx) Execute(_ context.Context, v string) autonomous.ActionResult {
	f.calls = append(f.calls, v)
	rs := f.results[v]
	if len(rs) == 0 {
		return autonomous.ActionResult{Success: true, Message: "ok"}
	}
	r := rs[0]
	if len(rs) > 1 {
		f.results[v] = rs[1:]
	}
	return r
}

func bank(steps ...testbank.TestStep) *testbank.BankFile {
	return &testbank.BankFile{Name: "t", TestCases: []testbank.TestCase{{ID: "T-1", Name: "n", Steps: steps}}}
}

var goodShoot Shooter = func(context.Context) ([]byte, error) { return []byte("png"), nil }
var goodSave = func(n string, b []byte) (string, string, error) { return n, "sha", nil }

func opts(retry time.Duration) Options {
	return Options{Retry: retry, Shoot: goodShoot, Save: goodSave, AllowUnpinned: true}
}

func run(bf *testbank.BankFile, ex Executor, o Options) *Report {
	return RunBank(context.Background(), bf, ex, o)
}

func pw(a string) testbank.TestStep { return testbank.TestStep{Name: a, Action: "playwright: " + a} }

func TestAllPassIsOK(t *testing.T) {
	r := run(bank(pw("navigate http://x/"), pw("assertVisible a")), &fakeEx{}, opts(0))
	if !r.OK() || r.Pass != 2 || r.Cases[0].Verdict != "PASS" || r.Cases[0].Steps[0].Attempts != 1 {
		t.Fatalf("want OK 2 pass, got %+v", r)
	}
}

func TestSkipStepIsNeverPass(t *testing.T) {
	s := pw("navigate http://x/")
	s.Skip, s.SkipReason = true, "pending"
	f := &fakeEx{}
	r := run(bank(s), f, opts(0))
	if r.OK() || r.Skip != 1 || r.Cases[0].Verdict != "SKIP" || len(f.calls) != 0 {
		t.Fatalf("skip must not pass and must not execute: %+v calls=%v", r, f.calls)
	}
}

func TestOnePassPlusOneSkipIsNotOK(t *testing.T) {
	s := pw("click b")
	s.Skip = true
	r := run(bank(pw("navigate u"), s), &fakeEx{}, opts(0))
	if r.OK() || r.Pass != 1 || r.Skip != 1 {
		t.Fatalf("a single skipped step must make the run not OK: %+v", r)
	}
}

func TestNonPlaywrightStepIsSkipNotPass(t *testing.T) {
	r := run(bank(testbank.TestStep{Name: "h", Action: "http: GET /healthz"}), &fakeEx{}, opts(0))
	if r.OK() || r.Skip != 1 {
		t.Fatalf("non-playwright step must SKIP, got %+v", r)
	}
}

func TestExecutorSkippedOrBlockedIsSkip(t *testing.T) {
	for _, res := range []autonomous.ActionResult{{Skipped: true}, {Blocked: true}} {
		f := &fakeEx{results: map[string][]autonomous.ActionResult{"navigate u": {res}}}
		r := run(bank(pw("navigate u")), f, opts(0))
		if r.OK() || r.Skip != 1 {
			t.Fatalf("%+v must be SKIP: %+v", res, r)
		}
	}
}

func TestFailStopsCaseAndIsNotOK(t *testing.T) {
	f := &fakeEx{results: map[string][]autonomous.ActionResult{"click b": {{Success: false, Message: "boom"}}}}
	r := run(bank(pw("click b"), pw("click c")), f, opts(0))
	if r.OK() || r.Fail != 1 || len(r.Cases[0].Steps) != 1 || r.Cases[0].Verdict != "FAIL" {
		t.Fatalf("fail must stop the case: %+v", r)
	}
}

func TestEmptyBankIsNotOK(t *testing.T) {
	if (&Report{}).OK() || run(&testbank.BankFile{}, &fakeEx{}, opts(0)).OK() {
		t.Fatal("a bank with no executed steps must not be OK")
	}
}

func TestAssertRetriesUntilTrueAndRecordsAttempts(t *testing.T) {
	f := &fakeEx{results: map[string][]autonomous.ActionResult{
		"assertNotVisible n": {{Success: false}, {Success: false}, {Success: true, Message: "gone"}},
	}}
	r := run(bank(pw("assertNotVisible n")), f, opts(3*time.Second))
	if !r.OK() || len(f.calls) != 3 || r.Cases[0].Steps[0].Attempts != 3 {
		t.Fatalf("assert should retry to success and record 3 attempts: %+v calls=%d", r, len(f.calls))
	}
	f2 := &fakeEx{results: map[string][]autonomous.ActionResult{"click b": {{Success: false}, {Success: true}}}}
	r2 := run(bank(pw("click b")), f2, opts(3*time.Second))
	if r2.OK() || len(f2.calls) != 1 || r2.Cases[0].Steps[0].Attempts != 1 {
		t.Fatalf("click must never be retried: %+v calls=%d", r2, len(f2.calls))
	}
}

func TestAssertRetryIsBounded(t *testing.T) {
	f := &fakeEx{results: map[string][]autonomous.ActionResult{"assertVisible a": {{Success: false, Message: "NOT visible"}}}}
	start := time.Now()
	r := run(bank(pw("assertVisible a")), f, opts(500*time.Millisecond))
	if r.OK() || r.Fail != 1 || time.Since(start) > 3*time.Second {
		t.Fatalf("persistent false assert must FAIL within the window: %+v after %v", r, time.Since(start))
	}
}

func TestScreenshotRecordedPerPassedStep(t *testing.T) {
	saved := map[string]string{}
	o := opts(0)
	o.Save = func(n string, b []byte) (string, string, error) { saved[n] = string(b); return n, "sha", nil }
	r := run(bank(pw("navigate u"), pw("click b")), &fakeEx{}, o)
	if len(saved) != 2 || r.Cases[0].Steps[1].SHA256 != "sha" || !strings.HasPrefix(r.Cases[0].Steps[0].Screenshot, "T-1-step01") {
		t.Fatalf("screenshots: %v %+v", saved, r)
	}
}

func TestScreenshotFailureWithholdsPass(t *testing.T) {
	cases := map[string]Options{
		"capture error": {Shoot: func(context.Context) ([]byte, error) { return nil, errors.New("boom") }, Save: goodSave, AllowUnpinned: true},
		"empty png":     {Shoot: func(context.Context) ([]byte, error) { return nil, nil }, Save: goodSave, AllowUnpinned: true},
		"save error":    {Shoot: goodShoot, Save: func(string, []byte) (string, string, error) { return "", "", errors.New("disk full") }, AllowUnpinned: true},
		"empty path":    {Shoot: goodShoot, Save: func(string, []byte) (string, string, error) { return "", "h", nil }, AllowUnpinned: true},
		"no shooter":    {AllowUnpinned: true},
	}
	for name, o := range cases {
		r := run(bank(pw("navigate u")), &fakeEx{}, o)
		if r.OK() || r.Fail != 1 || r.Pass != 0 {
			t.Errorf("%s: a PASS must require its stored screenshot: %+v", name, r)
		}
	}
}

func pinned(cases, steps int, b *testbank.BankFile) *testbank.BankFile {
	b.Metadata = map[string]any{"expected_cases": cases, "expected_steps": steps}
	return b
}

func TestPinnedCounts(t *testing.T) {
	o := opts(0)
	o.AllowUnpinned = false
	if r := run(pinned(1, 2, bank(pw("a b"), pw("c d"))), &fakeEx{}, o); !r.OK() {
		t.Fatalf("matching pins must be OK: %+v", r)
	}
	for name, b := range map[string]*testbank.BankFile{
		"bank shrank": pinned(1, 3, bank(pw("a b"), pw("c d"))),
		"case count":  pinned(2, 2, bank(pw("a b"), pw("c d"))),
		"unpinned":    bank(pw("a b"), pw("c d")),
		"only one pinned": func() *testbank.BankFile {
			x := bank(pw("a b"))
			x.Metadata = map[string]any{"expected_cases": 1}
			return x
		}(),
	} {
		r := run(b, &fakeEx{}, o)
		if r.OK() || len(r.PinErrors) == 0 {
			t.Errorf("%s: must be refused: %+v", name, r)
		}
	}
}

func TestEachPinFiresIndependently(t *testing.T) {
	o := opts(0)
	o.AllowUnpinned = false
	sk := pw("c d")
	sk.Skip = true
	// 3 steps, 2 pass (pin says 2): only the step-count pin is wrong
	r := run(pinned(1, 2, bank(pw("a b"), pw("e f"), sk)), &fakeEx{}, o)
	if len(r.PinErrors) != 1 || !strings.Contains(r.PinErrors[0], "steps in the selected cases") {
		t.Fatalf("step-count pin must fire alone: %+v", r.PinErrors)
	}
	// 2 steps (pin says 2) but one is skipped: only the pass-count pin is wrong
	r = run(pinned(1, 2, bank(pw("a b"), sk)), &fakeEx{}, o)
	if len(r.PinErrors) != 1 || !strings.Contains(r.PinErrors[0], "steps PASSed") {
		t.Fatalf("pass-count pin must fire alone: %+v", r.PinErrors)
	}
}

func TestScreenshotNamesAreSanitised(t *testing.T) {
	saved := []string{}
	o := opts(0)
	o.Save = func(n string, b []byte) (string, string, error) { saved = append(saved, n); return n, "sha", nil }
	b := &testbank.BankFile{Name: "t", TestCases: []testbank.TestCase{{ID: "../../etc/x y", Steps: []testbank.TestStep{pw("a b")}}, {ID: "..", Steps: []testbank.TestStep{pw("a b")}}, {ID: "", Steps: []testbank.TestStep{pw("a b")}}}}
	run(b, &fakeEx{}, o)
	if len(saved) != 3 {
		t.Fatalf("saved %v", saved)
	}
	for _, n := range saved {
		if strings.ContainsAny(n, "/\\ ") || strings.Contains(n, "..") || !strings.HasSuffix(n, ".png") || strings.HasPrefix(n, "-") {
			t.Errorf("unsafe screenshot name %q", n)
		}
	}
}

func TestDuplicateCaseIDsRejected(t *testing.T) {
	b := &testbank.BankFile{Name: "t", TestCases: []testbank.TestCase{{ID: "A", Steps: []testbank.TestStep{pw("a b")}}, {ID: "A", Steps: []testbank.TestStep{pw("c d")}}}}
	r := run(b, &fakeEx{}, opts(0))
	if r.OK() || len(r.PinErrors) == 0 || !strings.Contains(r.PinErrors[0], "duplicate case id") {
		t.Fatalf("duplicate ids must be refused: %+v", r)
	}
	b.TestCases[1].ID = "A/" // sanitises to A_ : distinct, fine
	if r := run(b, &fakeEx{}, opts(0)); !r.OK() {
		t.Fatalf("distinct sanitised ids ok: %+v", r)
	}
	b.TestCases[1].ID = "a b" // "a_b"
	b.TestCases[0].ID = "a_b"
	if r := run(b, &fakeEx{}, opts(0)); r.OK() {
		t.Fatalf("ids colliding after sanitising must be refused: %+v", r)
	}
}

func TestReportCarriesPins(t *testing.T) {
	o := opts(0)
	o.AllowUnpinned = false
	r := run(pinned(1, 1, bank(pw("a b"))), &fakeEx{}, o)
	if r.ExpectedCases != 1 || r.ExpectedSteps != 1 {
		t.Fatalf("pins must be in the report: %+v", r)
	}
}

func TestPlatformFilterHonoured(t *testing.T) {
	b := &testbank.BankFile{Name: "t", TestCases: []testbank.TestCase{
		{ID: "W", Steps: []testbank.TestStep{pw("a b")}, Platforms: []config.Platform{"web"}},
		{ID: "A", Steps: []testbank.TestStep{pw("c d")}, Platforms: []config.Platform{"android"}},
		{ID: "ALL", Steps: []testbank.TestStep{pw("e f")}},
	}}
	o := opts(0)
	o.Platform = "web"
	f := &fakeEx{}
	r := run(b, f, o)
	if !r.OK() || len(r.Cases) != 2 || len(r.Excluded) != 1 || r.Excluded[0] != "A" || len(f.calls) != 2 {
		t.Fatalf("android case must be excluded and not executed: %+v calls=%v", r, f.calls)
	}
	o.Platform = "desktop"
	if r := run(b, &fakeEx{}, o); len(r.Cases) != 1 || r.Cases[0].ID != "ALL" {
		t.Fatalf("only platform-agnostic case should run on desktop: %+v", r)
	}
	o.Platform = "ios"
	b.TestCases = b.TestCases[:2]
	if r := run(b, &fakeEx{}, o); r.OK() {
		t.Fatalf("no runnable case must not be OK: %+v", r)
	}
}

func TestPageStateRecordedAndViewportEnforced(t *testing.T) {
	o := opts(0)
	n := 0
	o.State = func(context.Context) (PageState, error) {
		n++
		return PageState{URL: "http://x/", Width: 1920, Height: 1080, TextSHA: fmt.Sprintf("t%d", n)}, nil
	}
	o.ExpectViewport = [2]int{1920, 1080}
	r := run(bank(pw("a b"), pw("c d")), &fakeEx{}, o)
	if !r.OK() || r.Cases[0].Steps[0].Page == nil || r.Cases[0].Steps[0].Page.TextSHA == r.Cases[0].Steps[1].Page.TextSHA {
		t.Fatalf("page state must be recorded per step: %+v", r)
	}
	o.State = func(context.Context) (PageState, error) { return PageState{Width: 1280, Height: 757}, nil }
	if r := run(bank(pw("a b")), &fakeEx{}, o); r.OK() || !strings.Contains(r.Cases[0].Steps[0].Message, "viewport") {
		t.Fatalf("viewport mismatch must FAIL: %+v", r)
	}
	o.State = func(context.Context) (PageState, error) { return PageState{}, errors.New("cdp gone") }
	if r := run(bank(pw("a b")), &fakeEx{}, o); r.OK() {
		t.Fatalf("a failing state probe must withhold PASS: %+v", r)
	}
}

// The checked-in bank must have no _skip step and only playwright steps (the HRD-038 acceptance).
func TestCheckedInBankHasNoSkipAndOnlyPlaywright(t *testing.T) {
	bf, err := testbank.LoadFile(bankPath())
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, tc := range bf.TestCases {
		for _, st := range tc.Steps {
			n++
			if typ, _ := st.ParseAction(); st.Skip || typ != testbank.ActionTypePlaywright {
				t.Errorf("%s/%s: skip=%v type=%v", tc.ID, st.Name, st.Skip, typ)
			}
			if _, val := st.ParseAction(); strings.HasPrefix(val, "fill ") {
				sel, _ := splitFill(val)
				if strings.ContainsAny(sel, " \t") {
					t.Errorf("%s/%s: fill selector %q has whitespace; upstream splits at the first space", tc.ID, st.Name, sel)
				}
			}
		}
	}
	ec, _ := intMeta(bf.Metadata, "expected_cases")
	es, _ := intMeta(bf.Metadata, "expected_steps")
	if ec != len(bf.TestCases) || es != n {
		t.Fatalf("bank pins expected_cases=%d expected_steps=%d but has %d cases %d steps", ec, es, len(bf.TestCases), n)
	}
	if len(bf.TestCases) != 2 || n < 10 {
		t.Fatalf("expected the 2 web cases with real steps, got %d cases %d steps", len(bf.TestCases), n)
	}
}

func splitFill(v string) (string, string) {
	rest := strings.TrimSpace(strings.TrimPrefix(v, "fill "))
	if i := strings.IndexAny(rest, " \t"); i > 0 {
		return rest[:i], rest[i+1:]
	}
	return rest, ""
}

func bankPath() string {
	if p := os.Getenv("HELIX_WEB_BANK"); p != "" {
		return p
	}
	return "../../qa/helixqa/banks/helix-web.yaml"
}
