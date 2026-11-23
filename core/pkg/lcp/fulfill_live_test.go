package lcp

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

const licenseMediaType = "application/vnd.readium.lcp.license.v1.0+json"

// TestFulfillOverLiveHTTP runs the consumer flow across a real TCP listener: a
// provider serves a License Document and, at the link it names, the encrypted
// publication; a client downloads both, fulfills, unlocks and reads. The
// provider here is this repository's own Issue/Protect, so this proves Helix
// agrees with itself over HTTP, not that a third-party LCP provider or reader
// interoperates (UNCONFIRMED: needs one). When HELIX_LCP_EVIDENCE names a file
// the observed facts are written there as JSON.
func TestFulfillOverLiveHTTP(t *testing.T) {
	f := newFulfillFixture(t, func(l map[string]any) {})
	var hits []string
	mux := http.NewServeMux()
	mux.HandleFunc("/license.lcpl", func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		w.Header().Set("Content-Type", licenseMediaType)
		w.Write(f.license)
	})
	mux.HandleFunc("/pub.epub", func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		w.Header().Set("Content-Type", "application/epub+zip")
		w.Write(f.pub)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	get := func(path string) ([]byte, string) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("GET %s: %d %v", path, resp.StatusCode, err)
		}
		return b, resp.Header.Get("Content-Type")
	}
	licDoc, licType := get("/license.lcpl")
	if licType != licenseMediaType {
		t.Fatalf("license media type %q", licType)
	}
	lic, err := Parse(licDoc)
	if err != nil {
		t.Fatal(err)
	}
	link, _ := lic.PublicationLink()
	// The advertised href points at the provider's production host; the live
	// test serves it from the local listener, which is the only substitution.
	pub, _ := get("/pub.epub")
	out, err := Fulfill(licDoc, bytes.NewReader(pub), int64(len(pub)))
	if err != nil {
		t.Fatalf("Fulfill: %v", err)
	}
	c := open(t, out)
	rep, err := c.Unlock(fixturePassphrase, f.chain.policy())
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	read := 0
	for name, w := range f.want {
		b, err := c.Resource(name, parseutil.MaxTextBytes)
		if err != nil || !bytes.Equal(b, w) {
			t.Fatalf("Resource(%s): err=%v equal=%v", name, err, bytes.Equal(b, w))
		}
		read++
	}
	// Negative control over the same wire: a hostile mirror serving one flipped
	// byte at the publication link must be refused.
	bad := append([]byte{}, pub...)
	bad[len(bad)/2] ^= 1
	_, refuseErr := Fulfill(licDoc, bytes.NewReader(bad), int64(len(bad)))
	if refuseErr == nil {
		t.Fatal("tampered download accepted")
	}
	if p := os.Getenv("HELIX_LCP_EVIDENCE"); p != "" {
		sum := sha256.Sum256(out)
		ev := map[string]any{
			"test": "TestFulfillOverLiveHTTP", "transport": "net/http over loopback TCP (httptest.NewServer)",
			"provider": "in-repo Issue+Protect (NOT a third-party LCP provider)", "license_media_type": licType,
			"license_id": lic.ID, "publication_link_length": link.Length, "publication_link_sha256": link.Hash,
			"downloaded_bytes": len(pub), "fulfilled_bytes": len(out), "fulfilled_sha256": hex.EncodeToString(sum[:]),
			"signature_mode_verified": int(rep.Mode), "signature_mode_is_reference": rep.Mode == CanonicalReference, "resources_decrypted_and_byte_equal": read,
			"http_requests_served": hits, "tampered_download_refused": refuseErr.Error(),
			"unconfirmed": []string{"interop with a real LCP provider/License Authority", "interop with a third-party reader app", "license status document (LSD) handling", "Readium root certificate trust (Helix embeds none)"},
		}
		b, _ := json.MarshalIndent(ev, "", "  ")
		if err := os.WriteFile(p, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
