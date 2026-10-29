package httpapi

import (
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

type compressionWriter struct {
	gin.ResponseWriter
	writer      io.Writer
	closer      io.Closer
	encoding    string
	wroteHeader bool
}

func (w *compressionWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	h := w.Header()
	contentType := h.Get("Content-Type")
	// Downloads (attachment disposition) and media that is already compressed
	// must pass through as identity: recompressing binary content would
	// corrupt advertised lengths and waste CPU for no transfer gain.
	incompressible := strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "video/") ||
		strings.HasPrefix(contentType, "audio/") ||
		strings.HasPrefix(h.Get("Content-Disposition"), "attachment")
	if code == http.StatusNoContent || code == http.StatusNotModified || h.Get("Content-Range") != "" || h.Get("Content-Encoding") != "" || strings.Contains(contentType, "text/event-stream") || incompressible {
		w.encoding = "identity"
		w.writer = w.ResponseWriter
		// The negotiated compressor has not emitted anything. Closing it here
		// would append an empty compressed stream after the identity body,
		// poisoning the next response on a persistent HTTP/1.1 connection.
		w.closer = nil
		w.ResponseWriter.WriteHeader(code)
		return
	}
	h.Del("Content-Length")
	h.Set("Content-Encoding", w.encoding)
	w.ResponseWriter.WriteHeader(code)
}
func (w *compressionWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(w.ResponseWriter.Status())
	}
	// Gin 1.12's Data renderer sets Content-Length after c.Status has called
	// WriteHeader on this wrapper but before the underlying network header is
	// committed. Remove that uncompressed length immediately before the first
	// compressed write as well as in WriteHeader.
	if w.encoding != "identity" {
		w.Header().Del("Content-Length")
	}
	return w.writer.Write(p)
}
func (w *compressionWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(w.ResponseWriter.Status())
	}
	if f, ok := w.writer.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
	w.ResponseWriter.Flush()
}

func parseEncodings(header string) map[string]float64 {
	out := map[string]float64{}
	for _, part := range strings.Split(header, ",") {
		bits := strings.Split(strings.TrimSpace(part), ";")
		name := strings.ToLower(strings.TrimSpace(bits[0]))
		if name == "" {
			continue
		}
		q := 1.0
		for _, p := range bits[1:] {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "q=") {
				v, err := strconv.ParseFloat(strings.TrimPrefix(p, "q="), 64)
				if err != nil || v < 0 || v > 1 {
					q = 0
				} else {
					q = v
				}
			}
		}
		out[name] = q
	}
	return out
}
func accepted(q map[string]float64, name string) float64 {
	if v, ok := q[name]; ok {
		return v
	}
	return q["*"]
}

// Compression is response-only. It never decodes request bodies and honors
// identity, q=0, Content-Range and pre-encoded responses.
func Compression(level int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Vary", "Accept-Encoding")
		q := parseEncodings(c.GetHeader("Accept-Encoding"))
		encoding := "identity"
		if accepted(q, "br") > 0 && accepted(q, "br") >= accepted(q, "gzip") {
			encoding = "br"
		} else if accepted(q, "gzip") > 0 {
			encoding = "gzip"
		}
		identityQ := 1.0
		if v, ok := q["identity"]; ok {
			identityQ = v
		} else if v, ok := q["*"]; ok && v == 0 {
			identityQ = 0
		}
		if identityQ == 0 && encoding == "identity" {
			c.AbortWithStatus(http.StatusNotAcceptable)
			return
		}
		if encoding == "identity" {
			c.Next()
			return
		}
		var w io.Writer
		var closer io.Closer
		switch encoding {
		case "br":
			bw := brotli.NewWriterLevel(c.Writer, level)
			w, closer = bw, bw
		case "gzip":
			gw := gzip.NewWriter(c.Writer)
			w, closer = gw, gw
		}
		cw := &compressionWriter{ResponseWriter: c.Writer, writer: w, closer: closer, encoding: encoding}
		c.Writer = cw
		c.Next()
		if !cw.wroteHeader {
			cw.WriteHeader(c.Writer.Status())
		}
		if cw.closer != nil {
			if err := cw.closer.Close(); err != nil {
				_ = c.Error(err)
			}
		}
	}
}
