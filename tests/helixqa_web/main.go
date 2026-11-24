// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"digital.vasic.challenges/pkg/userflow"
	"digital.vasic.helixqa/pkg/autonomous"
	"digital.vasic.helixqa/pkg/testbank"
)

func main() {
	bank := flag.String("bank", "", "bank YAML")
	cdp := flag.String("cdp", "", "CDP endpoint (http://host:port)")
	out := flag.String("out", "", "output dir (report.json + PNGs)")
	retry := flag.Duration("assert-retry", 6*time.Second, "retry window for assertVisible/assertNotVisible")
	platform := flag.String("platform", "web", "run only cases whose platforms are empty or include this one")
	vp := flag.String("expect-viewport", "1920x937", "viewport every step must report (WxH). Upstream Initialize asks for 1920x1080 but applies it to a NEW context the later steps never use (they use contexts()[0].pages()[0]); the real viewport is the headless window 1920x1080 minus 143px of browser chrome = 1920x937, pinned here so drift is caught")
	unpinned := flag.Bool("allow-unpinned", false, "do not require expected_cases/expected_steps in the bank metadata")
	flag.Parse()
	if *bank == "" || *cdp == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: webrunner -bank f.yaml -cdp http://127.0.0.1:9222 -out dir")
		os.Exit(2)
	}
	bf, err := testbank.LoadFile(*bank)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	ex := autonomous.NewPlaywrightExecutor(*cdp)
	ad := userflow.NewPlaywrightCLIAdapter(*cdp)
	save := func(name string, png []byte) (string, string, error) {
		sum := sha256.Sum256(png)
		want := hex.EncodeToString(sum[:])
		full := filepath.Join(*out, name)
		if err := os.WriteFile(full, png, 0o644); err != nil {
			return "", "", err
		}
		back, err := os.ReadFile(full) // the evidence file must exist and match what was captured
		if err != nil {
			return "", "", err
		}
		got := sha256.Sum256(back)
		if hex.EncodeToString(got[:]) != want {
			return "", "", fmt.Errorf("read-back sha mismatch for %s", name)
		}
		return name, want, nil
	}
	state := func(ctx context.Context) (PageState, error) {
		raw, err := ad.EvaluateJS(ctx, `({u: location.href, w: innerWidth, h: innerHeight, t: document.body ? document.body.innerText : ""})`)
		if err != nil {
			return PageState{}, err
		}
		var v struct {
			U string `json:"u"`
			W int    `json:"w"`
			H int    `json:"h"`
			T string `json:"t"`
		}
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return PageState{}, fmt.Errorf("page state %q: %w", raw, err)
		}
		sum := sha256.Sum256([]byte(v.T))
		return PageState{URL: v.U, Width: v.W, Height: v.H, TextSHA: hex.EncodeToString(sum[:])}, nil
	}
	var ev [2]int
	if _, err := fmt.Sscanf(*vp, "%dx%d", &ev[0], &ev[1]); err != nil {
		fmt.Fprintln(os.Stderr, "bad -expect-viewport:", err)
		os.Exit(2)
	}
	rep := RunBank(context.Background(), bf, ex, Options{Retry: *retry, Platform: *platform, Shoot: ad.Screenshot, Save: save, State: state, ExpectViewport: ev, AllowUnpinned: *unpinned})
	b, _ := json.MarshalIndent(rep, "", "  ")
	_ = os.WriteFile(filepath.Join(*out, "report.json"), b, 0o644)
	for _, c := range rep.Cases {
		fmt.Printf("%-5s %s %s\n", c.Verdict, c.ID, c.Name)
		for _, s := range c.Steps {
			fmt.Printf("      %-4s %s :: %s (attempts=%d)\n", s.Verdict, s.Name, s.Message, s.Attempts)
		}
	}
	fmt.Printf("\nweb bank summary: %d PASS, %d FAIL, %d SKIP steps\n", rep.Pass, rep.Fail, rep.Skip)
	for _, e := range rep.PinErrors {
		fmt.Println("PIN-ERROR:", e)
	}
	if !rep.OK() {
		os.Exit(1)
	}
}
