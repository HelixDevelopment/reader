package liveverify

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (b *backup) start(port int, db, redisURL, logName string) (string, Process) {
	base, p, err := b.cfg.StartServer(port, db, redisURL, logName)
	must(err)
	b.procs = append(b.procs, p)
	return base, p
}

func (b *backup) stop(p Process) {
	must(p.Stop())
	for i, q := range b.procs {
		if q == p {
			b.procs = append(b.procs[:i], b.procs[i+1:]...)
			break
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func (b *backup) body(library string) {
	r, c := b.rec, b.cfg
	run := c.Runner
	work := c.WorkDir
	b.stopRestoreRedis()
	b.dropRestoreDB()
	redisURL := "redis://" + c.RedisAddr
	base, process := b.start(28080, backupMainDB, redisURL, "source-server.log")
	get := func(path string, o Opts) *Resp { return b.hc.Do("GET", base+path, o) }
	// Deterministic rerun cleanup through the public management surface.
	for _, old := range JList(get("/v1/works", Opts{Query: url.Values{"q": {"backup-restore-book"}, "limit": {"100"}}, Basic: b.basic}).JSON(), "works") {
		b.hc.Do("DELETE", base+"/v1/works/"+JStr(old, "id"), Opts{Header: b.admin}).RaiseForStatus()
	}
	b.hc.Do("DELETE", base+"/v1/ai/memories/backup-restore/"+backupMemoryID, Opts{Header: b.admin})
	b.hc.Do("DELETE", base+"/v1/sync/progress/"+backupProgressUsr+"/"+backupProgressBk, Opts{Header: b.admin})

	created := b.hc.Do("POST", base+"/v1/sources", Opts{Header: b.admin, JSON: map[string]any{"protocol": "local", "root": library}})
	r.Require(created.Status == 201, "source fixture creation")
	scan := b.hc.Do("POST", base+"/v1/scans", Opts{Header: b.admin, JSON: map[string]any{"source_id": JStr(created.JSON(), "id")}})
	r.Require(scan.Status == 202, "source fixture scan accepted")
	jobID := JStr(scan.JSON(), "id")
	var job any
	for i := 0; i < 150; i++ {
		job = b.hc.Do("GET", base+"/v1/scans/"+jobID, Opts{Header: b.admin}).JSON()
		s := jstrOr(job, "state")
		if s == "succeeded" || s == "failed" || s == "canceled" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	files, okFiles := scanFiles(job)
	r.Require(jstrOr(job, "state") == "succeeded" && okFiles && files == 1, "source fixture ingestion")
	works := JList(get("/v1/works", Opts{Query: url.Values{"q": {"backup-restore-book"}}, Basic: b.basic}).JSON(), "works")
	r.Require(len(works) == 1, "source catalog fixture")
	b.workID = JStr(works[0], "id")
	memory := map[string]any{"record": map[string]any{"id": backupMemoryID, "scope": "backup-restore", "kind": "note", "content": "recovery memory", "model": "backup-model"}, "embedding": []any{1, 0, 0}}
	r.Require(b.hc.Do("PUT", base+"/v1/ai/memories", Opts{Header: b.admin, JSON: memory, Timeout: 5 * time.Second}).Status == 200, "source vector fixture")
	progress := map[string]any{"BookID": "ignored", "DeviceID": "backup-device", "Pos": map[string]any{"Chapter": 9, "Block": 2, "Offset": 1}, "Clock": map[string]any{"V": map[string]any{"backup-device": 1}}}
	r.Require(b.hc.Do("PUT", base+"/v1/sync/progress/"+backupProgressUsr+"/"+backupProgressBk, Opts{Header: b.admin, JSON: progress, Timeout: 5 * time.Second}).Status == 200, "source progress fixture")
	b.stop(process)

	dump := filepath.Join(work, "catalog.dump")
	must(run.Run(c.PGEnvironment(backupMainDB), 90*time.Second, "", "pg_dump", "--format=custom", "--file="+dump, "-d", backupMainDB))
	st, err := os.Stat(dump)
	r.Require(err == nil && st.Size() > 0, "PostgreSQL backup created")
	must(run.Run(c.PGEnvironment("postgres"), 30*time.Second, "", "createdb", backupRestoreDB))
	must(run.Run(c.PGEnvironment(backupRestoreDB), 90*time.Second, "", "pg_restore", "--exit-on-error", "--no-owner", "--no-privileges", "--dbname="+backupRestoreDB, dump))
	// pg_restore --exit-on-error exiting 0 is the proof; the verdict is recorded after it ran.
	r.Require(true, "PostgreSQL restore into isolated database")

	must(run.Run(nil, 20*time.Second, "", "redis-cli", append(c.redisCLI(""), "SAVE")...))
	// The snapshot directory is discovered from the running daemon, not assumed.
	rdbDir := c.RedisDataDir
	if rdbDir == "" {
		reply, err := run.Output(15*time.Second, "redis-cli", append(c.redisCLI(""), "CONFIG", "GET", "dir")...)
		must(err)
		lines := strings.Split(strings.TrimSpace(reply), "\n")
		rdbDir = strings.TrimSpace(lines[len(lines)-1])
	}
	sourceRDB := filepath.Join(rdbDir, "dump.rdb")
	restoreRDB := filepath.Join(work, "restore.rdb")
	must(copyFile(sourceRDB, restoreRDB))
	st, err = os.Stat(restoreRDB)
	r.Require(err == nil && st.Size() > 0, "Redis backup created")
	if !run.LookPath("redis-server") {
		r.Fatalf("redis-server binary unavailable for the isolated restore daemon")
	}
	must(run.Run(nil, 15*time.Second, "", "redis-server", "--bind", "127.0.0.1", "--port", strconv.Itoa(RestoreRedisPort),
		"--daemonize", "yes", "--dir", work, "--dbfilename", "restore.rdb", "--appendonly", "no", "--save", "",
		"--pidfile", filepath.Join(work, "redis.pid"), "--logfile", filepath.Join(work, "redis.log")))
	rp := strconv.Itoa(RestoreRedisPort)
	for i := 0; i < 100; i++ {
		if b.redisReady(rp) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	r.Require(b.redisReady(rp), "isolated Redis restore started")

	base, restored := b.start(28081, backupRestoreDB, fmt.Sprintf("redis://127.0.0.1:%d", RestoreRedisPort), "restored-server.log")
	restoredWorks := JList(b.hc.Do("GET", base+"/v1/works", Opts{Query: url.Values{"q": {"backup-restore-book"}}, Basic: b.basic}).JSON(), "works")
	r.Require(len(restoredWorks) == 1 && jstrOr(restoredWorks[0], "id") == b.workID, "restored catalog verified through fresh server")
	found := b.hc.Do("POST", base+"/v1/ai/memories/search", Opts{Header: b.admin, JSON: map[string]any{"scope": "backup-restore", "embedding": []any{1, 0, 0}, "limit": 5}, Timeout: 5 * time.Second})
	r.Require(found.Status == 200 && memoryHit(found.JSON(), backupMemoryID), "restored vector memory verified")
	saved := b.hc.Do("GET", base+"/v1/sync/progress/"+backupProgressUsr+"/"+backupProgressBk, Opts{Header: b.admin, Timeout: 5 * time.Second})
	r.Require(saved.Status == 200 && JNum(saved.JSON(), "Pos", "Chapter") == 9, "restored reading progress verified")
	b.stop(restored)
}
