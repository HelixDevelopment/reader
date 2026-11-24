package liveverify

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/helixreader/helix/server/internal/aimemory"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/httpapi"
	"github.com/helixreader/helix/server/internal/identify"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/kosync"
	"github.com/helixreader/helix/server/internal/sources"
	"github.com/helixreader/helix/server/internal/syncstore"
	"github.com/redis/go-redis/v9"
)

// These tests are GATE-LOGIC tests (constitution 11.4.289(C)(2)): they prove the
// verifier decides correctly against a real in-process helix server (the real
// httpapi router, catalog, KOSync, scan journal) and against deliberately broken
// variants of it. They are not evidence that a deployed server works; that is the
// live run against the real binary.

var (
	pseBinOnce sync.Once
	pseBin     string
	pseBinErr  error
	pseBinDir  string
)

func pseCheckBinary(t *testing.T) string {
	t.Helper()
	pseBinOnce.Do(func() {
		pseBinDir, pseBinErr = os.MkdirTemp("", "liveverify-pse-")
		if pseBinErr != nil {
			return
		}
		pseBin = filepath.Join(pseBinDir, "helix-pse-check")
		out, err := exec.Command("go", "build", "-p", "2", "-o", pseBin, "github.com/helixreader/helix/server/cmd/helix-pse-check").CombinedOutput()
		if err != nil {
			pseBinErr = fmt.Errorf("build helix-pse-check: %v: %s", err, out)
		}
	})
	if pseBinErr != nil {
		t.Fatal(pseBinErr)
	}
	return pseBin
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.ReleaseMode)
	code := m.Run()
	if pseBinDir != "" {
		os.RemoveAll(pseBinDir)
	}
	os.Exit(code)
}

// memStore is a test-only aimemory.Store (the real one needs PostgreSQL/pgvector).
type memStore struct {
	mu   sync.Mutex
	recs map[string]aimemory.Record
}

func (m *memStore) Put(_ context.Context, r aimemory.Record, _ []float32) (aimemory.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, err := aimemory.Prepare(r, 3)
	if err != nil {
		return r, err
	}
	m.recs[r.Scope+"/"+r.ID] = r
	return r, nil
}
func (m *memStore) Search(_ context.Context, scope string, _ []float32, _ int) ([]aimemory.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []aimemory.Result
	for _, r := range m.recs {
		if r.Scope == scope {
			out = append(out, aimemory.Result{Record: r, Similarity: 1})
		}
	}
	return out, nil
}
func (m *memStore) Delete(_ context.Context, scope, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.recs, scope+"/"+id)
	return nil
}
func (m *memStore) Close() error { return nil }

// inproc is a Launcher over the real httpapi.Server. State (catalog, Redis,
// AI memory) survives restarts exactly like the external stores of the real deployment.
type inproc struct {
	t      *testing.T
	repo   catalog.Repository
	redis  *miniredis.Miniredis
	memory *memStore
	wrap   func(http.Handler) http.Handler
	// onRestart runs between Stop and the next Start (broken-persistence mutants).
	onRestart func(*inproc)
	starts    int
	covers    string
	creds     [3]string // admin token, OPDS user, OPDS password (default: the live-server verifier's)
}

func newInproc(t *testing.T, wrap func(http.Handler) http.Handler) *inproc {
	return &inproc{t: t, repo: catalog.NewMemory(), redis: miniredis.RunT(t), memory: &memStore{recs: map[string]aimemory.Record{}}, wrap: wrap, covers: t.TempDir()}
}

type inprocProc struct {
	srv     *httptest.Server
	api     *httpapi.Server
	clients []*redis.Client
}

func (p *inprocProc) Stop() error {
	p.srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := p.api.Shutdown(ctx)
	for _, c := range p.clients {
		c.Close()
	}
	return err
}

func (l *inproc) Start() (string, Process, error) {
	l.starts++
	if l.starts > 1 && l.onRestart != nil {
		l.onRestart(l)
	}
	creds := l.creds
	if creds[0] == "" {
		creds = [3]string{"live-admin-token", "live-reader", "live-password"}
	}
	cfg := config.Config{Address: "127.0.0.1:0", AdminToken: creds[0], OPDSUsername: creds[1], OPDSPassword: creds[2],
		KOSyncRegistration: true, BrotliLevel: 4, CoverCacheDir: l.covers, ShutdownTimeout: 5 * time.Second, EmbeddingDimensions: 3}
	api := httpapi.New(cfg, l.repo, sources.DefaultRegistry(), &ingest.Service{Repository: l.repo, Identifier: &identify.Pipeline{}})
	mk := func() *redis.Client { return redis.NewClient(&redis.Options{Addr: l.redis.Addr()}) }
	pc, kc, sc := mk(), mk(), mk()
	api.Progress = syncstore.NewRedis(pc)
	api.KOSync = kosync.NewService(kosync.NewRedis(kc), true)
	api.Memory = l.memory
	api.UseScanJournal(sc)
	if err := api.RestoreScans(context.Background()); err != nil {
		return "", nil, err
	}
	var h http.Handler = api.Router()
	if l.wrap != nil {
		h = l.wrap(h)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	srv := httptest.NewUnstartedServer(h)
	srv.Listener.Close()
	srv.Listener = ln
	srv.Start()
	return srv.URL, &inprocProc{srv: srv, api: api, clients: []*redis.Client{pc, kc, sc}}, nil
}

func baseConfig(t *testing.T, l Launcher) LiveServerConfig {
	dir := t.TempDir()
	return LiveServerConfig{
		Launcher: l, EvidenceDir: filepath.Join(dir, "evidence"), Library: filepath.Join(dir, "library"), StreamLib: filepath.Join(dir, "streams"),
		PSECheck: pseCheckBinary(t),
		PSEEnv:   Environ(map[string]string{"HELIX_OPDS_USERNAME": "live-reader", "HELIX_OPDS_PASSWORD": "live-password"}),
	}
}

func freshRepo() catalog.Repository { return catalog.NewMemory() }

func miniredisFresh(t *testing.T) *miniredis.Miniredis { return miniredis.RunT(t) }
