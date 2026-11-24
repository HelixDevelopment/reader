package lcpfuzz

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/lcp"
)

// Oracle for every expected string below: LCP 1.0 §5.3 rules 1-5 applied by hand
// (sorted members, array order kept, no insignificant whitespace, numbers per
// rule 4, minimal escapes with upper-case hex in spec mode), and the reference
// producer's documented differences (lower-case hex, U+2028/9 escaped, numbers
// passed through) for reference mode. No expected value is produced by the code
// under test.

type canonCase struct {
	name, in, spec, ref string // ref "" = same as spec
}

func TestHRD070CanonicalOutput(t *testing.T) {
	cases := []canonCase{
		{"member order", `{"b":1,"a":2,"c":3}`, `{"a":2,"b":1,"c":3}`, ""},
		{"nested order, array order kept", `{"z":{"b":1,"a":2},"y":[{"d":1,"c":2},{"b":0,"a":9}],"x":[3,1,2]}`, `{"x":[3,1,2],"y":[{"c":2,"d":1},{"a":9,"b":0}],"z":{"a":2,"b":1}}`, ""},
		{"code point order not locale", `{"b":1,"B":2,"a":3,"é":4,"~":5}`, `{"B":2,"a":3,"b":1,"~":5,"é":4}`, ""},
		{"whitespace dropped", " {\n \"a\" : [ 1 , 2 ] ,\t\"b\" : { } }\r\n", `{"a":[1,2],"b":{}}`, ""},
		{"null true false", `{"n":null,"t":true,"f":false}`, `{"f":false,"n":null,"t":true}`, ""},
		{"scalar documents", `true`, `true`, ""},
		{"scalar false", `false`, `false`, ""},
		{"scalar null", `null`, `null`, ""},
		{"scalar string", `"x"`, `"x"`, ""},
		{"scalar number", `7`, `7`, ""},
		{"empty containers", `[{},[]]`, `[{},[]]`, ""},
		{"short escapes", `"\"\\\b\f\n\r\t"`, `"\"\\\b\f\n\r\t"`, ""},
		{"control hex case", `"\u0001\u000b\u001f"`, `"\u0001\u000B\u001F"`, `"\u0001\u000b\u001f"`},
		{"space and 0x7f and html are literal", "\" \u007f<>&'\"", "\" \u007f<>&'\"", ""},
		{"non-ascii literal", `"é€😀"`, `"é€😀"`, ""},
		{"U+2028 U+2029", `"a\u2028b\u2029c"`, "\"a\u2028b\u2029c\"", `"a\u2028b\u2029c"`},
		{"key escapes follow the mode", `{"k\u001f":1}`, `{"k\u001F":1}`, `{"k\u001f":1}`},
		{"key U+2028 follows the mode", `{"\u2028":1}`, "{\"\u2028\":1}", `{"\u2028":1}`},
		{"integers unchanged", `[0,-5,123,100,10]`, `[0,-5,123,100,10]`, ""},
		{"minus zero", `-0`, `0`, `-0`},
		{"fraction", `[1.5,-1.5]`, `[1.5E0,-1.5E0]`, `[1.5,-1.5]`},
		{"trailing zero fraction", `[1.0,1.50,12.50,1.20e1]`, `[1E0,1.5E0,1.25E1,1.2E1]`, `[1.0,1.50,12.50,1.20e1]`},
		{"zero fraction and exponent", `[0.0,-0.0,0e5,0.000]`, `[0,0,0,0]`, `[0.0,-0.0,0e5,0.000]`},
		{"leading zeros of the fraction", `[0.001,0.5,0.0120]`, `[1E-3,5E-1,1.2E-2]`, `[0.001,0.5,0.0120]`},
		{"exponent forms", `[1e2,1E+2,1e-2,1.5e3,12e0]`, `[1E2,1E2,1E-2,1.5E3,1.2E1]`, `[1e2,1E+2,1e-2,1.5e3,12e0]`},
		{"big exponent in range", `1e400`, `1E400`, `1e400`},
		{"exponent at the upper limit is accepted", `1e1073741824`, `1E1073741824`, `1e1073741824`},
		{"exponent at the lower limit is accepted", `1e-1073741824`, `1E-1073741824`, `1e-1073741824`},
		{"negative number with exponent", `-2.5e-3`, `-2.5E-3`, `-2.5e-3`},
		{"integral value with exponent keeps sign", `-100e-2`, `-1E0`, `-100e-2`},
		{"long digits keep precision", `123456789012345678901234567890.5`, `1.234567890123456789012345678905E29`, `123456789012345678901234567890.5`},
	}
	for _, c := range cases {
		ref := c.ref
		if ref == "" {
			ref = c.spec
		}
		got, err := lcp.Canonical([]byte(c.in))
		if err != nil || string(got) != c.spec {
			t.Errorf("%s: Canonical(%s) = %q, %v; want %q", c.name, c.in, got, err, c.spec)
		}
		got, err = lcp.CanonicalAsReference([]byte(c.in))
		if err != nil || string(got) != ref {
			t.Errorf("%s: CanonicalAsReference(%s) = %q, %v; want %q", c.name, c.in, got, err, ref)
		}
	}
}

func TestHRD070CanonicalRejections(t *testing.T) {
	syn, amb := lcp.ErrLicenseSyntax, lcp.ErrAmbiguousLicense
	cases := []struct {
		name, in   string
		want       error
		numberOnly bool
	}{
		{"duplicate top level", `{"a":1,"a":2}`, amb, false},
		{"duplicate nested object", `{"x":{"a":1,"b":2,"a":3}}`, amb, false},
		{"duplicate inside array element", `{"x":[{"a":1},{"b":1,"b":2}]}`, amb, false},
		{"duplicate far apart", `{"a":1,"b":2,"c":3,"a":4}`, amb, false},
		{"same key in sibling objects is fine", `{"x":{"a":1},"y":{"a":2}}`, nil, false},
		{"trailing value", `{} {}`, syn, false},
		{"trailing garbage", `{}x`, syn, false},
		{"trailing scalar after scalar", `1 2`, syn, false},
		{"empty", ``, syn, false},
		{"whitespace only", " \n", syn, false},
		{"truncated object", `{"a":1`, syn, false},
		{"truncated array", `[1`, syn, false},
		{"truncated nested", `{"a":[1,{"b":2}`, syn, false},
		{"missing value", `{"a":}`, syn, false},
		{"number key", `{1:2}`, syn, false},
		{"single quotes", `{'a':1}`, syn, false},
		{"stray closing", `}`, syn, false},
		{"bare word", `nul`, syn, false},
		{"bad exponent literal", `1e99999999999999999999`, syn, true},
		{"exponent out of range", `1e2147483648`, syn, true},
		{"negative exponent out of range", `1e-2147483648`, syn, true},
		{"exponent one past the upper limit", `1e1073741825`, syn, true},
		{"exponent one past the lower limit", `1e-1073741825`, syn, true},
		{"bad exponent as an object member value", `{"a":1e2147483648}`, syn, true},
		{"bad exponent inside structure", `{"a":[1e99999999999999999999]}`, syn, true},
	}
	for _, c := range cases {
		for mode, f := range map[string]func([]byte) ([]byte, error){"spec": lcp.Canonical, "reference": lcp.CanonicalAsReference} {
			if mode == "reference" && c.numberOnly {
				continue // reference mode passes numbers through (asserted below)
			}
			out, err := f([]byte(c.in))
			switch {
			case c.want == nil:
				if err != nil || out == nil {
					t.Errorf("%s/%s: %q rejected: %v", c.name, mode, c.in, err)
				}
			case out != nil || !errors.Is(err, c.want):
				t.Errorf("%s/%s: %q -> (%q, %v), want (nil, %v)", c.name, mode, c.in, out, err, c.want)
			case c.want == syn && errors.Is(err, amb), c.want == amb && errors.Is(err, syn):
				t.Errorf("%s/%s: error classes are not exclusive: %v", c.name, mode, err)
			}
		}
	}
	// reference mode passes a number through untouched, so it never fails on it.
	if out, err := lcp.CanonicalAsReference([]byte(`1e99999999999999999999`)); err != nil || string(out) != `1e99999999999999999999` {
		t.Errorf("reference mode must pass numbers through: %q %v", out, err)
	}
}

func TestHRD070CanonicalForSigning(t *testing.T) {
	syn, amb, miss := lcp.ErrLicenseSyntax, lcp.ErrAmbiguousLicense, lcp.ErrSignatureMissing
	good := []struct{ name, in, spec, ref string }{
		{"signature removed, rest canonical", `{"b":1,"signature":{"z":1},"a":2}`, `{"a":2,"b":1}`, ""},
		{"whitespace around colon and comma", "{ \"b\" : 1 , \"signature\" : {\"x\":1} , \"a\" :  2 }", `{"a":2,"b":1}`, ""},
		{"signature first", `{"signature":"s","a":[2,1]}`, `{"a":[2,1]}`, ""},
		{"signature last", `{"a":{"d":1,"c":2},"signature":null}`, `{"a":{"c":2,"d":1}}`, ""},
		{"signature only", `{"signature":{}}`, `{}`, ""},
		{"nested signature members are kept", `{"a":{"signature":1},"signature":2}`, `{"a":{"signature":1}}`, ""},
		{"escapes keep the mode", `{"k":"\u001f","signature":1}`, `{"k":"\u001F"}`, `{"k":"\u001f"}`},
		{"number rules by mode", `{"n":1.50,"signature":1}`, `{"n":1.5E0}`, `{"n":1.50}`},
	}
	for _, c := range good {
		ref := c.ref
		if ref == "" {
			ref = c.spec
		}
		got, err := lcp.CanonicalForSigning([]byte(c.in))
		if err != nil || string(got) != c.spec {
			t.Errorf("%s: CanonicalForSigning = %q, %v; want %q", c.name, got, err, c.spec)
		}
		got, err = lcp.CanonicalForSigningMode([]byte(c.in), lcp.CanonicalReference)
		if err != nil || string(got) != ref {
			t.Errorf("%s: reference = %q, %v; want %q", c.name, got, err, ref)
		}
	}
	// the package's own wording for a non-object top level (not encoding/json's text)
	for _, in := range []string{`[1]`, `"x"`, `7`, `null`} {
		if _, err := lcp.CanonicalForSigning([]byte(in)); err == nil || !strings.Contains(err.Error(), "top level is not an object") {
			t.Errorf("%s: want the top-level-not-an-object error, got %v", in, err)
		}
	}
	bad := []struct {
		name, in string
		want     error
	}{
		{"no signature member", `{"a":1}`, miss},
		{"empty object", `{}`, miss},
		{"signature only nested", `{"a":{"signature":1}}`, miss},
		{"top level is an array", `[1]`, syn},
		{"top level is a scalar", `"x"`, syn},
		{"empty input", ``, syn},
		{"truncated before the end", `{"a":1,"signature":2`, syn},
		{"object never closed after complete members", `{"a":"x","signature":"y"`, syn},
		{"truncated inside a value", `{"a":[1,`, syn},
		{"invalid value", `{"signature":1,"a":}`, syn},
		{"missing key", `{:1}`, syn},
		{"duplicate outside the signature", `{"a":1,"a":2,"signature":3}`, amb},
	}
	for _, c := range bad {
		for mode, f := range map[string]func([]byte) ([]byte, error){
			"spec":      lcp.CanonicalForSigning,
			"reference": func(b []byte) ([]byte, error) { return lcp.CanonicalForSigningMode(b, lcp.CanonicalReference) },
		} {
			out, err := f([]byte(c.in))
			if out != nil || !errors.Is(err, c.want) {
				t.Errorf("%s/%s: %q -> (%q, %v), want (nil, %v)", c.name, mode, c.in, out, err, c.want)
			}
		}
	}
}

// TestHRD070LinkLength: Link.UnmarshalJSON accepts a non-negative integer length
// as a JSON number or a string, treats absent/null as 0 and rejects the rest.
func TestHRD070LinkLength(t *testing.T) {
	ok := []struct {
		in   string
		want int64
	}{
		{`{"href":"h"}`, 0}, {`{"href":"h","length":null}`, 0}, {`{"href":"h","length":0}`, 0},
		{`{"href":"h","length":5}`, 5}, {`{"href":"h","length":"7"}`, 7},
		{`{"href":"h","length":9223372036854775807}`, 9223372036854775807},
	}
	for _, c := range ok {
		var l lcp.Link
		if err := json.Unmarshal([]byte(c.in), &l); err != nil || l.Length != c.want || l.Href != "h" {
			t.Errorf("%s: got %+v, %v; want length %d", c.in, l, err, c.want)
		}
	}
	for _, in := range []string{
		`{"href":"h","length":-1}`, `{"href":"h","length":" 9 "}`, `{"href":"h","length":"-1"}`, `{"href":"h","length":1.5}`, `{"href":"h","length":"abc"}`,
		`{"href":"h","length":""}`, `{"href":"h","length":true}`, `{"href":"h","length":[1]}`, `{"href":"h","length":{}}`,
		`{"href":"h","length":9223372036854775808}`, `{"href":"h","length":"0x5"}`, `{"href":5}`, `"x"`, `[]`,
	} {
		var l lcp.Link
		if err := json.Unmarshal([]byte(in), &l); err == nil {
			t.Errorf("%s accepted as %+v", in, l)
		}
	}
	// a reused Link must not keep the previous length
	l := lcp.Link{Length: 99}
	if err := json.Unmarshal([]byte(`{"href":"h"}`), &l); err != nil || l.Length != 0 {
		t.Errorf("stale length kept: %+v %v", l, err)
	}
}

// TestHRD070StatusVocabularyAndRefresh: every legal status word is accepted
// and only those; NeedsLicenseRefresh implements SPEC l.244.
func TestHRD070StatusVocabularyAndRefresh(t *testing.T) {
	for _, st := range []string{"ready", "active", "revoked", "returned", "cancelled", "expired"} {
		m := baseLSD()
		m["status"] = st
		doc, _ := json.Marshal(m)
		if s, err := lcp.ParseStatusDocument(doc); err != nil || string(s.Status) != st {
			t.Errorf("status %q: %v %v", st, s, err)
		}
	}
	for _, st := range []string{"", "Active", "ACTIVE", "activ", "active ", "bogus", "overdue"} {
		m := baseLSD()
		m["status"] = st
		doc, _ := json.Marshal(m)
		if s, err := lcp.ParseStatusDocument(doc); s != nil || !errors.Is(err, lcp.ErrStatusDocument) {
			t.Errorf("status %q accepted: %v %v", st, s, err)
		}
	}
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	status := &lcp.StatusDocument{UpdatedLicense: t0.Add(time.Hour)}
	for _, c := range []struct {
		name   string
		s      *lcp.StatusDocument
		l      *lcp.License
		want   bool
		reason string
	}{
		{"nil status", nil, &lcp.License{Updated: t0}, false, "a nil status says nothing"},
		{"nil status and nil license", nil, nil, false, "status nil wins"},
		{"nil license", status, nil, true, "always stale"},
		{"status newer than updated", status, &lcp.License{Updated: t0}, true, ""},
		{"status equal to updated", &lcp.StatusDocument{UpdatedLicense: t0}, &lcp.License{Updated: t0}, false, "not after"},
		{"status older than updated", &lcp.StatusDocument{UpdatedLicense: t0.Add(-time.Hour)}, &lcp.License{Updated: t0}, false, ""},
		{"no updated falls back to issued (stale)", status, &lcp.License{Issued: t0}, true, ""},
		{"no updated falls back to issued (current)", &lcp.StatusDocument{UpdatedLicense: t0}, &lcp.License{Issued: t0}, false, ""},
		{"updated wins over a later issued", &lcp.StatusDocument{UpdatedLicense: t0.Add(30 * time.Minute)}, &lcp.License{Issued: t0.Add(time.Hour), Updated: t0}, true, "updated, not issued, is the baseline"},
	} {
		if got := lcp.NeedsLicenseRefresh(c.s, c.l); got != c.want {
			t.Errorf("%s: got %v want %v (%s)", c.name, got, c.want, c.reason)
		}
	}
}

// TestHRD070LSDLinkHasRel: relation lookup over the array form, any position.
func TestHRD070LSDLinkHasRel(t *testing.T) {
	l := lcp.LSDLink{Rels: []string{"a", "b", "license"}}
	for rel, want := range map[string]bool{"a": true, "b": true, "license": true, "": false, "x": false, "License": false, "lic": false} {
		if got := l.HasRel(rel); got != want {
			t.Errorf("HasRel(%q) = %v, want %v", rel, got, want)
		}
	}
	if (lcp.LSDLink{}).HasRel("license") {
		t.Error("a link without relations has no relation")
	}
}

// TestHRD070LSDLinkRelForms: a link's rel is a string or an array of strings
// (link.schema.json); the array form must reach HasRel and the parsed Rels.
func TestHRD070LSDLinkRelForms(t *testing.T) {
	parse := func(links []any) (*lcp.StatusDocument, error) {
		m := baseLSD()
		m["links"] = links
		doc, _ := json.Marshal(m)
		return lcp.ParseStatusDocument(doc)
	}
	l := func(rel any) map[string]any { return map[string]any{"rel": rel, "href": "https://example.org/x"} }
	s, err := parse([]any{l([]any{"self", "license"})})
	if err != nil || len(s.Links) != 1 || len(s.Links[0].Rels) != 2 || s.Links[0].Rels[0] != "self" || s.Links[0].Rels[1] != "license" || !s.Links[0].HasRel("license") {
		t.Fatalf("array rel: %+v %v", s, err)
	}
	// an empty array is legal for a link but cannot satisfy the license requirement alone
	if s, err := parse([]any{l([]any{}), l("license")}); err != nil || len(s.Links[0].Rels) != 0 || !s.Links[1].HasRel("license") {
		t.Fatalf("empty array rel next to a license link: %+v %v", s, err)
	}
	if s, err := parse([]any{l([]any{})}); s != nil || !errors.Is(err, lcp.ErrStatusDocument) {
		t.Fatalf("empty array rel alone must fail (no license link): %+v %v", s, err)
	}
	for _, bad := range []any{5, true, []any{1}, map[string]any{}, ""} {
		if s, err := parse([]any{l(bad), l("license")}); s != nil || !errors.Is(err, lcp.ErrStatusDocument) {
			t.Errorf("rel %v accepted: %+v %v", bad, s, err)
		}
	}
}
