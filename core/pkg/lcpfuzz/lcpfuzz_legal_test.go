package lcpfuzz

import (
	"encoding/json"
	"testing"

	"github.com/helixreader/helix/core/pkg/lcp"
)

// TestHRD070LegalDocumentsAreAccepted is the over-rejection side of the fuzz
// oracles (which only judge ACCEPTED documents): every document below is
// legal under the contract and must parse. A parser that starts rejecting a
// legal member value (a status, an event type, an optional member) fails here.
func TestHRD070LegalDocumentsAreAccepted(t *testing.T) {
	lic := map[string]func(m map[string]any){
		"base":            func(m map[string]any) {},
		"empty text_hint": func(m map[string]any) { sub(sub(m, "encryption"), "user_key")["text_hint"] = "" },
		"updated absent":  func(m map[string]any) { delete(m, "updated") },
		"updated empty":   func(m map[string]any) { m["updated"] = "" },
		"rights valid": func(m map[string]any) {
			m["rights"] = map[string]any{"print": 10, "copy": 2, "start": "2020-01-01T00:00:00Z", "end": "2021-01-01T00:00:00Z"}
		},
		"rights null":     func(m map[string]any) { m["rights"] = nil },
		"rights empty":    func(m map[string]any) { m["rights"] = map[string]any{} },
		"unknown members": func(m map[string]any) { m["x-a"] = 1; m["x-b"] = "two" },
		"two links": func(m map[string]any) {
			m["links"] = []any{map[string]any{"rel": "hint", "href": "h1"}, map[string]any{"rel": "publication", "href": "h2"}}
		},
	}
	for name, edit := range lic {
		b := licenseVariant(edit)
		if b == nil {
			t.Fatalf("%s: seed generation failed", name)
		}
		if _, err := lcp.Parse(b); err != nil {
			t.Errorf("license %q rejected: %v", name, err)
		}
	}
	link := func(rel any) []any {
		return []any{map[string]any{"rel": "license", "href": "h"}, map[string]any{"rel": rel, "href": "h2"}}
	}
	lsd := map[string]func(m map[string]any){
		"base":                func(m map[string]any) {},
		"empty message":       func(m map[string]any) { m["message"] = "" },
		"status ready":        func(m map[string]any) { m["status"] = "ready" },
		"status revoked":      func(m map[string]any) { m["status"] = "revoked" },
		"status returned":     func(m map[string]any) { m["status"] = "returned" },
		"status cancelled":    func(m map[string]any) { m["status"] = "cancelled" },
		"status expired":      func(m map[string]any) { m["status"] = "expired" },
		"rel array":           func(m map[string]any) { m["links"] = link([]any{"license", "x"}) },
		"rel empty array":     func(m map[string]any) { m["links"] = link([]any{}) },
		"potential rights":    func(m map[string]any) { m["potential_rights"] = map[string]any{"end": "2030-01-01T00:00:00Z"} },
		"potential rights {}": func(m map[string]any) { m["potential_rights"] = map[string]any{} },
		"event renew":         func(m map[string]any) { m["events"] = []any{map[string]any{"type": "renew"}} },
		"event return":        func(m map[string]any) { m["events"] = []any{map[string]any{"type": "return"}} },
		"event revoke":        func(m map[string]any) { m["events"] = []any{map[string]any{"type": "revoke"}} },
		"event cancel":        func(m map[string]any) { m["events"] = []any{map[string]any{"type": "cancel"}} },
		"event register+time": func(m map[string]any) {
			m["events"] = []any{map[string]any{"type": "register", "timestamp": "2020-01-01T00:00:00Z"}, map[string]any{"name": "no type, no time"}}
		},
	}
	for name, edit := range lsd {
		m := baseLSD()
		edit(m)
		b, _ := json.Marshal(m)
		if _, err := lcp.ParseStatusDocument(b); err != nil {
			t.Errorf("status document %q rejected: %v", name, err)
		}
	}
}
