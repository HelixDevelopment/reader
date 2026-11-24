package liveverify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func failedName(res Results) string {
	for _, c := range res.Checks {
		if c.Status == "FAIL" {
			return c.Name
		}
	}
	return ""
}

// TestLiveServerPassesAgainstRealInProcessServer is the golden-TRUE case: the
// unmodified server must give a full PASS with the exact assertion inventory.
func TestLiveServerPassesAgainstRealInProcessServer(t *testing.T) {
	l := newInproc(t, nil)
	cfg := baseConfig(t, l)
	res := RunLiveServer(cfg)
	if res.Status != "PASS" || res.Fail != 0 {
		t.Fatalf("expected PASS, got %s fail=%d (%s) checks=%v", res.Status, res.Fail, failedName(res), res.Checks)
	}
	if res.ExitCode() != 0 {
		t.Fatalf("exit code %d", res.ExitCode())
	}
	if res.Pass != len(liveServerCheckNames) {
		t.Fatalf("pass count %d, want %d (assertion inventory drifted)", res.Pass, len(liveServerCheckNames))
	}
	for i, c := range res.Checks {
		want := liveServerCheckNames[i]
		if want != "" && !strings.HasPrefix(c.Name, want) {
			t.Fatalf("check %d = %q, want prefix %q", i, c.Name, want)
		}
	}
	raw, err := os.ReadFile(filepath.Join(cfg.EvidenceDir, "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var back Results
	if err := json.Unmarshal(raw, &back); err != nil || back.Pass != res.Pass || back.Status != "PASS" {
		t.Fatalf("results.json mismatch: %v %s", err, raw)
	}
	for _, f := range []string{"pse-client.json", "pse-client-after-restart.json"} {
		if _, err := os.Stat(filepath.Join(cfg.EvidenceDir, f)); err != nil {
			t.Fatalf("evidence file %s: %v", f, err)
		}
	}
}

// liveServerCheckNames is the behaviour-parity inventory of verify_live_server.py:
// the ordered assertion names (prefix match; "" = dynamic name).
var liveServerCheckNames = []string{
	"health endpoint", "admin authentication boundary", "catalog authentication boundary",
	"live local source creation", "live scan accepted", "live in-place ingestion", "live catalog query",
	"file identity and source association", "content identity and size",
	"OPDS 1.2 acquisition feed", "OPDS 2.0 publication feed", "OpenSearch description",
	"single-range content download", "HEAD content metadata without body", "stale If-Range returns complete representation",
	"multi-range request uses complete-response fallback", "attachment remains identity encoded", "gzip content negotiation",
	"live page-stream source creation", "page-stream scan accepted", "page-stream fixtures ingested",
	"every page-stream fixture is catalogued by digest",
	"a CBZ edition advertises exactly one page stream", "pse:count equals the pages in the archive",
	"the advertised stream type comes from the page bytes", "the stream href names this file and the page template",
	"a stream the server cannot position carries no pse:lastRead",
	"live-server-mixed.cbz is not advertised as pageable", "live-server-mixed.cbz still ships whole", "live-server-mixed.cbz refuses page requests",
	"live-server-forge.cbz is not advertised as pageable", "live-server-forge.cbz still ships whole", "live-server-forge.cbz refuses page requests",
	"live-server-cbr-stub.cbr is not advertised as pageable", "live-server-cbr-stub.cbr still ships whole", "live-server-cbr-stub.cbr refuses page requests",
	"live-server-plain.txt is not advertised as pageable", "live-server-plain.txt still ships whole", "live-server-plain.txt refuses page requests",
	"page 0 is served byte-identical", "page 0 is tagged with the catalogued file digest",
	"page 1 is served byte-identical", "page 1 is tagged with the catalogued file digest",
	"page 2 is served byte-identical", "page 2 is tagged with the catalogued file digest",
	"page past the last returns the valid range", "a non-integer page number is rejected", "an unknown file has no page stream",
	"the acquisition feed carries the stream link", "OPDS 2.0 invents no page-stream extension",
	"OPDS-PSE client round trip", "the PSE client fetched every advertised page",
	"KOSync registration or deterministic rerun", "KOSync authentication", "KOSync progress write", "KOSync progress read",
	"pgvector memory write", "pgvector scoped search", "internal progress write",
	"PostgreSQL catalog survives server restart", "completed scan job survives server restart", "restored scan job appears in the scan list",
	"Redis KOSync account survives server restart", "Redis KOSync progress survives server restart", "Redis internal progress survives server restart",
	"stable source identity reattaches after restart", "page-stream source reattaches after restart",
	"OPDS-PSE client round trip after restart", "existing file identity serves after source re-registration",
	"post-restart source rescan accepted", "post-restart reconciliation preserves present file", "stable source rescan does not duplicate files",
	"post-test catalog cleanup", "post-test page-stream catalog cleanup failed for", "post-test memory cleanup", "post-test progress cleanup",
}
