package kosync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// inContract is the independent oracle of PutProgress's input contract
// (RULES.md table K): document 1..255 bytes, progress 1..255, device 1..255,
// device_id 0..255, percentage finite and within [0,1].
func inContract(p Progress) bool {
	return p.Document != "" && len(p.Document) <= 255 &&
		p.Progress != "" && len(p.Progress) <= 255 &&
		p.Device != "" && len(p.Device) <= 255 && len(p.DeviceID) <= 255 &&
		!math.IsNaN(p.Percentage) && !math.IsInf(p.Percentage, 0) && p.Percentage >= 0 && p.Percentage <= 1
}

// FuzzPutProgressBody (HRD-070): arbitrary bytes decoded exactly as the
// handler decodes a KOReader progress body, then passed to the service.
// Acceptance must equal the independent contract model in BOTH directions
// (no over-acceptance, no over-rejection), and an accepted record must be
// in-contract with a server-set timestamp.
func FuzzPutProgressBody(f *testing.F) {
	f.Add([]byte(`{"document":"book-hash","progress":"/body/DocFragment[5]","percentage":0.47,"device":"Kobo","device_id":"device-a","timestamp":7}`))
	f.Add([]byte(`{"document":"d","progress":"p","percentage":2,"device":"d"}`))
	f.Add([]byte(`{"document":"","progress":"x","percentage":2,"device":"d"}`))
	f.Add([]byte(`{"document":"d","progress":"p","percentage":1e999,"device":"d"}`))
	f.Add([]byte(`{"document":"d","progress":"p","percentage":-0,"device":"d","timestamp":-9223372036854775808}`))
	// One boundary seed per rule; each uses its own document id so the in-memory
	// store's "stored winner" semantics cannot shadow the validator under test.
	f.Add([]byte(`{"document":"neg-pct","progress":"p","percentage":-0.5,"device":"d"}`))
	f.Add([]byte(`{"document":"neg-eps","progress":"p","percentage":-0.0000001,"device":"d"}`))
	f.Add([]byte(`{"document":"pct-1","progress":"p","percentage":1,"device":"d"}`))
	f.Add([]byte(`{"document":"pct-1eps","progress":"p","percentage":1.0000001,"device":"d"}`))
	f.Add([]byte(`{"document":"pct-1ulp","progress":"p","percentage":1.0000000000000002,"device":"d"}`))
	f.Add([]byte(`{"document":"pct-0","progress":"p","percentage":0,"device":"d"}`))
	f.Add([]byte(`{"document":"","progress":"p","percentage":0.5,"device":"d"}`))
	f.Add([]byte(`{"document":"no-progress","progress":"","percentage":0.5,"device":"d"}`))
	f.Add([]byte(`{"document":"no-device","progress":"p","percentage":0.5,"device":""}`))
	f.Add([]byte(`{"document":"no-device-id-ok","progress":"p","percentage":0.5,"device":"d","device_id":""}`))
	for _, n := range []int{255, 256} {
		long := strings.Repeat("x", n)
		f.Add([]byte(fmt.Sprintf(`{"document":%q,"progress":"p","percentage":0.5,"device":"d"}`, long)))
		f.Add([]byte(fmt.Sprintf(`{"document":"pl%d","progress":%q,"percentage":0.5,"device":"d"}`, n, long)))
		f.Add([]byte(fmt.Sprintf(`{"document":"dv%d","progress":"p","percentage":0.5,"device":%q}`, n, long)))
		f.Add([]byte(fmt.Sprintf(`{"document":"di%d","progress":"p","percentage":0.5,"device":"d","device_id":%q}`, n, long)))
	}
	f.Add([]byte(`[]`))
	f.Add([]byte(``))
	start := time.Now().Unix()            // every stored timestamp must be server-set at or after this
	svc := NewService(NewMemory(), false) // built once: NewService hashes a bcrypt dummy
	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > 16384 {
			t.Skip("handler caps bodies at 16384")
		}
		var p Progress
		if json.Unmarshal(body, &p) != nil {
			return
		}
		got, err := svc.PutProgress(context.Background(), "reader", p)
		if err != nil {
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("unexpected error class: %v", err)
			}
			if inContract(p) {
				t.Fatalf("in-contract input rejected: %+v", p)
			}
			return
		}
		if !inContract(p) {
			t.Fatalf("out-of-contract input accepted: %+v", p)
		}
		if !inContract(got) {
			t.Fatalf("persisted out-of-contract record: in=%+v out=%+v", p, got)
		}
		// The timestamp is server-set: a caller-supplied value never survives. A
		// stale write returns the stored winner, itself server-stamped after start.
		if got.Timestamp < start || got.Timestamp > time.Now().Unix()+1 {
			t.Fatalf("timestamp %d not server-set (caller sent %d, start=%d)", got.Timestamp, p.Timestamp, start)
		}
		again, err := svc.GetProgress(context.Background(), "reader", p.Document)
		if err != nil || again != got {
			t.Fatalf("round trip differs: %+v vs %+v (%v)", again, got, err)
		}
	})
}

// TestHRD070PutProgressUnreachableByJSON covers the rules the JSON-fed fuzz
// target cannot reach: NaN and +/-Inf percentages cannot be written in JSON.
func TestHRD070PutProgressUnreachableByJSON(t *testing.T) {
	svc := NewService(NewMemory(), false)
	for name, pct := range map[string]float64{"NaN": math.NaN(), "+Inf": math.Inf(1), "-Inf": math.Inf(-1)} {
		_, err := svc.PutProgress(context.Background(), "reader", Progress{Document: "d-" + name, Progress: "p", Device: "d", Percentage: pct})
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s percentage not rejected: %v", name, err)
		}
	}
}

// TestHRD070GetProgressBoundary: document 1..255 bytes.
func TestHRD070GetProgressBoundary(t *testing.T) {
	svc := NewService(NewMemory(), false)
	for n, wantInvalid := range map[int]bool{0: true, 1: false, 255: false, 256: true} {
		_, err := svc.GetProgress(context.Background(), "reader", strings.Repeat("x", n))
		if got := errors.Is(err, ErrInvalidInput); got != wantInvalid {
			t.Errorf("document length %d: invalid=%v want %v (%v)", n, got, wantInvalid, err)
		}
	}
}

// TestHRD070RegisterBoundaries covers the accept side of Register (the fuzz
// target skips valid inputs to avoid a bcrypt hash per iteration) and the
// registration switch.
func TestHRD070RegisterBoundaries(t *testing.T) {
	off := NewService(NewMemory(), false)
	if err := off.Register(context.Background(), "reader", "0123456789abcdef"); !errors.Is(err, ErrRegistrationDisabled) {
		t.Fatalf("registration off: %v", err)
	}
	on := NewService(NewMemory(), true)
	pw := func(n int) string { return strings.Repeat("p", n) }
	cases := []struct {
		user, pass string
		ok         bool
	}{
		{"abc", pw(8), true}, {"A_b-9", pw(8), true}, {strings.Repeat("u", 32), pw(72), true},
		{"ab", pw(8), false}, {strings.Repeat("u", 33), pw(8), false},
		{"abd", pw(7), false}, {"abe", pw(73), false},
	}
	for _, c := range cases {
		err := on.Register(context.Background(), c.user, c.pass)
		if c.ok && err != nil || !c.ok && !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Register(%d-byte user, %d-byte pass): err=%v want ok=%v", len(c.user), len(c.pass), err, c.ok)
		}
	}
}

// FuzzRegisterInput (HRD-070): Register acceptance must match an independent
// oracle of the documented contract (3-32 [A-Za-z0-9_-] user, 8-72 byte pw).
func FuzzRegisterInput(f *testing.F) {
	f.Add("reader", "0123456789abcdef")
	f.Add("a", "short")
	f.Add("ab", "0123456789abcdef")                                // username 2 bytes: below minimum
	f.Add("abc", "0123456")                                        // password 7 bytes: below minimum
	f.Add("abcdefghijklmnopqrstuvwxyz0123456", "0123456789abcdef") // username 33 bytes
	f.Add("a", "0123456789abcdef")
	f.Add("this_username_is_definitely_longer_than_32_chars", "0123456789abcdef")
	f.Add("bad name", "0123456789abcdef")
	f.Add("ok_name-1", string(make([]byte, 73)))
	svc := NewService(NewMemory(), true)
	f.Fuzz(func(t *testing.T, user, pass string) {
		if len(user) > 512 || len(pass) > 512 {
			t.Skip("bound input")
		}
		valid := len(user) >= 3 && len(user) <= 32 && len(pass) >= 8 && len(pass) <= 72
		for i := 0; valid && i < len(user); i++ {
			c := user[i]
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
				valid = false
			}
		}
		if valid {
			t.Skip("valid inputs cost a bcrypt hash; covered by unit tests")
		}
		err := svc.Register(context.Background(), user, pass)
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("out-of-contract registration not rejected: user=%q pass-len=%d err=%v", user, len(pass), err)
		}
	})
}
