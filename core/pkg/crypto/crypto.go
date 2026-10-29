// Package crypto implements the E2E sync-payload envelope (docs/15,
// plan P-2 #15). Interim implementation: AES-256-GCM with a
// SHA-256-counter KDF — stdlib-only so it runs everywhere today.
// PRODUCTION REPLACEMENT: filippo.io/age (X25519) per
// specs/research_go_libraries.md; the Envelope format is versioned so the
// swap is wire-compatible at the API level.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// EnvelopeVersion is the wire format version (1 = AES-GCM/SHA-256-KDF).
const EnvelopeVersion byte = 1

var (
	ErrBadEnvelope  = errors.New("crypto: malformed envelope")
	ErrDecrypt      = errors.New("crypto: decryption failed (wrong key or tampered)")
	ErrKeySize      = errors.New("crypto: key must be 32 bytes")
)

// kdf derives a 32-byte key from passphrase+salt via iterated SHA-256
// (interim; age/scrypt replaces this in production).
func kdf(passphrase string, salt []byte) []byte {
	h := sha256.New()
	h.Write([]byte(passphrase))
	h.Write(salt)
	sum := h.Sum(nil)
	// 10k stretch rounds (interim)
	for i := 0; i < 10000; i++ {
		h2 := sha256.New()
		h2.Write(sum)
		h2.Write(salt)
		binary.Write(h2, binary.BigEndian, uint64(i))
		sum = h2.Sum(nil)
	}
	return sum
}

// Encrypt seals plaintext with passphrase. Wire format:
//
//	version(1) | salt(16) | nonce(12) | ciphertext+tag
func Encrypt(passphrase string, plaintext []byte) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	key := kdf(passphrase, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := []byte{EnvelopeVersion}
	out = append(out, salt...)
	out = append(out, nonce...)
	out = gcm.Seal(out, nonce, plaintext, nil)
	return out, nil
}

// Decrypt opens an envelope produced by Encrypt.
func Decrypt(passphrase string, envelope []byte) ([]byte, error) {
	if len(envelope) < 1+16+12+16 {
		return nil, ErrBadEnvelope
	}
	if envelope[0] != EnvelopeVersion {
		return nil, fmt.Errorf("%w: version %d", ErrBadEnvelope, envelope[0])
	}
	salt := envelope[1:17]
	nonce := envelope[17:29]
	ct := envelope[29:]
	key := kdf(passphrase, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, ErrDecrypt
	}
	return pt, nil
}
