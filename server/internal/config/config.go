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
}

func FromEnv() (Config, error) {
	c := Config{
		Address: env("HELIX_ADDRESS", "127.0.0.1:8080"), TLSCertFile: os.Getenv("HELIX_TLS_CERT"), TLSKeyFile: os.Getenv("HELIX_TLS_KEY"),
		AdminToken: os.Getenv("HELIX_ADMIN_TOKEN"), OPDSUsername: os.Getenv("HELIX_OPDS_USERNAME"), OPDSPassword: os.Getenv("HELIX_OPDS_PASSWORD"),
		ExternalURL: strings.TrimRight(os.Getenv("HELIX_EXTERNAL_URL"), "/"), CatalogDSN: os.Getenv("HELIX_POSTGRES_DSN"), CoverCacheDir: env("HELIX_COVER_CACHE", "./data/covers"), RedisURL: os.Getenv("HELIX_REDIS_URL"),
		BrotliLevel: 4, ShutdownTimeout: 15 * time.Second, HTTP3: true, EmbeddingDimensions: 1536,
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
	allowInsecure, _ := strconv.ParseBool(os.Getenv("HELIX_ALLOW_INSECURE_ADMIN"))
	if c.AdminToken == "" && !allowInsecure && !loopback(c.Address) {
		return c, fmt.Errorf("HELIX_ADMIN_TOKEN is required for non-loopback listeners")
	}
	return c, nil
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
