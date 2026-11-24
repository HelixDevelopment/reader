package uiverify

import (
	"strings"
	"testing"
)

// The fixture's golden-good and golden-bad cases run for real through
// `uiverify --fixture` inside the isolation container (qa/isolation/run_uiverify.sh,
// evidence ui-fixture-*). Here: the SKIP is honest outside the container, and the
// verdict logic of RunFixture is checked against engines that are wrong in a known way.
func TestFixtureLiveInsideContainer(t *testing.T) {
	if RequireContainer(DefaultIsolationProbe()) != nil {
		t.Skip("not inside the isolation container: the live fixture run happens in qa/isolation/run_uiverify.sh")
	}
	t.Log("inside the container: use `uiverify --fixture`")
}

func TestFixtureHTMLHasEveryGoldenBadCase(t *testing.T) {
	for _, s := range []string{"Bravo hidden-display", "Charlie hidden-visibility", "Delta zero-size", `class="dup gone"`, `class="gone">Save`, "disabled"} {
		if !strings.Contains(FixtureHTML, s) {
			t.Fatalf("fixture lost %q", s)
		}
	}
}

// engine answers one fixed way to every question, like a visibility engine
// stuck on "true" (reviewer mutation: isVisible returns true) or on "false".
type engine struct {
	Page
	yes bool
}

type engineBrowser struct{ e *engine }

func (b engineBrowser) NewPage(Viewport) (Page, error) { return b.e, nil }
func (b engineBrowser) Close() error                   { return nil }
func (e *engine) Goto(string) error                    { return nil }
func (e *engine) Close() error                         { return nil }
func (e *engine) Visible(Locator) (bool, error)        { return e.yes, nil }
func (e *engine) Count(Locator) (int, error) {
	if e.yes {
		return 1, nil
	}
	return 0, nil
}
func (e *engine) Click(Locator) error                   { return nil }
func (e *engine) Fill(Locator, string) error            { return nil }
func (e *engine) Select(Locator, string) error          { return nil }
func (e *engine) InnerText(Locator) (string, error)     { return "", nil }
func (e *engine) Style(Locator, string) (string, error) { return "", nil }
func (e *engine) Attr(Locator, string) (string, error)  { return "", nil }

func TestFixtureRejectsAnEngineStuckOnOneAnswer(t *testing.T) {
	for _, yes := range []bool{true, false} {
		checks, err := RunFixture(engineBrowser{&engine{yes: yes}}, "http://x/")
		if err != nil {
			t.Fatal(err)
		}
		failed := map[string]bool{}
		for _, c := range checks {
			if c.Status == "FAIL" {
				failed[c.ID] = true
			}
		}
		if len(failed) == 0 || len(failed) == len(checks) {
			t.Fatalf("yes=%v: %d of %d cases failed; a one-answer engine must fail some but not all", yes, len(failed), len(checks))
		}
		if yes && !failed["bad-hidden-display-none"] {
			t.Fatalf("an always-visible engine was not caught by bad-hidden-display-none: %v", failed)
		}
		if !yes && !failed["good-visible-text"] {
			t.Fatalf("a never-visible engine was not caught by good-visible-text: %v", failed)
		}
	}
}
