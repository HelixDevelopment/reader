package httpapi

// rclone load run (HRD-027 "WebDAV/rclone load runs").
//
// The tree has rclone support: server/internal/sources/rclone.go (NewRclone —
// List via `rclone lsjson`, Open/OpenRange via `rclone cat`), server/Dockerfile
// (rclone is installed in the server image), and source.go's DefaultRegistry
// (sftp, ftp, smb, s3, nextcloud, ... → NewRclone).
//
// In this environment `rclone listremotes` is empty — there is no pre-existing
// rclone target — so the run uses a temporary rclone config with a `local`
// backend remote ("wfdav") whose remote path is the fixture directory. The
// ladder runs through the route exactly like the WebDAV run:
// client → route → sources.NewRclone → `rclone cat <remote:path>`, with a
// counting wrapper around the connector so "requests per file" is the number
// of rclone invocations.
//
// Gated on HELIX_LOAD_RUN=1 like the rest of the load suite.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/sources"
)

const rcloneSourceID = "load-remote-rclone"

// countingConnector counts connector-level opens, which for the rclone
// connector are `rclone cat` invocations — the measure of "requests per file".
type countingConnector struct {
	inner sources.Connector
	opens atomic.Int64
}

func (c *countingConnector) List(ctx context.Context, prefix string) ([]sources.Entry, error) {
	return c.inner.List(ctx, prefix)
}
func (c *countingConnector) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	c.opens.Add(1)
	return c.inner.Open(ctx, name)
}
func (c *countingConnector) Capabilities() sources.Capabilities { return c.inner.Capabilities() }
func (c *countingConnector) Close() error                       { return c.inner.Close() }

type rcloneFile struct {
	id, name string
	size     int64
	sha256   string
}

type rcloneRow struct {
	deliveryStats
	Opens int64 `json:"opens"`
}

func TestLiveLoadRemoteSourceRclone(t *testing.T) {
	if os.Getenv("HELIX_LOAD_RUN") != "1" {
		t.Skip("HELIX_LOAD_RUN is not 1; measurement harness opted out")
	}
	setRemoteLoadOutput(t, "evidence/campaign-2026-11-23/remote/remote_rclone_ladder.jsonl")
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone is not installed: no rclone target available in this environment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Fixture: random bytes under the rclone remote root.
	root := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	repo := catalog.NewMemory()
	files := []rcloneFile{}
	for index, size := range remoteSizes(t) {
		name := fmt.Sprintf("books/rc%d.bin", index)
		abs := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		body := make([]byte, size)
		if _, err := rand.Read(body); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		digest := hex.EncodeToString(sum[:])
		if err := os.WriteFile(abs, body, 0o644); err != nil {
			t.Fatal(err)
		}
		id := fmt.Sprintf("load-rclone-f%d", index)
		work := catalog.Work{
			ID: id + "-w", CanonicalTitle: "Rclone load " + id, CreatedAt: now, UpdatedAt: now,
			Editions: []catalog.Edition{{
				ID: id + "-e", Title: "Rclone load " + id, State: catalog.Identified, UpdatedAt: now,
				Files: []catalog.File{{
					ID: id, SourceID: rcloneSourceID, Path: name, Format: "bin",
					MIMEType: "application/octet-stream", Size: size, SHA256: digest, ModifiedAt: now,
				}},
			}},
		}
		if err := repo.UpsertWork(ctx, work); err != nil {
			t.Fatal(err)
		}
		files = append(files, rcloneFile{id: id, name: name, size: size, sha256: digest})
	}

	// rclone config: the `local` backend remote "wfdav" whose remote path is
	// the fixture root (Endpoint "wfdav:"+root; the target the connector builds
	// per Open is "wfdav:<root>/<name>").
	cfgPath := filepath.Join(t.TempDir(), "rclone.conf")
	if err := os.WriteFile(cfgPath, []byte("[wfdav]\ntype = local\nnolink = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewRclone(sources.Config{
		Protocol: "sftp", Endpoint: "wfdav:" + root, RclonePath: "rclone", RcloneConfig: cfgPath, ID: rcloneSourceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	counting := &countingConnector{inner: connector}

	// List: `rclone lsjson --recursive --files-only wfdav:<root>` — every
	// fixture file must be discoverable with its size.
	entries, err := counting.List(ctx, "")
	if err != nil {
		t.Fatalf("rclone lsjson: %v", err)
	}
	byName := map[string]int64{}
	for _, e := range entries {
		byName[e.Path] = e.Size
	}
	for _, f := range files {
		got, ok := byName[f.name]
		if !ok || got != f.size {
			t.Fatalf("rclone list missed %s (size %d): entries %v", f.name, f.size, byName)
		}
	}

	srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	srv.connectors[rcloneSourceID] = counting
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 256, MaxConnsPerHost: 0, DisableCompression: true}}

	// Ladder, same shape as TestLiveLoadRemoteSource / the WebDAV run.
	levels := deliveryLevels(t)
	timeout := time.Duration(deliveryEnvInt(t, "HELIX_DELIVERY_TIMEOUT_SECONDS", 60)) * time.Second
	rows := []rcloneRow{}
	for _, f := range files {
		url := ts.URL + "/v1/files/" + f.id + "/content"
		jobs := []deliveryJob{
			{kind: "full", url: url, wantStatus: http.StatusOK, wantBytes: f.size, wantSHA: f.sha256},
		}
		for _, clients := range levels {
			opensBefore := counting.opens.Load()
			stats := runDeliveryLadder(ctx, client, jobs, clients, remoteLoadRounds(clients), timeout)
			stats.SizeBytes = f.size
			opens := counting.opens.Load() - opensBefore
			rows = append(rows, rcloneRow{deliveryStats: stats, Opens: opens})
			if stats.Completed != stats.Requested {
				t.Fatalf("rclone size %d level %d: %d of %d did not complete (%v)", f.size, clients, stats.Requested-stats.Completed, stats.Requested, stats.Errors)
			}
			for class, count := range stats.Mismatches {
				t.Fatalf("rclone size %d level %d: %d %s responses failed their integrity needle (%v)", f.size, clients, count, class, stats.Mismatches)
			}
			// No retry storm: one rclone invocation per request, 1:1.
			if opens != int64(stats.Requested) {
				t.Fatalf("rclone size %d level %d: %d client requests caused %d rclone opens; the connector is not one-for-one", f.size, clients, stats.Requested, opens)
			}
			t.Logf("RCLONE LADDER size=%d clients=%d completed=%d req_per_sec=%.1f mib_per_sec=%.1f p50_ms=%.2f p95_ms=%.2f peak_inflight=%d opens=%d",
				f.size, clients, stats.Completed, stats.ReqPerSec, stats.BytesPerSec/(1<<20), stats.P50MS, stats.P95MS, stats.PeakInFlight, opens)
		}
	}

	httpEmitLoad(t, map[string]any{
		"test":             "remote_rclone_ladder",
		"connector":        "rclone",
		"remote":           "wfdav:" + root,
		"remote_kind":      "local backend (rclone listremotes was empty: no pre-existing target)",
		"measurement_path": "in_process_loopback",
		"note":             "rclone target stood up by the run: temp rclone config, local backend remote \"wfdav\"; `rclone cat <remote:path>` per Open, `rclone lsjson --recursive --files-only` for List",
		"opens_total":      counting.opens.Load(),
		"levels":           levels,
		"results":          rows,
	})
}
