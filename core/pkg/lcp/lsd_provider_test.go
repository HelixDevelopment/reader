package lcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// lsdProvider is a loopback License Status Document provider written from the
// Readium LSD 1.0 revision 5 text (readium/lcp-specs releases/lsd/latest.md,
// sections 3.3-3.5) and NOT from lsd.go. It is in-repo, so it shows Helix
// agrees with the spec as read by its authors; it is not a third-party provider.
//
// Server transitions (latest.md):
//
//	register: ready -> active, status timestamp updated        (l.282)
//	return:   active -> returned, ready -> cancelled, both
//	          timestamps updated, end set to now               (l.326)
//	renew:    end updated, both timestamps updated             (l.383)
type lsdProvider struct {
	mu     sync.Mutex
	srv    *httptest.Server
	clock  time.Time
	status string
	end    time.Time
	licUpd time.Time
	stUpd  time.Time
	events []map[string]any
	log    []lsdLogEntry

	// Fault injection for negative paths.
	skipActivate bool // register leaves status "ready" (contradicts l.282)
	wrongID      bool // responses carry another license id
}

type lsdLogEntry struct {
	Method    string   `json:"method"`
	Path      string   `json:"path"`
	QueryKeys []string `json:"query_keys"`
	Status    int      `json:"status"`
}

const lsdTestLicenseID = "lic-0001"

func newLSDProvider() *lsdProvider {
	p := &lsdProvider{
		clock:  time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC),
		status: "ready",
		end:    time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
	}
	p.licUpd, p.stUpd = p.clock, p.clock
	mux := http.NewServeMux()
	mux.HandleFunc("/status", p.handle("GET", p.serveStatus))
	mux.HandleFunc("/register", p.handle("POST", p.serveRegister))
	mux.HandleFunc("/return", p.handle("PUT", p.serveReturn))
	mux.HandleFunc("/renew", p.handle("PUT", p.serveRenew))
	mux.HandleFunc("/license", func(w http.ResponseWriter, r *http.Request) { p.record(r, 200); w.WriteHeader(200) })
	p.srv = httptest.NewServer(mux)
	return p
}

func (p *lsdProvider) tick() time.Time { p.clock = p.clock.Add(time.Minute); return p.clock }

func (p *lsdProvider) record(r *http.Request, code int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var keys []string
	for k := range r.URL.Query() {
		keys = append(keys, k)
	}
	p.log = append(p.log, lsdLogEntry{r.Method, r.URL.Path, keys, code})
}

func (p *lsdProvider) handle(verb string, f func(w http.ResponseWriter, r *http.Request) int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		code := 405
		if r.Method == verb {
			code = f(w, r)
		} else {
			w.WriteHeader(405)
		}
		p.mu.Unlock()
		p.mu.Lock()
		var keys []string
		for k := range r.URL.Query() {
			keys = append(keys, k)
		}
		p.log = append(p.log, lsdLogEntry{r.Method, r.URL.Path, keys, code})
		p.mu.Unlock()
	}
}

func (p *lsdProvider) problem(w http.ResponseWriter, code int, typ, title, detail string) int {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": typ, "title": title, "detail": detail})
	return code
}

func (p *lsdProvider) document() []byte {
	base := p.srv.URL
	id := lsdTestLicenseID
	if p.wrongID {
		id = "another-license"
	}
	doc := map[string]any{
		"id": id, "status": p.status, "message": "status " + p.status,
		"updated": map[string]any{"license": p.licUpd.Format(time.RFC3339), "status": p.stUpd.Format(time.RFC3339)},
		"links": []any{
			map[string]any{"rel": "license", "href": base + "/license", "type": "application/vnd.readium.lcp.license.v1.0+json", "profile": "http://readium.org/lcp/basic-profile"},
			map[string]any{"rel": "register", "href": base + "/register{?id,name}", "type": lsdMediaType, "templated": true},
			map[string]any{"rel": "return", "href": base + "/return{?id,name}", "type": lsdMediaType, "templated": true},
			map[string]any{"rel": "renew", "href": base + "/renew-ui", "type": "text/html"},
			map[string]any{"rel": "renew", "href": base + "/renew{?end,id,name}", "type": lsdMediaType, "templated": true},
		},
		"potential_rights": map[string]any{"end": p.end.Format(time.RFC3339)},
		"events":           p.events,
	}
	b, _ := json.Marshal(doc)
	return b
}

func (p *lsdProvider) writeDoc(w http.ResponseWriter) int {
	w.Header().Set("Content-Type", lsdMediaType)
	_, _ = w.Write(p.document())
	return 200
}

func (p *lsdProvider) addEvent(typ string, r *http.Request) {
	p.events = append(p.events, map[string]any{
		"type": typ, "name": r.URL.Query().Get("name"), "id": r.URL.Query().Get("id"),
		"timestamp": p.clock.Format(time.RFC3339),
	})
}

func (p *lsdProvider) serveStatus(w http.ResponseWriter, r *http.Request) int { return p.writeDoc(w) }

func (p *lsdProvider) serveRegister(w http.ResponseWriter, r *http.Request) int {
	if r.URL.Query().Get("id") == "" || r.URL.Query().Get("name") == "" {
		return p.problem(w, 400, "http://readium.org/license-status-document/error/registration", "Your device could not be registered properly.", "id and name are required")
	}
	if p.status == "revoked" {
		return p.problem(w, 400, "http://readium.org/license-status-document/error/registration", "Your device could not be registered properly.", "")
	}
	p.stUpd = p.tick()
	if p.status == "ready" && !p.skipActivate {
		p.status = "active"
	}
	p.addEvent("register", r)
	return p.writeDoc(w)
}

func (p *lsdProvider) serveReturn(w http.ResponseWriter, r *http.Request) int {
	const base = "http://readium.org/license-status-document/error/return"
	switch p.status {
	case "returned":
		return p.problem(w, 403, base+"/already", "Your publication has already been returned before.", "")
	case "expired":
		return p.problem(w, 403, base+"/expired", "Your publication has already expired.", "")
	case "active":
		p.status = "returned"
	case "ready":
		p.status = "cancelled"
	default:
		return p.problem(w, 400, base, "Your publication could not be returned properly.", "status "+p.status)
	}
	now := p.tick()
	p.end, p.licUpd, p.stUpd = now, now, now
	p.addEvent("return", r)
	return p.writeDoc(w)
}

func (p *lsdProvider) serveRenew(w http.ResponseWriter, r *http.Request) int {
	const base = "http://readium.org/license-status-document/error/renew"
	if p.status != "active" && p.status != "ready" {
		return p.problem(w, 400, base, "Your publication could not be renewed properly.", "status "+p.status)
	}
	if e := r.URL.Query().Get("end"); e != "" {
		t, err := time.Parse(time.RFC3339, e)
		if err != nil || !t.After(p.end) || t.After(p.end.AddDate(0, 0, 90)) {
			return p.problem(w, 403, base+"/date", "Incorrect renewal period, your publication could not be renewed.", "")
		}
		p.end = t
	} else {
		p.end = p.end.AddDate(0, 0, 14)
	}
	now := p.tick()
	p.licUpd, p.stUpd = now, now
	p.addEvent("renew", r)
	return p.writeDoc(w)
}

func (p *lsdProvider) statusURL() string { return p.srv.URL + "/status" }

func (p *lsdProvider) requests() []lsdLogEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]lsdLogEntry(nil), p.log...)
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if n != "" && strings.Contains(s, n) {
			return true
		}
	}
	return false
}
