package sources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Cross-origin acquisition to a loopback/private host requires the explicit
// AllowPrivateNetwork opt-in. Configured headers must never leak cross-origin.
func TestOPDSPrivateNetworkOptInAllowsCrossOriginAcquisitionWithoutHeaders(t *testing.T) {
	var leaked atomic.Bool
	files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Secret") != "" {
			leaked.Store(true)
		}
		w.Header().Set("Content-Type", "application/epub+zip")
		_, _ = io.WriteString(w, "epub-bytes")
	}))
	defer files.Close()
	var catalog *httptest.Server
	catalog = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/opds":
			w.Header().Set("Content-Type", "application/opds+json")
			fmt.Fprintf(w, `{"publications":[{"metadata":{"title":"Remote"},"links":[{"rel":"http://opds-spec.org/acquisition","href":%q,"type":"application/epub+zip"}]}],"links":[{"rel":"next","href":"/page2"}]}`, files.URL+"/book.epub?download=1")
		case "/page2":
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = io.WriteString(w, `<feed xmlns="http://www.w3.org/2005/Atom"><entry><title>Local</title><link rel="http://opds-spec.org/acquisition" href="/local.epub" type="application/epub+zip"/></entry></feed>`)
		case "/local.epub":
			_, _ = io.WriteString(w, "local")
		default:
			http.NotFound(w, r)
		}
	}))
	defer catalog.Close()
	connector, err := NewOPDS(Config{Endpoint: catalog.URL + "/opds", Headers: map[string]string{"X-Secret": "do-not-forward"}, AllowPrivateNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	entries, err := connector.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("wrong acquisitions: %+v", entries)
	}
	r, err := connector.Open(context.Background(), entries[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	r.Close()
	if err != nil || string(data) != "epub-bytes" {
		t.Fatalf("cross-origin open: %q %v", data, err)
	}
	if leaked.Load() {
		t.Fatal("source credential header leaked cross-origin")
	}
}

// Without AllowPrivateNetwork, a cross-origin acquisition pointing at a
// loopback/private address must be rejected at dial time and never contacted.
func TestOPDSRejectsCrossOriginLoopbackAcquisitionByDefault(t *testing.T) {
	var hits atomic.Int64
	files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = io.WriteString(w, "epub-bytes")
	}))
	defer files.Close()
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/opds+json")
		fmt.Fprintf(w, `{"publications":[{"metadata":{"title":"Remote"},"links":[{"rel":"acquisition","href":%q,"type":"application/epub+zip"}]}]}`, files.URL+"/book.epub")
	}))
	defer catalog.Close()
	connector, err := NewOPDS(Config{Endpoint: catalog.URL + "/opds", Headers: map[string]string{"X-Secret": "do-not-forward"}})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	entries, err := connector.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("wrong acquisitions: %+v", entries)
	}
	if _, err := connector.Open(context.Background(), entries[0].Path); err == nil || !strings.Contains(err.Error(), errAddrPrivate.Error()) {
		t.Fatalf("cross-origin loopback acquisition must be rejected, got %v", err)
	}
	if hits.Load() != 0 {
		t.Fatal("blocked acquisition target was contacted")
	}
}

// Same-origin loopback endpoints (e.g. a LAN NAS) keep working without the
// flag, and configured headers are sent to the configured origin.
func TestOPDSSameOriginLoopbackAllowedWithoutFlag(t *testing.T) {
	var sawHeader atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/opds":
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = io.WriteString(w, `<feed xmlns="http://www.w3.org/2005/Atom"><entry><title>Local</title><link rel="http://opds-spec.org/acquisition" href="/book.epub" type="application/epub+zip"/></entry></feed>`)
		case "/book.epub":
			if r.Header.Get("X-Secret") == "same-origin-ok" {
				sawHeader.Store(true)
			}
			_, _ = io.WriteString(w, "epub-bytes")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	connector, err := NewOPDS(Config{Endpoint: srv.URL + "/opds", Headers: map[string]string{"X-Secret": "same-origin-ok"}})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	entries, err := connector.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("wrong acquisitions: %+v", entries)
	}
	r, err := connector.Open(context.Background(), entries[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	r.Close()
	if err != nil || string(data) != "epub-bytes" {
		t.Fatalf("same-origin open: %q %v", data, err)
	}
	if !sawHeader.Load() {
		t.Fatal("configured header not sent to same-origin endpoint")
	}
}

// Link-local metadata endpoints stay rejected even with AllowPrivateNetwork,
// and are rejected before any network dial occurs.
func TestOPDSRejectsLinkLocalMetadataAcquisition(t *testing.T) {
	for _, allow := range []bool{false, true} {
		t.Run(fmt.Sprintf("allow_private_network=%t", allow), func(t *testing.T) {
			catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/opds+json")
				_, _ = io.WriteString(w, `{"publications":[{"metadata":{"title":"Meta"},"links":[{"rel":"acquisition","href":"http://169.254.169.254/latest/meta-data","type":"text/plain"}]}]}`)
			}))
			defer catalog.Close()
			connector, err := NewOPDS(Config{Endpoint: catalog.URL + "/opds", AllowPrivateNetwork: allow})
			if err != nil {
				t.Fatal(err)
			}
			defer connector.Close()
			entries, err := connector.List(context.Background(), "")
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("wrong acquisitions: %+v", entries)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := connector.Open(ctx, entries[0].Path); err == nil || !strings.Contains(err.Error(), errAddrRestricted.Error()) {
				t.Fatalf("link-local acquisition must be rejected, got %v", err)
			}
		})
	}
}

// A same-origin acquisition that redirects to a cross-origin loopback target
// is rejected at the redirect hop's dial, and the target is never contacted.
func TestOPDSRejectsRedirectToCrossOriginLoopback(t *testing.T) {
	var hits atomic.Int64
	files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = io.WriteString(w, "epub-bytes")
	}))
	defer files.Close()
	var catalog *httptest.Server
	catalog = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/opds":
			w.Header().Set("Content-Type", "application/opds+json")
			_, _ = io.WriteString(w, `{"publications":[{"metadata":{"title":"Redirect"},"links":[{"rel":"acquisition","href":"/redir","type":"application/epub+zip"}]}]}`)
		case "/redir":
			http.Redirect(w, r, files.URL+"/book.epub", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer catalog.Close()
	connector, err := NewOPDS(Config{Endpoint: catalog.URL + "/opds", Headers: map[string]string{"X-Secret": "do-not-forward"}})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	entries, err := connector.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("wrong acquisitions: %+v", entries)
	}
	if _, err := connector.Open(context.Background(), entries[0].Path); err == nil || !strings.Contains(err.Error(), errAddrPrivate.Error()) {
		t.Fatalf("redirect to loopback must be rejected, got %v", err)
	}
	if hits.Load() != 0 {
		t.Fatal("redirect target was contacted")
	}
}

func TestOPDSRejectsOversizedFeed(t *testing.T) {
	payload := bytes.Repeat([]byte{'x'}, 32<<20+1)
	for _, jsonFeed := range []bool{false, true} {
		if _, err := parseFeed(bytes.NewReader(payload), jsonFeed); err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("oversized feed must be rejected (json=%t), got %v", jsonFeed, err)
		}
	}
}

func TestOPDSRejectsTruncatedPaginationInsteadOfReturningPartialSnapshot(t *testing.T) {
	var hits atomic.Int64
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		page := 0
		if r.URL.Path != "/opds" {
			if _, err := fmt.Sscanf(r.URL.Path, "/page/%d", &page); err != nil {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "application/opds+json")
		fmt.Fprintf(w, `{"publications":[],"links":[{"rel":"next","href":%q}]}`, fmt.Sprintf("/page/%d", page+1))
	}))
	defer catalog.Close()
	connector, err := NewOPDS(Config{Endpoint: catalog.URL + "/opds"})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	entries, err := connector.List(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "page limit") || entries != nil {
		t.Fatalf("truncated OPDS listing must fail closed: entries=%+v err=%v", entries, err)
	}
	if hits.Load() != 100 {
		t.Fatalf("unexpected page-fetch count: %d", hits.Load())
	}
}

func TestOPDSKeepsSignedAcquisitionSecretsOutOfCatalogPaths(t *testing.T) {
	var sawSignedRequest atomic.Bool
	files := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") == "secret" && r.URL.Query().Get("download") == "1" {
			sawSignedRequest.Store(true)
			_, _ = io.WriteString(w, "signed-epub-bytes")
			return
		}
		http.Error(w, "missing signature", http.StatusForbidden)
	}))
	defer files.Close()
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/opds+json")
		fmt.Fprintf(w, `{"publications":[{"metadata":{"title":"Signed"},"links":[{"rel":"acquisition","href":%q,"type":"application/epub+zip"}]}]}`, files.URL+"/book.epub?download=1&token=secret#frag")
	}))
	defer catalog.Close()
	connector, err := NewOPDS(Config{Endpoint: catalog.URL + "/opds", AllowPrivateNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	entries, err := connector.List(context.Background(), "")
	if err != nil || len(entries) != 1 {
		t.Fatalf("signed acquisition list: entries=%+v err=%v", entries, err)
	}
	publicPath := entries[0].Path
	if strings.Contains(publicPath, "secret") || strings.Contains(publicPath, "token=") || strings.Contains(publicPath, "download=") || strings.Contains(publicPath, "#") {
		t.Fatalf("signed URL secrets entered public catalog path: %q", publicPath)
	}
	if publicPath != files.URL+"/book.epub" {
		t.Fatalf("unexpected public acquisition identity: %q", publicPath)
	}
	rc, err := connector.Open(context.Background(), publicPath)
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(rc)
	closeErr := rc.Close()
	if readErr != nil || closeErr != nil || string(data) != "signed-epub-bytes" || !sawSignedRequest.Load() {
		t.Fatalf("process-local signed resolution failed: data=%q read=%v close=%v signed=%t", data, readErr, closeErr, sawSignedRequest.Load())
	}
}

func TestOPDSQueryAddressedAcquisitionsKeepDistinctPublicIdentities(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("id") {
		case "1":
			_, _ = io.WriteString(w, "book-one-bytes")
		case "2":
			_, _ = io.WriteString(w, "book-two-bytes")
		default:
			http.Error(w, "missing id", http.StatusBadRequest)
		}
	}))
	defer backend.Close()
	catalogServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/opds+json")
		fmt.Fprintf(w, `{"publications":[{"metadata":{"title":"Book One"},"links":[{"rel":"acquisition","href":%q,"type":"application/epub+zip"}]},{"metadata":{"title":"Book Two"},"links":[{"rel":"acquisition","href":%q,"type":"application/epub+zip"}]}]}`, backend.URL+"/dl/book.epub?id=1", backend.URL+"/dl/book.epub?id=2")
	}))
	defer catalogServer.Close()
	connector, err := NewOPDS(Config{Endpoint: catalogServer.URL + "/opds", AllowPrivateNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	entries, err := connector.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Path == entries[1].Path {
		t.Fatalf("query-addressed acquisitions collapsed: %+v", entries)
	}
	bodies := map[string]bool{}
	for _, entry := range entries {
		if strings.Contains(entry.Path, "?") || strings.Contains(entry.Path, "id=") {
			t.Fatalf("query leaked into public identity %q", entry.Path)
		}
		rc, err := connector.Open(context.Background(), entry.Path)
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read=%v close=%v", readErr, closeErr)
		}
		bodies[string(data)] = true
	}
	if !bodies["book-one-bytes"] || !bodies["book-two-bytes"] {
		t.Fatalf("acquisition map returned wrong bytes: %v", bodies)
	}
}

func TestOPDSCollisionKeysSurviveReorderAndSiblingRemoval(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") == "2" {
			_, _ = io.WriteString(w, "book-two-bytes")
			return
		}
		_, _ = io.WriteString(w, "book-one-bytes")
	}))
	defer backend.Close()
	order := []string{"1", "2"}
	catalogServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/opds+json")
		publications := ""
		for _, id := range order {
			publications += fmt.Sprintf(`{"metadata":{"title":"Book %s"},"links":[{"rel":"acquisition","href":%q,"type":"application/epub+zip"}]},`, id, backend.URL+"/dl/book.epub?id="+id)
		}
		fmt.Fprintf(w, `{"publications":[%s]}`, strings.TrimSuffix(publications, ","))
	}))
	defer catalogServer.Close()
	connector, err := NewOPDS(Config{Endpoint: catalogServer.URL + "/opds", AllowPrivateNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	keyForBookTwo := func() string {
		entries, err := connector.List(context.Background(), "")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			rc, err := connector.Open(context.Background(), entry.Path)
			if err != nil {
				t.Fatal(err)
			}
			data, readErr := io.ReadAll(rc)
			closeErr := rc.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("read=%v close=%v", readErr, closeErr)
			}
			if string(data) == "book-two-bytes" {
				if !strings.Contains(entry.Path, "#helix-x-") {
					t.Fatalf("colliding member lacks stable synthetic key: %q", entry.Path)
				}
				return entry.Path
			}
		}
		t.Fatal("book two missing")
		return ""
	}
	initial := keyForBookTwo()
	order = []string{"2", "1"}
	if reordered := keyForBookTwo(); reordered != initial {
		t.Fatalf("feed reorder changed identity: %q -> %q", initial, reordered)
	}
	order = []string{"2"}
	if surviving := keyForBookTwo(); surviving != initial {
		t.Fatalf("sibling removal changed identity: %q -> %q", initial, surviving)
	}
	order = []string{"1", "2"}
	if restored := keyForBookTwo(); restored != initial {
		t.Fatalf("sibling restoration changed identity: %q -> %q", initial, restored)
	}
}
