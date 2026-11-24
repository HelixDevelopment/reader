package ingest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/lcp"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/ingest/lcpfixture"
)

// bumpRepo simulates a concurrent scan that merges into the target work and 150
// others while GC reads the catalog: at the start of every keyset pass, after
// page 1 is read, the target's updated_at jumps to the head of the ordering. A
// reader that pages by updated_at skips it; a single-snapshot read cannot.
type bumpRepo struct {
	catalog.Repository
	target string
	n      int
}

func (b *bumpRepo) Search(ctx context.Context, q catalog.Query) (catalog.Page, error) {
	if q.After == nil {
		for i := 0; i < 150; i++ {
			fw, _ := b.Repository.GetWork(ctx, fmt.Sprintf("filler-%03d", i))
			fw.UpdatedAt = time.Now().Add(time.Duration(b.n)*time.Hour + 30*time.Minute)
			b.Repository.UpsertWork(ctx, fw)
		}
	}
	p, err := b.Repository.Search(ctx, q)
	if q.After == nil {
		w, gerr := b.Repository.GetWork(ctx, b.target)
		if gerr != nil {
			return p, gerr
		}
		b.n++
		w.UpdatedAt = time.Now().Add(time.Duration(b.n) * time.Hour)
		if uerr := b.Repository.UpsertWork(ctx, w); uerr != nil {
			return p, uerr
		}
	}
	return p, err
}

// F2: GC must not delete an artifact the catalog still references, whatever the
// concurrent updates do to paging order.
func TestGCKeepsReferencedArtifactWhenWorkMovesDuringRead(t *testing.T) {
	s, c, _, _, _ := gcRig(t, nil) // CacheGrace = -1 (artifact counts as old)
	scanLCPL(t, s, c)
	page, _ := s.Repository.Search(context.Background(), catalog.Query{})
	w := page.Works[0]
	sha := w.Editions[0].Files[0].SHA256
	for i := 0; i < 150; i++ {
		fw := catalog.Work{ID: fmt.Sprintf("filler-%03d", i), UpdatedAt: time.Now().Add(time.Minute),
			Editions: []catalog.Edition{{ID: fmt.Sprintf("fe-%03d", i), Title: "x", State: catalog.Pending}}}
		if err := s.Repository.UpsertWork(context.Background(), fw); err != nil {
			t.Fatal(err)
		}
	}
	repo := &bumpRepo{Repository: s.Repository, target: w.ID}
	if _, err := s.LCP.GC(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.LCP.ArtifactPath(sha)); err != nil {
		t.Fatalf("REFERENCED artifact %s deleted by GC (catalog still references it): %v", sha[:12], err)
	}
}

// staleRefs returns an empty reference set on its first read (a snapshot taken
// before the work was merged) and the truth afterwards.
type staleRefs struct {
	catalog.Repository
	calls atomic.Int32
}

func (r *staleRefs) ReferencedLCPArtifacts(ctx context.Context) (map[string]bool, error) {
	if r.calls.Add(1) == 1 {
		return map[string]bool{}, nil
	}
	return r.Repository.ReferencedLCPArtifacts(ctx)
}

// F2: "still unreferenced" is re-verified immediately before each unlink.
func TestGCReverifiesUnreferencedBeforeUnlink(t *testing.T) {
	s, c, _, _, _ := gcRig(t, nil)
	scanLCPL(t, s, c)
	page, _ := s.Repository.Search(context.Background(), catalog.Query{})
	sha := page.Works[0].Editions[0].Files[0].SHA256
	if _, err := s.LCP.GC(context.Background(), &staleRefs{Repository: s.Repository}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.LCP.ArtifactPath(sha)); err != nil {
		t.Fatalf("artifact deleted off a stale reference snapshot: %v", err)
	}
}

// F1: cap pressure never deletes a file younger than CacheGrace.
func TestGCCapNeverEvictsInsideGraceWindow(t *testing.T) {
	s, _, _, _, fx := gcRig(t, func(f *LCPFulfiller) { f.CacheGrace = time.Hour; f.CacheMaxBytes = 1 })
	res, err := s.LCP.Fulfill(context.Background(), fx.License) // scan B fulfilled, not merged yet
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LCP.GC(context.Background(), s.Repository); err != nil { // scan A's GC
		t.Fatal(err)
	}
	if _, err := os.Stat(s.LCP.ArtifactPath(res.SHA256)); err != nil {
		t.Fatalf("in-flight artifact (age < CacheGrace) evicted before its scan merged it: %v", err)
	}
}

// F3: same inode, same size, restored mtime - only ctime moves.
func TestOpenVerifiedDetectsSameSizeRewriteWithRestoredMtime(t *testing.T) {
	s, _, _, _, fx := gcRig(t, nil)
	res, err := s.LCP.Fulfill(context.Background(), fx.License)
	if err != nil {
		t.Fatal(err)
	}
	fh, info, err := s.LCP.OpenVerified(res.SHA256, res.Size)
	if err != nil {
		t.Fatal(err)
	}
	fh.Close()
	p := s.LCP.ArtifactPath(res.SHA256)
	w, err := os.OpenFile(p, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	w.WriteAt([]byte("TAMPERED"), 100)
	w.Close()
	os.Chtimes(p, info.ModTime(), info.ModTime())
	fh2, _, err := s.LCP.OpenVerified(res.SHA256, res.Size)
	if err == nil {
		b, _ := io.ReadAll(fh2)
		fh2.Close()
		t.Fatalf("tampered bytes served as verified: %q at offset 100", b[100:108])
	}
	if !errors.Is(err, ErrLCPArtifactMismatch) {
		t.Fatalf("err = %v", err)
	}
}

// countConn counts bytes the CLIENT reads off the socket.
type countConn struct {
	net.Conn
	n *atomic.Int64
}

func (c *countConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.n.Add(int64(n))
	return n, err
}

// F4: deterministic version of the stated-length cap: bounded by what the client
// read, not by how much the server managed to push into socket buffers.
func TestLCPFulfillCapsBodyAtStatedLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl := w.(http.Flusher)
		for i := 0; i < 2048; i++ {
			if _, err := w.Write(make([]byte, 1024)); err != nil {
				return
			}
			fl.Flush()
		}
	}))
	defer srv.Close()
	fx := lcpfixture.Build(t, srv.URL+"/p.epub", nil)
	var read atomic.Int64
	f := newFulfiller(t)
	f.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		c, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &countConn{Conn: c, n: &read}, nil
	}
	stated := int64(fx.Link["length"].(int))
	_, err := f.Fulfill(context.Background(), fx.License)
	if !errors.Is(err, lcp.ErrPublicationMismatch) {
		t.Fatalf("err = %v", err)
	}
	// stated bytes + one extra probe byte + HTTP framing/read-ahead slack.
	if got := read.Load(); got > stated+64<<10 {
		t.Fatalf("client read %d bytes though the link states %d", got, stated)
	}
}
