package identify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

type staticResolver struct {
	name       string
	candidates []Candidate
	err        error
}

func (s staticResolver) Name() string { return s.name }
func (s staticResolver) Resolve(context.Context, Snapshot) ([]Candidate, error) {
	return s.candidates, s.err
}

type staticDecision struct{ d Decision }

func (s staticDecision) Decide(context.Context, Snapshot, Candidate) (Decision, error) {
	return s.d, nil
}

func fixedTime() time.Time { return time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC) }

func TestValidEmbeddedISBNAccepted(t *testing.T) {
	p := Pipeline{Now: fixedTime}
	r, err := p.Resolve(context.Background(), Snapshot{Embedded: catalog.Edition{ID: "e1", Title: "Book", Authors: []string{"Author"}, Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "978-0-441-47812-5"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Accepted || r.Edition.State != catalog.Identified {
		t.Fatalf("valid embedded identity not accepted: %+v", r)
	}
	if len(r.Edition.Identifiers) != 1 || r.Edition.Identifiers[0].Value != "9780441478125" {
		t.Fatalf("embedded ISBN normalization was discarded: %+v", r.Edition.Identifiers)
	}
}

func TestLLMCandidateCannotAutoAccept(t *testing.T) {
	r := staticResolver{name: "llm:test", candidates: []Candidate{{Edition: catalog.Edition{Title: "Model Guess", Authors: []string{"Someone"}}, Confidence: 1}}}
	p := Pipeline{Now: fixedTime, Resolvers: []Resolver{r}}
	got, err := p.Resolve(context.Background(), Snapshot{TextSample: "Title page"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Accepted || got.Edition.Confidence > 0.79 {
		t.Fatalf("model-only candidate bypassed corroboration: %+v", got)
	}
}

func TestDecisionModelCanOnlyDowngrade(t *testing.T) {
	s := Snapshot{Embedded: catalog.Edition{Title: "Book", Authors: []string{"Author"}, Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "9780441478125"}}}}
	p := Pipeline{Now: fixedTime, Decision: staticDecision{d: Decision{Choice: "review", Probability: 1}}}
	got, err := p.Resolve(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if got.Accepted || got.Edition.State != catalog.Partial {
		t.Fatalf("decision model review was ignored: %+v", got)
	}
}

func TestInvalidISBNRejected(t *testing.T) {
	r := staticResolver{name: "bad", candidates: []Candidate{{Edition: catalog.Edition{Title: "Book", Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "9780441478126"}}}, Confidence: 1}}}
	p := Pipeline{Now: fixedTime, Resolvers: []Resolver{r}}
	got, err := p.Resolve(context.Background(), Snapshot{TextSample: "text"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Accepted || len(got.Errors) == 0 {
		t.Fatalf("invalid ISBN accepted: %+v", got)
	}
}

func TestOpenLibraryClientAndNoKeyLeak(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("isbn") != "9780441478125" {
			t.Errorf("wrong query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"docs": []map[string]any{{"title": "The Left Hand of Darkness", "author_name": []string{"Ursula K. Le Guin"}, "isbn": []string{"9780441478125"}, "cover_i": 123}}})
	}))
	defer ts.Close()
	o := OpenLibrary{BaseURL: ts.URL, Client: ts.Client()}
	got, err := o.Resolve(context.Background(), Snapshot{Embedded: catalog.Edition{Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "9780441478125"}}}})
	if err != nil || len(got) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if got[0].Edition.CoverHREF == "" || got[0].Confidence < 0.9 {
		t.Fatalf("incomplete result: %+v", got[0])
	}
}

func TestOpenAICompatibleContract(t *testing.T) {
	var authorization string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("wrong endpoint: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"content": `{"title":"Actual Title","authors":["Actual Author"],"isbn":[],"confidence":0.8}`}}}})
	}))
	defer ts.Close()
	l := LLMResolver{Config: ProviderConfig{Name: "test", BaseURL: ts.URL, APIKey: "secret", Model: "flash"}, Client: ts.Client()}
	got, err := l.Resolve(context.Background(), Snapshot{TextSample: "untrusted text"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Edition.Title != "Actual Title" {
		t.Fatalf("bad extraction: %+v", got)
	}
	if authorization != "Bearer secret" {
		t.Fatalf("auth missing: %q", authorization)
	}
}

func TestModelResponseBodiesAreBounded(t *testing.T) {
	t.Run("llm", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", (2<<20)+1)))
		}))
		defer ts.Close()
		l := LLMResolver{Config: ProviderConfig{BaseURL: ts.URL, Model: "test"}, Client: ts.Client()}
		_, err := l.Resolve(context.Background(), Snapshot{TextSample: "book"})
		if err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("oversized LLM response must be rejected, got %v", err)
		}
	})
	t.Run("jev", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", (1<<20)+1)))
		}))
		defer ts.Close()
		j := JevClient{Endpoint: ts.URL, Client: ts.Client()}
		_, err := j.Decide(context.Background(), Snapshot{}, Candidate{})
		if err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("oversized Jev response must be rejected, got %v", err)
		}
	})
}

func TestLLMRedirectDoesNotLeakProviderKey(t *testing.T) {
	var leaked atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "" || r.Header.Get("Authorization") != "" {
			leaked.Store(true)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"content": []map[string]string{{"type": "text", "text": `{"title":"Safe","authors":[],"isbn":[],"confidence":0.5}`}}})
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	l := LLMResolver{Config: ProviderConfig{Kind: "anthropic", BaseURL: source.URL, APIKey: "top-secret", Model: "test"}, Client: source.Client()}
	if _, err := l.Resolve(context.Background(), Snapshot{TextSample: "book"}); err != nil {
		t.Fatal(err)
	}
	if leaked.Load() {
		t.Fatal("provider credential leaked to a cross-origin redirect")
	}
}
