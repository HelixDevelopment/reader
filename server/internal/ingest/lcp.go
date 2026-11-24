package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/helixreader/helix/core/formats"
	"github.com/helixreader/helix/core/pkg/lcp"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/sources"
)

// LCPLicenseMediaType is the media type of a License Document (LCP §3).
const LCPLicenseMediaType = "application/vnd.readium.lcp.license.v1.0+json"

const (
	defaultLCPMaxPublication = 256 << 20
	defaultLCPTimeout        = 2 * time.Minute
	// maxLCPLicenseBytes bounds a License Document. Real licenses are a few KiB.
	maxLCPLicenseBytes = 1 << 20
	maxLCPRedirects    = 3
)

var (
	// ErrLCPPrivateAddress reports a publication host that resolves to a
	// loopback, private, link-local, shared (CGNAT), multicast or unspecified
	// address while AllowPrivateNetwork is off.
	ErrLCPPrivateAddress = errors.New("lcp fulfillment: publication address is loopback or private and AllowPrivateNetwork is off")
	// ErrLCPTooLarge reports a publication larger than MaxPublicationBytes.
	ErrLCPTooLarge = errors.New("lcp fulfillment: publication exceeds the size limit")
	// ErrLCPBadURL reports a publication href this server will not fetch.
	ErrLCPBadURL = errors.New("lcp fulfillment: publication href is not an allowed http(s) URL")
	// ErrLCPFetch reports a failed publication download.
	ErrLCPFetch = errors.New("lcp fulfillment: publication download failed")
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// LCPFulfiller turns a standalone License Document (.lcpl) into the protected
// EPUB its publication link names, stored in a server-managed cache directory.
// It never decrypts anything, never reads a passphrase or key, and never writes
// into a source: source bytes stay untouched (HRD-004).
type LCPFulfiller struct {
	// CacheDir receives fulfilled artifacts (<sha256>.epub) and one small index
	// file per license. Required.
	CacheDir string
	// AllowPrivateNetwork lets the publication host be loopback/private. Off by
	// default: the href comes from an untrusted document, so allowing it by
	// default would turn license ingestion into an SSRF primitive.
	AllowPrivateNetwork bool
	// AllowUnverified passes lcp.AllowUnverified: fulfil a link that states no
	// length and no hash. Off by default (fail closed).
	AllowUnverified bool
	// MaxPublicationBytes bounds the download; <=0 means 256 MiB.
	MaxPublicationBytes int64
	// Timeout bounds the whole download; <=0 means 2 minutes.
	Timeout time.Duration
	// Dial overrides the final network dial (tests). The SSRF check still runs
	// on the resolved address first.
	Dial func(ctx context.Context, network, addr string) (net.Conn, error)
	// Resolver looks up the publication host; nil means net.DefaultResolver.
	Resolver interface {
		LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
	}
	// Now is the clock for failure backoff (tests); nil means time.Now.
	Now func() time.Time
	// CacheGrace protects unreferenced files younger than it from GC, so an
	// artifact fulfilled by a scan still in flight is never collected; 0 means
	// 10 minutes, negative means none.
	CacheGrace time.Duration
	// CacheMaxBytes is the cache size above which a collection warns. Nothing is
	// evicted for size: referenced artifacts and files younger than CacheGrace
	// are never deleted. 0 means 16 GiB, negative means unlimited.
	CacheMaxBytes int64
	// MaxScanFetches bounds the distinct publication downloads one scan may
	// start; 0 means 32, negative means unlimited. Cache hits are free.
	MaxScanFetches int
	// MaxScanBytes bounds the total stated bytes one scan may download; 0 means
	// 2 GiB, negative means unlimited.
	MaxScanBytes int64
	// HostFailureThreshold is the number of consecutive failures against one
	// publication host:port, across any licenses, that starts a host backoff;
	// 0 means 3, negative disables host backoff.
	HostFailureThreshold int

	mu        sync.Mutex
	fails     map[string]lcpFailure
	hostFails map[string]lcpFailure
	seq       uint64
	leases    map[string]int
	epoch     uint64
	verified  map[string]lcpVerdict
	flights   map[string]*lcpFlight
}

const (
	defaultLCPScanFetches = 32
	defaultLCPScanBytes   = int64(2) << 30
)

// ErrLCPScanBudget reports a license not fetched because this scan has used up
// its publication-download budget. It is not a failure of the license: nothing
// is backed off and the next scan retries it.
var ErrLCPScanBudget = errors.New("lcp fulfillment: scan fetch budget reached; retry on the next scan")

// LCPScan is one scan's view of the fulfiller: it enforces the per-scan fetch
// budget and leases every artifact it hands out until Release, so GC cannot
// collect a fulfilled artifact before its MergeWork has committed.
type LCPScan struct {
	f       *LCPFulfiller
	fetches int
	bytes   int64
	held    []string
	stopped bool
}

// BeginScan starts a scan-scoped fulfillment session. Close it when the scan
// ends; Release after each entry's MergeWork (or failure).
func (f *LCPFulfiller) BeginScan() *LCPScan {
	if f == nil {
		return nil
	}
	return &LCPScan{f: f}
}

// Fulfill is LCPFulfiller.Fulfill under this scan's budget, leasing the result.
func (sc *LCPScan) Fulfill(ctx context.Context, licenseDoc []byte) (Fulfilled, error) {
	return sc.f.fulfill(ctx, licenseDoc, sc)
}

// Release drops every lease taken since the last Release.
func (sc *LCPScan) Release() {
	for _, sha := range sc.held {
		sc.f.release(sha)
	}
	sc.held = nil
}

// Close releases all leases; call it when the scan ends.
func (sc *LCPScan) Close() { sc.Release() }

// Warnings returns the scan-report warnings this session earned.
func (sc *LCPScan) Warnings() []string {
	if !sc.stopped {
		return nil
	}
	return []string{fmt.Sprintf("LCP scan budget reached (%d publication fetches, %d bytes): remaining licenses were not fetched and are retried on the next scan (HELIX_LCP_MAX_SCAN_FETCHES, HELIX_LCP_MAX_SCAN_BYTES)", sc.fetches, sc.bytes)}
}

func (sc *LCPScan) leaseSHA(sha string) {
	sc.f.lease(sha)
	sc.held = append(sc.held, sha)
}

// settle trues a reservation up to what the fetch actually transferred, so a
// dead or refused publication costs no bytes against the scan.
func (sc *LCPScan) settle(reserved, actual int64) {
	if actual < 0 {
		actual = 0
	}
	if actual > reserved {
		actual = reserved
	}
	sc.bytes -= reserved - actual
	if sc.bytes < 0 {
		sc.bytes = 0
	}
}

// admit reserves budget for one fetch of up to cost bytes.
func (sc *LCPScan) admit(cost int64) error {
	maxF, maxB := sc.f.MaxScanFetches, sc.f.MaxScanBytes
	if maxF == 0 {
		maxF = defaultLCPScanFetches
	}
	if maxB == 0 {
		maxB = defaultLCPScanBytes
	}
	if (maxF > 0 && sc.fetches >= maxF) || (maxB > 0 && cost > maxB-sc.bytes) {
		sc.stopped = true
		return ErrLCPScanBudget
	}
	sc.fetches++
	sc.bytes += cost
	return nil
}

func (f *LCPFulfiller) maxPublication() int64 {
	if f.MaxPublicationBytes > 0 {
		return f.MaxPublicationBytes
	}
	return defaultLCPMaxPublication
}

// errLCPClientStatus marks a 4xx answer: the path was chosen by the license, so
// it is evidence about the license, not about the host.
var errLCPClientStatus = errors.New("client error status")

// errLCPRedirect: a redirect chain the license's URL set up (loop or too long).
var errLCPRedirect = errors.New("too many redirects")

type lcpFlight struct {
	done  chan struct{}
	err   error
	retry bool // the leader's outcome must not be inherited (its cancel/budget/panic)
}

func (f *LCPFulfiller) joinFlight(licenseSHA string) (*lcpFlight, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if fl, ok := f.flights[licenseSHA]; ok {
		return fl, false
	}
	if f.flights == nil {
		f.flights = map[string]*lcpFlight{}
	}
	fl := &lcpFlight{done: make(chan struct{})}
	f.flights[licenseSHA] = fl
	return fl, true
}

func (f *LCPFulfiller) endFlight(licenseSHA string, fl *lcpFlight) {
	f.mu.Lock()
	delete(f.flights, licenseSHA)
	f.mu.Unlock()
	close(fl.done)
}

// ErrLCPBackoff reports a license whose publication fetch failed recently.
var ErrLCPBackoff = errors.New("lcp fulfillment: publication fetch failed recently; retry later")

// LCPFromEnv returns a fulfiller when HELIX_LCP_CACHE_DIR is set and nil
// otherwise, which leaves .lcpl files skipped exactly as before.
//
//	HELIX_LCP_CACHE_DIR                 enable; directory for fulfilled artifacts
//	HELIX_LCP_ALLOW_PRIVATE_NETWORK=1   permit loopback/private publication hosts
//	HELIX_LCP_ALLOW_UNVERIFIED=1        permit links without length+hash
//	HELIX_LCP_MAX_BYTES                 download size limit
//	HELIX_LCP_CACHE_MAX_BYTES           total cache cap (default 16 GiB; <0 unlimited)
//	HELIX_LCP_TIMEOUT                   Go duration, e.g. 90s
//	HELIX_LCP_MAX_SCAN_FETCHES          publication downloads per scan (default 32; <0 unlimited)
//	HELIX_LCP_MAX_SCAN_BYTES            total stated bytes per scan (default 2 GiB; <0 unlimited)
func LCPFromEnv() *LCPFulfiller {
	dir := os.Getenv("HELIX_LCP_CACHE_DIR")
	if dir == "" {
		return nil
	}
	f := &LCPFulfiller{
		CacheDir:            dir,
		AllowPrivateNetwork: os.Getenv("HELIX_LCP_ALLOW_PRIVATE_NETWORK") == "1",
		AllowUnverified:     os.Getenv("HELIX_LCP_ALLOW_UNVERIFIED") == "1",
	}
	if v, err := strconv.ParseInt(os.Getenv("HELIX_LCP_MAX_BYTES"), 10, 64); err == nil && v > 0 {
		f.MaxPublicationBytes = v
	}
	if v, err := strconv.ParseInt(os.Getenv("HELIX_LCP_CACHE_MAX_BYTES"), 10, 64); err == nil && v != 0 {
		f.CacheMaxBytes = v
	}
	if v, err := strconv.Atoi(os.Getenv("HELIX_LCP_MAX_SCAN_FETCHES")); err == nil && v != 0 {
		f.MaxScanFetches = v
	}
	if v, err := strconv.ParseInt(os.Getenv("HELIX_LCP_MAX_SCAN_BYTES"), 10, 64); err == nil && v != 0 {
		f.MaxScanBytes = v
	}
	if v, err := time.ParseDuration(os.Getenv("HELIX_LCP_TIMEOUT")); err == nil && v > 0 {
		f.Timeout = v
	}
	return f
}

// Fulfilled describes one cached artifact.
type Fulfilled struct {
	SHA256 string
	Size   int64
}

// ArtifactPath returns the cache path for a fulfilled artifact hash, or "" when
// the hash is not a lowercase SHA-256 (so a catalog value can never address
// anything outside CacheDir).
func (f *LCPFulfiller) ArtifactPath(sha string) string {
	if f == nil || f.CacheDir == "" || !sha256Hex.MatchString(sha) {
		return ""
	}
	return filepath.Join(f.CacheDir, sha+".epub")
}

func (f *LCPFulfiller) indexPath(licenseSHA string) string {
	return filepath.Join(f.CacheDir, "license-"+licenseSHA+".idx")
}

// Fulfill returns the cached artifact for licenseDoc, downloading and
// fulfilling it first when there is none. A repeat call for the same license
// never touches the network.
func (f *LCPFulfiller) Fulfill(ctx context.Context, licenseDoc []byte) (Fulfilled, error) {
	return f.fulfill(ctx, licenseDoc, nil)
}

func (f *LCPFulfiller) fulfill(ctx context.Context, licenseDoc []byte, sc *LCPScan) (Fulfilled, error) {
	for {
		res, err, retry := f.fulfillOnce(ctx, licenseDoc, sc)
		if !retry {
			if err == nil && res.SHA256 == "" {
				return Fulfilled{}, errors.New("lcp fulfillment: no artifact produced")
			}
			return res, err
		}
		if cerr := ctx.Err(); cerr != nil {
			return Fulfilled{}, cerr
		}
	}
}

// fulfillOnce runs one pass; retry means a flight this call followed ended
// without a usable result for THIS caller (leader cancelled, out of budget,
// panicked, or its artifact vanished), so the caller should try again and may
// lead itself.
func (f *LCPFulfiller) fulfillOnce(ctx context.Context, licenseDoc []byte, sc *LCPScan) (res Fulfilled, rerr error, retry bool) {
	if f == nil || f.CacheDir == "" {
		return Fulfilled{}, errors.New("lcp fulfillment is not configured"), false
	}
	lic, err := lcp.Parse(licenseDoc)
	if err != nil {
		return Fulfilled{}, err, false
	}
	link, ok := lic.PublicationLink()
	if !ok {
		return Fulfilled{}, lcp.ErrNoPublicationLink, false
	}
	if link.Templated {
		// The href may carry a bearer token: never echo it.
		return Fulfilled{}, lcp.ErrTemplatedHref, false
	}
	// Refuse a non-EPUB link before any network use; lcp.Fulfill repeats this.
	if link.Type != "" && !strings.EqualFold(strings.TrimSpace(strings.SplitN(link.Type, ";", 2)[0]), "application/epub+zip") {
		return Fulfilled{}, fmt.Errorf("%w: %q (only application/epub+zip)", lcp.ErrUnsupportedFormat, link.Type), false
	}
	// Fail closed before any network use: a link that states no length and no
	// hash cannot be verified, so it must not cost a download.
	if (link.Length <= 0 || link.Hash == "") && !f.AllowUnverified {
		return Fulfilled{}, lcp.ErrPublicationUnverifiable, false
	}
	// A stated length above the publication limit is refused before any budget
	// or backoff is charged: it costs no network contact and no bytes.
	if link.Length > f.maxPublication() {
		return Fulfilled{}, ErrLCPTooLarge, false
	}
	if err := os.MkdirAll(f.CacheDir, 0o700); err != nil {
		return Fulfilled{}, err, false
	}
	lsum := sha256.Sum256(licenseDoc)
	licenseSHA := hex.EncodeToString(lsum[:])
	var got int64 // body bytes actually transferred by this fulfilment
	var lease func(string)
	if sc != nil {
		lease = sc.leaseSHA
	}
	if done, ok := f.cached(licenseSHA, lease); ok {
		f.touch(done.SHA256)
		return done, nil, false
	}
	// One origin download per license at a time: concurrent scans of the same
	// license wait for the leader and then hit the cache.
	if lcpBeforeJoinHook != nil {
		lcpBeforeJoinHook()
	}
	fl, leader := f.joinFlight(licenseSHA)
	if !leader {
		select {
		case <-fl.done:
		case <-ctx.Done():
			return Fulfilled{}, ctx.Err(), false
		}
		if done, ok := f.cached(licenseSHA, lease); ok {
			f.touch(done.SHA256)
			return done, nil, false
		}
		// The leader's own cancellation, budget refusal, panic or a vanished
		// artifact says nothing about this caller: try again, possibly leading.
		if fl.err == nil || fl.retry || errors.Is(fl.err, ErrLCPScanBudget) || errors.Is(fl.err, context.Canceled) || errors.Is(fl.err, context.DeadlineExceeded) {
			return Fulfilled{}, nil, true
		}
		return Fulfilled{}, fl.err, false
	}
	defer func() {
		p := recover()
		fl.err = rerr
		fl.retry = ctx.Err() != nil || errors.Is(rerr, ErrLCPScanBudget)
		if p != nil {
			fl.err, fl.retry = errors.New("lcp fulfillment: leader aborted"), true
		}
		f.endFlight(licenseSHA, fl)
		if p != nil {
			panic(p)
		}
	}()
	if lcpLeaderHook != nil {
		lcpLeaderHook()
	}
	// The previous leader may have finished between our cache miss and our
	// becoming leader: look again before downloading.
	if done, ok := f.cached(licenseSHA, lease); ok {
		f.touch(done.SHA256)
		return done, nil, false
	}
	host := ""
	if u, err := url.Parse(link.Href); err == nil {
		host = strings.ToLower(u.Host)
	}
	if f.backingOff(licenseSHA) || f.hostBackingOff(host) {
		return Fulfilled{}, ErrLCPBackoff, false
	}
	if sc != nil {
		cost := link.Length
		if cost <= 0 {
			cost = f.maxPublication()
		}
		if err := sc.admit(cost); err != nil {
			return Fulfilled{}, err, false
		}
		defer func() { sc.settle(cost, got) }()
	}
	done, err := f.fetchAndStore(ctx, licenseDoc, licenseSHA, link, lease, &got)
	if err != nil {
		if ctx.Err() != nil {
			// The operator cancelled: that says nothing about the license or host.
			return Fulfilled{}, err, false
		}
		// Only transport failures and non-2xx statuses say anything about the
		// host. Hash/length mismatches, size limits and verification errors are
		// defined by the (possibly hostile) document, so they back off the license
		// alone: otherwise a few bad licenses would lock out a healthy host.
		failHost := host
		if !errors.Is(err, ErrLCPFetch) || errors.Is(err, errLCPClientStatus) {
			// 4xx names a path the document chose: license-only backoff.
			failHost = ""
		}
		f.recordFailure(licenseSHA, failHost)
		return Fulfilled{}, err, false
	}
	f.clearFailure(licenseSHA, host)
	return done, nil, false
}

func (f *LCPFulfiller) fetchAndStore(ctx context.Context, licenseDoc []byte, licenseSHA string, link lcp.Link, lease func(string), got *int64) (Fulfilled, error) {
	spool, size, err := f.download(ctx, link.Href, link.Length, got)
	if err != nil {
		return Fulfilled{}, err
	}
	defer os.Remove(spool.Name())
	defer spool.Close()

	out, err := os.CreateTemp(f.CacheDir, "fulfill-*.tmp")
	if err != nil {
		return Fulfilled{}, err
	}
	tmp := out.Name()
	defer os.Remove(tmp) // no-op after a successful rename
	h := sha256.New()
	var opts []lcp.FulfillOption
	if f.AllowUnverified {
		opts = append(opts, lcp.AllowUnverified())
	}
	if err := lcp.FulfillTo(io.MultiWriter(out, h), licenseDoc, spool, size, opts...); err != nil {
		out.Close()
		return Fulfilled{}, err
	}
	n, err := out.Seek(0, io.SeekCurrent)
	if err != nil {
		out.Close()
		return Fulfilled{}, err
	}
	if err := out.Close(); err != nil {
		return Fulfilled{}, err
	}
	// Prove a reader will see the embedded license before the artifact is
	// published to the cache.
	if err := checkReadable(tmp, n); err != nil {
		return Fulfilled{}, err
	}
	artifactSHA := hex.EncodeToString(h.Sum(nil))
	if lease != nil {
		lease(artifactSHA) // pinned before it becomes visible to GC
	}
	if err := os.Rename(tmp, f.ArtifactPath(artifactSHA)); err != nil {
		return Fulfilled{}, err
	}
	if err := writeAtomic(f.CacheDir, f.indexPath(licenseSHA), []byte(artifactSHA+"\n")); err != nil {
		return Fulfilled{}, err
	}
	return Fulfilled{SHA256: artifactSHA, Size: n}, nil
}

func checkReadable(path string, size int64) error {
	fh, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	c, err := lcp.OpenContainer(fh, size)
	if err != nil {
		return err
	}
	if c.License == nil {
		return errors.New("lcp fulfillment: fulfilled container carries no license")
	}
	return c.EncryptionErr()
}

func writeAtomic(dir, dest string, data []byte) error {
	t, err := os.CreateTemp(dir, "idx-*.tmp")
	if err != nil {
		return err
	}
	name := t.Name()
	defer os.Remove(name)
	if _, err := t.Write(data); err != nil {
		t.Close()
		return err
	}
	if err := t.Close(); err != nil {
		return err
	}
	return os.Rename(name, dest)
}

// cached returns the artifact a previous Fulfill recorded for licenseSHA, but
// only when it is still present and still hashes to the recorded value.
func (f *LCPFulfiller) cached(licenseSHA string, lease func(string)) (Fulfilled, bool) {
	raw, err := os.ReadFile(f.indexPath(licenseSHA))
	if err != nil {
		return Fulfilled{}, false
	}
	sha := strings.TrimSpace(string(raw))
	p := f.ArtifactPath(sha)
	if p == "" {
		return Fulfilled{}, false
	}
	if lease != nil {
		lease(sha) // pinned before it is opened, so GC cannot race the hit
	}
	// Touch BEFORE opening: a collector in another process re-stats the file
	// after its catalog read and spares anything young, so the hit must be
	// visibly young before it can race an unlink.
	f.touch(sha)
	fh, err := os.Open(p)
	if err != nil {
		return Fulfilled{}, false
	}
	defer fh.Close()
	h := sha256.New()
	n, err := io.Copy(h, fh)
	if err != nil || hex.EncodeToString(h.Sum(nil)) != sha {
		return Fulfilled{}, false
	}
	if lcpAfterHashHook != nil {
		lcpAfterHashHook(p)
	}
	// ...and prove after hashing that the path still names the file we hashed.
	if now, serr := os.Stat(p); serr != nil || !os.SameFile(now, mustStat(fh)) {
		return Fulfilled{}, false
	}
	return Fulfilled{SHA256: sha, Size: n}, true
}

// download fetches href into an anonymous-ish temp file in CacheDir and returns
// it rewound. No credential is ever sent: the request carries nothing but the
// URL, so a redirect to another host cannot leak anything.
func (f *LCPFulfiller) download(ctx context.Context, href string, stated int64, got *int64) (*os.File, int64, error) {
	u, err := url.Parse(href)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return nil, 0, ErrLCPBadURL
	}
	max := f.MaxPublicationBytes
	if max <= 0 {
		max = defaultLCPMaxPublication
	}
	if stated > max {
		return nil, 0, ErrLCPTooLarge
	}
	if stated > 0 {
		max = stated // never pull more than the license says the publication is
	}
	timeout := f.Timeout
	if timeout <= 0 {
		timeout = defaultLCPTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tr := &http.Transport{
		Proxy:                 nil,
		DialContext:           f.dialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableKeepAlives:     true,
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > maxLCPRedirects {
			return errLCPRedirect
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return ErrLCPBadURL
		}
		if req.URL.User != nil {
			return ErrLCPBadURL
		}
		return nil
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, ErrLCPBadURL
	}
	req.Header.Set("Accept", "application/epub+zip")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, classifyFetch(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Statuses that follow from a path or redirect the LICENSE chose (other 4xx,
		// stray 3xx) are license-only evidence. 408/429 and 5xx are the host
		// speaking about itself (timeout, rate limit, failure) and stay host-level.
		if (resp.StatusCode >= 300 && resp.StatusCode < 500) && resp.StatusCode != http.StatusRequestTimeout && resp.StatusCode != http.StatusTooManyRequests {
			return nil, 0, fmt.Errorf("%w: status %d: %w", ErrLCPFetch, resp.StatusCode, errLCPClientStatus)
		}
		return nil, 0, fmt.Errorf("%w: status %d", ErrLCPFetch, resp.StatusCode)
	}
	if stated > 0 && resp.ContentLength >= 0 && resp.ContentLength != stated {
		return nil, 0, fmt.Errorf("%w: server announces %d bytes, link states %d", lcp.ErrPublicationMismatch, resp.ContentLength, stated)
	}
	if resp.ContentLength > max {
		return nil, 0, ErrLCPTooLarge
	}
	spool, err := os.CreateTemp(f.CacheDir, "download-*.tmp")
	if err != nil {
		return nil, 0, err
	}
	n, err := io.Copy(spool, io.LimitReader(resp.Body, max+1))
	*got = n
	if err != nil {
		spool.Close()
		os.Remove(spool.Name())
		return nil, 0, classifyFetch(err)
	}
	if n > max || (stated > 0 && n != stated) {
		spool.Close()
		os.Remove(spool.Name())
		if stated > 0 {
			return nil, 0, fmt.Errorf("%w: body is not the %d bytes the link states", lcp.ErrPublicationMismatch, stated)
		}
		return nil, 0, ErrLCPTooLarge
	}
	return spool, n, nil
}

// classifyFetch keeps policy errors recognisable and reduces everything else
// to a message that cannot echo the request URL (which may carry a token).
func classifyFetch(err error) error {
	switch {
	case errors.Is(err, ErrLCPPrivateAddress):
		return ErrLCPPrivateAddress
	case errors.Is(err, ErrLCPBadURL):
		return ErrLCPBadURL
	case errors.Is(err, errLCPRedirect):
		return fmt.Errorf("%w: redirect limit: %w", ErrLCPFetch, errLCPClientStatus)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: timed out", ErrLCPFetch)
	}
	return fmt.Errorf("%w: connection failed", ErrLCPFetch)
}

// dialContext resolves the host itself, rejects restricted addresses, and dials
// the validated IP so a DNS answer cannot change between check and connect.
func (f *LCPFulfiller) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, ErrLCPBadURL
	}
	var res interface {
		LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
	} = net.DefaultResolver
	if f.Resolver != nil {
		res = f.Resolver
	}
	ips, err := res.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("%w: address resolution failed", ErrLCPFetch)
	}
	dial := f.Dial
	if dial == nil {
		d := &net.Dialer{Timeout: 15 * time.Second}
		dial = d.DialContext
	}
	var policyErr, dialErr error
	for _, ip := range ips {
		if !f.AllowPrivateNetwork && restrictedIP(ip) {
			policyErr = ErrLCPPrivateAddress
			continue
		}
		c, err := dial(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return c, nil
		}
		dialErr = err
	}
	if dialErr != nil {
		return nil, fmt.Errorf("%w: connection failed", ErrLCPFetch)
	}
	return nil, policyErr
}

func mustCIDR(c string) *net.IPNet {
	_, n, err := net.ParseCIDR(c)
	if err != nil {
		panic(err)
	}
	return n
}

// restrictedNets are ranges that are never a public publication host: shared
// and benchmarking space, IETF protocol assignments, reserved/broadcast space,
// and the IPv6 forms that smuggle an IPv4 address past a naive check (NAT64,
// 6to4, Teredo, v4-compatible) or are deprecated site-local.
var restrictedNets = []*net.IPNet{
	mustCIDR("0.0.0.0/8"), mustCIDR("100.64.0.0/10"), mustCIDR("192.0.0.0/24"),
	mustCIDR("198.18.0.0/15"), mustCIDR("240.0.0.0/4"),
	mustCIDR("::/96"), mustCIDR("64:ff9b::/96"), mustCIDR("64:ff9b:1::/48"),
	mustCIDR("2002::/16"), mustCIDR("2001::/32"), mustCIDR("fec0::/10"), mustCIDR("100::/64"),
}

func restrictedIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range restrictedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ingestLCPL fulfills one License Document found in a source and returns a
// protected-EPUB work. The catalog file keeps the .lcpl's source path (so
// reconcile retention and file identity follow the license the operator put
// there) while its size and hash describe the fulfilled artifact in the cache.
// The source is only ever read. No metadata is parsed out of the fulfilled
// EPUB: its content documents are encrypted, so the title is the file name
// until a reader unlocks it.
func (s *Service) ingestLCPL(ctx context.Context, sc *LCPScan, sourceID string, connector sources.Connector, entry sources.Entry) (catalog.Work, []string, error) {
	doc, err := s.readBounded(ctx, connector, entry.Path, maxLCPLicenseBytes)
	if err != nil {
		return catalog.Work{}, nil, err
	}
	res, err := sc.Fulfill(ctx, doc)
	if err != nil {
		return catalog.Work{}, nil, err
	}
	d, _ := formats.Detect("fulfilled.epub")
	title := strings.TrimSuffix(entry.Name, filepath.Ext(entry.Name))
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(entry.Path), filepath.Ext(entry.Path))
	}
	edition := catalog.Edition{Title: title, State: catalog.Pending}
	work, warnings, err := s.assemble(ctx, sourceID, entry, d, edition, "", res.SHA256, res.Size, nil)
	if err != nil {
		return catalog.Work{}, warnings, err
	}
	warnings = append(warnings, SanitizeLocation(entry.Path)+": LCP license signature not verified server-side; readers verify and unlock")
	f := &work.Editions[0].Files[0]
	f.Protected, f.DRMScheme = true, "lcp"
	f.LicenseHREF = "/v1/files/" + url.PathEscape(f.ID) + "/license"
	return work, warnings, nil
}

// IsFulfilledLCP reports whether f is a fulfilled-license file: its bytes live
// in the LCP cache, not at Path in the source.
func IsFulfilledLCP(f catalog.File) bool {
	return f.DRMScheme == "lcp" && f.LicenseHREF != ""
}

// lcpLeaderHook runs in the flight leader right after it takes leadership.
// Nil in production.
var lcpLeaderHook func()

// lcpBeforeJoinHook lets a test run code between a cache miss and joining the
// flight. Nil in production.
var lcpBeforeJoinHook func()

// lcpAfterHashHook lets a test play the part of another process between a cache
// hit's hash and its final re-stat. Nil in production.
var lcpAfterHashHook func(path string)

func mustStat(fh *os.File) os.FileInfo {
	info, err := fh.Stat()
	if err != nil {
		return nil
	}
	return info
}
