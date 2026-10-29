// Package sources provides a uniform, read-only ingestion connector contract.
package sources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("source: not found")
	ErrTraversal = errors.New("source: path escapes source root")
)

// Entry is a remote or local file. Directories are not returned.
type Entry struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	ETag     string    `json:"etag,omitempty"`
}

// Capabilities describe behavior, not deployment assumptions.
type Capabilities struct {
	RecursiveList bool `json:"recursive_list"`
	RangeRead     bool `json:"range_read"`
	Watch         bool `json:"watch"`
	Native        bool `json:"native"`
}

// Connector is read-only by design. Source mutation belongs to a separately
// authorized management API and is never needed for indexing.
type Connector interface {
	List(context.Context, string) ([]Entry, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Capabilities() Capabilities
	Close() error
}

// RangeConnector is optional. length -1 means through end of file.
type RangeConnector interface {
	OpenRange(context.Context, string, int64, int64) (io.ReadCloser, error)
}

// Config carries credentials separately from serializable source metadata.
type Config struct {
	// ID optionally preserves source identity across process restarts. It is an
	// administrative identifier, not a credential or a connector-specific path.
	ID           string            `json:"id,omitempty"`
	Protocol     string            `json:"protocol"`
	Endpoint     string            `json:"endpoint"`
	Root         string            `json:"root"`
	Username     string            `json:"username"`
	Password     string            `json:"password"`
	RclonePath   string            `json:"rclone_path"`
	RcloneConfig string            `json:"rclone_config"`
	Headers      map[string]string `json:"headers"`
	// AllowPrivateNetwork permits cross-origin acquisition URLs (for example
	// OPDS download links pointing at a LAN host) to resolve to loopback or
	// private-range addresses. It is intended only for deployments that
	// intentionally trust the configured feed. Link-local, unspecified and
	// multicast destinations remain rejected even when this is set. The
	// configured endpoint itself is never restricted by this setting.
	AllowPrivateNetwork bool `json:"allow_private_network"`
}

type limitedWriter struct {
	w         io.Writer
	remaining int64
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	original := len(p)
	if w.remaining > 0 {
		if int64(len(p)) > w.remaining {
			p = p[:w.remaining]
		}
		n, err := w.w.Write(p)
		w.remaining -= int64(n)
		if err != nil {
			return n, err
		}
	}
	return original, nil
}

type Factory func(Config) (Connector, error)

// Registry is concurrency-safe and returns protocol names deterministically.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

func NewRegistry() *Registry { return &Registry{factories: map[string]Factory{}} }
func (r *Registry) Register(name string, f Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[strings.ToLower(name)] = f
}
func (r *Registry) New(cfg Config) (Connector, error) {
	r.mu.RLock()
	f, ok := r.factories[strings.ToLower(cfg.Protocol)]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("source: unsupported protocol %q", cfg.Protocol)
	}
	return f(cfg)
}
func (r *Registry) Protocols() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.factories))
	for p := range r.factories {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// DefaultRegistry includes native local/HTTP/WebDAV connectors, read-only
// mount support, OPDS clients and optional rclone-backed transports.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register("local", NewLocal)
	for _, p := range []string{"smb-mount", "nfs-mount"} {
		r.Register(p, NewLocal)
	}
	r.Register("http", NewHTTP)
	r.Register("https", NewHTTP)
	r.Register("webdav", NewWebDAV)
	r.Register("webdavs", NewWebDAV)
	r.Register("opds", NewOPDS)
	r.Register("opds2", NewOPDS)
	for _, p := range []string{"ftp", "ftps", "sftp", "smb", "s3", "google-drive", "dropbox", "onedrive", "box", "pcloud", "mega", "nextcloud"} {
		r.Register(p, NewRclone)
	}
	return r
}
