package liveverify

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Identity is the machine-readable proof of WHAT was verified and WHAT answered
// (constitution 11.4.289(D), 11.4.262): the artifacts under test, the source tree
// they were built from, and the real PostgreSQL and Redis that served the run.
// It never carries a credential: the DSN user and password are used to reach the
// database and are not recorded.
type Identity struct {
	Binaries   map[string]string `json:"binaries_sha256"`
	Git        GitIdentity       `json:"git"`
	PostgreSQL *PGIdentity       `json:"postgresql,omitempty"`
	Redis      *RedisIdentity    `json:"redis,omitempty"`
	Unresolved []string          `json:"unresolved,omitempty"`
}

// GitIdentity names the source tree the binaries were built from.
type GitIdentity struct {
	Head  string `json:"head"`
	Dirty bool   `json:"dirty"`
}

// PGIdentity is the PostgreSQL server that answered (version and endpoint only).
type PGIdentity struct {
	Version  string `json:"version"`
	Endpoint string `json:"endpoint"`
}

// RedisIdentity is the Redis server that answered.
type RedisIdentity struct {
	Version string `json:"version"`
	RunID   string `json:"run_id"`
	Address string `json:"address"`
}

// IdentityConfig says how to read the identity.
type IdentityConfig struct {
	Runner    Runner
	RepoRoot  string
	Binaries  map[string]string // name -> path of each built artifact under test
	TestDSN   string            // HELIX_TEST_POSTGRES_DSN ("" = peer-auth socket default)
	RedisAddr string            // host:port of the Redis that serves the run
	User      string
	// NoDatastores skips the PostgreSQL and Redis probes (the transport verifier
	// uses neither, so its identity must not claim them).
	NoDatastores bool
}

// envOutputRunner is implemented by runners that can run a probe with a custom environment.
type envOutputRunner interface {
	OutputEnv(env []string, timeout time.Duration, name string, args ...string) (string, error)
}

// CollectIdentity reads the identity from the real artifacts and daemons. A field
// that cannot be read is listed in Unresolved (and by Missing); it is never guessed.
func CollectIdentity(c IdentityConfig) Identity {
	id := Identity{Binaries: map[string]string{}}
	miss := func(what string) { id.Unresolved = append(id.Unresolved, what) }

	for name, path := range c.Binaries {
		raw, err := os.ReadFile(path)
		if err != nil {
			miss("binary:" + name)
			continue
		}
		sum := sha256.Sum256(raw)
		id.Binaries[name] = hex.EncodeToString(sum[:])
	}

	run := c.Runner
	head, err := run.Output(15*time.Second, "git", "-C", c.RepoRoot, "rev-parse", "HEAD")
	head = strings.TrimSpace(head)
	st, err2 := run.Output(15*time.Second, "git", "-C", c.RepoRoot, "status", "--porcelain")
	if err != nil || err2 != nil || head == "" {
		miss("git")
	} else {
		id.Git = GitIdentity{Head: head, Dirty: strings.TrimSpace(st) != ""}
	}

	if c.NoDatastores {
		sort.Strings(id.Unresolved)
		return id
	}
	bc := BackupConfig{TestDSN: c.TestDSN, User: c.User}
	bc.defaults()
	env := bc.PGEnvironment("postgres")
	version := ""
	if er, ok := run.(envOutputRunner); ok {
		out, perr := er.OutputEnv(env, 15*time.Second, "psql", "-X", "-A", "-t", "-c", "SHOW server_version")
		if perr == nil {
			version = strings.TrimSpace(out)
		}
	}
	if version == "" {
		miss("postgresql")
	} else {
		id.PostgreSQL = &PGIdentity{Version: version, Endpoint: pgEndpoint(env)}
	}

	h, p := "127.0.0.1", "6379"
	if c.RedisAddr != "" {
		rc := BackupConfig{RedisAddr: c.RedisAddr}
		h, p = rc.redisHostPort()
	}
	out, rerr := run.Output(15*time.Second, "redis-cli", "-h", h, "-p", p, "INFO", "server")
	ver, runID := infoField(out, "redis_version"), infoField(out, "run_id")
	if rerr != nil || ver == "" || runID == "" {
		miss("redis")
	} else {
		id.Redis = &RedisIdentity{Version: ver, RunID: runID, Address: h + ":" + p}
	}
	sort.Strings(id.Unresolved)
	return id
}

// Missing lists the identity fields that could not be resolved; empty means complete.
func (i Identity) Missing() []string { return append([]string(nil), i.Unresolved...) }

func infoField(info, key string) string {
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, key+":"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func pgEndpoint(env []string) string {
	host, port := "", "5432"
	for _, kv := range env {
		switch {
		case strings.HasPrefix(kv, "PGHOST="):
			host = strings.TrimPrefix(kv, "PGHOST=")
		case strings.HasPrefix(kv, "PGPORT="):
			port = strings.TrimPrefix(kv, "PGPORT=")
		}
	}
	if host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s:%s", host, port)
}

// AttachIdentity records the identity in results.json. An identity that cannot be
// resolved means the run is NOT proven live (11.4.289(D)(iv)): it is a FAIL check
// that names the gap, never a silent pass.
func AttachIdentity(dir string, res Results, id Identity) Results {
	res.Identity = &id
	if miss := id.Missing(); len(miss) > 0 {
		res.Checks = append(res.Checks, Check{"identity unresolved: " + strings.Join(miss, ", "), "FAIL"})
		res.Fail++
		res.Status = "FAIL"
	}
	if err := WriteResults(dir, res); err != nil {
		res.Status = "FAIL"
	}
	return res
}
