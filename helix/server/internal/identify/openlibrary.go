package identify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

// OpenLibrary resolves works/editions through the public Search API. BaseURL
// is configurable for deterministic tests and compatible mirrors.
type OpenLibrary struct {
	BaseURL string
	Client  *http.Client
}

func (o *OpenLibrary) Name() string { return "openlibrary" }

type olResponse struct {
	Docs []struct {
		Title            string   `json:"title"`
		AuthorName       []string `json:"author_name"`
		ISBN             []string `json:"isbn"`
		Publisher        []string `json:"publisher"`
		Language         []string `json:"language"`
		FirstPublishYear int      `json:"first_publish_year"`
		Key              string   `json:"key"`
		CoverI           int      `json:"cover_i"`
	} `json:"docs"`
}

func (o *OpenLibrary) Resolve(ctx context.Context, s Snapshot) ([]Candidate, error) {
	base := strings.TrimRight(o.BaseURL, "/")
	if base == "" {
		base = "https://openlibrary.org"
	}
	q := url.Values{}
	isbnValue := ""
	for _, id := range s.Embedded.Identifiers {
		if strings.HasPrefix(strings.ToLower(id.Scheme), "isbn") {
			isbnValue = id.Value
			break
		}
	}
	if isbnValue != "" {
		q.Set("isbn", isbnValue)
	} else {
		if s.Embedded.Title == "" {
			return nil, nil
		}
		q.Set("title", s.Embedded.Title)
		if len(s.Embedded.Authors) > 0 {
			q.Set("author", s.Embedded.Authors[0])
		}
	}
	q.Set("limit", "5")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/search.json?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	client := o.Client
	if client == nil {
		client = defaultProviderClient(20 * time.Second)
	}
	resp, err := redirectSafeClient(client, []string{"Authorization"}).Do(req)
	if err != nil {
		return nil, sanitizeError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openlibrary: HTTP %d", resp.StatusCode)
	}
	var decoded olResponse
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 4<<20)).Decode(&decoded); err != nil {
		return nil, err
	}
	out := make([]Candidate, 0, len(decoded.Docs))
	for _, d := range decoded.Docs {
		e := catalog.Edition{Title: d.Title, Authors: d.AuthorName, State: catalog.NeedsReview}
		if len(d.Publisher) > 0 {
			e.Publisher = d.Publisher[0]
		}
		if len(d.Language) > 0 {
			e.Language = d.Language[0]
		}
		if d.FirstPublishYear > 0 {
			e.Published = fmt.Sprintf("%d", d.FirstPublishYear)
		}
		for _, v := range d.ISBN {
			e.Identifiers = append(e.Identifiers, catalog.Identifier{Scheme: "isbn", Value: v, Source: "openlibrary", Confidence: 0.93})
		}
		if d.CoverI > 0 {
			e.CoverHREF = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", d.CoverI)
		}
		conf := 0.82
		if isbnValue != "" {
			conf = 0.96
		}
		out = append(out, Candidate{Edition: e, Confidence: conf})
	}
	return out, nil
}
