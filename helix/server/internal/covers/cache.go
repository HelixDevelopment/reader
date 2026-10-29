// Package covers retrieves and validates publication artwork without proxying
// arbitrary response bodies or allowing access to private network addresses.
package covers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "golang.org/x/image/webp"
)

type Cached struct {
	Path string
	MIME string
}
type Cache struct {
	Dir      string
	Client   *http.Client
	MaxBytes int64
}

func New(dir string) *Cache { return &Cache{Dir: dir, Client: safeClient(), MaxBytes: 16 << 20} }
func safeClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, ForceAttemptHTTP2: true, MaxIdleConns: 20, IdleConnTimeout: 90 * time.Second, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 20 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, fmt.Errorf("cover host resolves to a non-public address")
			}
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("cover host has no addresses")
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	return &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return fmt.Errorf("invalid redirect scheme")
		}
		if req.URL.User != nil {
			return fmt.Errorf("credentials in cover URL forbidden")
		}
		return nil
	}}
}
func publicIP(ip net.IP) bool {
	return !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
func (c *Cache) paths(key string) []Cached {
	return []Cached{{filepath.Join(c.Dir, key+".jpg"), "image/jpeg"}, {filepath.Join(c.Dir, key+".png"), "image/png"}, {filepath.Join(c.Dir, key+".gif"), "image/gif"}, {filepath.Join(c.Dir, key+".webp"), "image/webp"}}
}
func cacheKey(edition, raw string) string {
	h := sha256.Sum256([]byte(edition + "\x00" + raw))
	return hex.EncodeToString(h[:])
}
func (c *Cache) Get(ctx context.Context, edition, raw string) (Cached, error) {
	if c.Dir == "" {
		return Cached{}, fmt.Errorf("cover cache is not configured")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Cached{}, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Cached{}, fmt.Errorf("cover URL must use HTTP(S)")
	}
	if u.User != nil {
		return Cached{}, fmt.Errorf("credentials in cover URL forbidden")
	}
	key := cacheKey(edition, raw)
	for _, candidate := range c.paths(key) {
		if info, err := os.Stat(candidate.Path); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Cached{}, err
	}
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/gif")
	req.Header.Set("User-Agent", "Helix-Cover-Fetcher/1")
	client := c.Client
	if client == nil {
		client = safeClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return Cached{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Cached{}, fmt.Errorf("cover server returned HTTP %d", resp.StatusCode)
	}
	max := c.MaxBytes
	if max <= 0 {
		max = 16 << 20
	}
	if resp.ContentLength > max {
		return Cached{}, fmt.Errorf("cover exceeds size limit")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return Cached{}, err
	}
	if int64(len(data)) > max {
		return Cached{}, fmt.Errorf("cover exceeds size limit")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Cached{}, fmt.Errorf("invalid cover image")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 20000 || cfg.Height > 20000 || int64(cfg.Width)*int64(cfg.Height) > 100_000_000 {
		return Cached{}, fmt.Errorf("cover dimensions exceed limits")
	}
	mime := http.DetectContentType(data)
	ext := ""
	switch mime {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	default:
		return Cached{}, fmt.Errorf("unsupported cover media type")
	}
	if err := os.MkdirAll(c.Dir, 0o750); err != nil {
		return Cached{}, err
	}
	tmp, err := os.CreateTemp(c.Dir, ".cover-*")
	if err != nil {
		return Cached{}, err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o640); err != nil {
		tmp.Close()
		return Cached{}, err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return Cached{}, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return Cached{}, err
	}
	if err := tmp.Close(); err != nil {
		return Cached{}, err
	}
	final := filepath.Join(c.Dir, key+ext)
	if err := os.Rename(name, final); err != nil {
		return Cached{}, err
	}
	return Cached{Path: final, MIME: mime}, nil
}
