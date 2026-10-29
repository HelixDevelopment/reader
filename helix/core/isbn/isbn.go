// Package isbn implements deterministic ISBN-10/ISBN-13 validation and
// extraction from raw text (constitution: every claim test-backed).
package isbn

import (
	"regexp"
	"strings"
)

var candidate = regexp.MustCompile(`(?:97[89][- ]?)?[0-9][-0-9 ]{8,16}[0-9Xx]`)

func strip(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' {
			return -1
		}
		return r
	}, s)
}

// Valid10 reports whether s is a valid ISBN-10 (check digit verified).
func Valid10(s string) bool {
	s = strip(s)
	if len(s) != 10 {
		return false
	}
	sum := 0
	for i := 0; i < 10; i++ {
		var d int
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case (c == 'X' || c == 'x') && i == 9:
			d = 10
		default:
			return false
		}
		sum += (10 - i) * d
	}
	return sum%11 == 0
}

// Valid13 reports whether s is a valid ISBN-13 (check digit verified).
func Valid13(s string) bool {
	s = strip(s)
	if len(s) != 13 || (s[:3] != "978" && s[:3] != "979") {
		return false
	}
	sum := 0
	for i := 0; i < 13; i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
		d := int(c - '0')
		if i%2 == 1 {
			d *= 3
		}
		sum += d
	}
	return sum%10 == 0
}

// Canonical returns a normalized ISBN-13 for either a valid ISBN-13 or a
// valid ISBN-10. ISBN-10 values gain the 978 prefix and a recomputed check
// digit, giving identity code one representation for the same publication.
func Canonical(value string) (string, bool) {
	value = strings.ToUpper(strip(value))
	if Valid13(value) {
		return value, true
	}
	if !Valid10(value) {
		return "", false
	}
	base := "978" + value[:9]
	sum := 0
	for i := 0; i < len(base); i++ {
		d := int(base[i] - '0')
		if i%2 == 1 {
			d *= 3
		}
		sum += d
	}
	return base + string(rune('0'+(10-sum%10)%10)), true
}

// Extract returns all valid ISBNs found in text, in order, de-duplicated.
func Extract(text string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(value string) bool {
		if !Valid10(value) && !Valid13(value) {
			return false
		}
		norm := strings.NewReplacer("-", "", " ", "").Replace(strings.ToUpper(value))
		if !seen[norm] {
			seen[norm] = true
			out = append(out, norm)
		}
		return true
	}
	for offset := 0; offset < len(text); {
		location := candidate.FindStringIndex(text[offset:])
		if location == nil {
			break
		}
		start, end := offset+location[0], offset+location[1]
		match := text[start:end]
		if add(match) {
			offset = end
			continue
		}
		// A greedy candidate can span the space between adjacent ISBNs. Try
		// valid prefixes at separator boundaries before rescanning the suffix;
		// otherwise advance one byte so no later candidate is hidden.
		prefixEnd := -1
		for i, r := range match {
			if (r == ' ' || r == '-') && add(match[:i]) {
				prefixEnd = i
				break
			}
		}
		if prefixEnd >= 0 {
			offset = start + prefixEnd + 1
		} else {
			offset = start + 1
		}
	}
	return out
}
