package source

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// fakeDriver is an in-memory Driver for tests.
type fakeDriver struct {
	proto   Protocol
	conn    bool
	closed  bool
	closeErr error
}

func (f *fakeDriver) Protocol() Protocol { return f.proto }
func (f *fakeDriver) Capabilities() Capability {
	return Capability{Browse: true, RangeRead: true}
}
func (f *fakeDriver) Connect(ctx context.Context, cfg Config) error {
	f.conn = true
	return nil
}
func (f *fakeDriver) List(ctx context.Context, path string) ([]Entry, error) {
	if !f.conn {
		return nil, ErrNotConnected
	}
	return []Entry{{Path: "/a.epub", Name: "a.epub", Format: "epub", Size: 10}}, nil
}
func (f *fakeDriver) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("data")), nil
}
func (f *fakeDriver) OpenRange(ctx context.Context, path string, off, n int64) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("data"[off : off+n])), nil
}
func (f *fakeDriver) Close() error { f.closed = true; return f.closeErr }

func testRegistry() *Registry {
	r := NewRegistry()
	r.Register(ProtoWebDAV, func() Driver { return &fakeDriver{proto: ProtoWebDAV} })
	r.Register(ProtoLocal, func() Driver { return &fakeDriver{proto: ProtoLocal} })
	return r
}

func TestRegistryUnknownProtocol(t *testing.T) {
	_, err := testRegistry().New(ProtoMega)
	if !errors.Is(err, ErrUnknownProtocol) {
		t.Fatalf("want ErrUnknownProtocol, got %v", err)
	}
}

func TestManagerAddListBind(t *testing.T) {
	m := NewManager(testRegistry())
	ctx := context.Background()
	cfg := Config{ID: "nas", Label: "NAS", Endpoint: "webdav://nas.local/books"}
	if err := m.Add(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	d, ok := m.Get("nas")
	if !ok {
		t.Fatal("source not registered")
	}
	ents, err := d.List(ctx, "/")
	if err != nil || len(ents) != 1 || ents[0].Format != "epub" {
		t.Fatalf("bad list: %v %v", ents, err)
	}
	if err := m.BindSync("nas", "acct-1"); err != nil {
		t.Fatal(err)
	}
	if err := m.BindSync("nope", "acct-1"); err == nil {
		t.Fatal("binding unknown source must fail")
	}
}

func TestManagerCloseAllDeterministic(t *testing.T) {
	m := NewManager(testRegistry())
	ctx := context.Background()
	m.Add(ctx, Config{ID: "a", Endpoint: "webdav://x"})
	m.Add(ctx, Config{ID: "b", Endpoint: "webdav://y"})
	if errs := m.CloseAll(); len(errs) != 0 {
		t.Fatalf("unexpected close errors: %v", errs)
	}
	if _, ok := m.Get("a"); ok {
		t.Fatal("source a should be gone")
	}
}

func TestCloseAllCollectsErrors(t *testing.T) {
	r := NewRegistry()
	r.Register(ProtoWebDAV, func() Driver {
		return &fakeDriver{proto: ProtoWebDAV, closeErr: errors.New("boom")}
	})
	m := NewManager(r)
	m.Add(context.Background(), Config{ID: "a", Endpoint: "webdav://x"})
	errs := m.CloseAll()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "a: boom") {
		t.Fatalf("want wrapped error, got %v", errs)
	}
}

func TestProtocolOf(t *testing.T) {
	cases := map[string]Protocol{
		"opds2://feed": ProtoOPDS2, "opds://feed": ProtoOPDS1,
		"webdav://h": ProtoWebDAV, "sftp://h": ProtoSFTP,
		"smb://h": ProtoSMB, "s3://b": ProtoS3, "/home/user/books": ProtoLocal,
	}
	for in, want := range cases {
		if got := protocolOf(in); got != want {
			t.Errorf("protocolOf(%q)=%s want %s", in, got, want)
		}
	}
}
