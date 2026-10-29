package sources

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidateAcquisitionIP(t *testing.T) {
	cases := []struct {
		name       string
		ip         string
		defaultErr error
		privateErr error // expected error when AllowPrivateNetwork is set
	}{
		{"loopback v4", "127.0.0.1", errAddrPrivate, nil},
		{"loopback v6", "::1", errAddrPrivate, nil},
		{"private 10/8", "10.1.2.3", errAddrPrivate, nil},
		{"private 192.168", "192.168.1.10", errAddrPrivate, nil},
		{"private 172.16", "172.16.0.5", errAddrPrivate, nil},
		{"private v6 ula", "fd00::1", errAddrPrivate, nil},
		{"cgnat", "100.64.0.1", errAddrPrivate, nil},
		{"link-local metadata", "169.254.169.254", errAddrRestricted, errAddrRestricted},
		{"link-local v6", "fe80::1", errAddrRestricted, errAddrRestricted},
		{"unspecified v4", "0.0.0.0", errAddrRestricted, errAddrRestricted},
		{"unspecified v6", "::", errAddrRestricted, errAddrRestricted},
		{"multicast v4", "224.0.0.1", errAddrRestricted, errAddrRestricted},
		{"multicast v6", "ff02::1", errAddrRestricted, errAddrRestricted},
		{"broadcast", "255.255.255.255", errAddrRestricted, errAddrRestricted},
		{"public v4", "93.184.216.34", nil, nil},
		{"public v6", "2606:2800:220:1:248:1893:25c8:1946", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("bad test IP %q", tc.ip)
			}
			h := &HTTP{}
			if err := h.validateAcquisitionIP(ip); err != tc.defaultErr {
				t.Errorf("default policy: got %v, want %v", err, tc.defaultErr)
			}
			h.allowPrivateNetwork = true
			if err := h.validateAcquisitionIP(ip); err != tc.privateErr {
				t.Errorf("private-network policy: got %v, want %v", err, tc.privateErr)
			}
		})
	}
}

func TestSameOriginAddress(t *testing.T) {
	mk := func(endpoint string) *HTTP {
		c, err := NewHTTP(Config{Endpoint: endpoint})
		if err != nil {
			t.Fatal(err)
		}
		return c.(*HTTP)
	}
	h := mk("http://127.0.0.1:8080/opds")
	for _, tc := range []struct {
		host, port string
		want       bool
	}{
		{"127.0.0.1", "8080", true},
		{"127.0.0.1", "8081", false}, // same host, different port is a different origin
		{"127.0.0.2", "8080", false},
		{"localhost", "8080", false}, // host string must match the configured endpoint
	} {
		if got := h.sameOriginAddress(tc.host, tc.port); got != tc.want {
			t.Errorf("sameOriginAddress(%q, %q) = %t, want %t", tc.host, tc.port, got, tc.want)
		}
	}
	def := mk("https://example.com/feed")
	if !def.sameOriginAddress("EXAMPLE.com.", "443") {
		t.Error("default https port and case/trailing-dot mismatch must be normalized")
	}
	if def.sameOriginAddress("example.com", "8443") {
		t.Error("non-default port must not match")
	}
}

func TestHTTPRejectsCrossOriginRedirectBeforeRequest(t *testing.T) {
	var targetRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetRequests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/metadata", http.StatusFound)
	}))
	defer origin.Close()
	connector, err := NewHTTP(Config{Endpoint: origin.URL})
	if err != nil {
		t.Fatal(err)
	}
	h := connector.(*HTTP)
	defer h.Close()
	_, err = h.Open(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "cross-origin redirect rejected") {
		t.Fatalf("cross-origin redirect error = %v", err)
	}
	if got := targetRequests.Load(); got != 0 {
		t.Fatalf("cross-origin target received %d requests before rejection", got)
	}
}

func TestHTTPAllowsSameOriginRedirect(t *testing.T) {
	var origin *httptest.Server
	origin = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/book" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "complete")
	}))
	defer origin.Close()
	connector, err := NewHTTP(Config{Endpoint: origin.URL})
	if err != nil {
		t.Fatal(err)
	}
	h := connector.(*HTTP)
	defer h.Close()
	rc, err := h.Open(context.Background(), "book")
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(rc)
	closeErr := rc.Close()
	if readErr != nil || closeErr != nil || string(data) != "complete" {
		t.Fatalf("same-origin redirect: data=%q read=%v close=%v", data, readErr, closeErr)
	}
}

func TestHTTPBodyStreamingHasNoWholeTransferDeadline(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(50 * time.Millisecond)
		_, _ = io.WriteString(w, "slow-body-complete")
	}))
	defer origin.Close()
	connector, err := NewHTTP(Config{Endpoint: origin.URL})
	if err != nil {
		t.Fatal(err)
	}
	h := connector.(*HTTP)
	defer h.Close()
	if h.client.Timeout != 0 || h.acqClient.Timeout != 0 {
		t.Fatalf("whole-transfer deadlines configured: client=%v acquisition=%v", h.client.Timeout, h.acqClient.Timeout)
	}
	h.client.Transport.(*http.Transport).ResponseHeaderTimeout = 10 * time.Millisecond
	rc, err := h.Open(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(rc)
	closeErr := rc.Close()
	if readErr != nil || closeErr != nil || string(data) != "slow-body-complete" {
		t.Fatalf("streaming body: data=%q read=%v close=%v", data, readErr, closeErr)
	}
}
