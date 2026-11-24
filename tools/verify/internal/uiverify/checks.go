package uiverify

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Viewports are the four sizes the Python script ran.
var Viewports = []Viewport{{"phone", 375, 812}, {"tablet", 768, 1024}, {"desktop", 1024, 900}, {"wide", 1440, 1000}}

// Viewport is one browser size.
type Viewport struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// ConsoleError is one browser console error.
type ConsoleError struct {
	Text     string   `json:"text"`
	Location Location `json:"location"`
}

// Location is a console message source position.
type Location struct {
	URL          string `json:"url"`
	LineNumber   int    `json:"lineNumber"`
	ColumnNumber int    `json:"columnNumber"`
}

// Page is the browser surface the checks need. Count and Visible are immediate;
// actions (Click, Fill, Select) and InnerText wait for the element like the
// Playwright calls they replace.
type Page interface {
	Goto(url string) error // waits for network idle
	Screenshot(path string) error
	Count(Locator) (int, error)
	Visible(Locator) (bool, error)
	Click(Locator) error
	Fill(Locator, string) error
	Select(Locator, string) error
	InnerText(Locator) (string, error)
	Style(Locator, string) (string, error)
	Attr(Locator, string) (string, error)
	Dimensions() (scroll, client int, err error)
	ConsoleErrors() []ConsoleError
	Missing() []string
	Close() error
}

// Browser opens one isolated page per viewport (fresh storage, reduced motion).
type Browser interface {
	NewPage(v Viewport) (Page, error)
	Close() error
}

// Check is one recorded assertion.
type Check struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Evidence any    `json:"evidence"`
}

// ViewportResult is the checks of one viewport.
type ViewportResult struct {
	Viewport Viewport `json:"viewport"`
	Checks   []Check  `json:"checks"`
}

// Summary counts check outcomes.
type Summary struct {
	Pass int `json:"pass"`
	Fail int `json:"fail"`
	Skip int `json:"skip"`
}

// Report is ui-results.json (schema_version 1, same shape as the Python script).
type Report struct {
	SchemaVersion int              `json:"schema_version"`
	BaseURL       string           `json:"base_url"`
	Viewports     []ViewportResult `json:"viewports"`
	Summary       Summary          `json:"summary"`
}

type beforeAfter struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

type dims struct {
	Scroll int `json:"scroll"`
	Client int `json:"client"`
}

type reviewCounts struct {
	Items    int `json:"items"`
	Accepted int `json:"accepted"`
}

type exceptionEvidence struct {
	Exception string `json:"exception"`
	Message   string `json:"message"`
}

func navButton(width int, desktop, mobile string) Locator {
	if width > 700 {
		return Locator{Kind: "css", Value: ".sidebar .nav-item", HasText: desktop, First: true}
	}
	if mobile == "" {
		mobile = desktop
	}
	return Locator{Kind: "css", Value: ".mobile-nav button", HasText: mobile, First: true}
}

// RunView runs the sixteen checks of one viewport. Any error aborts the viewport,
// exactly like the Python run_view exception handler: the checks gathered so far
// are discarded and a single FAIL "viewport-run" is recorded.
func RunView(b Browser, base, outDir string, v Viewport) (res ViewportResult) {
	defer func() {
		if r := recover(); r != nil {
			res = failedView(v, fmt.Sprintf("%T", r), fmt.Sprint(r))
		}
	}()
	page, err := b.NewPage(v)
	if err != nil {
		return failedView(v, fmt.Sprintf("%T", err), err.Error())
	}
	defer page.Close()
	checks, err := runChecks(page, base, outDir, v)
	if err != nil {
		return failedView(v, fmt.Sprintf("%T", err), err.Error())
	}
	return ViewportResult{Viewport: v, Checks: checks}
}

func failedView(v Viewport, exc, msg string) ViewportResult {
	return ViewportResult{Viewport: v, Checks: []Check{{ID: "viewport-run", Status: "FAIL", Evidence: exceptionEvidence{exc, msg}}}}
}

func runChecks(page Page, base, outDir string, v Viewport) ([]Check, error) {
	var checks []Check
	check := func(id string, passed bool, evidence any) {
		st := "FAIL"
		if passed {
			st = "PASS"
		}
		checks = append(checks, Check{ID: id, Status: st, Evidence: evidence})
	}
	width := v.Width
	if err := page.Goto(base); err != nil {
		return nil, err
	}
	if err := page.Screenshot(filepath.Join(outDir, v.Name+"-library.png")); err != nil {
		return nil, err
	}
	n, err := page.Count(CSS(`meta[name="viewport"]`))
	if err != nil {
		return nil, err
	}
	check("viewport-meta", n == 1, "one viewport declaration")
	scroll, client, err := page.Dimensions()
	if err != nil {
		return nil, err
	}
	check("no-horizontal-overflow", scroll <= client, dims{scroll, client})
	disclosureText := "Sample data only · no backend connected"
	if width > 700 {
		disclosureText = "No backend connected"
	}
	vis, err := page.Visible(ByText(disclosureText, true).FirstMatch())
	if err != nil {
		return nil, err
	}
	check("sample-data-disclosure", vis, "backend state disclosed and visible: "+disclosureText)

	search := ByLabel("Search library", false)
	if err := page.Fill(search, "Orbital"); err != nil {
		return nil, err
	}
	cards, err := page.Count(CSS(".book-card"))
	if err != nil {
		return nil, err
	}
	orbital, err := page.Count(VisibleButton("Orbital Gardens"))
	if err != nil {
		return nil, err
	}
	check("search-filter", cards == 1 && orbital >= 1, "six books reduced to one")
	if err := page.Fill(search, ""); err != nil {
		return nil, err
	}
	if err := page.Click(VisibleButton("Resume chapter")); err != nil {
		return nil, err
	}
	h1 := CSS(".reading-page h1").FirstMatch()
	before, err := page.InnerText(h1)
	if err != nil {
		return nil, err
	}
	if err := page.Click(VisibleButton("Next")); err != nil {
		return nil, err
	}
	after, err := page.InnerText(h1)
	if err != nil {
		return nil, err
	}
	check("reader-next-delta", before != after && strings.Contains(after, "Chapter 8"), beforeAfter{before, after})
	readingPage := CSS(".reading-page").FirstMatch()
	sizeBefore, err := page.Style(readingPage, "fontSize")
	if err != nil {
		return nil, err
	}
	if err := page.Click(ByLabel("Appearance", true).FirstMatch()); err != nil {
		return nil, err
	}
	if err := page.Click(ByLabel("Decrease text size", false).FirstMatch()); err != nil {
		return nil, err
	}
	sizeAfter, err := page.Style(readingPage, "fontSize")
	if err != nil {
		return nil, err
	}
	check("reader-font-delta", sizeBefore != sizeAfter, beforeAfter{sizeBefore, sizeAfter})
	closePanel := ByLabel("Close panel", false).FirstMatch()
	if err := page.Click(closePanel); err != nil {
		return nil, err
	}
	if err := page.Click(VisibleButton("Save a passage")); err != nil {
		return nil, err
	}
	notes, err := page.Count(CSS(".reader-panel .note"))
	if err != nil {
		return nil, err
	}
	check("save-passage", notes >= 2, "passage added to notes panel")
	if err := page.Click(closePanel); err != nil {
		return nil, err
	}
	if err := page.Click(CSS(".reader-topbar .back-button").FirstMatch()); err != nil {
		return nil, err
	}
	title, err := page.InnerText(CSS(".detail-copy h1").FirstMatch())
	if err != nil {
		return nil, err
	}
	check("book-detail", title == "The Glass Meridian", "reader returns to selected edition")
	if err := page.Click(CSS(".detail-page > .back-button").FirstMatch()); err != nil {
		return nil, err
	}

	if err := page.Click(navButton(width, "Metadata review", "Review")); err != nil {
		return nil, err
	}
	reviewItem := CSS(".review-item")
	countBefore, err := page.Count(reviewItem)
	if err != nil {
		return nil, err
	}
	if err := page.Click(ByRoleButton("Accept proposal").Within(reviewItem.FirstMatch()).FirstMatch()); err != nil {
		return nil, err
	}
	accepted, err := page.Count(CSS(".review-item.accepted"))
	if err != nil {
		return nil, err
	}
	check("metadata-decision-delta", countBefore == 3 && accepted == 1, reviewCounts{countBefore, accepted})

	if err := page.Click(navButton(width, "Sources", "")); err != nil {
		return nil, err
	}
	if err := page.Click(VisibleButton("Add source")); err != nil {
		return nil, err
	}
	addBtn := ByRoleButton("Add to sample workspace").FirstMatch()
	if err := page.Click(addBtn); err != nil {
		return nil, err
	}
	vis, err = page.Visible(ByText("Enter a descriptive source name.", false).FirstMatch())
	if err != nil {
		return nil, err
	}
	check("source-validation", vis, "empty source rejected")
	if err := page.Fill(ByLabel("Source name", false), "Test OPDS"); err != nil {
		return nil, err
	}
	if err := page.Select(ByLabel("Connector", false), "OPDS 1.2"); err != nil {
		return nil, err
	}
	if err := page.Click(addBtn); err != nil {
		return nil, err
	}
	testOPDS := ByText("Test OPDS", true).FirstMatch()
	vis, err = page.Visible(testOPDS)
	if err != nil {
		return nil, err
	}
	check("source-add-delta", vis, "validated source appears")
	if err := page.Click(ByRoleButton("Scan").Within(ByText("Test OPDS", true).ClosestAncestor("article")).FirstMatch()); err != nil {
		return nil, err
	}
	vis, err = page.Visible(ByText("Scan queued in prototype", true).FirstMatch())
	if err != nil {
		return nil, err
	}
	check("source-scan-state", vis, "scan action changes explicit prototype state")

	if err := page.Click(navButton(width, "Synchronization", "")); err != nil {
		return nil, err
	}
	if err := page.Click(VisibleButton("Keep furthest position")); err != nil {
		return nil, err
	}
	vis, err = page.Visible(ByText("Conflict resolved", true).FirstMatch())
	if err != nil {
		return nil, err
	}
	check("sync-conflict-delta", vis, "47% position selected")

	if err := page.Click(navButton(width, "Reading settings", "Settings")); err != nil {
		return nil, err
	}
	if err := page.Click(CSS(".theme-card.sepia").FirstMatch()); err != nil {
		return nil, err
	}
	theme, err := page.Attr(CSS(".app").FirstMatch(), "data-theme")
	if err != nil {
		return nil, err
	}
	check("theme-delta", theme == "sepia", "sepia theme applied")
	if err := page.Screenshot(filepath.Join(outDir, v.Name+"-settings.png")); err != nil {
		return nil, err
	}
	ce := page.ConsoleErrors()
	if ce == nil {
		ce = []ConsoleError{}
	}
	check("console-errors", len(ce) == 0, ce)
	missing := page.Missing()
	if missing == nil {
		missing = []string{}
	}
	check("missing-assets", len(missing) == 0, missing)
	return checks, nil
}

// RunAll runs every viewport and builds the report. A viewport error never
// aborts the others (Python parity).
func RunAll(b Browser, base, outDir string) Report {
	rep := Report{SchemaVersion: 1, BaseURL: base}
	for _, v := range Viewports {
		rep.Viewports = append(rep.Viewports, RunView(b, base, outDir, v))
	}
	for _, vr := range rep.Viewports {
		for _, c := range vr.Checks {
			if c.Status == "PASS" {
				rep.Summary.Pass++
			} else {
				rep.Summary.Fail++
			}
		}
	}
	return rep
}
