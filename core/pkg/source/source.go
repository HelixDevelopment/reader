// Package source implements the Source Abstraction Layer from docs/25
// (Book Sources & Protocols): one uniform interface over OPDS, WebDAV,
// SFTP, SMB, S3, cloud drives, local storage and email/RSS imports.
// Concrete drivers live behind the Driver interface so the rest of the
// core is protocol-agnostic and drivers can be swapped or extended
// (e.g. by wrapping the containers submodule for remote mounts).
package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// Protocol enumerates supported source protocols (docs/25 §matrix).
type Protocol string

const (
	ProtoLocal     Protocol = "local"
	ProtoOPDS1     Protocol = "opds1"
	ProtoOPDS2     Protocol = "opds2"
	ProtoWebDAV    Protocol = "webdav"
	ProtoFTP       Protocol = "ftp"
	ProtoFTPS      Protocol = "ftps"
	ProtoSFTP      Protocol = "sftp"
	ProtoSMB       Protocol = "smb"
	ProtoNFS       Protocol = "nfs"
	ProtoHTTP      Protocol = "http"
	ProtoS3        Protocol = "s3"
	ProtoGDrive    Protocol = "gdrive"
	ProtoDropbox   Protocol = "dropbox"
	ProtoOneDrive  Protocol = "onedrive"
	ProtoBox       Protocol = "box"
	ProtoPCloud    Protocol = "pcloud"
	ProtoMega      Protocol = "mega"
	ProtoNextcloud Protocol = "nextcloud"
	ProtoEmail     Protocol = "email"
	ProtoRSS       Protocol = "rss"
)

// Capability flags advertised by a driver.
type Capability struct {
	Browse       bool // hierarchical listing
	Search       bool // server-side query
	RangeRead    bool // partial reads (needed for streaming large books)
	Write        bool // upload / sync-back
	Watch        bool // change notifications
	AuthRequired bool
}

// Entry is one item in a source listing.
type Entry struct {
	Path     string
	Name     string
	IsDir    bool
	Size     int64
	Modified time.Time
	Format   string // detected book format for files, "" otherwise
}

// Book is a downloadable/openable book on a source.
type Book struct {
	SourcePath string
	Title      string
	Format     string
	Size       int64
	CoverURL   string
}

// Driver is the uniform protocol interface.
type Driver interface {
	Protocol() Protocol
	Capabilities() Capability
	Connect(ctx context.Context, cfg Config) error
	List(ctx context.Context, path string) ([]Entry, error)
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	// OpenRange must be implemented when Capabilities().RangeRead.
	OpenRange(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error)
	Close() error
}

// Config is per-source configuration; secrets are injected, never stored
// in the source record (docs/15 Security).
type Config struct {
	ID       string
	Label    string
	Endpoint string
	Username string
	Secret   string // resolved from the OS keystore by the platform shell
	Extra    map[string]string
}

var (
	ErrNotConnected     = errors.New("source: driver not connected")
	ErrRangeUnsupported = errors.New("source: range reads unsupported by driver")
	ErrUnknownProtocol  = errors.New("source: unknown protocol")
)

// Registry maps protocols to driver factories.
type Registry struct {
	mu        sync.RWMutex
	factories map[Protocol]func() Driver
}

func NewRegistry() *Registry { return &Registry{factories: map[Protocol]func() Driver{}} }

// Register adds a driver factory; safe for concurrent use at init.
func (r *Registry) Register(p Protocol, f func() Driver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[p] = f
}

// New instantiates a driver for protocol p.
func (r *Registry) New(p Protocol) (Driver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.factories[p]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProtocol, p)
	}
	return f(), nil
}

// Protocols lists registered protocols, sorted for determinism.
func (r *Registry) Protocols() []Protocol {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Protocol, 0, len(r.factories))
	for p := range r.factories {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Manager owns connected sources and enforces the source-vs-sync binding
// decoupling from docs/25: a source provides books; a binding optionally
// links it to a sync account, never the reverse dependency.
type Manager struct {
	reg     *Registry
	mu      sync.Mutex
	sources map[string]Driver           // config ID -> connected driver
	binds   map[string]string           // source ID -> sync account ID (optional)
}

func NewManager(reg *Registry) *Manager {
	return &Manager{reg: reg, sources: map[string]Driver{}, binds: map[string]string{}}
}

// Add connects and registers a source.
func (m *Manager) Add(ctx context.Context, cfg Config) error {
	d, err := m.reg.New(protocolOf(cfg.Endpoint))
	if err != nil {
		return err
	}
	if err := d.Connect(ctx, cfg); err != nil {
		return fmt.Errorf("source: connect %s: %w", cfg.ID, err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sources[cfg.ID] = d
	return nil
}

// BindSync links a source to a sync account (optional, one-directional).
func (m *Manager) BindSync(sourceID, syncAccountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sources[sourceID]; !ok {
		return fmt.Errorf("source: unknown source %q", sourceID)
	}
	m.binds[sourceID] = syncAccountID
	return nil
}

// Get returns a connected driver.
func (m *Manager) Get(id string) (Driver, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.sources[id]
	return d, ok
}

// CloseAll disconnects every source; collects errors deterministically.
func (m *Manager) CloseAll() []error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var errs []error
	ids := make([]string, 0, len(m.sources))
	for id := range m.sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := m.sources[id].Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", id, err))
		}
		delete(m.sources, id)
	}
	return errs
}

func protocolOf(endpoint string) Protocol {
	switch {
	case len(endpoint) >= 6 && endpoint[:6] == "opds2:":
		return ProtoOPDS2
	case len(endpoint) >= 5 && endpoint[:5] == "opds:":
		return ProtoOPDS1
	case len(endpoint) >= 7 && endpoint[:7] == "webdav:":
		return ProtoWebDAV
	case len(endpoint) >= 5 && endpoint[:5] == "sftp:":
		return ProtoSFTP
	case len(endpoint) >= 4 && endpoint[:4] == "smb:":
		return ProtoSMB
	case len(endpoint) >= 3 && endpoint[:3] == "s3:":
		return ProtoS3
	default:
		return ProtoLocal
	}
}
