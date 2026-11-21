package httpapi

// Load harness for the remote-source delivery path (ATM-027).
//
// docs/api.md is explicit that "connectors without live revalidation use a full
// 200 without Content-Length, ETag, or ranges", and only sources.Local revalidates.
// So every non-local source — HTTP, WebDAV, OPDS, rclone — re-sends the whole
// object whenever a client resumes a download with a Range header. That is the
// documented contract, not a bug, but it is an unmeasured cost on the one surface
// where a book is largest and the network is slowest, which is what this file
// measures. It runs against an in-process origin and the in-memory catalog, so the
// numbers describe connector and route behaviour, not PostgreSQL.
//
// Gated on HELIX_LOAD_RUN=1 like the rest of the load suite.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/sources"
)

const remoteSourceID = "load-remote"

// remoteOrigin serves fixed objects with real range support and counts what left
// the building. The counters are the point: the route's response says nothing
// about how many bytes a remote source had to produce.
type remoteOrigin struct {
	server   *httptest.Server
	objects  map[string][]byte
	info     map[string]time.Time
	requests atomic.Int64
	ranged   atomic.Int64
	served26 atomic.Int64
	bytes    atomic.Int64
}

func (o *remoteOrigin) handler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path
	if len(name) > 0 {
		name = name[1:]
	}
	body, ok := o.objects[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	o.requests.Add(1)
	if r.Header.Get("Range") != "" {
		o.ranged.Add(1)
	}
	rec := &countingWriter{ResponseWriter: w}
	http.ServeContent(rec, r, name, o.info[name], newReadSeekerBytes(body))
	o.bytes.Add(int64(rec.written))
	if rec.status == http.StatusPartialContent {
		o.served26.Add(1)
	}
}

type countingWriter struct {
	http.ResponseWriter
	status   int
	written  int
	recorded bool
}

func (c *countingWriter) WriteHeader(status int) {
	if !c.recorded {
		c.status = status
		c.recorded = true
	}
	c.ResponseWriter.WriteHeader(status)
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.ResponseWriter.Write(p)
	c.written += n
	return n, err
}

type readSeekerBytes struct {
	data []byte
	pos  int64
}

func newReadSeekerBytes(data []byte) *readSeekerBytes { return &readSeekerBytes{data: data} }
func (r *readSeekerBytes) Read(p []byte) (int, error) {
	if r.pos >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += int64(n)
	return n, nil
}
func (r *readSeekerBytes) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.pos = offset
	case io.SeekCurrent:
		r.pos += offset
	case io.SeekEnd:
		r.pos = int64(len(r.data)) + offset
	}
	return r.pos, nil
}
func (r *readSeekerBytes) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
func (r *readSeekerBytes) Size() int64 { return int64(len(r.data)) }

func (o *remoteOrigin) Close() { o.server.Close() }

func remoteSizes(t *testing.T) []int64 {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("HELIX_REMOTE_SIZES"))
	if raw == "" {
		return []int64{262144, 4194304, 33554432}
	}
	out := []int64{}
	for _, part := range strings.Split(raw, ",") {
		v, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || v < 4096 {
			t.Fatalf("HELIX_REMOTE_SIZES contains %q", part)
		}
		out = append(out, v)
	}
	return out
}

func TestLiveLoadRemoteSource(t *testing.T) {
	if os.Getenv("HELIX_LOAD_RUN") != "1" {
		t.Skip("HELIX_LOAD_RUN is not 1; measurement harness opted out")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	origin := &remoteOrigin{objects: map[string][]byte{}, info: map[string]time.Time{}}
	now := time.Now().UTC().Truncate(time.Second)
	type remoteFile struct {
		id, name string
		size     int64
		sha256   string
		rangeReq string
		rangeLen int64
	}
	files := []remoteFile{}
	repo := catalog.NewMemory()
	for index, size := range remoteSizes(t) {
		name := "books/remote-" + strconv.Itoa(index) + ".bin"
		body := make([]byte, size)
		if _, err := rand.Read(body); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		digest := hex.EncodeToString(sum[:])
		origin.objects[name] = body
		origin.info[name] = now
		id := fmt.Sprintf("load-remote-f%d", index)
		work := catalog.Work{
			ID: id + "-w", CanonicalTitle: "Remote load " + id, CreatedAt: now, UpdatedAt: now,
			Editions: []catalog.Edition{{
				ID: id + "-e", Title: "Remote load " + id, State: catalog.Identified, UpdatedAt: now,
				Files: []catalog.File{{
					ID: id, SourceID: remoteSourceID, Path: name, Format: "bin",
					MIMEType: "application/octet-stream", Size: size, SHA256: digest, ModifiedAt: now,
				}},
			}},
		}
		if err := repo.UpsertWork(ctx, work); err != nil {
			t.Fatal(err)
		}
		request, window, _ := rangeFor(size)
		files = append(files, remoteFile{id: id, name: name, size: size, sha256: digest, rangeReq: request, rangeLen: window})
	}
	origin.server = httptest.NewServer(http.HandlerFunc(origin.handler))
	defer origin.Close()

	connector, err := sources.NewHTTP(sources.Config{Protocol: "http", Endpoint: origin.server.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{Address: "127.0.0.1:0", BrotliLevel: 4}, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo})
	srv.connectors[remoteSourceID] = connector
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 256, MaxConnsPerHost: 0, DisableCompression: true}}

	// --- the documented contract, checked before any throughput claim ---------
	//
	// A source that cannot revalidate cannot prove the bytes still match the scan,
	// so the route must withhold every validator that would let a client splice
	// representations: no ETag, no Accept-Ranges, no Content-Length from a stale
	// catalog size. Each of those absences is asserted, not assumed.
	for _, file := range files {
		url := ts.URL + "/v1/files/" + file.id + "/content"
		status, header, body := doContent(t, client, http.MethodGet, url, nil)
		if status != http.StatusOK {
			t.Fatalf("remote full GET %s: status %d", file.id, status)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(body)); got != file.sha256 {
			t.Fatalf("remote full GET %s delivered sha256 %s, catalog recorded %s", file.id, got, file.sha256)
		}
		if got := header.Get("ETag"); got != "" {
			t.Fatalf("remote %s advertised strong validator %q without live revalidation", file.id, got)
		}
		if got := header.Get("Accept-Ranges"); got != "" {
			t.Fatalf("remote %s advertised %q while the route cannot honour a range", file.id, got)
		}
		if got := header.Get("Content-Length"); got != "" {
			t.Fatalf("remote %s sent Content-Length %q from a catalog size the source was not asked to confirm", file.id, got)
		}

		before := origin.bytes.Load()
		status, header, body = doContent(t, client, http.MethodGet, url, map[string]string{"Range": file.rangeReq})
		transferred := origin.bytes.Load() - before
		if status != http.StatusOK {
			t.Fatalf("remote range GET %s returned %d; this source must get a full 200, because nothing confirmed the scanned size is still true", file.id, status)
		}
		if header.Get("Content-Range") != "" {
			t.Fatalf("remote range GET %s leaked Content-Range %q", file.id, header.Get("Content-Range"))
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(body)); got != file.sha256 {
			t.Fatalf("remote range fallback delivered sha256 %s, want the whole object %s", got, file.sha256)
		}
		if transferred < file.size {
			t.Fatalf("expected the origin to resend the whole %d-byte object for a refused range, it produced %d", file.size, transferred)
		}
		t.Logf("REMOTE RANGE COST size=%d client_window=%d origin_bytes=%d amplification=%.1fx",
			file.size, file.rangeLen, transferred, float64(transferred)/float64(file.rangeLen))
	}

	// --- envelope ---------------------------------------------------------------
	//
	// Concurrency still matters here even though the response is always the whole
	// object: this is the path where one reader can occupy a source connection and
	// a full buffer for the longest time, and the connector is doing HTTP to a
	// third party on behalf of every request.
	levels := deliveryLevels(t)
	rounds := func(clients int) int {
		need := 12 / clients
		if need < 2 {
			need = 2
		}
		return need
	}
	timeout := time.Duration(deliveryEnvInt(t, "HELIX_DELIVERY_TIMEOUT_SECONDS", 60)) * time.Second
	rows := []remoteRow{}
	for _, file := range files {
		url := ts.URL + "/v1/files/" + file.id + "/content"
		jobs := []deliveryJob{
			{kind: "full", url: url, wantStatus: http.StatusOK, wantBytes: file.size, wantSHA: file.sha256},
			{kind: "range", url: url, headers: map[string]string{"Range": file.rangeReq}, wantStatus: http.StatusOK, wantBytes: file.size, wantSHA: file.sha256},
		}
		for _, clients := range levels {
			requestsBefore := origin.requests.Load()
			bytesBefore := origin.bytes.Load()
			stats := runDeliveryLadder(ctx, client, jobs, clients, rounds(clients), timeout)
			stats.SizeBytes = file.size
			originBytes := origin.bytes.Load() - bytesBefore
			originRequests := origin.requests.Load() - requestsBefore
			rangeCount := int64(stats.StatusByKind["range"])
			row := remoteRow{
				deliveryStats:  stats,
				OriginBytes:    originBytes,
				OriginRequests: originRequests,
			}
			if rangeCount > 0 {
				// What a resuming client asked for versus what the source had to send.
				row.RequestedWindowBytes = rangeCount * file.rangeLen
				if row.RequestedWindowBytes > 0 {
					row.RangeAmplification = float64(stats.BytesByKind["range"]) / float64(row.RequestedWindowBytes)
				}
			}
			rows = append(rows, row)
			if stats.PeakInFlight < int64(clients) {
				t.Fatalf("remote size %d level %d: only %d requests were open at once", file.size, clients, stats.PeakInFlight)
			}
			for class, count := range stats.Mismatches {
				t.Fatalf("remote size %d level %d: %d %s responses failed their integrity needle (%v)", file.size, clients, count, class, stats.Mismatches)
			}
			if stats.Completed != stats.Requested {
				t.Fatalf("remote size %d level %d: %d of %d did not complete (%v)", file.size, clients, stats.Requested-stats.Completed, stats.Requested, stats.Errors)
			}
			if originBytes < stats.Bytes {
				t.Fatalf("remote size %d level %d: the route delivered %d bytes but the origin only produced %d",
					file.size, clients, stats.Bytes, originBytes)
			}
			if originRequests != int64(stats.Requested) {
				t.Fatalf("remote size %d level %d: %d client requests caused %d origin requests; the connector is not one-for-one",
					file.size, clients, stats.Requested, originRequests)
			}
			t.Logf("REMOTE LADDER size=%d clients=%d completed=%d req_per_sec=%.1f mib_per_sec=%.1f p50_ms=%.2f p95_ms=%.2f peak_inflight=%d client_bytes=%d origin_bytes=%d origin_requests=%d range_amplification=%.1fx",
				file.size, clients, stats.Completed, stats.ReqPerSec, stats.BytesPerSec/(1<<20), stats.P50MS, stats.P95MS,
				stats.PeakInFlight, stats.Bytes, originBytes, originRequests, row.RangeAmplification)
		}
	}
	httpEmitLoad(t, map[string]any{
		"test":             "remote_source_ladder",
		"measurement_path": "in_process_loopback",
		"connector":        "http",
		"catalog":          "memory",
		"note":             "only sources.Local revalidates, so non-local sources serve full 200s without validators by documented design",
		"origin_requests":  origin.requests.Load(),
		"origin_ranged":    origin.ranged.Load(),
		"origin_206":       origin.served26.Load(),
		"origin_bytes":     origin.bytes.Load(),
		"levels":           levels,
		"results":          rows,
	})
}

type remoteRow struct {
	deliveryStats
	OriginBytes          int64   `json:"origin_bytes_transferred"`
	OriginRequests       int64   `json:"origin_requests"`
	RequestedWindowBytes int64   `json:"client_requested_window_bytes"`
	RangeAmplification   float64 `json:"range_amplification_vs_requested_window"`
}
