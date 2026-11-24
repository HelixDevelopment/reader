package liveverify

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
)

// Resp is one fully read response. The body is already content-decoded the way
// the Python requests library decodes it (gzip, deflate, br) while the
// Content-Encoding header stays visible so negotiation can be asserted.
type Resp struct {
	Status int
	Header http.Header
	Body   []byte
}

func (r *Resp) Text() string { return string(r.Body) }

// JSON decodes the body into a generic value and aborts the run when it is not JSON.
func (r *Resp) JSON() any {
	var v any
	if err := json.Unmarshal(r.Body, &v); err != nil {
		panic(assertion{"response is not JSON: " + err.Error()})
	}
	return v
}

// Opts are per-request options.
type Opts struct {
	Header  map[string]string
	Query   url.Values
	JSON    any
	Basic   *[2]string
	Timeout time.Duration
}

// Client is a no-proxy, no-redirect-surprise HTTP client mirroring the Python
// session (trust_env False). It sends "Accept-Encoding: gzip, deflate" by
// default like requests, and decodes what comes back.
type Client struct{ hc *http.Client }

func NewClient() *Client {
	tr := &http.Transport{Proxy: nil, DisableCompression: true, MaxIdleConnsPerHost: 4}
	return &Client{hc: &http.Client{Transport: tr}}
}

func (c *Client) Close() { c.hc.CloseIdleConnections() }

func (c *Client) Do(method, rawurl string, o Opts) *Resp {
	r, err := c.TryDo(method, rawurl, o)
	if err != nil {
		panic(err)
	}
	return r
}

// TryDo returns the transport error instead of aborting (readiness polling).
func (c *Client) TryDo(method, rawurl string, o Opts) (*Resp, error) {
	if len(o.Query) > 0 {
		rawurl += "?" + o.Query.Encode()
	}
	var body io.Reader
	if o.JSON != nil {
		raw, err := json.Marshal(o.JSON)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, rawurl, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	if o.JSON != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range o.Header {
		req.Header.Set(k, v)
	}
	if o.Basic != nil {
		req.SetBasicAuth(o.Basic[0], o.Basic[1])
	}
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if method != http.MethodHead {
		raw, err = decodeBody(res.Header.Get("Content-Encoding"), raw)
		if err != nil {
			return nil, err
		}
	}
	return &Resp{Status: res.StatusCode, Header: res.Header, Body: raw}, nil
}

func decodeBody(encoding string, raw []byte) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "identity":
		return raw, nil
	case "gzip":
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("gzip body: %w", err)
		}
		return io.ReadAll(zr)
	case "deflate":
		return io.ReadAll(flate.NewReader(bytes.NewReader(raw)))
	case "br":
		return io.ReadAll(brotli.NewReader(bytes.NewReader(raw)))
	}
	return nil, fmt.Errorf("unsupported Content-Encoding %q", encoding)
}

// RaiseForStatus aborts when the status is a client or server error.
func (r *Resp) RaiseForStatus() *Resp {
	if r.Status >= 400 {
		panic(assertion{fmt.Sprintf("HTTP %d", r.Status)})
	}
	return r
}

// --- generic JSON navigation -------------------------------------------------

// J walks a decoded JSON value by string keys and int indexes. Any miss aborts
// the run (the Python KeyError/IndexError), never returns a silent zero value.
func J(v any, path ...any) any {
	for _, p := range path {
		switch k := p.(type) {
		case string:
			m, ok := v.(map[string]any)
			if !ok {
				panic(assertion{fmt.Sprintf("JSON: expected object at %q", k)})
			}
			next, ok := m[k]
			if !ok {
				panic(assertion{fmt.Sprintf("JSON: missing key %q", k)})
			}
			v = next
		case int:
			a, ok := v.([]any)
			if !ok || k >= len(a) || k < 0 {
				panic(assertion{fmt.Sprintf("JSON: no element %d", k)})
			}
			v = a[k]
		}
	}
	return v
}

// JList returns the array at path, or nil when the key is absent (Python .get(k, [])).
func JList(v any, key string) []any {
	m, ok := v.(map[string]any)
	if !ok {
		panic(assertion{"JSON: expected object for " + key})
	}
	a, _ := m[key].([]any)
	return a
}

// JStr is J asserting a string.
func JStr(v any, path ...any) string {
	s, ok := J(v, path...).(string)
	if !ok {
		panic(assertion{fmt.Sprintf("JSON: %v is not a string", path)})
	}
	return s
}

// JNum is J asserting a number.
func JNum(v any, path ...any) float64 {
	n, ok := J(v, path...).(float64)
	if !ok {
		panic(assertion{fmt.Sprintf("JSON: %v is not a number", path)})
	}
	return n
}

// jget is a tolerant lookup (Python dict.get): ok=false when absent.
func jget(v any, key string) (any, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	x, ok := m[key]
	return x, ok
}

func jstrOr(v any, key string) string {
	x, _ := jget(v, key)
	s, _ := x.(string)
	return s
}
