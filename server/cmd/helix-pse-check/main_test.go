package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/pse"
)

func advertised() pse.Link {
	return pse.Link{Rel: pse.StreamRel, Type: "image/png", Href: "/opds-pse/stream/f1/{pageNumber}", Count: 2}
}

// page returns the response a correct server would send for one page, so each
// case below mutates exactly one property the advertisement promises.
func page(mutate func(http.Header, *[]byte)) *response {
	header := http.Header{
		"Content-Type":   []string{"image/png"},
		"Content-Length": []string{strconv.Itoa(len(pngBody))},
		"Etag":           []string{`"$sha256-abc$page-1"`},
		"Accept-Ranges":  []string{"none"},
		"Cache-Control":  []string{"private, max-age=86400"},
		"Vary":           []string{"Authorization"},
	}
	body := append([]byte(nil), pngBody...)
	if mutate != nil {
		mutate(header, &body)
	}
	return &response{status: http.StatusOK, header: header, body: body}
}

var pngBody = append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, []byte("-page")...)

func TestVerifyPageAcceptsAHonestPage(t *testing.T) {
	if err := verifyPage(advertised(), 1, page(nil), map[string]bool{}); err != nil {
		t.Fatalf("a correct page was rejected: %v", err)
	}
}

func TestVerifyPageRejectsEveryBrokenDelivery(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(http.Header, *[]byte)
		want   string
	}{
		{name: "payload is not the image", mutate: func(h http.Header, b *[]byte) { (*b)[0] = 'X' }, want: "does not start with a image/png signature"},
		{name: "wrong type", mutate: func(h http.Header, b *[]byte) { h.Set("Content-Type", "application/zip") }, want: "content type"},
		{name: "short body", mutate: func(h http.Header, b *[]byte) { *b = (*b)[:len(pngBody)-2] }, want: "declares"},
		{name: "no etag", mutate: func(h http.Header, b *[]byte) { h.Del("Etag") }, want: "no ETag"},
		{name: "rangeable", mutate: func(h http.Header, b *[]byte) { h.Set("Accept-Ranges", "bytes") }, want: "not a byte range"},
		{name: "uncached", mutate: func(h http.Header, b *[]byte) { h.Del("Cache-Control") }, want: "no Cache-Control"},
		{name: "shared cache", mutate: func(h http.Header, b *[]byte) { h.Del("Vary") }, want: "does not vary on Authorization"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyPage(advertised(), 1, page(tc.mutate), map[string]bool{})
			if err == nil {
				t.Fatal("broken delivery accepted")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

// Two pages sharing an ETag would make a client's cache serve page 0 forever.
func TestVerifyPageRejectsRepeatedETag(t *testing.T) {
	seen := map[string]bool{}
	if err := verifyPage(advertised(), 0, page(nil), seen); err != nil {
		t.Fatalf("first page rejected: %v", err)
	}
	err := verifyPage(advertised(), 1, page(nil), seen)
	if err == nil {
		t.Fatal("a second page with the same ETag must not be accepted")
	}
	if !strings.Contains(err.Error(), "repeats the ETag") {
		t.Errorf("error = %v, want one stating the repeated ETag", err)
	}
}

func TestMatchesType(t *testing.T) {
	cases := []struct {
		media string
		want  bool
	}{
		{media: "image/png", want: true},
		{media: "image/jpeg", want: false},
	}
	for _, tc := range cases {
		if got := matchesType(tc.media, pngBody); got != tc.want {
			t.Errorf("matchesType(%q, png) = %v, want %v", tc.media, got, tc.want)
		}
	}
	if !matchesType("image/jpeg", []byte{0xff, 0xd8, 0xff, 'x'}) {
		t.Error("a jpeg signature must match image/jpeg")
	}
	if matchesType("application/zip", pngBody) {
		t.Error("a type OPDS-PSE does not permit must never match")
	}
}
