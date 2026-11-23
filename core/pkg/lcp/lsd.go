package lcp

// Client side of the Readium License Status Document (LSD) 1.0.
//
// Primary sources (read for this file, not recalled), both from
// https://github.com/readium/lcp-specs at commit
// 812dd91d6b90087e657a5c94d04e6f16abd8873b:
//
//	SPEC   releases/lsd/latest.md      (revision 5)
//	SCHEMA schema/status.schema.json
//
// Revision 5 is followed where it differs from revision 4 (lsd-1-0-4.md):
// after a return or renew the client must process the updated Status Document
// and download the updated License Document (latest.md l.326, l.383).
//
// This package never logs or formats a device id or name into an error. The
// device identity travels only in the query string of the request the spec
// prescribes, and transport errors are stripped of their URL.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// lsdMediaType is the Status Document media type (SPEC l.135).
	lsdMediaType = "application/vnd.readium.license.status.v1.0+json"
	// lsdMaxBody bounds every response we read. Status Documents are small;
	// the bound is ours, not the spec's.
	lsdMaxBody = 1 << 20
)

var (
	// ErrStatusDocument: the bytes are not a valid Status Document (SCHEMA,
	// SPEC section 2).
	ErrStatusDocument = errors.New("lcp: invalid License Status Document")
	// ErrLSDNoLink: the Status Document offers no link for the interaction.
	ErrLSDNoLink = errors.New("lcp: status document has no usable link for this interaction")
	// ErrLSDDevice: registration needs both a device id and a name (SPEC l.275-276).
	ErrLSDDevice = errors.New("lcp: device id and name are required to register")
	// ErrLSDUnexpectedState: the provider answered 200 with a document that
	// contradicts what the interaction must produce (SPEC l.282, l.326), or
	// that belongs to another license.
	// ErrLSDBadTemplate: a non-templated link href contains a URI template
	// expression; templated defaults to false (SPEC l.182), so the href is a
	// literal URI and sending it unexpanded would be wrong.
	ErrLSDBadTemplate     = errors.New("lcp: link href holds a template expression but is not marked templated")
	ErrLSDUnexpectedState = errors.New("lcp: provider response contradicts the interaction")
)

// StatusValue is the controlled vocabulary of SPEC l.156-161 / SCHEMA l.14-21.
type StatusValue string

const (
	StatusReady     StatusValue = "ready"
	StatusActive    StatusValue = "active"
	StatusRevoked   StatusValue = "revoked"
	StatusReturned  StatusValue = "returned"
	StatusCancelled StatusValue = "cancelled"
	StatusExpired   StatusValue = "expired"
)

func (s StatusValue) valid() bool {
	switch s {
	case StatusReady, StatusActive, StatusRevoked, StatusReturned, StatusCancelled, StatusExpired:
		return true
	}
	return false
}

// LSDLink is a Link Object of a Status Document (SPEC l.173-183, link.schema.json).
// rel may be a string or an array; both land in Rels.
type LSDLink struct {
	Rels      []string
	Href      string
	Type      string
	Title     string
	Profile   string
	Templated bool
}

// HasRel reports whether the link carries the relation.
func (l LSDLink) HasRel(rel string) bool {
	for _, r := range l.Rels {
		if r == rel {
			return true
		}
	}
	return false
}

func (l LSDLink) isHTML() bool {
	mt, _, err := mime.ParseMediaType(l.Type)
	return err == nil && mt == "text/html"
}

// StatusEvent is one entry of `events` (SPEC l.203-225, SCHEMA l.63-94).
type StatusEvent struct {
	Type      string
	Name      string
	ID        string
	Timestamp time.Time
}

// StatusDocument is a validated Status Document.
type StatusDocument struct {
	ID                 string
	Status             StatusValue
	Message            string
	UpdatedLicense     time.Time
	UpdatedStatus      time.Time
	Links              []LSDLink
	PotentialRightsEnd *time.Time
	Events             []StatusEvent
}

// LicenseUsable reports whether the status still permits opening the
// publication: ready and active (SPEC l.156-161). It is a hint from the
// Status Document only: when it contradicts an up-to-date License Document the
// License Document takes precedence (SPEC l.250).
func (d *StatusDocument) LicenseUsable() bool {
	return d.Status == StatusReady || d.Status == StatusActive
}

// Link returns the first link carrying the relation.
func (d *StatusDocument) Link(rel string) (LSDLink, bool) {
	for _, l := range d.Links {
		if l.HasRel(rel) {
			return l, true
		}
	}
	return LSDLink{}, false
}

type wireStatusLink struct {
	Href      *string         `json:"href"`
	Rel       json.RawMessage `json:"rel"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Profile   string          `json:"profile"`
	Templated bool            `json:"templated"`
}

type wireStatusEvent struct {
	Type      *string `json:"type"`
	Name      string  `json:"name"`
	ID        string  `json:"id"`
	Timestamp *string `json:"timestamp"`
}

type wireStatus struct {
	ID      *string `json:"id"`
	Status  *string `json:"status"`
	Message *string `json:"message"`
	Updated *struct {
		License *string `json:"license"`
		Status  *string `json:"status"`
	} `json:"updated"`
	Links           *[]wireStatusLink `json:"links"`
	PotentialRights *struct {
		End *string `json:"end"`
	} `json:"potential_rights"`
	Events []wireStatusEvent `json:"events"`
}

func statusErr(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrStatusDocument, fmt.Sprintf(format, a...))
}

func parseStatusTime(s, field string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, statusErr("%s is not an ISO 8601 date-time", field)
	}
	return t, nil
}

var statusEventTypes = map[string]bool{"register": true, "renew": true, "return": true, "revoke": true, "cancel": true}

// ParseStatusDocument validates doc against SCHEMA: required id, status,
// message, updated.{license,status}, links (SCHEMA l.96-102); status in the
// enumeration (l.14-21); at least one link with rel "license" (SCHEMA
// l.103-137, SPEC l.185); every link has href and rel (SPEC l.178-179);
// event types in the enumeration (SCHEMA l.71-77); date-times parse.
// Unknown members are ignored (forward compatibility).
func ParseStatusDocument(doc []byte) (*StatusDocument, error) {
	dec := json.NewDecoder(bytes.NewReader(doc))
	var w wireStatus
	if err := dec.Decode(&w); err != nil {
		return nil, statusErr("not a JSON object: %v", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) { // SPEC l.127: a single JSON object
		return nil, statusErr("trailing data after the JSON object")
	}
	if w.ID == nil || *w.ID == "" {
		return nil, statusErr("id is required")
	}
	if w.Status == nil {
		return nil, statusErr("status is required")
	}
	if w.Message == nil {
		return nil, statusErr("message is required")
	}
	if w.Updated == nil || w.Updated.License == nil || w.Updated.Status == nil {
		return nil, statusErr("updated.license and updated.status are required")
	}
	if w.Links == nil {
		return nil, statusErr("links is required")
	}
	d := &StatusDocument{ID: *w.ID, Status: StatusValue(*w.Status), Message: *w.Message}
	if !d.Status.valid() {
		return nil, statusErr("status %q is not in the controlled vocabulary", *w.Status)
	}
	var err error
	if d.UpdatedLicense, err = parseStatusTime(*w.Updated.License, "updated.license"); err != nil {
		return nil, err
	}
	if d.UpdatedStatus, err = parseStatusTime(*w.Updated.Status, "updated.status"); err != nil {
		return nil, err
	}
	hasLicense := false
	for i, wl := range *w.Links {
		if wl.Href == nil || *wl.Href == "" {
			return nil, statusErr("links[%d].href is required", i)
		}
		rels, err := parseRels(wl.Rel)
		if err != nil {
			return nil, statusErr("links[%d].rel: %v", i, err)
		}
		l := LSDLink{Rels: rels, Href: *wl.Href, Type: wl.Type, Title: wl.Title, Profile: wl.Profile, Templated: wl.Templated}
		hasLicense = hasLicense || l.HasRel("license")
		d.Links = append(d.Links, l)
	}
	if !hasLicense {
		return nil, statusErr("no link with relation license")
	}
	if w.PotentialRights != nil && w.PotentialRights.End != nil {
		t, err := parseStatusTime(*w.PotentialRights.End, "potential_rights.end")
		if err != nil {
			return nil, err
		}
		d.PotentialRightsEnd = &t
	}
	// SCHEMA l.63-94: event items declare no `required`, so a missing type or
	// timestamp is legal and never fatal; a PRESENT one must be valid.
	for i, we := range w.Events {
		ev := StatusEvent{Name: we.Name, ID: we.ID}
		if we.Type != nil {
			if !statusEventTypes[*we.Type] {
				return nil, statusErr("events[%d].type %q is not in the enumeration", i, *we.Type)
			}
			ev.Type = *we.Type
		}
		if we.Timestamp != nil {
			t, err := parseStatusTime(*we.Timestamp, fmt.Sprintf("events[%d].timestamp", i))
			if err != nil {
				return nil, err
			}
			ev.Timestamp = t
		}
		d.Events = append(d.Events, ev)
	}
	return d, nil
}

func parseRels(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, errors.New("required")
	}
	var one string
	if json.Unmarshal(raw, &one) == nil {
		if one == "" {
			return nil, errors.New("empty")
		}
		return []string{one}, nil
	}
	var many []string // link.schema.json: an array of strings, no minItems
	if json.Unmarshal(raw, &many) != nil {
		return nil, errors.New("must be a string or an array of strings")
	}
	return many, nil
}

// NeedsLicenseRefresh implements SPEC l.244: when updated.license of the
// Status Document is more recent than the local License Document's timestamp
// the client must download the License Document again. A local License with no
// `updated` falls back to `issued` (its last known change). A nil license is
// always stale; a nil status says nothing.
func NeedsLicenseRefresh(status *StatusDocument, local *License) bool {
	if status == nil {
		return false
	}
	if local == nil {
		return true
	}
	have := local.Updated
	if have.IsZero() {
		have = local.Issued
	}
	return status.UpdatedLicense.After(have)
}

// SelectLicenseLink implements SPEC l.246-248. supported lists the profile
// URIs the caller can handle, ordered lowest to highest; the link with the
// highest supported profile wins and link order is irrelevant. A sole license
// link without a profile is accepted. If nothing is usable, ok is false and
// the client must consider that no updated license is available (SPEC l.248).
func (d *StatusDocument) SelectLicenseLink(supported []string) (link LSDLink, ok bool) {
	best, n := -1, 0
	var sole LSDLink
	for _, l := range d.Links {
		if !l.HasRel("license") {
			continue
		}
		n++
		sole = l
		for i, p := range supported {
			if l.Profile == p && i > best {
				best, link, ok = i, l, true
			}
		}
	}
	if !ok && n == 1 && sole.Profile == "" {
		return sole, true
	}
	return link, ok
}

// expandLinkTemplate expands the RFC 6570 form-style query expressions the LSD
// uses: {?a,b} and {&a,b} (SPEC l.97, l.303, l.350). Variables absent from
// vars are omitted; values are percent-encoded with unreserved characters left
// alone. Any other expression form is refused rather than guessed.
func expandLinkTemplate(href string, vars map[string]string) (string, error) {
	var out strings.Builder
	for {
		i := strings.IndexByte(href, '{')
		if i < 0 {
			out.WriteString(href)
			return out.String(), nil
		}
		out.WriteString(href[:i])
		j := strings.IndexByte(href[i:], '}')
		if j < 0 {
			return "", errors.New("lcp: unterminated URI template expression")
		}
		expr := href[i+1 : i+j]
		href = href[i+j+1:]
		if expr == "" || (expr[0] != '?' && expr[0] != '&') {
			return "", fmt.Errorf("lcp: unsupported URI template expression {%s}", expr)
		}
		sep := "?"
		if expr[0] == '&' || strings.Contains(out.String(), "?") {
			sep = "&"
		}
		for _, name := range strings.Split(expr[1:], ",") {
			if name == "" || strings.ContainsAny(name, "*:{}+#./;=!@|$") {
				return "", fmt.Errorf("lcp: unsupported URI template variable %q", name)
			}
			v, defined := vars[name]
			if !defined {
				continue
			}
			out.WriteString(sep + name + "=" + pctEncode(v))
			sep = "&"
		}
	}
}

func pctEncode(s string) string {
	const hexd = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hexd[c>>4])
			b.WriteByte(hexd[c&15])
		}
	}
	return b.String()
}

// LSDProblem is a failed interaction. When the body is an RFC 7807 Problem
// Details object (SPEC l.231-238: title and type are required of servers) its
// members are filled; otherwise only HTTPStatus is set.
type LSDProblem struct {
	HTTPStatus int
	Type       string
	Title      string
	Detail     string
	parsed     bool
}

// IsProblemDocument reports whether the response body was a Problem Details object.
func (p *LSDProblem) IsProblemDocument() bool { return p.parsed }

func (p *LSDProblem) Error() string {
	if !p.parsed {
		return fmt.Sprintf("lcp: license status server answered HTTP %d without a problem document", p.HTTPStatus)
	}
	s := fmt.Sprintf("lcp: license status server HTTP %d %s: %s", p.HTTPStatus, p.Type, p.Title)
	if p.Detail != "" {
		s += ": " + p.Detail
	}
	return s
}

func newLSDProblem(code int, body []byte) *LSDProblem {
	p := &LSDProblem{HTTPStatus: code}
	var w struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(body, &w) == nil && (w.Type != "" || w.Title != "") {
		p.Type, p.Title, p.Detail, p.parsed = clip(w.Type), clip(w.Title), clip(w.Detail), true
	}
	return p
}

func clip(s string) string {
	if len(s) > 512 {
		return s[:512]
	}
	return s
}

// LSDInteractiveRenewal is returned by Renew when the only renew link needs a
// human (text/html, SPEC l.368): the caller should open URL in a browser.
type LSDInteractiveRenewal struct{ URL string }

// httpURL parses target and requires an absolute http(s) URL with a host.
func httpURL(target string) (*url.URL, bool) {
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, false
	}
	return u, true
}

func (e *LSDInteractiveRenewal) Error() string {
	return "lcp: renewal requires the provider's web page"
}

// LSDClient talks to a Status Document provider. HTTP is injected (and so
// are TLS, proxies and timeouts); nil means a client with a 30 s timeout.
// DeviceID should be stable per device (SPEC l.264). Failures never block the
// publication: callers must keep the book readable when any call errors
// (SPEC l.252, l.261).
type LSDClient struct {
	HTTP           *http.Client
	DeviceID       string
	DeviceName     string
	AcceptLanguage string
}

func (c *LSDClient) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// clientFor returns a copy of the HTTP client that refuses to follow any
// redirect of a state-changing request: net/http would turn a 301/302/303
// POST/PUT into a GET that the provider never meant as a registration, return
// or renewal (and a followed redirect could carry the device identity query to
// another host). The 3xx is surfaced as an *LSDProblem. Reads follow the
// caller's own redirect policy. The caller's *http.Client is never mutated.
func (c *LSDClient) clientFor(method string) *http.Client {
	hc := *c.httpClient()
	if method == http.MethodGet {
		return &hc
	}
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &hc
}

func (c *LSDClient) do(ctx context.Context, method, target string) (*StatusDocument, error) {
	u, ok := httpURL(target)
	if !ok {
		return nil, errors.New("lcp: status link is not an absolute http(s) URL")
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, errors.New("lcp: cannot build status request")
	}
	req.Header.Set("Accept", lsdMediaType)
	if c.AcceptLanguage != "" { // SPEC l.147: messages localized by Accept-Language
		req.Header.Set("Accept-Language", c.AcceptLanguage)
	}
	resp, err := c.clientFor(method).Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) { // drop the URL: its query holds the device identity
			err = ue.Err
		}
		return nil, fmt.Errorf("lcp: %s status request failed: %w", method, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, lsdMaxBody+1))
	if err != nil {
		return nil, fmt.Errorf("lcp: reading status response: %w", err)
	}
	if len(body) > lsdMaxBody {
		return nil, statusErr("response exceeds %d bytes", lsdMaxBody)
	}
	if resp.StatusCode != http.StatusOK { // every success row of SPEC is 200
		return nil, newLSDProblem(resp.StatusCode, body)
	}
	return ParseStatusDocument(body)
}

// FetchStatus GETs a Status Document (SPEC l.190: the license/status link is GET;
// l.134: no authentication is required).
func (c *LSDClient) FetchStatus(ctx context.Context, statusURL string) (*StatusDocument, error) {
	return c.do(ctx, http.MethodGet, statusURL)
}

func programmaticLink(d *StatusDocument, rel string) (LSDLink, error) {
	var html *LSDLink
	for i, l := range d.Links {
		if !l.HasRel(rel) {
			continue
		}
		if l.isHTML() {
			html = &d.Links[i]
			continue
		}
		return l, nil
	}
	if html != nil && rel == "renew" {
		if _, ok := httpURL(html.Href); !ok { // never hand javascript:/file:/intent: to a caller that opens it
			return LSDLink{}, fmt.Errorf("%w: %s (html link is not an absolute http(s) URL)", ErrLSDNoLink, rel)
		}
		return LSDLink{}, &LSDInteractiveRenewal{URL: html.Href}
	}
	return LSDLink{}, fmt.Errorf("%w: %s", ErrLSDNoLink, rel)
}

func (c *LSDClient) interact(ctx context.Context, method string, l LSDLink, vars map[string]string) (*StatusDocument, error) {
	target := l.Href
	if !l.Templated && strings.ContainsAny(l.Href, "{}") {
		return nil, ErrLSDBadTemplate
	}
	if l.Templated {
		var err error
		if target, err = expandLinkTemplate(l.Href, vars); err != nil {
			return nil, err
		}
	}
	return c.do(ctx, method, target)
}

func (c *LSDClient) deviceVars(required bool) (map[string]string, error) {
	if required && (c.DeviceID == "" || c.DeviceName == "") {
		return nil, ErrLSDDevice
	}
	v := map[string]string{}
	if c.DeviceID != "" {
		v["id"] = c.DeviceID
	}
	if c.DeviceName != "" {
		v["name"] = c.DeviceName
	}
	return v, nil
}

func sameLicense(sent, got *StatusDocument) error {
	if got.ID != sent.ID {
		return fmt.Errorf("%w: document is for a different license", ErrLSDUnexpectedState)
	}
	return nil
}

// Register POSTs the device to the register link (SPEC l.265-283). The
// returned document must be for the same license and must no longer be
// "ready" (l.282). The caller must not register again after success and must
// not block the user when this fails (l.261-262).
func (c *LSDClient) Register(ctx context.Context, d *StatusDocument) (*StatusDocument, error) {
	l, err := programmaticLink(d, "register")
	if err != nil {
		return nil, err
	}
	vars, err := c.deviceVars(true)
	if err != nil {
		return nil, err
	}
	got, err := c.interact(ctx, http.MethodPost, l, vars)
	if err != nil {
		return nil, err
	}
	if err := sameLicense(d, got); err != nil {
		return nil, err
	}
	if got.Status == StatusReady {
		return nil, fmt.Errorf("%w: still ready after registration", ErrLSDUnexpectedState)
	}
	return got, nil
}

// Return PUTs to the return link (SPEC l.309-327). The provider must answer
// "returned" (was active) or "cancelled" (was ready). The caller must then
// download the updated License Document (NeedsLicenseRefresh) and stop
// letting the user open the publication.
func (c *LSDClient) Return(ctx context.Context, d *StatusDocument) (*StatusDocument, error) {
	l, err := programmaticLink(d, "return")
	if err != nil {
		return nil, err
	}
	vars, _ := c.deviceVars(false)
	got, err := c.interact(ctx, http.MethodPut, l, vars)
	if err != nil {
		return nil, err
	}
	if err := sameLicense(d, got); err != nil {
		return nil, err
	}
	if got.Status != StatusReturned && got.Status != StatusCancelled {
		return nil, fmt.Errorf("%w: status %q after return", ErrLSDUnexpectedState, got.Status)
	}
	return got, nil
}

// Renew PUTs to the programmatic renew link (SPEC l.354-384), with `end` when
// given (ISO 8601, UTC). A provider that only offers the text/html renew link
// yields *LSDInteractiveRenewal. The caller must download the updated
// License Document afterwards (l.383).
func (c *LSDClient) Renew(ctx context.Context, d *StatusDocument, end *time.Time) (*StatusDocument, error) {
	l, err := programmaticLink(d, "renew")
	if err != nil {
		return nil, err
	}
	vars, _ := c.deviceVars(false)
	if end != nil {
		vars["end"] = end.UTC().Format(time.RFC3339)
	}
	got, err := c.interact(ctx, http.MethodPut, l, vars)
	if err != nil {
		return nil, err
	}
	if err := sameLicense(d, got); err != nil {
		return nil, err
	}
	return got, nil
}
