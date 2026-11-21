package lcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	// ErrLicenseSyntax reports a document that is not valid JSON.
	ErrLicenseSyntax = errors.New("lcp: license document is not valid JSON")
	// ErrAmbiguousLicense reports repeated member names in one object. The
	// canonical form is undefined for such a document and decoders silently
	// keep the last value, so it is refused rather than guessed.
	ErrAmbiguousLicense = errors.New("lcp: license document has duplicate JSON object members")
	// ErrSignatureMissing reports a License Document without the signature
	// object required by §3.8.
	ErrSignatureMissing = errors.New("lcp: license document has no signature object")
)

// CanonicalMode selects the escaping convention applied to strings.
//
// The two conventions differ for exactly two character classes: whether
// U+2028/U+2029 are escaped, and the hex case of \u00XX for control characters
// other than the JSON short forms. Rule 5 of §5.3 requires no escaping of
// U+2028/U+2029 and upper-case hex; the reference producer, readium-lcp-server
// sign.Canon at 32570b69d02948801f161e7923648dcc0bbfb178, is Go's
// json.Encoder with SetEscapeHTML(false), which escapes U+2028/U+2029 and
// emits lower-case hex. Signatures only verify when producer and reader agree
// byte for byte, so Helix signs in the reference convention (what real
// producers already emit) and verifies against both.
type CanonicalMode int

const (
	// CanonicalSpec applies §5.3 rule 5 literally.
	CanonicalSpec CanonicalMode = iota
	// CanonicalReference reproduces readium-lcp-server's sign.Canon exactly.
	CanonicalReference
)

// Canonical serializes doc into the canonical form of §5.3: object members
// sorted by Unicode code point at every nesting level, array element order
// preserved, numbers without insignificant zeroes, minimal string escaping,
// no non-significant whitespace.
//
// Rule 2 is recursive, so members of an object inside an array are sorted too.
// The informative example in §5.3.1 leaves {"rel","href","type"} in document
// order, contradicting rule 2; the reference implementation sorts them, and it
// is followed here.
func Canonical(doc []byte) ([]byte, error) {
	return canonicalWithMode(doc, CanonicalSpec)
}

// CanonicalAsReference serializes doc the way the reference LCP server does.
// See CanonicalMode for why both conventions exist.
func CanonicalAsReference(doc []byte) ([]byte, error) {
	return canonicalWithMode(doc, CanonicalReference)
}

func canonicalWithMode(doc []byte, mode CanonicalMode) ([]byte, error) {
	if err := noDuplicateMembers(doc); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
	}
	var b bytes.Buffer
	if err := writeCanonical(&b, v, mode); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// CanonicalForSigning returns the canonical form the §3.8 signature is
// calculated over: the License Document with the signature object removed
// (rule 1) and then canonicalized.
//
// The signature member is excised on the raw token stream instead of decoding
// into a map, so every other member keeps its exact source bytes. A map
// round-trip would collapse duplicate members before Canonical could reject
// them, and a license signed over an ambiguous document could then verify.
func CanonicalForSigning(doc []byte) ([]byte, error) {
	return CanonicalForSigningMode(doc, CanonicalSpec)
}

// CanonicalForSigningMode is CanonicalForSigning under a chosen mode.
func CanonicalForSigningMode(doc []byte, mode CanonicalMode) ([]byte, error) {
	rest, found, err := withoutTopLevelMember(doc, "signature")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrSignatureMissing
	}
	return canonicalWithMode(rest, mode)
}

// canonicalForNewSignature canonicalizes a document that is about to be signed.
// The same byte string results either way, but a producer has no signature
// member to remove yet, so absence is the normal case here rather than an error.
func canonicalForNewSignature(doc []byte, mode CanonicalMode) ([]byte, error) {
	rest, _, err := withoutTopLevelMember(doc, "signature")
	if err != nil {
		return nil, err
	}
	return canonicalWithMode(rest, mode)
}

func withoutTopLevelMember(doc []byte, name string) (rest []byte, found bool, err error) {
	dec := json.NewDecoder(bytes.NewReader(doc))
	tok, err := dec.Token()
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, false, fmt.Errorf("%w: top level is not an object", ErrLicenseSyntax)
	}
	var parts []string
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, false, fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, false, fmt.Errorf("%w: object member name is not a string", ErrLicenseSyntax)
		}
		start := dec.InputOffset()
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, false, fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
		}
		end := dec.InputOffset()
		if key == name {
			found = true
			continue
		}
		// InputOffset after the key token still points at the ":" that
		// separates it from its value, so the slice must step over it.
		value := strings.TrimLeft(string(doc[start:end]), " \t\r\n")
		value = strings.TrimPrefix(value, ":")
		value = strings.TrimLeft(value, " \t\r\n")
		parts = append(parts, strconv.Quote(key)+":"+value)
	}
	if _, err := dec.Token(); err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
	}
	return []byte("{" + strings.Join(parts, ",") + "}"), found, nil
}

func noDuplicateMembers(doc []byte) error {
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	if err := scanNoDupes(dec); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing data after the top-level value", ErrLicenseSyntax)
	}
	return nil
}

func scanNoDupes(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch d {
	case '{':
		seen := make(map[string]bool)
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
			}
			key, ok := keyTok.(string)
			if !ok {
				return fmt.Errorf("%w: object member name is not a string", ErrLicenseSyntax)
			}
			if seen[key] {
				return fmt.Errorf("%w: %q", ErrAmbiguousLicense, key)
			}
			seen[key] = true
			if err := scanNoDupes(dec); err != nil {
				return err
			}
		}
	case '[':
		for dec.More() {
			if err := scanNoDupes(dec); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: unexpected %v", ErrLicenseSyntax, d)
	}
	if _, err := dec.Token(); err != nil {
		return fmt.Errorf("%w: %v", ErrLicenseSyntax, err)
	}
	return nil
}

func writeCanonical(b *bytes.Buffer, v any, mode CanonicalMode) error {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if t {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case json.Number:
		n, err := canonicalNumber(t, mode)
		if err != nil {
			return err
		}
		b.WriteString(n)
	case string:
		writeCanonicalString(b, t, mode)
	case []any:
		b.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writeCanonical(b, e, mode); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys) // UTF-8 byte order is code point order
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			writeCanonicalString(b, k, mode)
			b.WriteByte(':')
			if err := writeCanonical(b, t[k], mode); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	default:
		return fmt.Errorf("lcp: cannot canonicalize %T", v)
	}
	return nil
}

// writeCanonicalString escapes only what JSON requires: quote, backslash and
// U+0000-U+001F. Rule 5 additionally demands upper-case hex digits, which is
// what CanonicalSpec emits. encoding/json cannot be used here because it also
// escapes <, > and & when HTML escaping is on. Invalid UTF-8 never reaches
// this function; decoding already replaced it with U+FFFD.
func writeCanonicalString(b *bytes.Buffer, s string, mode CanonicalMode) {
	format := `\u%04X`
	if mode == CanonicalReference {
		format = `\u%04x`
	}
	b.WriteByte('"')
	for i := 0; i < len(s); {
		c := s[i]
		if c >= utf8.RuneSelf {
			r, size := utf8.DecodeRuneInString(s[i:])
			if mode == CanonicalReference && (r == 0x2028 || r == 0x2029) {
				fmt.Fprintf(b, format, r)
				i += size
				continue
			}
			b.WriteString(s[i : i+size])
			i += size
			continue
		}
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 {
				fmt.Fprintf(b, format, c)
			} else {
				b.WriteByte(c)
			}
		}
		i++
	}
	b.WriteByte('"')
}

// canonicalNumber applies rule 4. Integers need no work: JSON already forbids
// leading zeroes and a leading "+". Anything else carrying a fraction part or
// an exponent is rewritten in normalized scientific notation with an
// upper-case E, using the literal's digits rather than a float64 so no
// precision is lost.
//
// "1e2" is treated as scientific because the rule keys on the notation
// present in the document, not on whether the value happens to be integral.
//
// In CanonicalReference mode the literal is passed through untouched, because
// that is what the reference producer's json.Encoder does with a json.Number.
// Rule 4 therefore binds only in spec mode; no field defined by the Basic
// Encryption Profile is a non-integer, so the divergence cannot arise in a
// real License Document.
func canonicalNumber(n json.Number, mode CanonicalMode) (string, error) {
	s := n.String()
	if mode == CanonicalReference {
		return s, nil
	}
	if !strings.ContainsAny(s, ".eE") {
		if s == "-0" {
			return "0", nil
		}
		return s, nil
	}

	sign := ""
	rest := s
	if strings.HasPrefix(rest, "-") {
		sign, rest = "-", rest[1:]
	}
	exp := 0
	if i := strings.IndexAny(rest, "eE"); i >= 0 {
		e, err := strconv.Atoi(rest[i+1:])
		if err != nil {
			return "", fmt.Errorf("%w: bad exponent in %s: %v", ErrLicenseSyntax, s, err)
		}
		if e > 1<<30 || e < -(1<<30) {
			return "", fmt.Errorf("%w: exponent out of range in %s", ErrLicenseSyntax, s)
		}
		exp, rest = e, rest[:i]
	}
	intPart, fracPart := rest, ""
	if i := strings.IndexByte(rest, '.'); i >= 0 {
		intPart, fracPart = rest[:i], rest[i+1:]
	}

	digits := intPart + fracPart
	// Value == digits as an integer, scaled by 10^power.
	power := exp - len(fracPart)
	trimmed := strings.TrimRight(digits, "0")
	power += len(digits) - len(trimmed) // each dropped trailing zero scales up
	digits = trimmed
	if digits == "" {
		return "0", nil // the value is zero, sign included
	}
	digits = strings.TrimLeft(digits, "0") // leading zeroes do not change scale
	if digits == "" {
		return "0", nil
	}

	mantissa := digits[:1]
	if len(digits) > 1 {
		mantissa += "." + digits[1:]
	}
	return sign + mantissa + "E" + strconv.Itoa(len(digits)-1+power), nil
}
