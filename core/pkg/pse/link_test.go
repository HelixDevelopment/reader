package pse

import (
	"strings"
	"testing"
)

const feedXML = `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:pse="http://vaemendis.net/opds-pse/ns">
<entry>
  <link rel="http://vaemendis.net/opds-pse/stream" type="image/jpeg"
        href="/opds-comics/stream/1217?page={pageNumber}&amp;width={maxWidth}"
        pse:count="35" pse:lastRead="10" pse:lastReadDate="2010-01-10T10:01:11Z"/>
  <link rel="http://vaemendis.net/opds-pse/stream" type="image/png"
        href="/s/2/{pageNumber}" pse:count="2"/>
</entry>
</feed>`

func TestParseAcquisitionLinks(t *testing.T) {
	links, err := ParseAcquisitionLinks([]byte(feedXML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("got %d links, want 2", len(links))
	}
	l := links[0]
	if l.Count != 35 || !l.HasLastRead || l.LastRead != 10 ||
		!l.HasLastReadDt || l.LastReadDate != "2010-01-10T10:01:11Z" {
		t.Errorf("link 0 fields wrong: %+v", l)
	}
	if links[1].HasLastRead || links[1].HasLastReadDt {
		t.Errorf("link 1 reports v1.1/v1.2 members it was not given: %+v", links[1])
	}
}

// The Atom date construct is RFC 3339 date-time or, with no time part, date.
func TestParseAcceptsBothAtomDateForms(t *testing.T) {
	for _, d := range []string{"2010-01-10T10:01:11Z", "2010-01-10"} {
		feed := `<feed xmlns:pse="http://vaemendis.net/opds-pse/ns"><link rel="` + StreamRel +
			`" type="image/png" href="/s/{pageNumber}" pse:count="3" pse:lastRead="2" pse:lastReadDate="` + d + `"/></feed>`
		links, err := ParseAcquisitionLinks([]byte(feed))
		if err != nil {
			t.Fatalf("date %q: %v", d, err)
		}
		if !links[0].HasLastReadDt || links[0].LastReadDate != d || links[0].LastRead != 2 {
			t.Errorf("date %q parsed as %+v", d, links[0])
		}
	}
}

func TestParseAcquisitionLinkErrors(t *testing.T) {
	tests := []struct {
		name    string
		xmlBody string
		wantErr string
	}{
		{
			name:    "no pse link",
			xmlBody: `<feed xmlns:pse="http://vaemendis.net/opds-pse/ns"><link rel="http://opds-spec.org/image" href="/x"/></feed>`,
			wantErr: `pse: no acquisition link with rel "http://vaemendis.net/opds-pse/stream" found`,
		},
		{
			name:    "bad type rejected per MUST image types",
			xmlBody: `<feed xmlns:pse="http://vaemendis.net/opds-pse/ns"><link rel="` + StreamRel + `" type="application/zip" href="/s/{pageNumber}" pse:count="3"/></feed>`,
			wantErr: `pse: link type "application/zip" is not an image type allowed by OPDS-PSE (image/jpeg, image/gif, image/png)`,
		},
		{
			name:    "missing pageNumber template",
			xmlBody: `<feed xmlns:pse="http://vaemendis.net/opds-pse/ns"><link rel="` + StreamRel + `" type="image/jpeg" href="/s/page" pse:count="3"/></feed>`,
			wantErr: `pse: href "/s/page" must contain the {pageNumber} template`,
		},
		{
			name:    "count zero invalid",
			xmlBody: `<feed xmlns:pse="http://vaemendis.net/opds-pse/ns"><link rel="` + StreamRel + `" type="image/jpeg" href="/s/{pageNumber}" pse:count="0"/></feed>`,
			wantErr: `pse: pse:count attribute is required and must be a positive integer, got "0"`,
		},
		{
			name:    "count missing",
			xmlBody: `<feed xmlns:pse="http://vaemendis.net/opds-pse/ns"><link rel="` + StreamRel + `" type="image/jpeg" href="/s/{pageNumber}"/></feed>`,
			wantErr: `pse: pse:count attribute is required and must be a positive integer`,
		},
		{
			name:    "lastRead zero violates numbering starts at 1",
			xmlBody: `<feed xmlns:pse="http://vaemendis.net/opds-pse/ns"><link rel="` + StreamRel + `" type="image/jpeg" href="/s/{pageNumber}" pse:count="3" pse:lastRead="0"/></feed>`,
			wantErr: `pse: pse:lastRead must be an integer >= 1 (numbering starts at 1), got "0"`,
		},
		{
			name:    "lastReadDate must conform to the Atom date construct",
			xmlBody: `<feed xmlns:pse="http://vaemendis.net/opds-pse/ns"><link rel="` + StreamRel + `" type="image/jpeg" href="/s/{pageNumber}" pse:count="3" pse:lastReadDate="10-01-2010"/></feed>`,
			wantErr: `pse: pse:lastReadDate "10-01-2010" must conform to the Atom date construct`,
		},
		{
			name:    "malformed xml",
			xmlBody: `<feed><link`,
			wantErr: `pse: malformed OPDS feed XML:`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseAcquisitionLinks([]byte(tc.xmlBody))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got err %v, want message containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestPageURL(t *testing.T) {
	links, err := ParseAcquisitionLinks([]byte(feedXML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	withMax, plain := links[0], links[1]

	tests := []struct {
		name    string
		link    Link
		page    int
		width   int
		want    string
		wantErr string
	}{
		{name: "first page", link: plain, page: 0, width: 0, want: "/s/2/0"},
		{name: "last page N-1 ok", link: plain, page: 1, width: 0, want: "/s/2/1"},
		{name: "page N rejected", link: plain, page: 2, width: 0, wantErr: "pse: page 2 out of range 0..1"},
		{name: "negative page rejected", link: plain, page: -1, wantErr: "pse: page -1 out of range 0..1"},
		{name: "both templates", link: withMax, page: 34, width: 800,
			want: "/opds-comics/stream/1217?page=34&width=800"},
		{name: "maxWidth required when template present", link: withMax, page: 1, width: 0,
			wantErr: "pse: href contains {maxWidth} but maxWidth 0 is not positive"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.link.PageURL(tc.page, tc.width)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got err %v, want message containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("PageURL = %q, want %q", got, tc.want)
			}
		})
	}
}
