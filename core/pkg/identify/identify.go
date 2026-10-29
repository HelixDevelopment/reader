// Package identify implements the book identification cascade from docs/26:
// ISBN-10/13 validation and conversion, ASIN shape checks, and identifier
// normalisation feeding the metadata + cover pipeline.
package identify

import (
	"errors"
	"strings"

	"github.com/helixreader/helix/core/isbn"
)

// ErrInvalidISBN is returned for checksum/length failures.
var ErrInvalidISBN = errors.New("identify: invalid ISBN")

// CleanISBN strips hyphens/spaces and uppercases X.
func CleanISBN(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if r != '-' && r != ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ValidateISBN reports whether s is a valid ISBN-10 or ISBN-13.
func ValidateISBN(s string) bool {
	s = CleanISBN(s)
	return isbn.Valid10(s) || isbn.Valid13(s)
}

// ISBN10to13 converts a valid ISBN-10 to ISBN-13 (978 prefix + new check digit).
func ISBN10to13(s string) (string, error) {
	s = CleanISBN(s)
	if len(s) != 10 || !isbn.Valid10(s) {
		return "", ErrInvalidISBN
	}
	base := "978" + s[:9]
	sum := 0
	for i := 0; i < 12; i++ {
		v := int(base[i] - '0')
		if i%2 == 1 {
			v *= 3
		}
		sum += v
	}
	check := (10 - sum%10) % 10
	return base + string(rune('0'+check)), nil
}

// ISBN13to10 converts a 978-prefixed ISBN-13 back to ISBN-10.
func ISBN13to10(s string) (string, error) {
	s = CleanISBN(s)
	if len(s) != 13 || !isbn.Valid13(s) || !strings.HasPrefix(s, "978") {
		return "", ErrInvalidISBN
	}
	base := s[3:12]
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(base[i]-'0') * (10 - i)
	}
	rem := sum % 11
	check := (11 - rem) % 11
	var last byte
	switch {
	case check == 10:
		last = 'X'
	default:
		last = byte('0' + check)
	}
	return base + string(last), nil
}

// IsASIN reports whether s looks like an Amazon ASIN (10 chars, starts B0
// for most books, alphanumeric uppercase).
func IsASIN(s string) bool {
	if len(s) != 10 {
		return false
	}
	for _, r := range s {
		if !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return strings.HasPrefix(s, "B0")
}

// Candidate is one identifier extracted from a book's opening pages.
type Candidate struct {
	Scheme string // isbn13, isbn10, asin, oclc, lccn
	Value  string
	// Confidence 0..1: ISBN from OPF metadata ~1.0, regex on OCR text lower.
	Confidence float64
}

// RankCandidates orders candidates deterministically: confidence desc,
// then scheme precedence (isbn13 > isbn10 > asin > oclc > lccn), then value.
func RankCandidates(in []Candidate) []Candidate {
	prec := map[string]int{"isbn13": 0, "isbn10": 1, "asin": 2, "oclc": 3, "lccn": 4}
	out := append([]Candidate(nil), in...)
	// simple insertion sort: small N, fully deterministic
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && less(out[j], out[j-1], prec); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func less(a, b Candidate, prec map[string]int) bool {
	if a.Confidence != b.Confidence {
		return a.Confidence > b.Confidence
	}
	if prec[a.Scheme] != prec[b.Scheme] {
		return prec[a.Scheme] < prec[b.Scheme]
	}
	return a.Value < b.Value
}
