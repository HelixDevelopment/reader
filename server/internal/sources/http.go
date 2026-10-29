package sources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// errAddrRestricted marks destinations that are never valid cross-origin
	// acquisition targets, regardless of configuration.
	errAddrRestricted = errors.New("source: acquisition target address is in a restricted range")
	// errAddrPrivate marks loopback/private destinations rejected because the
	// source was not configured with AllowPrivateNetwork.
	errAddrPrivate = errors.New("source: cross-origin acquisition to loopback or private addresses is disabled")
)

// cgnatRange is the RFC 6598 shared address space: not globally routable and
// treated as private for cross-origin acquisition purposes.
var cgnatRange = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// HTTP exposes one resource or a configured manifest endpoint. Directory
// listing is deliberately not inferred from arbitrary HTML.
type HTTP struct {
	base     *url.URL
	basePort string
	// client serves the configured endpoint. Plain HTTP/WebDAV redirects are
	// rejected before any cross-origin request is issued.
	client *http.Client
	// acqClient serves feed-controlled acquisition URLs (OPDS). Its
	// transport validates every dialed IP against the SSRF policy.
	acqClient *http.Client
	headers   map[string]string
	username  string
	password  string
	// allowPrivateNetwork permits cross-origin acquisition dials to loopback
	// and private-range addresses (Config.AllowPrivateNetwork).
	allowPrivateNetwork bool
	// allowCrossOriginRedirects permits acquisition requests to follow
	// redirects to a different origin. Every hop is still validated by the
	// dialer and configured headers/credentials are stripped on origin change.
	// Plain HTTP sources keep strict same-origin redirect handling.
	allowCrossOriginRedirects bool
}

func NewHTTP(cfg Config) (Connector, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("HTTP source requires http(s) endpoint")
	}
	if u.User != nil {
		return nil, fmt.Errorf("credentials in endpoint URLs are forbidden")
	}
	h := &HTTP{base: u, headers: cfg.Headers, username: cfg.Username, password: cfg.Password, allowPrivateNetwork: cfg.AllowPrivateNetwork}
	h.basePort = u.Port()
	if h.basePort == "" {
		if u.Scheme == "https" {
			h.basePort = "443"
		} else {
			h.basePort = "80"
		}
	}
	checkRedirect := func(allowCrossOrigin bool) func(*http.Request, []*http.Request) error {
		return func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("source: too many redirects")
			}
			if req.URL.User != nil {
				return fmt.Errorf("source: credentials in redirect URL forbidden")
			}
			if len(via) > 0 && (req.URL.Scheme != via[len(via)-1].URL.Scheme || req.URL.Host != via[len(via)-1].URL.Host) {
				if !allowCrossOrigin {
					return fmt.Errorf("source: cross-origin redirect rejected")
				}
				for key := range req.Header {
					req.Header.Del(key)
				}
			}
			return nil
		}
	}
	// Client.Timeout covers body streaming and would abort long books on slow
	// links. Bound connection/header establishment instead; request contexts
	// remain responsible for cancellation of the complete transfer.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 60 * time.Second
	h.client = &http.Client{Transport: transport, CheckRedirect: checkRedirect(false)}
	acqTransport := http.DefaultTransport.(*http.Transport).Clone()
	acqTransport.ResponseHeaderTimeout = 60 * time.Second
	acqTransport.DialContext = h.dialContext
	h.acqClient = &http.Client{Transport: acqTransport, CheckRedirect: checkRedirect(true)}
	return h, nil
}

// sameOriginAddress reports whether a dial target belongs to the configured
// endpoint origin. The configured endpoint (including LAN/loopback NAS hosts)
// is never restricted; everything else is validated per resolved IP.
func (h *HTTP) sameOriginAddress(host, port string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	base := strings.TrimSuffix(strings.ToLower(h.base.Hostname()), ".")
	return host == base && port == h.basePort
}

// validateAcquisitionIP rejects destinations a cross-origin feed must not
// reach. Link-local (including cloud metadata ranges), unspecified, broadcast
// and multicast addresses are always rejected; loopback and private ranges
// require AllowPrivateNetwork.
func (h *HTTP) validateAcquisitionIP(ip net.IP) error {
	if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.Equal(net.IPv4bcast) {
		return errAddrRestricted
	}
	if h.allowPrivateNetwork {
		return nil
	}
	if ip.IsLoopback() || ip.IsPrivate() || cgnatRange.Contains(ip) {
		return errAddrPrivate
	}
	return nil
}

// dialContext validates the actual resolved IPs of cross-origin acquisition
// targets at connect time, pinning the dial to validated addresses so DNS
// answers cannot be swapped after a check. The configured endpoint origin is
// dialed without restriction. Errors are sanitized and never include
// credentials.
func (h *HTTP) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("source: invalid dial address")
	}
	d := &net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	if h.sameOriginAddress(host, port) {
		return d.DialContext(ctx, network, addr)
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("source: acquisition address resolution failed")
	}
	var policyErr, dialErr error
	for _, ip := range ips {
		if err := h.validateAcquisitionIP(ip); err != nil {
			policyErr = err
			continue
		}
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		dialErr = err
	}
	if dialErr != nil {
		return nil, fmt.Errorf("source: acquisition connection failed")
	}
	if policyErr != nil {
		return nil, policyErr
	}
	return nil, fmt.Errorf("source: acquisition address rejected")
}
func (h *HTTP) Capabilities() Capabilities { return Capabilities{RangeRead: true, Native: true} }
func (h *HTTP) Close() error {
	h.client.CloseIdleConnections()
	h.acqClient.CloseIdleConnections()
	return nil
}
func (h *HTTP) resource(name string) (*url.URL, error) {
	u := *h.base
	if name != "" {
		if strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
			return nil, ErrTraversal
		}
		ref, err := url.Parse(name)
		if err != nil || ref.IsAbs() {
			return nil, ErrTraversal
		}
		u = *h.base.ResolveReference(ref)
	}
	if u.Scheme != h.base.Scheme || u.Host != h.base.Host {
		return nil, fmt.Errorf("source: cross-origin redirect target rejected")
	}
	return &u, nil
}
func (h *HTTP) do(ctx context.Context, method, name string) (*http.Response, error) {
	u, err := h.resource(name)
	if err != nil {
		return nil, err
	}
	return h.doURL(ctx, method, u, "")
}
func (h *HTTP) doURL(ctx context.Context, method string, u *url.URL, byteRange string) (*http.Response, error) {
	return h.doWith(ctx, h.client, method, u, byteRange, false)
}

// doAcquisition fetches a feed-controlled URL through the SSRF-validating
// acquisition client. Cross-origin redirects are followed (credential-free,
// each hop dial-validated) only when allowCrossOriginRedirects is set.
func (h *HTTP) doAcquisition(ctx context.Context, method string, u *url.URL, byteRange string) (*http.Response, error) {
	return h.doWith(ctx, h.acqClient, method, u, byteRange, h.allowCrossOriginRedirects)
}

func (h *HTTP) doWith(ctx context.Context, client *http.Client, method string, u *url.URL, byteRange string, allowCrossOriginRedirects bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if u.Scheme == h.base.Scheme && u.Host == h.base.Host {
		for k, v := range h.headers {
			req.Header.Set(k, v)
		}
		if h.username != "" {
			req.SetBasicAuth(h.username, h.password)
		}
	}
	if byteRange != "" {
		req.Header.Set("Range", byteRange)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		resp.Body.Close()
		return nil, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("source: HTTP %d", resp.StatusCode)
	}
	// Redirects to another origin are not implicitly trusted with headers.
	if !allowCrossOriginRedirects && (resp.Request.URL.Scheme != u.Scheme || resp.Request.URL.Host != u.Host) {
		resp.Body.Close()
		return nil, fmt.Errorf("source: cross-origin redirect rejected")
	}
	return resp, nil
}
func (h *HTTP) List(ctx context.Context, prefix string) ([]Entry, error) {
	resp, err := h.do(ctx, http.MethodHead, prefix)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	mod, _ := http.ParseTime(resp.Header.Get("Last-Modified"))
	u, _ := h.resource(prefix)
	return []Entry{{Path: prefix, Name: u.Path, Size: resp.ContentLength, Modified: mod, ETag: resp.Header.Get("ETag")}}, nil
}
func (h *HTTP) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	resp, err := h.do(ctx, http.MethodGet, name)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
func (h *HTTP) OpenRange(ctx context.Context, name string, offset, length int64) (io.ReadCloser, error) {
	if offset < 0 || length < -1 {
		return nil, fmt.Errorf("source: invalid range")
	}
	value := fmt.Sprintf("bytes=%d-", offset)
	if length >= 0 {
		value = fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)
	}
	u, err := h.resource(name)
	if err != nil {
		return nil, err
	}
	resp, err := h.doURL(ctx, http.MethodGet, u, value)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusPartialContent && offset != 0 {
		resp.Body.Close()
		return nil, fmt.Errorf("source: server ignored byte range")
	}
	return resp.Body, nil
}
func (h *HTTP) getAcquisition(ctx context.Context, raw, byteRange string) (*http.Response, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if !u.IsAbs() {
		resolved, err := h.resource(raw)
		if err != nil {
			return nil, err
		}
		u = resolved
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("source: unsupported acquisition scheme")
	}
	if u.User != nil {
		return nil, fmt.Errorf("source: credentials in acquisition URL forbidden")
	}
	return h.doAcquisition(ctx, http.MethodGet, u, byteRange)
}
func (h *HTTP) openAcquisition(ctx context.Context, raw string) (io.ReadCloser, error) {
	resp, err := h.getAcquisition(ctx, raw, "")
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
func (h *HTTP) openAcquisitionRange(ctx context.Context, raw string, offset, length int64) (io.ReadCloser, error) {
	if offset < 0 || length < -1 {
		return nil, fmt.Errorf("source: invalid range")
	}
	value := fmt.Sprintf("bytes=%d-", offset)
	if length >= 0 {
		value = fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)
	}
	resp, err := h.getAcquisition(ctx, raw, value)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusPartialContent && offset != 0 {
		resp.Body.Close()
		return nil, fmt.Errorf("source: server ignored byte range")
	}
	return resp.Body, nil
}
