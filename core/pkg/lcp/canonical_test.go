package lcp

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func readTestdataTB(tb testing.TB, name string) string {
	tb.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		tb.Fatalf("read fixture %s: %v", name, err)
	}
	return strings.TrimRight(string(b), "\n")
}

// TestCanonicalSpecExample checks the §5.3.1 example end to end: the pretty
// License Document in the spec must canonicalize to the string the spec prints,
// except that rule 2 is recursive while the example left the members of
// links[0] in document order. The assertion is written as that exact
// relationship, so the divergence stays visible instead of hidden in a blob.
func TestCanonicalSpecExample(t *testing.T) {
	pretty := []byte(readTestdataTB(t, "spec-5.3.1-pretty.json"))
	specCanonical := readTestdataTB(t, "spec-5.3.1-canonical.json")

	got, err := Canonical(pretty)
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}

	unsorted := `{"rel":"hint","href":"https://www.imaginaryebookretailer.com/lcp/hint","type":"text/html"}`
	sorted := `{"href":"https://www.imaginaryebookretailer.com/lcp/hint","rel":"hint","type":"text/html"}`
	if !strings.Contains(specCanonical, unsorted) {
		t.Fatalf("fixture drifted from the published example: %s", specCanonical)
	}
	want := strings.Replace(specCanonical, unsorted, sorted, 1)
	if string(got) != want {
		t.Errorf("canonical form mismatch\n got: %s\nwant: %s", got, want)
	}
}

// TestCanonicalSpecDigestVector pins the digest chain of §5.4: SHA-256 over the
// canonical bytes and nothing else. It also proves the fixture is a faithful
// copy, since one altered character would move the digest.
func TestCanonicalSpecDigestVector(t *testing.T) {
	canonical := []byte(readTestdataTB(t, "spec-5.3.1-canonical.json"))
	if got := fmt.Sprintf("%x", sha256.Sum256(canonical)); got != specDigest {
		t.Errorf("sha256(canonical) = %s, want %s", got, specDigest)
	}
}

// TestSpecExampleIsNotAConformantLicense documents a second defect in the same
// example: §5.4.1's License Document has no `provider` member, although §3.3
// and the License Document JSON Schema both require it. Helix refuses it.
//
// The consequence for this package is that the example cannot be pushed through
// Parse, so the signature probe below reads its members straight from JSON.
func TestSpecExampleIsNotAConformantLicense(t *testing.T) {
	signed := []byte(readTestdataTB(t, "spec-5.4.1-signed-license.json"))
	if _, err := Parse(signed); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("Parse(spec example) error = %v, want ErrIncomplete", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(signed, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["provider"]; ok {
		t.Error("the spec example now carries a provider member; it should parse, and this finding is stale")
	}
}

// TestSpecExampleSignatureIsUnverifiable records the third defect, so that a
// revision fixing it fails this test and gets noticed.
//
// §5.4.1 publishes a canonical form, its SHA-256 (which does match), a
// signature value and a Provider Certificate. That signature verifies against
// neither the published canonical form nor the document minus its signature
// under either member ordering or either escaping convention. The embedded
// certificate is a self-signed, non-CA, SHA-1-signed "localhost" certificate
// that expired 2015-01-02. Consequence: the example supplies a format and
// digest vector but no cryptographic interoperability vector, which is why
// interop is tested against the reference implementation instead.
func TestSpecExampleSignatureIsUnverifiable(t *testing.T) {
	signed := []byte(readTestdataTB(t, "spec-5.4.1-signed-license.json"))
	var m map[string]json.RawMessage
	if err := json.Unmarshal(signed, &m); err != nil {
		t.Fatal(err)
	}
	var so struct{ Algorithm, Certificate, Value string }
	if err := json.Unmarshal(m["signature"], &so); err != nil {
		t.Fatal(err)
	}
	if so.Algorithm != AlgRSASHA256 {
		t.Fatalf("spec example signature algorithm changed to %s; re-read §5.4.1", so.Algorithm)
	}
	der, err := base64.StdEncoding.DecodeString(so.Certificate)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := base64.StdEncoding.DecodeString(so.Value)
	if err != nil {
		t.Fatal(err)
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("spec example certificate key is %T", cert.PublicKey)
	}
	if cert.IsCA {
		t.Error("spec example certificate is now a CA certificate; re-read §5.2.1 before trusting this finding")
	}

	candidates := map[string][]byte{
		"published canonical form":    []byte(readTestdataTB(t, "spec-5.3.1-canonical.json")),
		"sorted pretty example":       mustCanonical(t, []byte(readTestdataTB(t, "spec-5.3.1-sorted.json"))),
		"minus signature, spec mode":  mustCanonicalSigning(t, signed, CanonicalSpec),
		"minus signature, ref mode":   mustCanonicalSigning(t, signed, CanonicalReference),
		"array order preserved":       arrayOrderCanonical(t, signed),
		"reference encoder, unsorted": referenceCanon(t, mustRemoveSignature(t, signed)),
	}
	for name, canonical := range candidates {
		sum := sha256.Sum256(canonical)
		if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err == nil {
			t.Errorf("spec example signature unexpectedly verifies over %s: the finding is stale, replace this test with a positive vector", name)
		}
	}
	if len(candidates) != 6 {
		t.Fatalf("candidate set shrank to %d; this test only means something if all six are tried", len(candidates))
	}
	if !cert.NotAfter.Before(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("spec example certificate is no longer expired (%s); re-read §5.4.1 before trusting this finding", cert.NotAfter)
	}
}

// TestCanonicalIsDeterministicAcrossMapOrder pins rule 2 by its real
// consequence: a canonicalizer that emitted members in Go's randomized map
// order would still pass a two-key fixture some of the time. Repeating one
// document many times makes that probability negligible, and two different
// byte strings for one document would mean two different signatures.
func TestCanonicalIsDeterministicAcrossMapOrder(t *testing.T) {
	const doc = `{"z":1,"m":2,"a":3,"Q":4,"_":5,"é":6,"B":7,"1":8,"😀":9,"b":10,"A":11,"0":12}`
	const want = `{"0":12,"1":8,"A":11,"B":7,"Q":4,"_":5,"a":3,"b":10,"m":2,"z":1,"é":6,"😀":9}`
	distinct := map[string]int{}
	for i := 0; i < 200; i++ {
		got, err := Canonical([]byte(doc))
		if err != nil {
			t.Fatalf("Canonical: %v", err)
		}
		distinct[string(got)]++
	}
	if len(distinct) != 1 {
		t.Fatalf("canonical form depends on map iteration order: %d distinct outputs over 200 runs", len(distinct))
	}
	for got := range distinct {
		if got != want {
			t.Errorf("Canonical\n got: %s\nwant: %s", got, want)
		}
	}
}

func mustCanonical(tb testing.TB, doc []byte) []byte {
	tb.Helper()
	b, err := Canonical(doc)
	if err != nil {
		tb.Fatal(err)
	}
	return b
}

func mustCanonicalSigning(tb testing.TB, doc []byte, mode CanonicalMode) []byte {
	tb.Helper()
	b, err := CanonicalForSigningMode(doc, mode)
	if err != nil {
		tb.Fatal(err)
	}
	return b
}

func mustRemoveSignature(tb testing.TB, doc []byte) []byte {
	tb.Helper()
	b, _, err := withoutTopLevelMember(doc, "signature")
	if err != nil {
		tb.Fatal(err)
	}
	return b
}

// arrayOrderCanonical canonicalizes with members of objects inside arrays left
// in document order, which is what the §5.3.1 example shows.
func arrayOrderCanonical(tb testing.TB, doc []byte) []byte {
	tb.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(mustRemoveSignature(tb, doc), &m); err != nil {
		tb.Fatal(err)
	}
	var links []map[string]any
	if err := json.Unmarshal(m["links"], &links); err != nil {
		tb.Fatal(err)
	}
	// Rebuild with links emitted as rel, href, type, the example's order.
	var b strings.Builder
	b.WriteByte('[')
	for i, l := range links {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"rel":%q,"href":%q,"type":%q}`, l["rel"], l["href"], l["type"])
	}
	b.WriteByte(']')
	m["links"] = json.RawMessage(b.String())
	out, err := json.Marshal(m)
	if err != nil {
		tb.Fatal(err)
	}
	return out
}

// referenceCanon reproduces readium-lcp-server@32570b69d02948801f161e7923648dcc0bbfb178
// sign.Canon: decode with UseNumber, re-encode with encoding/json and HTML
// escaping disabled, drop the trailing newline. It is an independent oracle
// because it is stdlib behaviour, not this package's code.
func referenceCanon(tb testing.TB, doc []byte) []byte {
	tb.Helper()
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		tb.Fatal(err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		tb.Fatal(err)
	}
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// TestCanonicalMatchesReferenceProducer is the interoperability half of
// canonicalization: for the field content real License Documents carry, this
// package must emit exactly the bytes the reference LCP server signs.
func TestCanonicalMatchesReferenceProducer(t *testing.T) {
	sealed, _ := buildLicenseTB(t, "pasSED1!", nil)
	docs := [][]byte{
		[]byte(readTestdataTB(t, "spec-5.3.1-pretty.json")),
		sealed,
		[]byte(`{"a":"<b>&'\"","z":[1,2,3],"nested":{"deep":{"deeper":[{"b":1,"a":2}]}}}`),
		[]byte(`{"user":{"name":"Ünïcødé – émile","email":"a@b.c"},"rights":{"print":0}}`),
		[]byte(`{"hint":"tab\tnewline\nquote\"back\\slash"}`),
		// Rule 4 must stay inert in reference mode, so a non-integer belongs in
		// this corpus even though the Basic Profile defines no such field.
		[]byte(`{"w":1.5,"scaled":100.0,"e":1e2}`),
		[]byte(`{"n":12345678901234567890.5,"neg":-0}`),
		[]byte(`{"u":"a b line-separator","e":"emoji 📖 tail"}`),
	}
	for _, doc := range docs {
		got, err := CanonicalAsReference(doc)
		if err != nil {
			t.Fatalf("CanonicalAsReference(%s): %v", doc, err)
		}
		want := referenceCanon(t, doc)
		if !bytes.Equal(got, want) {
			t.Errorf("reference-mode divergence\n got: %s\nwant: %s", got, want)
		}
	}
}

// TestCanonicalSpecModeEscaping pins rule 5 where it deliberately departs from
// the reference encoder: no escaping of U+2028/U+2029, upper-case hex.
func TestCanonicalSpecModeEscaping(t *testing.T) {
	// U+2028/U+2029 written raw (valid inside a JSON string), U+001F written as
	// an escape, since a raw C0 control character is not legal JSON at all.
	in := "{\"s\":\"a b c\\u001F\",\"h\":\"<x>&y\"}"
	got, err := Canonical([]byte(in))
	if err != nil {
		t.Fatalf("Canonical(%q): %v", in, err)
	}
	want := "{\"h\":\"<x>&y\",\"s\":\"a b c\\u001F\"}"
	if string(got) != want {
		t.Errorf("spec-mode escaping\n got: %q\nwant: %q", got, want)
	}

	ref, err := CanonicalAsReference([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, ref) {
		t.Error("spec and reference modes must differ on this input, otherwise the mode switch is dead code")
	}
	if !strings.Contains(string(ref), `\u2028`) || !strings.Contains(string(ref), `\u2029`) {
		t.Errorf("reference mode should escape U+2028/U+2029: %q", ref)
	}
	// Lower-case hex is the only difference the reference mode may show for
	// U+001F, and it is exactly the case rule 5 forbids.
	if !strings.Contains(string(ref), `\u001f`) {
		t.Errorf("reference mode should use lower-case hex: %q", ref)
	}
	if strings.Contains(string(ref), `\u001F`) {
		t.Errorf("reference mode leaked the upper-case form: %q", ref)
	}
	if strings.Contains(string(got), `\u2028`) {
		t.Errorf("spec mode must not escape U+2028: %q", got)
	}
	// The two modes must agree on every real License Document field, which is
	// why the mode switch is invisible in practice.
	for _, doc := range []string{
		`{"text_hint":"Enter your email address"}`,
		`{"href":"https://x.example/a?b=1&c=2"}`,
		`{"name":"Ünïcødé émile – Œ"}`,
	} {
		a, b := mustCanonical(t, []byte(doc)), mustCanonicalAsRef(t, []byte(doc))
		if !bytes.Equal(a, b) {
			t.Errorf("modes diverge on ordinary field content: spec=%s ref=%s", a, b)
		}
	}
}

func mustCanonicalAsRef(tb testing.TB, doc []byte) []byte {
	tb.Helper()
	b, err := CanonicalAsReference(doc)
	if err != nil {
		tb.Fatal(err)
	}
	return b
}
func TestCanonicalSortingAndArrayOrder(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"b":1,"a":2}`, `{"a":2,"b":1}`},
		{`{"é":1,"A":2,"_":3,"b":4}`, `{"A":2,"_":3,"b":4,"é":1}`},
		{`{"😀":1,"é":2}`, `{"é":2,"😀":1}`},
		{`[{"b":1,"a":2}]`, `[{"a":2,"b":1}]`},         // rule 2 is recursive
		{`{"k":[3,{"x":1},2]}`, `{"k":[3,{"x":1},2]}`}, // rule 3 keeps array order
		{`{"l":[{"n":2},{"n":1}]}`, `{"l":[{"n":2},{"n":1}]}`},
		{`{"a":null,"b":true,"c":false}`, `{"a":null,"b":true,"c":false}`},
	}
	for _, c := range cases {
		got, err := Canonical([]byte(c.in))
		if err != nil {
			t.Errorf("Canonical(%s): %v", c.in, err)
			continue
		}
		if string(got) != c.want {
			t.Errorf("Canonical(%s)\n got: %s\nwant: %s", c.in, got, c.want)
		}
	}
}

func TestCanonicalNumbers(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{"n":10}`, `{"n":10}`},
		{`{"n":-0}`, `{"n":0}`},
		{`{"n":0}`, `{"n":0}`},
		{`{"n":1.5}`, `{"n":1.5E0}`},
		{`{"n":100.0}`, `{"n":1E2}`},
		{`{"n":0.0015}`, `{"n":1.5E-3}`},
		{`{"n":123.450}`, `{"n":1.2345E2}`},
		{`{"n":1e2}`, `{"n":1E2}`},
		{`{"n":0.0}`, `{"n":0}`},
		{`{"n":-2.5}`, `{"n":-2.5E0}`},
		// Digit-exact rewriting, where a float64 round-trip would lose them.
		{`{"n":1.0000000000000000001}`, `{"n":1.0000000000000000001E0}`},
		{`{"n":12345678901234567890.5}`, `{"n":1.23456789012345678905E19}`},
	}
	for _, c := range cases {
		got, err := Canonical([]byte(c.in))
		if err != nil {
			t.Errorf("Canonical(%s): %v", c.in, err)
			continue
		}
		if string(got) != c.want {
			t.Errorf("Canonical(%s) = %s, want %s", c.in, got, c.want)
		}
	}
}

// TestCanonicalFloatDivergenceFromReference documents, rather than hides, the
// one place where the normative text and the reference producer disagree in
// practice-reachable input space: rule 4 demands scientific notation for
// non-integers, encoding/json does not. No field in the Basic Encryption
// Profile is a non-integer, so a real License Document cannot hit this.
func TestCanonicalFloatDivergenceFromReference(t *testing.T) {
	doc := []byte(`{"w":1.5}`)
	spec, err := Canonical(doc)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := CanonicalAsReference(doc)
	if err != nil {
		t.Fatal(err)
	}
	if string(spec) != `{"w":1.5E0}` || string(ref) != `{"w":1.5}` {
		t.Fatalf("unexpected pair: spec=%s ref=%s", spec, ref)
	}
}

func TestCanonicalForSigningRemovesOnlyTopLevelSignature(t *testing.T) {
	doc := []byte(`{"id":"x","signature":{"value":"deadbeef"},"nested":{"signature":{"value":"keepme"}}}`)
	got, err := CanonicalForSigning(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"x","nested":{"signature":{"value":"keepme"}}}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}

	noSig := []byte(`{"id":"x"}`)
	if _, err := CanonicalForSigning(noSig); err == nil {
		t.Fatal("missing signature object must be reported")
	}
}

// TestCanonicalRejectsDuplicateMembers: the ambiguity is rejected in both
// entry points. CanonicalForSigning excises the signature on the raw token
// stream precisely so that duplicates cannot be laundered by a map round-trip.
func TestCanonicalRejectsDuplicateMembers(t *testing.T) {
	doc := []byte(`{"id":"a","id":"b","signature":{"value":"x"}}`)
	if _, err := Canonical(doc); !errors.Is(err, ErrAmbiguousLicense) {
		t.Errorf("Canonical error = %v, want ErrAmbiguousLicense", err)
	}
	// CanonicalForSigning must not launder the ambiguity by way of a map
	// round-trip while excising the signature.
	if _, err := CanonicalForSigning(doc); !errors.Is(err, ErrAmbiguousLicense) {
		t.Errorf("CanonicalForSigning error = %v, want ErrAmbiguousLicense", err)
	}
	nested := []byte(`{"e":{"k":1,"k":2}}`)
	if _, err := Canonical(nested); !errors.Is(err, ErrAmbiguousLicense) {
		t.Errorf("nested duplicate error = %v, want ErrAmbiguousLicense", err)
	}
}

func TestCanonicalRejectsMalformedInput(t *testing.T) {
	for _, in := range []string{``, `not json`, `{"a":1} {"b":2}`, `{"a":}`, `{"a":01}`, `{"a":1e}`, `{"a":1,}`} {
		if _, err := Canonical([]byte(in)); !errors.Is(err, ErrLicenseSyntax) {
			t.Errorf("Canonical(%q) error = %v, want ErrLicenseSyntax", in, err)
		}
	}
	// Canonical is a generic canonicalizer, so non-object top levels are legal
	// JSON it will happily serialize; CanonicalForSigning is the one that
	// requires a License Document.
	if got, err := Canonical([]byte(`[2,{"b":1,"a":2}]`)); err != nil || string(got) != `[2,{"a":2,"b":1}]` {
		t.Errorf("Canonical on array = %s, %v", got, err)
	}
	if _, err := CanonicalForSigning([]byte(`[1,2]`)); err == nil {
		t.Error("CanonicalForSigning accepted a non-object document")
	}
}
