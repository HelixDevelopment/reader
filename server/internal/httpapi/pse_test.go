package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/helixreader/helix/core/formats"
	"github.com/helixreader/helix/core/pkg/pse"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/opds"
	"github.com/helixreader/helix/server/internal/sources"
)

var (
	pngPage  = append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, []byte("png-page")...)
	jpgPage  = append([]byte{0xff, 0xd8, 0xff, 0xe0}, []byte("jpg-page")...)
	gifPage  = append([]byte("GIF89a"), []byte("gif-pg")...)
	webpPage = append([]byte("RIFF\x08\x00\x00\x00WEBP"), []byte("vp")...)
)

// writeArchive lays down a comic and its catalog record, with the size and
// hash the scanner would have recorded.
func writeArchive(t *testing.T, root string, repo *catalog.Memory, fileID, name string, pages ...[]byte) catalog.File {
	t.Helper()
	path := filepath.Join(root, name)
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for i, p := range pages {
		w, err := zw.Create(entryNameFor(p, i))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	format := name[strings.LastIndex(name, ".")+1:]
	// The media type comes from the same registry a scan consults, so this
	// fixture cannot record a type the server would never have stored.
	desc, known := formats.Detect(name)
	if !known || len(desc.MIMETypes) == 0 {
		t.Fatalf("fixture file %s is not a registered format", name)
	}
	f := catalog.File{ID: fileID, SourceID: "source-test", Path: name, Format: format, MIMEType: desc.MIMETypes[0],
		Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:]), ModifiedAt: info.ModTime()}
	work := catalog.Work{ID: "w-" + fileID, CanonicalTitle: name, CreatedAt: info.ModTime(), UpdatedAt: info.ModTime(),
		Editions: []catalog.Edition{{ID: "e-" + fileID, Title: name, State: catalog.Identified, UpdatedAt: info.ModTime(), Files: []catalog.File{f}}}}
	if err := repo.UpsertWork(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	return f
}

// entryNameFor names an archive entry after the payload it holds, so an
// honest archive stays honest and the deliberate liar case can carry a name
// that contradicts its bytes.
func entryNameFor(page []byte, i int) string {
	base := "page-" + strconv.Itoa(i)
	switch {
	case bytes.HasPrefix(page, pngPage[:8]):
		return base + ".png"
	case bytes.HasPrefix(page, jpgPage[:4]):
		return base + ".jpg"
	case bytes.HasPrefix(page, gifPage[:6]):
		return base + ".gif"
	case bytes.HasPrefix(page, []byte("RIFF")):
		return base + ".webp"
	default:
		return base + ".png"
	}
}

func streamFeed(t *testing.T, handler http.Handler, path string) []byte {
	t.Helper()
	w := request(t, handler, http.MethodGet, path, "", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s: %d %s", path, w.Code, w.Body.String())
	}
	return w.Body.Bytes()
}

// advertisedStream returns the single PSE link the feed offers for fileID,
// parsed by core/pkg/pse rather than string-matched, so the feed is checked
// against the same rules a client applies.
func advertisedStream(t *testing.T, feed []byte, fileID string) (pse.Link, bool) {
	t.Helper()
	links, err := pse.ParseAcquisitionLinks(feed)
	if err != nil {
		if strings.Contains(err.Error(), "no acquisition link") {
			return pse.Link{}, false
		}
		t.Fatalf("feed XML rejected by the PSE parser: %v\n%s", err, feed)
	}
	for _, l := range links {
		if strings.Contains(l.Href, "/"+fileID+"/") {
			return l, true
		}
	}
	return pse.Link{}, false
}

func TestPageStreamFeedAndServe(t *testing.T) {
	s, repo, root := testServer(t)
	defer s.Close()
	writeArchive(t, root, repo, "comic", "comic.cbz", pngPage, append(append([]byte{}, pngPage...), '2'), append(append([]byte{}, pngPage...), '3'))
	r := s.Router()

	feed := streamFeed(t, r, "/opds/1.2/publications")
	link, ok := advertisedStream(t, feed, "comic")
	if !ok {
		t.Fatalf("no stream link for comic.cbz:\n%s", feed)
	}
	if link.Count != 3 || link.Type != "image/png" {
		t.Fatalf("link = %+v, want 3 pages of image/png", link)
	}
	if want := opds.PageStreamHREF("comic"); link.Href != want {
		t.Fatalf("href = %q, want the href this server registers: %q", link.Href, want)
	}

	for page := 0; page < link.Count; page++ {
		href, err := link.PageURL(page, 0)
		if err != nil {
			t.Fatalf("client template for page %d: %v", page, err)
		}
		w := request(t, r, http.MethodGet, href, "", nil, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("page %d: %d %s", page, w.Code, w.Body.String())
		}
		if got := w.Header().Get("Content-Type"); got != "image/png" {
			t.Errorf("page %d Content-Type = %q, want image/png", page, got)
		}
		if !strings.HasPrefix(w.Body.String(), string(pngPage)) {
			t.Errorf("page %d body = %q, want the stored page bytes", page, w.Body.String())
		}
		if etag := w.Header().Get("ETag"); !strings.Contains(etag, `$page-`+strconv.Itoa(page)) {
			t.Errorf("page %d ETag = %q, want it to bind the page index", page, etag)
		}
	}

	// A client following the template cannot even ask for page 3: the parser
	// refuses it. These URLs are built by hand to check the server side of the
	// same rule for a client that ignores the advertised count.
	w := request(t, r, http.MethodGet, opds.PageStreamPrefix+"comic/3", "", nil, nil)
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "page 3 out of range 0..2") {
		t.Fatalf("beyond-count page: %d %q", w.Code, w.Body.String())
	}
	// A negative page is well-formed syntax for an impossible page: like the
	// client-side template, the server answers 404 with the valid range.
	w = request(t, r, http.MethodGet, opds.PageStreamPrefix+"comic/-1", "", nil, nil)
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "page -1 out of range 0..2") {
		t.Fatalf("negative page: %d %q", w.Code, w.Body.String())
	}
	w = request(t, r, http.MethodGet, "/opds-pse/stream/comic/notanumber", "", nil, nil)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "must be an integer") {
		t.Fatalf("non-numeric page: %d %q", w.Code, w.Body.String())
	}
	w = request(t, r, http.MethodGet, "/opds-pse/stream/nosuchfile/0", "", nil, nil)
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "file not found") {
		t.Fatalf("unknown file: %d %q", w.Code, w.Body.String())
	}
}

// A stream link declares one type for every page, so an archive that cannot
// answer "which type?" with one value gets no link at all.
func TestPageStreamRefusesArchivesWithoutOneHonestType(t *testing.T) {
	s, repo, root := testServer(t)
	defer s.Close()
	writeArchive(t, root, repo, "mixed", "mixed.cbz", pngPage, jpgPage)
	writeArchive(t, root, repo, "webp", "webp.cbz", webpPage, webpPage)
	writeArchive(t, root, repo, "liar", "liar.cbz", []byte("this is not an image, only named like one"))
	r := s.Router()
	feed := streamFeed(t, r, "/opds/1.2/publications")
	for _, id := range []string{"mixed", "webp", "liar"} {
		if link, ok := advertisedStream(t, feed, id); ok {
			t.Errorf("%s was advertised as streamable: %+v", id, link)
		}
		w := request(t, r, http.MethodGet, "/opds-pse/stream/"+id+"/0", "", nil, nil)
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "no page stream") {
			t.Errorf("%s page request: %d %q, want 404 no page stream", id, w.Code, w.Body.String())
		}
		// The whole file must still be deliverable.
		if w := request(t, r, http.MethodGet, "/v1/files/"+id+"/content", "", nil, nil); w.Code != http.StatusOK {
			t.Errorf("%s whole-file fetch: %d", id, w.Code)
		}
	}
}

func TestPageStreamRefusesProtectedAndUnknownFormats(t *testing.T) {
	s, repo, root := testServer(t)
	defer s.Close()
	// Named .cbr but containing the ZIP bytes this server can read: the point is
	// that the recorded format gates advertisement, not whether the payload
	// happens to be parseable. RAR page extraction is not implemented, so a
	// cbr record gets no stream even when its bytes would allow one.
	writeArchive(t, root, repo, "rar", "comic.cbr", pngPage, pngPage)
	f := writeArchive(t, root, repo, "locked", "locked.cbz", pngPage, pngPage)
	f.Protected = true
	f.DRMScheme = "lcp"
	work, _, _, err := repo.FindFile(context.Background(), "locked")
	if err != nil {
		t.Fatal(err)
	}
	work.Editions[0].Files[0] = f
	if err := repo.UpsertWork(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	r := s.Router()
	feed := streamFeed(t, r, "/opds/1.2/publications")
	if link, ok := advertisedStream(t, feed, "locked"); ok {
		t.Errorf("a protected archive was advertised for plaintext page streaming: %+v", link)
	}
	w := request(t, r, http.MethodGet, "/opds-pse/stream/locked/0", "", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("protected page request: %d %q", w.Code, w.Body.String())
	}
	// core/formats marks cbr as Stream-capable, meaning the whole file can be
	// delivered progressively. That is not a claim that pages are extractable,
	// and the feed must not confuse the two.
	if link, ok := advertisedStream(t, feed, "rar"); ok {
		t.Errorf("a cbr record was advertised for page streaming although RAR extraction is unimplemented: %+v", link)
	}
	if w := request(t, r, http.MethodGet, "/opds-pse/stream/rar/0", "", nil, nil); w.Code != http.StatusNotFound {
		t.Errorf("cbr page request: %d %q, want 404", w.Code, w.Body.String())
	}
	if w := request(t, r, http.MethodGet, "/v1/files/rar/content", "", nil, nil); w.Code != http.StatusOK {
		t.Errorf("cbr whole-file fetch: %d, want the file itself still deliverable", w.Code)
	}
	// f1 is the plain text file from the base fixture: never pageable.
	if link, ok := advertisedStream(t, feed, "f1"); ok {
		t.Errorf("a text file was advertised as streamable: %+v", link)
	}
	if w := request(t, r, http.MethodGet, "/opds-pse/stream/f1/0", "", nil, nil); w.Code != http.StatusNotFound {
		t.Fatalf("text file page request: %d %q", w.Code, w.Body.String())
	}
}

// Pages come from a random-access handle whose size must still match what the
// catalog recorded, or the advertised count and ETag would describe different
// bytes than the ones served.
func TestPageStreamRefusesFileChangedAfterScan(t *testing.T) {
	s, repo, root := testServer(t)
	defer s.Close()
	f := writeArchive(t, root, repo, "grew", "grew.cbz", pngPage, pngPage)
	work, _, _, err := repo.FindFile(context.Background(), "grew")
	if err != nil {
		t.Fatal(err)
	}
	stale := f
	stale.Size = f.Size + 1000 // the scan saw a different file than the source has now
	work.Editions[0].Files[0] = stale
	if err := repo.UpsertWork(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	r := s.Router()
	if link, ok := advertisedStream(t, streamFeed(t, r, "/opds/1.2/publications"), "grew"); ok {
		t.Errorf("a stale catalog record was advertised as a stream: %+v", link)
	}
	w := request(t, r, http.MethodGet, "/opds-pse/stream/grew/0", "", nil, nil)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "changed since it was scanned") {
		t.Fatalf("stale page request: %d %q, want 409", w.Code, w.Body.String())
	}
}

func TestPageStreamRequiresFeedCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	repo := catalog.NewMemory()
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{OPDSUsername: "reader", OPDSPassword: "secret"}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	s.connectors["source-test"] = connector
	// A genuinely streamable archive, so a 401 can only come from the
	// credential check and not from the file being uninteresting.
	writeArchive(t, root, repo, "comic", "comic.cbz", pngPage, pngPage)
	r := s.Router()
	for _, path := range []string{"/opds/1.2/publications", "/opds-pse/stream/comic/0"} {
		if w := request(t, r, http.MethodGet, path, "", nil, nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s unauthenticated: %d, want 401", path, w.Code)
		}
	}
	// With credentials the same pair works, and the stream really is served.
	cred := map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("reader:secret"))}
	w := request(t, r, http.MethodGet, "/opds-pse/stream/comic/0", "", nil, cred)
	if w.Code != http.StatusOK || !bytes.HasPrefix(w.Body.Bytes(), pngPage) {
		t.Fatalf("authenticated page: %d %q", w.Code, w.Body.String())
	}
	if w := request(t, r, http.MethodGet, "/opds-pse/stream/comic/0", "", nil,
		map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("reader:wrong"))}); w.Code != http.StatusUnauthorized {
		t.Errorf("wrong password: %d, want 401", w.Code)
	}
}
