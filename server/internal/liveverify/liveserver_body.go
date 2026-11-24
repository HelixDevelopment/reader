package liveverify

import (
	"bytes"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

func jobReportText(job any) string {
	if job == nil {
		return "null"
	}
	rep, _ := jget(job, "report")
	raw, _ := json.Marshal(rep)
	return string(raw)
}

func hasPrefix(b []byte, p string) bool { return bytes.HasPrefix(b, []byte(p)) }

func hdr(r *Resp, k string) string { return r.Header.Get(k) }

// body is the ordered assertion list of verify_live_server.py main().
func (v *liveServer) body(digests map[string]string) {
	r := v.rec
	cfg := v.cfg
	r.Require(v.get("/healthz", Opts{}).Status == 200, "health endpoint")
	r.Require(v.post("/v1/sources", Opts{JSON: map[string]any{}}).Status == 401, "admin authentication boundary")
	r.Require(v.get("/v1/works", Opts{}).Status == 401, "catalog authentication boundary")

	for _, query := range []string{"live-server-book", "live-server-pse", "live-server-mixed", "live-server-forge", "live-server-cbr-stub", "live-server-plain"} {
		existing := v.get("/v1/works", v.basicOpts(Opts{Query: url.Values{"q": {query}, "limit": {"100"}}})).RaiseForStatus()
		for _, work := range JList(existing.JSON(), "works") {
			if v.del("/v1/works/"+JStr(work, "id"), v.adminOpts(Opts{})).Status != 204 {
				r.Fatalf("pre-test catalog cleanup")
			}
		}
	}

	created := v.post("/v1/sources", v.adminOpts(Opts{JSON: map[string]any{"id": "live-server-source", "protocol": "local", "root": cfg.Library}}))
	r.Require(created.Status == 201, "live local source creation")
	sourceID := JStr(created.JSON(), "id")
	scan := v.post("/v1/scans", v.adminOpts(Opts{JSON: map[string]any{"source_id": sourceID, "prefix": ""}}))
	r.Require(scan.Status == 202, "live scan accepted")
	jobID := JStr(scan.JSON(), "id")
	job := v.waitJob(jobID, 100)
	files, okFiles := scanFiles(job)
	r.Require(job != nil && jstrOr(job, "state") == "succeeded" && okFiles && files == 1, "live in-place ingestion")

	works := v.get("/v1/works", v.basicOpts(Opts{Query: url.Values{"q": {"live-server-book"}}})).JSON()
	r.Require(len(JList(works, "works")) == 1, "live catalog query")
	work := J(works, "works", 0)
	file := J(work, "editions", 0, "files", 0)
	fileID, fileSize := JStr(file, "id"), JNum(file, "size")
	r.Require(fileID != "" && jstrOr(file, "source_id") == sourceID, "file identity and source association")
	r.Require(jstrOr(file, "sha256") != "" && fileSize > 0, "content identity and size")

	atom := v.get("/opds/1.2/publications", v.basicOpts(Opts{}))
	r.Require(atom.Status == 200 && strings.Contains(hdr(atom, "Content-Type"), "application/atom+xml") && bytes.Contains(atom.Body, []byte("live-server-book")), "OPDS 1.2 acquisition feed")
	feed := v.get("/opds/2.0/publications", v.basicOpts(Opts{}))
	_, hasPubs := jget(feed.JSON(), "publications")
	r.Require(feed.Status == 200 && strings.Contains(feed.Text(), "live-server-book") && hasPubs, "OPDS 2.0 publication feed")
	r.Require(v.get("/opds/1.2/opensearch.xml", v.basicOpts(Opts{})).Status == 200, "OpenSearch description")

	contentPath := "/v1/files/" + fileID + "/content"
	ranged := v.get(contentPath, v.basicOpts(Opts{Header: map[string]string{"Range": "bytes=0-3"}}))
	r.Require(ranged.Status == 206 && string(ranged.Body) == "live" && strings.HasPrefix(hdr(ranged, "Content-Range"), "bytes 0-3/"), "single-range content download")
	head := v.hc.Do("HEAD", v.base+contentPath, v.basicOpts(Opts{}))
	clen, _ := strconv.Atoi(hdr(head, "Content-Length"))
	r.Require(head.Status == 200 && len(head.Body) == 0 && float64(clen) == fileSize, "HEAD content metadata without body")
	stale := v.get(contentPath, v.basicOpts(Opts{Header: map[string]string{"Range": "bytes=0-3", "If-Range": `"sha256-stale"`}}))
	r.Require(stale.Status == 200 && hasPrefix(stale.Body, "live-server-book"), "stale If-Range returns complete representation")
	multiple := v.get(contentPath, v.basicOpts(Opts{Header: map[string]string{"Range": "bytes=0-3,5-8"}}))
	r.Require(multiple.Status == 200 && hasPrefix(multiple.Body, "live-server-book"), "multi-range request uses complete-response fallback")
	binary := v.get(contentPath, v.basicOpts(Opts{Header: map[string]string{"Accept-Encoding": "br, gzip"}}))
	r.Require(binary.Status == 200 && hdr(binary, "Content-Encoding") == "" && hasPrefix(binary.Body, "live-server-book"), "attachment remains identity encoded")
	compressed := v.get("/v1/formats", Opts{Header: map[string]string{"Accept-Encoding": "gzip"}})
	r.Require(compressed.Status == 200 && hdr(compressed, "Content-Encoding") == "gzip" && len(JList(compressed.JSON(), "formats")) > 0, "gzip content negotiation")

	// --- OPDS-PSE page streaming ------------------------------------
	streamsSource := v.post("/v1/sources", v.adminOpts(Opts{JSON: map[string]any{"id": "live-server-streams", "protocol": "local", "root": cfg.StreamLib}}))
	r.Require(streamsSource.Status == 201, "live page-stream source creation")
	streamScan := v.post("/v1/scans", v.adminOpts(Opts{JSON: map[string]any{"source_id": JStr(streamsSource.JSON(), "id"), "prefix": ""}}))
	r.Require(streamScan.Status == 202, "page-stream scan accepted")
	streamJob := v.waitJob(JStr(streamScan.JSON(), "id"), 200)
	sFiles, sOK := scanFiles(streamJob)
	r.Require(streamJob != nil && jstrOr(streamJob, "state") == "succeeded" && sOK && int(sFiles) == len(digests),
		"page-stream fixtures ingested: "+jobReportText(streamJob))

	cat := v.catalogByDigest(digests)
	names := make([]string, 0, len(cat))
	for n := range cat {
		names = append(names, n)
	}
	sort.Strings(names)
	r.Require(len(cat) == len(digests), "every page-stream fixture is catalogued by digest: "+strings.Join(names, ","))
	comicRec := cat["live-server-pse.cbz"]
	comic := comicRec.file
	comicID, comicSHA := JStr(comic, "id"), JStr(comic, "sha256")

	advertised := v.editionStreams(comicRec.editionID)
	r.Require(len(advertised) == 1, "a CBZ edition advertises exactly one page stream")
	stream := advertised[0]
	r.Require(stream.HasCount && stream.Count == strconv.Itoa(len(psePages)), "pse:count equals the pages in the archive")
	r.Require(stream.Type == "image/png", "the advertised stream type comes from the page bytes, not the file name")
	r.Require(stream.Href == "/opds-pse/stream/"+comicID+"/{pageNumber}", "the stream href names this file and the page template")
	noLastRead := true
	for _, a := range stream.Attrs {
		if strings.Contains(a, "lastRead") {
			noLastRead = false
		}
	}
	r.Require(noLastRead, "a stream the server cannot position carries no pse:lastRead")

	for _, refused := range []string{"live-server-mixed.cbz", "live-server-forge.cbz", "live-server-cbr-stub.cbr", "live-server-plain.txt"} {
		rec, ok := cat[refused]
		if !ok {
			r.Fatalf("KeyError: %s", refused)
		}
		r.Require(len(v.editionStreams(rec.editionID)) == 0, refused+" is not advertised as pageable")
		// Refusing a stream must not cost access to the file itself.
		whole := v.get("/v1/files/"+JStr(rec.file, "id")+"/content", v.basicOpts(Opts{}))
		r.Require(whole.Status == 200 && len(whole.Body) > 0, refused+" still ships whole")
		page := v.get("/opds-pse/stream/"+JStr(rec.file, "id")+"/0", v.basicOpts(Opts{}))
		r.Require(page.Status == 404, refused+" refuses page requests as well as the advertisement")
	}

	for index, member := range psePages {
		page := v.get("/opds-pse/stream/"+comicID+"/"+strconv.Itoa(index), v.basicOpts(Opts{}))
		r.Require(page.Status == 200 && bytes.Equal(page.Body, member.payload) && hdr(page, "Content-Type") == "image/png",
			"page "+strconv.Itoa(index)+" is served byte-identical to the archive member")
		r.Require(hdr(page, "ETag") == `"$sha256-`+comicSHA+`$page-`+strconv.Itoa(index)+`"`,
			"page "+strconv.Itoa(index)+" is tagged with the catalogued file digest")
	}

	beyond := v.get("/opds-pse/stream/"+comicID+"/"+strconv.Itoa(len(psePages)), v.basicOpts(Opts{}))
	r.Require(beyond.Status == 404 && strings.Contains(beyond.Text(), "0.."+strconv.Itoa(len(psePages)-1)), "page past the last returns the valid range")
	r.Require(v.get("/opds-pse/stream/"+comicID+"/notanumber", v.basicOpts(Opts{})).Status == 400, "a non-integer page number is rejected")
	r.Require(v.get("/opds-pse/stream/no-such-file/0", v.basicOpts(Opts{})).Status == 404, "an unknown file has no page stream")

	listed := v.get("/opds/1.2/publications", v.basicOpts(Opts{Query: url.Values{"q": {"live-server-pse"}, "limit": {"100"}}}))
	r.Require(listed.Status == 200 && len(feedStreams(listed.Body)[comicRec.editionID]) == 1, "the acquisition feed carries the stream link")
	pubs20 := v.get("/opds/2.0/publications", v.basicOpts(Opts{Query: url.Values{"q": {"live-server-pse"}, "limit": {"100"}}}))
	r.Require(pubs20.Status == 200 && !strings.Contains(pubs20.Text(), "vaemendis"), "OPDS 2.0 invents no page-stream extension")

	client := v.runPSEClient("OPDS-PSE client round trip", "pse-client.json")
	links, _ := client["links"].(float64)
	pages, _ := client["pages"].(float64)
	r.Require(links >= 1 && int(pages) == len(psePages), "the PSE client fetched every advertised page")

	cred := map[string]string{"x-auth-user": "live-server-reader", "x-auth-key": "0123456789abcdef0123456789abcdef"}
	registration := v.post("/users/create", Opts{JSON: map[string]any{"username": cred["x-auth-user"], "password": cred["x-auth-key"]}, Timeout: 5 * time.Second})
	r.Require(registration.Status == 201 || registration.Status == 402, "KOSync registration or deterministic rerun")
	if registration.Status == 402 {
		r.Require(jstrOr(registration.JSON(), "message") == "Username is already registered.", "KOSync 402 is a duplicate account, not disabled registration")
	}
	r.Require(v.get("/users/auth", Opts{Header: cred, Timeout: 5 * time.Second}).Status == 200, "KOSync authentication")
	progress := map[string]any{"document": "live-server-document", "progress": "42", "percentage": 0.25, "device": "live-device", "device_id": "live-device"}
	r.Require(v.put("/syncs/progress", Opts{Header: cred, JSON: progress, Timeout: 5 * time.Second}).Status == 200, "KOSync progress write")
	synced := v.get("/syncs/progress/live-server-document", Opts{Header: cred, Timeout: 5 * time.Second})
	r.Require(synced.Status == 200 && jstrOr(synced.JSON(), "progress") == "42" && JNum(synced.JSON(), "timestamp") > 0, "KOSync progress read")

	memory := map[string]any{"record": map[string]any{"id": "live-server-memory", "scope": "live-server", "kind": "note", "content": "deterministic memory", "model": "live-model"}, "embedding": []any{1, 0, 0}}
	r.Require(v.put("/v1/ai/memories", v.adminOpts(Opts{JSON: memory, Timeout: 5 * time.Second})).Status == 200, "pgvector memory write")
	found := v.post("/v1/ai/memories/search", v.adminOpts(Opts{JSON: map[string]any{"scope": "live-server", "embedding": []any{1, 0, 0}, "limit": 5}, Timeout: 5 * time.Second}))
	r.Require(found.Status == 200 && memoryHit(found.JSON(), "live-server-memory"), "pgvector scoped search")

	internal := map[string]any{"BookID": "ignored", "DeviceID": "live-device", "Pos": map[string]any{"Chapter": 2, "Block": 1, "Offset": 0}, "Clock": map[string]any{"V": map[string]any{"live-device": 1}}}
	r.Require(v.put("/v1/sync/progress/live-user/live-book", v.adminOpts(Opts{JSON: internal, Timeout: 5 * time.Second})).Status == 200, "internal progress write")

	v.stop()
	if cfg.RestartDaemons {
		must(cfg.Daemons.Restart())
		r.Require(cfg.Daemons.RedisReady(), "Redis and PostgreSQL daemons restarted")
	}
	v.start()
	persisted := v.get("/v1/works", v.basicOpts(Opts{Query: url.Values{"q": {"live-server-book"}}})).JSON()
	r.Require(len(JList(persisted, "works")) == 1, "PostgreSQL catalog survives server restart")
	// The scan finished before the process stopped, so its record must be
	// readable by the new one.
	restoredJob := v.get("/v1/scans/"+jobID, v.adminOpts(Opts{}))
	r.Require(restoredJob.Status == 200 && jstrOr(restoredJob.JSON(), "state") == "succeeded", "completed scan job survives server restart")
	jobs := JList(v.get("/v1/scans", v.adminOpts(Opts{})).JSON(), "jobs")
	inList := false
	for _, e := range jobs {
		if jstrOr(e, "id") == jobID {
			inList = true
		}
	}
	r.Require(inList, "restored scan job appears in the scan list")
	r.Require(v.get("/users/auth", Opts{Header: cred, Timeout: 5 * time.Second}).Status == 200, "Redis KOSync account survives server restart")
	r.Require(jstrOr(v.get("/syncs/progress/live-server-document", Opts{Header: cred, Timeout: 5 * time.Second}).JSON(), "progress") == "42", "Redis KOSync progress survives server restart")
	internalSaved := v.get("/v1/sync/progress/live-user/live-book", v.adminOpts(Opts{Timeout: 5 * time.Second}))
	r.Require(internalSaved.Status == 200 && JNum(internalSaved.JSON(), "Pos", "Chapter") == 2, "Redis internal progress survives server restart")
	reattached := v.post("/v1/sources", v.adminOpts(Opts{JSON: map[string]any{"id": sourceID, "protocol": "local", "root": cfg.Library}}))
	r.Require(reattached.Status == 201 && JStr(reattached.JSON(), "id") == sourceID, "stable source identity reattaches after restart")
	restream := v.post("/v1/sources", v.adminOpts(Opts{JSON: map[string]any{"id": "live-server-streams", "protocol": "local", "root": cfg.StreamLib}}))
	r.Require(restream.Status == 201, "page-stream source reattaches after restart")
	// The advertisement is a property of the bytes, not of state the first process held open.
	v.runPSEClient("OPDS-PSE client round trip after restart", "pse-client-after-restart.json")
	restored := v.get(contentPath, v.basicOpts(Opts{}))
	r.Require(restored.Status == 200 && hasPrefix(restored.Body, "live-server-book"), "existing file identity serves after source re-registration")
	rescan := v.post("/v1/scans", v.adminOpts(Opts{JSON: map[string]any{"source_id": sourceID, "prefix": ""}}))
	r.Require(rescan.Status == 202, "post-restart source rescan accepted")
	rescanJob := v.waitJob(JStr(rescan.JSON(), "id"), 100)
	removed, okRemoved := scanRemoved(rescanJob)
	r.Require(rescanJob != nil && jstrOr(rescanJob, "state") == "succeeded" && okRemoved && removed == 0, "post-restart reconciliation preserves present file")
	persisted = v.get("/v1/works", v.basicOpts(Opts{Query: url.Values{"q": {"live-server-book"}}})).JSON()
	restoredFiles := JList(J(persisted, "works", 0, "editions", 0), "files")
	r.Require(len(JList(persisted, "works")) == 1 && len(restoredFiles) == 1 && jstrOr(restoredFiles[0], "id") == fileID, "stable source rescan does not duplicate files")
	r.Require(v.del("/v1/works/"+JStr(work, "id"), v.adminOpts(Opts{})).Status == 204, "post-test catalog cleanup")
	var left []string
	for _, name := range names {
		if v.del("/v1/works/"+cat[name].workID, v.adminOpts(Opts{})).Status != 204 {
			left = append(left, name)
		}
	}
	r.Require(len(left) == 0, "post-test page-stream catalog cleanup failed for: "+strings.Join(left, ","))
	r.Require(v.del("/v1/ai/memories/live-server/live-server-memory", v.adminOpts(Opts{})).Status == 204, "post-test memory cleanup")
	r.Require(v.del("/v1/sync/progress/live-user/live-book", v.adminOpts(Opts{})).Status == 204, "post-test progress cleanup")
}

func memoryHit(body any, id string) bool {
	for _, item := range JList(body, "results") {
		if jstrOr(J(item, "record"), "id") == id {
			return true
		}
	}
	return false
}
