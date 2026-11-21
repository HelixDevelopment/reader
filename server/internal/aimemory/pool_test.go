package aimemory

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// The catalog's pool limits became deployment-configurable in the same change as
// these, and the server wires both from the same environment. The AI-memory pool
// had no test of any kind until this file, so its validation rules are asserted
// where a deleted check cannot hide.

func TestPoolSettingsValidation(t *testing.T) {
	for _, test := range []struct {
		name             string
		maxOpen, maxIdle int
		lifetime         time.Duration
		want             string
	}{
		{"defaults are valid", 10, 3, 30 * time.Minute, ""},
		{"zero idle is valid", 4, 0, time.Minute, ""},
		{"zero max open", 0, 0, time.Minute, "MaxOpenConns"},
		{"negative max open", -1, 0, time.Minute, "MaxOpenConns"},
		{"idle above open", 4, 5, time.Minute, "MaxIdleConns"},
		{"negative idle", 4, -1, time.Minute, "MaxIdleConns"},
		{"zero lifetime", 4, 1, 0, "ConnMaxLifetime"},
		{"negative lifetime", 4, 1, -time.Second, "ConnMaxLifetime"},
	} {
		t.Run(test.name, func(t *testing.T) {
			settings := poolSettings{maxOpenConns: test.maxOpen, maxIdleConns: test.maxIdle, connMaxLifetime: test.lifetime}
			err := settings.validate()
			if test.want == "" {
				if err != nil {
					t.Fatalf("valid settings rejected: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("settings %+v accepted", settings)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error %q does not name %s", err, test.want)
			}
		})
	}
}

// TestNewPostgresRejectsPoolBeforeDialing asserts which error arrives, because
// construction fails on an unreachable DSN too and a bare "got an error"
// assertion is satisfied by the dial.
func TestNewPostgresRejectsPoolBeforeDialing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, test := range []struct {
		name             string
		want             string
		dimensions       int
		maxOpen, maxIdle int
		lifetime         time.Duration
	}{
		{"idle above open", "MaxIdleConns", 3, 4, 9, time.Minute},
		{"zero max open", "MaxOpenConns", 3, 0, 0, time.Minute},
		{"zero lifetime", "ConnMaxLifetime", 3, 4, 1, 0},
		{"zero dimensions", "dimensions", 0, 4, 1, time.Minute},
		{"dimensions above the ceiling", "dimensions", 2001, 4, 1, time.Minute},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory, err := NewPostgres(ctx, "postgres://unreachable.invalid:5432/none", test.dimensions, WithPool(test.maxOpen, test.maxIdle, test.lifetime))
			if err == nil {
				memory.Close()
				t.Fatal("invalid input accepted")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("rejected for the wrong reason: %v (want %s)", err, test.want)
			}
		})
	}
}

func TestWithPoolAppliesLimits(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	memory, err := NewPostgres(ctx, dsn, 3, WithPool(7, 2, 3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	defer memory.Close()
	if got := memory.db.Stats().MaxOpenConnections; got != 7 {
		t.Fatalf("MaxOpenConnections = %d, want 7", got)
	}
	defaulted, err := NewPostgres(ctx, dsn, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer defaulted.Close()
	if got := defaulted.db.Stats().MaxOpenConnections; got != 10 {
		t.Fatalf("default MaxOpenConnections = %d, want 10", got)
	}
}
