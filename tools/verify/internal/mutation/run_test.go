package mutation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests drive the runner against a tiny real Go module in a temp tree
// (real `go test`, no doubles). The paired mutants of the runner itself live in
// ../../mutation/selftest_catalogue.json and are run by `gomutate --selftest`.

const fxCalc = `package calc

func Add(a, b int) int { return a + b }

func Pos(x int) bool {
	if x > 0 {
		return true
	}
	return false
}
`

const fxTest = `package calc

import (
	"testing"
	"time"
)

func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Fatalf("add wrong: %d", Add(2, 3))
	}
}

func TestPosWeak(t *testing.T) {
	if !Pos(5) {
		t.Fatal("pos wrong")
	}
}

func TestAlwaysFails(t *testing.T) { t.Fatal("always fails") }

func TestSleeps(t *testing.T) { time.Sleep(25 * time.Second) }
`

func fixture(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	write(t, filepath.Join(root, "mod", "go.mod"), "module fx\n\ngo 1.21\n")
	write(t, filepath.Join(root, "mod", "calc", "calc.go"), fxCalc)
	write(t, filepath.Join(root, "mod", "calc", "calc_test.go"), fxTest)
	return root
}

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func m(id, old, new, test, expect string) Mutant {
	return Mutant{ID: id, Module: "mod", File: "calc/calc.go", Old: old, New: new, Package: "./calc", Test: test, Expect: expect}
}

func run(t *testing.T, root string, ms []Mutant, wall time.Duration) Summary {
	t.Helper()
	sum, err := Run(Config{Root: root, Wall: wall}, ms)
	if err != nil {
		t.Fatal(err)
	}
	return sum
}

func byID(s Summary, id string) Result {
	for _, r := range s.Results {
		if r.ID == id {
			return r
		}
	}
	return Result{}
}

func TestKilledSurvivedAndWrongAssertion(t *testing.T) {
	root := fixture(t)
	ms := []Mutant{
		m("killed", "return a + b", "return a - b", "TestAdd", "add wrong"),
		m("survivor", "if x > 0 {", "if x > 1 {", "TestPosWeak", "pos wrong"),
		m("wrong-assertion", "return a + b", "return a - b", "TestAdd", "some other text"),
	}
	sum := run(t, root, ms, 300*time.Second)
	if sum.Status != "FAIL" || sum.Pass != 1 || sum.Fail != 2 || sum.Mutants != 3 || sum.Controls != 2 {
		t.Fatalf("summary = %+v", sum)
	}
	if r := byID(sum, "killed"); r.Status != "PASS" || r.Reason != "rejected with the predicted assertion" || r.Exit == nil || *r.Exit == 0 {
		t.Fatalf("killed = %+v", r)
	}
	if r := byID(sum, "survivor"); r.Status != "FAIL" || r.Reason != "targeted test accepted the mutant" {
		t.Fatalf("survivor = %+v", r)
	}
	if r := byID(sum, "wrong-assertion"); r.Status != "FAIL" || !strings.HasPrefix(r.Reason, "test failed, but not with the predicted assertion") {
		t.Fatalf("wrong-assertion = %+v", r)
	}
	if !sum.OriginalTreeUnchanged {
		t.Fatal("original tree changed")
	}
	if got, _ := os.ReadFile(filepath.Join(root, "mod", "calc", "calc.go")); string(got) != fxCalc {
		t.Fatal("campaign modified the original file")
	}
}

func TestAllKilledIsPass(t *testing.T) {
	root := fixture(t)
	sum := run(t, root, []Mutant{m("killed", "return a + b", "return a - b", "TestAdd", "add wrong")}, 300*time.Second)
	if sum.Status != "PASS" || sum.Pass != 1 || sum.Fail != 0 {
		t.Fatalf("summary = %+v", sum)
	}
}

func TestAnchorMustOccurExactlyOnce(t *testing.T) {
	root := fixture(t)
	sum := run(t, root, []Mutant{
		m("absent", "does not occur", "x", "TestAdd", "add wrong"),
		m("twice", "return", "return", "TestAdd", "add wrong"), // "return" occurs 3 times
	}, 300*time.Second)
	if r := byID(sum, "absent"); r.Status != "FAIL" || r.Reason != "mutation anchor count 0" {
		t.Fatalf("absent = %+v", r)
	}
	if r := byID(sum, "twice"); r.Status != "FAIL" || r.Reason != "mutation anchor count 3" {
		t.Fatalf("twice = %+v", r)
	}
	if sum.Status != "FAIL" {
		t.Fatal("anchor failure must fail the campaign")
	}
}

func TestBuildFailureIsNotARejectedMutant(t *testing.T) {
	root := fixture(t)
	sum := run(t, root, []Mutant{m("broken-build", "return a + b", "return a +", "TestAdd", "x")}, 300*time.Second)
	r := byID(sum, "broken-build")
	if r.Status != "FAIL" || !strings.HasPrefix(r.Reason, "mutant never ran: the harness could not build the package (") {
		t.Fatalf("broken-build = %+v", r)
	}
}

func TestControlFailureFailsCampaign(t *testing.T) {
	root := fixture(t)
	sum := run(t, root, []Mutant{m("on-failing-test", "return a + b", "return a - b", "TestAlwaysFails", "always fails")}, 300*time.Second)
	c := byID(sum, "control-mod-TestAlwaysFails")
	if c.Status != "FAIL" || !strings.Contains(c.Reason, "cannot reject anything") {
		t.Fatalf("control = %+v", c)
	}
	if sum.Status != "FAIL" {
		t.Fatalf("status = %s", sum.Status)
	}
}

func TestNoVerdictOnTimeout(t *testing.T) {
	root := fixture(t)
	start := time.Now()
	sum := run(t, root, []Mutant{m("hangs", "return a + b", "return a - b", "TestSleeps", "x")}, 4*time.Second)
	r := byID(sum, "hangs")
	if r.Status != "FAIL" || !strings.HasPrefix(r.Reason, "no verdict: the targeted test did not finish") || r.Exit == nil || *r.Exit != 124 {
		t.Fatalf("hangs = %+v", r)
	}
	if time.Since(start) > 20*time.Second {
		t.Fatalf("harness waited %s; the wall budget was not enforced", time.Since(start))
	}
}

func TestMutantWithoutPredictedAssertionIsRefusedUpFront(t *testing.T) {
	root := fixture(t)
	sum := run(t, root, []Mutant{m("no-expect", "return a + b", "return a - b", "TestAdd", "")}, 300*time.Second)
	if sum.Status != "FAIL" || sum.Reason != "mutants with no predicted assertion" || len(sum.IDs) != 1 || sum.IDs[0] != "no-expect" || sum.Pass != 0 || sum.Fail != 1 {
		t.Fatalf("summary = %+v", sum)
	}
}

func TestEvidenceWritten(t *testing.T) {
	root := fixture(t)
	run(t, root, []Mutant{m("killed", "return a + b", "return a - b", "TestAdd", "add wrong")}, 300*time.Second)
	ev := filepath.Join(root, "evidence", "go-mutation")
	for _, f := range []string{"results.json", "killed.log", "control-mod-TestAdd.log"} {
		if _, err := os.Stat(filepath.Join(ev, f)); err != nil {
			t.Fatalf("missing %s: %v", f, err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(ev, "results.json"))
	var back map[string]any
	if json.Unmarshal(b, &back) != nil || back["status"] != "PASS" {
		t.Fatalf("results.json = %s", b)
	}
	log, _ := os.ReadFile(filepath.Join(ev, "killed.log"))
	if !strings.HasPrefix(string(log), "$ go test -trimpath ./calc -run ^TestAdd$ -count=1 -timeout") || !strings.Contains(string(log), "exit=1") {
		t.Fatalf("log = %q", log)
	}
	if _, err := os.Stat(filepath.Join(root, ".helix-verify", "go-mutation")); !os.IsNotExist(err) {
		t.Fatal("scratch work tree was not removed")
	}
}

func TestOnlyFiltersCatalogue(t *testing.T) {
	root := fixture(t)
	ms := []Mutant{
		m("a", "return a + b", "return a - b", "TestAdd", "add wrong"),
		m("b", "if x > 0 {", "if x > 1 {", "TestPosWeak", "pos wrong"),
	}
	sum, err := Run(Config{Root: root, Only: []string{"a"}}, ms)
	if err != nil || sum.Mutants != 1 || sum.Status != "PASS" {
		t.Fatalf("%+v %v", sum, err)
	}
	if _, err := Run(Config{Root: root, Only: []string{"nope"}}, ms); err == nil {
		t.Fatal("unknown --only id must be an error, never a vacuous PASS")
	}
}

func TestEmbeddedCatalogueIsWellFormed(t *testing.T) {
	ms, err := DefaultCatalogue()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 104 {
		t.Fatalf("catalogue has %d mutants, the Python catalogue had 104", len(ms))
	}
	seen := map[string]bool{}
	for _, x := range ms {
		if seen[x.ID] {
			t.Fatalf("duplicate id %s", x.ID)
		}
		seen[x.ID] = true
		if x.Expect == "" || x.Old == "" || x.Old == x.New || x.Module == "" || x.File == "" || x.Package == "" || x.Test == "" {
			t.Fatalf("malformed mutant %+v", x)
		}
	}
}
