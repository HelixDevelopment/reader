package sources

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const propOK = `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/b.bin</D:href><D:propstat><D:prop>` +
	`<D:getcontentlength>42</D:getcontentlength><D:getlastmodified>Mon, 23 Nov 2026 21:47:33 GMT</D:getlastmodified><D:resourcetype/>` +
	`</D:prop></D:propstat></D:response></D:multistatus>`

func TestWebDAVRevalidate(t *testing.T) {
	mode := "ok"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" || r.Header.Get("Depth") != "0" {
			t.Errorf("unexpected %s depth=%q", r.Method, r.Header.Get("Depth"))
		}
		switch mode {
		case "ok":
			w.WriteHeader(207)
			_, _ = io.WriteString(w, propOK)
		case "missing":
			w.WriteHeader(404)
		case "dir":
			w.WriteHeader(207)
			_, _ = io.WriteString(w, strings.Replace(propOK, "<D:resourcetype/>", "<D:resourcetype><D:collection/></D:resourcetype>", 1))
		case "nomtime":
			w.WriteHeader(207)
			_, _ = io.WriteString(w, strings.Replace(propOK, "<D:getlastmodified>Mon, 23 Nov 2026 21:47:33 GMT</D:getlastmodified>", "", 1))
		case "boom":
			w.WriteHeader(500)
		}
	}))
	defer srv.Close()
	c, err := NewWebDAV(Config{Protocol: "webdav", Endpoint: srv.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	w := c.(*WebDAV)
	ctx := context.Background()
	size, mod, err := w.Revalidate(ctx, "b.bin")
	if err != nil || size != 42 || mod.UTC().Format(http.TimeFormat) != "Mon, 23 Nov 2026 21:47:33 GMT" {
		t.Fatalf("ok: %d %v %v", size, mod, err)
	}
	mode = "missing"
	if _, _, err := w.Revalidate(ctx, "b.bin"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	mode = "dir"
	if _, _, err := w.Revalidate(ctx, "b.bin"); err == nil {
		t.Fatal("collection must not revalidate as a file")
	}
	mode = "nomtime"
	if size, _, err := w.Revalidate(ctx, "b.bin"); err != nil || size != -1 {
		t.Fatalf("no mtime must report unknown, got %d %v", size, err)
	}
	mode = "boom"
	if _, _, err := w.Revalidate(ctx, "b.bin"); err == nil {
		t.Fatal("5xx must be an error")
	}
	if _, _, err := w.Revalidate(ctx, "../x"); !errors.Is(err, ErrTraversal) {
		t.Fatalf("traversal: %v", err)
	}
}

// A server that ignores Range and answers 200 with the whole body must never
// over-deliver past the requested window.
func TestWebDAVOpenRangeBoundsIgnoringServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", 1000))
	}))
	defer srv.Close()
	c, _ := NewWebDAV(Config{Protocol: "webdav", Endpoint: srv.URL + "/"})
	rc, err := c.(*WebDAV).OpenRange(context.Background(), "f", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	if len(b) != 10 {
		t.Fatalf("delivered %d bytes, window was 10", len(b))
	}
}

// Captured from `rclone serve webdav` (rclone v1.60.1): a collection answers
// with a 200 propstat carrying resourcetype/collection and a SECOND 404 propstat
// for the properties directories lack. The directory must not be listed as a
// zero-byte file.
const rcloneDepthInfinity = `<?xml version="1.0" encoding="UTF-8"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/</D:href><D:propstat><D:prop><D:displayname></D:displayname><D:getlastmodified>Mon, 23 Nov 2026 21:54:04 GMT</D:getlastmodified><D:resourcetype><D:collection xmlns:D="DAV:"/></D:resourcetype></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat><D:propstat><D:prop><D:getcontentlength></D:getcontentlength><D:getetag></D:getetag></D:prop><D:status>HTTP/1.1 404 Not Found</D:status></D:propstat></D:response><D:response><D:href>/books/</D:href><D:propstat><D:prop><D:displayname>books</D:displayname><D:getlastmodified>Mon, 23 Nov 2026 21:54:04 GMT</D:getlastmodified><D:resourcetype><D:collection xmlns:D="DAV:"/></D:resourcetype></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat><D:propstat><D:prop><D:getcontentlength></D:getcontentlength><D:getetag></D:getetag></D:prop><D:status>HTTP/1.1 404 Not Found</D:status></D:propstat></D:response><D:response><D:href>/books/r.bin</D:href><D:propstat><D:prop><D:displayname>r.bin</D:displayname><D:getcontentlength>3</D:getcontentlength><D:getlastmodified>Mon, 23 Nov 2026 21:54:04 GMT</D:getlastmodified><D:getetag>"18eacb384b37b5293"</D:getetag><D:resourcetype></D:resourcetype></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response></D:multistatus>`

func TestWebDAVListSkipsCollectionsWithSecondPropstat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(207)
		_, _ = io.WriteString(w, rcloneDepthInfinity)
	}))
	defer srv.Close()
	c, _ := NewWebDAV(Config{Protocol: "webdav", Endpoint: srv.URL + "/"})
	entries, err := c.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Path != "books/r.bin" || entries[0].Size != 3 {
		t.Fatalf("want exactly the one file, got %+v", entries)
	}
}

func revalServer(t *testing.T, xmlBody string) *WebDAV {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(207)
		_, _ = io.WriteString(w, xmlBody)
	}))
	t.Cleanup(srv.Close)
	c, err := NewWebDAV(Config{Protocol: "webdav", Endpoint: srv.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	return c.(*WebDAV)
}

// Origin omits getcontentlength entirely (valid for dynamic resources): the
// size is UNKNOWN (-1), never 0, or the route would serve an empty body.
func TestWebDAVRevalidateAbsentLengthIsUnknownNotZero(t *testing.T) {
	absent := strings.Replace(propOK, "<D:getcontentlength>42</D:getcontentlength>", "", 1)
	emptyEl := strings.Replace(propOK, "<D:getcontentlength>42</D:getcontentlength>", "<D:getcontentlength></D:getcontentlength>", 1)
	notNum := strings.Replace(propOK, "<D:getcontentlength>42</D:getcontentlength>", "<D:getcontentlength>abc</D:getcontentlength>", 1)
	for name, body := range map[string]string{"absent": absent, "empty": emptyEl, "non-numeric": notNum} {
		size, _, err := revalServer(t, body).Revalidate(context.Background(), "b.bin")
		if err != nil || size != -1 {
			t.Fatalf("%s: size=%d err=%v, want -1,nil", name, size, err)
		}
	}
}

// A 404 propstat listed BEFORE the 200 one must not hide the real properties.
func TestWebDAVRevalidateMixedPropstatOrder(t *testing.T) {
	body := `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/b.bin</D:href>` +
		`<D:propstat><D:prop><D:getetag></D:getetag></D:prop><D:status>HTTP/1.1 404 Not Found</D:status></D:propstat>` +
		`<D:propstat><D:prop><D:getcontentlength>42</D:getcontentlength><D:getlastmodified>Mon, 23 Nov 2026 21:47:33 GMT</D:getlastmodified><D:resourcetype/></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat>` +
		`</D:response></D:multistatus>`
	size, mod, err := revalServer(t, body).Revalidate(context.Background(), "b.bin")
	if err != nil || size != 42 || mod.IsZero() {
		t.Fatalf("size=%d mod=%v err=%v", size, mod, err)
	}
	// Same ordering for List.
	entries, err := revalServer(t, strings.Replace(body, "/b.bin", "/d/b.bin", 1)).List(context.Background(), "")
	if err != nil || len(entries) != 1 || entries[0].Size != 42 {
		t.Fatalf("list: %+v %v", entries, err)
	}
}

// List must not invent sizes: only a plain non-negative decimal that fits int64
// is a size. Anything else (absent, empty, signed, hex, float, separators,
// overflow) has no trustworthy size, so the entry is skipped rather than
// catalogued with a made-up one.
func TestWebDAVListStrictContentLength(t *testing.T) {
	entry := func(length string) string {
		prop := "<D:getcontentlength>" + length + "</D:getcontentlength>"
		if length == "<absent>" {
			prop = ""
		}
		return `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/b.bin</D:href><D:propstat><D:prop>` + prop +
			`<D:getlastmodified>Mon, 23 Nov 2026 21:47:33 GMT</D:getlastmodified><D:resourcetype/></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response></D:multistatus>`
	}
	good := map[string]int64{"0": 0, "42": 42, "9223372036854775807": 9223372036854775807}
	for in, want := range good {
		es, err := revalServer(t, entry(in)).List(context.Background(), "")
		if err != nil || len(es) != 1 || es[0].Size != want {
			t.Fatalf("%q: %+v %v, want size %d", in, es, err, want)
		}
	}
	for _, in := range []string{"99999999999999999999", "-5", "0x10", "1.5", "1,000", "", "<absent>", "+7", " "} {
		es, err := revalServer(t, entry(in)).List(context.Background(), "")
		if err != nil || len(es) != 0 {
			t.Fatalf("%q: want the entry skipped, got %+v %v", in, es, err)
		}
	}
}

// A collection that reports a content length (some servers do) is still a
// directory and must not be listed as a file.
func TestWebDAVListSkipsCollectionWithLength(t *testing.T) {
	body := `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/d/</D:href><D:propstat><D:prop><D:getcontentlength>4096</D:getcontentlength><D:getlastmodified>Mon, 23 Nov 2026 21:47:33 GMT</D:getlastmodified><D:resourcetype><D:collection/></D:resourcetype></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response></D:multistatus>`
	es, err := revalServer(t, body).List(context.Background(), "")
	if err != nil || len(es) != 0 {
		t.Fatalf("%+v %v", es, err)
	}
}
