package covers

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestCacheValidatesAndReusesImage(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		img := image.NewRGBA(image.Rect(0, 0, 2, 3))
		img.Set(0, 0, color.RGBA{R: 255, A: 255})
		w.Header().Set("Content-Type", "image/png")
		_ = png.Encode(w, img)
	}))
	defer server.Close()
	cache := New(t.TempDir())
	cache.Client = server.Client()
	cached, err := cache.Get(context.Background(), "edition/unsafe", server.URL+"/cover")
	if err != nil {
		t.Fatal(err)
	}
	if cached.MIME != "image/png" {
		t.Fatalf("wrong MIME: %s", cached.MIME)
	}
	if _, err := cache.Get(context.Background(), "edition/unsafe", server.URL+"/cover"); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("cache did not reuse image: %d", requests.Load())
	}
}
func TestCacheRejectsNonImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("not an image")) }))
	defer server.Close()
	cache := New(t.TempDir())
	cache.Client = server.Client()
	if _, err := cache.Get(context.Background(), "e", server.URL); err == nil {
		t.Fatal("accepted non-image cover")
	}
}
func TestPublicIPRejectsLocalAndPrivate(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "::1", "fe80::1", "0.0.0.0"} {
		if publicIP(net.ParseIP(raw)) {
			t.Errorf("accepted %s", raw)
		}
	}
	if !publicIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("rejected public address")
	}
}
