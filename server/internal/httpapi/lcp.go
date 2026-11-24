package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/helixreader/helix/core/pkg/lcp"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/ingest"
)

const maxLicenseServeBytes = 1 << 20

// lcpArtifactName is the download name of a fulfilled file: the license's name
// with its .lcpl extension replaced by .epub.
func lcpArtifactName(path string) string {
	name := entryName(path)
	if i := strings.LastIndexByte(name, '.'); i > 0 && strings.EqualFold(name[i:], ".lcpl") {
		name = name[:i]
	}
	return name + ".epub"
}

// lcpContent serves a fulfilled, still LCP-encrypted EPUB from the server's
// cache. The cache is keyed by the artifact's SHA-256, which the catalog holds,
// so the file cannot be mistaken for another license's artifact. Nothing here
// decrypts: the bytes go out exactly as fulfillment stored them.
func (s *Server) lcpContent(c *gin.Context, f catalog.File) {
	if s.Ingest == nil || s.Ingest.LCP == nil {
		Error(c, http.StatusConflict, "LCP fulfillment cache is not configured")
		return
	}
	fh, info, err := s.Ingest.LCP.OpenVerified(f.SHA256, f.Size)
	switch {
	case errors.Is(err, ingest.ErrLCPArtifactMissing):
		// Present in the catalog, absent from the cache: a rescan re-fulfills it.
		Error(c, http.StatusConflict, "fulfilled artifact is missing from the cache; rescan the source")
		return
	case errors.Is(err, ingest.ErrLCPArtifactMismatch):
		Error(c, http.StatusConflict, "fulfilled artifact does not match the catalog; rescan the source")
		return
	case err != nil:
		Error(c, http.StatusInternalServerError, "fulfilled artifact is unreadable")
		return
	}
	defer fh.Close()
	c.Header("ETag", `"sha256-`+f.SHA256+`"`)
	c.Header("Content-Disposition", contentDisposition(lcpArtifactName(f.Path)))
	c.Header("Content-Type", f.MIMEType)
	// ServeContent provides Range, If-Range, If-None-Match and HEAD handling.
	http.ServeContent(c.Writer, c.Request, f.SHA256+".epub", info.ModTime(), fh)
}

// lcpLicense serves the License Document embedded in the fulfilled artifact
// this file is served as, never the live source file: a source .lcpl replaced
// after fulfillment must not be paired with the older artifact. The artifact is
// hash-verified first. A License Document carries no passphrase: only a hint, a
// hash-derived key check and the content key encrypted under the user key.
func (s *Server) lcpLicense(c *gin.Context) {
	_, _, f, err := s.Catalog.FindFile(c.Request.Context(), c.Param("id"))
	if errors.Is(err, catalog.ErrNotFound) || (err == nil && !ingest.IsFulfilledLCP(f)) {
		Error(c, http.StatusNotFound, "license not found")
		return
	}
	if err != nil {
		Error(c, http.StatusInternalServerError, "catalog query failed")
		return
	}
	if s.Ingest == nil || s.Ingest.LCP == nil {
		Error(c, http.StatusConflict, "LCP fulfillment cache is not configured")
		return
	}
	fh, info, err := s.Ingest.LCP.OpenVerified(f.SHA256, f.Size)
	if err != nil {
		Error(c, http.StatusConflict, "fulfilled artifact is missing or does not match the catalog; rescan the source")
		return
	}
	defer fh.Close()
	ctr, err := lcp.OpenContainer(fh, info.Size())
	if err != nil || ctr.License == nil || len(ctr.License.Raw) == 0 {
		Error(c, http.StatusConflict, "fulfilled artifact carries no readable license; rescan the source")
		return
	}
	if len(ctr.License.Raw) > maxLicenseServeBytes {
		Error(c, http.StatusBadGateway, "license document exceeds the size limit")
		return
	}
	c.Header("Content-Disposition", contentDisposition(entryName(f.Path)))
	c.Data(http.StatusOK, ingest.LCPLicenseMediaType, ctr.License.Raw)
}
