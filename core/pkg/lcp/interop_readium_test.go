package lcp

// Interop against a THIRD-PARTY provider: the Readium reference License
// Server + Status Server (github.com/readium/readium-lcp-server), run as real
// processes by qa/lcp-interop/run_interop.sh inside a rootless, network-less
// container (HRD-046). Nothing here starts a provider; without
// HELIX_LCP_INTEROP_LCP_URL the test skips (an honest SKIP, never a pass).
//
// Inputs are environment variables (credential VALUES are read from the
// environment and never written to the evidence):
//
//	HELIX_LCP_INTEROP_LCP_URL     base URL of the License Server
//	HELIX_LCP_INTEROP_ADMIN_USER  basic-auth user of the License Server
//	HELIX_LCP_INTEROP_ADMIN_PASS  basic-auth password of the License Server
//	HELIX_LCP_INTEROP_CONTENT_ID  content id the publication was registered under
//	HELIX_LCP_INTEROP_PASSPHRASE  the end user's passphrase (hint: the license's text_hint)
//	HELIX_LCP_INTEROP_ROOT_PEM    path of the root certificate that signed the provider certificate
//	HELIX_LCP_INTEROP_FOREIGN_PEM path of a certificate that did NOT sign the provider certificate (must be refused as a root)
//	HELIX_LCP_INTEROP_SOURCE_EPUB path of the unencrypted source EPUB (the plaintext oracle)
//	HELIX_LCP_INTEROP_EVIDENCE    path of the JSON evidence file to write

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type interopStep struct {
	Name   string `json:"name"`
	Result string `json:"result"` // PASS | FAIL (assertions) | INFO (recorded, not asserted)
	Detail string `json:"detail,omitempty"`
}

func TestInteropReadiumLCPServer(t *testing.T) {
	base := os.Getenv("HELIX_LCP_INTEROP_LCP_URL")
	if base == "" {
		t.Skip("HELIX_LCP_INTEROP_LCP_URL not set: no third-party provider running (UNCONFIRMED)")
	}
	user, pass := os.Getenv("HELIX_LCP_INTEROP_ADMIN_USER"), os.Getenv("HELIX_LCP_INTEROP_ADMIN_PASS")
	cid, phrase := os.Getenv("HELIX_LCP_INTEROP_CONTENT_ID"), os.Getenv("HELIX_LCP_INTEROP_PASSPHRASE")
	if user == "" || pass == "" || cid == "" || phrase == "" || os.Getenv("HELIX_LCP_INTEROP_FOREIGN_PEM") == "" {
		t.Fatal("interop environment incomplete")
	}
	var steps []interopStep
	rec := func(name string, err error, detail string) {
		s := interopStep{Name: name, Result: "PASS", Detail: detail}
		if err != nil {
			// keep the diagnostic detail (timestamps, sizes) next to the error
			s.Result = "FAIL"
			s.Detail = strings.TrimSpace(detail + " | error: " + err.Error())
			t.Errorf("%s: %v", name, err)
		}
		steps = append(steps, s)
	}
	info := func(name, detail string) {
		steps = append(steps, interopStep{Name: name, Result: "INFO", Detail: detail})
	}
	// A step that never ran is absent, and absent is not PASS: every expected assertion must have a record.
	expected := []string{
		"provider_issues_license", "helix_parses_provider_license", "provider_license_is_basic_profile",
		"helix_fulfills_provider_publication_default_strict", "helix_opens_fulfilled_container",
		"helix_unlocks_with_provider_signature_chain_and_passphrase",
		"helix_decrypts_every_protected_resource_byte_identical_to_source",
		"wrong_passphrase_refused", "no_trust_root_refused", "foreign_root_refused", "tampered_license_signature_refused",
		"helix_fetches_provider_status_document", "status_ready_and_license_usable",
		"helix_registers_device_status_becomes_active", "helix_returns_license_status_becomes_returned",
		"needs_license_refresh_after_return", "returned_status_offers_license_link",
		"raw_status_document_after_return", "license_document_sha256",
	}
	defer func() {
		// runs before the evidence is written (same deferred func), so the FAIL records land in the evidence
		have := map[string]bool{}
		for _, s := range steps {
			have[s.Name] = true
		}
		for _, n := range expected {
			if !have[n] {
				steps = append(steps, interopStep{Name: n, Result: "FAIL", Detail: "expected step never recorded"})
				t.Errorf("%s: expected step never recorded", n)
			}
		}
		if p := os.Getenv("HELIX_LCP_INTEROP_EVIDENCE"); p != "" {
			b, _ := json.MarshalIndent(map[string]any{"provider": "readium-lcp-server", "steps": steps}, "", "  ")
			_ = os.WriteFile(p, b, 0o600)
		}
	}()

	httpc := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()

	// 1. Ask the provider for a license.
	reqBody, _ := json.Marshal(map[string]any{
		"provider": "https://helix.example/interop",
		"user":     map[string]any{"id": "interop-user-1", "email": "interop@example.invalid", "name": "Interop User"},
		"encryption": map[string]any{"user_key": map[string]any{
			"text_hint": "interop passphrase", "algorithm": "http://www.w3.org/2001/04/xmlenc#sha256",
			"hex_value": UserKeyHex(phrase)}},
		// the reference Status Server offers a return link only for a license with an end date (a loan)
		"rights": map[string]any{"print": 10, "copy": 2000, "end": time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)},
	})
	req, _ := http.NewRequest(http.MethodPost, base+"/contents/"+cid+"/license", bytes.NewReader(reqBody))
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		t.Fatalf("license request: %v", iopStripURL(err))
	}
	licBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("license request: HTTP %d", resp.StatusCode)
	}
	rec("provider_issues_license", nil, "HTTP 201 "+resp.Header.Get("Content-Type"))

	// 2. Parse.
	lic, err := Parse(licBytes)
	rec("helix_parses_provider_license", err, "")
	if err != nil {
		return
	}
	rec("provider_license_is_basic_profile", lic.CheckProfile(), lic.Encryption.Profile)

	// 3. Download and fulfill.
	pl, ok := lic.PublicationLink()
	if !ok {
		rec("publication_link_present", errors.New("no publication link"), "")
		return
	}
	pubReq, _ := http.NewRequest(http.MethodGet, pl.Href, nil)
	pubResp, err := httpc.Do(pubReq)
	if err != nil {
		t.Fatalf("publication download: %v", iopStripURL(err))
	}
	pub, _ := io.ReadAll(io.LimitReader(pubResp.Body, 64<<20))
	pubResp.Body.Close()
	detail := "len=" + iopItoa(len(pub)) + " linkLength=" + iopItoa(int(pl.Length)) + " linkHashSet=" + iopBoolStr(pl.Hash != "")
	fulfilled, err := Fulfill(licBytes, bytes.NewReader(pub), int64(len(pub)))
	rec("helix_fulfills_provider_publication_default_strict", err, detail)
	if err != nil {
		// Provider-quirk fallback, recorded as its own step: the default is
		// fail-closed, so a provider omitting length/hash is a finding.
		fulfilled, err = Fulfill(licBytes, bytes.NewReader(pub), int64(len(pub)), AllowUnverified())
		rec("helix_fulfills_provider_publication_allow_unverified", err, detail)
		if err != nil {
			return
		}
	}

	// 4. Open + unlock under the operator's root.
	rootPEM, err := os.ReadFile(os.Getenv("HELIX_LCP_INTEROP_ROOT_PEM"))
	if err != nil {
		t.Fatal(err)
	}
	blk, _ := pem.Decode(rootPEM)
	if blk == nil {
		t.Fatal("root pem undecodable")
	}
	root, err := x509.ParseCertificate(blk.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	c, err := OpenContainer(bytes.NewReader(fulfilled), int64(len(fulfilled)))
	rec("helix_opens_fulfilled_container", err, "")
	if err != nil {
		return
	}
	rep, err := c.Unlock(phrase, CertificatePolicy{Roots: []*x509.Certificate{root}})
	rec("helix_unlocks_with_provider_signature_chain_and_passphrase", err, "canonical_mode="+iopItoa(int(rep.Mode))+" crl_checked="+iopBoolStr(rep.CRLChecked))
	if err != nil {
		return
	}

	// 5. Plaintext oracle: the original, never-encrypted EPUB.
	src, err := os.ReadFile(os.Getenv("HELIX_LCP_INTEROP_SOURCE_EPUB"))
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		t.Fatal(err)
	}
	orig := map[string][]byte{}
	for _, f := range zr.File {
		rc, e := f.Open()
		if e != nil {
			continue
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		orig[f.Name] = b
	}
	prot := c.ProtectedResources()
	if len(prot) == 0 {
		rec("provider_encrypted_some_resource", errors.New("no protected resources"), "")
		return
	}
	matched, mism := 0, 0
	var firstMismatch string
	for _, name := range prot {
		want, have := orig[name]
		got, e := c.Resource(name, 64<<20)
		if e != nil || !have || !bytes.Equal(got, want) {
			mism++
			if firstMismatch == "" {
				firstMismatch = name
				if e != nil {
					firstMismatch += ": " + e.Error()
				}
			}
			continue
		}
		matched++
	}
	var derr error
	if mism > 0 {
		derr = errors.New("decrypted != original for " + iopItoa(mism) + " of " + iopItoa(len(prot)) + " resources; first: " + firstMismatch)
	}
	rec("helix_decrypts_every_protected_resource_byte_identical_to_source", derr,
		"protected="+iopItoa(len(prot))+" identical="+iopItoa(matched))

	// 6. Negative paths against the real provider's output.
	c2, _ := OpenContainer(bytes.NewReader(fulfilled), int64(len(fulfilled)))
	_, err = c2.Unlock(phrase+"x", CertificatePolicy{Roots: []*x509.Certificate{root}})
	rec("wrong_passphrase_refused", iopMustIs(err, ErrWrongUserKey), iopErrStr(err))
	c3, _ := OpenContainer(bytes.NewReader(fulfilled), int64(len(fulfilled)))
	_, err = c3.Unlock(phrase, CertificatePolicy{})
	rec("no_trust_root_refused", iopMustIs(err, ErrUntrustedRoot), iopErrStr(err))
	// A NON-empty root set that does not contain the issuing CA must be refused: a root check that only
	// distinguishes "right root" from "no roots" would let any unrelated root through.
	foreignPEM, ferr := os.ReadFile(os.Getenv("HELIX_LCP_INTEROP_FOREIGN_PEM"))
	if ferr != nil {
		t.Fatal(ferr)
	}
	var foreign *x509.Certificate
	for rest := foreignPEM; foreign == nil; {
		var fb *pem.Block
		fb, rest = pem.Decode(rest)
		if fb == nil {
			break
		}
		if fb.Type == "CERTIFICATE" {
			foreign, _ = x509.ParseCertificate(fb.Bytes)
		}
	}
	if foreign == nil {
		t.Fatal("foreign pem holds no parseable certificate")
	}
	c5, _ := OpenContainer(bytes.NewReader(fulfilled), int64(len(fulfilled)))
	_, err = c5.Unlock(phrase, CertificatePolicy{Roots: []*x509.Certificate{foreign}})
	rec("foreign_root_refused", iopMustIs(err, ErrUntrustedRoot), "foreign_subject="+foreign.Subject.String()+" got="+iopErrStr(err))
	tampered := bytes.Replace(licBytes, []byte(`"Interop User"`), []byte(`"Interop Usex"`), 1)
	if bytes.Equal(tampered, licBytes) {
		tampered = bytes.Replace(licBytes, []byte(`interop-user-1`), []byte(`interop-user-2`), 1)
	}
	if fl, e := Fulfill(tampered, bytes.NewReader(pub), int64(len(pub)), AllowUnverified()); e != nil {
		rec("tampered_license_signature_refused", errors.New("could not build the tampered container: "+e.Error()), "")
	} else if c4, e2 := OpenContainer(bytes.NewReader(fl), int64(len(fl))); e2 != nil {
		rec("tampered_license_signature_refused", errors.New("could not open the tampered container: "+e2.Error()), "")
	} else {
		_, err = c4.Unlock(phrase, CertificatePolicy{Roots: []*x509.Certificate{root}})
		rec("tampered_license_signature_refused", iopMustErr(err), iopErrStr(err))
	}

	// 7. LSD against the real Status Server.
	var statusHref string
	for _, l := range lic.Links {
		if l.Rel == "status" || iopContains(l.Rels, "status") {
			statusHref = l.Href
		}
	}
	if statusHref == "" {
		rec("license_has_status_link", errors.New("no status link"), "")
		return
	}
	lsd := &LSDClient{HTTP: httpc, DeviceID: "interop-device-1", DeviceName: "Helix interop device"}
	st, err := lsd.FetchStatus(ctx, statusHref)
	rec("helix_fetches_provider_status_document", err, iopStatusOf(st))
	if err != nil {
		return
	}
	rec("status_ready_and_license_usable", iopBoolErr(st.Status == StatusReady && st.LicenseUsable(), "status "+string(st.Status)), iopStatusOf(st))
	// The reference provider stamps updated.license with whole-second resolution. NeedsLicenseRefresh
	// compares strictly (After), so an update inside the issuing second is not detectable by ANY
	// client; a first, uncommitted attempt of HRD-046 hit exactly that. Wait out the second so the check is deterministic.
	time.Sleep(1100 * time.Millisecond)
	st2, err := lsd.Register(ctx, st)
	rec("helix_registers_device_status_becomes_active", err, iopStatusOf(st2))
	if err != nil {
		return
	}
	st3, err := lsd.Return(ctx, st2)
	rec("helix_returns_license_status_becomes_returned", err, iopStatusOf(st3))
	if err == nil {
		d := "status.updated.license=" + st3.UpdatedLicense.Format(time.RFC3339Nano) + " status.updated.status=" + st3.UpdatedStatus.Format(time.RFC3339Nano) +
			" license.issued=" + lic.Issued.Format(time.RFC3339Nano) + " license.updated=" + lic.Updated.Format(time.RFC3339Nano)
		rec("needs_license_refresh_after_return", iopBoolErr(NeedsLicenseRefresh(st3, lic), "license not flagged stale"), d)
		if raw, e := httpc.Get(statusHref); e != nil {
			rec("raw_status_document_after_return", errors.New("fetch failed: "+iopStripURL(e).Error()), "")
		} else {
			b, _ := io.ReadAll(io.LimitReader(raw.Body, 1<<20))
			raw.Body.Close()
			info("raw_status_document_after_return", string(b))
		}
		if l, ok := st3.SelectLicenseLink([]string{""}); ok {
			rec("returned_status_offers_license_link", nil, l.Href)
		} else {
			rec("returned_status_offers_license_link", errors.New("no license link selected from the returned status"), "")
		}
	}
	h := sha256.Sum256(licBytes)
	info("license_document_sha256", hex.EncodeToString(h[:]))
}

func iopStripURL(err error) error {
	if err == nil {
		return nil
	}
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		return errors.New(s[i+2:])
	}
	return err
}
func iopItoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
func iopBoolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
func iopMustErr(err error) error {
	if err == nil {
		return errors.New("expected a refusal, got success")
	}
	return nil
}
func iopMustIs(err, target error) error {
	if !errors.Is(err, target) {
		return errors.New("expected " + target.Error() + ", got " + iopErrStr(err))
	}
	return nil
}
func iopBoolErr(ok bool, msg string) error {
	if ok {
		return nil
	}
	return errors.New(msg)
}
func iopErrStr(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}
func iopContains(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}
func iopStatusOf(d *StatusDocument) string {
	if d == nil {
		return ""
	}
	return "status=" + string(d.Status)
}
