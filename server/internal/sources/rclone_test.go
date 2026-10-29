package sources

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeRclone installs a stand-in executable that floods stderr far beyond the
// bounded capture limit while emitting the requested payload on stdout.
func fakeRclone(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "rclone")
	script := "#!/bin/sh\nprintf 'payload'\nhead -c 1048576 /dev/zero | tr '\\000' 'e' >&2\nexit 0\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestRcloneOpenBoundsStderr(t *testing.T) {
	r, err := NewRclone(Config{Protocol: "sftp", Endpoint: "remote:bucket", RclonePath: fakeRclone(t)})
	if err != nil {
		t.Fatal(err)
	}
	rc, err := r.Open(context.Background(), "book.epub")
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if err := rc.Close(); err != nil {
		t.Fatal(err)
	}
	if string(b) != "payload" {
		t.Fatalf("wrong content: %q", b)
	}
}

func TestRcloneOpenRangeBoundsStderr(t *testing.T) {
	r, err := NewRclone(Config{Protocol: "sftp", Endpoint: "remote:bucket", RclonePath: fakeRclone(t)})
	if err != nil {
		t.Fatal(err)
	}
	rc, ok := r.(RangeConnector)
	if !ok {
		t.Fatal("rclone must expose range reads")
	}
	body, err := rc.OpenRange(context.Background(), "book.epub", 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := body.Close(); err != nil {
		t.Fatal(err)
	}
	if string(b) != "payload" {
		t.Fatalf("wrong content: %q", b)
	}
}
