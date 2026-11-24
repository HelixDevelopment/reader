package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// HRD-cronet: the access log is the SERVER-SIDE evidence of the negotiated protocol and the
// response encoding (HTTP/3 clients are proven by what the server saw, not by the client's claim).
func TestAccessLogRecordsProtocolAndEncoding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	r := gin.New()
	r.Use(AccessLog(logger), Compression(4))
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, strings.Repeat("a", 2048)) })
	req := httptest.NewRequest(http.MethodGet, "/x?token=SECRET", nil)
	req.Header.Set("Accept-Encoding", "br")
	req.Header.Set("Authorization", "Bearer SECRET")
	req.Proto, req.ProtoMajor, req.ProtoMinor = "HTTP/3.0", 3, 0
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	line := buf.String()
	for _, want := range []string{"proto=HTTP/3.0", "path=/x", "status=200", "encoding=br", "method=GET"} {
		if !strings.Contains(line, want) {
			t.Fatalf("access log %q lacks %q", line, want)
		}
	}
	if strings.Contains(line, "SECRET") {
		t.Fatalf("access log leaked a credential or query: %q", line)
	}
}

func TestAccessLogDisabledIsNoop(t *testing.T) {
	if AccessLog(nil) == nil {
		t.Fatal("nil logger must yield a pass-through handler, not nil")
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AccessLog(nil))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
}
