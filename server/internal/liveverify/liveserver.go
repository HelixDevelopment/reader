package liveverify

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	atomNS = "http://www.w3.org/2005/Atom"
	pseNS  = "http://vaemendis.net/opds-pse/ns"
	pseRel = "http://vaemendis.net/opds-pse/stream"
)

var (
	pngSig  = []byte("\x89PNG\r\n\x1a\n")
	jpegSig = []byte("\xff\xd8\xff")
)

type pseMember struct {
	name    string
	payload []byte
}

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// psePages is the streamable fixture and the pages it must promise, in archive
// order. Payloads carry their own signature so the advertised type is read from bytes.
var psePages = []pseMember{
	{"p01.png", cat(pngSig, []byte("page-one"))},
	{"p02.png", cat(pngSig, []byte("page-two"))},
	{"p03.png", cat(pngSig, []byte("page-three"))},
}

// LiveServerConfig configures the live server verifier. Defaults match the Python verifier.
type LiveServerConfig struct {
	Launcher     Launcher
	EvidenceDir  string
	Library      string // local source root for the plain book
	StreamLib    string // local source root for the page-stream fixtures
	PSECheck     string // path of the helix-pse-check binary (built from this tree)
	PSEEnv       []string
	AdminToken   string // default live-admin-token
	OPDSUser     string // default live-reader
	OPDSPassword string // default live-password
	// RestartDaemons mirrors --restart-daemons: restart Redis and PostgreSQL between server runs.
	RestartDaemons bool
	Daemons        Daemons
}

// Daemons controls the Redis/PostgreSQL daemons for the --restart-daemons leg.
type Daemons interface {
	Restart() error // save, restart redis and postgres, wait until ready
	RedisReady() bool
	StartRedis() error // idempotent: ensure a Redis daemon is up (restoration in finally)
}

func (c *LiveServerConfig) defaults() {
	if c.AdminToken == "" {
		c.AdminToken = "live-admin-token"
	}
	if c.OPDSUser == "" {
		c.OPDSUser = "live-reader"
	}
	if c.OPDSPassword == "" {
		c.OPDSPassword = "live-password"
	}
}

func writeArchive(path string, members []pseMember) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for _, m := range members {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: m.name, Method: zip.Deflate})
		if err != nil {
			return err
		}
		if _, err := w.Write(m.payload); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return f.Close()
}

// streamFixtures lays down one archive a page stream is promised for plus the
// files it must be refused for and maps each fixture's sha256 to its file name.
func streamFixtures(dir string) (map[string]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	// Empty directory first: a stale fixture would change the file count the scan is asserted against.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return nil, err
		}
	}
	steps := []func() error{
		func() error { return writeArchive(filepath.Join(dir, "live-server-pse.cbz"), psePages) },
		// Pages of two types have no single honest stream type, so they get no link.
		func() error {
			return writeArchive(filepath.Join(dir, "live-server-mixed.cbz"), []pseMember{{"p01.png", cat(pngSig, []byte("a"))}, {"p02.jpg", cat(jpegSig, []byte("b"))}})
		},
		// An extension is not proof of a type.
		func() error {
			return writeArchive(filepath.Join(dir, "live-server-forge.cbz"), []pseMember{{"cover.png", []byte("an image by extension only")}})
		},
		// A registered container this server cannot enumerate pages for.
		func() error {
			return os.WriteFile(filepath.Join(dir, "live-server-cbr-stub.cbr"), []byte("Rar!\x1a\x07\x01\x00not a zip archive"), 0o644)
		},
		func() error {
			return os.WriteFile(filepath.Join(dir, "live-server-plain.txt"), []byte("not pageable\n"), 0o644)
		},
	}
	for _, s := range steps {
		if err := s(); err != nil {
			return nil, err
		}
	}
	entries, err = os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	digests := map[string]string{}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(raw)
		digests[hex.EncodeToString(sum[:])] = e.Name()
	}
	return digests, nil
}

// --- Atom / PSE parsing ------------------------------------------------------

type atomLink struct {
	Attrs []xml.Attr `xml:",any,attr"`
}

func (l atomLink) attr(space, local string) (string, bool) {
	for _, a := range l.Attrs {
		if a.Name.Local == local && a.Name.Space == space {
			return a.Value, true
		}
	}
	return "", false
}

type atomEntry struct {
	Links []atomLink `xml:"http://www.w3.org/2005/Atom link"`
}
type atomFeed struct {
	Entries []atomEntry `xml:"http://www.w3.org/2005/Atom entry"`
}

// StreamLink is a PSE link as a client reads it.
type StreamLink struct {
	Href, Type, Count string
	HasCount          bool
	Attrs             []string
}

func streamLinks(e atomEntry) []StreamLink {
	var out []StreamLink
	for _, l := range e.Links {
		if rel, _ := l.attr("", "rel"); rel != pseRel {
			continue
		}
		href, _ := l.attr("", "href")
		typ, _ := l.attr("", "type")
		count, has := l.attr(pseNS, "count")
		var names []string
		for _, a := range l.Attrs {
			if a.Name.Space != "" {
				names = append(names, "{"+a.Name.Space+"}"+a.Name.Local)
			} else {
				names = append(names, a.Name.Local)
			}
		}
		sort.Strings(names)
		out = append(out, StreamLink{Href: href, Type: typ, Count: count, HasCount: has, Attrs: names})
	}
	return out
}

func feedStreams(body []byte) map[string][]StreamLink {
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		panic(assertion{"feed is not Atom XML: " + err.Error()})
	}
	byEdition := map[string][]StreamLink{}
	for _, e := range feed.Entries {
		edition := ""
		for _, l := range e.Links {
			rel, _ := l.attr("", "rel")
			href, _ := l.attr("", "href")
			if rel == "alternate" && strings.HasPrefix(href, "/opds/1.2/editions/") {
				tail := href[strings.LastIndex(href, "/")+1:]
				if un, err := url.PathUnescape(tail); err == nil {
					tail = un
				}
				edition = tail
			}
		}
		byEdition[edition] = streamLinks(e)
	}
	return byEdition
}

func pathEscapeAll(s string) string { return url.PathEscape(s) }

// --- verifier ----------------------------------------------------------------

type liveServer struct {
	proc  Process
	cfg   LiveServerConfig
	rec   *Recorder
	hc    *Client
	base  string
	admin map[string]string
	basic *[2]string
}

func (v *liveServer) get(path string, o Opts) *Resp  { return v.hc.Do("GET", v.base+path, o) }
func (v *liveServer) post(path string, o Opts) *Resp { return v.hc.Do("POST", v.base+path, o) }
func (v *liveServer) put(path string, o Opts) *Resp  { return v.hc.Do("PUT", v.base+path, o) }
func (v *liveServer) del(path string, o Opts) *Resp  { return v.hc.Do("DELETE", v.base+path, o) }

func (v *liveServer) adminOpts(o Opts) Opts {
	o.Header = merge(v.admin, o.Header)
	return o
}
func (v *liveServer) basicOpts(o Opts) Opts { o.Basic = v.basic; return o }

func merge(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, x := range a {
		out[k] = x
	}
	for k, x := range b {
		out[k] = x
	}
	return out
}

func terminal(state string) bool {
	return state == "succeeded" || state == "failed" || state == "cancelled"
}

func (v *liveServer) waitJob(id string, polls int) any {
	var job any
	for i := 0; i < polls; i++ {
		job = v.get("/v1/scans/"+id, v.adminOpts(Opts{})).JSON()
		if terminal(jstrOr(job, "state")) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return job
}

func scanFiles(job any) (float64, bool) {
	rep, ok := jget(job, "report")
	if !ok {
		return 0, false
	}
	x, ok := jget(rep, "files")
	n, isNum := x.(float64)
	return n, ok && isNum
}

func scanRemoved(job any) (float64, bool) {
	rep, ok := jget(job, "report")
	if !ok {
		return 0, false
	}
	x, ok := jget(rep, "removed")
	n, isNum := x.(float64)
	return n, ok && isNum
}

type catalogued struct {
	workID, editionID string
	file              any
}

func (v *liveServer) catalogByDigest(digests map[string]string) map[string]catalogued {
	found := map[string]catalogued{}
	for digest, name := range digests {
		stem := name[:strings.LastIndex(name, ".")]
		resp := v.get("/v1/works", v.basicOpts(Opts{Query: url.Values{"q": {stem}, "limit": {"100"}}})).RaiseForStatus()
		body := resp.JSON()
		for _, work := range JList(body, "works") {
			for _, edition := range JList(work, "editions") {
				for _, file := range JList(edition, "files") {
					if jstrOr(file, "sha256") == digest {
						found[name] = catalogued{workID: JStr(work, "id"), editionID: JStr(edition, "id"), file: file}
					}
				}
			}
		}
	}
	return found
}

func (v *liveServer) editionStreams(editionID string) []StreamLink {
	resp := v.get("/opds/1.2/editions/"+pathEscapeAll(editionID), v.basicOpts(Opts{})).RaiseForStatus()
	var e atomEntry
	if err := xml.Unmarshal(resp.Body, &e); err != nil {
		panic(assertion{"edition entry is not Atom XML: " + err.Error()})
	}
	return streamLinks(e)
}

func (v *liveServer) runPSEClient(name, evidence string) map[string]any {
	cmd := exec.Command(v.cfg.PSECheck, "-base", v.base, "-q", "live-server-pse")
	cmd.Env = v.cfg.PSEEnv
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := runWithTimeout(cmd, 120*time.Second)
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(stdout.String()), &payload); err != nil || payload == nil {
		payload = map[string]any{"status": "FAIL", "failures": []any{"checker produced no JSON: " + clip(stdout.String(), 400) + clip(stderr.String(), 400)}}
	}
	raw, _ := json.MarshalIndent(payload, "", "  ")
	if err := os.WriteFile(filepath.Join(v.cfg.EvidenceDir, evidence), append(raw, '\n'), 0o644); err != nil {
		panic(err)
	}
	// The child's verdict, not the harness's reading of it, decides: a checker
	// that exited non-zero is a failure even if its JSON looks tidy.
	status, _ := payload["status"].(string)
	v.rec.Require(runErr == nil && status == "PASS", name+": "+fmt.Sprint(payload["failures"]))
	return payload
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func runWithTimeout(cmd *exec.Cmd, d time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		_ = cmd.Process.Kill()
		<-done
		return errors.New("timeout")
	}
}

// RunLiveServer runs the whole live server verification and returns the results
// (also written to EvidenceDir/results.json). It never panics.
func RunLiveServer(cfg LiveServerConfig) Results {
	cfg.defaults()
	rec := &Recorder{}
	v := &liveServer{cfg: cfg, rec: rec, hc: NewClient(),
		admin: map[string]string{"Authorization": "Bearer " + cfg.AdminToken},
		basic: &[2]string{cfg.OPDSUser, cfg.OPDSPassword}}
	defer v.hc.Close()
	if err := os.MkdirAll(cfg.EvidenceDir, 0o755); err != nil {
		rec.Fail("evidence directory: " + err.Error())
		return finishLive(cfg, rec, true)
	}
	failed := !rec.Guard(func() {
		must(os.MkdirAll(cfg.Library, 0o755))
		must(os.WriteFile(filepath.Join(cfg.Library, "live-server-book.txt"), []byte("live-server-book\n\nA deterministic paragraph for live ingestion.\n"), 0o644))
		digests, err := streamFixtures(cfg.StreamLib)
		must(err)
		v.start()
		v.body(digests)
	})
	if cfg.RestartDaemons && cfg.Daemons != nil {
		if restoreRedis(rec, cfg.Daemons) {
			failed = true
		}
	}
	if v.proc != nil {
		if err := v.proc.Stop(); err != nil {
			failed = true
			rec.Fail("server shutdown: " + err.Error())
		}
	}
	return finishLive(cfg, rec, failed)
}

// restoreRedis ensures a Redis daemon is up after the run. A host without
// redis-server records an honest SKIP (never a pass); any other error is a FAIL.
// It reports whether the step failed.
func restoreRedis(rec *Recorder, d Daemons) (failed bool) {
	err := d.StartRedis()
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrRedisNotInstalled):
		rec.Skip("Redis restoration: NOT RUN (" + err.Error() + ")")
		return false
	default:
		rec.Fail("Redis restoration: " + err.Error())
		return true
	}
}

func (v *liveServer) start() {
	base, p, err := v.cfg.Launcher.Start()
	must(err)
	v.base, v.proc = base, p
}

func (v *liveServer) stop() {
	p := v.proc
	v.proc = nil
	must(p.Stop())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func finishLive(cfg LiveServerConfig, rec *Recorder, failed bool) Results {
	res := rec.Results(failed)
	if err := WriteResults(cfg.EvidenceDir, res); err != nil {
		res.Status = "FAIL"
	}
	return res
}
