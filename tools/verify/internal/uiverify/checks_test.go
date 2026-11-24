package uiverify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var wantIDs = []string{
	"viewport-meta", "no-horizontal-overflow", "sample-data-disclosure", "search-filter", "reader-next-delta",
	"reader-font-delta", "save-passage", "book-detail", "metadata-decision-delta", "source-validation",
	"source-add-delta", "source-scan-state", "sync-conflict-delta", "theme-delta", "console-errors", "missing-assets",
}

func run(t *testing.T, faults ...string) Report {
	t.Helper()
	f := map[string]bool{}
	for _, x := range faults {
		f[x] = true
	}
	return RunAll(&fakeApp{faults: f}, "http://127.0.0.1:4173/", t.TempDir())
}

func TestHealthyAppPassesEverySixteenChecksOnFourViewports(t *testing.T) {
	r := run(t)
	if len(r.Viewports) != 4 {
		t.Fatalf("healthy app: %d viewports", len(r.Viewports))
	}
	for _, vr := range r.Viewports {
		var ids []string
		for _, c := range vr.Checks {
			ids = append(ids, c.ID)
			if c.Status != "PASS" {
				t.Errorf("healthy app: %s/%s = %s (%v)", vr.Viewport.Name, c.ID, c.Status, c.Evidence)
			}
		}
		if strings.Join(ids, ",") != strings.Join(wantIDs, ",") {
			t.Errorf("healthy app: %s check ids = %v", vr.Viewport.Name, ids)
		}
	}
	if r.Summary != (Summary{Pass: 64, Fail: 0, Skip: 0}) || r.SchemaVersion != 1 {
		t.Fatalf("healthy app: summary = %+v", r.Summary)
	}
}

// Paired mutants of the page under test: each seeded fault must fail exactly the check that guards it.
func TestEachSeededFaultFailsItsCheck(t *testing.T) {
	faults := map[string]string{
		"no-viewport-meta": "viewport-meta", "overflow": "no-horizontal-overflow", "wrong-disclosure": "sample-data-disclosure",
		"search-noop": "search-filter", "next-noop": "reader-next-delta", "font-noop": "reader-font-delta",
		"save-noop": "save-passage", "wrong-detail": "book-detail", "accept-noop": "metadata-decision-delta",
		"no-validation": "source-validation", "add-source-noop": "source-add-delta", "scan-noop": "source-scan-state",
		"sync-noop": "sync-conflict-delta", "theme-noop": "theme-delta", "console-error": "console-errors",
		"missing-asset": "missing-assets",
	}
	if len(faults) != len(wantIDs) {
		t.Fatalf("fault table covers %d of %d checks", len(faults), len(wantIDs))
	}
	for fault, id := range faults {
		r := run(t, fault)
		failed := map[string]bool{}
		for _, vr := range r.Viewports {
			for _, c := range vr.Checks {
				if c.Status == "FAIL" {
					failed[c.ID] = true
				}
			}
		}
		if !failed[id] {
			t.Errorf("fault %q: check %s did not FAIL (failed: %v)", fault, id, failed)
		}
		if r.Summary.Fail == 0 {
			t.Errorf("fault %q: summary reports no failure", fault)
		}
	}
}

func TestViewportErrorIsRecordedAndDoesNotAbortTheOthers(t *testing.T) {
	r := RunAll(&fakeApp{faults: map[string]bool{}, failNP: map[string]bool{"tablet": true}}, "http://x/", t.TempDir())
	for _, vr := range r.Viewports {
		if vr.Viewport.Name == "tablet" {
			if len(vr.Checks) != 1 || vr.Checks[0].ID != "viewport-run" || vr.Checks[0].Status != "FAIL" {
				t.Fatalf("tablet = %+v", vr.Checks)
			}
			ev, _ := json.Marshal(vr.Checks[0].Evidence)
			if !strings.Contains(string(ev), `"exception"`) || !strings.Contains(string(ev), "chromium failed to start") {
				t.Fatalf("evidence = %s", ev)
			}
		} else if len(vr.Checks) != 16 {
			t.Fatalf("%s has %d checks", vr.Viewport.Name, len(vr.Checks))
		}
	}
	if r.Summary.Fail != 1 || r.Summary.Pass != 48 {
		t.Fatalf("summary = %+v", r.Summary)
	}
}

func TestUnmodelledInteractionIsAnErrorNotAPass(t *testing.T) {
	// A check that talks to the page in a way the model (and so, by extension, the prototype contract) does not know must not pass silently.
	f := &fakeApp{faults: map[string]bool{}, w: 1440}
	if err := f.Click(CSS(".nope")); err == nil {
		t.Fatal("unmodelled click returned nil")
	}
}

func TestReportJSONShapeMatchesPython(t *testing.T) {
	r := run(t, "console-error", "missing-asset")
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	_ = json.Unmarshal(b, &top)
	for _, k := range []string{"schema_version", "base_url", "viewports", "summary"} {
		if _, ok := top[k]; !ok {
			t.Fatalf("missing top-level key %s in %s", k, b)
		}
	}
	sum := top["summary"].(map[string]any)
	for _, k := range []string{"pass", "fail", "skip"} {
		if _, ok := sum[k]; !ok {
			t.Fatalf("summary lacks %s", k)
		}
	}
	healthy, _ := json.Marshal(run(t))
	if !strings.Contains(string(healthy), `"id":"console-errors","status":"PASS","evidence":[]`) || !strings.Contains(string(healthy), `"id":"missing-assets","status":"PASS","evidence":[]`) {
		t.Fatalf("empty console/missing evidence must serialise as [] like Python: %s", healthy)
	}
	vp := top["viewports"].([]any)[0].(map[string]any)["viewport"].(map[string]any)
	if vp["name"] != "phone" || vp["width"].(float64) != 375 || vp["height"].(float64) != 812 {
		t.Fatalf("viewport = %v", vp)
	}
}

func TestScreenshotNamesMatchPython(t *testing.T) {
	dir := t.TempDir()
	app := &fakeApp{faults: map[string]bool{}}
	p, _ := app.NewPage(Viewports[0])
	if _, err := runChecks(p, "http://x/", dir, Viewports[0]); err != nil {
		t.Fatal(err)
	}
	fp := p.(*fakeApp)
	want := []string{filepath.Join(dir, "phone-library.png"), filepath.Join(dir, "phone-settings.png")}
	if strings.Join(fp.shots, "|") != strings.Join(want, "|") {
		t.Fatalf("shots = %v", fp.shots)
	}
}

// Parity with the Python source: the same check ids, in the same order, appear in scripts/verify_ui.py.
func TestCheckIDsMatchThePythonSourceWhenPresent(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "scripts", "verify_ui.py"))
	if err != nil {
		t.Skipf("scripts/verify_ui.py is gone (retired after parity): %v", err)
	}
	re := regexp.MustCompile(`check\("([a-z-]+)"`)
	var ids []string
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		ids = append(ids, m[1])
	}
	// the Python script also names the failure id "viewport-run" in its exception handler
	if got := strings.Join(ids, ","); got != strings.Join(wantIDs, ",") {
		t.Fatalf("python check ids = %s", got)
	}
}

func isoFixture(t *testing.T) (IsolationProbe, map[string]string) {
	t.Helper()
	d := t.TempDir()
	cenv := filepath.Join(d, ".containerenv")
	launch := filepath.Join(d, "launch.sh")
	net := filepath.Join(d, "net")
	for _, f := range []string{cenv, launch} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(net, "lo"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"HRDISO_ISOLATED": "1"}
	return IsolationProbe{ContainerEnv: cenv, LaunchMarker: launch, NetDir: net, Getenv: func(k string) string { return env[k] }}, env
}

func TestRequireContainerAcceptsTheSanctionedContainer(t *testing.T) {
	p, _ := isoFixture(t)
	if err := RequireContainer(p); err != nil {
		t.Fatalf("err = %v", err)
	}
}

// A rootful podman container, a toolbox or a distrobox has /run/.containerenv
// but not the marker only tools/isolation/run_container.sh provides.
func TestRequireContainerRefusesOutsideTheIsolationContainer(t *testing.T) {
	cases := map[string]func(p *IsolationProbe, env map[string]string){
		"no containerenv":     func(p *IsolationProbe, _ map[string]string) { p.ContainerEnv = filepath.Join(t.TempDir(), "absent") },
		"no launch marker":    func(p *IsolationProbe, _ map[string]string) { p.LaunchMarker = filepath.Join(t.TempDir(), "absent") },
		"no isolation env":    func(_ *IsolationProbe, env map[string]string) { delete(env, "HRDISO_ISOLATED") },
		"wrong isolation env": func(_ *IsolationProbe, env map[string]string) { env["HRDISO_ISOLATED"] = "0" },
		"host network": func(p *IsolationProbe, _ map[string]string) {
			_ = os.MkdirAll(filepath.Join(p.NetDir, "enp3s0"), 0o755)
		},
		"host display":       func(_ *IsolationProbe, env map[string]string) { env["DISPLAY"] = ":0" },
		"wayland":            func(_ *IsolationProbe, env map[string]string) { env["WAYLAND_DISPLAY"] = "wayland-0" },
		"unreadable net dir": func(p *IsolationProbe, _ map[string]string) { p.NetDir = filepath.Join(t.TempDir(), "absent") },
	}
	for name, mutate := range cases {
		p, env := isoFixture(t)
		mutate(&p, env)
		if err := RequireContainer(p); err != ErrNotIsolated {
			t.Errorf("%s: err = %v, want ErrNotIsolated", name, err)
		}
	}
	// the container-private DISPLAY is allowed
	p, env := isoFixture(t)
	env["DISPLAY"] = ":99"
	if err := RequireContainer(p); err != nil {
		t.Errorf("DISPLAY=:99: err = %v", err)
	}
}

func TestLocatorStringIsStableAndScoped(t *testing.T) {
	l := ByRoleButton("Scan").Within(ByText("Test OPDS", true).ClosestAncestor("article")).FirstMatch()
	want := `role(button "Scan") in[text("Test OPDS" exact=true) ancestor=article]`
	if l.String() != want {
		t.Fatalf("String = %s", l)
	}
	var back Locator
	if err := json.Unmarshal([]byte(l.json()), &back); err != nil || back.Scope == nil || back.Scope.Ancestor != "article" || !back.First {
		t.Fatalf("json round trip: %+v %v", back, err)
	}
}

func TestActionBudgetBoundsEveryBrowserCall(t *testing.T) {
	if got := actionBudget(0); got != 30*time.Second {
		t.Fatalf("default budget = %v", got)
	}
	if got := actionBudget(3 * time.Second); got != 6*time.Second {
		t.Fatalf("budget = %v", got)
	}
}

func TestEveryChromedpCallGoesThroughTheDeadline(t *testing.T) {
	src, err := os.ReadFile("driver_chromedp.go")
	if err != nil {
		t.Fatal(err)
	}
	n := strings.Count(string(src), "chromedp.Run(")
	if n != 2 {
		t.Fatalf("chromedp.Run is called %d times; only p.run and the bare launch may call it", n)
	}
}
