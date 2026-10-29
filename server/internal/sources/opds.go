package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// OPDS adapts OPDS 1.2 and 2.0 acquisition feeds into downloadable entries.
// Signed acquisition URLs are process-local connector state. Catalog entries
// contain only their credential-free public form, never query/userinfo secrets.
type OPDS struct {
	http         *HTTP
	mu           sync.RWMutex
	acquisitions map[string]string
}

func NewOPDS(cfg Config) (Connector, error) {
	h, err := NewHTTP(cfg)
	if err != nil {
		return nil, err
	}
	inner := h.(*HTTP)
	// Acquisition downloads may follow cross-origin redirects: each hop is
	// validated by the dialer and credentials/headers are stripped whenever
	// the origin changes.
	inner.allowCrossOriginRedirects = true
	return &OPDS{http: inner, acquisitions: map[string]string{}}, nil
}
func (o *OPDS) Capabilities() Capabilities { return Capabilities{Native: true, RangeRead: true} }
func (o *OPDS) Close() error {
	o.mu.Lock()
	o.acquisitions = map[string]string{}
	o.mu.Unlock()
	return o.http.Close()
}
func (o *OPDS) acquisition(name string) string {
	o.mu.RLock()
	raw, ok := o.acquisitions[name]
	o.mu.RUnlock()
	if ok {
		return raw
	}
	return name
}
func (o *OPDS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	return o.http.openAcquisition(ctx, o.acquisition(name))
}
func (o *OPDS) OpenRange(ctx context.Context, name string, offset, length int64) (io.ReadCloser, error) {
	return o.http.openAcquisitionRange(ctx, o.acquisition(name), offset, length)
}

type feedLink struct {
	HREF string
	Type string
	Rel  any
}
type opdsFeed struct {
	Entries []opdsEntry
	Links   []feedLink
}
type opdsEntry struct {
	Title string
	Links []feedLink
}

func relString(rel any) string {
	switch v := rel.(type) {
	case string:
		return v
	case []any:
		for _, r := range v {
			if s := relString(r); s != "" {
				return s
			}
		}
	}
	return ""
}
func hasRel(rel any, want string) bool {
	switch v := rel.(type) {
	case string:
		return v == want
	case []any:
		for _, r := range v {
			if hasRel(r, want) {
				return true
			}
		}
	}
	return false
}
func isAcquisition2(rel any) bool {
	switch v := rel.(type) {
	case string:
		return v == "acquisition" || v == "download" || strings.HasPrefix(v, "http://opds-spec.org/acquisition")
	case []any:
		for _, r := range v {
			if isAcquisition2(r) {
				return true
			}
		}
	}
	return false
}
func parseFeed(data io.Reader, jsonFeed bool) (opdsFeed, error) {
	payload, err := parseutil.ReadAll(data, 32<<20)
	if err != nil {
		return opdsFeed{}, err
	}
	if jsonFeed {
		var raw struct {
			Publications []struct {
				Metadata struct {
					Title string `json:"title"`
				} `json:"metadata"`
				Links []feedLink `json:"links"`
			} `json:"publications"`
			Links []feedLink `json:"links"`
		}
		if err := json.Unmarshal(payload, &raw); err != nil {
			return opdsFeed{}, err
		}
		f := opdsFeed{Links: raw.Links}
		for _, p := range raw.Publications {
			f.Entries = append(f.Entries, opdsEntry{Title: p.Metadata.Title, Links: p.Links})
		}
		return f, nil
	}
	var raw struct {
		Entries []struct {
			Title string `xml:"title"`
			Links []struct {
				Rel  string `xml:"rel,attr"`
				HREF string `xml:"href,attr"`
				Type string `xml:"type,attr"`
			} `xml:"link"`
		} `xml:"entry"`
		Links []struct {
			Rel  string `xml:"rel,attr"`
			HREF string `xml:"href,attr"`
			Type string `xml:"type,attr"`
		} `xml:"link"`
	}
	if err := xml.Unmarshal(payload, &raw); err != nil {
		return opdsFeed{}, err
	}
	f := opdsFeed{}
	for _, l := range raw.Links {
		f.Links = append(f.Links, feedLink{HREF: l.HREF, Type: l.Type, Rel: l.Rel})
	}
	for _, e := range raw.Entries {
		entry := opdsEntry{Title: e.Title}
		for _, l := range e.Links {
			entry.Links = append(entry.Links, feedLink{HREF: l.HREF, Type: l.Type, Rel: l.Rel})
		}
		f.Entries = append(f.Entries, entry)
	}
	return f, nil
}

type opdsAcquisition struct {
	raw  string
	base string
	name string
}

func (o *OPDS) List(ctx context.Context, prefix string) ([]Entry, error) {
	seenPages := map[string]bool{}
	previousKeys := map[string]string{}
	o.mu.RLock()
	for key, raw := range o.acquisitions {
		previousKeys[raw] = key
	}
	o.mu.RUnlock()
	// Gather a complete generation off to the side. Failed listings must not
	// partially replace usable acquisition state, and successful listings prune
	// signed URLs that no longer appear in the feed.
	candidates := []opdsAcquisition{}
	next := prefix
	first := true
	for pages := 0; (next != "" || first) && pages < 100; pages++ {
		first = false
		resp, err := o.http.getAcquisition(ctx, next, "")
		if err != nil {
			return nil, err
		}
		base := resp.Request.URL
		feed, err := parseFeed(resp.Body, strings.Contains(resp.Header.Get("Content-Type"), "json"))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		for _, entry := range feed.Entries {
			for _, link := range entry.Links {
				acquire := hasRel(link.Rel, "http://opds-spec.org/acquisition") || strings.HasPrefix(relString(link.Rel), "http://opds-spec.org/acquisition/") || isAcquisition2(link.Rel)
				if !acquire {
					continue
				}
				u, err := base.Parse(link.HREF)
				if err != nil || u.Scheme == "" || u.User != nil {
					continue
				}
				name := path.Base(u.Path)
				if name == "." || name == "/" || name == "" {
					name = entry.Title
				}
				raw := u.String()
				public := *u
				public.User = nil
				public.RawQuery = ""
				public.ForceQuery = false
				public.Fragment = ""
				candidates = append(candidates, opdsAcquisition{raw: raw, base: public.String(), name: name})
				if len(candidates) >= 100000 {
					return nil, fmt.Errorf("OPDS feed entry limit reached")
				}
			}
		}
		next = ""
		for _, link := range feed.Links {
			if !hasRel(link.Rel, "next") {
				continue
			}
			u, err := base.Parse(link.HREF)
			if err != nil {
				continue
			}
			if u.Scheme != o.http.base.Scheme || u.Host != o.http.base.Host {
				return nil, fmt.Errorf("OPDS cross-origin pagination rejected")
			}
			if !seenPages[u.String()] {
				next = u.String()
				seenPages[next] = true
			}
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	if next != "" {
		// A capped listing is not a complete source snapshot and must never be
		// presented as one to reconciliation or callers.
		return nil, fmt.Errorf("OPDS feed page limit reached")
	}
	groups := map[string]map[string]struct{}{}
	for _, candidate := range candidates {
		if groups[candidate.base] == nil {
			groups[candidate.base] = map[string]struct{}{}
		}
		groups[candidate.base][candidate.raw] = struct{}{}
	}
	out := make([]Entry, 0, len(candidates))
	seenFiles := map[string]bool{}
	acquisitions := map[string]string{}
	for _, candidate := range candidates {
		key := candidate.base
		previous, seenBefore := previousKeys[candidate.raw]
		validPrevious := seenBefore && (previous == candidate.base || strings.HasPrefix(previous, candidate.base+"#helix-x-"))
		if validPrevious {
			// Preserve identity when colliding siblings arrive or disappear.
			key = previous
		} else if len(groups[candidate.base]) > 1 {
			// Fragment every distinct member of a new colliding set, not merely
			// the later ones, making first-generation assignment order-independent.
			digest := sha256.Sum256([]byte(candidate.raw))
			key += "#helix-x-" + hex.EncodeToString(digest[:])
		}
		if previous, exists := acquisitions[key]; exists && previous != candidate.raw {
			return nil, fmt.Errorf("OPDS acquisition identity collision")
		}
		acquisitions[key] = candidate.raw
		if !seenFiles[key] {
			out = append(out, Entry{Path: key, Name: candidate.name, Size: -1})
			seenFiles[key] = true
		}
	}
	o.mu.Lock()
	o.acquisitions = acquisitions
	o.mu.Unlock()
	return out, nil
}
