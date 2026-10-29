// Package httpapi exposes the catalog, ingestion and OPDS HTTP contracts.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/helixreader/helix/core/formats"
	coreSync "github.com/helixreader/helix/core/pkg/sync"
	"github.com/helixreader/helix/server/internal/aimemory"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/covers"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/kosync"
	"github.com/helixreader/helix/server/internal/opds"
	"github.com/helixreader/helix/server/internal/sources"
	"github.com/helixreader/helix/server/internal/syncstore"
)

type Server struct {
	Config     config.Config
	Catalog    catalog.Repository
	Ingest     *ingest.Service
	Sources    *sources.Registry
	Covers     *covers.Cache
	Progress   syncstore.Store
	KOSync     *kosync.Service
	Memory     aimemory.Store
	mu         sync.RWMutex
	connectors map[string]sources.Connector
	jobs       *jobStore
}

func New(cfg config.Config, repo catalog.Repository, registry *sources.Registry, identifierIngest *ingest.Service) *Server {
	if registry == nil {
		registry = sources.DefaultRegistry()
	}
	if identifierIngest == nil {
		identifierIngest = &ingest.Service{Repository: repo}
	}
	return &Server{Config: cfg, Catalog: repo, Ingest: identifierIngest, Sources: registry, Covers: covers.New(cfg.CoverCacheDir), connectors: map[string]sources.Connector{}, jobs: newJobStore()}
}
func (s *Server) Close() error {
	timeout := s.Config.ShutdownTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.Shutdown(ctx)
}

// Shutdown stops new scans, cancels and drains existing workers before closing
// their connectors. A timeout leaves connectors open so no worker observes a
// premature close; a later Close/Shutdown can finish the drain.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.jobs.shutdown(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out error
	for id, c := range s.connectors {
		if err := c.Close(); err != nil && out == nil {
			out = err
		}
		delete(s.connectors, id)
	}
	return out
}

func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), RequestID(), SecurityHeaders(s.Config.TLSCertFile != "" && s.Config.HTTP3, s.Config.Address), Compression(s.Config.BrotliLevel))
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()}) })
	r.GET("/readyz", func(c *gin.Context) {
		if s.Catalog == nil {
			Error(c, http.StatusServiceUnavailable, "catalog unavailable")
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/v1/formats", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"formats": formats.All()}) })
	r.GET("/v1/protocols", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"protocols": s.Sources.Protocols()}) })
	read := r.Group("/v1")
	read.Use(Basic(s.Config.OPDSUsername, s.Config.OPDSPassword))
	read.GET("/works", s.works)
	read.GET("/works/:id", s.work)
	read.GET("/editions/:id", s.edition)
	read.GET("/covers/:id", s.cover)
	read.HEAD("/covers/:id", s.cover)
	read.GET("/files/:id/content", s.content)
	read.HEAD("/files/:id/content", s.content)
	r.POST("/users/create", s.kosyncRegister)
	r.GET("/users/auth", s.kosyncAuth)
	r.PUT("/syncs/progress", s.kosyncPutProgress)
	r.GET("/syncs/progress/:document", s.kosyncGetProgress)
	admin := r.Group("/v1")
	admin.Use(Bearer(s.Config.AdminToken))
	admin.POST("/sources", s.createSource)
	admin.DELETE("/sources/:id", s.deleteSource)
	admin.POST("/scans", s.startScan)
	admin.GET("/scans", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"jobs": s.jobs.list()}) })
	admin.GET("/scans/:id", s.scan)
	admin.DELETE("/scans/:id", s.cancelScan)
	admin.DELETE("/works/:id", s.deleteWork)
	admin.PUT("/ai/memories", s.putMemory)
	admin.POST("/ai/memories/search", s.searchMemory)
	admin.DELETE("/ai/memories/:scope/:id", s.deleteMemory)
	admin.GET("/sync/progress/:user/:book", s.getProgress)
	admin.PUT("/sync/progress/:user/:book", s.putProgress)
	admin.DELETE("/sync/progress/:user/:book", s.deleteProgress)
	feeds := r.Group("/opds")
	feeds.Use(Basic(s.Config.OPDSUsername, s.Config.OPDSPassword))
	feeds.GET("/1.2", func(c *gin.Context) { data, err := opds.Navigation12(); s.data(c, data, opds.MIME12Navigation, err) })
	feeds.GET("/1.2/opensearch.xml", func(c *gin.Context) {
		data, err := opds.OpenSearch12(s.Config.ExternalURL)
		s.data(c, data, opds.MIMEOpenSearch, err)
	})
	feeds.GET("/1.2/publications", func(c *gin.Context) {
		q := query(c)
		page, err := s.Catalog.Search(c.Request.Context(), q)
		if err != nil {
			Error(c, 500, "catalog query failed")
			return
		}
		data, err := opds.Acquisition12(c.Request.URL.Path, page, q)
		s.data(c, data, opds.MIME12Acquisition, err)
	})
	feeds.GET("/2.0", func(c *gin.Context) { data, err := opds.Navigation20(); s.data(c, data, opds.MIME2Feed, err) })
	feeds.GET("/1.2/editions/:id", func(c *gin.Context) {
		w, e, ok := s.findEdition(c)
		if !ok {
			return
		}
		data, err := opds.Edition12(w, e)
		s.data(c, data, opds.MIME12Entry, err)
	})
	feeds.GET("/2.0/editions/:id", func(c *gin.Context) {
		w, e, ok := s.findEdition(c)
		if !ok {
			return
		}
		data, err := opds.Edition20(w, e)
		s.data(c, data, opds.MIME2Publication, err)
	})
	feeds.GET("/2.0/publications", func(c *gin.Context) {
		q := query(c)
		page, err := s.Catalog.Search(c.Request.Context(), q)
		if err != nil {
			Error(c, 500, "catalog query failed")
			return
		}
		data, err := opds.Acquisition20(c.Request.URL.Path, page, q)
		s.data(c, data, opds.MIME2Feed, err)
	})
	return r
}

const (
	sourceJSONLimit   int64 = 64 << 10
	memoryJSONLimit   int64 = 1 << 20
	progressJSONLimit int64 = 64 << 10
	scanJSONLimit     int64 = 4 << 10
)

func bindJSON(c *gin.Context, limit int64, destination any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	return c.ShouldBindJSON(destination)
}

func (s *Server) data(c *gin.Context, data []byte, mime string, err error) {
	if err != nil {
		Error(c, 500, "feed generation failed")
		return
	}
	c.Data(http.StatusOK, mime, data)
}
func query(c *gin.Context) catalog.Query {
	offset, _ := strconv.Atoi(c.Query("offset"))
	if offset < 0 {
		offset = 0
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return catalog.Query{Text: c.Query("q"), Author: c.Query("author"), State: catalog.IdentificationState(c.Query("state")), Offset: offset, Limit: limit}
}
func (s *Server) works(c *gin.Context) {
	page, err := s.Catalog.Search(c.Request.Context(), query(c))
	if err != nil {
		Error(c, 500, "catalog query failed")
		return
	}
	c.JSON(http.StatusOK, page)
}
func (s *Server) work(c *gin.Context) {
	w, err := s.Catalog.GetWork(c.Request.Context(), c.Param("id"))
	if errors.Is(err, catalog.ErrNotFound) {
		Error(c, 404, "work not found")
		return
	}
	if err != nil {
		Error(c, 500, "catalog query failed")
		return
	}
	c.JSON(http.StatusOK, w)
}
func (s *Server) cover(c *gin.Context) {
	_, e, ok := s.findEdition(c)
	if !ok {
		return
	}
	if e.CoverHREF == "" {
		Error(c, 404, "cover not found")
		return
	}
	cached, err := s.Covers.Get(c.Request.Context(), e.ID, e.CoverHREF)
	if err != nil {
		Error(c, http.StatusBadGateway, "cover retrieval failed")
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Type", cached.MIME)
	c.File(cached.Path)
}
func (s *Server) edition(c *gin.Context) {
	_, e, ok := s.findEdition(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, e)
}
func (s *Server) findEdition(c *gin.Context) (catalog.Work, catalog.Edition, bool) {
	w, e, err := s.Catalog.FindEdition(c.Request.Context(), c.Param("id"))
	if errors.Is(err, catalog.ErrNotFound) {
		Error(c, 404, "edition not found")
		return catalog.Work{}, catalog.Edition{}, false
	}
	if err != nil {
		Error(c, 500, "catalog query failed")
		return catalog.Work{}, catalog.Edition{}, false
	}
	return w, e, true
}
func (s *Server) deleteWork(c *gin.Context) {
	err := s.Catalog.DeleteWork(c.Request.Context(), c.Param("id"))
	if errors.Is(err, catalog.ErrNotFound) {
		Error(c, 404, "work not found")
		return
	}
	if err != nil {
		Error(c, 500, "catalog delete failed")
		return
	}
	c.Status(http.StatusNoContent)
}

type sourceResponse struct {
	ID           string               `json:"id"`
	Protocol     string               `json:"protocol"`
	Capabilities sources.Capabilities `json:"capabilities"`
}

func validSourceID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || (i > 0 && (c == '-' || c == '_' || c == '.')) {
			continue
		}
		return false
	}
	return true
}

func (s *Server) createSource(c *gin.Context) {
	var cfg sources.Config
	if err := bindJSON(c, sourceJSONLimit, &cfg); err != nil {
		Error(c, 400, "invalid source configuration")
		return
	}
	id := strings.TrimSpace(cfg.ID)
	if cfg.ID != "" && !validSourceID(id) {
		Error(c, http.StatusBadRequest, "invalid source id")
		return
	}
	if id != "" {
		s.mu.RLock()
		_, exists := s.connectors[id]
		s.mu.RUnlock()
		if exists {
			Error(c, http.StatusConflict, "source id is already connected")
			return
		}
	}
	connector, err := s.Sources.New(cfg)
	if err != nil {
		Error(c, 400, "source configuration rejected")
		return
	}
	if id == "" {
		id, err = randomID("source")
		if err != nil {
			connector.Close()
			Error(c, 500, "source id generation failed")
			return
		}
	}
	s.mu.Lock()
	if _, exists := s.connectors[id]; exists {
		s.mu.Unlock()
		_ = connector.Close()
		Error(c, http.StatusConflict, "source id is already connected")
		return
	}
	s.connectors[id] = connector
	s.mu.Unlock()
	c.JSON(http.StatusCreated, sourceResponse{ID: id, Protocol: strings.ToLower(cfg.Protocol), Capabilities: connector.Capabilities()})
}
func (s *Server) connector(id string) (sources.Connector, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.connectors[id]
	return c, ok
}
func (s *Server) deleteSource(c *gin.Context) {
	id := c.Param("id")
	s.mu.Lock()
	connector, ok := s.connectors[id]
	if ok && s.jobs.sourceBusy(id) {
		s.mu.Unlock()
		Error(c, http.StatusConflict, "source has unfinished scans")
		return
	}
	if ok {
		delete(s.connectors, id)
	}
	s.mu.Unlock()
	if !ok {
		Error(c, 404, "source not found")
		return
	}
	if err := connector.Close(); err != nil {
		Error(c, 500, "source close failed")
		return
	}
	c.Status(http.StatusNoContent)
}

type memoryRequest struct {
	Record    aimemory.Record `json:"record"`
	Embedding []float32       `json:"embedding"`
}
type memorySearchRequest struct {
	Scope     string    `json:"scope"`
	Embedding []float32 `json:"embedding"`
	Limit     int       `json:"limit"`
}

func (s *Server) putMemory(c *gin.Context) {
	if s.Memory == nil {
		Error(c, http.StatusServiceUnavailable, "AI memory is not configured")
		return
	}
	var request memoryRequest
	if err := bindJSON(c, memoryJSONLimit, &request); err != nil {
		Error(c, 400, "invalid AI memory")
		return
	}
	record, err := s.Memory.Put(c.Request.Context(), request.Record, request.Embedding)
	if err != nil {
		Error(c, 400, "AI memory rejected")
		return
	}
	c.JSON(http.StatusOK, record)
}
func (s *Server) searchMemory(c *gin.Context) {
	if s.Memory == nil {
		Error(c, http.StatusServiceUnavailable, "AI memory is not configured")
		return
	}
	var request memorySearchRequest
	if err := bindJSON(c, memoryJSONLimit, &request); err != nil {
		Error(c, 400, "invalid vector search")
		return
	}
	results, err := s.Memory.Search(c.Request.Context(), request.Scope, request.Embedding, request.Limit)
	if err != nil {
		Error(c, 400, "vector search rejected")
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}
func (s *Server) deleteMemory(c *gin.Context) {
	if s.Memory == nil {
		Error(c, http.StatusServiceUnavailable, "AI memory is not configured")
		return
	}
	if err := s.Memory.Delete(c.Request.Context(), c.Param("scope"), c.Param("id")); err != nil {
		Error(c, 500, "AI memory delete failed")
		return
	}
	c.Status(http.StatusNoContent)
}
func (s *Server) getProgress(c *gin.Context) {
	if s.Progress == nil {
		Error(c, http.StatusServiceUnavailable, "progress store is not configured")
		return
	}
	record, err := s.Progress.Get(c.Request.Context(), c.Param("user"), c.Param("book"))
	if errors.Is(err, syncstore.ErrNotFound) {
		Error(c, 404, "progress not found")
		return
	}
	if err != nil {
		Error(c, 500, "progress read failed")
		return
	}
	c.JSON(http.StatusOK, record)
}
func (s *Server) putProgress(c *gin.Context) {
	if s.Progress == nil {
		Error(c, http.StatusServiceUnavailable, "progress store is not configured")
		return
	}
	var record coreSync.ProgressRecord
	if err := bindJSON(c, progressJSONLimit, &record); err != nil {
		Error(c, 400, "invalid progress record")
		return
	}
	record.BookID = c.Param("book")
	record, err := s.Progress.Put(c.Request.Context(), c.Param("user"), record)
	if err != nil {
		Error(c, 400, "progress record rejected")
		return
	}
	c.JSON(http.StatusOK, record)
}
func (s *Server) deleteProgress(c *gin.Context) {
	if s.Progress == nil {
		Error(c, http.StatusServiceUnavailable, "progress store is not configured")
		return
	}
	if err := s.Progress.Delete(c.Request.Context(), c.Param("user"), c.Param("book")); err != nil {
		Error(c, 500, "progress delete failed")
		return
	}
	c.Status(http.StatusNoContent)
}

type scanRequest struct {
	SourceID string `json:"source_id"`
	Prefix   string `json:"prefix"`
}

func (s *Server) startScan(c *gin.Context) {
	var req scanRequest
	if err := bindJSON(c, scanJSONLimit, &req); err != nil {
		Error(c, 400, "invalid scan request")
		return
	}
	// Keep the connector registered until the job is visible to deleteSource.
	// Both paths use lock order Server.mu -> jobStore.mu, so deletion cannot
	// close a connector in the gap between lookup and scan admission.
	s.mu.RLock()
	connector, ok := s.connectors[req.SourceID]
	if !ok {
		s.mu.RUnlock()
		Error(c, 404, "source not found")
		return
	}
	job, ctx, err := s.jobs.create(req.SourceID)
	s.mu.RUnlock()
	if errors.Is(err, errScanStoreClosed) {
		Error(c, http.StatusServiceUnavailable, "scan service is shutting down")
		return
	}
	if errors.Is(err, errScanQueueFull) {
		Error(c, http.StatusTooManyRequests, "scan queue is full")
		return
	}
	if err != nil {
		Error(c, http.StatusInternalServerError, "scan id generation failed")
		return
	}
	go s.jobs.run(ctx, job.ID, func(ctx context.Context) (ingest.Report, error) {
		return s.Ingest.ScanSource(ctx, req.SourceID, connector, req.Prefix)
	})
	c.JSON(http.StatusAccepted, job)
}
func (s *Server) scan(c *gin.Context) {
	j, ok := s.jobs.get(c.Param("id"))
	if !ok {
		Error(c, 404, "scan not found")
		return
	}
	c.JSON(http.StatusOK, j)
}
func (s *Server) cancelScan(c *gin.Context) {
	j, ok := s.jobs.cancelJob(c.Param("id"))
	if !ok {
		Error(c, 404, "scan not found")
		return
	}
	c.JSON(http.StatusAccepted, j)
}

// contentRevalidator is optionally implemented by connectors that can report
// current live metadata. Catalogued sizes and hashes can go stale when bytes
// change after a scan; only matching live size and modification time retain
// the strong scan-time ETag and range eligibility.
type contentRevalidator interface {
	Revalidate(ctx context.Context, path string) (int64, time.Time, error)
}

type openContentRevalidator interface {
	OpenRevalidated(ctx context.Context, path string) (io.ReadSeekCloser, int64, time.Time, error)
}

func sourceReadFailure(c *gin.Context, err error) {
	if errors.Is(err, sources.ErrNotFound) {
		Error(c, http.StatusNotFound, "source file not found")
		return
	}
	Error(c, http.StatusBadGateway, "source read failed")
}

func sameModification(a, b time.Time) bool {
	// PostgreSQL timestamptz values have microsecond precision. Normalize both
	// sides so durable-catalog round trips do not invalidate otherwise equal
	// filesystem modification timestamps.
	return a.UTC().Truncate(time.Microsecond).Equal(b.UTC().Truncate(time.Microsecond))
}

func (s *Server) content(c *gin.Context) {
	_, _, f, err := s.Catalog.FindFile(c.Request.Context(), c.Param("id"))
	if errors.Is(err, catalog.ErrNotFound) {
		Error(c, 404, "file not found")
		return
	}
	if err != nil {
		Error(c, 500, "catalog query failed")
		return
	}
	connector, ok := s.connector(f.SourceID)
	if !ok {
		Error(c, http.StatusConflict, "file source is not connected")
		return
	}
	etag := `"sha256-` + f.SHA256 + `"`
	c.Header("Content-Disposition", contentDisposition(entryName(f.Path)))
	// Revalidate live metadata when possible. A strong hash from scan time must
	// not label changed bytes, and an old If-Range validator must never splice
	// old and new representations. Changed or unverifiable files fall back to a
	// full 200 without ETag; unknown sizes also omit Content-Length.
	size, sizeKnown, current := int64(-1), false, false
	head := c.Request.Method == http.MethodHead
	var opened io.ReadSeekCloser
	var modified time.Time
	if !head {
		if oc, ok := connector.(openContentRevalidator); ok {
			opened, size, modified, err = oc.OpenRevalidated(c.Request.Context(), f.Path)
			if err != nil {
				sourceReadFailure(c, err)
				return
			}
			defer opened.Close()
		}
	}
	if opened == nil {
		if rv, ok := connector.(contentRevalidator); ok {
			size, modified, err = rv.Revalidate(c.Request.Context(), f.Path)
			if err != nil {
				sourceReadFailure(c, err)
				return
			}
		}
	}
	if size >= 0 {
		sizeKnown = true
		current = size == f.Size && sameModification(modified, f.ModifiedAt)
	} else {
		size = -1
	}
	if current {
		c.Header("Accept-Ranges", "bytes")
		c.Header("ETag", etag)
	}
	rangeHeader := c.GetHeader("Range")
	// Multi-range requests are not supported; serve a full 200 instead of an
	// invalid 416. A mismatched If-Range validator also forces a full 200.
	serveRange := current && sizeKnown && rangeHeader != "" &&
		!strings.Contains(rangeHeader, ",") &&
		(c.GetHeader("If-Range") == "" || c.GetHeader("If-Range") == etag)
	if serveRange {
		start, length, err := singleRange(rangeHeader, size)
		if err != nil {
			c.Header("Content-Range", fmt.Sprintf("bytes */%d", size))
			Error(c, http.StatusRequestedRangeNotSatisfiable, "invalid byte range")
			return
		}
		var rc sources.RangeConnector
		if opened == nil {
			var ok bool
			rc, ok = connector.(sources.RangeConnector)
			if !ok {
				Error(c, http.StatusNotImplemented, "source does not support range reads")
				return
			}
		}
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, start+length-1, size))
		c.Header("Content-Length", strconv.FormatInt(length, 10))
		if head {
			c.Header("Content-Type", f.MIMEType)
			c.Status(http.StatusPartialContent)
			return
		}
		if opened != nil {
			if _, err := opened.Seek(start, io.SeekStart); err != nil {
				sourceReadFailure(c, err)
				return
			}
			c.DataFromReader(http.StatusPartialContent, length, f.MIMEType, io.LimitReader(opened, length), nil)
			return
		}
		reader, err := rc.OpenRange(c.Request.Context(), f.Path, start, length)
		if err != nil {
			sourceReadFailure(c, err)
			return
		}
		defer reader.Close()
		c.DataFromReader(http.StatusPartialContent, length, f.MIMEType, reader, nil)
		return
	}
	// Full response. Unknown sizes stay identity-chunked with no
	// Content-Length so the client never trusts a stale byte count.
	if sizeKnown {
		c.Header("Content-Length", strconv.FormatInt(size, 10))
	}
	if head {
		c.Header("Content-Type", f.MIMEType)
		c.Status(http.StatusOK)
		return
	}
	var reader io.Reader = opened
	if opened == nil {
		rc, err := connector.Open(c.Request.Context(), f.Path)
		if err != nil {
			sourceReadFailure(c, err)
			return
		}
		defer rc.Close()
		reader = rc
	}
	if sizeKnown {
		c.DataFromReader(http.StatusOK, size, f.MIMEType, reader, nil)
		return
	}
	// Unknown size: commit and flush headers before streaming so net/http
	// cannot synthesize a Content-Length from buffered bytes; the body is
	// chunked instead.
	c.Header("Content-Type", f.MIMEType)
	c.Status(http.StatusOK)
	c.Writer.Flush()
	if _, err := io.Copy(c.Writer, reader); err != nil {
		_ = c.Error(err)
	}
}
func entryName(path string) string {
	if u, err := url.Parse(path); err == nil && u.IsAbs() && strings.HasPrefix(u.Fragment, "helix-x-") {
		u.Fragment = ""
		path = u.String()
	}
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
func contentDisposition(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '"' || r == '\\' || r == '\r' || r == '\n' {
			return '_'
		}
		return r
	}, name)
	return `attachment; filename="` + name + `"`
}
func singleRange(value string, size int64) (start, length int64, err error) {
	if size <= 0 || !strings.HasPrefix(value, "bytes=") || strings.Contains(value, ",") {
		return 0, 0, fmt.Errorf("invalid")
	}
	parts := strings.Split(strings.TrimPrefix(value, "bytes="), "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid")
	}
	if parts[0] == "" {
		n, e := strconv.ParseInt(parts[1], 10, 64)
		if e != nil || n <= 0 {
			return 0, 0, fmt.Errorf("invalid")
		}
		if n > size {
			n = size
		}
		return size - n, n, nil
	}
	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 || start >= size {
		return 0, 0, fmt.Errorf("invalid")
	}
	end := size - 1
	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return 0, 0, fmt.Errorf("invalid")
		}
		if end >= size {
			end = size - 1
		}
	}
	return start, end - start + 1, nil
}
