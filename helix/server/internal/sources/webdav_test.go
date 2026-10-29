package sources

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWebDAVList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" || r.Header.Get("Depth") != "infinity" {
			t.Errorf("unexpected request: %s depth=%q", r.Method, r.Header.Get("Depth"))
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		_, _ = w.Write([]byte(`<D:multistatus xmlns:D="DAV:"><D:response><D:href>/dav/book.txt</D:href><D:propstat><D:prop><D:getcontentlength>12</D:getcontentlength><D:getlastmodified>Wed, 28 Oct 2026 10:00:00 GMT</D:getlastmodified><D:getetag>"abc"</D:getetag><D:resourcetype/></D:prop></D:propstat></D:response></D:multistatus>`))
	}))
	defer ts.Close()
	connector, err := NewWebDAV(Config{Protocol: "webdav", Endpoint: ts.URL + "/dav/"})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	entries, err := connector.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Path != "book.txt" || entries[0].Size != 12 || entries[0].ETag != "abc" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestWebDAVListAppliesConfiguredHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Source-Token") != "configured-secret" {
			t.Errorf("configured header missing on PROPFIND: %v", r.Header)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		_, _ = w.Write([]byte(`<D:multistatus xmlns:D="DAV:"><D:response><D:href>/dav/book.txt</D:href><D:propstat><D:prop><D:getcontentlength>12</D:getcontentlength><D:resourcetype/></D:prop></D:propstat></D:response></D:multistatus>`))
	}))
	defer ts.Close()
	connector, err := NewWebDAV(Config{Protocol: "webdav", Endpoint: ts.URL + "/dav/", Headers: map[string]string{"X-Source-Token": "configured-secret"}})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	entries, err := connector.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Path != "book.txt" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestWebDAVRejectsCrossOriginRedirectBeforeRequest(t *testing.T) {
	var targetRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Source-Token") != "configured-secret" {
			t.Errorf("configured header missing on initial PROPFIND")
		}
		http.Redirect(w, r, target.URL+"/dav/", http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	connector, err := NewWebDAV(Config{
		Protocol: "webdav", Endpoint: source.URL + "/dav/",
		Username: "user", Password: "pass",
		Headers: map[string]string{"X-Source-Token": "configured-secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	if _, err := connector.List(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "cross-origin redirect rejected") {
		t.Fatalf("cross-origin redirect must be rejected before request: %v", err)
	}
	if got := targetRequests.Load(); got != 0 {
		t.Fatalf("cross-origin WebDAV target received %d requests", got)
	}
}

func TestWebDAVRejectsOversizedResponse(t *testing.T) {
	payload := bytes.Repeat([]byte{'x'}, (16<<20)+1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(207)
		_, _ = w.Write(payload)
	}))
	defer ts.Close()
	connector, err := NewWebDAV(Config{Protocol: "webdav", Endpoint: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	if _, err := connector.List(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized response must be rejected, got %v", err)
	}
}
