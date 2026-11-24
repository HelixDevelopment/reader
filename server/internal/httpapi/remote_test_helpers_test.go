package httpapi

// Minimal helpers for remote_range_test.go. They are copied (under unique
// remoteTest* names, so a later file with the generic names cannot collide)
// from the live-test helpers that are not yet tracked on main.

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

// remoteTestFreeAddr returns a currently unused 127.0.0.1:port address.
func remoteTestFreeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

// remoteTestLogBuffer accumulates a child process output for failure messages.
type remoteTestLogBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *remoteTestLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *remoteTestLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

// remoteTestWaitWebDAV polls GET base/name until it returns 200 with the
// expected sha256, so readiness implies a working path.
func remoteTestWaitWebDAV(t *testing.T, base, name, wantSHA string, log *remoteTestLogBuffer) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(base + "/" + name)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			t.Fatalf("webdav readiness GET %s: read body: %v", name, readErr)
		}
		if resp.StatusCode == http.StatusOK {
			if got := fmt.Sprintf("%x", sha256.Sum256(body)); got == wantSHA {
				return
			}
			t.Fatalf("webdav readiness GET %s: digest mismatch", name)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("webdav target at %s never served %s: log:\n%s", base, name, log.String())
}
