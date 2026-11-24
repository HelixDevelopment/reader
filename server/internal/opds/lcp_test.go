package opds

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

func lcpWork() (catalog.Work, catalog.Edition) {
	e := catalog.Edition{ID: "e9", Title: "Sealed", Files: []catalog.File{{ID: "f9", Format: "epub", MIMEType: "application/epub+zip", Protected: true, DRMScheme: "lcp", LicenseHREF: "/v1/files/f9/license"}}}
	return catalog.Work{ID: "w9", Editions: []catalog.Edition{e}}, e
}

// OPDS 1.2 §5.2.1/§5.3 + readium lcp-key-retrieval: an LCP publication is
// acquired through its License Document (rel acquisition, license media type)
// with an indirectAcquisition naming the EPUB it yields. The protected EPUB is
// not offered as an ordinary plain-EPUB acquisition.
func TestLCPEntry12LicenseAcquisitionWithIndirect(t *testing.T) {
	w, e := lcpWork()
	raw, err := Edition12(w, e, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	var parsed struct {
		Links []struct {
			Rel      string `xml:"rel,attr"`
			Href     string `xml:"href,attr"`
			Type     string `xml:"type,attr"`
			Indirect []struct {
				Type string `xml:"type,attr"`
			} `xml:"indirectAcquisition"`
		} `xml:"link"`
	}
	if err := xml.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("not well formed: %v\n%s", err, body)
	}
	found := false
	for _, l := range parsed.Links {
		if l.Rel == RelAcquisition && l.Type == "application/epub+zip" {
			t.Errorf("protected EPUB offered as a plain acquisition: %+v", l)
		}
		if l.Rel == RelAcquisition && l.Type == LCPLicenseMIME && l.Href == "/v1/files/f9/license" {
			if len(l.Indirect) != 1 || l.Indirect[0].Type != "application/epub+zip" {
				t.Errorf("license acquisition lacks indirectAcquisition epub: %+v", l)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("no license acquisition link:\n%s", body)
	}
	if !strings.Contains(body, `xmlns:opds="http://opds-spec.org/2010/catalog"`) {
		t.Fatal("opds namespace not declared")
	}
}

func TestLCPPublication20LicenseAcquisitionWithIndirect(t *testing.T) {
	w, e := lcpWork()
	raw, err := Edition20(w, e)
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Links []struct {
			Href       string `json:"href"`
			Type       string `json:"type"`
			Rel        any    `json:"rel"`
			Properties struct {
				Indirect []struct {
					Type string `json:"type"`
				} `json:"indirectAcquisition"`
			} `json:"properties"`
		} `json:"links"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, l := range p.Links {
		if l.Rel == RelAcquisition && l.Type == "application/epub+zip" {
			t.Errorf("protected EPUB offered as a plain acquisition: %+v", l)
		}
		if l.Rel == RelAcquisition && l.Type == LCPLicenseMIME && l.Href == "/v1/files/f9/license" {
			if len(l.Properties.Indirect) != 1 || l.Properties.Indirect[0].Type != "application/epub+zip" {
				t.Errorf("license acquisition lacks properties.indirectAcquisition: %+v", l)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("no license acquisition link:\n%s", raw)
	}
}

// TestDumpLCPFeeds writes the LCP feeds to HELIX_OPDS_DUMP (evidence tooling:
// the files are then validated against the opds-community JSON schema).
func TestDumpLCPFeeds(t *testing.T) {
	dir := os.Getenv("HELIX_OPDS_DUMP")
	if dir == "" {
		t.Skip("HELIX_OPDS_DUMP not set")
	}
	w, _ := lcpWork()
	w.Editions[0].UpdatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	page := catalog.Page{Total: 1, Limit: 20, Works: []catalog.Work{w}}
	j, err := Acquisition20("/opds/2.0/publications", page, catalog.Query{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	x, err := Acquisition12("/opds/1.2/publications", page, catalog.Query{Limit: 20}, nil)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "feed-2.0.json"), j, 0o644)
	os.WriteFile(filepath.Join(dir, "feed-1.2.xml"), x, 0o644)
}
