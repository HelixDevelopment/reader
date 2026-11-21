package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/helixreader/helix/core/pkg/parse/cbz"
	"github.com/helixreader/helix/core/pkg/pse"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/opds"
)

// streamableFormats lists the containers this server can enumerate pages for.
// It is deliberately narrow: a page count advertised in a feed has to come from
// the archive itself, so every format here needs a resolver in resolveStream
// and nothing may be listed that only *looks* pageable.
var streamableFormats = map[string]bool{"cbz": true}

// A stream is refused, never guessed. These are the reasons a file that exists
// in the catalog still gets no pse:count.
var (
	errNoStreamResolver = errors.New("httpapi: format has no page-stream resolver")
	errStreamProtected  = errors.New("httpapi: protected files are not streamed page by page")
	errNoRandomAccess   = errors.New("httpapi: source cannot provide a random-access handle")
	errStreamStale      = errors.New("httpapi: live file no longer matches the catalogued size")
)

// pageSource is an open archive plus the size it was opened at, so directory
// reads and page reads can never disagree with each other or with the catalog.
type pageSource struct {
	at     io.ReaderAt
	closer io.Closer
	size   int64
}

func (p *pageSource) Close() error { return p.closer.Close() }

// openPageSource obtains a random-access handle whose size still matches the
// catalog record. Connectors that cannot seek are not substituted with a
// whole-file download: page streaming exists precisely to avoid that.
func (s *Server) openPageSource(ctx context.Context, f catalog.File) (*pageSource, error) {
	if !streamableFormats[f.Format] {
		return nil, errNoStreamResolver
	}
	if f.Protected {
		return nil, errStreamProtected
	}
	connector, ok := s.connector(f.SourceID)
	if !ok {
		return nil, errNoRandomAccess
	}
	oc, ok := connector.(openContentRevalidator)
	if !ok {
		return nil, errNoRandomAccess
	}
	h, size, _, err := oc.OpenRevalidated(ctx, f.Path)
	if err != nil {
		return nil, err
	}
	at, ok := h.(io.ReaderAt)
	if !ok {
		h.Close()
		return nil, errNoRandomAccess
	}
	if size != f.Size {
		h.Close()
		return nil, errStreamStale
	}
	return &pageSource{at: at, closer: h, size: size}, nil
}

// resolveStream reads the page count and the shared image type out of the
// archive. An archive whose pages mix media types gets no stream: one link
// declares exactly one type for every page the client will fetch, and the
// extension permits only jpeg, gif and png.
func resolveStream(ps *pageSource, fileID string) (opds.PageStream, error) {
	refs, err := cbz.Pages(ps.at, ps.size)
	if err != nil {
		return opds.PageStream{}, err
	}
	// The type comes from each payload's signature, never from a file name.
	mime, err := cbz.StreamMediaType(ps.at, ps.size)
	if err != nil {
		return opds.PageStream{}, err
	}
	if !pse.IsImageType(mime) {
		return opds.PageStream{}, fmt.Errorf("httpapi: %q pages are %s, which OPDS-PSE does not permit", fileID, mime)
	}
	return opds.PageStream{Count: len(refs), Type: mime}, nil
}

// pageStreams resolves the advertisement for every streamable file on one
// catalog page. Failures are silent by design: a feed that cannot describe a
// file omits the link rather than inventing one, and the ordinary acquisition
// link still serves that file whole.
func (s *Server) pageStreams(ctx context.Context, page catalog.Page) map[string]opds.PageStream {
	streams := map[string]opds.PageStream{}
	for _, w := range page.Works {
		for _, e := range w.Editions {
			for _, f := range e.Files {
				ps, err := s.openPageSource(ctx, f)
				if err != nil {
					continue
				}
				stream, err := resolveStream(ps, f.ID)
				ps.Close()
				if err == nil {
					streams[f.ID] = stream
				}
			}
		}
	}
	return streams
}

// editionStreams resolves the advertisement for one edition's files.
func (s *Server) editionStreams(ctx context.Context, e catalog.Edition) map[string]opds.PageStream {
	return s.pageStreams(ctx, catalog.Page{Works: []catalog.Work{{Editions: []catalog.Edition{e}}}})
}

// streamPage serves one page image: GET /opds-pse/stream/:file/:page, the href
// shape opds.PageStreamHREF advertises. Pages are numbered 0 to N-1, so a
// number outside the archive is a 404 that states the valid range.
func (s *Server) streamPage(c *gin.Context) {
	_, _, f, err := s.Catalog.FindFile(c.Request.Context(), c.Param("file"))
	if errors.Is(err, catalog.ErrNotFound) {
		Error(c, http.StatusNotFound, "file not found")
		return
	}
	if err != nil {
		Error(c, http.StatusInternalServerError, "catalog query failed")
		return
	}
	page, err := strconv.Atoi(c.Param("page"))
	if err != nil {
		Error(c, http.StatusBadRequest, "page number must be an integer")
		return
	}
	src, err := s.openPageSource(c.Request.Context(), f)
	if err != nil {
		switch {
		case errors.Is(err, errStreamStale):
			Error(c, http.StatusConflict, "file changed since it was scanned; re-scan before streaming")
		case errors.Is(err, errNoStreamResolver), errors.Is(err, errStreamProtected), errors.Is(err, errNoRandomAccess):
			Error(c, http.StatusNotFound, "file has no page stream")
		default:
			sourceReadFailure(c, err)
		}
		return
	}
	defer src.Close()
	stream, err := resolveStream(src, f.ID)
	if err != nil {
		Error(c, http.StatusNotFound, "file has no page stream")
		return
	}
	if page < 0 || page >= stream.Count {
		Error(c, http.StatusNotFound, fmt.Sprintf("page %d out of range 0..%d", page, stream.Count-1))
		return
	}
	ref, data, err := cbz.Page(src.at, src.size, page)
	if err != nil {
		// The enumeration and the payload read use the same open handle and the
		// same size, so a failure here is a corrupt archive, not a wrong request.
		Error(c, http.StatusBadGateway, "page could not be read from the archive")
		return
	}
	c.Header("Accept-Ranges", "none")
	c.Header("ETag", `"$sha256-`+f.SHA256+`$page-`+strconv.Itoa(ref.Index)+`"`)
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("Vary", "Authorization")
	c.Header("Content-Disposition", contentDisposition(entryName(ref.Name)))
	c.Data(http.StatusOK, stream.Type, data)
}
