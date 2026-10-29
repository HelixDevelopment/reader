package crypto

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	pt := []byte(`{"book":"b1","pos":{"chapter":3}}`)
	env, err := Encrypt("correct horse battery staple", pt)
	if err != nil {
		t.Fatal(err)
	}
	if env[0] != EnvelopeVersion {
		t.Fatal("version byte")
	}
	out, err := Decrypt("correct horse battery staple", env)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, pt) {
		t.Fatal("round trip mismatch")
	}
}

func TestWrongPassphrase(t *testing.T) {
	env, _ := Encrypt("right", []byte("secret"))
	if _, err := Decrypt("wrong", env); err != ErrDecrypt {
		t.Fatalf("want ErrDecrypt, got %v", err)
	}
}

func TestTamperDetected(t *testing.T) {
	env, _ := Encrypt("k", []byte("payload-data"))
	env[len(env)-1] ^= 0xFF
	if _, err := Decrypt("k", env); err != ErrDecrypt {
		t.Fatalf("tamper must be detected, got %v", err)
	}
}

func TestNonDeterministicCiphertext(t *testing.T) {
	a, _ := Encrypt("k", []byte("same"))
	b, _ := Encrypt("k", []byte("same"))
	if bytes.Equal(a, b) {
		t.Fatal("random salt+nonce must make ciphertexts differ")
	}
}

func TestBadEnvelope(t *testing.T) {
	if _, err := Decrypt("k", []byte{9, 1, 2}); err == nil {
		t.Fatal("short envelope must error")
	}
	if _, err := Decrypt("k", append([]byte{99}, make([]byte, 60)...)); err == nil {
		t.Fatal("bad version must error")
	}
}
