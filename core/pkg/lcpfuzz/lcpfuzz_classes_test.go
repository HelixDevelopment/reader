package lcpfuzz

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/lcp"
)

// TestHRD070LicenseRejectionClasses enumerates EVERY rejection rule of
// lcp.Parse together with the typed error it must carry (RULES.md, table L,
// column "error class"): an absent or empty required member is
// ErrIncomplete, a value that is present but malformed (or a document that is
// not a JSON object of the right shape) is ErrLicenseSyntax. The fuzz oracle
// accepts either sentinel for a rejection, so this table is what pins which.
func TestHRD070LicenseRejectionClasses(t *testing.T) {
	inc, syn := lcp.ErrIncomplete, lcp.ErrLicenseSyntax
	uk := func(m map[string]any) map[string]any { return sub(sub(m, "encryption"), "user_key") }
	ck := func(m map[string]any) map[string]any { return sub(sub(m, "encryption"), "content_key") }
	cases := []struct {
		name string
		edit func(m map[string]any)
		want error // nil = accepted
	}{
		{"base is accepted", func(m map[string]any) {}, nil},
		{"id absent", func(m map[string]any) { delete(m, "id") }, inc},
		{"id empty", func(m map[string]any) { m["id"] = "" }, inc},
		{"id wrong type", func(m map[string]any) { m["id"] = 5 }, syn},
		{"provider absent", func(m map[string]any) { delete(m, "provider") }, inc},
		{"provider empty", func(m map[string]any) { m["provider"] = "" }, inc},
		{"issued absent", func(m map[string]any) { delete(m, "issued") }, inc},
		{"issued empty", func(m map[string]any) { m["issued"] = "" }, inc},
		{"issued malformed", func(m map[string]any) { m["issued"] = "not-a-date" }, syn},
		{"updated malformed", func(m map[string]any) { m["updated"] = "not-a-date" }, syn},
		{"updated absent is legal", func(m map[string]any) { delete(m, "updated") }, nil},
		{"updated empty is legal", func(m map[string]any) { m["updated"] = "" }, nil},
		{"encryption absent", func(m map[string]any) { delete(m, "encryption") }, inc},
		{"encryption wrong type", func(m map[string]any) { m["encryption"] = "x" }, syn},
		{"profile absent", func(m map[string]any) { delete(sub(m, "encryption"), "profile") }, inc},
		{"profile empty", func(m map[string]any) { sub(m, "encryption")["profile"] = "" }, inc},
		{"content_key absent", func(m map[string]any) { delete(sub(m, "encryption"), "content_key") }, inc},
		{"content_key.encrypted_value absent", func(m map[string]any) { delete(ck(m), "encrypted_value") }, inc},
		{"content_key.algorithm absent", func(m map[string]any) { delete(ck(m), "algorithm") }, inc},
		{"user_key absent", func(m map[string]any) { delete(sub(m, "encryption"), "user_key") }, inc},
		{"user_key.algorithm absent", func(m map[string]any) { delete(uk(m), "algorithm") }, inc},
		{"user_key.key_check absent", func(m map[string]any) { delete(uk(m), "key_check") }, inc},
		{"user_key.text_hint absent", func(m map[string]any) { delete(uk(m), "text_hint") }, inc},
		{"user_key.text_hint null", func(m map[string]any) { uk(m)["text_hint"] = nil }, inc},
		{"user_key.text_hint empty is legal", func(m map[string]any) { uk(m)["text_hint"] = "" }, nil},
		{"links absent", func(m map[string]any) { delete(m, "links") }, inc},
		{"links empty", func(m map[string]any) { m["links"] = []any{} }, inc},
		{"links wrong type", func(m map[string]any) { m["links"] = "x" }, syn},
		{"link href empty", func(m map[string]any) { m["links"] = []any{map[string]any{"rel": "hint", "href": ""}} }, inc},
		{"second link href absent", func(m map[string]any) {
			m["links"] = []any{map[string]any{"rel": "hint", "href": "h"}, map[string]any{"rel": "x"}}
		}, inc},
		{"signature absent", func(m map[string]any) { delete(m, "signature") }, inc},
		{"signature.algorithm absent", func(m map[string]any) { delete(sub(m, "signature"), "algorithm") }, inc},
		{"signature.certificate absent", func(m map[string]any) { delete(sub(m, "signature"), "certificate") }, inc},
		{"signature.value absent", func(m map[string]any) { delete(sub(m, "signature"), "value") }, inc},
		{"rights wrong type", func(m map[string]any) { m["rights"] = "bad" }, syn},
		{"rights member wrong type", func(m map[string]any) { m["rights"] = map[string]any{"print": "x"} }, syn},
		{"rights date malformed", func(m map[string]any) { m["rights"] = map[string]any{"start": "not-a-date"} }, syn},
		{"rights null is legal", func(m map[string]any) { m["rights"] = nil }, nil},
	}
	for _, c := range cases {
		b := licenseVariant(c.edit)
		if b == nil {
			t.Fatalf("%s: seed generation failed", c.name)
		}
		l, err := lcp.Parse(b)
		switch {
		case c.want == nil && err != nil:
			t.Errorf("%s: rejected: %v", c.name, err)
		case c.want == nil && l == nil:
			t.Errorf("%s: nil license with nil error", c.name)
		case c.want != nil && !errors.Is(err, c.want):
			t.Errorf("%s: error %v, want class %v", c.name, err, c.want)
		case c.want != nil && l != nil:
			t.Errorf("%s: rejected but returned a license", c.name)
		}
		// The classes are exclusive.
		if c.want == inc && errors.Is(err, syn) || c.want == syn && errors.Is(err, inc) {
			t.Errorf("%s: error carries both classes: %v", c.name, err)
		}
	}
	raw := map[string]struct {
		doc  string
		want error
	}{
		"empty input":   {``, syn},
		"truncated":     {`{`, syn},
		"array":         {`[]`, syn},
		"string":        {`"x"`, syn},
		"null":          {`null`, inc},
		"empty object":  {`{}`, inc},
		"trailing data": {string(licenseVariant(func(m map[string]any) {})) + ` {}`, syn},
	}
	for name, c := range raw {
		if _, err := lcp.Parse([]byte(c.doc)); !errors.Is(err, c.want) {
			t.Errorf("%s: error %v, want class %v", name, err, c.want)
		}
	}
}

// TestHRD070LicenseValuesPropagate asserts the VALUE of every parsed field
// of an accepted license (not only its presence): a parser that drops or
// rewrites a field it accepted fails here.
func TestHRD070LicenseValuesPropagate(t *testing.T) {
	b := licenseVariant(func(m map[string]any) { richLicense(m) })
	l, err := lcp.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if l.ID != "urn:uuid:rich" || l.Provider != "https://provider.rich" {
		t.Errorf("id/provider: %q %q", l.ID, l.Provider)
	}
	if want := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC); !l.Issued.Equal(want) {
		t.Errorf("issued %v", l.Issued)
	}
	if want := time.Date(2021, 6, 7, 8, 9, 10, 0, time.UTC); !l.Updated.Equal(want) {
		t.Errorf("updated %v", l.Updated)
	}
	if l.User.ID != "u-1" || l.User.Email != "u@example.org" || l.User.Name != "U Ser" || len(l.User.Encrypted) != 2 || l.User.Encrypted[0] != "email" || l.User.Encrypted[1] != "name" {
		t.Errorf("user %+v", l.User)
	}
	if len(l.Links) != 2 || l.Links[0].Rel != "hint" || l.Links[0].Href != "https://h/1" || l.Links[0].Type != "text/html" ||
		!l.Links[1].Templated || l.Links[1].Href != "https://h/{x}" || len(l.Links[1].Rels) != 2 {
		t.Errorf("links %+v", l.Links)
	}
	if l.Rights.Print == nil || *l.Rights.Print != 10 || l.Rights.Copy == nil || *l.Rights.Copy != 2 || l.Rights.Start == nil || l.Rights.End == nil {
		t.Errorf("rights %+v", l.Rights)
	}
	if l.Signature.Algorithm != "alg" || l.Signature.Certificate != "cert" || l.Signature.Value != "sig" {
		t.Errorf("signature %+v", l.Signature)
	}
	if len(l.UnknownMembers) != 1 || l.UnknownMembers[0] != "x-extra" {
		t.Errorf("unknown %v", l.UnknownMembers)
	}
	if string(l.Raw) != string(b) {
		t.Errorf("Raw is not the received bytes")
	}
}

// richLicense fills every optional member of the license with distinctive values.
func richLicense(m map[string]any) {
	m["id"], m["provider"] = "urn:uuid:rich", "https://provider.rich"
	m["issued"], m["updated"] = "2020-01-02T03:04:05Z", "2021-06-07T08:09:10Z"
	m["user"] = map[string]any{"id": "u-1", "email": "u@example.org", "name": "U Ser", "encrypted": []any{"email", "name"}}
	m["links"] = []any{
		map[string]any{"rel": "hint", "href": "https://h/1", "type": "text/html"},
		map[string]any{"rels": []any{"a", "b"}, "href": "https://h/{x}", "templated": true},
	}
	m["rights"] = map[string]any{"print": 10, "copy": 2, "start": "2020-01-01T00:00:00Z", "end": "2021-01-01T00:00:00Z"}
	m["signature"] = map[string]any{"algorithm": "alg", "certificate": "cert", "value": "sig"}
	m["x-extra"] = 1
}

var _ = json.Marshal

// TestHRD070LSDRejectionClasses: every Status Document rejection carries the
// single typed error ErrStatusDocument and returns no document, including a
// WRONG-TYPED member inside an otherwise valid document (encoding/json keeps
// decoding after a type mismatch, so the decode error is the only thing that
// rejects it).
func TestHRD070LSDRejectionClasses(t *testing.T) {
	edits := map[string]func(m map[string]any){
		"id wrong type":               func(m map[string]any) { m["id"] = 5 },
		"status wrong type":           func(m map[string]any) { m["status"] = 5 },
		"message wrong type":          func(m map[string]any) { m["message"] = 5 },
		"updated wrong type":          func(m map[string]any) { m["updated"] = "x" },
		"updated.license wrong type":  func(m map[string]any) { m["updated"].(map[string]any)["license"] = 5 },
		"links wrong type":            func(m map[string]any) { m["links"] = "x" },
		"link href wrong type":        func(m map[string]any) { richLSD(m); m["links"].([]any)[0].(map[string]any)["href"] = 5 },
		"link templated wrong type":   func(m map[string]any) { richLSD(m); m["links"].([]any)[0].(map[string]any)["templated"] = "yes" },
		"link type wrong type":        func(m map[string]any) { richLSD(m); m["links"].([]any)[0].(map[string]any)["type"] = 5 },
		"event name wrong type":       func(m map[string]any) { richLSD(m); m["events"].([]any)[0].(map[string]any)["name"] = 5 },
		"event id wrong type":         func(m map[string]any) { richLSD(m); m["events"].([]any)[0].(map[string]any)["id"] = 5 },
		"event type wrong type":       func(m map[string]any) { richLSD(m); m["events"].([]any)[0].(map[string]any)["type"] = 5 },
		"events wrong type":           func(m map[string]any) { m["events"] = "x" },
		"potential_rights wrong type": func(m map[string]any) { m["potential_rights"] = "x" },
		"potential_rights.end wrong":  func(m map[string]any) { m["potential_rights"] = map[string]any{"end": 5} },
		"status outside vocabulary":   func(m map[string]any) { m["status"] = "bogus" },
		"id empty":                    func(m map[string]any) { m["id"] = "" },
		"rel empty string":            func(m map[string]any) { m["links"] = []any{map[string]any{"rel": "", "href": "h"}} },
		"rel wrong type":              func(m map[string]any) { m["links"] = []any{map[string]any{"rel": 5, "href": "h"}} },
		"no license link":             func(m map[string]any) { m["links"] = []any{map[string]any{"rel": "status", "href": "h"}} },
		"event timestamp bad":         func(m map[string]any) { richLSD(m); m["events"].([]any)[0].(map[string]any)["timestamp"] = "nope" },
		"event type not enumerated":   func(m map[string]any) { richLSD(m); m["events"].([]any)[0].(map[string]any)["type"] = "bogus" },
	}
	for name, edit := range edits {
		m := baseLSD()
		edit(m)
		doc, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		s, perr := lcp.ParseStatusDocument(doc)
		if s != nil || !errors.Is(perr, lcp.ErrStatusDocument) {
			t.Errorf("%s: got (%v, %v), want (nil, ErrStatusDocument)", name, s, perr)
		}
	}
	for name, raw := range map[string]string{"empty": ``, "truncated": `{`, "array": `[]`, "string": `"x"`, "null": `null`, "trailing": `{"a":1} {}`} {
		if s, perr := lcp.ParseStatusDocument([]byte(raw)); s != nil || !errors.Is(perr, lcp.ErrStatusDocument) {
			t.Errorf("raw %s: got (%v, %v)", name, s, perr)
		}
	}
	m := baseLSD()
	richLSD(m)
	doc, _ := json.Marshal(m)
	if s, perr := lcp.ParseStatusDocument(doc); perr != nil || s == nil {
		t.Errorf("valid rich document rejected: %v", perr)
	}
}
