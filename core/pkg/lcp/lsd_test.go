package lcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// Spec citations: SPEC = readium/lcp-specs releases/lsd/latest.md (revision 5),
// SCHEMA = schema/status.schema.json, both at commit 812dd91.

const (
	testDeviceID   = "dev-7f3a9c52-SECRET-ID"
	testDeviceName = "Helix Reader (test)"
)

func lsdDoc(mut func(m map[string]any)) []byte {
	m := map[string]any{
		"id": "lic-1", "status": "active", "message": "ok",
		"updated": map[string]any{"license": "2026-11-01T00:00:00Z", "status": "2026-11-02T00:00:00Z"},
		"links": []any{
			map[string]any{"rel": "license", "href": "https://example.org/license/1", "profile": "http://readium.org/lcp/basic-profile"},
		},
	}
	if mut != nil {
		mut(m)
	}
	b, _ := json.Marshal(m)
	return b
}

func TestParseStatusDocumentEveryStatus(t *testing.T) { // SPEC l.156-161, SCHEMA l.14-21
	for _, s := range []StatusValue{StatusReady, StatusActive, StatusRevoked, StatusReturned, StatusCancelled, StatusExpired} {
		t.Run(string(s), func(t *testing.T) {
			d, err := ParseStatusDocument(lsdDoc(func(m map[string]any) { m["status"] = string(s) }))
			if err != nil {
				t.Fatal(err)
			}
			if d.Status != s || d.ID != "lic-1" || d.Message != "ok" {
				t.Fatalf("got %+v", d)
			}
			wantLive := s == StatusReady || s == StatusActive
			if d.LicenseUsable() != wantLive {
				t.Fatalf("LicenseUsable(%s) = %v", s, d.LicenseUsable())
			}
		})
	}
}

func TestParseStatusDocumentRejects(t *testing.T) {
	del := func(path ...string) func(map[string]any) {
		return func(m map[string]any) {
			cur := m
			for _, p := range path[:len(path)-1] {
				cur = cur[p].(map[string]any)
			}
			delete(cur, path[len(path)-1])
		}
	}
	set := func(k string, v any) func(map[string]any) { return func(m map[string]any) { m[k] = v } }
	cases := map[string]func(map[string]any){
		"missing id":              del("id"),
		"missing status":          del("status"),
		"missing message":         del("message"),
		"missing updated":         del("updated"),
		"missing updated.license": del("updated", "license"),
		"missing updated.status":  del("updated", "status"),
		"missing links":           del("links"),
		"unknown status":          set("status", "lent"),
		"empty status":            set("status", ""),
		"uppercase status":        set("status", "ACTIVE"),
		"bad updated.license":     func(m map[string]any) { m["updated"].(map[string]any)["license"] = "yesterday" },
		"bad potential end":       set("potential_rights", map[string]any{"end": "soon"}),
		"no license link":         set("links", []any{map[string]any{"rel": "return", "href": "https://x/r"}}),
		"license link no href":    set("links", []any{map[string]any{"rel": "license"}}),
		"link without rel":        set("links", []any{map[string]any{"rel": "license", "href": "https://x/l"}, map[string]any{"href": "https://x/y"}}),
		"link rel wrong type":     set("links", []any{map[string]any{"rel": 7, "href": "https://x/l"}}),
		"event bad type":          set("events", []any{map[string]any{"type": "lend", "timestamp": "2026-11-01T00:00:00Z"}}),
		"event bad timestamp":     set("events", []any{map[string]any{"type": "register", "timestamp": "x"}}),
		"id wrong json type":      set("id", 5),
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseStatusDocument(lsdDoc(mut)); !errors.Is(err, ErrStatusDocument) {
				t.Fatalf("err = %v, want ErrStatusDocument", err)
			}
		})
	}
	for name, raw := range map[string]string{"not json": "<html>", "array": "[]", "trailing": string(lsdDoc(nil)) + "{}", "empty": ""} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseStatusDocument([]byte(raw)); !errors.Is(err, ErrStatusDocument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestParseStatusDocumentFullExample(t *testing.T) { // SPEC l.60-115 (the normative example)
	const ex = `{"id":"234-5435-3453-345354","status":"active","message":"m",
"updated":{"license":"2016-08-05T00:00:00Z","status":"2016-08-08T00:00:00Z"},
"links":[
{"rel":"license","href":"https://example.org/license/35d9b2d6","type":"application/vnd.readium.lcp.license.v1.0+json","profile":"http://readium.org/lcp/basic-profile"},
{"rel":"register","href":"https://example.org/license/35d9b2d6/register{?id,name}","type":"application/vnd.readium.license.status.v1.0+json","templated":true},
{"rel":["return"],"href":"https://example.org/license/35d9b2d6/return{?id,name}","templated":true},
{"rel":"renew","href":"https://example.org/license/35d9b2d6/renew","type":"text/html"},
{"rel":"renew","href":"https://example.org/license/35d9b2d6/renew{?end,id,name}","type":"application/vnd.readium.license.status.v1.0+json","templated":true}],
"potential_rights":{"end":"2014-09-13T00:00:00Z"},
"events":[{"type":"register","name":"eBook App (Android)","timestamp":"2016-07-14T00:00:00Z","id":"709e1380"}]}`
	d, err := ParseStatusDocument([]byte(ex))
	if err != nil {
		t.Fatal(err)
	}
	if !d.UpdatedLicense.Equal(time.Date(2016, 8, 5, 0, 0, 0, 0, time.UTC)) || !d.UpdatedStatus.Equal(time.Date(2016, 8, 8, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("updated: %v %v", d.UpdatedLicense, d.UpdatedStatus)
	}
	if d.PotentialRightsEnd == nil || d.PotentialRightsEnd.Year() != 2014 {
		t.Fatalf("potential_rights.end: %v", d.PotentialRightsEnd)
	}
	if len(d.Events) != 1 || d.Events[0].Type != "register" || d.Events[0].Name != "eBook App (Android)" || d.Events[0].ID != "709e1380" {
		t.Fatalf("events: %+v", d.Events)
	}
	if len(d.Links) != 5 || !d.Links[2].HasRel("return") || !d.Links[1].Templated || d.Links[3].Templated {
		t.Fatalf("links: %+v", d.Links)
	}
	if l, ok := d.Link("return"); !ok || !strings.HasSuffix(l.Href, "{?id,name}") {
		t.Fatal("rel given as array not found")
	}
}

func TestExpandLinkTemplate(t *testing.T) { // SPEC l.97, l.303, l.350 (templates) and l.275-276, 319-320, 375-377 (params)
	cases := []struct {
		name, href string
		vars       map[string]string
		want       string
		wantErr    bool
	}{
		{"register id+name", "https://e.org/r{?id,name}", map[string]string{"id": "d1", "name": "My Phone"}, "https://e.org/r?id=d1&name=My%20Phone", false},
		{"renew with end", "https://e.org/n{?end,id,name}", map[string]string{"end": "2026-12-01T00:00:00Z", "id": "d", "name": "n"}, "https://e.org/n?end=2026-12-01T00%3A00%3A00Z&id=d&name=n", false},
		{"renew without end drops it", "https://e.org/n{?end,id,name}", map[string]string{"id": "d", "name": "n"}, "https://e.org/n?id=d&name=n", false},
		{"nothing defined", "https://e.org/n{?end,id,name}", nil, "https://e.org/n", false},
		{"reserved chars encoded", "https://e.org/r{?id}", map[string]string{"id": "a&b=c/d+e"}, "https://e.org/r?id=a%26b%3Dc%2Fd%2Be", false},
		{"unicode utf8", "https://e.org/r{?name}", map[string]string{"name": "Čitač"}, "https://e.org/r?name=%C4%8Cita%C4%8D", false},
		{"existing query uses &-continuation", "https://e.org/r?token=t{?id}", map[string]string{"id": "d"}, "https://e.org/r?token=t&id=d", false},
		{"no expression", "https://e.org/plain", map[string]string{"id": "d"}, "https://e.org/plain", false},
		{"unsupported operator", "https://e.org/{+path}", nil, "", true},
		{"unterminated", "https://e.org/r{?id", nil, "", true},
		{"unknown variable", "https://e.org/r{?id,secret}", map[string]string{"id": "d", "secret": "s"}, "https://e.org/r?id=d&secret=s", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := expandLinkTemplate(c.href, c.vars)
			if (err != nil) != c.wantErr || got != c.want {
				t.Fatalf("got %q err=%v, want %q err=%v", got, err, c.want, c.wantErr)
			}
		})
	}
}

func TestNeedsLicenseRefresh(t *testing.T) { // SPEC l.244
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	doc := func(lic time.Time) *StatusDocument { return &StatusDocument{UpdatedLicense: lic} }
	cases := []struct {
		name      string
		status    time.Time
		local     time.Time
		wantFresh bool
	}{
		{"status newer", base.Add(time.Second), base, true},
		{"equal", base, base, false},
		{"status older", base.Add(-time.Hour), base, false},
		{"different zone same instant", base.In(time.FixedZone("x", 3600)), base, false},
		{"local has no updated, uses issued", base, time.Time{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lic := &License{Updated: c.local, Issued: base.Add(-time.Hour)}
			if got := NeedsLicenseRefresh(doc(c.status), lic); got != c.wantFresh {
				t.Fatalf("got %v", got)
			}
		})
	}
	if NeedsLicenseRefresh(nil, &License{}) || NeedsLicenseRefresh(doc(base), nil) != true {
		t.Fatal("nil handling: nil status => false, nil license => true")
	}
}

func TestSelectLicenseLinkByProfile(t *testing.T) { // SPEC l.246-248
	basic, v11 := "http://readium.org/lcp/basic-profile", "http://readium.org/lcp/profile-1.0"
	mk := func(ls ...LSDLink) *StatusDocument { return &StatusDocument{Links: ls} }
	a := LSDLink{Rels: []string{"license"}, Href: "https://x/basic", Profile: basic}
	b := LSDLink{Rels: []string{"license"}, Href: "https://x/v11", Profile: v11}
	other := LSDLink{Rels: []string{"license"}, Href: "https://x/other", Profile: "http://example.org/p"}
	sole := LSDLink{Rels: []string{"license"}, Href: "https://x/sole"}
	supported := []string{basic, v11} // ascending
	cases := []struct {
		name string
		doc  *StatusDocument
		want string
	}{
		{"highest supported wins, order irrelevant 1", mk(a, b), "https://x/v11"},
		{"highest supported wins, order irrelevant 2", mk(b, a), "https://x/v11"},
		{"unsupported ignored", mk(other, a), "https://x/basic"},
		{"nothing supported => none", mk(other), ""},
		{"sole link without profile accepted", mk(sole), "https://x/sole"},
		{"profile-less ignored when others exist", mk(sole, other), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, ok := c.doc.SelectLicenseLink(supported)
			if (c.want != "") != ok || l.Href != c.want {
				t.Fatalf("got %q ok=%v want %q", l.Href, ok, c.want)
			}
		})
	}
}

func newClient(srv *httptest.Server) *LSDClient {
	return &LSDClient{HTTP: srv.Client(), DeviceID: testDeviceID, DeviceName: testDeviceName}
}

func TestLSDProblemDocuments(t *testing.T) { // SPEC l.231-238 + failure tables l.286-289, 331-336, 387-391
	cases := []struct {
		name, ctype, body string
		code              int
		wantType, title   string
		wantDetail        string
		wantProblemJSON   bool
	}{
		{"registration 400", "application/problem+json", `{"type":"http://readium.org/license-status-document/error/registration","title":"Your device could not be registered properly.","detail":"id missing"}`, 400, "http://readium.org/license-status-document/error/registration", "Your device could not be registered properly.", "id missing", true},
		{"return already 403", "application/problem+json; charset=utf-8", `{"type":"http://readium.org/license-status-document/error/return/already","title":"Your publication has already been returned before."}`, 403, "http://readium.org/license-status-document/error/return/already", "Your publication has already been returned before.", "", true},
		{"renew date 403 plain json ctype", "application/json", `{"type":"http://readium.org/license-status-document/error/renew/date","title":"Incorrect renewal period, your publication could not be renewed."}`, 403, "http://readium.org/license-status-document/error/renew/date", "Incorrect renewal period, your publication could not be renewed.", "", true},
		{"server 503", "application/problem+json", `{"type":"http://readium.org/license-status-document/error/server","title":"An unexpected error has occurred."}`, 503, "http://readium.org/license-status-document/error/server", "An unexpected error has occurred.", "", true},
		{"html error page", "text/html", `<html>oops</html>`, 502, "", "", "", false},
		{"problem body that is not an object", "application/problem+json", `[1]`, 400, "", "", "", false},
		{"empty body", "", ``, 500, "", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if c.ctype != "" {
					w.Header().Set("Content-Type", c.ctype)
				}
				w.WriteHeader(c.code)
				_, _ = w.Write([]byte(c.body))
			}))
			defer srv.Close()
			_, err := newClient(srv).FetchStatus(context.Background(), srv.URL+"/s")
			var p *LSDProblem
			if !errors.As(err, &p) {
				t.Fatalf("err = %v (%T), want *LSDProblem", err, err)
			}
			if p.HTTPStatus != c.code || p.Type != c.wantType || p.Title != c.title || p.Detail != c.wantDetail {
				t.Fatalf("got %+v", p)
			}
			if got := p.IsProblemDocument(); got != c.wantProblemJSON {
				t.Fatalf("IsProblemDocument = %v", got)
			}
			if containsAny(err.Error(), testDeviceID) {
				t.Fatalf("error leaks the device id: %v", err)
			}
		})
	}
}

func TestLSDRefusesSuccessThatIsNotAStatusDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"hello":1}`))
	}))
	defer srv.Close()
	if _, err := newClient(srv).FetchStatus(context.Background(), srv.URL); !errors.Is(err, ErrStatusDocument) {
		t.Fatalf("err = %v", err)
	}
}

func TestLSDTransportErrorsCarryNoQueryString(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // connection refused
	c := &LSDClient{HTTP: http.DefaultClient, DeviceID: testDeviceID, DeviceName: testDeviceName}
	d := &StatusDocument{ID: "x", Links: []LSDLink{{Rels: []string{"register"}, Href: url + "/register{?id,name}", Templated: true}}}
	_, err := c.Register(context.Background(), d)
	if err == nil || containsAny(err.Error(), testDeviceID, testDeviceName, "?") {
		t.Fatalf("err = %v", err)
	}
}

func TestLSDClientRefusesWhatItCannotDoWithoutNetwork(t *testing.T) {
	d := &StatusDocument{ID: "x", Links: []LSDLink{{Rels: []string{"license"}, Href: "https://x/l"}}}
	cases := map[string]func(c *LSDClient) error{
		"register: no link": func(c *LSDClient) error { _, e := c.Register(context.Background(), d); return e },
		"return: no link":   func(c *LSDClient) error { _, e := c.Return(context.Background(), d); return e },
		"renew: no link":    func(c *LSDClient) error { _, e := c.Renew(context.Background(), d, nil); return e },
	}
	for n, f := range cases {
		t.Run(n, func(t *testing.T) {
			if err := f(&LSDClient{DeviceID: "a", DeviceName: "b"}); !errors.Is(err, ErrLSDNoLink) {
				t.Fatalf("err = %v", err)
			}
		})
	}
	reg := &StatusDocument{ID: "x", Links: []LSDLink{{Rels: []string{"register"}, Href: "https://x/r{?id,name}", Templated: true}}}
	if _, err := (&LSDClient{}).Register(context.Background(), reg); !errors.Is(err, ErrLSDDevice) {
		t.Fatalf("register without device identity: %v", err) // SPEC l.275-276: id and name required
	}
	htmlOnly := &StatusDocument{ID: "x", Links: []LSDLink{{Rels: []string{"renew"}, Href: "https://x/renew", Type: "text/html"}}}
	_, err := (&LSDClient{DeviceID: "a", DeviceName: "b"}).Renew(context.Background(), htmlOnly, nil)
	var ir *LSDInteractiveRenewal
	if !errors.As(err, &ir) || ir.URL != "https://x/renew" { // SPEC l.368
		t.Fatalf("html-only renew: %v", err)
	}
}

func TestLSDRequestShape(t *testing.T) { // verbs: SPEC l.270, 314, 367; params l.275-277, 319-320, 375-377
	type seen struct {
		method string
		q      url.Values
		accept string
	}
	var got []seen
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, seen{r.Method, r.URL.Query(), r.Header.Get("Accept")})
		d := lsdDoc(func(m map[string]any) {
			m["status"] = map[string]string{"/register": "active", "/return": "returned", "/renew": "active"}[r.URL.Path]
		})
		_, _ = w.Write(d)
	}))
	defer srv.Close()
	doc := &StatusDocument{ID: "lic-1", Links: []LSDLink{
		{Rels: []string{"register"}, Href: srv.URL + "/register{?id,name}", Templated: true, Type: lsdMediaType},
		{Rels: []string{"return"}, Href: srv.URL + "/return{?id,name}", Templated: true},
		{Rels: []string{"renew"}, Href: srv.URL + "/renew-ui", Type: "text/html"},
		{Rels: []string{"renew"}, Href: srv.URL + "/renew{?end,id,name}", Templated: true, Type: lsdMediaType},
	}}
	c := newClient(srv)
	end := time.Date(2026, 12, 24, 12, 0, 0, 0, time.FixedZone("x", 7200))
	ctx := context.Background()
	if _, err := c.Register(ctx, doc); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Renew(ctx, doc, &end); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Renew(ctx, doc, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Return(ctx, doc); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		method string
		keys   []string
		end    string
	}{{"POST", []string{"id", "name"}, ""}, {"PUT", []string{"end", "id", "name"}, "2026-12-24T10:00:00Z"}, {"PUT", []string{"id", "name"}, ""}, {"PUT", []string{"id", "name"}, ""}}
	if len(got) != len(want) {
		t.Fatalf("%d requests", len(got))
	}
	for i, w := range want {
		var keys []string
		for k := range got[i].q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if got[i].method != w.method || !reflect.DeepEqual(keys, w.keys) || got[i].q.Get("end") != w.end || got[i].accept != lsdMediaType {
			t.Fatalf("request %d = %+v, want %+v", i, got[i], w)
		}
		if got[i].q.Get("id") != testDeviceID || got[i].q.Get("name") != testDeviceName {
			t.Fatalf("request %d device params wrong", i)
		}
	}
}

func TestLSDStateTransitionChecks(t *testing.T) { // SPEC l.282 (ready->active), l.326 (returned/cancelled)
	cases := []struct {
		name string
		op   func(c *LSDClient, d *StatusDocument) (*StatusDocument, error)
		resp string // status the (faulty) server answers
		ok   bool
	}{
		{"register ready->active", func(c *LSDClient, d *StatusDocument) (*StatusDocument, error) {
			return c.Register(context.Background(), d)
		}, "active", true},
		{"register still ready is a contradiction", func(c *LSDClient, d *StatusDocument) (*StatusDocument, error) {
			return c.Register(context.Background(), d)
		}, "ready", false},
		{"register may report revoked", func(c *LSDClient, d *StatusDocument) (*StatusDocument, error) {
			return c.Register(context.Background(), d)
		}, "revoked", true},
		{"return -> returned", func(c *LSDClient, d *StatusDocument) (*StatusDocument, error) {
			return c.Return(context.Background(), d)
		}, "returned", true},
		{"return -> cancelled", func(c *LSDClient, d *StatusDocument) (*StatusDocument, error) {
			return c.Return(context.Background(), d)
		}, "cancelled", true},
		{"return left active is a contradiction", func(c *LSDClient, d *StatusDocument) (*StatusDocument, error) {
			return c.Return(context.Background(), d)
		}, "active", false},
		{"return -> expired is a contradiction", func(c *LSDClient, d *StatusDocument) (*StatusDocument, error) {
			return c.Return(context.Background(), d)
		}, "expired", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(lsdDoc(func(m map[string]any) { m["status"] = c.resp; m["id"] = "lic-1" }))
			}))
			defer srv.Close()
			d := &StatusDocument{ID: "lic-1", Links: []LSDLink{
				{Rels: []string{"register"}, Href: srv.URL + "/r{?id,name}", Templated: true},
				{Rels: []string{"return"}, Href: srv.URL + "/t{?id,name}", Templated: true},
			}}
			_, err := c.op(newClient(srv), d)
			if c.ok && err != nil || !c.ok && !errors.Is(err, ErrLSDUnexpectedState) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestLSDRefusesAnotherLicensesDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(lsdDoc(func(m map[string]any) { m["id"] = "someone-else" }))
	}))
	defer srv.Close()
	d := &StatusDocument{ID: "lic-1", Links: []LSDLink{{Rels: []string{"register"}, Href: srv.URL + "/r{?id,name}", Templated: true}}}
	if _, err := newClient(srv).Register(context.Background(), d); !errors.Is(err, ErrLSDUnexpectedState) {
		t.Fatalf("err = %v", err)
	}
}

func TestLSDOversizedBodyRefused(t *testing.T) {
	// A VALID Status Document padded past the limit: only the size limit can
	// reject it, so removing or raising lsdMaxBody makes this fail.
	big := lsdDoc(func(m map[string]any) { m["message"] = strings.Repeat("a", 1<<20+1024) })
	if len(big) <= 1<<20 { // literal, NOT lsdMaxBody: raising the limit must fail this test
		t.Fatalf("fixture only %d bytes", len(big))
	}
	if _, err := ParseStatusDocument(big); err != nil {
		t.Fatalf("fixture must be a valid document on its own: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(big)
	}))
	defer srv.Close()
	_, err := newClient(srv).FetchStatus(context.Background(), srv.URL)
	if !errors.Is(err, ErrStatusDocument) || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized valid document: err = %v, want a size-limit refusal", err)
	}
}

// TestLSDLiveLoopbackLifecycle drives register -> renew -> return over a real
// TCP listener against a provider written independently from the spec text.
// When HELIX_LSD_EVIDENCE names a file the observed facts go there as JSON
// (query KEYS only: no device id or name is ever recorded).
func TestLSDLiveLoopbackLifecycle(t *testing.T) {
	p := newLSDProvider()
	defer p.srv.Close()
	c := &LSDClient{HTTP: p.srv.Client(), DeviceID: testDeviceID, DeviceName: testDeviceName}
	ctx := context.Background()
	trace := []map[string]any{}
	step := func(name string, d *StatusDocument) {
		trace = append(trace, map[string]any{"step": name, "status": string(d.Status),
			"updated_license": d.UpdatedLicense.Format(time.RFC3339), "updated_status": d.UpdatedStatus.Format(time.RFC3339), "events": len(d.Events)})
	}

	d0, err := c.FetchStatus(ctx, p.statusURL())
	if err != nil {
		t.Fatal(err)
	}
	step("fetch", d0)
	if d0.Status != StatusReady || len(d0.Events) != 0 {
		t.Fatalf("initial: %+v", d0)
	}
	local := &License{Updated: d0.UpdatedLicense}
	if NeedsLicenseRefresh(d0, local) {
		t.Fatal("fresh license reported stale")
	}
	if l, ok := d0.SelectLicenseLink([]string{"http://readium.org/lcp/basic-profile"}); !ok || l.Href != p.srv.URL+"/license" {
		t.Fatalf("license link: %+v %v", l, ok)
	}

	d1, err := c.Register(ctx, d0)
	if err != nil {
		t.Fatal(err)
	}
	step("register", d1)
	if d1.Status != StatusActive || !d1.UpdatedStatus.After(d0.UpdatedStatus) || !d1.UpdatedLicense.Equal(d0.UpdatedLicense) {
		t.Fatalf("after register: %+v", d1)
	}
	if len(d1.Events) != 1 || d1.Events[0].Type != "register" || d1.Events[0].ID != testDeviceID {
		t.Fatalf("register event: %+v", d1.Events)
	}
	if NeedsLicenseRefresh(d1, local) {
		t.Fatal("register must not make the license stale (license timestamp unchanged)")
	}

	end := d1.PotentialRightsEnd.Add(7 * 24 * time.Hour)
	d2, err := c.Renew(ctx, d1, &end)
	if err != nil {
		t.Fatal(err)
	}
	step("renew", d2)
	if !d2.PotentialRightsEnd.Equal(end) || !d2.UpdatedLicense.After(d1.UpdatedLicense) {
		t.Fatalf("after renew: %+v", d2)
	}
	if !NeedsLicenseRefresh(d2, local) {
		t.Fatal("renewed license must be reported stale so the client re-fetches it")
	}

	var errs []error // every error the lifecycle produced, all leak-checked below
	badEnd := end.AddDate(1, 0, 0)
	_, err = c.Renew(ctx, d2, &badEnd)
	errs = append(errs, err)
	var prob *LSDProblem
	if !errors.As(err, &prob) || prob.Type != "http://readium.org/license-status-document/error/renew/date" || prob.HTTPStatus != 403 {
		t.Fatalf("bad renewal period: %v", err)
	}

	d3, err := c.Return(ctx, d2)
	if err != nil {
		t.Fatal(err)
	}
	step("return", d3)
	if d3.Status != StatusReturned || d3.LicenseUsable() || !d3.UpdatedLicense.After(d2.UpdatedLicense) {
		t.Fatalf("after return: %+v", d3)
	}

	_, err = c.Return(ctx, d3)
	errs = append(errs, err)
	if !errors.As(err, &prob) || prob.Type != "http://readium.org/license-status-document/error/return/already" || prob.HTTPStatus != 403 {
		t.Fatalf("second return: %v", err)
	}
	trace = append(trace, map[string]any{"step": "second_return", "problem_type": prob.Type, "http_status": prob.HTTPStatus})

	for _, e := range errs {
		if e == nil {
			t.Fatal("expected an error in the collected set")
		}
		if containsAny(e.Error(), testDeviceID, testDeviceName) {
			t.Fatalf("error leaks device identity: %v", e)
		}
	}

	if path := os.Getenv("HELIX_LSD_EVIDENCE"); path != "" {
		out, _ := json.MarshalIndent(map[string]any{
			"spec":     "readium/lcp-specs releases/lsd/latest.md revision 5, commit 812dd91d6b90087e657a5c94d04e6f16abd8873b",
			"provider": "in-repo loopback provider written from the spec text (NOT a third-party LSD provider)",
			"trace":    trace, "requests": p.requests(),
			"note": "query values (device id/name) deliberately not recorded",
		}, "", "  ")
		if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLSDLiveRegisterRefusalsAndFaults(t *testing.T) {
	ctx := context.Background()
	t.Run("provider that does not activate is caught", func(t *testing.T) {
		p := newLSDProvider()
		defer p.srv.Close()
		p.skipActivate = true
		c := &LSDClient{HTTP: p.srv.Client(), DeviceID: "d", DeviceName: "n"}
		d, err := c.FetchStatus(ctx, p.statusURL())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Register(ctx, d); !errors.Is(err, ErrLSDUnexpectedState) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("provider answering for another license is caught", func(t *testing.T) {
		p := newLSDProvider()
		defer p.srv.Close()
		c := &LSDClient{HTTP: p.srv.Client(), DeviceID: "d", DeviceName: "n"}
		d, _ := c.FetchStatus(ctx, p.statusURL())
		p.wrongID = true
		if _, err := c.Register(ctx, d); !errors.Is(err, ErrLSDUnexpectedState) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("return before activation cancels", func(t *testing.T) {
		p := newLSDProvider()
		defer p.srv.Close()
		c := &LSDClient{HTTP: p.srv.Client(), DeviceID: "d", DeviceName: "n"}
		d, _ := c.FetchStatus(ctx, p.statusURL())
		r, err := c.Return(ctx, d)
		if err != nil || r.Status != StatusCancelled {
			t.Fatalf("got %+v %v", r, err)
		}
	})
}

func TestLSDInteractiveRenewalURLMustBeAbsoluteHTTP(t *testing.T) { // review item 1
	for _, href := range []string{"javascript:alert(1)", "file:///etc/passwd", "intent://x#Intent;end", "/renew", "https:///nohost", "//evil.example/x", "data:text/html,x", ""} {
		d := &StatusDocument{ID: "x", Links: []LSDLink{{Rels: []string{"renew"}, Href: href, Type: "text/html"}}}
		_, err := (&LSDClient{DeviceID: "a", DeviceName: "b"}).Renew(context.Background(), d, nil)
		var ir *LSDInteractiveRenewal
		if errors.As(err, &ir) || !errors.Is(err, ErrLSDNoLink) {
			t.Fatalf("href %q: err = %v, want ErrLSDNoLink and no interactive URL", href, err)
		}
	}
	for _, href := range []string{"http://x.example/r", "https://x.example/r?a=1"} {
		d := &StatusDocument{ID: "x", Links: []LSDLink{{Rels: []string{"renew"}, Href: href, Type: "text/html"}}}
		_, err := (&LSDClient{}).Renew(context.Background(), d, nil)
		var ir *LSDInteractiveRenewal
		if !errors.As(err, &ir) || ir.URL != href {
			t.Fatalf("href %q: err = %v", href, err)
		}
	}
}

func TestParseStatusDocumentSparseEvents(t *testing.T) { // review item 2: SCHEMA events items have no `required`
	ts := "2026-11-01T00:00:00Z"
	d, err := ParseStatusDocument(lsdDoc(func(m map[string]any) {
		m["events"] = []any{
			map[string]any{"type": "register", "name": "n"},     // no timestamp
			map[string]any{"timestamp": ts, "name": "typeless"}, // no type
			map[string]any{}, // nothing
			map[string]any{"type": "renew", "timestamp": ts},
		}
	}))
	if err != nil {
		t.Fatalf("sparse events must not be fatal: %v", err)
	}
	last := d.Events[len(d.Events)-1]
	if last.Type != "renew" || !last.Timestamp.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("complete event lost: %+v", d.Events)
	}
	for _, bad := range []map[string]any{{"type": "renew", "timestamp": ""}, {"type": "", "timestamp": ts}, {"type": "renew", "timestamp": "x"}} {
		if _, err := ParseStatusDocument(lsdDoc(func(m map[string]any) { m["events"] = []any{bad} })); !errors.Is(err, ErrStatusDocument) {
			t.Fatalf("present-but-invalid event %v accepted: %v", bad, err)
		}
	}
}

func TestParseStatusDocumentEmptyRelArray(t *testing.T) { // review item 5d: link.schema.json rel has no minItems
	d, err := ParseStatusDocument(lsdDoc(func(m map[string]any) {
		m["links"] = append(m["links"].([]any), map[string]any{"rel": []any{}, "href": "https://x/y"})
	}))
	if err != nil || len(d.Links) != 2 || len(d.Links[1].Rels) != 0 {
		t.Fatalf("empty rel array: %v %+v", err, d)
	}
	// the license-link requirement is not weakened by it
	if _, err := ParseStatusDocument(lsdDoc(func(m map[string]any) {
		m["links"] = []any{map[string]any{"rel": []any{}, "href": "https://x/l"}}
	})); !errors.Is(err, ErrStatusDocument) {
		t.Fatalf("doc whose only link has rel [] accepted: %v", err)
	}
}

func TestLSDStateChangingRequestsNeverFollowRedirects(t *testing.T) { // review item 5a
	for _, code := range []int{301, 302, 303, 307, 308} {
		var hits []string
		var srv *httptest.Server
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits = append(hits, r.Method+" "+r.URL.Path)
			if r.URL.Path == "/final" {
				_, _ = w.Write(lsdDoc(func(m map[string]any) { m["status"] = "active"; m["id"] = "lic-1" }))
				return
			}
			http.Redirect(w, r, "/final", code)
		}))
		d := &StatusDocument{ID: "lic-1", Links: []LSDLink{
			{Rels: []string{"register"}, Href: srv.URL + "/register{?id,name}", Templated: true},
			{Rels: []string{"return"}, Href: srv.URL + "/return{?id,name}", Templated: true},
			{Rels: []string{"renew"}, Href: srv.URL + "/renew{?id,name}", Templated: true},
		}}
		c := newClient(srv)
		ctx := context.Background()
		_, e1 := c.Register(ctx, d)
		_, e2 := c.Return(ctx, d)
		_, e3 := c.Renew(ctx, d, nil)
		srv.Close()
		for i, e := range []error{e1, e2, e3} {
			var p *LSDProblem
			if !errors.As(e, &p) || p.HTTPStatus != code {
				t.Fatalf("HTTP %d call %d: err = %v, want *LSDProblem with the redirect status", code, i, e)
			}
		}
		for _, h := range hits {
			if strings.HasSuffix(h, "/final") {
				t.Fatalf("HTTP %d: redirect followed: %v", code, hits)
			}
		}
	}
	// the caller's own CheckRedirect is not clobbered for reads
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			_, _ = w.Write(lsdDoc(nil))
			return
		}
		http.Redirect(w, r, "/final", 302)
	}))
	defer srv.Close()
	if _, err := newClient(srv).FetchStatus(context.Background(), srv.URL+"/s"); err != nil {
		t.Fatalf("GET redirect must still work: %v", err)
	}
}

func TestLSDUntemplatedHrefWithTemplateExpressionIsRefused(t *testing.T) { // review item 5b; SPEC l.182: templated defaults to false
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()
	d := &StatusDocument{ID: "lic-1", Links: []LSDLink{{Rels: []string{"register"}, Href: srv.URL + "/register{?id,name}"}}}
	_, err := newClient(srv).Register(context.Background(), d)
	if !errors.Is(err, ErrLSDBadTemplate) || hit {
		t.Fatalf("err = %v, request sent = %v", err, hit)
	}
	if containsAny(err.Error(), testDeviceID, testDeviceName) {
		t.Fatalf("leak: %v", err)
	}
}
