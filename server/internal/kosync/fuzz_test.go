package kosync

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"
)

// FuzzPutProgressBody (HRD-070): arbitrary bytes decoded exactly as the
// handler decodes a KOReader progress body, then passed to the service. The
// service must never panic, must reject out-of-contract input with
// ErrInvalidInput, and must only persist/return in-contract records.
func FuzzPutProgressBody(f *testing.F) {
	f.Add([]byte(`{"document":"book-hash","progress":"/body/DocFragment[5]","percentage":0.47,"device":"Kobo","device_id":"device-a","timestamp":7}`))
	f.Add([]byte(`{"document":"d","progress":"p","percentage":2,"device":"d"}`))
	f.Add([]byte(`{"document":"","progress":"x","percentage":2,"device":"d"}`))
	f.Add([]byte(`{"document":"d","progress":"p","percentage":1e999,"device":"d"}`))
	f.Add([]byte(`{"document":"d","progress":"p","percentage":-0,"device":"d","timestamp":-9223372036854775808}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(``))
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
			return
		}
		if got.Document != p.Document || got.Document == "" || len(got.Document) > 255 ||
			len(got.Progress) > 255 || len(got.Device) > 255 || len(got.DeviceID) > 255 ||
			math.IsNaN(got.Percentage) || got.Percentage < 0 || got.Percentage > 1 {
			t.Fatalf("persisted out-of-contract record: in=%+v out=%+v", p, got)
		}
		again, err := svc.GetProgress(context.Background(), "reader", p.Document)
		if err != nil || again != got {
			t.Fatalf("round trip differs: %+v vs %+v (%v)", again, got, err)
		}
	})
}

// FuzzRegisterInput (HRD-070): Register acceptance must match an independent
// oracle of the documented contract (3-32 [A-Za-z0-9_-] user, 8-72 byte pw).
func FuzzRegisterInput(f *testing.F) {
	f.Add("reader", "0123456789abcdef")
	f.Add("a", "short")
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
