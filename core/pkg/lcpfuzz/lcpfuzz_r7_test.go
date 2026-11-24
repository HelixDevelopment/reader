package lcpfuzz

import (
	"errors"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/lcp"
)

// TestHRD070ParseCaseVariantEncryptionMember kills mutant X18 (the text_hint
// presence decode error ignored). encoding/json matches struct members
// case-insensitively, so a top-level "Encryption" member satisfies every typed
// check in Parse, while the raw member lookup generic["encryption"] is
// case-sensitive and yields nil; decoding nil fails, and Parse reports that as
// ErrLicenseSyntax. Without the guard the document would reach the text_hint
// nil check and report ErrIncomplete instead (a different error class).
// Oracle: the expected class is read off the source's own comments (the
// decode failure is a syntax failure) and was observed by hand on the
// unmutated and mutated code; it is not produced by this test. This pins
// CURRENT behaviour. That Parse accepts wrong-case member names at all is a
// production defect tracked as a follow-up (N1), not fixed here.
func TestHRD070ParseCaseVariantEncryptionMember(t *testing.T) {
	doc := `{"id":"u","provider":"p","issued":"2020-01-01T00:00:00Z","Encryption":{"profile":"http://readium.org/lcp/basic-profile","content_key":{"encrypted_value":"AA==","algorithm":"http://www.w3.org/2001/04/xmlenc#aes256-cbc"},"user_key":{"text_hint":"h","algorithm":"http://www.w3.org/2001/04/xmlenc#sha256","key_check":"AA=="}},"links":[{"rel":"hint","href":"https://p.example/h"}],"signature":{"algorithm":"a","certificate":"c","value":"v"}}`
	_, err := lcp.Parse([]byte(doc))
	if !errors.Is(err, lcp.ErrLicenseSyntax) {
		t.Fatalf("case-variant Encryption member: want ErrLicenseSyntax, got %v", err)
	}
	if errors.Is(err, lcp.ErrIncomplete) {
		t.Fatalf("must not be classed incomplete: %v", err)
	}
	if !strings.Contains(err.Error(), "encryption:") {
		t.Fatalf("message should name the encryption decode: %v", err)
	}
}
