package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/sources"
)

func TestScanTextIndexesInPlaceAndReportsUnknown(t *testing.T) {
	root := t.TempDir()
	content := []byte("Example Book\n\nA deterministic first paragraph with enough text.\n")
	if err := os.WriteFile(filepath.Join(root, "sample.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.xyz"), []byte("not a book"), 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	s := Service{Repository: repo, Now: func() time.Time { return now }}
	report, err := s.Scan(context.Background(), connector, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Files != 1 || len(report.Skipped) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	page, err := repo.Search(context.Background(), catalog.Query{Text: "sample"})
	if err != nil || len(page.Works) != 1 {
		t.Fatalf("catalog: %+v, %v", page, err)
	}
	e := page.Works[0].Editions[0]
	if len(e.Files) != 1 || e.Files[0].SHA256 == "" || e.Files[0].Path != "sample.txt" {
		t.Fatalf("file not indexed in place: %+v", e)
	}
	if _, err := os.Stat(filepath.Join(root, "sample.txt")); err != nil {
		t.Fatal("source file was modified or moved")
	}
	if e.State != catalog.NeedsReview && e.State != catalog.Partial {
		t.Fatalf("unverifiable text must not be identified automatically: %s", e.State)
	}
}

func TestScanSameContentAtDistinctPathsSharesOneWork(t *testing.T) {
	root := t.TempDir()
	content := []byte("Shared Book\n\nIdentical bytes in two locations.\n")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	s := Service{Repository: repo, Now: func() time.Time { return now }}
	report, err := s.ScanSource(context.Background(), "src", connector, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Files != 2 || report.Works != 1 {
		t.Fatalf("same content must join one work as two files: %+v", report)
	}
	page, err := repo.Search(context.Background(), catalog.Query{})
	if err != nil || page.Total != 1 {
		t.Fatalf("expected one work: %+v %v", page, err)
	}
	w := page.Works[0]
	if len(w.Editions) != 1 {
		t.Fatalf("same bytes must not conflate or fork editions: %+v", w.Editions)
	}
	files := w.Editions[0].Files
	if len(files) != 2 {
		t.Fatalf("expected both locations as distinct files: %+v", files)
	}
	if files[0].ID == files[1].ID {
		t.Fatalf("file identity must include source/path, not only content: %+v", files)
	}
	sum := sha256.Sum256(content)
	wantSHA := hex.EncodeToString(sum[:])
	for _, f := range files {
		if f.SHA256 != wantSHA {
			t.Fatalf("content hash not retained separately from identity: %+v", f)
		}
		if f.ID == f.SHA256 {
			t.Fatalf("file ID must not be the bare content hash: %+v", f)
		}
	}
}

func TestRescanKeepsStableWorkEditionFileIdentity(t *testing.T) {
	root := t.TempDir()
	content := []byte("Stable Book\n\nBody text that does not change.\n")
	if err := os.WriteFile(filepath.Join(root, "stable.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	clock := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	s := Service{Repository: repo, Now: func() time.Time { return clock }}
	first, err := s.ScanSource(context.Background(), "src", connector, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Files != 1 || first.Works != 1 {
		t.Fatalf("unexpected first scan report: %+v", first)
	}
	before, err := repo.Search(context.Background(), catalog.Query{})
	if err != nil || before.Total != 1 {
		t.Fatalf("expected one work: %+v %v", before, err)
	}
	workID := before.Works[0].ID
	editionID := before.Works[0].Editions[0].ID
	fileID := before.Works[0].Editions[0].Files[0].ID

	// Rescan with an advanced clock: identity must not move.
	clock = clock.Add(24 * time.Hour)
	second, err := s.ScanSource(context.Background(), "src", connector, "")
	if err != nil {
		t.Fatal(err)
	}
	if second.Files != 1 || second.Works != 1 {
		t.Fatalf("unexpected rescan report: %+v", second)
	}
	after, err := repo.Search(context.Background(), catalog.Query{})
	if err != nil || after.Total != 1 {
		t.Fatalf("rescan forked works: %+v %v", after, err)
	}
	w := after.Works[0]
	if w.ID != workID || len(w.Editions) != 1 || w.Editions[0].ID != editionID {
		t.Fatalf("rescan changed work/edition identity: before=%s/%s after=%s/%+v", workID, editionID, w.ID, w.Editions)
	}
	files := w.Editions[0].Files
	if len(files) != 1 || files[0].ID != fileID {
		t.Fatalf("rescan duplicated files: %+v", files)
	}
}

func TestPrefixScanNeverPrunesOutsideListingScope(t *testing.T) {
	root := t.TempDir()
	content := []byte("Prefix reconciliation boundary.\n")
	if err := os.WriteFile(filepath.Join(root, "outside.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "inside.txt"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	service := Service{Repository: repo}
	if report, err := service.ScanSource(context.Background(), "prefix-source", connector, ""); err != nil || report.Files != 2 {
		t.Fatalf("initial scan: report=%+v err=%v", report, err)
	}
	page, err := repo.Search(context.Background(), catalog.Query{})
	if err != nil || page.Total != 1 {
		t.Fatalf("initial catalog: %+v err=%v", page, err)
	}
	outsideID := ""
	for _, file := range page.Works[0].Editions[0].Files {
		if file.Path == "outside.txt" {
			outsideID = file.ID
		}
	}
	if err := os.Remove(filepath.Join(root, "outside.txt")); err != nil {
		t.Fatal(err)
	}
	report, err := service.ScanSource(context.Background(), "prefix-source", connector, "sub")
	if err != nil || report.Files != 1 || report.Removed != 0 {
		t.Fatalf("prefix scan: report=%+v err=%v", report, err)
	}
	if _, _, file, err := repo.FindFile(context.Background(), outsideID); err != nil || file.Path != "outside.txt" {
		t.Fatalf("prefix scan used incomplete listing as deletion evidence: file=%+v err=%v", file, err)
	}
}

func TestFullScanRetainsListedFileWhenIngestFails(t *testing.T) {
	root := t.TempDir()
	entry := sources.Entry{Path: "book.txt", Name: "book.txt", Size: 32}
	if err := os.WriteFile(filepath.Join(root, entry.Path), []byte("Still listed but temporarily unreadable.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	service := Service{Repository: repo}
	if report, err := service.ScanSource(context.Background(), "flaky-source", connector, ""); err != nil || report.Files != 1 {
		t.Fatalf("initial scan: report=%+v err=%v", report, err)
	}
	storedID := fileID("flaky-source", entry.Path)
	failing := &secretConnector{entries: []sources.Entry{entry}, openErr: errors.New("temporary open failure"), recursive: true}
	report, err := service.ScanSource(context.Background(), "flaky-source", failing, "")
	if err != nil || report.Files != 0 || report.Removed != 0 || len(report.Skipped) != 1 {
		t.Fatalf("failed-ingest rescan: report=%+v err=%v", report, err)
	}
	if _, _, file, err := repo.FindFile(context.Background(), storedID); err != nil || file.Path != entry.Path {
		t.Fatalf("listed file was deleted after transient ingest failure: file=%+v err=%v", file, err)
	}
	// A non-recursive connector never authorizes source-wide pruning, even with
	// an empty listing.
	nonRecursive := &secretConnector{}
	report, err = service.ScanSource(context.Background(), "flaky-source", nonRecursive, "")
	if err != nil || report.Removed != 0 {
		t.Fatalf("non-recursive scan: report=%+v err=%v", report, err)
	}
	if _, _, _, err := repo.FindFile(context.Background(), storedID); err != nil {
		t.Fatalf("non-recursive empty listing deleted prior file: %v", err)
	}
}

func TestScanReportCountsWorksTouchedByThisScan(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("New Book\n\nText.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	// Pre-existing catalog content must not inflate the per-scan work count.
	pre := catalog.Work{ID: "pre-existing", CanonicalTitle: "Old", Editions: []catalog.Edition{{ID: "pe", Title: "Old", State: catalog.Identified}}}
	if err := repo.UpsertWork(context.Background(), pre); err != nil {
		t.Fatal(err)
	}
	s := Service{Repository: repo}
	report, err := s.Scan(context.Background(), connector, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Works != 1 || report.Files != 1 {
		t.Fatalf("report must count works touched by this scan, not catalog total: %+v", report)
	}
	page, err := repo.Search(context.Background(), catalog.Query{})
	if err != nil || page.Total != 2 {
		t.Fatalf("catalog should hold both works: %+v %v", page, err)
	}
}

func TestIdentityIDsStableAgainstMetadataImprovement(t *testing.T) {
	base := catalog.Edition{Title: "Rough Title", Authors: []string{"Unknown"}}
	improved := catalog.Edition{Title: "Corrected Title", Authors: []string{"Actual Author"}, Publisher: "Publisher", Published: "2020", Language: "en"}
	hash := strings.Repeat("ab", 32)
	if identityID(base, hash) != identityID(improved, hash) {
		t.Fatal("edition identity must not change when mutable metadata improves")
	}
	withISBN := catalog.Edition{Title: "Anything", Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "9780441478125"}}}
	other := catalog.Edition{Title: "Different Title", Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "9780441478125"}}}
	if identityID(withISBN, hash) != identityID(other, strings.Repeat("cd", 32)) {
		t.Fatal("ISBN-keyed edition identity must not depend on metadata or content")
	}
	if identityID(withISBN, hash) == identityID(base, hash) {
		t.Fatal("ISBN and content fallback identities must not collide")
	}
	if fileID("src", "a.txt") == fileID("src", "b.txt") || fileID("src", "a.txt") == fileID("other", "a.txt") {
		t.Fatal("file identity must include source and path")
	}
	if fileID("src", "a.txt") != fileID("src", "a.txt") {
		t.Fatal("file identity must be deterministic")
	}
}

func TestSanitizeErrorStripsURLCredentials(t *testing.T) {
	secretURL := "https://user:s3cr3t-pw@example.com/books?token=abc123#frag"
	out := SanitizeError(&url.Error{Op: "Get", URL: secretURL, Err: errors.New("connection refused")})
	for _, secret := range []string{"s3cr3t-pw", "abc123", "user@", "frag"} {
		if strings.Contains(out, secret) {
			t.Fatalf("sanitized error leaked %q: %q", secret, out)
		}
	}
	for _, kept := range []string{"Get", "https://example.com/books", "connection refused"} {
		if !strings.Contains(out, kept) {
			t.Fatalf("sanitized error lost useful context %q: %q", kept, out)
		}
	}
	// A wrapped *url.Error cause must be sanitized as well.
	nested := SanitizeError(&url.Error{Op: "Get", URL: "https://example.com/a", Err: &url.Error{Op: "Get", URL: secretURL, Err: errors.New("x")}})
	if strings.Contains(nested, "abc123") || strings.Contains(nested, "s3cr3t-pw") {
		t.Fatalf("nested url.Error leaked credentials: %q", nested)
	}
	// Non-URL errors pass through unchanged.
	if got := SanitizeError(errors.New("plain failure")); got != "plain failure" {
		t.Fatalf("non-URL error altered: %q", got)
	}
	if got := SanitizeError(nil); got != "" {
		t.Fatalf("nil error rendered as %q", got)
	}
}

// secretConnector fails Open (always, or on a chosen call) with a *url.Error
// carrying credentials, emulating an authenticated HTTP source.
type secretConnector struct {
	entries   []sources.Entry
	openErr   error
	failOn    int // 1-based Open call that fails; 0 fails every call
	opens     int
	recursive bool
}

func (c *secretConnector) List(context.Context, string) ([]sources.Entry, error) {
	return c.entries, nil
}
func (c *secretConnector) Open(context.Context, string) (io.ReadCloser, error) {
	c.opens++
	if c.failOn == 0 || c.opens == c.failOn {
		return nil, c.openErr
	}
	return io.NopCloser(strings.NewReader("Secret Book\n\nReadable body text.\n")), nil
}
func (c *secretConnector) Capabilities() sources.Capabilities {
	return sources.Capabilities{RecursiveList: c.recursive}
}
func (c *secretConnector) Close() error { return nil }

func secretURLError() error {
	return &url.Error{
		Op:  "Get",
		URL: "https://user:s3cr3t-pw@example.com/books/book.txt?token=abc123#frag",
		Err: errors.New("connection refused"),
	}
}

func TestScanSanitizesConnectorErrorsInReport(t *testing.T) {
	entry := sources.Entry{Path: "book.txt", Name: "book.txt", Size: 32}
	repo := catalog.NewMemory()
	s := Service{Repository: repo}

	// Hash open fails: the file is skipped and the error lands in the report.
	failing := &secretConnector{entries: []sources.Entry{entry}, openErr: secretURLError()}
	report, err := s.Scan(context.Background(), failing, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Files != 0 || len(report.Skipped) != 1 {
		t.Fatalf("expected one skipped file: %+v", report)
	}
	assertNoSecret(t, report.Skipped[0])
	if !strings.Contains(report.Skipped[0], "book.txt") || !strings.Contains(report.Skipped[0], "https://example.com/books/book.txt") {
		t.Fatalf("sanitized skip lost useful context: %q", report.Skipped[0])
	}

	// Hash open succeeds but the metadata re-read fails: a report warning.
	flaky := &secretConnector{entries: []sources.Entry{entry}, openErr: secretURLError(), failOn: 2}
	report, err = s.Scan(context.Background(), flaky, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Warnings) != 1 {
		t.Fatalf("expected one metadata warning: %+v", report)
	}
	assertNoSecret(t, report.Warnings[0])
}

func assertNoSecret(t *testing.T, s string) {
	t.Helper()
	for _, secret := range []string{"s3cr3t-pw", "abc123", "user@", "#frag", "token="} {
		if strings.Contains(s, secret) {
			t.Fatalf("report entry leaked %q: %q", secret, s)
		}
	}
}

func TestSanitizeLocationStripsOnlyAbsoluteURLCredentials(t *testing.T) {
	got := SanitizeLocation("https://user:pw@host/book.epub?token=secret#frag")
	if got != "https://host/book.epub" {
		t.Fatalf("presigned URL not sanitized: %q", got)
	}
	// Local paths keep '?' and '#' literally; they are not URLs.
	for _, local := range []string{"books/what?.epub", "books/c#sharp.epub", "plain.txt", "/abs/dir/a b.txt"} {
		if got := SanitizeLocation(local); got != local {
			t.Fatalf("local path altered: %q -> %q", local, got)
		}
	}
}

func TestSanitizationFailsClosedForMalformedAndOpaqueHTTPURLs(t *testing.T) {
	for _, value := range []string{
		"https://user:pw@%zz/book.epub?token=secret",
		"https:user:pw@host/book.epub?token=secret",
	} {
		if got := SanitizeLocation(value); strings.Contains(got, "pw") || strings.Contains(got, "secret") {
			t.Fatalf("location sanitizer leaked %q as %q", value, got)
		}
		if got := SanitizeError(&url.Error{Op: "Get", URL: value, Err: errors.New("failed")}); strings.Contains(got, "pw") || strings.Contains(got, "secret") {
			t.Fatalf("error sanitizer leaked %q as %q", value, got)
		}
	}
}

func TestScanSanitizesURLEntryLocationsInReport(t *testing.T) {
	entry := sources.Entry{
		Path: "https://user:pw@host/book.epub?token=secret#frag",
		Name: "book.epub",
		Size: 32,
	}
	assertLocationSafe := func(s string) {
		t.Helper()
		for _, secret := range []string{"user:pw", "pw@", "token=secret", "#frag"} {
			if strings.Contains(s, secret) {
				t.Fatalf("report entry leaked %q: %q", secret, s)
			}
		}
		if !strings.Contains(s, "https://host/book.epub") {
			t.Fatalf("report entry lost sanitized location: %q", s)
		}
	}
	s := Service{Repository: catalog.NewMemory()}

	// Open fails outright: the file is skipped with a sanitized location.
	failing := &secretConnector{entries: []sources.Entry{entry}, openErr: errors.New("connection refused")}
	report, err := s.Scan(context.Background(), failing, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Skipped) != 1 {
		t.Fatalf("expected one skipped file: %+v", report)
	}
	assertLocationSafe(report.Skipped[0])

	// Hash open succeeds but the metadata re-read fails: a sanitized warning.
	flaky := &secretConnector{entries: []sources.Entry{entry}, openErr: errors.New("connection refused"), failOn: 2}
	report, err = s.Scan(context.Background(), flaky, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Warnings) != 1 {
		t.Fatalf("expected one metadata warning: %+v", report)
	}
	assertLocationSafe(report.Warnings[0])
}

func TestScanSourceListErrorPreservesSemantics(t *testing.T) {
	// The returned error keeps its *url.Error identity for callers; the
	// admin-facing ScanJob.Error sanitization happens at the job boundary.
	c := &listFailConnector{err: secretURLError()}
	s := Service{Repository: catalog.NewMemory()}
	_, err := s.Scan(context.Background(), c, "")
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("list error lost its type: %T %v", err, err)
	}
	if got := SanitizeError(err); strings.Contains(got, "s3cr3t-pw") || strings.Contains(got, "abc123") {
		t.Fatalf("SanitizeError leaked credentials for job record: %q", got)
	}
}

type listFailConnector struct{ err error }

func (c *listFailConnector) List(context.Context, string) ([]sources.Entry, error) {
	return nil, c.err
}
func (c *listFailConnector) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, c.err
}
func (c *listFailConnector) Capabilities() sources.Capabilities { return sources.Capabilities{} }
func (c *listFailConnector) Close() error                       { return nil }

func TestReadBoundedRejectsOverflow(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "book.txt"), []byte("123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	s := Service{}
	data, err := s.readBounded(context.Background(), connector, "book.txt", 9)
	if err != nil || len(data) != 9 {
		t.Fatalf("exact-limit read failed: len=%d err=%v", len(data), err)
	}
	if _, err := s.readBounded(context.Background(), connector, "book.txt", 8); err == nil {
		t.Fatal("over-limit source must be rejected rather than truncated")
	}
}
