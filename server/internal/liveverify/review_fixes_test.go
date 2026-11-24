package liveverify

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Gate-logic tests (11.4.289(C)(2)) for the review fixes I-1 and M-1..M-10. They prove the
// verifier's own decisions; none of them is evidence that a deployed server works.

// identRunner answers the identity probes with canned text ONLY to test how the
// verifier assembles and polices the identity block.
type identRunner struct {
	out      map[string]string // "name arg0" prefix -> output
	failName string
	envs     [][]string
}

func (r *identRunner) Run([]string, time.Duration, string, string, ...string) error { return nil }
func (r *identRunner) Output(_ time.Duration, name string, args ...string) (string, error) {
	return r.OutputEnv(nil, 0, name, args...)
}
func (r *identRunner) OutputEnv(env []string, _ time.Duration, name string, args ...string) (string, error) {
	r.envs = append(r.envs, env)
	if name == r.failName {
		return "", errors.New(name + ": exit status 1")
	}
	key := name + " " + strings.Join(args, " ")
	for k, v := range r.out {
		if strings.Contains(key, k) {
			return v, nil
		}
	}
	return "", nil
}
func (r *identRunner) LookPath(string) bool { return true }

func identFixture(t *testing.T) (IdentityConfig, *identRunner, string) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "helix-server-live")
	if err := os.WriteFile(bin, []byte("binary-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("binary-bytes"))
	run := &identRunner{out: map[string]string{
		"rev-parse HEAD":      "0123456789abcdef0123456789abcdef01234567\n",
		"status --porcelain":  " M a.go\n",
		"SHOW server_version": "16.4\n",
		"INFO server":         "# Server\r\nredis_version:7.4.1\r\nrun_id:abc123def456\r\ntcp_port:6379\r\n",
	}}
	cfg := IdentityConfig{
		Runner: run, RepoRoot: dir, Binaries: map[string]string{"helix-server-live": bin},
		TestDSN: "postgres://alice:s3cretpw@db.example:5433/helix_test?sslmode=disable", RedisAddr: "127.0.0.1:6379", User: "tester",
	}
	return cfg, run, hex.EncodeToString(sum[:])
}

func TestIdentityBlockRecordsRealTargetFields(t *testing.T) {
	cfg, _, sum := identFixture(t)
	id := CollectIdentity(cfg)
	if m := id.Missing(); len(m) != 0 {
		t.Fatalf("unexpected missing identity fields: %v", m)
	}
	raw, _ := json.Marshal(id)
	text := string(raw)
	for _, want := range []string{sum, "0123456789abcdef0123456789abcdef01234567", `"dirty":true`, "16.4", "db.example:5433", "7.4.1", "abc123def456", "127.0.0.1:6379"} {
		if !strings.Contains(text, want) {
			t.Fatalf("identity lacks %q: %s", want, text)
		}
	}
	for _, secret := range []string{"s3cretpw", "alice", "PGPASSWORD", "password"} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(secret)) {
			t.Fatalf("identity leaks %q: %s", secret, text)
		}
	}
}

func TestIdentityMissingWhenAProbeFails(t *testing.T) {
	for _, c := range []struct{ fail, want string }{{"psql", "postgresql"}, {"redis-cli", "redis"}, {"git", "git"}} {
		cfg, run, _ := identFixture(t)
		run.failName = c.fail
		id := CollectIdentity(cfg)
		got := strings.Join(id.Missing(), ",")
		if !strings.Contains(got, c.want) {
			t.Fatalf("probe %s failing must be reported missing (%s), got %q", c.fail, c.want, got)
		}
	}
	cfg, _, _ := identFixture(t)
	cfg.Binaries["ghost"] = filepath.Join(t.TempDir(), "nope")
	if got := strings.Join(CollectIdentity(cfg).Missing(), ","); !strings.Contains(got, "ghost") {
		t.Fatalf("unreadable binary must be missing, got %q", got)
	}
}

func TestResultsCarryIdentityAndNeverNull(t *testing.T) {
	cfg, _, _ := identFixture(t)
	id := CollectIdentity(cfg)
	rec := &Recorder{}
	rec.Require(true, "x")
	res := rec.Results(false)
	res.Identity = &id
	dir := t.TempDir()
	if err := WriteResults(dir, res); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "results.json"))
	if !strings.Contains(string(raw), `"identity"`) || !strings.Contains(string(raw), "run_id") {
		t.Fatalf("results.json lacks identity block: %s", raw)
	}
	var back map[string]any
	if json.Unmarshal(raw, &back) != nil || back["identity"] == nil {
		t.Fatal("identity not round-tripped")
	}
}

type fakeDaemonsRunner struct {
	calls   []string
	missing map[string]bool
}

func (f *fakeDaemonsRunner) Run(_ []string, _ time.Duration, _ string, name string, args ...string) error {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	return nil
}
func (f *fakeDaemonsRunner) LookPath(n string) bool { return !f.missing[n] }

func TestRestartRefusesBeforeShuttingDownWhenItCannotRestart(t *testing.T) {
	for _, c := range []struct {
		name    string
		missing map[string]bool
		env     string
	}{
		{"no-redis-server", map[string]bool{"redis-server": true}, ""},
		{"no-pg-ctlcluster", map[string]bool{"pg_ctlcluster": true}, ""},
		{"external-redis", nil, "127.0.0.1:6379"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("HELIX_TEST_REDIS_ADDR", c.env)
			f := &fakeDaemonsRunner{missing: c.missing}
			d := ExecDaemons{R: f, RedisHost: "127.0.0.1", RedisPort: "6379", ScratchDir: t.TempDir()}
			if err := d.Restart(); err == nil {
				t.Fatal("Restart must refuse")
			}
			for _, call := range f.calls {
				if strings.Contains(call, "SHUTDOWN") || strings.Contains(call, "SAVE") || strings.Contains(call, "restart") {
					t.Fatalf("a destructive step ran before the refusal: %v", f.calls)
				}
			}
		})
	}
}

func TestRedisRestorationSkipIsHonest(t *testing.T) {
	rec := &Recorder{}
	if failed := restoreRedis(rec, stubDaemons{err: ErrRedisNotInstalled}); failed {
		t.Fatal("a missing redis-server is a SKIP, not a failure of the restoration step")
	}
	cs := rec.Checks()
	if len(cs) != 1 || cs[0].Status != "SKIP" || !strings.Contains(cs[0].Name, "NOT RUN") {
		t.Fatalf("want one SKIP, got %v", cs)
	}
	res := rec.Results(false)
	if res.Pass != 0 || res.Fail != 0 {
		t.Fatalf("a SKIP must not count as pass/fail: %+v", res)
	}
	rec2 := &Recorder{}
	if !restoreRedis(rec2, stubDaemons{err: errors.New("boom")}) || rec2.Checks()[0].Status != "FAIL" {
		t.Fatal("any other error stays a FAIL")
	}
}

type stubDaemons struct{ err error }

func (s stubDaemons) Restart() error    { return nil }
func (s stubDaemons) RedisReady() bool  { return true }
func (s stubDaemons) StartRedis() error { return s.err }

func TestCheckRemovableFollowsProtectedPathSemantics(t *testing.T) {
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		"", "/", "/home", "/usr", "/usr/lib/x/y", "/etc", "/var", "/tmp", filepath.Dir(home), home,
		filepath.Join(home, "Projects"), filepath.Join(home, ".ssh"), filepath.Join(home, ".claude-claude4"),
		filepath.Join(home, "scripts"), home + "/../" + filepath.Base(home), "relative/dir", "/home/someone",
	} {
		if checkRemovable(p) == nil {
			t.Errorf("checkRemovable(%q) must refuse", p)
		}
	}
	for _, p := range []string{
		filepath.Join(home, "Projects", "helix_reader", ".helix-verify", "backup-restore"),
		filepath.Join(t.TempDir(), "backup-restore"),
	} {
		if err := checkRemovable(p); err != nil {
			t.Errorf("checkRemovable(%q) must allow: %v", p, err)
		}
	}
	// a symlink parent that resolves into a protected dir must be refused
	link := filepath.Join(t.TempDir(), "lnk")
	if err := os.Symlink("/usr", link); err == nil {
		if checkRemovable(filepath.Join(link, "x")) == nil {
			t.Error("symlinked parent into /usr must be refused")
		}
	}
}

func TestExecRunnerErrorNamesTheCommand(t *testing.T) {
	err := ExecRunner{}.Run(nil, 5*time.Second, "", "false")
	if err == nil || !strings.Contains(err.Error(), "false: exit status 1") {
		t.Fatalf("error must name the command: %v", err)
	}
}

func writeTermIgnoringServer(t *testing.T) (bin, pidfile string) {
	dir := t.TempDir()
	pidfile = filepath.Join(dir, "pid")
	bin = filepath.Join(dir, "srv.sh")
	script := "#!/bin/sh\ntrap '' TERM\necho $$ > \"$PIDFILE\"\nwhile :; do sleep 1; done\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, pidfile
}

func alive(t *testing.T, pidfile string) bool {
	raw, err := os.ReadFile(pidfile)
	if err != nil {
		t.Fatalf("pidfile: %v", err)
	}
	var pid int
	if _, err := fmt.Sscan(strings.TrimSpace(string(raw)), &pid); err != nil {
		t.Fatal(err)
	}
	return syscall.Kill(pid, 0) == nil
}

func TestReadinessTimeoutReapsTheServerWithSigkill(t *testing.T) {
	bin, pidfile := writeTermIgnoringServer(t)
	l := &ExecLauncher{Binary: bin, Env: append(os.Environ(), "PIDFILE="+pidfile), Base: "http://127.0.0.1:1", Log: os.Stderr,
		ReadyAttempts: 3, Ready: func(string) bool { return false }, KillGrace: 300 * time.Millisecond}
	if _, _, err := l.Start(); err == nil {
		t.Fatal("expected readiness timeout")
	}
	if alive(t, pidfile) {
		t.Fatal("wedged server left running after the readiness timeout")
	}
}

func TestStopEscalatesToSigkill(t *testing.T) {
	bin, pidfile := writeTermIgnoringServer(t)
	l := &ExecLauncher{Binary: bin, Env: append(os.Environ(), "PIDFILE="+pidfile), Log: os.Stderr}
	p, err := l.StartNoWait()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	err = p.(*execProcess).StopWithin(300 * time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "SIGKILL") {
		t.Fatalf("a server that ignores SIGTERM is an honest failure that names the escalation: %v", err)
	}
	if alive(t, pidfile) {
		t.Fatal("server survived the escalation")
	}
}

func TestEnsureCertRegeneratesWhenExpiredOrKeyMissing(t *testing.T) {
	dir := t.TempDir()
	if err := writeCert(dir, time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := EnsureCert(dir); err != nil {
		t.Fatal(err)
	}
	if notAfter(t, dir).Before(time.Now().Add(time.Hour)) {
		t.Fatal("expired certificate was reused")
	}
	if err := os.Remove(filepath.Join(dir, "key.pem")); err != nil {
		t.Fatal(err)
	}
	before := notAfter(t, dir)
	time.Sleep(1100 * time.Millisecond)
	if err := EnsureCert(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "key.pem")); err != nil {
		t.Fatal("key.pem not regenerated")
	}
	if !notAfter(t, dir).After(before) {
		t.Fatal("certificate with a missing key was reused")
	}
	// a fresh valid pair is reused
	again := notAfter(t, dir)
	if err := EnsureCert(dir); err != nil || !notAfter(t, dir).Equal(again) {
		t.Fatal("a valid pair must be reused")
	}
}

func TestPSEClientNullStdoutIsAFailureNotNull(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "pse.sh")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho null\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rec := &Recorder{}
	v := &liveServer{cfg: LiveServerConfig{PSECheck: bin, EvidenceDir: dir}, rec: rec, base: "http://127.0.0.1:1"}
	ok := rec.Guard(func() { v.runPSEClient("pse", "pse.json") })
	raw, _ := os.ReadFile(filepath.Join(dir, "pse.json"))
	if ok {
		t.Fatal("a checker that printed null must FAIL")
	}
	if strings.TrimSpace(string(raw)) == "null" || !strings.Contains(string(raw), `"status": "FAIL"`) {
		t.Fatalf("evidence must never be null and must carry a FAIL payload: %s", raw)
	}
}

func notAfter(t *testing.T, dir string) time.Time {
	raw, err := os.ReadFile(filepath.Join(dir, "cert.pem"))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := pem.Decode(raw)
	c, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return c.NotAfter
}

func TestCheckRemovableRefusesAncestorOfAnExtraHome(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "a", "fakehome")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SAFE_RM_EXTRA_HOMES", home)
	for _, p := range []string{filepath.Join(base, "a"), home, filepath.Join(home, ".ssh")} {
		if checkRemovable(p) == nil {
			t.Errorf("checkRemovable(%q) must refuse", p)
		}
	}
	if err := checkRemovable(filepath.Join(home, "work", "backup-restore")); err != nil {
		t.Errorf("a deep child of a home must be allowed: %v", err)
	}
}

func TestAttachIdentityRewritesResultsAndFailsWhenUnresolved(t *testing.T) {
	cfg, run, _ := identFixture(t)
	dir := t.TempDir()
	rec := &Recorder{}
	rec.Require(true, "x")
	res := rec.Results(false)
	got := AttachIdentity(dir, res, CollectIdentity(cfg))
	raw, _ := os.ReadFile(filepath.Join(dir, "results.json"))
	if got.Status != "PASS" || !strings.Contains(string(raw), `"identity"`) {
		t.Fatalf("complete identity must keep PASS and be written: %s %s", got.Status, raw)
	}
	run.failName = "redis-cli"
	got = AttachIdentity(dir, res, CollectIdentity(cfg))
	if got.Status != "FAIL" || got.Fail != 1 || got.ExitCode() != 1 {
		t.Fatalf("unresolved identity means not proven live: %+v", got)
	}
	raw, _ = os.ReadFile(filepath.Join(dir, "results.json"))
	if !strings.Contains(string(raw), "identity unresolved") {
		t.Fatalf("results.json must name the gap: %s", raw)
	}
}

func TestIdentityWithoutDatastoresOmitsThem(t *testing.T) {
	cfg, _, _ := identFixture(t)
	cfg.NoDatastores = true
	id := CollectIdentity(cfg)
	raw, _ := json.Marshal(id)
	if len(id.Missing()) != 0 || strings.Contains(string(raw), "postgresql") || strings.Contains(string(raw), "redis") {
		t.Fatalf("transport identity must not claim datastores: %s %v", raw, id.Missing())
	}
}

func TestExecRunnerErrorCarriesStderrTail(t *testing.T) {
	err := ExecRunner{}.Run(nil, 5*time.Second, "", "sh", "-c", "echo transaction_timeout unknown >&2; exit 3")
	if err == nil || !strings.Contains(err.Error(), "sh: exit status 3") || !strings.Contains(err.Error(), "transaction_timeout unknown") {
		t.Fatalf("a failed command must say why: %v", err)
	}
}
