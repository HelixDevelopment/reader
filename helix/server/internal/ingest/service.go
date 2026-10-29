// Package ingest scans connectors into the work/edition/file catalog.
package ingest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/helixreader/helix/core/formats"
	"github.com/helixreader/helix/core/pkg/hdm"
	"github.com/helixreader/helix/core/pkg/parse/epub"
	"github.com/helixreader/helix/core/pkg/parse/fb2"
	"github.com/helixreader/helix/core/pkg/parse/mobi"
	"github.com/helixreader/helix/core/pkg/parse/txt"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/identify"
	"github.com/helixreader/helix/server/internal/sources"
)

// Service performs streaming hash/discovery and bounded metadata extraction.
type Service struct {
	Repository    catalog.Repository
	Identifier    *identify.Pipeline
	MaxParseBytes int64
	Now           func() time.Time

	reconcileOnce  sync.Once
	reconcileLocks [64]chan struct{}
}

// Report is a complete per-file account; skipped files are never silent.
type Report struct {
	Works    int      `json:"works"`
	Files    int      `json:"files"`
	Removed  int      `json:"removed"`
	Skipped  []string `json:"skipped,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// SanitizeError renders err for administrator-facing scan reports and job
// records. Connector errors such as *url.Error can embed credentials in the
// request URL; userinfo, query and fragment are stripped while the operation,
// host and path context is preserved. Callers keep the original error value,
// so errors.Is/errors.As behavior upstream is unchanged.
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		sanitized := *urlErr
		if u, perr := url.Parse(urlErr.URL); perr == nil {
			u.User = nil
			u.RawQuery = ""
			u.ForceQuery = false
			u.Fragment = ""
			if u.Opaque != "" && (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) {
				sanitized.URL = u.Scheme + ":<redacted>"
			} else {
				sanitized.URL = u.String()
			}
		} else {
			sanitized.URL = "<redacted>"
		}
		// A wrapped cause may itself be a *url.Error carrying credentials.
		sanitized.Err = errors.New(SanitizeError(urlErr.Err))
		return sanitized.Error()
	}
	return err.Error()
}

// SanitizeLocation renders a connector entry path for public scan reports.
// Absolute URLs (for example presigned OPDS acquisition links) are stripped
// of userinfo, query and fragment; ordinary local paths, including file
// names containing '?' or '#', are returned unchanged.
func SanitizeLocation(path string) string {
	u, err := url.Parse(path)
	httpScheme := strings.HasPrefix(strings.ToLower(path), "http:") || strings.HasPrefix(strings.ToLower(path), "https:")
	if err != nil {
		if httpScheme {
			return "<redacted-url>"
		}
		return path
	}
	if !u.IsAbs() {
		return path
	}
	if u.Opaque != "" && (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) {
		return u.Scheme + ":<redacted>"
	}
	if u.Host == "" {
		return path
	}
	u.User = nil
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	return u.String()
}

func (s *Service) clock() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) Scan(ctx context.Context, connector sources.Connector, prefix string) (Report, error) {
	return s.ScanSource(ctx, "", connector, prefix)
}

// lockReconcile serializes complete scans of the same source. Without this
// boundary an older listing could prune a file merged by a newer concurrent
// scan. Fixed lock stripes avoid retaining deleted source IDs indefinitely;
// collisions merely serialize unrelated scans. Waiting observes cancellation.
func (s *Service) lockReconcile(ctx context.Context, sourceID string) (func(), error) {
	s.reconcileOnce.Do(func() {
		for i := range s.reconcileLocks {
			s.reconcileLocks[i] = make(chan struct{}, 1)
		}
	})
	h := fnv.New32a()
	_, _ = h.Write([]byte(sourceID))
	lock := s.reconcileLocks[h.Sum32()%uint32(len(s.reconcileLocks))]
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Service) ScanSource(ctx context.Context, sourceID string, connector sources.Connector, prefix string) (Report, error) {
	reconcile := sourceID != "" && prefix == "" && connector.Capabilities().RecursiveList
	if reconcile {
		unlock, err := s.lockReconcile(ctx, sourceID)
		if err != nil {
			return Report{}, err
		}
		defer unlock()
	}
	entries, err := connector.List(ctx, prefix)
	if err != nil {
		return Report{}, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	report := Report{}
	present := make([]string, 0, len(entries))
	for _, entry := range entries {
		// Presence, not ingest success, is retention evidence. A transient open,
		// parse, identification, or repository error must not delete the prior
		// catalog reference when the source still lists the file.
		present = append(present, entry.Path)
	}
	touchedWorks := map[string]bool{}
	max := s.MaxParseBytes
	if max <= 0 {
		max = 64 << 20
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		d, ok := formats.Detect(entry.Path)
		if !ok {
			if u, err := url.Parse(entry.Path); err == nil && u.Path != entry.Path {
				d, ok = formats.Detect(u.Path)
			}
		}
		if !ok {
			d, ok = formats.Detect(entry.Name)
		}
		if !ok {
			report.Skipped = append(report.Skipped, SanitizeLocation(entry.Path)+": unrecognized publication extension")
			continue
		}
		if d.Format == formats.Format("lcpl") || d.Format == formats.Format("zip") {
			// Licenses and ambiguous ZIP containers are detected but not exposed
			// as independent books without a publication manifest.
			report.Skipped = append(report.Skipped, SanitizeLocation(entry.Path)+": auxiliary/container resource retained for package resolution")
			continue
		}
		work, warnings, err := s.ingestOne(ctx, sourceID, connector, entry, d, max)
		report.Warnings = append(report.Warnings, warnings...)
		if err != nil {
			report.Skipped = append(report.Skipped, SanitizeLocation(entry.Path)+": "+SanitizeError(err))
			continue
		}
		if err := s.Repository.MergeWork(ctx, work); err != nil {
			report.Skipped = append(report.Skipped, SanitizeLocation(entry.Path)+": "+SanitizeError(err))
			continue
		}
		report.Files++
		touchedWorks[work.ID] = true
	}
	// Works counts works touched by this scan, never the catalog total.
	report.Works = len(touchedWorks)
	if reconcile {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		removed, err := s.Repository.ReconcileSource(ctx, sourceID, present)
		if err != nil {
			return report, err
		}
		report.Removed = removed
	}
	return report, nil
}

func (s *Service) ingestOne(ctx context.Context, sourceID string, connector sources.Connector, entry sources.Entry, d formats.Descriptor, max int64) (catalog.Work, []string, error) {
	r, err := connector.Open(ctx, entry.Path)
	if err != nil {
		return catalog.Work{}, nil, err
	}
	h := sha256.New()
	n, copyErr := io.Copy(h, r)
	closeErr := r.Close()
	if copyErr != nil {
		return catalog.Work{}, nil, copyErr
	}
	if closeErr != nil {
		return catalog.Work{}, nil, closeErr
	}
	hash := hex.EncodeToString(h.Sum(nil))
	now := s.clock()
	title := strings.TrimSuffix(entry.Name, filepath.Ext(entry.Name))
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(entry.Path), filepath.Ext(entry.Path))
	}
	edition := catalog.Edition{Title: title, State: catalog.Pending, UpdatedAt: now}
	var sample string
	var warnings []string
	if n <= max && (d.Format == formats.EPUB || d.Format == formats.FB2 || d.Format == formats.MOBI || d.Format == formats.TXT) {
		data, err := s.readBounded(ctx, connector, entry.Path, max)
		if err != nil {
			warnings = append(warnings, SanitizeLocation(entry.Path)+": metadata read: "+SanitizeError(err))
		} else if doc, err := parseDocument(d.Format, data, title); err != nil {
			warnings = append(warnings, SanitizeLocation(entry.Path)+": metadata parse: "+SanitizeError(err))
		} else {
			applyHDM(&edition, doc)
			sample = textSample(doc, 32768)
		}
	}
	if s.Identifier != nil {
		result, err := s.Identifier.Resolve(ctx, identify.Snapshot{FileName: entry.Name, Embedded: edition, TextSample: sample, ContentSHA256: hash})
		if err != nil {
			warnings = append(warnings, SanitizeLocation(entry.Path)+": identification: "+SanitizeError(err))
			edition.State = catalog.NeedsReview
		} else {
			files := edition.Files
			edition = result.Edition
			edition.Files = files
			warnings = append(warnings, result.Errors...)
		}
	}
	if edition.State == catalog.Pending {
		edition.State = catalog.NeedsReview
	}
	if edition.ID == "" {
		edition.ID = identityID(edition, hash)
	}
	edition.UpdatedAt = now
	mime := ""
	if len(d.MIMETypes) > 0 {
		mime = d.MIMETypes[0]
	}
	edition.Files = []catalog.File{{ID: fileID(sourceID, entry.Path), SourceID: sourceID, Path: entry.Path, Format: string(d.Format), MIMEType: mime, Size: n, SHA256: hash, Protected: d.Protected, DRMScheme: drm(d), ModifiedAt: entry.Modified}}
	// Work identity is anchored to content, not mutable metadata: improving
	// embedded title/authors on a later scan must not fork the work, while
	// identical bytes at distinct locations join one work as separate files.
	workID := workIdentityID(edition, hash)
	work := catalog.Work{ID: workID, CanonicalTitle: edition.Title, Editions: []catalog.Edition{edition}, CreatedAt: now, UpdatedAt: now}
	return work, warnings, nil
}

func (s *Service) readBounded(ctx context.Context, connector sources.Connector, name string, max int64) ([]byte, error) {
	r, err := connector.Open(ctx, name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("source exceeds %d byte metadata limit", max)
	}
	return data, nil
}

func parseDocument(f formats.Format, data []byte, title string) (*hdm.Document, error) {
	switch f {
	case formats.EPUB:
		return epub.Parser{}.Parse(bytes.NewReader(data), int64(len(data)))
	case formats.FB2:
		return fb2.Parser{}.Parse(bytes.NewReader(data))
	case formats.MOBI:
		return mobi.Parser{}.Parse(bytes.NewReader(data), int64(len(data)))
	case formats.TXT:
		return txt.Parser{FallbackTitle: title}.Parse(bytes.NewReader(data))
	default:
		return nil, fmt.Errorf("no native metadata parser")
	}
}

func applyHDM(e *catalog.Edition, doc *hdm.Document) {
	if doc.Meta.Title != "" {
		e.Title = doc.Meta.Title
	}
	e.Authors = append([]string(nil), doc.Meta.Creators...)
	e.Language = doc.Meta.Language
	e.Publisher = doc.Meta.Publisher
	e.Published = doc.Meta.Date
	e.Description = doc.Meta.Description
	for scheme, value := range doc.Meta.Identifiers {
		e.Identifiers = append(e.Identifiers, catalog.Identifier{Scheme: scheme, Value: value, Source: "embedded", Confidence: 0.95})
	}
	sort.Slice(e.Identifiers, func(i, j int) bool {
		if e.Identifiers[i].Scheme != e.Identifiers[j].Scheme {
			return e.Identifiers[i].Scheme < e.Identifiers[j].Scheme
		}
		return e.Identifiers[i].Value < e.Identifiers[j].Value
	})
}

func textSample(doc *hdm.Document, limit int) string {
	var b strings.Builder
	var walk func([]*hdm.Chapter)
	walk = func(cs []*hdm.Chapter) {
		for _, c := range cs {
			for _, block := range c.Blocks {
				if b.Len() >= limit {
					return
				}
				b.WriteString(block.Text)
				b.WriteByte('\n')
			}
			walk(c.Children)
		}
	}
	walk(doc.Chapters)
	out := b.String()
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func stableID(kind string, parts ...string) string {
	h := sha256.New()
	h.Write([]byte(kind))
	for _, p := range parts {
		h.Write([]byte{0})
		h.Write([]byte(strings.ToLower(strings.Join(strings.Fields(p), " "))))
	}
	return kind + "-" + hex.EncodeToString(h.Sum(nil))[:24]
}

// fileID is stable per source/path identity. The content SHA-256 is retained
// separately on File.SHA256 so identical bytes at distinct locations never
// collide on the file primary key.
func fileID(sourceID, path string) string {
	// Paths are case-sensitive on supported local and remote sources. Unlike
	// bibliographic metadata, they must never be case-folded during identity
	// generation. Length prefixes also make the encoding unambiguous.
	h := sha256.New()
	h.Write([]byte("file"))
	for _, part := range []string{sourceID, path} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(part)))
		h.Write(size[:])
		h.Write([]byte(part))
	}
	return "file-" + hex.EncodeToString(h.Sum(nil))[:24]
}
func identityID(e catalog.Edition, contentHash string) string {
	if value, ok := isbnIdentity(e); ok {
		return stableID("edition", "isbn", value)
	}
	// Without an ISBN the only stable manifestation key is the byte stream
	// itself; mutable metadata (title, authors, publisher) must not fork
	// edition identity on rescan.
	return stableID("edition", contentHash)
}
func drm(d formats.Descriptor) string {
	if d.Protected {
		return "lcp"
	}
	return ""
}
