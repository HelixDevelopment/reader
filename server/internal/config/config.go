// Package config loads environment-only service configuration.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address             string
	TLSCertFile         string
	TLSKeyFile          string
	AdminToken          string
	OPDSUsername        string
	OPDSPassword        string
	ExternalURL         string
	CatalogDSN          string
	CoverCacheDir       string
	RedisURL            string
	KOSyncRegistration  bool
	AIMemory            bool
	EmbeddingDimensions int
	ShutdownTimeout     time.Duration
	BrotliLevel         int
	HTTP3               bool
	AccessLog           bool
	CatalogPool         Pool
	AIMemoryPool        Pool
}

// Pool bounds one PostgreSQL connection pool. Limits are coordinated with
// server-side capacity (max_connections, worker count, replicas) rather than
// raised blindly.
type Pool struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func FromEnv() (Config, error) {
	c := Config{
		Address: env("HELIX_ADDRESS", "127.0.0.1:8080"), TLSCertFile: os.Getenv("HELIX_TLS_CERT"), TLSKeyFile: os.Getenv("HELIX_TLS_KEY"),
		AdminToken: os.Getenv("HELIX_ADMIN_TOKEN"), OPDSUsername: os.Getenv("HELIX_OPDS_USERNAME"), OPDSPassword: os.Getenv("HELIX_OPDS_PASSWORD"),
		ExternalURL: strings.TrimRight(os.Getenv("HELIX_EXTERNAL_URL"), "/"), CatalogDSN: os.Getenv("HELIX_POSTGRES_DSN"), CoverCacheDir: env("HELIX_COVER_CACHE", "./data/covers"), RedisURL: os.Getenv("HELIX_REDIS_URL"),
		BrotliLevel: 4, ShutdownTimeout: 15 * time.Second, HTTP3: true, EmbeddingDimensions: 1536,
		CatalogPool:  Pool{MaxOpenConns: 20, MaxIdleConns: 5, ConnMaxLifetime: 30 * time.Minute},
		AIMemoryPool: Pool{MaxOpenConns: 10, MaxIdleConns: 3, ConnMaxLifetime: 30 * time.Minute},
	}
	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return c, fmt.Errorf("HELIX_TLS_CERT and HELIX_TLS_KEY must be configured together")
	}
	if (c.OPDSUsername == "") != (c.OPDSPassword == "") {
		return c, fmt.Errorf("OPDS username and password must be configured together")
	}
	c.KOSyncRegistration, _ = strconv.ParseBool(os.Getenv("HELIX_KOSYNC_REGISTRATION"))
	c.AIMemory, _ = strconv.ParseBool(os.Getenv("HELIX_AI_MEMORY"))
	if v := os.Getenv("HELIX_EMBEDDING_DIMENSIONS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 2000 {
			return c, fmt.Errorf("HELIX_EMBEDDING_DIMENSIONS must be between 1 and 2000")
		}
		c.EmbeddingDimensions = n
	}
	if c.AIMemory && c.CatalogDSN == "" {
		return c, fmt.Errorf("HELIX_AI_MEMORY requires HELIX_POSTGRES_DSN")
	}
	if v := os.Getenv("HELIX_HTTP3"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return c, fmt.Errorf("HELIX_HTTP3 must be boolean")
		}
		c.HTTP3 = enabled
	}
	if v := os.Getenv("HELIX_BROTLI_LEVEL"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 11 {
			return c, fmt.Errorf("HELIX_BROTLI_LEVEL must be between 0 and 11")
		}
		c.BrotliLevel = n
	}
	c.AccessLog, _ = strconv.ParseBool(os.Getenv("HELIX_ACCESS_LOG"))
	allowInsecure, _ := strconv.ParseBool(os.Getenv("HELIX_ALLOW_INSECURE_ADMIN"))
	if c.AdminToken == "" && !allowInsecure && !loopback(c.Address) {
		return c, fmt.Errorf("HELIX_ADMIN_TOKEN is required for non-loopback listeners")
	}
	for _, pool := range []struct {
		prefix string
		target *Pool
	}{
		{"HELIX_CATALOG_POOL", &c.CatalogPool},
		{"HELIX_AI_MEMORY_POOL", &c.AIMemoryPool},
	} {
		if err := applyPoolEnv(pool.prefix, pool.target); err != nil {
			return c, err
		}
	}
	return c, nil
}

// applyPoolEnv reads <prefix>_MAX_OPEN_CONNS, <prefix>_MAX_IDLE_CONNS and
// <prefix>_LIFETIME_SECONDS, keeping the current value when unset.
func applyPoolEnv(prefix string, target *Pool) error {
	if v := os.Getenv(prefix + "_MAX_OPEN_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return fmt.Errorf("%s_MAX_OPEN_CONNS must be a positive integer", prefix)
		}
		target.MaxOpenConns = n
	}
	if v := os.Getenv(prefix + "_MAX_IDLE_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return fmt.Errorf("%s_MAX_IDLE_CONNS must be a non-negative integer", prefix)
		}
		target.MaxIdleConns = n
	}
	if v := os.Getenv(prefix + "_LIFETIME_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return fmt.Errorf("%s_LIFETIME_SECONDS must be a positive integer", prefix)
		}
		target.ConnMaxLifetime = time.Duration(n) * time.Second
	}
	if target.MaxIdleConns > target.MaxOpenConns {
		return fmt.Errorf("%s_MAX_IDLE_CONNS %d exceeds %s_MAX_OPEN_CONNS %d", prefix, target.MaxIdleConns, prefix, target.MaxOpenConns)
	}
	return nil
}
func loopback(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
