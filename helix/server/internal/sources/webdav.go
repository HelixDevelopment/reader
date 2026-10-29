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
	"strings"

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

type multistatus struct {
	Responses []struct {
		Href string `xml:"href"`
		Prop []struct {
			Length       int64  `xml:"prop>getcontentlength"`
			Modified     string `xml:"prop>getlastmodified"`
			ETag         string `xml:"prop>getetag"`
			ResourceType struct {
				Collection *struct{} `xml:"collection"`
			} `xml:"prop>resourcetype"`
		} `xml:"propstat"`
	} `xml:"response"`
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
		for _, p := range r.Prop {
			if p.ResourceType.Collection != nil {
				continue
			}
			name := strings.TrimPrefix(path.Clean(href.Path), path.Clean(w.http.base.Path))
			name = strings.TrimPrefix(name, "/")
			if name == "" || name == "." {
				continue
			}
			mod, _ := http.ParseTime(p.Modified)
			out = append(out, Entry{Path: name, Name: path.Base(name), Size: p.Length, Modified: mod, ETag: strings.Trim(p.ETag, `"`)})
			break
		}
	}
	return out, nil
}
