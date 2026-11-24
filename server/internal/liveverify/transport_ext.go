package liveverify

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"
	"time"

	"github.com/quic-go/quic-go/http3"
)

// extendedTransport adds the checks verify_live_transport.py does not make:
// the TLS version floor, Brotli/gzip negotiation and the identity fallback, over
// HTTP/2 and HTTP/3. Every name carries "[ext]" so a parity comparison can tell
// them from the Python assertion inventory.
func extendedTransport(rec *Recorder, cfg TransportConfig, roots *x509.CertPool, h2 *http.Client, h3 *http3.Transport) {
	host := strings.TrimPrefix(cfg.Base, "https://")
	// A TLS 1.1 client must be refused: the server floor is TLS 1.2.
	conn, err := tls.DialWithDialer(&dialer, "tcp", host, &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11})
	if conn != nil {
		conn.Close()
	}
	// Only a server-sent alert counts: a refusal raised locally by the client
	// stack would pass vacuously without ever reaching the server floor.
	rec.Require(err != nil && strings.Contains(err.Error(), "remote error: tls: protocol version not supported"), "[ext] TLS 1.1 handshake is refused by the server (floor is TLS 1.2)")
	// A TLS 1.3 client negotiates 1.3 on the TCP listener.
	conn13, err := tls.DialWithDialer(&dialer, "tcp", host, &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS13})
	ok13 := err == nil && conn13.ConnectionState().Version == tls.VersionTLS13
	if conn13 != nil {
		conn13.Close()
	}
	rec.Require(ok13, "[ext] TLS 1.3 handshake succeeds on the TCP listener")

	h3c := &http.Client{Transport: h3, Timeout: 15 * time.Second}
	for _, tr := range []struct {
		label  string
		client *http.Client
	}{{"HTTP/2", h2}, {"HTTP/3", h3c}} {
		get := func(accept string) (*http.Response, []byte) {
			headers := map[string]string{}
			if accept != "" {
				headers["Accept-Encoding"] = accept
			}
			res, body, err := fetch(tr.client, cfg.Base+"/v1/formats", headers)
			must(err)
			return res, body
		}
		res, body := get("br, gzip")
		plain, err := decodeBody(res.Header.Get("Content-Encoding"), body)
		must(err)
		rec.Require(res.StatusCode == 200 && res.Header.Get("Content-Encoding") == "br" && strings.Contains(string(plain), "formats"), "[ext] "+tr.label+": Brotli preferred when offered with gzip")
		res, body = get("gzip")
		plain, err = decodeBody(res.Header.Get("Content-Encoding"), body)
		must(err)
		rec.Require(res.StatusCode == 200 && res.Header.Get("Content-Encoding") == "gzip" && strings.Contains(string(plain), "formats"), "[ext] "+tr.label+": gzip when only gzip is offered")
		res, body = get("identity")
		rec.Require(res.StatusCode == 200 && res.Header.Get("Content-Encoding") == "" && strings.Contains(string(body), "formats"), "[ext] "+tr.label+": identity fallback when only identity is offered")
		res, body = get("")
		rec.Require(res.StatusCode == 200 && res.Header.Get("Content-Encoding") == "" && strings.Contains(string(body), "formats"), "[ext] "+tr.label+": no Accept-Encoding gives an identity body")
	}
}

var dialer = netDialer()
