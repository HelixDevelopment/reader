// Package lcpfuzz holds fuzz targets (HRD-070 / GAP-035) for the LCP License
// Document, Status Document and canonicalisation parsers. It lives outside
// core/pkg/lcp and uses only that package's exported API.
package lcpfuzz

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/helixreader/helix/core/pkg/lcp"
)

func seed(f *testing.F) {
	for _, p := range []string{"../lcp/testdata/spec-5.4.1-signed-license.json", "../lcp/testdata/spec-5.3.1-canonical.json", "../lcp/testdata/spec-5.3.1-pretty.json"} {
		if b, err := os.ReadFile(p); err == nil {
			f.Add(b)
			// The spec example omits id/provider; add them to get a document
			// that Parse accepts, then remove one mandatory member at a time: a
			// parser that stops enforcing it would accept an incomplete license.
			var base map[string]json.RawMessage
			if json.Unmarshal(b, &base) != nil {
				continue
			}
			base["id"] = json.RawMessage(`"urn:uuid:fuzz-seed"`)
			base["provider"] = json.RawMessage(`"https://provider.example"`)
			if nb, err := json.Marshal(base); err == nil {
				f.Add(nb)
			}
			for _, k := range []string{"id", "provider", "issued", "encryption", "links", "signature"} {
				m := map[string]json.RawMessage{}
				for kk, v := range base {
					if kk != k {
						m[kk] = v
					}
				}
				if nb, err := json.Marshal(m); err == nil {
					f.Add(nb)
				}
			}
		}
	}
	for _, s := range []string{``, `{}`, `null`, `[]`, `{"id":"x","provider":"p","issued":"2020-01-01T00:00:00Z"}`, `{"status":"active","id":"a","message":"m","updated":{},"links":[]}`, `{"a":{"b":[1,2,3]},"c":"\u0000"}`, "\xff\xfe"} {
		f.Add([]byte(s))
	}
}

// FuzzParseLicense: Parse must never panic; a nil error must imply a non-nil
// license with the mandatory members (§7.2) populated.
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
			if !errors.Is(err, lcp.ErrLicenseSyntax) && !errors.Is(err, lcp.ErrIncomplete) && err.Error() == "" {
				t.Fatalf("untyped empty error")
			}
			return
		}
		if l == nil || l.ID == "" || l.Provider == "" {
			t.Fatalf("accepted license lacks mandatory members: %+v", l)
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
