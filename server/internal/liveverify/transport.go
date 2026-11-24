package liveverify

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/quic-go/quic-go/http3"
)

// TransportConfig configures the TLS HTTP/1.1, HTTP/2 and HTTP/3 verifier.
type TransportConfig struct {
	Starter ProcessStarter // starts the server (no readiness wait; the verifier polls)
	Base    string         // https://127.0.0.1:28443
	// BaseFunc, when set, supplies Base after the server has started (tests that pick a free port).
	BaseFunc    func() string
	WorkDir     string // holds cert.pem/key.pem and the *-body.json files
	EvidenceDir string
	// Extended adds the checks the Python verifier lacks (TLS floor, Brotli/gzip
	// negotiation, identity fallback), each named with the "[ext]" prefix.
	Extended bool
}

// EnsureCert makes sure dir holds a usable self-signed RSA-2048 certificate for
// 127.0.0.1 (same shape as the Python verifier's openssl call, one day valid). An
// existing pair is reused only when key.pem loads with cert.pem and the certificate
// has more than an hour left; an expired certificate or a missing key is regenerated.
func EnsureCert(dir string) error {
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if pair, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil && len(pair.Certificate) > 0 {
		if leaf, err := x509.ParseCertificate(pair.Certificate[0]); err == nil && time.Until(leaf.NotAfter) > time.Hour {
			return nil
		}
	}
	return writeCert(dir, time.Now().Add(-time.Minute), time.Now().Add(24*time.Hour))
}

func writeCert(dir string, notBefore, notAfter time.Time) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tpl := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: "127.0.0.1"},
		NotBefore: notBefore, NotAfter: notAfter,
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "key.pem"), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "cert.pem"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644)
}

func rootsFrom(file string) (*x509.CertPool, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("no CA certificates found in %s", file)
	}
	return pool, nil
}

type tlsFlavor struct {
	roots       *x509.CertPool
	minV, maxV  uint16
	http2, http bool // http2: negotiate h2; otherwise force HTTP/1.1
}

func (f tlsFlavor) client() *http.Client {
	cfg := &tls.Config{RootCAs: f.roots, MinVersion: f.minV, MaxVersion: f.maxV}
	tr := &http.Transport{TLSClientConfig: cfg, DisableCompression: true, Proxy: nil}
	if f.http2 {
		tr.ForceAttemptHTTP2 = true
	} else {
		tr.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
		cfg.NextProtos = []string{"http/1.1"}
	}
	return &http.Client{Transport: tr, Timeout: 20 * time.Second}
}

// fetch performs GET url and returns the response with its body fully read.
func fetch(c *http.Client, url string, headers map[string]string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := c.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return res, body, err
}

// protoVersion renders the response protocol the way curl's %{http_version} does.
func protoVersion(res *http.Response) string {
	switch {
	case res.ProtoMajor == 1 && res.ProtoMinor == 1:
		return "1.1"
	case res.ProtoMajor == 1 && res.ProtoMinor == 0:
		return "1.0"
	default:
		return fmt.Sprint(res.ProtoMajor)
	}
}

// RunTransport runs the transport verification and returns (and writes) the results.
func RunTransport(cfg TransportConfig) Results {
	rec := &Recorder{}
	failed := false
	var proc Process
	finish := func() Results {
		res := rec.Results(failed)
		if err := WriteResults(cfg.EvidenceDir, res); err != nil {
			res.Status = "FAIL"
		}
		return res
	}
	if err := os.MkdirAll(cfg.EvidenceDir, 0o755); err != nil {
		rec.Fail("evidence directory: " + err.Error())
		failed = true
		return finish()
	}
	ok := rec.Guard(func() {
		must(os.MkdirAll(cfg.WorkDir, 0o755))
		must(EnsureCert(cfg.WorkDir))
		roots, err := rootsFrom(filepath.Join(cfg.WorkDir, "cert.pem"))
		must(err)
		p, err := cfg.Starter.StartNoWait()
		must(err)
		proc = p
		if cfg.BaseFunc != nil {
			cfg.Base = cfg.BaseFunc()
		}
		verified := tlsFlavor{roots: roots, minV: tls.VersionTLS12, http2: true}.client()
		var response *http.Response
		lastError := "no request completed"
		for i := 0; i < 300; i++ {
			if ex, ok := p.(Exiter); ok && ex.Exited() {
				panic(fmt.Errorf("TLS server exited before readiness"))
			}
			res, _, err := fetch(&http.Client{Transport: verified.Transport, Timeout: time.Second}, cfg.Base+"/healthz", nil)
			if err == nil {
				response = res
				if res.StatusCode == 200 {
					break
				}
				lastError = fmt.Sprintf("HTTP %d", res.StatusCode)
				continue
			}
			lastError = fmt.Sprintf("%T: %v", err, err)
			time.Sleep(100 * time.Millisecond)
		}
		rec.Require(response != nil && response.StatusCode == 200, fmt.Sprintf("verified TLS readiness (last error: %s)", lastError))
		rec.Require(strings.Contains(response.Header.Get("Alt-Svc"), "h3="), "HTTP/3 Alt-Svc advertisement")

		h1 := tlsFlavor{roots: roots, minV: tls.VersionTLS12, maxV: tls.VersionTLS12}.client()
		res, body, err := fetch(h1, cfg.Base+"/healthz", nil)
		must(err)
		must(os.WriteFile(filepath.Join(cfg.WorkDir, "http1-body.json"), body, 0o644))
		rec.Require(res.StatusCode < 400 && protoVersion(res) == "1.1" && res.TLS != nil && res.TLS.Version == tls.VersionTLS12, "TLS 1.2 HTTP/1.1 request")

		h2 := tlsFlavor{roots: roots, minV: tls.VersionTLS12, http2: true}.client()
		res, body, err = fetch(h2, cfg.Base+"/healthz", nil)
		must(err)
		must(os.WriteFile(filepath.Join(cfg.WorkDir, "http2-body.json"), body, 0o644))
		rec.Require(res.StatusCode < 400 && protoVersion(res) == "2", "TLS HTTP/2 request")

		h3 := &http3.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS13}}
		defer h3.Close()
		res, _, err = fetch(&http.Client{Transport: h3, Timeout: 15 * time.Second}, cfg.Base+"/healthz", nil)
		must(err)
		rec.Require(res.ProtoMajor == 3 && res.StatusCode == 200, "TLS 1.3 HTTP/3 QUIC request")

		if cfg.Extended {
			extendedTransport(rec, cfg, roots, h2, h3)
		}
	})
	if !ok {
		failed = true
	}
	if proc != nil {
		if err := proc.Stop(); err != nil {
			failed = true
			rec.Fail("server shutdown: " + err.Error())
		}
	}
	return finish()
}
