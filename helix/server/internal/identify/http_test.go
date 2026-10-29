package identify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/helixreader/helix/server/internal/catalog"
)

func TestRequireSecureEndpoint(t *testing.T) {
	cases := []struct {
		name        string
		endpoint    string
		credentials bool
		wantErr     bool
	}{
		{"https with credentials", "https://api.example.com/v1", true, false},
		{"http loopback IP with credentials", "http://127.0.0.1:8080/v1", true, false},
		{"http localhost with credentials", "http://localhost:11434/v1", true, false},
		{"http IPv6 loopback with credentials", "http://[::1]:8080/v1", true, false},
		{"http remote with credentials", "http://api.example.com/v1", true, true},
		{"http remote without credentials", "http://api.example.com/v1", false, false},
		{"cleartext hostname that merely looks local", "http://localhost.evil.example/v1", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := requireSecureEndpoint(tc.endpoint, tc.credentials)
			if (err != nil) != tc.wantErr {
				t.Fatalf("requireSecureEndpoint(%q, %v) = %v, wantErr %v", tc.endpoint, tc.credentials, err, tc.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "cleartext") {
				t.Fatalf("error should explain the cleartext rejection: %v", err)
			}
		})
	}
}

func TestSanitizeError(t *testing.T) {
	inner := &url.Error{Op: "Get", URL: "https://user:pw@api.example.com/volumes?q=x&key=secret-key#frag", Err: errors.New("boom")}
	got := sanitizeError(inner)
	if got == nil {
		t.Fatal("expected non-nil error")
	}
	for _, secret := range []string{"secret-key", "user:pw", "q=x", "frag"} {
		if strings.Contains(got.Error(), secret) {
			t.Fatalf("sanitized error still contains %q: %v", secret, got)
		}
	}
	if !strings.Contains(got.Error(), "api.example.com") {
		t.Fatalf("sanitized error should retain host context: %v", got)
	}
	if sanitizeError(nil) != nil {
		t.Fatal("nil error must stay nil")
	}
	plain := fmt.Errorf("plain failure")
	if sanitizeError(plain) != plain {
		t.Fatal("non-URL errors must pass through unchanged")
	}
}

func TestGoogleBooksErrorDoesNotLeakAPIKey(t *testing.T) {
	g := &GoogleBooks{BaseURL: "http://127.0.0.1:1", APIKey: "supersecret-key"}
	_, err := g.Resolve(context.Background(), Snapshot{Embedded: embeddedTitle("Book")})
	if err == nil {
		t.Fatal("expected connection error")
	}
	if strings.Contains(err.Error(), "supersecret-key") || strings.Contains(err.Error(), "key=") {
		t.Fatalf("error leaks API key or query: %v", err)
	}
}

func TestOpenLibraryErrorDoesNotLeakQueryOrUserinfo(t *testing.T) {
	o := &OpenLibrary{BaseURL: "http://user:pw@127.0.0.1:1"}
	_, err := o.Resolve(context.Background(), Snapshot{Embedded: embeddedTitle("Book")})
	if err == nil {
		t.Fatal("expected connection error")
	}
	if strings.Contains(err.Error(), "user") && strings.Contains(err.Error(), "pw") {
		t.Fatalf("error leaks userinfo: %v", err)
	}
	if strings.Contains(err.Error(), "title=") {
		t.Fatalf("error leaks query: %v", err)
	}
}

func TestGoogleBooksCrossOriginRedirectDoesNotLeakKey(t *testing.T) {
	var leakedQuery string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leakedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/volumes", http.StatusFound)
	}))
	defer source.Close()
	g := &GoogleBooks{BaseURL: source.URL, APIKey: "supersecret-key", Client: source.Client()}
	if _, err := g.Resolve(context.Background(), Snapshot{Embedded: embeddedTitle("Book")}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(leakedQuery, "supersecret-key") || strings.Contains(leakedQuery, "key=") {
		t.Fatalf("API key leaked across cross-origin redirect: %q", leakedQuery)
	}
}

func TestLLMRejectsCleartextEndpointWithCredentials(t *testing.T) {
	l := &LLMResolver{Config: ProviderConfig{Name: "remote", BaseURL: "http://llm.example.com/v1", APIKey: "secret", Model: "m"}}
	_, err := l.Resolve(context.Background(), Snapshot{TextSample: "text"})
	if err == nil || !strings.Contains(err.Error(), "cleartext") {
		t.Fatalf("cleartext LLM endpoint with credentials must be rejected, got %v", err)
	}
}

func TestLLMLoopbackCleartextAllowed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("auth header missing on loopback request")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"content": `{"title":"T","authors":["A"],"isbn":[],"confidence":0.5}`}}}})
	}))
	defer ts.Close()
	if !strings.HasPrefix(ts.URL, "http://") {
		t.Fatal("fixture must be cleartext")
	}
	l := &LLMResolver{Config: ProviderConfig{Name: "local", BaseURL: ts.URL, APIKey: "secret", Model: "m"}, Client: ts.Client()}
	got, err := l.Resolve(context.Background(), Snapshot{TextSample: "text"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Edition.Title != "T" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestJevRejectsCleartextEndpointWithCredentials(t *testing.T) {
	j := &JevClient{Endpoint: "http://jev.example.com/decide", APIKey: "secret"}
	_, err := j.Decide(context.Background(), Snapshot{}, Candidate{})
	if err == nil || !strings.Contains(err.Error(), "cleartext") {
		t.Fatalf("cleartext Jev endpoint with credentials must be rejected, got %v", err)
	}
}

func TestJevLoopbackCleartextAllowed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("auth header missing on loopback request")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"answers": map[string]any{"route": map[string]any{"choice": "review", "probability": 0.9}}})
	}))
	defer ts.Close()
	j := &JevClient{Endpoint: ts.URL, APIKey: "secret", Client: ts.Client()}
	d, err := j.Decide(context.Background(), Snapshot{}, Candidate{})
	if err != nil {
		t.Fatal(err)
	}
	if d.Choice != "review" || d.Probability != 0.9 {
		t.Fatalf("unexpected decision: %+v", d)
	}
}

func TestJevErrorDoesNotLeakUserinfo(t *testing.T) {
	j := &JevClient{Endpoint: "http://user:pw@127.0.0.1:1/decide"}
	_, err := j.Decide(context.Background(), Snapshot{}, Candidate{})
	if err == nil {
		t.Fatal("expected connection error")
	}
	if strings.Contains(err.Error(), "user:pw") {
		t.Fatalf("error leaks userinfo: %v", err)
	}
}

func TestPipelineStoredErrorsAreSanitized(t *testing.T) {
	p := Pipeline{
		Now:       fixedTime,
		Resolvers: []Resolver{&GoogleBooks{BaseURL: "http://127.0.0.1:1", APIKey: "supersecret-key"}},
		Decision:  &JevClient{Endpoint: "http://user:pw@127.0.0.1:1/decide"},
	}
	got, err := p.Resolve(context.Background(), Snapshot{Embedded: catalog.Edition{Title: "Book", Authors: []string{"Author"}, Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "9780441478125"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Errors) == 0 {
		t.Fatal("expected resolver and decision errors to be recorded")
	}
	joined := strings.Join(got.Errors, "\n")
	for _, secret := range []string{"supersecret-key", "key=", "user:pw"} {
		if strings.Contains(joined, secret) {
			t.Fatalf("stored pipeline errors leak %q:\n%s", secret, joined)
		}
	}
}

func embeddedTitle(title string) catalog.Edition {
	return catalog.Edition{Title: title, Authors: []string{"Author"}}
}
