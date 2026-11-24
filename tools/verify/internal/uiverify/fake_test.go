package uiverify

import (
	"errors"
	"fmt"
	"strings"
)

// fakeApp is a UNIT-CLASS stand-in for the prototype (constitution 11.4.289(C)):
// it proves the CHECK LOGIC decides correctly (healthy app passes, each seeded
// fault fails exactly the check that guards it). It is never evidence that the
// prototype works; that needs the live container run.
type fakeApp struct {
	faults map[string]bool
	failNP map[string]bool // viewport names whose NewPage fails

	w                            int
	screen, search, panel, theme string
	chapter, fontPx, notes       int
	accepted                     int
	sources                      []string
	modal, scanned, resolved     bool
	sourceErr                    string
	nameInput                    string
	shots                        []string
}

func (a *fakeApp) NewPage(v Viewport) (Page, error) {
	if a.failNP[v.Name] {
		return nil, errors.New("chromium failed to start")
	}
	return &fakeApp{faults: a.faults, w: v.Width, screen: "library", fontPx: 18, notes: 1, theme: "paper"}, nil
}
func (a *fakeApp) Close() error { return nil }

func (a *fakeApp) Goto(string) error { return nil }
func (a *fakeApp) Screenshot(p string) error {
	a.shots = append(a.shots, p)
	return nil
}
func (a *fakeApp) Dimensions() (int, int, error) {
	if a.faults["overflow"] {
		return a.w + 40, a.w, nil
	}
	return a.w, a.w, nil
}
func (a *fakeApp) ConsoleErrors() []ConsoleError {
	if a.faults["console-error"] {
		return []ConsoleError{{Text: "boom"}}
	}
	return nil
}
func (a *fakeApp) Missing() []string {
	if a.faults["missing-asset"] {
		return []string{"404 /assets/x.js"}
	}
	return nil
}

func (a *fakeApp) filtered() int {
	if a.search == "" || a.faults["search-noop"] {
		return 6
	}
	if strings.Contains(strings.ToLower("Orbital Gardens"), strings.ToLower(a.search)) {
		return 1
	}
	return 0
}

func (a *fakeApp) Count(l Locator) (int, error) {
	switch l.String() {
	case `css(meta[name="viewport"])`:
		if a.faults["no-viewport-meta"] {
			return 0, nil
		}
		return 1, nil
	case "css(.book-card)":
		return a.filtered(), nil
	case `css(button) has="Orbital Gardens" visible`:
		return map[bool]int{true: 1, false: 0}[a.filtered() >= 1], nil
	case "css(.reader-panel .note)":
		if a.panel == "notes" {
			return a.notes, nil
		}
		return 0, nil
	case "css(.review-item)":
		return 3, nil
	case "css(.review-item.accepted)":
		return a.accepted, nil
	}
	return 0, fmt.Errorf("fake: unmodelled locator %s", l)
}

func (a *fakeApp) Visible(l Locator) (bool, error) {
	disclosure := "No backend connected"
	if a.w <= 700 {
		disclosure = "Sample data only · no backend connected"
	}
	if a.faults["wrong-disclosure"] {
		disclosure = "Connected"
	}
	switch l.String() {
	case fmt.Sprintf("text(%q exact=true)", disclosure):
		return true, nil
	case `text("No backend connected" exact=true)`, `text("Sample data only · no backend connected" exact=true)`:
		return false, nil
	case `text("Enter a descriptive source name." exact=false)`:
		return a.sourceErr != "", nil
	case `text("Test OPDS" exact=true)`:
		for _, s := range a.sources {
			if s == "Test OPDS" {
				return true, nil
			}
		}
		return false, nil
	case `text("Scan queued in prototype" exact=true)`:
		return a.scanned, nil
	case `text("Conflict resolved" exact=true)`:
		return a.resolved, nil
	}
	return false, fmt.Errorf("fake: unmodelled locator %s", l)
}

func (a *fakeApp) Click(l Locator) error {
	switch s := l.String(); s {
	case `css(button) has="Resume chapter" visible`:
		a.screen, a.chapter = "reader", 7
	case `css(button) has="Next" visible`:
		if !a.faults["next-noop"] {
			a.chapter++
		}
	case `label("Appearance" exact=true)`:
		a.panel = "appearance"
	case `label("Decrease text size" exact=false)`:
		if a.panel != "appearance" {
			return errors.New("timeout waiting for Decrease text size")
		}
		if !a.faults["font-noop"] {
			a.fontPx--
		}
	case `label("Close panel" exact=false)`:
		a.panel = ""
	case `css(button) has="Save a passage" visible`:
		if !a.faults["save-noop"] {
			a.notes++
		}
		a.panel = "notes"
	case "css(.reader-topbar .back-button)":
		a.screen = "detail"
	case "css(.detail-page > .back-button)":
		a.screen = "library"
	case `css(.sidebar .nav-item) has="Metadata review"`, `css(.sidebar .nav-item) has="Sources"`,
		`css(.sidebar .nav-item) has="Synchronization"`, `css(.sidebar .nav-item) has="Reading settings"`:
		if a.w <= 700 {
			return errors.New("sidebar is not visible at this width")
		}
		a.screen = l.HasText
	case `css(.mobile-nav button) has="Review"`, `css(.mobile-nav button) has="Sources"`,
		`css(.mobile-nav button) has="Synchronization"`, `css(.mobile-nav button) has="Settings"`:
		if a.w > 700 {
			return errors.New("mobile nav is not visible at this width")
		}
		a.screen = l.HasText
	case `role(button "Accept proposal") in[css(.review-item)]`:
		if !a.faults["accept-noop"] {
			a.accepted = 1
		}
	case `css(button) has="Add source" visible`:
		a.modal = true
	case `role(button "Add to sample workspace")`:
		if strings.TrimSpace(a.nameInput) == "" && !a.faults["no-validation"] {
			a.sourceErr = "Enter a descriptive source name."
			return nil
		}
		a.sourceErr = ""
		if !a.faults["add-source-noop"] {
			a.sources = append(a.sources, a.nameInput)
		}
	case `role(button "Scan") in[text("Test OPDS" exact=true) ancestor=article]`:
		if !a.faults["scan-noop"] {
			a.scanned = true
		}
	case `css(button) has="Keep furthest position" visible`:
		if !a.faults["sync-noop"] {
			a.resolved = true
		}
	case "css(.theme-card.sepia)":
		if !a.faults["theme-noop"] {
			a.theme = "sepia"
		}
	default:
		return fmt.Errorf("fake: unmodelled click %s", s)
	}
	return nil
}

func (a *fakeApp) Fill(l Locator, v string) error {
	switch l.String() {
	case `label("Search library" exact=false)`:
		a.search = v
	case `label("Source name" exact=false)`:
		a.nameInput = v
	default:
		return fmt.Errorf("fake: unmodelled fill %s", l)
	}
	return nil
}

func (a *fakeApp) Select(l Locator, label string) error {
	if l.String() != `label("Connector" exact=false)` || label != "OPDS 1.2" {
		return fmt.Errorf("fake: unmodelled select %s %q", l, label)
	}
	return nil
}

func (a *fakeApp) InnerText(l Locator) (string, error) {
	switch l.String() {
	case "css(.reading-page h1)":
		return fmt.Sprintf("Chapter %d", a.chapter), nil
	case "css(.detail-copy h1)":
		if a.faults["wrong-detail"] {
			return "Orbital Gardens", nil
		}
		return "The Glass Meridian", nil
	}
	return "", fmt.Errorf("fake: unmodelled text %s", l)
}

func (a *fakeApp) Style(l Locator, prop string) (string, error) {
	if l.String() == "css(.reading-page)" && prop == "fontSize" {
		return fmt.Sprintf("%dpx", a.fontPx), nil
	}
	return "", fmt.Errorf("fake: unmodelled style %s", l)
}

func (a *fakeApp) Attr(l Locator, name string) (string, error) {
	if l.String() == "css(.app)" && name == "data-theme" {
		return a.theme, nil
	}
	return "", fmt.Errorf("fake: unmodelled attr %s", l)
}
