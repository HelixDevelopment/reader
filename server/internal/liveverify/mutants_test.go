package liveverify

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// rewrite wraps h so match(r) requests are served by h with identity encoding and
// then passed through mut, which may change status, headers and body. Everything
// else is untouched. This is how a "broken server" mutant is expressed: the
// verifier must turn each one into the named FAIL.
func rewrite(match func(*http.Request) bool, mut func(r *http.Request, status *int, h http.Header, body *[]byte)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !match(r) {
				next.ServeHTTP(w, r)
				return
			}
			r.Header.Set("Accept-Encoding", "identity")
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)
			status := rec.Code
			body := rec.Body.Bytes()
			h := rec.Header().Clone()
			mut(r, &status, h, &body)
			for k, v := range h {
				w.Header()[k] = v
			}
			if r.Method != http.MethodHead {
				w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			}
			w.WriteHeader(status)
			if r.Method != http.MethodHead {
				w.Write(body)
			}
		})
	}
}

func path(p string) func(*http.Request) bool {
	return func(r *http.Request) bool { return r.URL.Path == p }
}
func pathPrefix(p string) func(*http.Request) bool {
	return func(r *http.Request) bool { return strings.HasPrefix(r.URL.Path, p) }
}
func and(fs ...func(*http.Request) bool) func(*http.Request) bool {
	return func(r *http.Request) bool {
		for _, f := range fs {
			if !f(r) {
				return false
			}
		}
		return true
	}
}
func method(m string) func(*http.Request) bool {
	return func(r *http.Request) bool { return r.Method == m }
}
func hasSuffix(s string) func(*http.Request) bool {
	return func(r *http.Request) bool { return strings.HasSuffix(r.URL.Path, s) }
}
func noAuth(r *http.Request) bool { return r.Header.Get("Authorization") == "" }
func setStatus(code int) func(*http.Request, *int, http.Header, *[]byte) {
	return func(_ *http.Request, s *int, _ http.Header, _ *[]byte) { *s = code }
}
func replaceBody(old, new string) func(*http.Request, *int, http.Header, *[]byte) {
	return func(_ *http.Request, _ *int, _ http.Header, b *[]byte) {
		*b = bytes.ReplaceAll(*b, []byte(old), []byte(new))
	}
}

type mutant struct {
	name string
	// want is the check name (prefix) the verifier must report FAILed.
	want string
	wrap func(http.Handler) http.Handler
	// restart mutates persistence between server runs.
	restart func(*inproc)
}

var liveServerMutants = []mutant{
	{name: "healthz-500", want: "health endpoint", wrap: rewrite(path("/healthz"), setStatus(500))},
	{name: "admin-unauthenticated-open", want: "admin authentication boundary", wrap: rewrite(and(path("/v1/sources"), method("POST"), noAuth), setStatus(200))},
	{name: "catalog-unauthenticated-open", want: "catalog authentication boundary", wrap: rewrite(and(path("/v1/works"), method("GET"), noAuth), setStatus(200))},
	{name: "source-create-200", want: "live local source creation", wrap: rewrite(and(path("/v1/sources"), method("POST"), func(r *http.Request) bool { return !noAuth(r) }), setStatus(200))},
	{name: "scan-accepted-200", want: "live scan accepted", wrap: rewrite(and(path("/v1/scans"), method("POST")), setStatus(200))},
	{name: "atom-content-type-json", want: "OPDS 1.2 acquisition feed", wrap: rewrite(and(path("/opds/1.2/publications"), func(r *http.Request) bool { return r.URL.Query().Get("q") == "" }),
		func(_ *http.Request, _ *int, h http.Header, _ *[]byte) { h.Set("Content-Type", "application/json") })},
	{name: "opds2-no-publications", want: "OPDS 2.0 publication feed", wrap: rewrite(and(path("/opds/2.0/publications"), func(r *http.Request) bool { return r.URL.Query().Get("q") == "" }),
		replaceBody(`"publications"`, `"pubs"`))},
	{name: "opensearch-404", want: "OpenSearch description", wrap: rewrite(path("/opds/1.2/opensearch.xml"), setStatus(404))},
	{name: "range-ignored", want: "single-range content download", wrap: func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/content") {
				r.Header.Del("Range")
			}
			next.ServeHTTP(w, r)
		})
	}},
	{name: "head-wrong-length", want: "HEAD content metadata without body", wrap: rewrite(and(hasSuffix("/content"), method("HEAD")),
		func(_ *http.Request, _ *int, h http.Header, _ *[]byte) { h.Set("Content-Length", "1") })},
	{name: "stale-if-range-honoured", want: "stale If-Range returns complete representation", wrap: func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("If-Range") != "" {
				r.Header.Del("If-Range")
			}
			next.ServeHTTP(w, r)
		})
	}},
	{name: "multi-range-served-as-single", want: "multi-range request uses complete-response fallback", wrap: func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rg := r.Header.Get("Range"); strings.Contains(rg, ",") {
				r.Header.Set("Range", "bytes=0-3")
			}
			next.ServeHTTP(w, r)
		})
	}},
	{name: "attachment-labelled-encoded", want: "attachment remains identity encoded", wrap: rewrite(and(hasSuffix("/content"), method("GET"), func(r *http.Request) bool {
		return r.Header.Get("Range") == "" && strings.Contains(r.Header.Get("Accept-Encoding"), "br")
	}),
		func(_ *http.Request, _ *int, h http.Header, _ *[]byte) { h.Set("Content-Encoding", "identity") })},
	{name: "formats-never-gzip", want: "gzip content negotiation", wrap: func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/formats" {
				r.Header.Set("Accept-Encoding", "identity")
			}
			next.ServeHTTP(w, r)
		})
	}},
	{name: "stream-count-off-by-one", want: "pse:count equals the pages in the archive", wrap: rewrite(pathPrefix("/opds/1.2/editions/"), replaceBody(`count="3"`, `count="4"`))},
	{name: "stream-type-jpeg", want: "the advertised stream type comes from the page bytes", wrap: rewrite(pathPrefix("/opds/1.2/editions/"), replaceBody(`type="image/png"`, `type="image/jpeg"`))},
	{name: "stream-href-wrong-template", want: "the stream href names this file and the page template", wrap: rewrite(pathPrefix("/opds/1.2/editions/"), replaceBody(`/{pageNumber}`, `/{page}`))},
	{name: "stream-lastread-invented", want: "a stream the server cannot position carries no pse:lastRead", wrap: rewrite(pathPrefix("/opds/1.2/editions/"), func(_ *http.Request, _ *int, _ http.Header, b *[]byte) {
		*b = bytes.Replace(*b, []byte(`count="3"`), []byte(`count="3" pse:lastRead="1"`), 1)
	})},
	{name: "page-bytes-corrupted", want: "page 0 is served byte-identical", wrap: rewrite(and(pathPrefix("/opds-pse/stream/"), hasSuffix("/0"), func(r *http.Request) bool { return true }),
		func(_ *http.Request, s *int, _ http.Header, b *[]byte) {
			if *s == 200 && len(*b) > 0 {
				(*b)[len(*b)-1] ^= 0xff
			}
		})},
	{name: "page-etag-wrong", want: "page 1 is tagged with the catalogued file digest", wrap: rewrite(and(pathPrefix("/opds-pse/stream/"), hasSuffix("/1")),
		func(_ *http.Request, _ *int, h http.Header, _ *[]byte) { h.Set("ETag", `"x"`) })},
	{name: "page-past-end-200", want: "page past the last returns the valid range", wrap: rewrite(and(pathPrefix("/opds-pse/stream/"), hasSuffix("/3")), setStatus(200))},
	{name: "non-integer-page-404", want: "a non-integer page number is rejected", wrap: rewrite(and(pathPrefix("/opds-pse/stream/"), hasSuffix("/notanumber")), setStatus(404))},
	{name: "unknown-file-200", want: "an unknown file has no page stream", wrap: rewrite(pathPrefix("/opds-pse/stream/no-such-file/"), setStatus(200))},
	{name: "refused-fixture-page-served", want: "live-server-mixed.cbz refuses page requests", wrap: func(next http.Handler) http.Handler {
		// serve the pageable file's page 0 for a request on any other file's page 0
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(&mixedLeak{ResponseWriter: w, r: r}, r) })
	}},
	{name: "opds2-invents-extension", want: "OPDS 2.0 invents no page-stream extension", wrap: rewrite(and(path("/opds/2.0/publications"), func(r *http.Request) bool { return r.URL.Query().Get("q") == "live-server-pse" }),
		func(_ *http.Request, _ *int, _ http.Header, b *[]byte) { *b = append(*b, []byte(" vaemendis")...) })},
	{name: "kosync-auth-denied", want: "KOSync authentication", wrap: rewrite(path("/users/auth"), setStatus(401))},
	{name: "kosync-progress-altered", want: "KOSync progress read", wrap: rewrite(pathPrefix("/syncs/progress/live-server-document"), replaceBody(`"42"`, `"41"`))},
	{name: "memory-search-empty", want: "pgvector scoped search", wrap: rewrite(path("/v1/ai/memories/search"), func(_ *http.Request, _ *int, _ http.Header, b *[]byte) { *b = []byte(`{"results":[]}`) })},
	{name: "work-delete-200", want: "post-test catalog cleanup", wrap: rewrite(and(pathPrefix("/v1/works/"), method("DELETE")), setStatus(200))},
	{name: "catalog-lost-on-restart", want: "PostgreSQL catalog survives server restart", restart: func(l *inproc) { l.repo = freshRepo() }},
	{name: "redis-lost-on-restart", want: "completed scan job survives server restart", restart: func(l *inproc) { l.redis.FlushAll() }},
}

// mixedLeak makes the response to a page request for a non-pageable fixture look like a served page.
type mixedLeak struct {
	http.ResponseWriter
	r *http.Request
}

func (m *mixedLeak) WriteHeader(code int) {
	if code == http.StatusNotFound && strings.HasPrefix(m.r.URL.Path, "/opds-pse/stream/") && strings.HasSuffix(m.r.URL.Path, "/0") && !strings.Contains(m.r.URL.Path, "no-such-file") {
		code = http.StatusOK
	}
	m.ResponseWriter.WriteHeader(code)
}

func TestLiveServerMutantsAreEachRejected(t *testing.T) {
	for _, m := range liveServerMutants {
		m := m
		t.Run(m.name, func(t *testing.T) {
			l := newInproc(t, m.wrap)
			l.onRestart = m.restart
			cfg := baseConfig(t, l)
			res := RunLiveServer(cfg)
			if res.Status != "FAIL" || res.ExitCode() != 1 {
				t.Fatalf("mutant %s was NOT rejected: status=%s fail=%d", m.name, res.Status, res.Fail)
			}
			got := failedName(res)
			if !strings.HasPrefix(got, m.want) {
				t.Fatalf("mutant %s failed on %q, want a FAIL on %q", m.name, got, m.want)
			}
		})
	}
}
