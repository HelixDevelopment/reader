package lcpfuzz

import (
	"errors"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/lcp"
)

// TestHRD070CanonicalErrorMessages pins WHICH check rejected a malformed
// document, not only that some check did. Both the sentinel and the message
// are asserted, because several error branches in canonical.go return the same
// sentinel and differ only in what they report; dropping one of them is
// otherwise invisible. Oracle: the document is malformed by construction and
// the expected fragment is the standard library JSON decoder's own wording for
// exactly that defect (written down by hand from a probe run of the decoder,
// never produced by the function under test).
func TestHRD070CanonicalErrorMessages(t *testing.T) {
	deep := strings.Repeat("[", 10001) + strings.Repeat("]", 10001)
	cases := []struct {
		name    string
		fn      func([]byte) ([]byte, error)
		in      string
		is      error
		message string
	}{
		{"signing: empty input reaches the first-token check", lcp.CanonicalForSigning, "", lcp.ErrLicenseSyntax, ": EOF"},
		{"signing: blank input", lcp.CanonicalForSigning, " ", lcp.ErrLicenseSyntax, ": EOF"},
		{"signing: array top level", lcp.CanonicalForSigning, "[1]", lcp.ErrLicenseSyntax, "top level is not an object"},
		{"signing: member name token error", lcp.CanonicalForSigning, `{"a":1,}`, lcp.ErrLicenseSyntax, "invalid character ',' looking for beginning of value"},
		{"signing: member value decode error", lcp.CanonicalForSigning, `{"a":}`, lcp.ErrLicenseSyntax, "invalid character '}' looking for beginning of value"},
		{"signing: object never closed", lcp.CanonicalForSigning, `{"a":1`, lcp.ErrLicenseSyntax, "unexpected end of JSON input"},
		// A value that parses, then a closing bracket of the wrong kind: the
		// closing-delimiter token is the only check that sees it.
		{"signing: wrong closing delimiter", lcp.CanonicalForSigning, `{"a":1]`, lcp.ErrLicenseSyntax, "invalid character ']' after object key:value pair"},
		{"signing: wrong closing delimiter after a signature", lcp.CanonicalForSigning, `{"signature":{},"a":1]`, lcp.ErrLicenseSyntax, "invalid character ']' after object key:value pair"},
		{"canonical: wrong closing delimiter", lcp.Canonical, `{"a":1]`, lcp.ErrLicenseSyntax, "invalid character ']' after object key:value pair"},
		{"canonical: wrong closing delimiter in an array", lcp.Canonical, `[1}`, lcp.ErrLicenseSyntax, "invalid character '}' after array element"},
		{"canonical: member name token error", lcp.Canonical, `{"a":1,}`, lcp.ErrLicenseSyntax, "invalid character ',' looking for beginning of value"},
		{"canonical: object never closed", lcp.Canonical, `{"a":1`, lcp.ErrLicenseSyntax, "unexpected end of JSON input"},
		{"canonical: missing value", lcp.Canonical, `{"a":}`, lcp.ErrLicenseSyntax, "missing value after object key"},
		{"canonical: trailing data", lcp.Canonical, `{"a":1}x`, lcp.ErrLicenseSyntax, "trailing data after the top-level value"},
		// encoding/json limits nesting to 10000. Which of the two readers (the token
		// scan or the value decode) reports it depends on the module's go directive
		// (GODEBUG defaults), so this pins the outcome, not the reporter.
		{"canonical: nesting beyond the decoder limit", lcp.Canonical, deep, lcp.ErrLicenseSyntax, "exceeded max depth"},
		{"canonical: unparseable exponent", lcp.Canonical, "1e99999999999999999999", lcp.ErrLicenseSyntax, "bad exponent in 1e99999999999999999999"},
		{"canonical: exponent beyond the supported range", lcp.Canonical, "1e1073741825", lcp.ErrLicenseSyntax, "exponent out of range"},
		{"canonical: duplicate member", lcp.Canonical, `{"a":1,"a":2}`, lcp.ErrAmbiguousLicense, `"a"`},
	}
	for _, c := range cases {
		out, err := c.fn([]byte(c.in))
		if err == nil || out != nil {
			t.Errorf("%s: accepted (out %q, err %v)", c.name, out, err)
			continue
		}
		if !errors.Is(err, c.is) {
			t.Errorf("%s: error class %v, want %v", c.name, err, c.is)
		}
		if !strings.Contains(err.Error(), c.message) {
			t.Errorf("%s: message %q does not contain %q", c.name, err.Error(), c.message)
		}
	}
}

// TestHRD070LSDLinkRelMessages pins the two distinct reasons a link relation
// is refused: absent (required) versus present but not a string or an array of
// strings. Both are ErrStatusDocument; only the message tells them apart, and
// a mutation that merges them used to survive. Oracle: schema link.schema.json
// (rel required; string or array of strings), cases written by hand.
func TestHRD070LSDLinkRelMessages(t *testing.T) {
	mk := func(link string) []byte {
		return []byte(`{"id":"u","status":"active","message":"m","updated":{"license":"2020-01-01T00:00:00Z","status":"2020-01-02T00:00:00Z"},"links":[` + link + `,{"rel":"license","href":"https://p.example/l"}]}`)
	}
	cases := []struct{ name, link, message string }{
		{"rel absent", `{"href":"https://p.example/x"}`, "links[0].rel: required"},
		{"rel wrong type", `{"rel":7,"href":"https://p.example/x"}`, "links[0].rel: must be a string or an array of strings"},
		{"rel empty string", `{"rel":"","href":"https://p.example/x"}`, "links[0].rel: empty"},
	}
	for _, c := range cases {
		s, err := lcp.ParseStatusDocument(mk(c.link))
		if s != nil || !errors.Is(err, lcp.ErrStatusDocument) || !strings.Contains(err.Error(), c.message) {
			t.Errorf("%s: got %v %v, want ErrStatusDocument containing %q", c.name, s, err, c.message)
		}
	}
}

// TestHRD070TimestampFractionalSeconds records the standard-library fact the
// timestamp equivalence argument in equiv.tsv relies on: time.Parse with the
// RFC3339 layout already accepts fractional seconds, so a parser switched to the
// Nano layout would be behaviourally identical. Oracle: RFC 3339 §5.6 date-time
// grammar (time-secfrac is optional), values written by hand.
func TestHRD070TimestampFractionalSeconds(t *testing.T) {
	doc := func(issued string) []byte {
		return []byte(`{"id":"u","provider":"p","issued":"` + issued + `","encryption":{"profile":"http://readium.org/lcp/basic-profile","content_key":{"encrypted_value":"AA==","algorithm":"http://www.w3.org/2001/04/xmlenc#aes256-cbc"},"user_key":{"text_hint":"h","algorithm":"http://www.w3.org/2001/04/xmlenc#sha256","key_check":"AA=="}},"links":[{"rel":"hint","href":"https://p.example/h"}],"signature":{"algorithm":"a","certificate":"c","value":"v"}}`)
	}
	for _, ts := range []string{"2020-01-01T00:00:00Z", "2020-01-01T00:00:00.5Z", "2020-01-01T00:00:00.123456789+02:00"} {
		lic, err := lcp.Parse(doc(ts))
		if err != nil || lic.Issued.IsZero() {
			t.Errorf("issued %q: %v", ts, err)
		}
	}
	for _, ts := range []string{"2020-01-01", "2020-01-01T00:00:00", "2020-01-01T00:00:00.Z", "yesterday"} {
		if _, err := lcp.Parse(doc(ts)); !errors.Is(err, lcp.ErrLicenseSyntax) {
			t.Errorf("issued %q: %v, want ErrLicenseSyntax", ts, err)
		}
	}
}
