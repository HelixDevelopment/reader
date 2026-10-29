package identify

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultProviderClient is used when a resolver does not supply its own
// client. Timeouts are always bounded; http.DefaultClient is never used.
func defaultProviderClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// redirectSafeClient preserves the configured client's transport, timeout and
// redirect policy while ensuring provider credentials never cross an origin.
// Go's default redirect handling does not cover arbitrary API-key headers.
func redirectSafeClient(client *http.Client, secretHeaders []string) *http.Client {
	clone := *client
	previous := client.CheckRedirect
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		if previous != nil {
			if err := previous(req, via); err != nil {
				return err
			}
		}
		if len(via) > 0 && !sameOrigin(via[0].URL, req.URL) {
			req.Header.Del("Authorization")
			req.Header.Del("Cookie")
			for _, header := range secretHeaders {
				req.Header.Del(header)
			}
		}
		return nil
	}
	return &clone
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

// sanitizeError strips query strings, fragments and userinfo from any URL
// embedded in an error so API keys and credentials never reach logs,
// stored evidence, or API responses.
func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		sanitized := *urlErr
		if u, perr := url.Parse(urlErr.URL); perr == nil {
			u.RawQuery = ""
			u.Fragment = ""
			u.User = nil
			sanitized.URL = u.String()
		} else {
			sanitized.URL = "<redacted>"
		}
		return &sanitized
	}
	return err
}

// requireSecureEndpoint rejects cleartext provider endpoints when API
// credentials would be sent. Loopback remains allowed so deterministic
// tests and local gateways can run without TLS.
func requireSecureEndpoint(raw string, hasCredentials bool) error {
	if !hasCredentials {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("identify: invalid endpoint URL: %w", err)
	}
	if strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	if strings.EqualFold(u.Scheme, "http") && isLoopbackHost(u.Hostname()) {
		return nil
	}
	return fmt.Errorf("identify: cleartext endpoint %s forbidden when API credentials are configured", u.Scheme+"://"+u.Host)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
