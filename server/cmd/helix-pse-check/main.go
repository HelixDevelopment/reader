// Command helix-pse-check reads a live OPDS-PSE advertisement the way a reader
// client must: parse the feed with the extension's own rules, then fetch every
// page the advertisement promised. A server that invents a page count fails
// here on the last page, not in a unit test of its own output.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/helixreader/helix/core/pkg/pse"
)

// maxPagesPerLink bounds one stream so a mis-advertised count cannot make this
// client download an archive's worth of pages before reporting.
const maxPagesPerLink = 200

type result struct {
	Status   string   `json:"status"`
	FeedURL  string   `json:"feed_url"`
	Links    int      `json:"links"`
	Pages    int      `json:"pages"`
	Failures []string `json:"failures"`
}

type response struct {
	request *http.Request
	status  int
	header  http.Header
	body    []byte
}

func (r *response) text() string { return strings.TrimSpace(string(r.body)) }

func main() {
	base := flag.String("base", "http://127.0.0.1:28080", "server base URL")
	query := flag.String("q", "", "catalog search query for the acquisition feed")
	flag.Parse()

	feedURL := *base + "/opds/1.2/publications"
	if *query != "" {
		feedURL += "?q=" + url.QueryEscape(*query)
	}
	out := result{Status: "FAIL", FeedURL: feedURL}
	// Credentials come from the environment like every other client of this
	// server: a password in argv would be visible in the process list and in
	// the verifier's command line.
	user, pass := os.Getenv("HELIX_OPDS_USERNAME"), os.Getenv("HELIX_OPDS_PASSWORD")
	if user == "" || pass == "" {
		record(&out, "HELIX_OPDS_USERNAME and HELIX_OPDS_PASSWORD must be set")
		emit(out)
		return
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 15 * time.Second}

	feed, err := fetch(client, feedURL, user, pass)
	if err != nil {
		record(&out, "feed request: "+err.Error())
		emit(out)
		return
	}
	if feed.status != http.StatusOK {
		record(&out, fmt.Sprintf("feed returned %d, want 200 (%s)", feed.status, feed.text()))
		emit(out)
		return
	}
	links, err := pse.ParseAcquisitionLinks(feed.body)
	if err != nil {
		record(&out, "feed rejected by the OPDS-PSE parser: "+err.Error())
		emit(out)
		return
	}
	out.Links = len(links)
	// pse:lastRead and pse:lastReadDate are parsed and validated but never
	// produced, because the server holds no per-file reading position.
	if bytes.Contains(feed.body, []byte("lastRead")) {
		record(&out, "feed carries a pse:lastRead attribute the server cannot support")
	}

	for _, link := range links {
		if link.Count > maxPagesPerLink {
			record(&out, fmt.Sprintf("link %s advertises %d pages, above this checker's cap", link.Href, link.Count))
			continue
		}
		etags := map[string]bool{}
		for page := 0; page < link.Count; page++ {
			pageURL, err := link.PageURL(page, 0)
			if err != nil {
				record(&out, err.Error())
				break
			}
			resp, err := fetchResolved(client, feed, pageURL, user, pass)
			if err != nil {
				record(&out, err.Error())
				break
			}
			out.Pages++
			if err := verifyPage(link, page, resp, etags); err != nil {
				record(&out, err.Error())
			}
		}
		// The advertised count must be the exact boundary: page Count is one
		// past the archive, so a truthful advertisement ends in a refusal that
		// states the range it just proved.
		over := strings.ReplaceAll(link.Href, pse.PageNumberTemplate, strconv.Itoa(link.Count))
		resp, err := fetchResolved(client, feed, over, user, pass)
		if err != nil {
			record(&out, err.Error())
			continue
		}
		if resp.status != http.StatusNotFound {
			record(&out, fmt.Sprintf("page %d of a %d-page stream returned %d, want 404", link.Count, link.Count, resp.status))
		} else if want := "0.." + strconv.Itoa(link.Count-1); !strings.Contains(resp.text(), want) {
			record(&out, fmt.Sprintf("refusal past the advertised count must state the valid range %q, got %q", want, resp.text()))
		}
	}

	// A page image is catalog content: an anonymous request must not receive it.
	if first, err := links[0].PageURL(0, 0); err == nil {
		resp, err := fetchResolved(client, feed, first, "", "")
		if err != nil {
			record(&out, "anonymous page request: "+err.Error())
		} else if resp.status != http.StatusUnauthorized {
			record(&out, fmt.Sprintf("anonymous page request returned %d, want 401", resp.status))
		}
	}

	if len(out.Failures) == 0 {
		out.Status = "PASS"
	}
	emit(out)
}

// verifyPage checks one delivered page against what its advertisement claimed.
func verifyPage(link pse.Link, page int, resp *response, etags map[string]bool) error {
	if resp.status != http.StatusOK {
		return fmt.Errorf("page %d returned %d, want 200 (%s)", page, resp.status, resp.text())
	}
	if got := resp.header.Get("Content-Type"); got != link.Type {
		return fmt.Errorf("page %d content type %q, want the advertised %q", page, got, link.Type)
	}
	if !matchesType(link.Type, resp.body) {
		return fmt.Errorf("page %d payload %q does not start with a %s signature", page, resp.body, link.Type)
	}
	if declared := resp.header.Get("Content-Length"); declared != "" {
		if n, err := strconv.Atoi(declared); err != nil || int64(n) != int64(len(resp.body)) {
			return fmt.Errorf("page %d declares %s bytes and delivers %d", page, declared, len(resp.body))
		}
	}
	tag := resp.header.Get("ETag")
	if tag == "" {
		return fmt.Errorf("page %d has no ETag, so a client cannot revalidate a cached page", page)
	}
	if etags[tag] {
		return fmt.Errorf("page %d repeats the ETag of an earlier page: %s", page, tag)
	}
	etags[tag] = true
	if got := resp.header.Get("Accept-Ranges"); got != "none" {
		return fmt.Errorf("page %d offers Accept-Ranges %q; a rendered page is not a byte range of the archive", page, got)
	}
	if got := resp.header.Get("Cache-Control"); got == "" {
		return fmt.Errorf("page %d has no Cache-Control", page)
	}
	if !strings.Contains(resp.header.Get("Vary"), "Authorization") {
		return fmt.Errorf("page %d does not vary on Authorization, so a shared cache could serve it across permissions", page)
	}
	return nil
}

func matchesType(mediaType string, payload []byte) bool {
	switch mediaType {
	case "image/png":
		return bytes.HasPrefix(payload, []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
	case "image/jpeg":
		return bytes.HasPrefix(payload, []byte{0xff, 0xd8, 0xff})
	case "image/gif":
		return bytes.HasPrefix(payload, []byte("GIF87a")) || bytes.HasPrefix(payload, []byte("GIF89a"))
	}
	return false
}

func fetchResolved(client *http.Client, against *response, href, user, pass string) (*response, error) {
	target, err := against.request.URL.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("href %q is not resolvable against the feed URL: %v", href, err)
	}
	return fetch(client, target.String(), user, pass)
}

func fetch(client *http.Client, rawURL, user, pass string) (*response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	return &response{request: req, status: resp.StatusCode, header: resp.Header, body: body}, nil
}

func record(out *result, message string) {
	out.Failures = append(out.Failures, message)
}

func emit(out result) {
	encoded, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(encoded))
	if out.Status != "PASS" {
		os.Exit(1)
	}
	os.Exit(0)
}
