package config

import "testing"

var configKeys = []string{
	"HELIX_ADDRESS", "HELIX_TLS_CERT", "HELIX_TLS_KEY", "HELIX_ADMIN_TOKEN",
	"HELIX_OPDS_USERNAME", "HELIX_OPDS_PASSWORD", "HELIX_EXTERNAL_URL",
	"HELIX_POSTGRES_DSN", "HELIX_COVER_CACHE", "HELIX_REDIS_URL",
	"HELIX_KOSYNC_REGISTRATION", "HELIX_AI_MEMORY", "HELIX_EMBEDDING_DIMENSIONS",
	"HELIX_HTTP3", "HELIX_BROTLI_LEVEL", "HELIX_ALLOW_INSECURE_ADMIN",
}

func cleanEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range configKeys {
		t.Setenv(key, "")
	}
}

func TestLoopbackClassification(t *testing.T) {
	for _, test := range []struct {
		address string
		expect  bool
	}{{"127.0.0.1:8080", true}, {"[::1]:8080", true}, {"localhost:8080", false}, {"0.0.0.0:8080", false}, {"bad", false}} {
		if actual := loopback(test.address); actual != test.expect {
			t.Errorf("loopback(%q)=%v, expected %v", test.address, actual, test.expect)
		}
	}
}

func TestEnvironmentValidation(t *testing.T) {
	t.Run("non-loopback requires admin token", func(t *testing.T) {
		cleanEnvironment(t)
		t.Setenv("HELIX_ADDRESS", "0.0.0.0:8080")
		if _, err := FromEnv(); err == nil {
			t.Fatal("missing admin token accepted")
		}
		t.Setenv("HELIX_ADMIN_TOKEN", "test-token")
		if _, err := FromEnv(); err != nil {
			t.Fatalf("configured admin token rejected: %v", err)
		}
	})
	for _, test := range []struct {
		name  string
		key   string
		value string
	}{
		{"TLS certificate without key", "HELIX_TLS_CERT", "cert.pem"},
		{"OPDS user without password", "HELIX_OPDS_USERNAME", "reader"},
		{"invalid HTTP3", "HELIX_HTTP3", "maybe"},
		{"invalid Brotli", "HELIX_BROTLI_LEVEL", "12"},
		{"invalid dimensions", "HELIX_EMBEDDING_DIMENSIONS", "0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cleanEnvironment(t)
			t.Setenv(test.key, test.value)
			if _, err := FromEnv(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
	t.Run("AI memory requires PostgreSQL", func(t *testing.T) {
		cleanEnvironment(t)
		t.Setenv("HELIX_AI_MEMORY", "true")
		if _, err := FromEnv(); err == nil {
			t.Fatal("AI memory without PostgreSQL accepted")
		}
	})
}
