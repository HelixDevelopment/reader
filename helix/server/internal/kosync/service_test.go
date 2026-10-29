package kosync

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestAccountValidationAuthenticationAndProgress(t *testing.T) {
	store := NewMemory()
	s := NewService(store, true)
	fixed := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return fixed }
	ctx := context.Background()
	if err := s.Register(ctx, "reader", "0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(ctx, "reader", "0123456789abcdef0123456789abcdef"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("duplicate registration: %v", err)
	}
	if store.users["reader"] == "0123456789abcdef0123456789abcdef" {
		t.Fatal("password digest stored without hashing")
	}
	if err := s.Authenticate(ctx, "reader", "wrong-password"); !errors.Is(err, ErrInvalidAuth) {
		t.Fatalf("wrong password accepted: %v", err)
	}
	if err := s.Authenticate(ctx, "missing", "0123456789abcdef0123456789abcdef"); !errors.Is(err, ErrInvalidAuth) {
		t.Fatalf("unknown user accepted: %v", err)
	}
	if err := s.Authenticate(ctx, "reader", "0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	incoming := Progress{Document: "book-hash", Progress: "/body/DocFragment[5]", Percentage: .47, Device: "Kobo", DeviceID: "device-a", Timestamp: 1}
	stored, err := s.PutProgress(ctx, "reader", incoming)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Timestamp != fixed.Unix() {
		t.Fatalf("client timestamp trusted: %d", stored.Timestamp)
	}
	got, err := s.GetProgress(ctx, "reader", "book-hash")
	if err != nil || got != stored {
		t.Fatalf("progress round trip: %+v %v", got, err)
	}
	if _, err := s.GetProgress(ctx, "other", "book-hash"); !errors.Is(err, ErrProgressNotFound) {
		t.Fatalf("cross-user progress leak: %v", err)
	}
}

func TestStaleProgressCannotRewindReader(t *testing.T) {
	s := NewService(NewMemory(), true)
	now := time.Unix(100, 0)
	s.now = func() time.Time { return now }
	ctx := context.Background()
	newer, err := s.PutProgress(ctx, "reader", Progress{Document: "book", Progress: "chapter-8", Percentage: .8, Device: "phone"})
	if err != nil {
		t.Fatal(err)
	}
	now = time.Unix(200, 0)
	stale, err := s.PutProgress(ctx, "reader", Progress{Document: "book", Progress: "chapter-2", Percentage: .2, Device: "offline"})
	if err != nil {
		t.Fatal(err)
	}
	if stale.Percentage != .8 || stale.Progress != "chapter-8" || stale.Timestamp != newer.Timestamp {
		t.Fatalf("stale write replaced progress: %+v", stale)
	}
}

func TestRegistrationAndProgressRejection(t *testing.T) {
	s := NewService(NewMemory(), false)
	if err := s.Register(context.Background(), "reader", "0123456789abcdef"); !errors.Is(err, ErrRegistrationDisabled) {
		t.Fatalf("registration was not disabled: %v", err)
	}
	s.registration = true
	for _, tc := range [][2]string{{"x", "0123456789abcdef"}, {"bad user", "0123456789abcdef"}, {"reader", "short"}, {"reader", strings.Repeat("x", 73)}} {
		if err := s.Register(context.Background(), tc[0], tc[1]); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("accepted invalid account %q: %v", tc, err)
		}
	}
	for _, p := range []Progress{
		{Progress: "x", Device: "d"},
		{Document: "d", Device: "d"},
		{Document: "d", Progress: "x"},
		{Document: "d", Progress: "x", Device: "d", Percentage: -0.1},
		{Document: "d", Progress: "x", Device: "d", Percentage: 1.1},
		{Document: "d", Progress: "x", Device: "d", Percentage: math.NaN()},
		{Document: "d", Progress: "x", Device: "d", Percentage: math.Inf(1)},
		{Document: strings.Repeat("d", 256), Progress: "x", Device: "d"},
	} {
		if _, err := s.PutProgress(context.Background(), "reader", p); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("accepted invalid progress %+v: %v", p, err)
		}
	}
}
