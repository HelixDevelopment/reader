package sources

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// WebDAV implements bounded PROPFIND discovery and GET downloads.
type WebDAV struct {
	http               *HTTP
	username, password string
}

func NewWebDAV(cfg Config) (Connector, error) {
	if cfg.Protocol == "webdav" && strings.HasPrefix(cfg.Endpoint, "https://") { /* scheme remains authoritative */
	}
	h, err := NewHTTP(cfg)
	if err != nil {
		return nil, err
	}
	return &WebDAV{http: h.(*HTTP), username: cfg.Username, password: cfg.Password}, nil
}
func (w *WebDAV) Capabilities() Capabilities {
	return Capabilities{RecursiveList: true, RangeRead: true, Native: true}
}
func (w *WebDAV) Close() error { return w.http.Close() }
func (w *WebDAV) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	return w.http.Open(ctx, name)
}

// OpenRange serves a bounded byte window. The window is enforced locally so a
// server that ignores Range (full 200 at offset 0) can never over-deliver.
func (w *WebDAV) OpenRange(ctx context.Context, name string, offset, length int64) (io.ReadCloser, error) {
	return w.OpenRangeSized(ctx, name, offset, length, -1)
}

// contentRangeRE matches a satisfied byte Content-Range: "bytes first-last/total".
var contentRangeRE = regexp.MustCompile(`^bytes ([0-9]{1,18})-([0-9]{1,18})/([0-9]{1,18})$`)

// OpenRangeSized is OpenRange plus the revalidated total size (-1 = unknown).
// A 206 must carry a Content-Range naming exactly the requested window (and,
// when size is known, the same complete length); anything else is refused so a
// different or replaced representation is never relabelled as the requested one.
func (w *WebDAV) OpenRangeSized(ctx context.Context, name string, offset, length, size int64) (io.ReadCloser, error) {
	resp, err := w.http.rangeResponse(ctx, name, offset, length)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusPartialContent {
		if err := checkContentRange(resp.Header.Get("Content-Range"), offset, length, size); err != nil {
			resp.Body.Close()
			return nil, err
		}
	}
	if length < 0 {
		return resp.Body, nil
	}
	return &limitedReadCloser{Reader: io.LimitReader(resp.Body, length), Closer: resp.Body}, nil
}

func checkContentRange(value string, offset, length, size int64) error {
	m := contentRangeRE.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return fmt.Errorf("source: 206 without a valid Content-Range")
	}
	first, _ := strconv.ParseInt(m[1], 10, 64)
	last, _ := strconv.ParseInt(m[2], 10, 64)
	total, _ := strconv.ParseInt(m[3], 10, 64)
	if first != offset || (length >= 0 && last != offset+length-1) || last < first {
		return fmt.Errorf("source: 206 Content-Range does not match the requested window")
	}
	if size >= 0 && total != size {
		return fmt.Errorf("source: 206 complete length differs from the revalidated size")
	}
	return nil
}

type limitedReadCloser struct {
	io.Reader
	io.Closer
}

// Revalidate reports the live size and modification time with a Depth: 0
// PROPFIND, so content serving can decide whether the scan-time validators and
// byte ranges are still safe to offer for this file.
func (w *WebDAV) Revalidate(ctx context.Context, name string) (int64, time.Time, error) {
	u, err := w.http.resource(name)
	if err != nil {
		return -1, time.Time{}, err
	}
	body := []byte(`<?xml version="1.0"?><propfind xmlns="DAV:"><prop><getcontentlength/><getlastmodified/><resourcetype/></prop></propfind>`)
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", u.String(), bytes.NewReader(body))
	if err != nil {
		return -1, time.Time{}, err
	}
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml")
	for k, v := range w.http.headers {
		req.Header.Set(k, v)
	}
	if w.username != "" {
		req.SetBasicAuth(w.username, w.password)
	}
	resp, err := w.http.client.Do(req)
	if err != nil {
		return -1, time.Time{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return -1, time.Time{}, ErrNotFound
	}
	if resp.StatusCode != 207 {
		return -1, time.Time{}, fmt.Errorf("WebDAV PROPFIND: HTTP %d", resp.StatusCode)
	}
	payload, err := parseutil.ReadAll(resp.Body, 1<<20)
	if err != nil {
		return -1, time.Time{}, err
	}
	var ms multistatus
	if err := xml.Unmarshal(payload, &ms); err != nil {
		return -1, time.Time{}, err
	}
	for _, r := range ms.Responses {
		pr := r.merged()
		if pr.collection {
			return -1, time.Time{}, fmt.Errorf("source: not a regular file")
		}
		// A server that omits either property cannot revalidate: report
		// unknown (size -1) so the route serves a plain 200 without ranges.
		// Absent is not zero: a zero here would make the route serve an empty body.
		mod, err := http.ParseTime(pr.modified)
		if pr.length == "" || pr.modified == "" || err != nil {
			return -1, time.Time{}, nil
		}
		n, ok := parseContentLength(pr.length)
		if !ok {
			return -1, time.Time{}, nil
		}
		return n, mod, nil
	}
	return -1, time.Time{}, nil
}

type propstat struct {
	Length       string `xml:"prop>getcontentlength"`
	Status       string `xml:"status"`
	Modified     string `xml:"prop>getlastmodified"`
	ETag         string `xml:"prop>getetag"`
	ResourceType struct {
		Collection *struct{} `xml:"collection"`
	} `xml:"prop>resourcetype"`
}

type multistatusResponse struct {
	Href string     `xml:"href"`
	Prop []propstat `xml:"propstat"`
}

type multistatus struct {
	Responses []multistatusResponse `xml:"response"`
}

// parseContentLength accepts only a plain non-negative decimal that fits in an
// int64: no sign, hex, fraction, separator, empty value or overflow.
func parseContentLength(v string) (int64, bool) {
	if v == "" {
		return 0, false
	}
	for i := 0; i < len(v); i++ {
		if v[i] < '0' || v[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(v, 10, 64)
	return n, err == nil
}

type mergedProps struct {
	collection             bool
	length, modified, etag string
}

// merged folds every successful propstat of one response into a single view,
// whatever order the server lists them in (a 404 propstat for the properties
// a resource lacks may come before or after the 200 one).
func (r multistatusResponse) merged() mergedProps {
	var m mergedProps
	for _, p := range r.Prop {
		if st := strings.Fields(p.Status); len(st) >= 2 && !strings.HasPrefix(st[1], "2") {
			continue
		}
		if p.ResourceType.Collection != nil {
			m.collection = true
		}
		if v := strings.TrimSpace(p.Length); v != "" {
			m.length = v
		}
		if v := strings.TrimSpace(p.Modified); v != "" {
			m.modified = v
		}
		if v := strings.TrimSpace(p.ETag); v != "" {
			m.etag = v
		}
	}
	return m
}

func (w *WebDAV) List(ctx context.Context, prefix string) ([]Entry, error) {
	u, err := w.http.resource(prefix)
	if err != nil {
		return nil, err
	}
	body := []byte(`<?xml version="1.0"?><propfind xmlns="DAV:"><prop><displayname/><getcontentlength/><getlastmodified/><getetag/><resourcetype/></prop></propfind>`)
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "infinity")
	req.Header.Set("Content-Type", "application/xml")
	// Configured headers and credentials apply only to the configured origin;
	// the shared client's redirect policy strips them on cross-origin hops.
	if u.Scheme == w.http.base.Scheme && u.Host == w.http.base.Host {
		for k, v := range w.http.headers {
			req.Header.Set(k, v)
		}
		if w.username != "" {
			req.SetBasicAuth(w.username, w.password)
		}
	}
	resp, err := w.http.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 207 {
		return nil, fmt.Errorf("WebDAV PROPFIND: HTTP %d", resp.StatusCode)
	}
	payload, err := parseutil.ReadAll(resp.Body, 16<<20)
	if err != nil {
		return nil, err
	}
	var ms multistatus
	if err := xml.Unmarshal(payload, &ms); err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, r := range ms.Responses {
		href, err := url.Parse(r.Href)
		if err != nil {
			continue
		}
		// Judge the whole response: a collection may answer with a 200 propstat
		// naming it a collection plus a 404 propstat for properties it lacks.
		pr := r.merged()
		if pr.collection {
			continue
		}
		name := strings.TrimPrefix(path.Clean(href.Path), path.Clean(w.http.base.Path))
		name = strings.TrimPrefix(name, "/")
		if name == "" || name == "." {
			continue
		}
		mod, _ := http.ParseTime(pr.modified)
		size, ok := parseContentLength(pr.length)
		if !ok {
			continue // no trustworthy size: never catalogue an invented one
		}
		out = append(out, Entry{Path: name, Name: path.Base(name), Size: size, Modified: mod, ETag: strings.Trim(pr.etag, `"`)})
	}
	return out, nil
}
