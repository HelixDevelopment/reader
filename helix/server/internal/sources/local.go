package sources

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Local indexes an existing directory without modifying it. SMB/NFS mounts
// use this connector; their metadata database remains elsewhere.
type Local struct{ root string }

func NewLocal(cfg Config) (Connector, error) {
	root := cfg.Root
	if root == "" {
		root = cfg.Endpoint
	}
	if root == "" {
		return nil, fmt.Errorf("local source root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local source is not a directory")
	}
	return &Local{root: filepath.Clean(abs)}, nil
}
func (l *Local) Capabilities() Capabilities {
	return Capabilities{RecursiveList: true, RangeRead: true, Native: true}
}
func (l *Local) Close() error { return nil }
func (l *Local) resolve(name string) (string, error) {
	if name == "" || name == "." {
		return l.root, nil
	}
	if filepath.IsAbs(name) {
		return "", ErrTraversal
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrTraversal
	}
	p := filepath.Join(l.root, clean)
	rel, err := filepath.Rel(l.root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrTraversal
	}
	// Resolve the parent to prevent symlink escapes while allowing a final
	// regular file symlink only when its target stays under root.
	parent, err := filepath.EvalSymlinks(filepath.Dir(p))
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	relParent, err := filepath.Rel(l.root, parent)
	if err != nil || relParent == ".." || strings.HasPrefix(relParent, ".."+string(filepath.Separator)) {
		return "", ErrTraversal
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	relResolved, err := filepath.Rel(l.root, resolved)
	if err != nil || relResolved == ".." || strings.HasPrefix(relResolved, ".."+string(filepath.Separator)) {
		return "", ErrTraversal
	}
	return resolved, nil
}
func (l *Local) List(ctx context.Context, prefix string) ([]Entry, error) {
	base, err := l.resolve(prefix)
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	err = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(l.root, path)
		if err != nil {
			return err
		}
		out = append(out, Entry{Path: filepath.ToSlash(rel), Name: d.Name(), Size: info.Size(), Modified: info.ModTime()})
		return nil
	})
	return out, err
}

// Revalidate returns live size and modification time. Content serving uses
// both to decide whether the scan-time strong hash validator is still safe.
func (l *Local) Revalidate(_ context.Context, name string) (int64, time.Time, error) {
	p, err := l.resolve(name)
	if err != nil {
		return -1, time.Time{}, err
	}
	info, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return -1, time.Time{}, ErrNotFound
		}
		return -1, time.Time{}, err
	}
	if !info.Mode().IsRegular() {
		return -1, time.Time{}, fmt.Errorf("source: not a regular file")
	}
	return info.Size(), info.ModTime(), nil
}

// OpenRevalidated opens a file and stats that same descriptor. GET content
// responses can therefore validate and read one inode even if the catalog path
// is renamed or replaced after the open.
func (l *Local) OpenRevalidated(_ context.Context, name string) (io.ReadSeekCloser, int64, time.Time, error) {
	p, err := l.resolve(name)
	if err != nil {
		return nil, -1, time.Time{}, err
	}
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, -1, time.Time{}, ErrNotFound
		}
		return nil, -1, time.Time{}, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, -1, time.Time{}, err
	}
	if !info.Mode().IsRegular() {
		f.Close()
		return nil, -1, time.Time{}, fmt.Errorf("source: not a regular file")
	}
	return f, info.Size(), info.ModTime(), nil
}

func (l *Local) Open(_ context.Context, name string) (io.ReadCloser, error) {
	p, err := l.resolve(name)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("source: not a regular file")
	}
	return os.Open(p)
}
func (l *Local) OpenRange(ctx context.Context, name string, offset, length int64) (io.ReadCloser, error) {
	if offset < 0 || length < -1 {
		return nil, fmt.Errorf("source: invalid range")
	}
	f, err := l.Open(ctx, name)
	if err != nil {
		return nil, err
	}
	if _, err := f.(*os.File).Seek(offset, io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}
	if length < 0 {
		return f, nil
	}
	return struct {
		io.Reader
		io.Closer
	}{Reader: io.LimitReader(f, length), Closer: f}, nil
}
