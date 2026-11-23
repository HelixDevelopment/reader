// The LCP Status Document (LSD) parser lives in core/pkg/lcp/lsd.go, which is
// owned by another work stream and was absent from the tree when this was
// written. Build with -tags lsd once lsd.go is present:
//   go test -tags lsd ./pkg/lcpfuzz -run '^$' -fuzz FuzzParseStatusDocument

package lcpfuzz

import (
	"testing"

	"github.com/helixreader/helix/core/pkg/lcp"
)

// FuzzParseStatusDocument: LSD parser must never panic and a nil error must
// imply a non-nil document.
func FuzzParseStatusDocument(f *testing.F) {
	seed(f)
	f.Fuzz(func(t *testing.T, doc []byte) {
		if len(doc) > 1<<20 {
			t.Skip("bound input")
		}
		s, err := lcp.ParseStatusDocument(doc)
		if err != nil {
			return
		}
		if s == nil {
			t.Fatal("nil status document with nil error")
		}
		// Must be safe to evaluate against an empty local license.
		_ = lcp.NeedsLicenseRefresh(s, nil)
	})
}
