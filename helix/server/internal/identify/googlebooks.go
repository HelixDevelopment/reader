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

// GoogleBooks uses the public Volumes API with an optional server-side key.
type GoogleBooks struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func (g *GoogleBooks) Name() string { return "google-books" }

type gbResponse struct {
	Items []struct {
		ID         string `json:"id"`
		VolumeInfo struct {
			Title               string   `json:"title"`
			Subtitle            string   `json:"subtitle"`
			Authors             []string `json:"authors"`
			Publisher           string   `json:"publisher"`
			PublishedDate       string   `json:"publishedDate"`
			Description         string   `json:"description"`
			IndustryIdentifiers []struct {
				Type       string `json:"type"`
				Identifier string `json:"identifier"`
			} `json:"industryIdentifiers"`
			ImageLinks struct {
				Thumbnail string `json:"thumbnail"`
			} `json:"imageLinks"`
			Language string `json:"language"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

func (g *GoogleBooks) Resolve(ctx context.Context, s Snapshot) ([]Candidate, error) {
	base := strings.TrimRight(g.BaseURL, "/")
	if base == "" {
		base = "https://www.googleapis.com/books/v1"
	}
	query := ""
	for _, id := range s.Embedded.Identifiers {
		if strings.HasPrefix(strings.ToLower(id.Scheme), "isbn") {
			query = "isbn:" + id.Value
			break
		}
	}
	if query == "" {
		if s.Embedded.Title == "" {
			return nil, nil
		}
		query = "intitle:" + s.Embedded.Title
		if len(s.Embedded.Authors) > 0 {
			query += "+inauthor:" + s.Embedded.Authors[0]
		}
	}
	q := url.Values{"q": {query}, "maxResults": {"5"}}
	if g.APIKey != "" {
		q.Set("key", g.APIKey)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/volumes?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	client := g.Client
	if client == nil {
		client = defaultProviderClient(20 * time.Second)
	}
	resp, err := redirectSafeClient(client, []string{"Authorization", "X-API-Key"}).Do(req)
	if err != nil {
		return nil, sanitizeError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("google-books: HTTP %d", resp.StatusCode)
	}
	var decoded gbResponse
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 4<<20)).Decode(&decoded); err != nil {
		return nil, err
	}
	out := make([]Candidate, 0, len(decoded.Items))
	for _, item := range decoded.Items {
		v := item.VolumeInfo
		e := catalog.Edition{Title: v.Title, Subtitle: v.Subtitle, Authors: v.Authors, Publisher: v.Publisher, Published: v.PublishedDate, Description: v.Description, Language: v.Language, CoverHREF: strings.ReplaceAll(v.ImageLinks.Thumbnail, "http://", "https://")}
		for _, id := range v.IndustryIdentifiers {
			scheme := "other"
			if id.Type == "ISBN_13" {
				scheme = "isbn13"
			}
			if id.Type == "ISBN_10" {
				scheme = "isbn10"
			}
			e.Identifiers = append(e.Identifiers, catalog.Identifier{Scheme: scheme, Value: id.Identifier, Source: "google-books", Confidence: 0.94})
		}
		conf := 0.84
		if strings.HasPrefix(query, "isbn:") {
			conf = 0.96
		}
		out = append(out, Candidate{Edition: e, Confidence: conf})
	}
	return out, nil
}
