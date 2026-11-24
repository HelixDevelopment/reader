package uiverify

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

//go:embed hv.js
var hvJS string

// ErrNotIsolated is returned when a live browser launch is attempted outside
// the rootless isolation container (HARD RULE: tests never touch the
// operator's desktop, display, browser or apps; docs/test-isolation.md).
var ErrNotIsolated = errors.New("refusing to launch a browser outside the rootless isolation container (tools/isolation/run_container.sh)")

// IsolationProbe names the facts RequireContainer reads (injectable for tests).
type IsolationProbe struct {
	ContainerEnv string              // podman's /run/.containerenv
	LaunchMarker string              // /opt/isoinside/launch.sh: bound read-only by tools/isolation/run_container.sh only
	NetDir       string              // /sys/class/net
	Getenv       func(string) string // environment lookup
}

// DefaultIsolationProbe reads the real container.
func DefaultIsolationProbe() IsolationProbe {
	return IsolationProbe{ContainerEnv: "/run/.containerenv", LaunchMarker: "/opt/isoinside/launch.sh", NetDir: "/sys/class/net", Getenv: os.Getenv}
}

// RequireContainer returns nil only inside the container that
// tools/isolation/run_container.sh starts: podman's /run/.containerenv exists
// (not enough alone: rootful podman, toolbox and distrobox have it too), the
// read-only launch.sh marker that only run_container.sh binds exists, launch.sh
// exported HRDISO_ISOLATED=1 after its own host-display and socket assertions,
// no network interface other than loopback exists (--network none) and no host
// display variable is set. There is deliberately no override.
func RequireContainer(p IsolationProbe) error {
	for _, f := range []string{p.ContainerEnv, p.LaunchMarker} {
		if _, err := os.Stat(f); err != nil {
			return ErrNotIsolated
		}
	}
	if p.Getenv("HRDISO_ISOLATED") != "1" {
		return ErrNotIsolated
	}
	if d := p.Getenv("DISPLAY"); d != "" && d != ":99" {
		return ErrNotIsolated
	}
	if p.Getenv("WAYLAND_DISPLAY") != "" {
		return ErrNotIsolated
	}
	ents, err := os.ReadDir(p.NetDir)
	if err != nil {
		return ErrNotIsolated
	}
	for _, e := range ents {
		if e.Name() != "lo" {
			return ErrNotIsolated
		}
	}
	return nil
}

// ChromeBrowser launches one headless Chromium per page (so each viewport has
// fresh storage, like a new Playwright context).
type ChromeBrowser struct {
	ExecPath string
	Timeout  time.Duration // action/wait budget (default 15s)
}

// Close is a no-op: every page owns its browser process.
func (b *ChromeBrowser) Close() error { return nil }

type chromePage struct {
	ctx     context.Context
	cancel  []context.CancelFunc
	timeout time.Duration
	mu      sync.Mutex
	console []ConsoleError
	missing []string
	idle    chan string // loader ids that reached networkIdle
}

// NewPage opens a headless Chromium sized to v with prefers-reduced-motion: reduce.
func (b *ChromeBrowser) NewPage(v Viewport) (Page, error) {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts, chromedp.ExecPath(b.ExecPath), chromedp.NoSandbox, chromedp.Flag("disable-dev-shm-usage", true))
	actx, acancel := chromedp.NewExecAllocator(context.Background(), opts...)
	cctx, ccancel := chromedp.NewContext(actx)
	p := &chromePage{ctx: cctx, cancel: []context.CancelFunc{ccancel, acancel}, timeout: b.Timeout, idle: make(chan string, 64)}
	if p.timeout == 0 {
		p.timeout = 15 * time.Second
	}
	chromedp.ListenTarget(cctx, p.onEvent)
	// The first chromedp.Run on a context starts the browser and ties its life to THAT context, so it must be the
	// bare page context; a watchdog bounds a hung launch instead of a per-call deadline.
	started := make(chan struct{})
	wd := time.AfterFunc(actionBudget(p.timeout), func() {
		select {
		case <-started:
		default:
			ccancel()
		}
	})
	err := chromedp.Run(cctx)
	close(started)
	wd.Stop()
	if err == nil {
		err = p.run(
			network.Enable(), log.Enable(), page.SetLifecycleEventsEnabled(true),
			emulation.SetDeviceMetricsOverride(int64(v.Width), int64(v.Height), 1, false),
			emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{{Name: "prefers-reduced-motion", Value: "reduce"}}),
		)
	}
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("launch %s: %w", b.ExecPath, err)
	}
	return p, nil
}

func (p *chromePage) onEvent(ev any) {
	switch e := ev.(type) {
	case *cdpruntime.EventConsoleAPICalled:
		if e.Type != cdpruntime.APITypeError {
			return
		}
		var parts []string
		for _, a := range e.Args {
			switch {
			case a.Value != nil && len(a.Value) > 0:
				var s string
				if json.Unmarshal(a.Value, &s) == nil {
					parts = append(parts, s)
				} else {
					parts = append(parts, string(a.Value))
				}
			default:
				parts = append(parts, a.Description)
			}
		}
		loc := Location{}
		if e.StackTrace != nil && len(e.StackTrace.CallFrames) > 0 {
			f := e.StackTrace.CallFrames[0]
			loc = Location{URL: f.URL, LineNumber: int(f.LineNumber), ColumnNumber: int(f.ColumnNumber)}
		}
		p.mu.Lock()
		p.console = append(p.console, ConsoleError{Text: strings.Join(parts, " "), Location: loc})
		p.mu.Unlock()
	case *log.EventEntryAdded:
		// failed resource loads and CSP violations surface as browser-logged errors. UNCONFIRMED whether Playwright's console events also include them (PARITY.md lists this as an intentional difference)
		if e.Entry.Level == log.LevelError {
			p.mu.Lock()
			p.console = append(p.console, ConsoleError{Text: e.Entry.Text, Location: Location{URL: e.Entry.URL, LineNumber: int(e.Entry.LineNumber)}})
			p.mu.Unlock()
		}
	case *network.EventResponseReceived:
		if e.Response.Status == 404 {
			p.mu.Lock()
			p.missing = append(p.missing, fmt.Sprintf("404 %s", e.Response.URL))
			p.mu.Unlock()
		}
	case *page.EventLifecycleEvent:
		if e.Name == "networkIdle" {
			select {
			case p.idle <- string(e.LoaderID):
			default:
			}
		}
	}
}

// actionBudget bounds one browser call, so a hung renderer cannot stall the run
// until the container timeout.
func actionBudget(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 30 * time.Second
	}
	return 2 * timeout
}

func (p *chromePage) run(actions ...chromedp.Action) error {
	ctx, cancel := context.WithTimeout(p.ctx, actionBudget(p.timeout))
	defer cancel()
	return chromedp.Run(ctx, actions...)
}

func (p *chromePage) Close() error {
	for _, c := range p.cancel {
		c()
	}
	return nil
}

func (p *chromePage) Goto(url string) error {
	for len(p.idle) > 0 {
		<-p.idle
	}
	var loader string
	if err := p.run(chromedp.ActionFunc(func(ctx context.Context) error {
		_, lid, errText, _, err := page.Navigate(url).Do(ctx)
		if err == nil && errText != "" {
			err = errors.New(errText)
		}
		loader = string(lid)
		return err
	})); err != nil {
		return fmt.Errorf("goto %s: %w", url, err)
	}
	deadline := time.After(30 * time.Second)
	for {
		select {
		case id := <-p.idle:
			if id == loader || loader == "" {
				return p.run(chromedp.Evaluate(hvJS, nil))
			}
		case <-deadline:
			return fmt.Errorf("goto %s: network did not go idle within 30s", url)
		case <-p.ctx.Done():
			return p.ctx.Err()
		}
	}
}

func (p *chromePage) Screenshot(path string) error {
	var buf []byte
	if err := p.run(chromedp.FullScreenshot(&buf, 100)); err != nil {
		return err
	}
	return os.WriteFile(path, buf, 0o644)
}

func (p *chromePage) call(out any, fn string, args ...any) error {
	parts := make([]string, len(args))
	for i, a := range args {
		if l, ok := a.(Locator); ok {
			parts[i] = l.json()
			continue
		}
		b, _ := json.Marshal(a)
		parts[i] = string(b)
	}
	expr := fmt.Sprintf("window.__hv.%s(%s)", fn, strings.Join(parts, ","))
	return p.run(chromedp.Evaluate(expr, out))
}

func (p *chromePage) Count(l Locator) (n int, err error) { err = p.call(&n, "count", l); return }
func (p *chromePage) Visible(l Locator) (v bool, err error) {
	err = p.call(&v, "visible", l)
	return
}

func (p *chromePage) Dimensions() (scroll, client int, err error) {
	var d dims
	err = p.call(&d, "dims")
	return d.Scroll, d.Client, err
}

// waitFor polls fn until it reports found or the action budget elapses (Playwright
// auto-waits for the element and raises on timeout).
func (p *chromePage) waitFor(l Locator, fn func() (found bool, err error)) error {
	deadline := time.Now().Add(p.timeout)
	for {
		found, err := fn()
		if err != nil {
			return err
		}
		if found {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %s", l)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (p *chromePage) Style(l Locator, prop string) (out string, err error) {
	err = p.waitFor(l, func() (bool, error) {
		var s *string
		if err := p.call(&s, "style", l, prop); err != nil {
			return false, err
		}
		if s == nil {
			return false, nil
		}
		out = *s
		return true, nil
	})
	return
}

func (p *chromePage) Attr(l Locator, name string) (out string, err error) {
	err = p.waitFor(l, func() (bool, error) {
		var r *struct {
			V *string `json:"v"`
		}
		if err := p.call(&r, "attr", l, name); err != nil {
			return false, err
		}
		if r == nil {
			return false, nil
		}
		if r.V != nil {
			out = *r.V
		}
		return true, nil
	})
	return
}

// InnerText waits for the element to exist (Playwright auto-wait), then reads its innerText.
func (p *chromePage) InnerText(l Locator) (string, error) {
	deadline := time.Now().Add(p.timeout)
	for {
		var s *string
		if err := p.call(&s, "text", l); err != nil {
			return "", err
		}
		if s != nil {
			return *s, nil
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for %s", l)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

type prepared struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason"`
	Key    string `json:"key"`
}

// waitActionable polls until the first match is visible, enabled, stable (same
// rectangle twice in a row) and, for clicks, would receive the click.
func (p *chromePage) waitActionable(l Locator, forClick bool) error {
	deadline := time.Now().Add(p.timeout)
	lastKey := ""
	var last prepared
	for {
		if err := p.call(&last, "prepare", l, forClick); err != nil {
			return err
		}
		if last.OK && last.Key == lastKey {
			return nil
		}
		if last.OK {
			lastKey = last.Key
		} else {
			lastKey = ""
		}
		if time.Now().After(deadline) {
			reason := last.Reason
			if reason == "" {
				reason = "element did not settle"
			}
			return fmt.Errorf("timeout waiting for %s: %s", l, reason)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (p *chromePage) Click(l Locator) error {
	if err := p.waitActionable(l, true); err != nil {
		return err
	}
	err := p.run(chromedp.Click(`[data-hv-target="1"]`, chromedp.ByQuery))
	var ignore any
	_ = p.call(&ignore, "clear")
	return err
}

func (p *chromePage) Fill(l Locator, v string) error {
	if err := p.waitActionable(l, false); err != nil {
		return err
	}
	var ok bool
	if err := p.call(&ok, "fill", l, v); err != nil {
		return err
	}
	return nil
}

func (p *chromePage) Select(l Locator, label string) error {
	if err := p.waitActionable(l, false); err != nil {
		return err
	}
	var ok bool
	if err := p.call(&ok, "select", l, label); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no option %q in %s", label, l)
	}
	return nil
}

func (p *chromePage) ConsoleErrors() []ConsoleError {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]ConsoleError(nil), p.console...)
}

func (p *chromePage) Missing() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.missing...)
}
