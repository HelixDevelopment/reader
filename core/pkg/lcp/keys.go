package lcp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrCiphertextSize reports a ciphertext too short to hold an IV and one
	// padded block, or not a whole multiple of the block size.
	ErrCiphertextSize = errors.New("lcp: ciphertext has invalid size")
	// ErrBadPadding reports a padding length outside 1..16 or longer than the
	// data. The Basic Profile uses the xmlenc padding scheme (pad length in
	// the final byte only, other pad bytes arbitrary), so the other pad bytes
	// are deliberately not inspected; see padLength.
	ErrBadPadding = errors.New("lcp: invalid padding")
	// ErrWrongUserKey reports that the passphrase produced no usable key for
	// this license, detected through the §3.4 key_check.
	ErrWrongUserKey = errors.New("lcp: passphrase does not match this license")
	// ErrContentKeySize reports an unwrapped Content Key of the wrong length.
	ErrContentKeySize = errors.New("lcp: content key must be 32 bytes")
)

// UserKey derives the User Key from a User Passphrase: SHA-256 over the raw
// UTF-8 bytes, with no normalization or escaping of any kind (§4.2).
func UserKey(passphrase string) []byte {
	sum := sha256.Sum256([]byte(passphrase))
	return sum[:]
}

// UserKeyHex returns the hex rendering of the same User Key. Some providers
// express the User Key in hex when handing it between systems (§4.4 point 3);
// the key used for AES is always the 32 raw bytes.
func UserKeyHex(passphrase string) string {
	return fmt.Sprintf("%x", UserKey(passphrase))
}

// CheckPassphrase confirms a passphrase against the license's key_check. It is
// the only sanctioned way to tell a wrong passphrase apart from a corrupt
// archive without attempting a full resource decryption.
func (l *License) CheckPassphrase(passphrase string) error {
	if err := l.CheckProfile(); err != nil {
		return err
	}
	return l.CheckUserKey(UserKey(passphrase))
}

// CheckUserKey confirms a derived or externally supplied User Key against
// key_check, which is the license id encrypted with the User Key under the
// Content Key algorithm (§3.4).
func (l *License) CheckUserKey(userKey []byte) error {
	want, err := base64.StdEncoding.DecodeString(l.Encryption.UserKey.KeyCheck)
	if err != nil {
		return fmt.Errorf("%w: key_check is not valid base64: %v", ErrLicenseSyntax, err)
	}
	got, err := DecryptCBC(userKey, want)
	if err != nil {
		return ErrWrongUserKey
	}
	if string(got) != l.ID {
		return ErrWrongUserKey
	}
	return nil
}

// ContentKey unwraps the Content Key with a User Key already checked by
// CheckUserKey. Reading Systems must not keep the unwrapped key next to the
// license file, and §7.2 forbids storing it unencrypted at all.
func (l *License) ContentKey(userKey []byte) ([]byte, error) {
	if err := l.CheckProfile(); err != nil {
		return nil, err
	}
	enc, err := base64.StdEncoding.DecodeString(l.Encryption.ContentKey.EncryptedValue)
	if err != nil {
		return nil, fmt.Errorf("%w: content_key/encrypted_value is not valid base64: %v", ErrLicenseSyntax, err)
	}
	key, err := DecryptCBC(userKey, enc)
	if err != nil {
		return nil, err
	}
	if len(key) != ContentKeyBytes {
		return nil, fmt.Errorf("%w: got %d", ErrContentKeySize, len(key))
	}
	return key, nil
}

// DecryptUser returns a copy of the user object with any member named in
// user/encrypted decrypted with the User Key (§3.7).
func (l *License) DecryptUser(userKey []byte) (User, error) {
	out := l.User
	fields := map[string]*string{
		"id": &out.ID, "email": &out.Email, "name": &out.Name,
	}
	for _, name := range l.User.Encrypted {
		f, ok := fields[name]
		if !ok {
			continue // §7.2: ignore what we do not understand
		}
		raw, err := base64.StdEncoding.DecodeString(*f)
		if err != nil {
			return out, fmt.Errorf("%w: user/%s is not valid base64: %v", ErrLicenseSyntax, name, err)
		}
		plain, err := DecryptCBC(userKey, raw)
		if err != nil {
			return out, fmt.Errorf("lcp: cannot decrypt user/%s: %w", name, err)
		}
		*f = string(plain)
	}
	return out, nil
}

// DecryptCBC opens a Basic Profile ciphertext: the first 16 bytes are the IV,
// the remainder is AES-256-CBC with a pad block.
func DecryptCBC(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 2*aes.BlockSize || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("%w: %d bytes", ErrCiphertextSize, len(ciphertext))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv := ciphertext[:aes.BlockSize]
	body := make([]byte, len(ciphertext)-aes.BlockSize)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(body, ciphertext[aes.BlockSize:])
	n, err := padLength(body)
	if err != nil {
		return nil, err
	}
	return body[:len(body)-n], nil
}

// EncryptCBC seals plaintext the way the Basic Profile expects: a random IV
// followed by AES-256-CBC with the xmlenc pad scheme.
func EncryptCBC(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded, err := pad(plaintext, aes.BlockSize)
	if err != nil {
		return nil, err
	}
	out := make([]byte, aes.BlockSize+len(padded))
	if _, err := io.ReadFull(rand.Reader, out[:aes.BlockSize]); err != nil {
		return nil, err
	}
	cipher.NewCBCEncrypter(block, out[:aes.BlockSize]).CryptBlocks(out[aes.BlockSize:], padded)
	return out, nil
}

// padLength reads the xmlenc/W3C pad scheme: the final byte is the pad length,
// the preceding pad bytes carry arbitrary values (readium-lcp-server fills them
// with random bytes). Requiring PKCS#7 uniformity here would reject ciphertext
// that the reference implementation produces, so only the length byte is used.
// A malformed length is still refused rather than clamped. The only caller is
// DecryptCBC, whose size guard guarantees at least one full block, so 1..16 is
// also <= len(plaintext).
func padLength(plaintext []byte) (int, error) {
	n := int(plaintext[len(plaintext)-1])
	if n < 1 || n > aes.BlockSize {
		return 0, fmt.Errorf("%w: length byte %d over %d bytes", ErrBadPadding, n, len(plaintext))
	}
	return n, nil
}

// pad appends an xmlenc-scheme pad block: random bytes, then the count.
func pad(plaintext []byte, blockSize int) ([]byte, error) {
	n := blockSize - len(plaintext)%blockSize
	out := make([]byte, len(plaintext)+n)
	copy(out, plaintext)
	if n > 1 {
		if _, err := rand.Read(out[len(plaintext) : len(out)-1]); err != nil {
			return nil, err
		}
	}
	out[len(out)-1] = byte(n)
	return out, nil
}

// SealKeys encrypts a Content Key and the license id for a User Key, yielding
// the encryption/content_key/encrypted_value and encryption/user_key/key_check
// members of a License Document.
func SealKeys(userKey, contentKey []byte, licenseID string) (encryptedValue, keyCheck string, err error) {
	if len(contentKey) != ContentKeyBytes {
		return "", "", fmt.Errorf("%w: got %d", ErrContentKeySize, len(contentKey))
	}
	kb, err := EncryptCBC(userKey, contentKey)
	if err != nil {
		return "", "", err
	}
	cb, err := EncryptCBC(userKey, []byte(licenseID))
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(kb), base64.StdEncoding.EncodeToString(cb), nil
}
