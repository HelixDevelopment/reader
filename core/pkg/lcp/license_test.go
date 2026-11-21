package lcp

import (
	"errors"
	"strings"
	"testing"
)

// removeMember returns a mutator that deletes the member at a "/"-separated
// path, walking objects at any depth.
func removeMember(path string) func(map[string]any) {
	return func(m map[string]any) {
		parts := strings.Split(path, "/")
		cur := m
		for _, p := range parts[:len(parts)-1] {
			next, ok := cur[p].(map[string]any)
			if !ok {
				panic("no such nested member: " + path)
			}
			cur = next
		}
		delete(cur, parts[len(parts)-1])
	}
}

// setMember returns a mutator assigning a member at a "/"-separated path.
func setMember(path string, value any) func(map[string]any) {
	return func(m map[string]any) {
		parts := strings.Split(path, "/")
		cur := m
		for _, p := range parts[:len(parts)-1] {
			next, ok := cur[p].(map[string]any)
			if !ok {
				panic("no such nested member: " + path)
			}
			cur = next
		}
		cur[parts[len(parts)-1]] = value
	}
}

// TestParseRejectsIncompleteLicenses covers §7.2 point 1. The named member is
// asserted as well as the sentinel, because a license refused for the wrong
// reason would otherwise look like a passing validation.
func TestParseRejectsIncompleteLicenses(t *testing.T) {
	for _, tc := range []struct {
		path string
		want string
	}{
		{"id", "id"},
		{"provider", "provider"},
		{"issued", "issued"},
		{"links", "links"},
		{"signature", "signature"},
		{"encryption/profile", "encryption/profile"},
		{"encryption/content_key/algorithm", "encryption/content_key"},
		{"encryption/content_key/encrypted_value", "encryption/content_key"},
		{"encryption/user_key/algorithm", "encryption/user_key"},
		{"encryption/user_key/key_check", "encryption/user_key"},
		{"encryption/user_key/text_hint", "encryption/user_key/text_hint"},
		{"signature/algorithm", "signature"},
		{"signature/certificate", "signature"},
		{"signature/value", "signature"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			doc, _ := buildLicenseTB(t, "pasSED1!", removeMember(tc.path))
			_, err := Parse(doc)
			if !errors.Is(err, ErrIncomplete) {
				t.Fatalf("Parse without %s error = %v, want ErrIncomplete", tc.path, err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to name %q", err, tc.want)
			}
		})
	}
}

// TestParseAcceptsAnEmptyTextHint: §3.4 requires the text_hint member and puts
// no constraint on its value, so "" is a complete license and must parse. This
// is the other half of the text_hint rule -- refusing absence is only correct
// if emptiness is still accepted, otherwise the check is just a non-empty test
// with a different name.
func TestParseAcceptsAnEmptyTextHint(t *testing.T) {
	doc, _ := buildLicenseTB(t, "pasSED1!", setMember("encryption/user_key/text_hint", ""))
	lic := mustParseTB(t, doc)
	if lic.Encryption.UserKey.TextHint != "" {
		t.Errorf("TextHint = %q, want the empty string preserved", lic.Encryption.UserKey.TextHint)
	}
	if err := lic.CheckPassphrase("pasSED1!"); err != nil {
		t.Errorf("CheckPassphrase on a license with an empty hint: %v", err)
	}
}

// TestParseRejectsEmptyLinkHref: links/href is the one required member of a
// Link Object, and an empty string is how a producer omits it.
func TestParseRejectsEmptyLinkHref(t *testing.T) {
	doc, _ := buildLicenseTB(t, "pasSED1!", func(m map[string]any) {
		m["links"].([]any)[1].(map[string]any)["href"] = ""
	})
	_, err := Parse(doc)
	if !errors.Is(err, ErrIncomplete) {
		t.Fatalf("error = %v, want ErrIncomplete", err)
	}
	if want := "links[1]/href"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %v, want it to name %q", err, want)
	}
}

// TestParseRejectsMalformedDates pins §3.3's RFC 3339 requirement. A producer
// emitting a bare date or a space separator must be refused rather than read as
// a zero time, which would silently disable the rights window.
func TestParseRejectsMalformedDates(t *testing.T) {
	for _, bad := range []string{"2026-11-21", "21/11/2026", "2026-11-21 00:00:00", "not-a-date"} {
		for _, field := range []string{"issued", "updated"} {
			doc, _ := buildLicenseTB(t, "pasSED1!", setMember(field, bad))
			if _, err := Parse(doc); !errors.Is(err, ErrLicenseSyntax) {
				t.Errorf("Parse with %s=%q error = %v, want ErrLicenseSyntax", field, bad, err)
			}
		}
	}
}

// TestParseDefaultsUpdatedToIssued documents the field the signature validity
// window in §5.5.1 point 4 depends on.
func TestParseDefaultsUpdatedToIssued(t *testing.T) {
	doc, _ := buildLicenseTB(t, "pasSED1!", removeMember("updated"))
	lic := mustParseTB(t, doc)
	if !lic.Updated.Equal(lic.Issued) {
		t.Errorf("Updated = %s, want it to default to Issued %s", lic.Updated, lic.Issued)
	}
}

// TestParseRightsAbsentIsNotZero: an absent right means unrestricted, a zero
// right means forbidden. Collapsing the two would either strip every print
// permission from licenses that never declared one, or grant unlimited printing.
func TestParseRightsAbsentIsNotZero(t *testing.T) {
	absent, _ := buildLicenseTB(t, "pasSED1!", removeMember("rights"))
	lic := mustParseTB(t, absent)
	if lic.Rights.Print != nil || lic.Rights.Copy != nil {
		t.Errorf("absent rights read as %+v, want both nil", lic.Rights)
	}
	zero, _ := buildLicenseTB(t, "pasSED1!", setMember("rights", map[string]any{"print": 0, "copy": 0}))
	got := mustParseTB(t, zero)
	if got.Rights.Print == nil || *got.Rights.Print != 0 {
		t.Errorf("print:0 read as %v, want a present zero", got.Rights.Print)
	}
	if got.Rights.Copy == nil || *got.Rights.Copy != 0 {
		t.Errorf("copy:0 read as %v, want a present zero", got.Rights.Copy)
	}
	// A producer that serializes an empty rights object as null is saying the
	// same thing as omitting it; refusing the document would be stricter than
	// §3.6, which makes rights optional and constrains nothing about its value.
	null, _ := buildLicenseTB(t, "pasSED1!", setMember("rights", nil))
	if nlic := mustParseTB(t, null); nlic.Rights.Print != nil || nlic.Rights.Copy != nil {
		t.Errorf("rights:null read as %+v, want both nil", nlic.Rights)
	}
}

func TestParseRightsRejectsWrongType(t *testing.T) {
	doc, _ := buildLicenseTB(t, "pasSED1!", setMember("rights", "unlimited"))
	if _, err := Parse(doc); !errors.Is(err, ErrLicenseSyntax) {
		t.Fatalf("error = %v, want ErrLicenseSyntax", err)
	}
}

// TestParseAcceptsAndReportsUnknownMembers: §7.2 requires unknown name/value
// pairs be ignored, and the report of what was ignored must be stable so two
// runs on one document agree.
func TestParseAcceptsAndReportsUnknownMembers(t *testing.T) {
	doc, _ := buildLicenseTB(t, "pasSED1!", func(m map[string]any) {
		m["zeta_future"] = map[string]any{"a": 1}
		m["alpha_future"] = "keep me"
		m["encryption"].(map[string]any)["unknown_submember"] = true
	})
	lic := mustParseTB(t, doc)
	want := []string{"alpha_future", "zeta_future"}
	// Determinism and correctness are separate claims: the report is built by
	// ranging a map, so a stably wrong order is a different bug from an
	// order that varies between runs of the same document.
	seen := map[string]int{}
	for i := 0; i < 50; i++ {
		seen[strings.Join(mustParseTB(t, doc).UnknownMembers, ",")]++
	}
	if len(seen) != 1 {
		t.Fatalf("UnknownMembers order is not deterministic: %d distinct reports over 50 parses of one document", len(seen))
	}
	if strings.Join(lic.UnknownMembers, ",") != strings.Join(want, ",") {
		t.Fatalf("UnknownMembers = %v, want %v in sorted order", lic.UnknownMembers, want)
	}
	// Ignoring must not mean dropping: the license still decrypts.
	if err := lic.CheckPassphrase("pasSED1!"); err != nil {
		t.Errorf("CheckPassphrase on a license with unknown members: %v", err)
	}
}

// TestLinkLookupAcceptsRelAndRels: §3.5 allows a single rel, a rels array, or
// both, and real licenses in the wild do all three.
func TestLinkLookupAcceptsRelAndRels(t *testing.T) {
	doc, _ := buildLicenseTB(t, "pasSED1!", func(m map[string]any) {
		m["links"] = []any{
			map[string]any{"href": "https://x.example/status", "rel": "status"},
			map[string]any{"href": "https://x.example/hint", "rels": []string{"hint", "alternate"}},
			map[string]any{"href": "https://x.example/pub.epub", "rel": "self",
				"rels": []string{"publication"}},
		}
	})
	lic := mustParseTB(t, doc)
	for _, tc := range []struct{ rel, want string }{
		{"status", "https://x.example/status"},
		{"hint", "https://x.example/hint"},
		{"alternate", "https://x.example/hint"},
		{"publication", "https://x.example/pub.epub"},
		{"self", "https://x.example/pub.epub"},
	} {
		got, ok := lic.Link(tc.rel)
		if !ok || got.Href != tc.want {
			t.Errorf("Link(%q) = %+v, %v; want %q", tc.rel, got, ok, tc.want)
		}
	}
	if _, ok := lic.Link("absent"); ok {
		t.Error("Link reported a relation the license does not carry")
	}
}

// TestCheckProfileMatrix: §3.4 and §6.3 bind four URIs together, and every one
// of them can arrive wrong from a producer using another profile.
func TestCheckProfileMatrix(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		value any
		want  error
		dial  string
	}{
		{"profile", "encryption/profile", "http://readium.org/lcp/profile-1.0", ErrUnsupportedProfile, "profile"},
		{"content key", "encryption/content_key/algorithm", "http://www.w3.org/2001/04/xmlenc#aes128-cbc", ErrUnsupportedAlgorithm, "content key"},
		{"user key", "encryption/user_key/algorithm", "http://www.w3.org/2001/04/xmlenc#sha512", ErrUnsupportedAlgorithm, "user key"},
		{"signature", "signature/algorithm", "http://www.w3.org/2001/04/xmldsig-more#rsa-sha1", ErrUnsupportedAlgorithm, "signature"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, _ := buildLicenseTB(t, "pasSED1!", setMember(tc.path, tc.value))
			lic := mustParseTB(t, doc)
			err := lic.CheckProfile()
			if !errors.Is(err, tc.want) {
				t.Fatalf("CheckProfile error = %v, want %v", err, tc.want)
			}
			if !strings.Contains(err.Error(), tc.dial) {
				t.Errorf("error = %v, want it to identify the %s", err, tc.dial)
			}
		})
	}
	ok, _ := buildLicenseTB(t, "pasSED1!", nil)
	if err := mustParseTB(t, ok).CheckProfile(); err != nil {
		t.Errorf("CheckProfile on a conforming license: %v", err)
	}
}

// TestCheckPassphraseRefusesUnsupportedProfile: the profile must be settled
// before any key material is handled, or a license naming another algorithm
// would be decrypted under Helix's assumptions about this one.
func TestCheckPassphraseRefusesUnsupportedProfile(t *testing.T) {
	doc, _ := buildLicenseTB(t, "pasSED1!", setMember("encryption/profile", "http://readium.org/lcp/profile-1.0"))
	lic := mustParseTB(t, doc)
	if err := lic.CheckPassphrase("pasSED1!"); !errors.Is(err, ErrUnsupportedProfile) {
		t.Fatalf("error = %v, want ErrUnsupportedProfile", err)
	}
	if _, err := lic.ContentKey(UserKey("pasSED1!")); !errors.Is(err, ErrUnsupportedProfile) {
		t.Fatalf("ContentKey error = %v, want ErrUnsupportedProfile", err)
	}
}

func TestParseRejectsNonObjectTopLevel(t *testing.T) {
	for _, in := range []string{`[]`, `"just a string"`, `42`} {
		if _, err := Parse([]byte(in)); !errors.Is(err, ErrIncomplete) && !errors.Is(err, ErrLicenseSyntax) {
			t.Errorf("Parse(%s) error = %v, want a syntax or completeness failure", in, err)
		}
	}
}
