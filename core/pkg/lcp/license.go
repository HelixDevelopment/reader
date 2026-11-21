package lcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

var (
	// ErrIncomplete reports a License Document missing a member required by
	// §3 or by the License Document JSON Schema.
	ErrIncomplete = errors.New("lcp: license document is incomplete")
	// ErrUnsupportedProfile reports a profile Helix cannot fulfill. The
	// registry at lcp-specs/registries/profiles.md lists
	// http://readium.org/lcp/profile-1.0 as the production profile defined by
	// EDRLab; Helix implements only the Basic Encryption Profile 1.0.
	ErrUnsupportedProfile = errors.New("lcp: unsupported encryption profile")
	// ErrUnsupportedAlgorithm reports an algorithm URI outside the profile.
	ErrUnsupportedAlgorithm = errors.New("lcp: unsupported algorithm")
)

// Link is the §3.5 Link Object. Only href is required; rel and rels are both
// accepted because License Documents in the wild carry one, the other, or both.
type Link struct {
	Href   string            `json:"href"`
	Type   string            `json:"type,omitempty"`
	Rel    string            `json:"rel,omitempty"`
	Rels   []string          `json:"rels,omitempty"`
	Titles map[string]string `json:"titles,omitempty"`
	Length int64             `json:"length,omitempty"`
	Hash   string            `json:"hash,omitempty"`
	// Properties carries profile- or relation-specific members, e.g. the
	// opacity/position hints of an LCP status document.
	Properties map[string]any `json:"properties,omitempty"`
}

// HasRel reports whether the link carries the given relation.
func (l Link) HasRel(rel string) bool {
	if l.Rel == rel {
		return true
	}
	for _, r := range l.Rels {
		if r == rel {
			return true
		}
	}
	return false
}

// Encryption is the §3.4 encryption object.
type Encryption struct {
	Profile    string         `json:"profile"`
	ContentKey ContentKeyInfo `json:"content_key"`
	UserKey    UserKeyInfo    `json:"user_key"`
}

// ContentKeyInfo holds the Content Key encrypted with the User Key.
type ContentKeyInfo struct {
	EncryptedValue string `json:"encrypted_value"`
	Algorithm      string `json:"algorithm"`
}

// UserKeyInfo describes how the User Key is derived and checked.
type UserKeyInfo struct {
	TextHint  string `json:"text_hint"`
	Algorithm string `json:"algorithm"`
	KeyCheck  string `json:"key_check"`
}

// Rights is the §3.6 rights object. Absent means unrestricted.
type Rights struct {
	Print *int       `json:"print,omitempty"`
	Copy  *int       `json:"copy,omitempty"`
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// User is the §3.7 user object. Encrypted lists members of this object whose
// values are AES-256-CBC encrypted with the User Key rather than plaintext.
type User struct {
	ID        string   `json:"id,omitempty"`
	Email     string   `json:"email,omitempty"`
	Name      string   `json:"name,omitempty"`
	Encrypted []string `json:"encrypted,omitempty"`
}

// Signature is the §3.8 signature object.
type Signature struct {
	Algorithm   string `json:"algorithm"`
	Certificate string `json:"certificate"`
	Value       string `json:"value"`
}

// License is a parsed License Document. Raw keeps the exact received bytes;
// every signature operation canonicalizes Raw rather than a re-serialization,
// so bytes a producer signed are the bytes a verifier hashes.
type License struct {
	Raw        []byte
	ID         string
	Issued     time.Time
	Updated    time.Time
	Provider   string
	Encryption Encryption
	Links      []Link
	Rights     Rights
	User       User
	Signature  Signature

	// UnknownMembers counts name/value pairs outside §3, kept only so
	// capability reporting can state that they were ignored, as §7.2 requires.
	UnknownMembers []string
}

var knownMembers = map[string]bool{
	"id": true, "issued": true, "updated": true, "provider": true,
	"encryption": true, "links": true, "rights": true, "user": true,
	"signature": true,
}

type wireLicense struct {
	ID         string          `json:"id"`
	Issued     string          `json:"issued"`
	Updated    string          `json:"updated"`
	Provider   string          `json:"provider"`
	Encryption Encryption      `json:"encryption"`
	Links      []Link          `json:"links"`
	Rights     json.RawMessage `json:"rights"`
	User       User            `json:"user"`
	Signature  Signature       `json:"signature"`
}

// Parse reads a License Document and validates its syntax and completeness
// (§7.2). It deliberately accepts unknown members, which §7.2 requires be
// ignored, and it accepts an unsupported profile so callers can report the
// reason; decryption then fails with ErrUnsupportedProfile.
func Parse(doc []byte) (*License, error) {
	var w wireLicense
	if err := json.Unmarshal(doc, &w); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
	}
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(doc, &generic); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
	}
	if w.ID == "" {
		return nil, fmt.Errorf("%w: id", ErrIncomplete)
	}
	if w.Provider == "" {
		return nil, fmt.Errorf("%w: provider", ErrIncomplete)
	}
	// Absent and malformed are different complaints: §3.3 requires issued, so a
	// document without it is incomplete, while a present value that will not
	// parse is a syntax failure.
	if w.Issued == "" {
		return nil, fmt.Errorf("%w: issued", ErrIncomplete)
	}
	issued, err := parseTime(w.Issued, "issued")
	if err != nil {
		return nil, err
	}
	updated := issued
	if w.Updated != "" {
		if updated, err = parseTime(w.Updated, "updated"); err != nil {
			return nil, err
		}
	}
	if w.Encryption.Profile == "" {
		return nil, fmt.Errorf("%w: encryption/profile", ErrIncomplete)
	}
	if w.Encryption.ContentKey.EncryptedValue == "" || w.Encryption.ContentKey.Algorithm == "" {
		return nil, fmt.Errorf("%w: encryption/content_key", ErrIncomplete)
	}
	// §3.4 requires three user_key members. algorithm and key_check carry
	// constrained values, so an empty string is a failure in itself; text_hint
	// is unconstrained, and an empty string is legitimate while the member is
	// not, so its presence is read off the raw document, where a decoded struct
	// cannot tell an absent member from an empty one.
	if w.Encryption.UserKey.Algorithm == "" || w.Encryption.UserKey.KeyCheck == "" {
		return nil, fmt.Errorf("%w: encryption/user_key", ErrIncomplete)
	}
	var hint struct {
		UserKey struct {
			TextHint *string `json:"text_hint"`
		} `json:"user_key"`
	}
	if err := json.Unmarshal(generic["encryption"], &hint); err != nil {
		return nil, fmt.Errorf("%w: encryption: %v", ErrLicenseSyntax, err)
	}
	if hint.UserKey.TextHint == nil {
		return nil, fmt.Errorf("%w: encryption/user_key/text_hint", ErrIncomplete)
	}
	if len(w.Links) == 0 {
		return nil, fmt.Errorf("%w: links", ErrIncomplete)
	}
	for i, l := range w.Links {
		if l.Href == "" {
			return nil, fmt.Errorf("%w: links[%d]/href", ErrIncomplete, i)
		}
	}
	if w.Signature.Algorithm == "" || w.Signature.Certificate == "" || w.Signature.Value == "" {
		return nil, fmt.Errorf("%w: signature", ErrIncomplete)
	}

	lic := &License{
		Raw: doc, ID: w.ID, Issued: issued, Updated: updated, Provider: w.Provider,
		Encryption: w.Encryption, Links: w.Links, User: w.User, Signature: w.Signature,
	}
	// Absent rights decode to the zero struct, which is what §3.6 means by an
	// optional object: every right unset, therefore unrestricted.
	if len(w.Rights) > 0 {
		if err := json.Unmarshal(w.Rights, &lic.Rights); err != nil {
			return nil, fmt.Errorf("%w: rights: %v", ErrLicenseSyntax, err)
		}
	}
	for k := range generic {
		if !knownMembers[k] {
			lic.UnknownMembers = append(lic.UnknownMembers, k)
		}
	}
	// Sourced from a map range: without this the report order of the same
	// document varies between runs.
	sort.Strings(lic.UnknownMembers)
	return lic, nil
}

func parseTime(s, field string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s is not an RFC 3339 date-time: %v", ErrLicenseSyntax, field, err)
	}
	return t, nil
}

// Link returns the first link carrying the given relation.
func (l *License) Link(rel string) (Link, bool) {
	for _, lk := range l.Links {
		if lk.HasRel(rel) {
			return lk, true
		}
	}
	return Link{}, false
}

// IsBasicProfile reports whether the license names the Basic Encryption
// Profile 1.0, the only profile Helix fulfills.
func (l *License) IsBasicProfile() bool {
	return l.Encryption.Profile == ProfileBasic
}

// CheckProfile verifies that the profile and the three key algorithm URIs
// agree, as §3.4 and §6.3 require.
func (l *License) CheckProfile() error {
	if !l.IsBasicProfile() {
		return fmt.Errorf("%w: %s", ErrUnsupportedProfile, l.Encryption.Profile)
	}
	if l.Encryption.ContentKey.Algorithm != AlgAES256CBC {
		return fmt.Errorf("%w: content key %s", ErrUnsupportedAlgorithm, l.Encryption.ContentKey.Algorithm)
	}
	if l.Encryption.UserKey.Algorithm != AlgSHA256 {
		return fmt.Errorf("%w: user key %s", ErrUnsupportedAlgorithm, l.Encryption.UserKey.Algorithm)
	}
	if l.Signature.Algorithm != AlgRSASHA256 {
		return fmt.Errorf("%w: signature %s", ErrUnsupportedAlgorithm, l.Signature.Algorithm)
	}
	return nil
}
