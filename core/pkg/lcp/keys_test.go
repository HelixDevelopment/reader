package lcp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestUserKeyIsRawSHA256OfPassphrase(t *testing.T) {
	// §4.2: SHA-256 over the UTF-8 bytes, with no normalization or escaping of
	// any kind before hashing.
	for _, pass := range []string{"", "pasSED1!", "  leading and trailing  ", "éŒ… ñ", "a\nc\td"} {
		sum := sha256.Sum256([]byte(pass))
		got := UserKey(pass)
		if string(got) != string(sum[:]) {
			t.Errorf("UserKey(%q) = %x, want %x", pass, got, sum[:])
		}
		if len(got) != 32 {
			t.Errorf("UserKey(%q) is %d bytes, want 32", pass, len(got))
		}
	}
	// Whitespace differences must produce different keys: any normalization
	// shortcut would silently widen the passphrase space.
	if string(UserKey("a b")) == string(UserKey("ab")) || string(UserKey("a b")) == string(UserKey(" a b ")) {
		t.Error("passphrase appears to be normalized before hashing")
	}
	if got, want := UserKeyHex("pasSED1!"), fmt.Sprintf("%x", UserKey("pasSED1!")); got != want {
		t.Errorf("UserKeyHex = %s, want %s", got, want)
	}
}

func TestKeyCheckRoundTripAndWrongPassphrase(t *testing.T) {
	const pass = "correct horse battery staple"
	doc, _ := buildLicenseTB(t, pass, nil)
	lic := mustParseTB(t, doc)

	if err := lic.CheckPassphrase(pass); err != nil {
		t.Fatalf("CheckPassphrase(right): %v", err)
	}
	// The named failure matters: a generic error here would also be produced
	// by a malformed document, hiding a broken key_check implementation.
	for _, wrong := range []string{"", "Correct horse battery staple", "correct horse battery stapl"} {
		if err := lic.CheckPassphrase(wrong); !errors.Is(err, ErrWrongUserKey) {
			t.Errorf("CheckPassphrase(%q) error = %v, want ErrWrongUserKey", wrong, err)
		}
	}
}

func TestKeyCheckDetectsTamperedLicenseID(t *testing.T) {
	const pass = "pasSED1!"
	doc, _ := buildLicenseTB(t, pass, func(m map[string]any) {
		// key_check binds the license id, so changing the id after issuing must
		// be visible even with the correct passphrase.
		m["id"] = "00000000-0000-0000-0000-000000000000"
	})
	lic := mustParseTB(t, doc)
	if err := lic.CheckPassphrase(pass); !errors.Is(err, ErrWrongUserKey) {
		t.Fatalf("tampered id error = %v, want ErrWrongUserKey", err)
	}
}

func TestContentKeyRoundTrip(t *testing.T) {
	const pass = "pasSED1!"
	doc, want := buildLicenseTB(t, pass, nil)
	lic := mustParseTB(t, doc)
	got, err := lic.ContentKey(UserKey(pass))
	if err != nil {
		t.Fatalf("ContentKey: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("content key mismatch:\n got %x\nwant %x", got, want)
	}
}

func TestContentKeyRejectsWrongLength(t *testing.T) {
	// A user key is 32 bytes and a content key is 32 bytes, so the ciphertext
	// sizes are identical; the guard that matters is the unwrapped length.
	const pass = "pasSED1!"
	short := make([]byte, 16)
	if _, err := rand.Read(short); err != nil {
		t.Fatal(err)
	}
	if _, _, err := SealKeys(UserKey(pass), short, "id"); !errors.Is(err, ErrContentKeySize) {
		t.Fatalf("SealKeys(16-byte key) error = %v, want ErrContentKeySize", err)
	}

	doc, _ := buildLicenseTB(t, pass, func(m map[string]any) {
		enc := m["encryption"].(map[string]any)
		ck := enc["content_key"].(map[string]any)
		// 48 plaintext bytes -> a valid CBC ciphertext that is not a key.
		bogus := make([]byte, 48)
		ck["encrypted_value"] = base64.StdEncoding.EncodeToString(mustSeal(t, pass, bogus))
	})
	lic := mustParseTB(t, doc)
	if err := lic.CheckPassphrase(pass); err != nil {
		t.Fatalf("key_check should still pass for a wrong-sized content key: %v", err)
	}
	if _, err := lic.ContentKey(UserKey(pass)); !errors.Is(err, ErrContentKeySize) {
		t.Fatalf("ContentKey error = %v, want ErrContentKeySize", err)
	}
}

func mustSeal(tb testing.TB, pass string, plain []byte) []byte {
	tb.Helper()
	b, err := EncryptCBC(UserKey(pass), plain)
	if err != nil {
		tb.Fatal(err)
	}
	return b
}

func TestDecryptCBCRejectsMalformedCiphertext(t *testing.T) {
	key := UserKey("k")
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{"empty", nil, ErrCiphertextSize},
		{"iv only", make([]byte, 16), ErrCiphertextSize},
		{"half a block after iv", make([]byte, 24), ErrCiphertextSize},
		{"not a block multiple", make([]byte, 45), ErrCiphertextSize},
	} {
		if _, err := DecryptCBC(key, tc.data); !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v, want %v", tc.name, err, tc.want)
		}
	}
	// The pad byte is set on the PLAINTEXT body and then CBC-encrypted, because
	// padLength inspects decrypted bytes: writing it into the ciphertext would
	// assert on whatever a random block happens to decrypt to, which is a
	// ~6%-flaky coin flip rather than a test of padLength.
	// Ordered so that a body longer than one block is tried first: with the
	// block-size bound loosened, pad 25 then truncates silently, which is the
	// outcome worth reporting, while pad 17 over a single block would panic
	// first and hide it.
	for _, tc := range []struct {
		name    string
		bodyLen int
		pad     byte
	}{
		{"pad length 25 over two blocks", 32, 25},   // beyond the block size
		{"pad length 255 over two blocks", 32, 255}, // maximum byte value
		{"pad length 0", 16, 0},                     // refused, not a full block
		{"pad length 17", 16, 17},                   // beyond the block size
	} {
		body := make([]byte, tc.bodyLen)
		if _, err := rand.Read(body); err != nil {
			t.Fatal(err)
		}
		body[tc.bodyLen-1] = tc.pad
		if _, err := DecryptCBC(key, sealRawBody(t, key, body)); !errors.Is(err, ErrBadPadding) {
			t.Errorf("%s: error = %v, want ErrBadPadding", tc.name, err)
		}
	}
}

// sealRawBody CBC-encrypts an exactly block-aligned body under a fresh random
// IV and returns iv||ciphertext, so DecryptCBC recovers body verbatim. It
// bypasses EncryptCBC, which would pad the body and overwrite the byte under
// test.
func sealRawBody(tb testing.TB, key, body []byte) []byte {
	tb.Helper()
	if len(body)%aes.BlockSize != 0 {
		tb.Fatalf("body must be block aligned, got %d bytes", len(body))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		tb.Fatal(err)
	}
	out := make([]byte, aes.BlockSize+len(body))
	if _, err := rand.Read(out[:aes.BlockSize]); err != nil {
		tb.Fatal(err)
	}
	cipher.NewCBCEncrypter(block, out[:aes.BlockSize]).CryptBlocks(out[aes.BlockSize:], body)
	return out
}

// TestDecryptCBCAcceptsReferencePaddingScheme is the interoperability guard for
// block ciphers. The xmlenc pad scheme the reference implementation emits puts
// the pad length in the final byte only and fills the rest with random bytes.
// A reader that insisted on PKCS#7 uniformity would reject every real LCP
// publication, so this test exists to fail if anyone "tightens" padLength.
func TestDecryptCBCAcceptsReferencePaddingScheme(t *testing.T) {
	key := UserKey("pasSED1!")
	plain := []byte("publication resource bytes, deliberately not block aligned")

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		t.Fatal(err)
	}
	padLen := aes.BlockSize - len(plain)%aes.BlockSize
	padded := make([]byte, len(plain)+padLen)
	copy(padded, plain)
	if _, err := rand.Read(padded[len(plain) : len(padded)-1]); err != nil {
		t.Fatal(err)
	}
	padded[len(padded)-1] = byte(padLen)

	out := make([]byte, aes.BlockSize+len(padded))
	copy(out, iv)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out[aes.BlockSize:], padded)

	got, err := DecryptCBC(key, out)
	if err != nil {
		t.Fatalf("DecryptCBC: %v", err)
	}
	if string(got) != string(plain) {
		t.Errorf("got %q, want %q", got, plain)
	}
	// Prove the fixture really is non-PKCS#7, otherwise this test could pass
	// for the reason it was written to exclude.
	if allEqual(padded[len(plain):], byte(padLen)) {
		t.Error("fixture degenerated to PKCS#7; the reference-scheme case is not being exercised")
	}
}

func allEqual(b []byte, v byte) bool {
	for _, x := range b {
		if x != v {
			return false
		}
	}
	return true
}

func TestEncryptDecryptRoundTripEveryLength(t *testing.T) {
	key := UserKey("k")
	for n := 0; n <= 64; n++ {
		plain := make([]byte, n)
		if _, err := rand.Read(plain); err != nil {
			t.Fatal(err)
		}
		sealed, err := EncryptCBC(key, plain)
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		if len(sealed)%aes.BlockSize != 0 || len(sealed) < 2*aes.BlockSize {
			t.Fatalf("n=%d: ciphertext size %d is not a padded CBC output", n, len(sealed))
		}
		got, err := DecryptCBC(key, sealed)
		if err != nil {
			t.Fatalf("n=%d decrypt: %v", n, err)
		}
		if string(got) != string(plain) {
			t.Errorf("n=%d: round trip mismatch", n)
		}
	}
}

func TestDecryptUserFields(t *testing.T) {
	const pass = "pasSED1!"
	uk := UserKey(pass)
	name, err := EncryptCBC(uk, []byte("Ünïcødé Émile"))
	if err != nil {
		t.Fatal(err)
	}
	email, err := EncryptCBC(uk, []byte("emile@example.org"))
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := buildLicenseTB(t, pass, func(m map[string]any) {
		m["user"] = map[string]any{
			"id":        "user-42",
			"name":      base64.StdEncoding.EncodeToString(name),
			"email":     base64.StdEncoding.EncodeToString(email),
			"encrypted": []string{"name", "email"},
		}
	})
	lic := mustParseTB(t, doc)
	user, err := lic.DecryptUser(uk)
	if err != nil {
		t.Fatalf("DecryptUser: %v", err)
	}
	if user.Name != "Ünïcødé Émile" || user.Email != "emile@example.org" {
		t.Errorf("decrypted user = %+v", user)
	}
	if user.ID != "user-42" {
		t.Errorf("unlisted member changed: %q", user.ID)
	}
	// With the wrong key the ciphertext must not resolve to plausible text, and
	// DecryptUser must surface that rather than returning garbage silently.
	other, err := lic.DecryptUser(UserKey("wrong"))
	if err == nil && (other.Name == user.Name || other.Email == user.Email) {
		t.Error("wrong user key recovered the plaintext")
	}
}

func TestDecryptUserIgnoresUnknownEncryptedMembers(t *testing.T) {
	// §7.2: ignore name/value pairs the implementation does not understand.
	doc, _ := buildLicenseTB(t, "pasSED1!", func(m map[string]any) {
		m["user"] = map[string]any{"id": "u1", "encrypted": []string{"telepathy"}}
	})
	lic := mustParseTB(t, doc)
	if _, err := lic.DecryptUser(UserKey("pasSED1!")); err != nil {
		t.Errorf("unknown encrypted member should be ignored, got %v", err)
	}
}

func TestSealKeysRejectsUndecodableBase64(t *testing.T) {
	doc, _ := buildLicenseTB(t, "pasSED1!", func(m map[string]any) {
		enc := m["encryption"].(map[string]any)
		enc["user_key"].(map[string]any)["key_check"] = "not base64 !!!"
	})
	lic := mustParseTB(t, doc)
	err := lic.CheckPassphrase("pasSED1!")
	if err == nil || !strings.Contains(err.Error(), "key_check") {
		t.Fatalf("error = %v, want a key_check base64 failure", err)
	}
}

func TestParseRejectsNonJSONLicense(t *testing.T) {
	if _, err := Parse([]byte(`<license/>`)); !errors.Is(err, ErrLicenseSyntax) {
		t.Errorf("error = %v, want ErrLicenseSyntax", err)
	}
}

// TestLicenseRoundTripsThroughJSONKeepsRawBytes underpins signature
// verification: the producer signs bytes, so the verifier must hash bytes
// derived from Raw rather than from a re-serialization.
func TestLicenseRoundTripsThroughJSONKeepsRawBytes(t *testing.T) {
	doc, _ := buildLicenseTB(t, "pasSED1!", nil)
	lic := mustParseTB(t, doc)
	if string(lic.Raw) != string(doc) {
		t.Error("License.Raw is not the document it was parsed from")
	}
	var generic map[string]any
	if err := json.Unmarshal(lic.Raw, &generic); err != nil {
		t.Fatal(err)
	}
}
