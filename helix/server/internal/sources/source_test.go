package sources

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalListOpenAndTraversal(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "book.epub"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := NewLocal(Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if c.Capabilities().Watch {
		t.Fatal("local connector advertised watch support without a watch service")
	}
	entries, err := c.List(context.Background(), "")
	if err != nil || len(entries) != 1 {
		t.Fatalf("list: %+v, %v", entries, err)
	}
	r, err := c.Open(context.Background(), "sub/book.epub")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(r)
	r.Close()
	if string(b) != "content" {
		t.Fatalf("wrong content: %q", b)
	}
	for _, p := range []string{"../secret", "/etc/passwd", "sub/../../secret"} {
		if _, err := c.Open(context.Background(), p); err == nil {
			t.Errorf("traversal %q accepted", p)
		}
	}
}

func TestDefaultRegistryCoverage(t *testing.T) {
	r := DefaultRegistry()
	need := []string{"local", "http", "https", "webdav", "webdavs", "opds", "opds2", "ftp", "ftps", "sftp", "smb", "smb-mount", "nfs-mount", "s3", "google-drive", "dropbox", "onedrive", "box", "pcloud", "mega", "nextcloud"}
	got := map[string]bool{}
	for _, p := range r.Protocols() {
		got[p] = true
	}
	for _, p := range need {
		if !got[p] {
			t.Errorf("protocol %s not registered", p)
		}
	}
}
