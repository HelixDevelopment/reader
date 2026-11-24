// Fuzz target for the LCP Status Document (LSD) parser in core/pkg/lcp/lsd.go.
// No build tag is needed:
//   go test ./pkg/lcpfuzz -run '^$' -fuzz '^FuzzParseStatusDocument$'

package lcpfuzz

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/helixreader/helix/core/pkg/lcp"
)

// FuzzParseStatusDocument: LSD parser must never panic and a nil error must
// imply a non-nil document.
func FuzzParseStatusDocument(f *testing.F) {
	seed(f)
	for _, d := range lsdSeeds() {
		f.Add(d)
	}
	f.Fuzz(func(t *testing.T, doc []byte) {
		if len(doc) > 1<<20 {
			t.Skip("bound input")
		}
		s, err := lcp.ParseStatusDocument(doc)
		if err != nil {
			if s != nil || !errors.Is(err, lcp.ErrStatusDocument) {
				t.Fatalf("bad rejection (doc=%v): %v", s, err)
			}
			return
		}
		if s == nil {
			t.Fatal("nil status document with nil error")
		}
		// Mandatory members (LSD schema) must be populated on acceptance.
		if s.ID == "" {
			t.Fatal("accepted status document lacks id")
		}
		switch s.Status {
		case lcp.StatusReady, lcp.StatusActive, lcp.StatusRevoked, lcp.StatusReturned, lcp.StatusCancelled, lcp.StatusExpired:
		default:
			t.Fatalf("accepted status outside vocabulary: %q", s.Status)
		}
		if (s.UpdatedLicense.IsZero() || s.UpdatedStatus.IsZero()) && !bytes.Contains(doc, []byte("0001-01-01")) {
			t.Fatal("accepted status document lacks updated timestamps")
		}
		hasLicense := false
		for i, l := range s.Links {
			if l.Href == "" {
				t.Fatalf("link %d has empty href", i)
			}
			hasLicense = hasLicense || l.HasRel("license")
		}
		if !hasLicense {
			t.Fatal("accepted status document has no license link")
		}
		// Must be safe to evaluate against an empty local license.
		_ = lcp.NeedsLicenseRefresh(s, nil)
	})
}

const validLSD = `{"id":"urn:uuid:lsd-seed","status":"active","message":"ok","updated":{"license":"2020-01-01T00:00:00Z","status":"2020-01-02T00:00:00Z"},"links":[{"rel":"license","href":"https://provider.example/lic","type":"application/vnd.readium.lcp.license.v1.0+json"}]}`

// lsdSeeds returns a valid status document plus variants each missing one
// mandatory member, so the acceptance oracle is exercised on both sides.
func lsdSeeds() [][]byte {
	out := [][]byte{[]byte(validLSD)}
	var base map[string]json.RawMessage
	if json.Unmarshal([]byte(validLSD), &base) != nil {
		return out
	}
	for _, k := range []string{"id", "status", "message", "updated", "links"} {
		m := map[string]json.RawMessage{}
		for kk, v := range base {
			if kk != k {
				m[kk] = v
			}
		}
		if nb, err := json.Marshal(m); err == nil {
			out = append(out, nb)
		}
	}
	out = append(out,
		[]byte(`{"id":"x","status":"bogus","message":"m","updated":{"license":"2020-01-01T00:00:00Z","status":"2020-01-01T00:00:00Z"},"links":[{"rel":"license","href":"h"}]}`),
		[]byte(`{"id":"x","status":"active","message":"m","updated":{"license":"2020-01-01T00:00:00Z","status":"2020-01-01T00:00:00Z"},"links":[{"rel":"status","href":"h"}]}`))
	return out
}

// TestSeedStatusIsAccepted pins the seed oracle: validLSD must parse.
func TestSeedStatusIsAccepted(t *testing.T) {
	if _, err := lcp.ParseStatusDocument([]byte(validLSD)); err != nil {
		t.Fatalf("valid LSD seed rejected: %v", err)
	}
}
