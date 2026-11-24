package liveverify

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/httpapi"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/sources"
	"github.com/quic-go/quic-go/http3"
)

// tlsOpts shapes a real TLS + HTTP/3 helix server (the real router with its real
// Alt-Svc and Brotli/gzip middleware) and, for mutants, breaks one property.
type tlsOpts struct {
	tcpMin     uint16 // server TCP floor; default TLS 1.2
	tcpMax     uint16
	protos     []string // ALPN; default h2, http/1.1
	noHTTP3    bool
	http3Flag  *bool // HELIX_HTTP3 config (Alt-Svc advertisement); default true
	stripBr    bool  // request middleware removing br from Accept-Encoding
	stripAltSv bool
}

type tlsServer struct {
	tcp  *http.Server
	quic *http3.Server
}

type tlsStarter struct {
	t    *testing.T
	dir  string
	opts tlsOpts
	port int
	srv  *tlsServer
}

func (s *tlsStarter) StartNoWait() (Process, error) {
	if err := EnsureCert(s.dir); err != nil {
		return nil, err
	}
	var ln net.Listener
	var err error
	for i := 0; i < 20; i++ {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, err
		}
		s.port = ln.Addr().(*net.TCPAddr).Port
		if pc, perr := net.ListenPacket("udp", "127.0.0.1:"+strconv.Itoa(s.port)); perr == nil {
			pc.Close()
			break
		}
		ln.Close()
		ln = nil
	}
	if ln == nil {
		return nil, context.DeadlineExceeded
	}
	addr := "127.0.0.1:" + strconv.Itoa(s.port)
	h3flag := true
	if s.opts.http3Flag != nil {
		h3flag = *s.opts.http3Flag
	}
	cfg := config.Config{Address: addr, TLSCertFile: filepath.Join(s.dir, "cert.pem"), TLSKeyFile: filepath.Join(s.dir, "key.pem"), HTTP3: h3flag, BrotliLevel: 4, OPDSUsername: "", AdminToken: "transport-admin-token"}
	repo := catalog.NewMemory()
	api := httpapi.New(cfg, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	var h http.Handler = api.Router()
	inner := h
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.opts.stripBr {
			r.Header.Set("Accept-Encoding", strings.ReplaceAll(r.Header.Get("Accept-Encoding"), "br", ""))
		}
		if s.opts.stripAltSv {
			w = &noAltSvc{w}
		}
		inner.ServeHTTP(w, r)
	})
	min := s.opts.tcpMin
	if min == 0 {
		min = tls.VersionTLS12
	}
	protos := s.opts.protos
	if protos == nil {
		protos = []string{"h2", "http/1.1"}
	}
	cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, err
	}
	tcp := &http.Server{Handler: h, TLSConfig: &tls.Config{MinVersion: min, MaxVersion: s.opts.tcpMax, NextProtos: protos, Certificates: []tls.Certificate{cert}}}
	if !s.opts.noHTTP3 {
		s.srv = &tlsServer{tcp: tcp, quic: &http3.Server{Addr: addr, Handler: h, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}}}}
		go s.srv.quic.ListenAndServe()
	} else {
		s.srv = &tlsServer{tcp: tcp}
	}
	go tcp.ServeTLS(ln, "", "")
	return s.srv, nil
}

type noAltSvc struct{ http.ResponseWriter }

func (n *noAltSvc) WriteHeader(code int) {
	n.ResponseWriter.Header().Del("Alt-Svc")
	n.ResponseWriter.WriteHeader(code)
}
func (n *noAltSvc) Write(b []byte) (int, error) {
	n.ResponseWriter.Header().Del("Alt-Svc")
	return n.ResponseWriter.Write(b)
}

func (t *tlsServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if t.quic != nil {
		t.quic.Close()
	}
	return t.tcp.Shutdown(ctx)
}

func runTransportAgainst(t *testing.T, o tlsOpts, extended bool) Results {
	dir := t.TempDir()
	s := &tlsStarter{t: t, dir: filepath.Join(dir, "work"), opts: o}
	return RunTransport(TransportConfig{Starter: s, BaseFunc: func() string { return "https://127.0.0.1:" + strconv.Itoa(s.port) },
		WorkDir: filepath.Join(dir, "work"), EvidenceDir: filepath.Join(dir, "evidence"), Extended: extended})
}

var transportParityNames = []string{
	"verified TLS readiness", "HTTP/3 Alt-Svc advertisement", "TLS 1.2 HTTP/1.1 request", "TLS HTTP/2 request", "TLS 1.3 HTTP/3 QUIC request",
}

func TestTransportPassesAgainstRealInProcessTLSServer(t *testing.T) {
	res := runTransportAgainst(t, tlsOpts{}, true)
	if res.Status != "PASS" {
		t.Fatalf("expected PASS, got %s (%s): %v", res.Status, failedName(res), res.Checks)
	}
	for i, want := range transportParityNames {
		if !strings.HasPrefix(res.Checks[i].Name, want) {
			t.Fatalf("check %d = %q, want prefix %q", i, res.Checks[i].Name, want)
		}
	}
	if res.Pass != len(transportParityNames)+2+8 {
		t.Fatalf("pass count %d", res.Pass)
	}
}

func TestTransportParityOnlyHasExactlyThePythonAssertions(t *testing.T) {
	res := runTransportAgainst(t, tlsOpts{}, false)
	if res.Status != "PASS" || res.Pass != len(transportParityNames) {
		t.Fatalf("parity run: %s pass=%d %v", res.Status, res.Pass, res.Checks)
	}
}

func boolp(b bool) *bool { return &b }

func TestTransportMutantsAreEachRejected(t *testing.T) {
	cases := []struct {
		name string
		opts tlsOpts
		want string
		// passes is the number of checks that must have PASSed before the FAIL (-1: not asserted).
		passes int
	}{
		{"no-alt-svc", tlsOpts{stripAltSv: true}, "HTTP/3 Alt-Svc advertisement", 1},
		{"http3-flag-off-no-advertisement", tlsOpts{http3Flag: boolp(false)}, "HTTP/3 Alt-Svc advertisement", 1},
		{"tls12-unsupported", tlsOpts{tcpMin: tls.VersionTLS13}, "Get ", 2},
		{"no-http2-alpn", tlsOpts{protos: []string{"http/1.1"}}, "TLS HTTP/2 request", 3},
		{"no-http3-listener", tlsOpts{noHTTP3: true}, "Get ", 4},
		{"tls-floor-lowered-to-1.0", tlsOpts{tcpMin: tls.VersionTLS10}, "[ext] TLS 1.1 handshake is refused", 5},
		{"brotli-unavailable", tlsOpts{stripBr: true}, "[ext] HTTP/2: Brotli preferred", 7},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			res := runTransportAgainst(t, c.opts, true)
			if res.Status != "FAIL" || res.ExitCode() != 1 {
				t.Fatalf("mutant %s not rejected: %s", c.name, res.Status)
			}
			if c.passes >= 0 && res.Pass != c.passes {
				t.Fatalf("mutant %s: %d checks passed before the failure, want %d", c.name, res.Pass, c.passes)
			}
			if got := failedName(res); !strings.HasPrefix(got, c.want) {
				t.Fatalf("mutant %s failed on %q, want %q", c.name, got, c.want)
			}
		})
	}
}
