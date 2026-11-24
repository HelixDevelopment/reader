// Package uiverify is the Go port of scripts/verify_ui.py: deterministic
// browser checks for the static Helix product prototype (demos/interactive_static_demo).
//
// The checks are written against the small Page interface so that the same
// check code is exercised hermetically (fake page, unit-class evidence) and
// live (chromedp driving Chromium, which must run inside the rootless
// isolation container, never on the operator's desktop).
package uiverify

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Locator describes an element query with the semantics of the Playwright
// locators the Python script used (substring/case-insensitive unless Exact).
type Locator struct {
	Kind     string   `json:"kind"`               // css | text | label | role
	Value    string   `json:"value,omitempty"`    // css selector, text, or label text
	Exact    bool     `json:"exact,omitempty"`    // text/label/role-name: exact match
	HasText  string   `json:"hasText,omitempty"`  // css: keep elements whose text contains this
	Visible  bool     `json:"visible,omitempty"`  // keep only visible matches (":visible")
	Role     string   `json:"role,omitempty"`     // role: "button"
	Name     string   `json:"name,omitempty"`     // role: accessible name
	Scope    *Locator `json:"scope,omitempty"`    // search inside the first-level matches of Scope
	Ancestor string   `json:"ancestor,omitempty"` // map each match to its closest ancestor with this tag
	First    bool     `json:"first,omitempty"`    // keep only the first match
}

// CSS matches a CSS selector.
func CSS(sel string) Locator { return Locator{Kind: "css", Value: sel} }

// VisibleButton is the Python visible_button helper.
func VisibleButton(text string) Locator {
	return Locator{Kind: "css", Value: "button", HasText: text, Visible: true, First: true}
}

// ByText matches the deepest elements containing the text.
func ByText(text string, exact bool) Locator { return Locator{Kind: "text", Value: text, Exact: exact} }

// ByLabel matches form controls by aria-label, <label> or aria-labelledby.
func ByLabel(text string, exact bool) Locator {
	return Locator{Kind: "label", Value: text, Exact: exact}
}

// ByRoleButton matches buttons by accessible name.
func ByRoleButton(name string) Locator { return Locator{Kind: "role", Role: "button", Name: name} }

// Within scopes l to the matches of s.
func (l Locator) Within(s Locator) Locator { l.Scope = &s; return l }

// ClosestAncestor maps each match to its closest ancestor element with the given tag.
func (l Locator) ClosestAncestor(tag string) Locator { l.Ancestor = tag; return l }

// FirstMatch keeps only the first match.
func (l Locator) FirstMatch() Locator { l.First = true; return l }

// String is a stable, human-readable form (also used as the key by the fake page).
func (l Locator) String() string {
	var b strings.Builder
	switch l.Kind {
	case "css":
		fmt.Fprintf(&b, "css(%s)", l.Value)
	case "text":
		fmt.Fprintf(&b, "text(%q exact=%t)", l.Value, l.Exact)
	case "label":
		fmt.Fprintf(&b, "label(%q exact=%t)", l.Value, l.Exact)
	case "role":
		fmt.Fprintf(&b, "role(%s %q)", l.Role, l.Name)
	default:
		fmt.Fprintf(&b, "?%s", l.Kind)
	}
	if l.HasText != "" {
		fmt.Fprintf(&b, " has=%q", l.HasText)
	}
	if l.Visible {
		b.WriteString(" visible")
	}
	if l.Ancestor != "" {
		fmt.Fprintf(&b, " ancestor=%s", l.Ancestor)
	}
	if l.Scope != nil {
		fmt.Fprintf(&b, " in[%s]", l.Scope.String())
	}
	return b.String()
}

func (l Locator) json() string {
	b, _ := json.Marshal(l)
	return string(b)
}
