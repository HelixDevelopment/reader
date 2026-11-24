package httpapi

import "testing"

// RFC 9110 14.1.2: an overlong last-byte-pos clamps to the end; a first-byte-pos
// at or beyond the size (including one too large for int64) is unsatisfiable;
// an overlong suffix-length selects the whole representation.
func TestSingleRangeRFC9110(t *testing.T) {
	const size = 1000
	cases := []struct {
		in         string
		start, len int64
		ok         bool
	}{
		{"bytes=0-99", 0, 100, true},
		{"bytes=0-999", 0, 1000, true},
		{"bytes=0-1000", 0, 1000, true},
		{"bytes=999-1000", 999, 1, true},
		{"bytes=0-99999999999999999999", 0, 1000, true},
		{"bytes=5-99999999999999999999", 5, 995, true},
		{"bytes=0-9223372036854775807", 0, 1000, true},
		{"bytes=500-", 500, 500, true},
		{"bytes=-100", 900, 100, true},
		{"bytes=-99999999999999999999", 0, 1000, true},
		{"bytes=99999999999999999999-", 0, 0, false},
		{"bytes=99999999999999999999-99999999999999999999", 0, 0, false},
		{"bytes=1000-", 0, 0, false},
		{"bytes=9-3", 0, 0, false},
		{"bytes=-0", 0, 0, false},
		{"bytes=a-5", 0, 0, false},
		{"bytes=0-b", 0, 0, false},
		{"bytes=-", 0, 0, false},
		{"bytes=0-5-7", 0, 0, false},
		{"bytes=0-9x", 0, 0, false},
		{"bytes=+1-5", 0, 0, false},
		{"bytes=0-5,7-9", 0, 0, false},
		{"items=0-5", 0, 0, false},
	}
	for _, c := range cases {
		s, l, err := singleRange(c.in, size)
		if (err == nil) != c.ok || (c.ok && (s != c.start || l != c.len)) {
			t.Errorf("%q -> start=%d len=%d err=%v; want ok=%v start=%d len=%d", c.in, s, l, err, c.ok, c.start, c.len)
		}
	}
}
