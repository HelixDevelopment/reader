package httpapi

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/kosync"
	"github.com/helixreader/helix/server/internal/sources"
)

const testSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func testServer(t *testing.T) (*Server, *catalog.Memory, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "book.txt"), []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(root, "book.txt"))
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	work := catalog.Work{ID: "w1", CanonicalTitle: "Book", CreatedAt: now, UpdatedAt: now, Editions: []catalog.Edition{{ID: "e1", Title: "Book", State: catalog.Identified, UpdatedAt: now, Files: []catalog.File{{ID: "f1", SourceID: "source-test", Path: "book.txt", Format: "txt", MIMEType: "text/plain", Size: 10, SHA256: testSHA256, ModifiedAt: info.ModTime()}}}}}
	if err := repo.UpsertWork(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{AdminToken: "secret", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	s.connectors["source-test"] = connector
	return s, repo, root
}
func request(t *testing.T, handler http.Handler, method, path, token string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}
func TestContentRangeAndTraversalSafeDisposition(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	r := s.Router()
	w := request(t, r, http.MethodGet, "/v1/files/f1/content", "", nil, map[string]string{"Range": "bytes=2-5"})
	if w.Code != http.StatusPartialContent || w.Body.String() != "2345" {
		t.Fatalf("range failed: %d %q", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Range"); got != "bytes 2-5/10" {
		t.Fatalf("wrong Content-Range: %q", got)
	}
	w = request(t, r, http.MethodGet, "/v1/files/f1/content", "", nil, map[string]string{"Range": "bytes=100-"})
	if w.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("unsatisfiable range: %d", w.Code)
	}
}
func TestCompressionBrotliGzipAndIdentityRefusal(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	r := s.Router()
	for _, encoding := range []string{"br", "gzip"} {
		w := request(t, r, http.MethodGet, "/v1/formats", "", nil, map[string]string{"Accept-Encoding": encoding})
		if w.Code != 200 || w.Header().Get("Content-Encoding") != encoding {
			t.Fatalf("%s negotiation: %d %+v", encoding, w.Code, w.Header())
		}
		var reader io.Reader = w.Body
		if encoding == "br" {
			reader = brotli.NewReader(w.Body)
		} else {
			gz, err := gzip.NewReader(w.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer gz.Close()
			reader = gz
		}
		var payload struct {
			Formats []json.RawMessage `json:"formats"`
		}
		if err := json.NewDecoder(reader).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Formats) == 0 {
			t.Fatal("empty format registry")
		}
	}
	w := request(t, r, http.MethodGet, "/v1/formats", "", nil, map[string]string{"Accept-Encoding": "identity;q=0, br;q=0, gzip;q=0"})
	if w.Code != http.StatusNotAcceptable {
		t.Fatalf("identity refusal not honored: %d", w.Code)
	}
}
func TestCompressionNetworkContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Compression(4))
	payload := bytes.Repeat([]byte("live compression response\n"), 64)
	router.GET("/data", func(c *gin.Context) { c.Data(http.StatusOK, "application/octet-stream", payload) })
	server := httptest.NewServer(router)
	defer server.Close()
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	for _, encoding := range []string{"br", "gzip"} {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/data", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Accept-Encoding", encoding)
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		raw, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("%s truncated network response: read=%v close=%v", encoding, readErr, closeErr)
		}
		if response.ContentLength >= 0 && response.ContentLength != int64(len(raw)) {
			t.Fatalf("%s Content-Length=%d, compressed bytes=%d", encoding, response.ContentLength, len(raw))
		}
		var reader io.Reader = bytes.NewReader(raw)
		if encoding == "br" {
			reader = brotli.NewReader(reader)
		} else {
			gz, err := gzip.NewReader(reader)
			if err != nil {
				t.Fatal(err)
			}
			reader = gz
		}
		decoded, err := io.ReadAll(reader)
		if err != nil || !bytes.Equal(decoded, payload) {
			t.Fatalf("%s decoded response mismatch: %v", encoding, err)
		}
	}
}

func TestCompressionIdentityFallbackPreservesKeepAlive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Compression(4))
	payload := []byte("complete attachment bytes\n")
	router.GET("/attachment", func(c *gin.Context) {
		c.Header("Content-Disposition", `attachment; filename="book.txt"`)
		c.Data(http.StatusOK, "text/plain", payload)
	})
	router.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	server := httptest.NewServer(router)
	defer server.Close()

	for _, encoding := range []string{"br", "gzip"} {
		t.Run(encoding, func(t *testing.T) {
			address := strings.TrimPrefix(server.URL, "http://")
			conn, err := net.Dial("tcp", address)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			reader := bufio.NewReader(conn)
			for _, step := range []struct {
				path string
				body []byte
			}{{"/attachment", payload}, {"/data", []byte("{\"status\":\"ok\"}")}} {
				req, err := http.NewRequest(http.MethodGet, server.URL+step.path, nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Accept-Encoding", encoding)
				if err := req.Write(conn); err != nil {
					t.Fatal(err)
				}
				response, err := http.ReadResponse(reader, req)
				if err != nil {
					t.Fatalf("read %s after identity fallback: %v", step.path, err)
				}
				raw, readErr := io.ReadAll(response.Body)
				closeErr := response.Body.Close()
				if readErr != nil || closeErr != nil {
					t.Fatalf("read %s body: read=%v close=%v", step.path, readErr, closeErr)
				}
				decoded := raw
				if response.Header.Get("Content-Encoding") == "gzip" {
					gz, err := gzip.NewReader(bytes.NewReader(raw))
					if err != nil {
						t.Fatal(err)
					}
					decoded, err = io.ReadAll(gz)
					if err != nil {
						t.Fatal(err)
					}
					if err := gz.Close(); err != nil {
						t.Fatal(err)
					}
				} else if response.Header.Get("Content-Encoding") == "br" {
					decoded, err = io.ReadAll(brotli.NewReader(bytes.NewReader(raw)))
					if err != nil {
						t.Fatal(err)
					}
				}
				if !bytes.Equal(decoded, step.body) {
					t.Fatalf("%s body=%q, want %q", step.path, decoded, step.body)
				}
			}
		})
	}
}

func TestAdminAuthenticationAndOPDS(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	s.Config.OPDSUsername = "reader"
	s.Config.OPDSPassword = "password"
	r := s.Router()
	w := request(t, r, http.MethodPost, "/v1/sources", "", bytes.NewBufferString(`{}`), nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("admin route exposed: %d", w.Code)
	}
	for _, path := range []string{"/opds/1.2/publications", "/v1/works", "/v1/files/f1/content", "/v1/covers/e1"} {
		w = request(t, r, http.MethodGet, path, "", nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("read auth missing for %s: %d", path, w.Code)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/opds/1.2/publications", nil)
	req.SetBasicAuth("reader", "password")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("urn:helix:edition:e1")) {
		t.Fatalf("OPDS feed: %d %s", w.Code, w.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/opds/2.0/editions/e1", nil)
	req.SetBasicAuth("reader", "password")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"@context"`)) {
		t.Fatalf("OPDS 2 edition: %d %s", w.Code, w.Body.String())
	}
}
func TestSourceRegistrationAndScanJob(t *testing.T) {
	s, repo, root := testServer(t)
	defer s.Close()
	r := s.Router()
	body, _ := json.Marshal(sources.Config{Protocol: "local", Root: root})
	w := request(t, r, http.MethodPost, "/v1/sources", "secret", bytes.NewReader(body), nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("create source: %d %s", w.Code, w.Body.String())
	}
	var source sourceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &source); err != nil {
		t.Fatal(err)
	}
	scanBody, _ := json.Marshal(scanRequest{SourceID: source.ID})
	w = request(t, r, http.MethodPost, "/v1/scans", "secret", bytes.NewReader(scanBody), nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("start scan: %d %s", w.Code, w.Body.String())
	}
	var job ScanJob
	_ = json.Unmarshal(w.Body.Bytes(), &job)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		w = request(t, r, http.MethodGet, "/v1/scans/"+job.ID, "secret", nil, nil)
		_ = json.Unmarshal(w.Body.Bytes(), &job)
		if job.State == "succeeded" || job.State == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if job.State != "succeeded" || job.Report == nil || job.Report.Files != 1 {
		t.Fatalf("scan did not succeed: %+v", job)
	}
	page, err := repo.Search(context.Background(), catalog.Query{Text: "book"})
	if err != nil || page.Total < 1 {
		t.Fatalf("scan catalog missing: %+v %v", page, err)
	}
}

type blockingScanConnector struct {
	started   chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
}

func newBlockingScanConnector() *blockingScanConnector {
	return &blockingScanConnector{started: make(chan struct{}), closed: make(chan struct{})}
}

func (c *blockingScanConnector) List(ctx context.Context, _ string) ([]sources.Entry, error) {
	close(c.started)
	<-ctx.Done()
	return nil, ctx.Err()
}
func (c *blockingScanConnector) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, sources.ErrNotFound
}
func (c *blockingScanConnector) Capabilities() sources.Capabilities { return sources.Capabilities{} }
func (c *blockingScanConnector) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

func TestSourceDeletionWaitsForScanAndScanAdmission(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	connector := newBlockingScanConnector()
	s.mu.Lock()
	s.connectors["blocking-source"] = connector
	s.mu.Unlock()
	r := s.Router()
	body, _ := json.Marshal(scanRequest{SourceID: "blocking-source"})
	w := request(t, r, http.MethodPost, "/v1/scans", "secret", bytes.NewReader(body), nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("start blocking scan: %d %s", w.Code, w.Body.String())
	}
	var job ScanJob
	if err := json.Unmarshal(w.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	select {
	case <-connector.started:
	case <-time.After(3 * time.Second):
		t.Fatal("scan did not enter connector List")
	}
	w = request(t, r, http.MethodDelete, "/v1/sources/blocking-source", "secret", nil, nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("delete during active scan: %d %s", w.Code, w.Body.String())
	}
	select {
	case <-connector.closed:
		t.Fatal("active scan connector was closed")
	default:
	}
	w = request(t, r, http.MethodDelete, "/v1/scans/"+job.ID, "secret", nil, nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("cancel scan: %d %s", w.Code, w.Body.String())
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		current, ok := s.jobs.get(job.ID)
		if !ok {
			t.Fatal("scan disappeared before cancellation completed")
		}
		if current.State == "canceled" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("scan did not cancel: %+v", current)
		}
		time.Sleep(5 * time.Millisecond)
	}
	w = request(t, r, http.MethodDelete, "/v1/sources/blocking-source", "secret", nil, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete after cancellation: %d %s", w.Code, w.Body.String())
	}
	select {
	case <-connector.closed:
	default:
		t.Fatal("deleted source connector was not closed")
	}
}

func TestSingleRangeForms(t *testing.T) {
	for _, tc := range []struct {
		value         string
		start, length int64
	}{{"bytes=0-0", 0, 1}, {"bytes=8-", 8, 2}, {"bytes=-3", 7, 3}, {"bytes=8-100", 8, 2}} {
		start, length, err := singleRange(tc.value, 10)
		if err != nil || start != tc.start || length != tc.length {
			t.Errorf("%s: %d %d %v", tc.value, start, length, err)
		}
	}
	for _, value := range []string{"bytes=10-", "bytes=1-2,4-5", "items=1-2", "bytes=--"} {
		if _, _, err := singleRange(value, 10); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}

func TestKOReaderSyncProtocol(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	r := s.Router()
	w := request(t, r, http.MethodPost, "/users/create", "", bytes.NewBufferString(`{"username":"reader","password":"0123456789abcdef0123456789abcdef"}`), nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured sync service: %d %s", w.Code, w.Body.String())
	}
	s.KOSync = kosync.NewService(kosync.NewMemory(), false)
	w = request(t, r, http.MethodPost, "/users/create", "", bytes.NewBufferString(`{"username":"reader","password":"0123456789abcdef0123456789abcdef"}`), nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("registration default is not closed: %d %s", w.Code, w.Body.String())
	}
	s.KOSync = kosync.NewService(kosync.NewMemory(), true)
	w = request(t, r, http.MethodPost, "/users/create", "", bytes.NewBufferString(`{"username":"reader","password":"0123456789abcdef0123456789abcdef"}`), nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: %d %s", w.Code, w.Body.String())
	}
	w = request(t, r, http.MethodPost, "/users/create", "", bytes.NewBufferString(`{"username":"reader","password":"0123456789abcdef0123456789abcdef"}`), nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate user: %d", w.Code)
	}
	auth := map[string]string{"x-auth-user": "reader", "x-auth-key": "0123456789abcdef0123456789abcdef"}
	w = request(t, r, http.MethodGet, "/users/auth", "", nil, auth)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"authorized":"OK"`)) {
		t.Fatalf("auth: %d %s", w.Code, w.Body.String())
	}
	badAuth := map[string]string{"x-auth-user": "reader", "x-auth-key": "wrong-password"}
	w = request(t, r, http.MethodGet, "/users/auth", "", nil, badAuth)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad auth accepted: %d", w.Code)
	}
	progress := `{"document":"book-hash","progress":"/body/DocFragment[5]","percentage":0.47,"device":"Kobo","device_id":"device-a","timestamp":7}`
	w = request(t, r, http.MethodPut, "/syncs/progress", "", bytes.NewBufferString(progress), auth)
	if w.Code != http.StatusOK || bytes.Contains(w.Body.Bytes(), []byte(`"timestamp":7`)) {
		t.Fatalf("update progress: %d %s", w.Code, w.Body.String())
	}
	w = request(t, r, http.MethodGet, "/syncs/progress/book-hash", "", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("get progress: %d %s", w.Code, w.Body.String())
	}
	var stored kosync.Progress
	if err := json.Unmarshal(w.Body.Bytes(), &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Document != "book-hash" || stored.Percentage != .47 || stored.Timestamp == 7 {
		t.Fatalf("stored progress: %+v", stored)
	}
	w = request(t, r, http.MethodPut, "/syncs/progress", "", bytes.NewBufferString(progress), nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated progress update: %d", w.Code)
	}
	w = request(t, r, http.MethodPost, "/users/create", "", bytes.NewBufferString(`{"username":"other","password":"fedcba9876543210fedcba9876543210"}`), nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("create isolation user: %d", w.Code)
	}
	otherAuth := map[string]string{"x-auth-user": "other", "x-auth-key": "fedcba9876543210fedcba9876543210"}
	w = request(t, r, http.MethodGet, "/syncs/progress/book-hash", "", nil, otherAuth)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-user progress leak: %d %s", w.Code, w.Body.String())
	}
}

func TestKOReaderSyncRejectsInvalidAndOversizeProgress(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	s.KOSync = kosync.NewService(kosync.NewMemory(), true)
	r := s.Router()
	w := request(t, r, http.MethodPost, "/users/create", "", bytes.NewBufferString(`{"username":"reader","password":"0123456789abcdef0123456789abcdef"}`), nil)
	if w.Code != http.StatusCreated {
		t.Fatal(w.Body.String())
	}
	auth := map[string]string{"x-auth-user": "reader", "x-auth-key": "0123456789abcdef0123456789abcdef"}
	for _, body := range []string{`{}`, `{"document":"d","progress":"p","device":"d","percentage":1.5}`, `{"document":"d","progress":"p","device":"d","percentage":-0.1}`} {
		w = request(t, r, http.MethodPut, "/syncs/progress", "", bytes.NewBufferString(body), auth)
		if w.Code != http.StatusBadRequest {
			t.Errorf("invalid progress accepted: %d %s", w.Code, body)
		}
	}
	w = request(t, r, http.MethodPut, "/syncs/progress", "", bytes.NewBufferString(`{"document":"`+strings.Repeat("d", 17000)+`"}`), auth)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("oversize body accepted: %d", w.Code)
	}
}

func TestScanRequestBodyLimit(t *testing.T) {
	s, _, _ := testServer(t)
	defer s.Close()
	r := s.Router()
	payload := `{"source_id":"source-test","prefix":"` + strings.Repeat("x", int(scanJSONLimit)) + `"}`
	w := request(t, r, http.MethodPost, "/v1/scans", "secret", bytes.NewBufferString(payload), nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("oversized scan body must be rejected before starting a job: %d", w.Code)
	}
	if jobs := s.jobs.list(); len(jobs) != 0 {
		t.Fatalf("oversized request started jobs: %+v", jobs)
	}
}

func TestStableSourceIDReattachesCatalogAfterRestart(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "book.txt"), []byte("stable-source-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	newServer := func() *Server {
		return New(config.Config{AdminToken: "secret"}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	}
	create := func(s *Server, id string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(sources.Config{ID: id, Protocol: "local", Root: root})
		return request(t, s.Router(), http.MethodPost, "/v1/sources", "secret", bytes.NewReader(body), nil)
	}
	first := newServer()
	if w := create(first, "library-main"); w.Code != http.StatusCreated {
		t.Fatalf("stable source creation: %d %s", w.Code, w.Body.String())
	}
	if w := create(first, "library-main"); w.Code != http.StatusConflict {
		t.Fatalf("duplicate active source ID: %d %s", w.Code, w.Body.String())
	}
	if w := create(first, "invalid/id"); w.Code != http.StatusBadRequest {
		t.Fatalf("invalid source ID: %d %s", w.Code, w.Body.String())
	}
	connector, _ := first.connector("library-main")
	if _, err := first.Ingest.ScanSource(context.Background(), "library-main", connector, ""); err != nil {
		t.Fatal(err)
	}
	page, err := repo.Search(context.Background(), catalog.Query{Limit: 100})
	if err != nil || page.Total != 1 || len(page.Works[0].Editions[0].Files) != 1 {
		t.Fatalf("initial scan: %+v %v", page, err)
	}
	file := page.Works[0].Editions[0].Files[0]
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second := newServer()
	defer second.Close()
	if w := create(second, "library-main"); w.Code != http.StatusCreated {
		t.Fatalf("source re-registration: %d %s", w.Code, w.Body.String())
	}
	w := request(t, second.Router(), http.MethodGet, "/v1/files/"+file.ID+"/content", "", nil, nil)
	if w.Code != http.StatusOK || w.Body.String() != "stable-source-bytes" {
		t.Fatalf("old file identity did not reattach: %d %q", w.Code, w.Body.String())
	}
	connector, _ = second.connector("library-main")
	report, err := second.Ingest.ScanSource(context.Background(), "library-main", connector, "")
	if err != nil || report.Removed != 0 {
		t.Fatalf("re-registration scan: %+v %v", report, err)
	}
	page, err = repo.Search(context.Background(), catalog.Query{Limit: 100})
	if err != nil || page.Total != 1 || len(page.Works[0].Editions) != 1 || len(page.Works[0].Editions[0].Files) != 1 {
		t.Fatalf("stable source re-registration duplicated files: %+v %v", page, err)
	}
	if page.Works[0].Editions[0].Files[0].ID != file.ID {
		t.Fatal("stable rescan changed file identity")
	}
}
