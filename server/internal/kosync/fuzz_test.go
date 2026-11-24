package kosync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
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
		// The handler binds with gin's ShouldBindJSON, which is
		// json.NewDecoder(body).Decode (first value only; trailing data after it
		// is NOT an error), then runs validators (Progress has no binding tags).
		// This decode mirrors that for the default encoding/json build; the
		// handler itself (body limits, status mapping) is NOT fuzzed here.
		var p Progress
		if json.NewDecoder(bytes.NewReader(body)).Decode(&p) != nil {
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

// TestHRD070Authenticate enumerates every rule of Service.Authenticate
// (RULES.md table K): empty username, empty password, username > 255 bytes,
// password > 72 bytes (security relevant: bcrypt only reads the first 72 bytes,
// so without this rule any suffix after a correct 72-byte password would
// authenticate), unknown user, wrong password, and the accepting boundaries.
// Users are created straight in the store so the rules are exercised on
// accounts Register itself would never create (empty or long names).
func TestHRD070Authenticate(t *testing.T) {
	ctx := context.Background()
	store := NewMemory()
	svc := NewService(store, false)
	hash := func(pw string) string {
		h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		return string(h)
	}
	pw72 := strings.Repeat("p", 72)
	long255, long256 := strings.Repeat("u", 255), strings.Repeat("u", 256)
	for user, pw := range map[string]string{"reader": pw72, "": "secret-for-empty-user", "nopass": "", long255: "pw", long256: "pw"} {
		if err := store.CreateUser(ctx, user, hash(pw)); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name, user, pass string
		ok               bool
	}{
		{"valid 72-byte password", "reader", pw72, true},
		{"73-byte password (72 correct bytes + suffix)", "reader", pw72 + "X", false},
		{"4096-byte password sharing the 72-byte prefix", "reader", pw72 + strings.Repeat("Y", 4024), false},
		{"wrong password", "reader", "wrong", false},
		{"unknown user", "ghost", pw72, false},
		{"empty username (account exists)", "", "secret-for-empty-user", false},
		{"empty password (account exists)", "nopass", "", false},
		{"255-byte username accepted", long255, "pw", true},
		{"256-byte username refused (account exists)", long256, "pw", false},
	}
	for _, c := range cases {
		err := svc.Authenticate(ctx, c.user, c.pass)
		if c.ok && err != nil {
			t.Errorf("%s: rejected: %v", c.name, err)
		}
		if !c.ok && !errors.Is(err, ErrInvalidAuth) {
			t.Errorf("%s: error %v, want ErrInvalidAuth", c.name, err)
		}
	}
}

// TestHRD070PutProgressReturnsStoredWinner: PutProgress returns the record the
// store persisted (service.go: "return the persisted winner so a rejected stale
// write cannot acknowledge a newer timestamp"), not the caller's input. The
// memory store keeps the higher percentage, so a lower one is the stale write.
func TestHRD070PutProgressReturnsStoredWinner(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemory(), false)
	clock := int64(1000)
	svc.now = func() time.Time { return time.Unix(clock, 0) }
	first := Progress{Document: "doc", Progress: "/a", Percentage: 0.9, Device: "kobo", DeviceID: "d1"}
	got, err := svc.PutProgress(ctx, "reader", first)
	if err != nil || got.Timestamp != 1000 || got.Percentage != 0.9 || got.Device != "kobo" || got.DeviceID != "d1" || got.Progress != "/a" {
		t.Fatalf("fresh write: %+v (%v)", got, err)
	}
	clock = 2000
	stale := Progress{Document: "doc", Progress: "/b", Percentage: 0.1, Device: "phone", DeviceID: "d2", Timestamp: 5}
	got, err = svc.PutProgress(ctx, "reader", stale)
	if err != nil {
		t.Fatal(err)
	}
	if got.Percentage != 0.9 || got.Device != "kobo" || got.Progress != "/a" || got.Timestamp != 1000 {
		t.Fatalf("stale write must return the stored winner, got %+v", got)
	}
	again, err := svc.GetProgress(ctx, "reader", "doc")
	if err != nil || again != got {
		t.Fatalf("stored record %+v (%v) != returned %+v", again, err, got)
	}
}

// failingStore makes one store call fail so the service's error pass-through
// branches are exercised.
type failingStore struct {
	*Memory
	failVerifier, failPut bool
}

var errBoom = errors.New("store boom")

func (f failingStore) UserVerifier(ctx context.Context, u string) (string, error) {
	if f.failVerifier {
		return "", errBoom
	}
	return f.Memory.UserVerifier(ctx, u)
}

func (f failingStore) PutProgress(ctx context.Context, u string, p Progress) error {
	if f.failPut {
		return errBoom
	}
	return f.Memory.PutProgress(ctx, u, p)
}

// TestHRD070StoreErrorsPropagate: a store failure other than "user not found"
// is returned as is (never disguised as ErrInvalidAuth, never swallowed), and
// a failed write is not reported as stored.
func TestHRD070StoreErrorsPropagate(t *testing.T) {
	ctx := context.Background()
	svc := NewService(failingStore{Memory: NewMemory(), failVerifier: true}, false)
	if err := svc.Authenticate(ctx, "reader", "password1"); !errors.Is(err, errBoom) || errors.Is(err, ErrInvalidAuth) {
		t.Errorf("verifier failure: %v, want the store error", err)
	}
	svc = NewService(failingStore{Memory: NewMemory(), failPut: true}, false)
	if _, err := svc.PutProgress(ctx, "reader", Progress{Document: "d", Progress: "p", Device: "d", Percentage: 0.5}); !errors.Is(err, errBoom) {
		t.Errorf("write failure: %v, want the store error", err)
	}
	if _, err := svc.GetProgress(ctx, "reader", "never-stored"); !errors.Is(err, ErrProgressNotFound) {
		t.Errorf("unstored document: %v, want ErrProgressNotFound", err)
	}
}

// TestHRD070ErrorClasses pins WHICH error each rejection rule of Register and
// GetProgress carries (the HTTP layer maps them to different statuses).
func TestHRD070ErrorClasses(t *testing.T) {
	ctx := context.Background()
	off := NewService(NewMemory(), false)
	if err := off.Register(ctx, "reader", "password1"); !errors.Is(err, ErrRegistrationDisabled) {
		t.Errorf("registration off: %v", err)
	}
	on := NewService(NewMemory(), true)
	for _, c := range []struct{ u, p string }{{"ab", "password1"}, {"has space", "password1"}, {"reader", "short"}, {"reader", strings.Repeat("p", 73)}} {
		if err := on.Register(ctx, c.u, c.p); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Register(%q,%d bytes): %v, want ErrInvalidInput", c.u, len(c.p), err)
		}
	}
	if err := on.Register(ctx, "reader", "password1"); err != nil {
		t.Fatalf("valid register: %v", err)
	}
	if err := on.Register(ctx, "reader", "password1"); err == nil {
		t.Errorf("duplicate register accepted")
	}
	for _, d := range []string{"", strings.Repeat("d", 256)} {
		if _, err := on.GetProgress(ctx, "reader", d); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("GetProgress(%d bytes): %v, want ErrInvalidInput", len(d), err)
		}
	}
	bad := Progress{Progress: "p", Device: "d", Percentage: 0.5}
	if _, err := on.PutProgress(ctx, "reader", bad); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("empty document: %v, want ErrInvalidInput", err)
	}
}
