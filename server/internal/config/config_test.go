package config

import (
	"testing"
	"time"
)

var configKeys = []string{
	"HELIX_ADDRESS", "HELIX_TLS_CERT", "HELIX_TLS_KEY", "HELIX_ADMIN_TOKEN",
	"HELIX_OPDS_USERNAME", "HELIX_OPDS_PASSWORD", "HELIX_EXTERNAL_URL",
	"HELIX_POSTGRES_DSN", "HELIX_COVER_CACHE", "HELIX_REDIS_URL",
	"HELIX_KOSYNC_REGISTRATION", "HELIX_AI_MEMORY", "HELIX_EMBEDDING_DIMENSIONS",
	"HELIX_HTTP3", "HELIX_ACCESS_LOG", "HELIX_BROTLI_LEVEL", "HELIX_ALLOW_INSECURE_ADMIN",
	"HELIX_CATALOG_POOL_MAX_OPEN_CONNS", "HELIX_CATALOG_POOL_MAX_IDLE_CONNS",
	"HELIX_CATALOG_POOL_LIFETIME_SECONDS",
	"HELIX_AI_MEMORY_POOL_MAX_OPEN_CONNS", "HELIX_AI_MEMORY_POOL_MAX_IDLE_CONNS",
	"HELIX_AI_MEMORY_POOL_LIFETIME_SECONDS",
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

func TestPoolConfiguration(t *testing.T) {
	t.Run("defaults match the historical hardcoded limits", func(t *testing.T) {
		cleanEnvironment(t)
		c, err := FromEnv()
		if err != nil {
			t.Fatal(err)
		}
		if want := (Pool{MaxOpenConns: 20, MaxIdleConns: 5, ConnMaxLifetime: 30 * time.Minute}); c.CatalogPool != want {
			t.Fatalf("catalog pool = %+v, want %+v", c.CatalogPool, want)
		}
		if want := (Pool{MaxOpenConns: 10, MaxIdleConns: 3, ConnMaxLifetime: 30 * time.Minute}); c.AIMemoryPool != want {
			t.Fatalf("AI memory pool = %+v, want %+v", c.AIMemoryPool, want)
		}
	})
	t.Run("catalog override", func(t *testing.T) {
		cleanEnvironment(t)
		t.Setenv("HELIX_CATALOG_POOL_MAX_OPEN_CONNS", "64")
		t.Setenv("HELIX_CATALOG_POOL_MAX_IDLE_CONNS", "16")
		t.Setenv("HELIX_CATALOG_POOL_LIFETIME_SECONDS", "900")
		c, err := FromEnv()
		if err != nil {
			t.Fatal(err)
		}
		want := Pool{MaxOpenConns: 64, MaxIdleConns: 16, ConnMaxLifetime: 15 * time.Minute}
		if c.CatalogPool != want {
			t.Fatalf("catalog pool = %+v, want %+v", c.CatalogPool, want)
		}
		if c.AIMemoryPool.MaxOpenConns != 10 {
			t.Fatalf("AI memory pool changed with catalog override: %+v", c.AIMemoryPool)
		}
	})
	for _, test := range []struct{ name, key, value string }{
		{"idle above open", "HELIX_CATALOG_POOL_MAX_IDLE_CONNS", "21"},
		{"zero max open", "HELIX_CATALOG_POOL_MAX_OPEN_CONNS", "0"},
		{"negative idle", "HELIX_AI_MEMORY_POOL_MAX_IDLE_CONNS", "-1"},
		{"zero lifetime", "HELIX_AI_MEMORY_POOL_LIFETIME_SECONDS", "0"},
		{"non-integer max open", "HELIX_CATALOG_POOL_MAX_OPEN_CONNS", "many"},
		{"non-integer lifetime", "HELIX_AI_MEMORY_POOL_LIFETIME_SECONDS", "30m"},
	} {
		t.Run(test.name+" rejected", func(t *testing.T) {
			cleanEnvironment(t)
			t.Setenv(test.key, test.value)
			if _, err := FromEnv(); err == nil {
				t.Fatalf("%s=%s accepted", test.key, test.value)
			}
		})
	}
}

func TestAccessLogFlag(t *testing.T) {
	t.Setenv("HELIX_ACCESS_LOG", "true")
	c, err := FromEnv()
	if err != nil || !c.AccessLog {
		t.Fatalf("HELIX_ACCESS_LOG=true: %+v %v", c.AccessLog, err)
	}
	t.Setenv("HELIX_ACCESS_LOG", "")
	if c, _ = FromEnv(); c.AccessLog {
		t.Fatal("access log must default off")
	}
}
