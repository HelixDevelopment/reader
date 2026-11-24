package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

const (
	defaultLCPCacheGrace   = 10 * time.Minute
	defaultLCPCacheMax     = int64(16) << 30
	lcpBackoffBase         = 5 * time.Minute
	lcpBackoffMax          = time.Hour
	lcpBackoffMaxExponent  = 6
	lcpMaxTrackedFailures  = 4096
	defaultLCPHostFailures = 3
)

type lcpFailure struct {
	n     int
	until time.Time
	seq   uint64
}

func (f *LCPFulfiller) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

// hostThreshold is how many consecutive failures against one publication host
// (across any licenses) start that host's backoff. <0 disables host backoff.
func (f *LCPFulfiller) hostThreshold() int {
	switch {
	case f.HostFailureThreshold < 0:
		return 0
	case f.HostFailureThreshold == 0:
		return defaultLCPHostFailures
	}
	return f.HostFailureThreshold
}

// backingOff reports whether a recent failure for this license still blocks a
// new publication fetch. State is in memory: a restart forgets it, which costs
// at most one extra bounded fetch per license.
func (f *LCPFulfiller) backingOff(licenseSHA string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	fl, ok := f.fails[licenseSHA]
	return ok && f.now().Before(fl.until)
}

// hostBackingOff reports whether the publication host (host:port) is in its
// failure window, no matter which license names it.
func (f *LCPFulfiller) hostBackingOff(host string) bool {
	if host == "" || f.hostThreshold() == 0 {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	fl, ok := f.hostFails[host]
	return ok && f.now().Before(fl.until)
}

func backoffFor(n int) time.Duration {
	exp := n - 1
	if exp < 0 {
		exp = 0
	}
	if exp > lcpBackoffMaxExponent {
		exp = lcpBackoffMaxExponent
	}
	d := lcpBackoffBase << exp
	if d > lcpBackoffMax {
		d = lcpBackoffMax
	}
	return d
}

// evictOldest drops exactly the entry with the smallest sequence number, so a
// flood of new keys never wipes backoff recorded for others.
func evictOldest(m map[string]lcpFailure) {
	var key string
	var min uint64
	first := true
	for k, v := range m {
		if first || v.seq < min {
			key, min, first = k, v.seq, false
		}
	}
	if !first {
		delete(m, key)
	}
}

func (f *LCPFulfiller) recordFailure(licenseSHA, host string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	if f.fails == nil {
		f.fails = map[string]lcpFailure{}
	}
	if _, ok := f.fails[licenseSHA]; !ok && len(f.fails) >= lcpMaxTrackedFailures {
		evictOldest(f.fails)
	}
	fl := f.fails[licenseSHA]
	fl.n++
	fl.until = f.now().Add(backoffFor(fl.n))
	fl.seq = f.seq
	f.fails[licenseSHA] = fl
	th := f.hostThreshold()
	if host == "" || th == 0 {
		return
	}
	if f.hostFails == nil {
		f.hostFails = map[string]lcpFailure{}
	}
	if _, ok := f.hostFails[host]; !ok && len(f.hostFails) >= lcpMaxTrackedFailures {
		evictOldest(f.hostFails)
	}
	hf := f.hostFails[host]
	hf.n++
	hf.seq = f.seq
	if hf.n >= th {
		hf.until = f.now().Add(backoffFor(hf.n - th + 1))
	}
	f.hostFails[host] = hf
}

func (f *LCPFulfiller) clearFailure(licenseSHA, host string) {
	f.mu.Lock()
	delete(f.fails, licenseSHA)
	delete(f.hostFails, host)
	f.mu.Unlock()
}

// lease pins an artifact against GC until the matching release. Leases are
// in-process: they cover the window between a fetch finishing and the scan's
// MergeWork committing, which the grace window alone only approximates.
func (f *LCPFulfiller) lease(sha string) {
	f.mu.Lock()
	if f.leases == nil {
		f.leases = map[string]int{}
	}
	f.leases[sha]++
	f.mu.Unlock()
}

func (f *LCPFulfiller) release(sha string) {
	f.mu.Lock()
	if f.leases[sha] > 1 {
		f.leases[sha]--
	} else {
		delete(f.leases, sha)
	}
	f.epoch++
	f.mu.Unlock()
}

// touch refreshes an artifact's mtime so a cache hit is "recent" for GC.
func (f *LCPFulfiller) touch(sha string) {
	if p := f.ArtifactPath(sha); p != "" {
		now := time.Now()
		_ = os.Chtimes(p, now, now)
	}
}

// GCResult reports one cache collection.
type GCResult struct {
	Removed  int
	Warnings []string
}

// GC is Collect without the warnings.
func (f *LCPFulfiller) GC(ctx context.Context, repo catalog.Repository) (int, error) {
	r, err := f.Collect(ctx, repo)
	return r.Removed, err
}

// reclaim unlinks path (an artifact or an index pointing at artifact sha) only
// if, at the moment of unlinking, no in-flight scan leases sha and a fresh
// single-statement catalog read still does not reference it. A lease released
// or taken while the catalog is being read changes the epoch and the file is
// left for the next collection, so there is no window in which a merge can
// commit between the check and the unlink unseen.
func (f *LCPFulfiller) reclaim(ctx context.Context, repo catalog.Repository, sha, path string, grace time.Duration) bool {
	f.mu.Lock()
	if f.leases[sha] > 0 {
		f.mu.Unlock()
		return false
	}
	ep := f.epoch
	f.mu.Unlock()
	refs, err := repo.ReferencedLCPArtifacts(ctx)
	if err != nil || refs[sha] {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.leases[sha] > 0 || f.epoch != ep {
		return false
	}
	// Another process sharing CacheDir has its own leases, so the file's own age
	// is the cross-process guard: look again AFTER the catalog read. A cache hit
	// touches the file before its work is merged, so a file that became young
	// (mtime or ctime) in the meantime is in use.
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	now := time.Now()
	if now.Sub(info.ModTime()) < grace {
		return false
	}
	if ct, ok := fileCtime(info); ok && now.Sub(time.Unix(0, ct)) < grace {
		return false
	}
	return os.Remove(path) == nil
}

// Collect removes cache files no catalog row references: unreferenced
// artifacts, index files whose artifact is unreferenced, and stale temp files,
// each only once older than CacheGrace and not leased by an in-flight scan. The
// reference set comes from one consistent catalog read and is re-verified right
// before every unlink. Nothing younger than the grace window is ever removed,
// even over the size cap, and a referenced artifact is never removed; a cache
// larger than CacheMaxBytes only produces a warning. Any error reading the
// catalog aborts the collection and deletes nothing.
func (f *LCPFulfiller) Collect(ctx context.Context, repo catalog.Repository) (GCResult, error) {
	var res GCResult
	if f == nil || f.CacheDir == "" || repo == nil {
		return res, nil
	}
	refs, err := repo.ReferencedLCPArtifacts(ctx)
	if err != nil {
		return res, err
	}
	grace := f.CacheGrace
	if grace == 0 {
		grace = defaultLCPCacheGrace
	}
	if grace < 0 {
		grace = 0
	}
	limit := f.CacheMaxBytes
	if limit == 0 {
		limit = defaultLCPCacheMax
	}
	ents, err := os.ReadDir(f.CacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return res, err
	}
	var total int64
	now := time.Now()
	for _, de := range ents {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		info, ierr := de.Info()
		if ierr != nil || !info.Mode().IsRegular() {
			continue
		}
		name := de.Name()
		p := filepath.Join(f.CacheDir, name)
		old := now.Sub(info.ModTime()) >= grace
		switch {
		case strings.HasSuffix(name, ".epub") && sha256Hex.MatchString(strings.TrimSuffix(name, ".epub")):
			sha := strings.TrimSuffix(name, ".epub")
			total += info.Size()
			if refs[sha] || !old {
				continue
			}
			if f.reclaim(ctx, repo, sha, p, grace) {
				res.Removed++
				total -= info.Size()
			}
		case strings.HasPrefix(name, "license-") && strings.HasSuffix(name, ".idx"):
			raw, rerr := os.ReadFile(p)
			if rerr != nil {
				continue
			}
			sha := strings.TrimSpace(string(raw))
			if !refs[sha] && old && f.reclaim(ctx, repo, sha, p, grace) {
				res.Removed++
			}
		case strings.HasSuffix(name, ".tmp"):
			if old && now.Sub(info.ModTime()) >= time.Hour && os.Remove(p) == nil {
				res.Removed++
			}
		}
	}
	if limit > 0 && total > limit {
		res.Warnings = append(res.Warnings, fmt.Sprintf(
			"LCP cache holds %d bytes, which exceeds the %d-byte cap (HELIX_LCP_CACHE_MAX_BYTES); referenced artifacts and files inside the grace window are never evicted, so nothing was removed for size",
			total, limit))
	}
	return res, nil
}

var (
	// ErrLCPArtifactMissing: the cached artifact is not there.
	ErrLCPArtifactMissing = errors.New("lcp cache: artifact is missing")
	// ErrLCPArtifactMismatch: the cached bytes are not the artifact the catalog
	// recorded (wrong size or wrong SHA-256).
	ErrLCPArtifactMismatch = errors.New("lcp cache: artifact does not match the catalog")
)

// OpenVerified opens the cached artifact for sha and proves its bytes are that
// artifact: regular file, size == size, SHA-256 == sha. The full hash is
// computed once per (path, inode, size, mtime, ctime) and the verdict remembered,
// so a steady stream of requests does not rehash a large file each time. ctime
// cannot be set from userspace, so an in-place rewrite that keeps the size and
// restores mtime still invalidates it; where ctime is unavailable the verdict
// cache is off and every call rehashes. The returned file is
// positioned at 0.
func (f *LCPFulfiller) OpenVerified(sha string, size int64) (*os.File, os.FileInfo, error) {
	path := f.ArtifactPath(sha)
	if path == "" {
		return nil, nil, ErrLCPArtifactMismatch
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, nil, ErrLCPArtifactMissing
	}
	info, err := fh.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		fh.Close()
		return nil, nil, ErrLCPArtifactMismatch
	}
	ctime, haveCtime := fileCtime(info)
	if haveCtime {
		f.mu.Lock()
		v, ok := f.verified[path]
		f.mu.Unlock()
		if ok && v.sha == sha && v.ctime == ctime && v.info.Size() == info.Size() && v.info.ModTime().Equal(info.ModTime()) && os.SameFile(v.info, info) {
			return fh, info, nil
		}
	}
	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil || hex.EncodeToString(h.Sum(nil)) != sha {
		fh.Close()
		return nil, nil, ErrLCPArtifactMismatch
	}
	if _, err := fh.Seek(0, io.SeekStart); err != nil {
		fh.Close()
		return nil, nil, err
	}
	if haveCtime {
		f.mu.Lock()
		if f.verified == nil || len(f.verified) >= lcpMaxTrackedFailures {
			f.verified = map[string]lcpVerdict{}
		}
		f.verified[path] = lcpVerdict{sha: sha, info: info, ctime: ctime}
		f.mu.Unlock()
	}
	return fh, info, nil
}

type lcpVerdict struct {
	sha   string
	info  os.FileInfo
	ctime int64
}
