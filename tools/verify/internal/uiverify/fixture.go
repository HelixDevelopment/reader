package uiverify

import "fmt"

// FixtureHTML is a page of KNOWN content for validating the hand-written
// locator and visibility engine (hv.js) that replaces Playwright's validated
// locators in this port. It holds golden-good cases (things that must be found
// or visible) and golden-bad cases (things that must NOT be found or visible),
// so an engine that returns true for everything, or false for everything, fails
// (constitution 11.4.107(10), 11.4.201). It is served by `uiverify --fixture`
// and, like every browser run, executes only inside the isolation container.
const FixtureHTML = `<!doctype html><html><head><meta charset="utf-8"><title>hv fixture</title>
<style>.zero{width:0;height:0;overflow:hidden}.hv{visibility:hidden}.gone{display:none}
#cover{position:absolute;left:0;top:0;right:0;bottom:0;background:rgba(0,0,0,.1)}</style></head><body>
<p>Alpha visible</p><p class="gone">Bravo hidden-display</p><p class="hv">Charlie hidden-visibility</p><p class="zero">Delta zero-size</p>
<div class="dup">dup one</div><div class="dup gone">dup two</div>
<div><span>Nested <b>leaf</b> text</span></div>
<input aria-label="Email address"><label for="pw">Password field</label><input id="pw">
<span id="lbl">Search box</span><input aria-labelledby="lbl">
<button>Save</button><button class="gone">Save</button>
<article><h3>Card A title</h3><button>Go</button></article><article><h3>Card B title</h3><button>Go</button></article>
<button id="inc" onclick="document.getElementById('count').textContent=String(+document.getElementById('count').textContent+1)">Increment</button><span id="count">0</span>
<button disabled>Disabled one</button>
<label>Echo <input id="echo-in" oninput="document.getElementById('echo').textContent=this.value"></label><span id="echo"></span>
<select aria-label="Pick" onchange="document.getElementById('chosen').textContent=this.value"><option value="one">One</option><option value="two">Two</option></select><span id="chosen"></span>
<div id="colored" data-x="y" style="color:rgb(255,0,0)">Colored</div>
<div style="position:relative;display:inline-block"><button id="covered">Covered button</button><div id="cover"></div></div>
</body></html>`

// RunFixture drives one page against FixtureHTML served at base and returns
// one Check per case. A case passes only when the engine's answer equals the
// known answer, so the golden-bad cases are what detect an always-true engine.
func RunFixture(b Browser, base string) ([]Check, error) {
	p, err := b.NewPage(Viewports[2])
	if err != nil {
		return nil, err
	}
	defer p.Close()
	if err := p.Goto(base); err != nil {
		return nil, err
	}
	var out []Check
	rec := func(id string, got, want any, err error) {
		c := Check{ID: id, Evidence: map[string]any{"got": fmt.Sprint(got), "want": fmt.Sprint(want)}}
		switch {
		case err != nil:
			c.Status, c.Evidence = "FAIL", map[string]any{"error": err.Error(), "want": fmt.Sprint(want)}
		case fmt.Sprint(got) == fmt.Sprint(want):
			c.Status = "PASS"
		default:
			c.Status = "FAIL"
		}
		out = append(out, c)
	}
	vis := func(id string, l Locator, want bool) { v, err := p.Visible(l); rec(id, v, want, err) }
	cnt := func(id string, l Locator, want int) { n, err := p.Count(l); rec(id, n, want, err) }
	mustFail := func(id string, err error) {
		c := Check{ID: id, Evidence: map[string]any{"want": "an error"}}
		if err != nil {
			c.Status, c.Evidence = "PASS", map[string]any{"error": err.Error()}
		} else {
			c.Status = "FAIL"
		}
		out = append(out, c)
	}

	vis("good-visible-text", ByText("Alpha visible", true), true)
	vis("bad-hidden-display-none", ByText("Bravo hidden-display", false), false)
	vis("bad-hidden-visibility", ByText("Charlie hidden-visibility", false), false)
	vis("bad-hidden-zero-size", ByText("Delta zero-size", false), false)
	vis("bad-absent-element", ByText("no such text anywhere", false), false)
	cnt("good-count-without-visible-filter", CSS(".dup"), 2)
	cnt("bad-visible-filter-drops-hidden", Locator{Kind: "css", Value: ".dup", Visible: true}, 1)
	cnt("good-text-substring-case-insensitive", ByText("alpha VIS", false), 1)
	cnt("bad-text-exact-is-case-sensitive", ByText("alpha vis", true), 0)
	cnt("good-text-deepest-element-only", ByText("Nested leaf text", false), 1)
	cnt("good-label-aria-label", ByLabel("Email address", true), 1)
	cnt("good-label-for", ByLabel("Password field", true), 1)
	cnt("good-label-aria-labelledby", ByLabel("Search box", true), 1)
	cnt("bad-label-absent", ByLabel("no such label", false), 0)
	cnt("good-role-button-visible-only", ByRoleButton("Save"), 1)
	cnt("bad-role-button-absent", ByRoleButton("no such button"), 0)
	cnt("good-scope-ancestor", ByText("Card B title", true).ClosestAncestor("article"), 1)
	cnt("good-scoped-button", ByRoleButton("Go").Within(ByText("Card B title", true).ClosestAncestor("article")), 1)
	cnt("bad-unscoped-button-matches-both", ByRoleButton("Go"), 2)

	err = p.Click(CSS("#inc"))
	txt, terr := p.InnerText(CSS("#count"))
	if err == nil {
		err = terr
	}
	rec("good-click-increments", txt, "1", err)
	err = p.Fill(ByLabel("Echo", false), "hello")
	txt, terr = p.InnerText(CSS("#echo"))
	if err == nil {
		err = terr
	}
	rec("good-fill-fires-input", txt, "hello", err)
	err = p.Select(ByLabel("Pick", true), "Two")
	txt, terr = p.InnerText(CSS("#chosen"))
	if err == nil {
		err = terr
	}
	rec("good-select-fires-change", txt, "two", err)
	st, err := p.Style(CSS("#colored"), "color")
	rec("good-style", st, "rgb(255, 0, 0)", err)
	at, err := p.Attr(CSS("#colored"), "data-x")
	rec("good-attr", at, "y", err)
	_, aerr := p.Attr(CSS("#no-such-element"), "data-x")
	mustFail("bad-attr-missing-element-errors", aerr)
	_, serr := p.Style(CSS("#no-such-element"), "color")
	mustFail("bad-style-missing-element-errors", serr)
	missing, err := p.Attr(CSS("#colored"), "data-absent-attr")
	rec("good-attr-missing-attribute-is-empty", missing, "", err)
	mustFail("bad-click-disabled-button-is-refused", p.Click(Locator{Kind: "css", Value: "button", HasText: "Disabled one"}))
	mustFail("bad-click-covered-button-is-refused", p.Click(CSS("#covered")))
	mustFail("bad-click-hidden-button-is-refused", p.Click(Locator{Kind: "css", Value: "button.gone"}))
	return out, nil
}
