// Package lcpfuzz holds fuzz targets (HRD-070 / GAP-035) for the LCP License
// Document, Status Document and canonicalisation parsers. It lives outside
// core/pkg/lcp and uses only that package's exported API. The rule table that
// maps every enforced check to its oracle, seed and mutant is
// evidence/campaign-2026-11-23/gap-fix/hrd070-r4/RULES.md.
package lcpfuzz

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/lcp"
)

const specLicense = "../lcp/testdata/spec-5.4.1-signed-license.json"

// baseLicense is the spec example plus the id/provider it omits: a document
// lcp.Parse accepts. Every boundary seed is derived from it.
func baseLicense() map[string]any {
	b, err := os.ReadFile(specLicense)
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	m["id"] = "urn:uuid:fuzz-seed"
	m["provider"] = "https://provider.example"
	return m
}

// licenseVariant returns the base license after edit, marshalled.
func licenseVariant(edit func(m map[string]any)) []byte {
	m := baseLicense()
	if m == nil {
		return nil
	}
	edit(m)
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return b
}

func sub(m map[string]any, k string) map[string]any { v, _ := m[k].(map[string]any); return v }

// licenseSeeds: one boundary seed per rule of lcp.Parse (see RULES.md).
func licenseSeeds() [][]byte {
	var out [][]byte
	add := func(edit func(m map[string]any)) {
		if b := licenseVariant(edit); b != nil {
			out = append(out, b)
		}
	}
	add(func(m map[string]any) {}) // accepted
	for _, k := range []string{"id", "provider", "issued", "encryption", "links", "signature"} {
		k := k
		add(func(m map[string]any) { delete(m, k) })
	}
	for _, k := range []string{"id", "provider", "issued"} {
		k := k
		add(func(m map[string]any) { m[k] = "" })
	}
	for _, p := range [][2]string{{"encryption", "profile"}, {"encryption", "content_key"}, {"encryption", "user_key"}, {"signature", "algorithm"}, {"signature", "certificate"}, {"signature", "value"}} {
		p := p
		add(func(m map[string]any) { delete(sub(m, p[0]), p[1]) })
		add(func(m map[string]any) { sub(m, p[0])[p[1]] = "" })
	}
	for _, k := range []string{"encrypted_value", "algorithm"} {
		k := k
		add(func(m map[string]any) { delete(sub(sub(m, "encryption"), "content_key"), k) })
		add(func(m map[string]any) { sub(sub(m, "encryption"), "content_key")[k] = "" })
	}
	for _, k := range []string{"algorithm", "key_check", "text_hint"} {
		k := k
		add(func(m map[string]any) { delete(sub(sub(m, "encryption"), "user_key"), k) })
	}
	for _, k := range []string{"algorithm", "key_check"} {
		k := k
		add(func(m map[string]any) { sub(sub(m, "encryption"), "user_key")[k] = "" })
	}
	add(func(m map[string]any) { sub(sub(m, "encryption"), "user_key")["text_hint"] = "" }) // empty hint is legal
	add(func(m map[string]any) { m["issued"] = "not-a-date" })
	add(func(m map[string]any) { m["updated"] = "not-a-date" })
	add(func(m map[string]any) { delete(m, "updated") }) // legal: updated defaults to issued
	add(func(m map[string]any) { m["updated"] = "" })    // legal: empty means absent
	add(func(m map[string]any) { m["links"] = []any{} })
	add(func(m map[string]any) { m["links"] = []any{map[string]any{"rel": "hint", "href": ""}} })
	add(func(m map[string]any) { m["rights"] = "bad" })
	add(func(m map[string]any) { m["rights"] = map[string]any{"print": "x"} })
	add(func(m map[string]any) {
		m["rights"] = map[string]any{"print": 10, "copy": 2, "start": "2020-01-01T00:00:00Z", "end": "2021-01-01T00:00:00Z"}
	})
	add(func(m map[string]any) { m["rights"] = map[string]any{"start": "not-a-date"} })
	add(func(m map[string]any) { m["rights"] = nil })
	add(func(m map[string]any) { // many unknown members: report order must be sorted
		for i := 0; i < 24; i++ {
			m[fmt.Sprintf("x-unknown-%02d", (i*7)%24)] = i
		}
	})
	return out
}

func seed(f *testing.F) {
	for _, p := range []string{specLicense, "../lcp/testdata/spec-5.3.1-canonical.json", "../lcp/testdata/spec-5.3.1-pretty.json"} {
		if b, err := os.ReadFile(p); err == nil {
			f.Add(b)
		}
	}
	for _, d := range licenseSeeds() {
		f.Add(d)
	}
	for _, s := range []string{``, `{}`, `null`, `[]`, `{"id":"x","provider":"p","issued":"2020-01-01T00:00:00Z"}`, `{"status":"active","id":"a","message":"m","updated":{},"links":[]}`, `{"a":{"b":[1,2,3]},"c":"\u0000"}`, "\xff\xfe"} {
		f.Add([]byte(s))
	}
	if b := licenseVariant(func(m map[string]any) {}); b != nil {
		f.Add(append(append([]byte{}, b...), []byte(" {}")...)) // trailing data
	}
}

// ci looks a member up the way encoding/json matches struct fields: exact name
// first, otherwise case-insensitively. Documents holding two members whose
// names differ only in case are skipped by the oracles (hasCaseDup), because
// which one encoding/json keeps is not part of the contract under test.
func ci(m map[string]any, k string) (any, bool) {
	if v, ok := m[k]; ok {
		return v, true
	}
	for kk, v := range m {
		if strings.EqualFold(kk, k) {
			return v, true
		}
	}
	return nil, false
}

func hasCaseDup(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		seen := map[string]bool{}
		for k, c := range x {
			lk := strings.ToLower(k)
			if seen[lk] || hasCaseDup(c) {
				return true
			}
			seen[lk] = true
		}
	case []any:
		for _, c := range x {
			if hasCaseDup(c) {
				return true
			}
		}
	}
	return false
}

func str(m map[string]any, k string) (string, bool) {
	v, _ := ci(m, k)
	s, ok := v.(string)
	return s, ok
}
func obj(m map[string]any, k string) map[string]any {
	v, _ := ci(m, k)
	o, _ := v.(map[string]any)
	return o
}

// FuzzParseLicense: Parse must never panic and must reject only with a typed
// error; an accepted license must satisfy EVERY rule of the contract, checked
// against an independent decoding of the same bytes (RULES.md, table L).
func FuzzParseLicense(f *testing.F) {
	seed(f)
	f.Fuzz(func(t *testing.T, doc []byte) {
		if len(doc) > 1<<20 {
			t.Skip("bound input")
		}
		l, err := lcp.Parse(doc)
		if err != nil {
			if l != nil {
				t.Fatalf("error %v returned a non-nil license", err)
			}
			if !errors.Is(err, lcp.ErrLicenseSyntax) && !errors.Is(err, lcp.ErrIncomplete) {
				t.Fatalf("error not typed ErrLicenseSyntax/ErrIncomplete: %v", err)
			}
			return
		}
		if l == nil {
			t.Fatal("nil license with nil error")
		}
		var m map[string]any
		if json.Unmarshal(doc, &m) != nil { // also rejects trailing data
			t.Fatalf("accepted a document that is not a single JSON object")
		}
		if hasCaseDup(m) {
			t.Skip("members differing only in case: encoding/json's choice is out of scope")
		}
		if id, _ := str(m, "id"); id == "" || id != l.ID {
			t.Fatalf("id: %q vs %q", id, l.ID)
		}
		if p, _ := str(m, "provider"); p == "" || p != l.Provider {
			t.Fatalf("provider: %q vs %q", p, l.Provider)
		}
		is, _ := str(m, "issued")
		it, ierr := time.Parse(time.RFC3339, is)
		if is == "" || ierr != nil || !it.Equal(l.Issued) {
			t.Fatalf("issued %q accepted but not a valid RFC 3339 date-time equal to %v (%v)", is, l.Issued, ierr)
		}
		if us, _ := str(m, "updated"); us != "" {
			ut, uerr := time.Parse(time.RFC3339, us)
			if uerr != nil || !ut.Equal(l.Updated) {
				t.Fatalf("updated %q accepted but not valid / equal to %v (%v)", us, l.Updated, uerr)
			}
		} else if !l.Updated.Equal(l.Issued) {
			t.Fatalf("absent updated must default to issued: %v vs %v", l.Updated, l.Issued)
		}
		enc := obj(m, "encryption")
		ck := obj(enc, "content_key")
		uk := obj(enc, "user_key")
		for name, v := range map[string]string{"profile": l.Encryption.Profile, "content_key.encrypted_value": l.Encryption.ContentKey.EncryptedValue,
			"content_key.algorithm": l.Encryption.ContentKey.Algorithm, "user_key.algorithm": l.Encryption.UserKey.Algorithm, "user_key.key_check": l.Encryption.UserKey.KeyCheck} {
			if v == "" {
				t.Fatalf("accepted license lacks encryption %s", name)
			}
		}
		if p, _ := str(enc, "profile"); p != l.Encryption.Profile {
			t.Fatalf("profile mismatch")
		}
		if v, _ := str(ck, "encrypted_value"); v != l.Encryption.ContentKey.EncryptedValue {
			t.Fatalf("content_key.encrypted_value mismatch")
		}
		if _, present := ci(uk, "text_hint"); !present {
			t.Fatalf("accepted license lacks user_key.text_hint member")
		}
		lv, _ := ci(m, "links")
		links, _ := lv.([]any)
		if len(links) == 0 || len(links) != len(l.Links) {
			t.Fatalf("links: raw %d parsed %d", len(links), len(l.Links))
		}
		for i, k := range l.Links {
			if k.Href == "" {
				t.Fatalf("accepted license link %d has empty href", i)
			}
		}
		if l.Signature.Algorithm == "" || l.Signature.Certificate == "" || l.Signature.Value == "" {
			t.Fatalf("accepted license lacks signature members: %+v", l.Signature)
		}
		if r, present := ci(m, "rights"); present && r != nil {
			rb, _ := json.Marshal(r)
			var rr lcp.Rights
			if json.Unmarshal(rb, &rr) != nil {
				t.Fatalf("malformed rights %s accepted", rb)
			}
		}
		if !sort.StringsAreSorted(l.UnknownMembers) {
			t.Fatalf("UnknownMembers not sorted: %v", l.UnknownMembers)
		}
		for _, k := range l.UnknownMembers {
			if _, ok := m[k]; !ok {
				t.Fatalf("UnknownMembers names %q which is not in the document", k)
			}
			switch k {
			case "id", "issued", "updated", "provider", "encryption", "links", "rights", "user", "signature":
				t.Fatalf("known member %q reported as unknown", k)
			}
		}
		nUnknown := 0
		for k := range m {
			switch k {
			case "id", "issued", "updated", "provider", "encryption", "links", "rights", "user", "signature":
			default:
				nUnknown++
			}
		}
		if nUnknown != len(l.UnknownMembers) {
			t.Fatalf("UnknownMembers %d vs %d unknown members in the document", len(l.UnknownMembers), nUnknown)
		}
	})
}

// FuzzCanonical: canonicalisation must never panic and must be idempotent
// (canonical(canonical(x)) == canonical(x)) whenever it succeeds.
func FuzzCanonical(f *testing.F) {
	seed(f)
	f.Fuzz(func(t *testing.T, doc []byte) {
		if len(doc) > 1<<20 {
			t.Skip("bound input")
		}
		c1, err := lcp.Canonical(doc)
		if err != nil {
			return
		}
		c2, err := lcp.Canonical(c1)
		if err != nil {
			t.Fatalf("canonical output is not re-canonicalisable: %v", err)
		}
		if !bytes.Equal(c1, c2) {
			t.Fatalf("canonical not idempotent:\n%q\n%q", c1, c2)
		}
		_, _ = lcp.CanonicalForSigning(doc)
		_, _ = lcp.CanonicalAsReference(doc)
	})
}

// TestSeedLicenseIsAccepted pins the seed oracle: the augmented spec example
// must parse, otherwise the missing-member seeds above prove nothing.
func TestSeedLicenseIsAccepted(t *testing.T) {
	b, err := os.ReadFile("../lcp/testdata/spec-5.4.1-signed-license.json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	m["id"] = json.RawMessage(`"urn:uuid:fuzz-seed"`)
	m["provider"] = json.RawMessage(`"https://provider.example"`)
	nb, _ := json.Marshal(m)
	if _, err := lcp.Parse(nb); err != nil {
		t.Fatalf("augmented spec license rejected: %v", err)
	}
}
