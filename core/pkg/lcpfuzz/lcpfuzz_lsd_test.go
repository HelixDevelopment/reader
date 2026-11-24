// Fuzz target for the LCP Status Document (LSD) parser in core/pkg/lcp/lsd.go.
// No build tag is needed:
//   go test ./pkg/lcpfuzz -run '^$' -fuzz '^FuzzParseStatusDocument$'
// The rule table (table S) is evidence/campaign-2026-11-23/gap-fix/hrd070-r4/RULES.md.

package lcpfuzz

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/lcp"
)

const validLSD = `{"id":"urn:uuid:lsd-seed","status":"active","message":"ok","updated":{"license":"2020-01-01T00:00:00Z","status":"2020-01-02T00:00:00Z"},"links":[{"rel":"license","href":"https://provider.example/lic","type":"application/vnd.readium.lcp.license.v1.0+json"}]}`

func baseLSD() map[string]any {
	var m map[string]any
	_ = json.Unmarshal([]byte(validLSD), &m)
	return m
}

// lsdSeeds returns one boundary seed per rule of lcp.ParseStatusDocument.
func lsdSeeds() [][]byte {
	var out [][]byte
	add := func(edit func(m map[string]any)) {
		m := baseLSD()
		edit(m)
		if b, err := json.Marshal(m); err == nil {
			out = append(out, b)
		}
	}
	link := func(rel any, href any) []any { return []any{map[string]any{"rel": rel, "href": href}} }
	add(func(m map[string]any) {})
	for _, k := range []string{"id", "status", "message", "updated", "links"} {
		k := k
		add(func(m map[string]any) { delete(m, k) })
	}
	add(func(m map[string]any) { m["id"] = "" })
	add(func(m map[string]any) { m["message"] = "" }) // legal
	add(func(m map[string]any) { m["status"] = "bogus" })
	for _, st := range []string{"ready", "revoked", "returned", "cancelled", "expired"} {
		st := st
		add(func(m map[string]any) { m["status"] = st })
	}
	add(func(m map[string]any) { delete(m["updated"].(map[string]any), "license") })
	add(func(m map[string]any) { delete(m["updated"].(map[string]any), "status") })
	add(func(m map[string]any) { m["updated"].(map[string]any)["license"] = "not-a-date" })
	add(func(m map[string]any) { m["updated"].(map[string]any)["status"] = "not-a-date" })
	add(func(m map[string]any) { m["links"] = link("license", "") })
	add(func(m map[string]any) { m["links"] = []any{map[string]any{"rel": "license"}} }) // href absent
	add(func(m map[string]any) { m["links"] = link("status", "https://p/x") })           // no license rel
	add(func(m map[string]any) { m["links"] = []any{} })
	for _, rel := range []any{nil, "", 5, []any{1}, map[string]any{}} {
		rel := rel
		// A valid license link first, then a link with a bad rel: the license-link
		// rule must not mask the per-link rel rule. (Single-link variants are kept
		// below to pin the license-link rule itself.)
		add(func(m map[string]any) {
			bad := map[string]any{"href": "h"}
			if rel != nil {
				bad["rel"] = rel
			}
			m["links"] = []any{map[string]any{"rel": "license", "href": "h"}, bad}
		})
	}
	add(func(m map[string]any) { m["links"] = link([]any{}, "h") })               // empty rel array: no license rel
	add(func(m map[string]any) { m["links"] = link([]any{"license", "x"}, "h") }) // rel array
	add(func(m map[string]any) { m["links"] = link([]any{1}, "h") })              // rel array of non-strings
	add(func(m map[string]any) { m["potential_rights"] = map[string]any{"end": "2030-01-01T00:00:00Z"} })
	add(func(m map[string]any) { m["potential_rights"] = map[string]any{"end": "not-a-date"} })
	add(func(m map[string]any) { m["potential_rights"] = map[string]any{} })
	add(func(m map[string]any) {
		m["events"] = []any{map[string]any{"type": "register", "name": "n", "id": "i", "timestamp": "2020-01-01T00:00:00Z"}, map[string]any{"name": "no type or time"}}
	})
	for _, ty := range []string{"renew", "return", "revoke", "cancel"} {
		ty := ty
		add(func(m map[string]any) { m["events"] = []any{map[string]any{"type": ty}} })
	}
	add(func(m map[string]any) { m["events"] = []any{map[string]any{"type": "bogus"}} })
	add(func(m map[string]any) {
		m["events"] = []any{map[string]any{"type": "register", "timestamp": "not-a-date"}}
	})
	out = append(out, []byte(validLSD+" {}"), []byte(`[]`), []byte(`"x"`), []byte(`{`))
	return out
}

func rawStr(m map[string]any, k string) (string, bool) { return str(m, k) }

// FuzzParseStatusDocument: the LSD parser must never panic, must reject only
// with ErrStatusDocument and an accepted document must satisfy every rule of
// the contract against an independent decoding of the same bytes.
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
		var m map[string]any
		if !json.Valid(doc) || json.Unmarshal(doc, &m) != nil {
			t.Fatal("accepted a document that is not a single JSON object")
		}
		if hasCaseDup(m) {
			t.Skip("members differing only in case: encoding/json's choice is out of scope")
		}
		if id, _ := rawStr(m, "id"); id == "" || id != s.ID {
			t.Fatalf("id %q vs %q", id, s.ID)
		}
		if _, ok := rawStr(m, "message"); !ok {
			t.Fatal("accepted without a message member")
		}
		switch st, _ := rawStr(m, "status"); st {
		case "ready", "active", "revoked", "returned", "cancelled", "expired":
			if string(s.Status) != st {
				t.Fatalf("status %q vs %q", st, s.Status)
			}
		default:
			t.Fatalf("accepted status outside vocabulary: %q", st)
		}
		upd := obj(m, "updated")
		for name, got := range map[string]time.Time{"license": s.UpdatedLicense, "status": s.UpdatedStatus} {
			v, _ := rawStr(upd, name)
			tv, perr := time.Parse(time.RFC3339, v)
			if perr != nil || !tv.Equal(got) {
				t.Fatalf("updated.%s %q accepted but not a valid date-time equal to %v (%v)", name, v, got, perr)
			}
		}
		lv, _ := ci(m, "links")
		links, ok := lv.([]any)
		if !ok || len(links) != len(s.Links) {
			t.Fatalf("links: raw %v parsed %d", lv, len(s.Links))
		}
		hasLicense := false
		for i, raw := range links {
			lm, _ := raw.(map[string]any)
			if h, _ := rawStr(lm, "href"); h == "" || h != s.Links[i].Href {
				t.Fatalf("link %d href %q vs %q", i, h, s.Links[i].Href)
			}
			relv, _ := ci(lm, "rel")
			switch rel := relv.(type) {
			case string:
				if rel == "" {
					t.Fatalf("link %d accepted with empty rel", i)
				}
			case []any:
				for _, r := range rel {
					if _, isStr := r.(string); !isStr {
						t.Fatalf("link %d rel array holds a non-string", i)
					}
				}
			default:
				t.Fatalf("link %d accepted with rel %v (absent or wrong type)", i, relv)
			}
			hasLicense = hasLicense || s.Links[i].HasRel("license")
		}
		if !hasLicense {
			t.Fatal("accepted status document has no license link")
		}
		if pr := obj(m, "potential_rights"); pr != nil {
			if end, has := rawStr(pr, "end"); has {
				te, perr := time.Parse(time.RFC3339, end)
				if perr != nil || s.PotentialRightsEnd == nil || !te.Equal(*s.PotentialRightsEnd) {
					t.Fatalf("potential_rights.end %q accepted but not valid / equal (%v)", end, perr)
				}
			} else if s.PotentialRightsEnd != nil {
				t.Fatal("PotentialRightsEnd set without potential_rights.end")
			}
		} else if s.PotentialRightsEnd != nil {
			t.Fatal("PotentialRightsEnd set without potential_rights")
		}
		ev, _ := ci(m, "events")
		evs, _ := ev.([]any)
		if len(evs) != len(s.Events) {
			t.Fatalf("events: raw %d parsed %d", len(evs), len(s.Events))
		}
		for i, raw := range evs {
			em, _ := raw.(map[string]any)
			if ty, has := rawStr(em, "type"); has {
				switch ty {
				case "register", "renew", "return", "revoke", "cancel":
				default:
					t.Fatalf("events[%d].type %q accepted outside the enumeration", i, ty)
				}
				if s.Events[i].Type != ty {
					t.Fatalf("events[%d].type mismatch", i)
				}
			}
			if ts, has := rawStr(em, "timestamp"); has {
				tt, perr := time.Parse(time.RFC3339, ts)
				if perr != nil || !tt.Equal(s.Events[i].Timestamp) {
					t.Fatalf("events[%d].timestamp %q accepted but not valid / equal (%v)", i, ts, perr)
				}
			}
		}
		// Must be safe to evaluate against an empty local license.
		_ = lcp.NeedsLicenseRefresh(s, nil)
	})
}

// TestSeedStatusIsAccepted pins the seed oracle: validLSD must parse.
func TestSeedStatusIsAccepted(t *testing.T) {
	if _, err := lcp.ParseStatusDocument([]byte(validLSD)); err != nil {
		t.Fatalf("valid LSD seed rejected: %v", err)
	}
}
