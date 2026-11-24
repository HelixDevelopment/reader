package sources

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

// A non-2xx propstat (properties the resource lacks) must not contribute: its
// length, mtime and collection flag are not facts about the resource.
func TestWebDAVRevalidateIgnoresNon2xxPropstat(t *testing.T) {
	x := `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/b.bin</D:href>` +
		`<D:propstat><D:prop><D:getcontentlength>99</D:getcontentlength><D:getlastmodified>Tue, 01 Dec 2026 00:00:00 GMT</D:getlastmodified></D:prop><D:status>HTTP/1.1 404 Not Found</D:status></D:propstat>` +
		`<D:propstat><D:prop><D:getcontentlength>42</D:getcontentlength><D:getlastmodified>Mon, 23 Nov 2026 21:47:33 GMT</D:getlastmodified><D:resourcetype/></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat>` +
		`</D:response></D:multistatus>`
	size, mod, err := revalServer(t, x).Revalidate(context.Background(), "b.bin")
	if err != nil || size != 42 || mod.UTC().Format(http.TimeFormat) != "Mon, 23 Nov 2026 21:47:33 GMT" {
		t.Fatalf("size=%d mod=%v err=%v, want 42 / 23 Nov", size, mod, err)
	}
}

// scriptedRclone writes an executable stand-in for the rclone binary that prints
// stdout and exits with rc (stderr text on failure).
func scriptedRclone(t *testing.T, stdout, stderr string, rc int) *Rclone {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s' '" + stdout + "'\nprintf '%s' '" + stderr + "' >&2\nexit " + strconv.Itoa(rc) + "\n"
	bin := filepath.Join(dir, "rclone")
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	c, err := NewRclone(Config{Protocol: "sftp", Endpoint: "r:root", RclonePath: bin})
	if err != nil {
		t.Fatal(err)
	}
	return c.(*Rclone)
}

func TestRcloneRevalidateUnit(t *testing.T) {
	ctx := context.Background()
	ok := `{"Path":"b.bin","Size":42,"ModTime":"2026-11-23T21:47:33.123456789Z","IsDir":false}`
	size, mod, err := scriptedRclone(t, ok, "", 0).Revalidate(ctx, "b.bin")
	if err != nil || size != 42 || mod.Nanosecond() != 123456789 {
		t.Fatalf("ok: %d %v %v", size, mod, err)
	}
	if _, _, err := scriptedRclone(t, `{"Size":0,"IsDir":true,"ModTime":"2026-11-23T21:47:33Z"}`, "", 0).Revalidate(ctx, "d"); err == nil {
		t.Fatal("directory must not revalidate as a file")
	}
	if _, _, err := scriptedRclone(t, ok, "", 0).Revalidate(ctx, "../x"); !errors.Is(err, ErrTraversal) {
		t.Fatalf("leading traversal: %v", err)
	}
	if _, _, err := scriptedRclone(t, ok, "", 0).Revalidate(ctx, "a/../x"); !errors.Is(err, ErrTraversal) {
		t.Fatalf("embedded traversal: %v", err)
	}
	// Unparsable ModTime or negative size: unknown metadata (-1, nil), not an error.
	for _, bad := range []string{`{"Size":42,"ModTime":"garbage","IsDir":false}`, `{"Size":-1,"ModTime":"2026-11-23T21:47:33Z","IsDir":false}`} {
		size, _, err := scriptedRclone(t, bad, "", 0).Revalidate(ctx, "b.bin")
		if err != nil || size != -1 {
			t.Fatalf("unknown metadata %s: size=%d err=%v, want -1,nil", bad, size, err)
		}
	}
	if _, _, err := scriptedRclone(t, "", "directory not found", 3).Revalidate(ctx, "b.bin"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("exit 3: %v", err)
	}
	if _, _, err := scriptedRclone(t, "", "boom", 1).Revalidate(ctx, "b.bin"); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("generic failure must be a non-404 error: %v", err)
	}
}

func TestWebDAVRevalidateLaterEmptyLengthDoesNotOverwrite(t *testing.T) {
	x := `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/b.bin</D:href>` +
		`<D:propstat><D:prop><D:getcontentlength>42</D:getcontentlength><D:getlastmodified>Mon, 23 Nov 2026 21:47:33 GMT</D:getlastmodified><D:resourcetype/></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat>` +
		`<D:propstat><D:prop><D:getcontentlength></D:getcontentlength></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat>` +
		`</D:response></D:multistatus>`
	size, _, err := revalServer(t, x).Revalidate(context.Background(), "b.bin")
	if err != nil || size != 42 {
		t.Fatalf("size=%d err=%v, want 42", size, err)
	}
}

func TestWebDAVRevalidateFirstResponseWinsWithSeveral(t *testing.T) {
	r := func(href, n string) string {
		return `<D:response><D:href>` + href + `</D:href><D:propstat><D:prop><D:getcontentlength>` + n + `</D:getcontentlength><D:getlastmodified>Mon, 23 Nov 2026 21:47:33 GMT</D:getlastmodified><D:resourcetype/></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response>`
	}
	x := `<?xml version="1.0"?><D:multistatus xmlns:D="DAV:">` + r("/b.bin", "42") + r("/c.bin", "7") + `</D:multistatus>`
	size, _, err := revalServer(t, x).Revalidate(context.Background(), "b.bin")
	if err != nil || size != 42 {
		t.Fatalf("size=%d err=%v, want 42 from the first response", size, err)
	}
}

func TestWebDAVRevalidateSendsAuthAndConfiguredHeaders(t *testing.T) {
	var user, pass, hdr string
	var authed bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, authed = r.BasicAuth()
		hdr = r.Header.Get("X-Test-Token")
		w.WriteHeader(207)
		_, _ = io.WriteString(w, propOK)
	}))
	defer srv.Close()
	c, err := NewWebDAV(Config{Protocol: "webdav", Endpoint: srv.URL + "/", Username: "alice", Password: "pw",
		Headers: map[string]string{"X-Test-Token": "t1"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.(*WebDAV).Revalidate(context.Background(), "b.bin"); err != nil {
		t.Fatal(err)
	}
	if !authed || user != "alice" || pass != "pw" {
		t.Fatalf("PROPFIND basic auth missing: ok=%v user=%q", authed, user)
	}
	if hdr != "t1" {
		t.Fatalf("configured header not sent on PROPFIND: %q", hdr)
	}
}

// Only 207 Multi-Status is a PROPFIND answer; a plain 200 is not.
func TestWebDAVRevalidateRejectsPlain200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = io.WriteString(w, propOK)
	}))
	defer srv.Close()
	c, _ := NewWebDAV(Config{Protocol: "webdav", Endpoint: srv.URL + "/"})
	if _, _, err := c.(*WebDAV).Revalidate(context.Background(), "b.bin"); err == nil {
		t.Fatal("HTTP 200 must not be accepted as a PROPFIND answer")
	}
}

// Exit codes 3/4 alone map to ErrNotFound (stderr carries no marker text).
func TestRcloneRevalidateExitCodesMapToNotFound(t *testing.T) {
	for _, rc := range []int{3, 4} {
		if _, _, err := scriptedRclone(t, "", "", rc).Revalidate(context.Background(), "b.bin"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("exit %d: %v, want ErrNotFound", rc, err)
		}
	}
}
