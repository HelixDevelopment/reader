package pse

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Normative constants, quoted from the retrieved OPDS-PSE specifications
// (see doc.go for URLs).
const (
	// NamespaceURI is the OPDS-PSE namespace every PSE feed MUST declare:
	// "An OPDS-PSE feed MUST declare the OPDS-PSE namespace:
	// http://vaemendis.net/opds-pse/ns"
	NamespaceURI = "http://vaemendis.net/opds-pse/ns"

	// StreamRel is the mandatory rel value of a PSE page streaming link:
	// "rel MUST have the value http://vaemendis.net/opds-pse/stream"
	StreamRel = "http://vaemendis.net/opds-pse/stream"

	// PageNumberTemplate and MaxWidthTemplate are the href template strings
	// the client MUST replace ("This URL MUST contain the string
	// {pageNumber} ... MAY contain the string {maxWidth}").
	PageNumberTemplate = "{pageNumber}"
	MaxWidthTemplate   = "{maxWidth}"
)

// ImageTypes lists the only permitted link type values: "value MUST be one
// of the following: image/jpeg, image/gif or image/png".
var ImageTypes = []string{"image/jpeg", "image/gif", "image/png"}

// Link is a parsed OPDS-PSE acquisition link. Field comments cite the
// retrieved MUST/MAY rules they enforce.
type Link struct {
	Rel   string // MUST equal StreamRel for a PSE link
	Type  string // MUST be one of ImageTypes
	Href  string // MUST contain PageNumberTemplate
	Count int    // pse:count: "MUST provide the number of pages"

	// LastRead (pse:lastRead) and LastReadDate (pse:lastReadDate) were
	// added in v1.1 / v1.2. Optional; when present LastRead is >= 1
	// ("The numbering starts at 1").
	LastRead      int
	HasLastRead   bool
	LastReadDate  string
	HasLastReadDt bool
}

// ParseAcquisitionLinks scans OPDS Atom feed XML and returns every link
// whose rel equals StreamRel, validating the mandatory attributes of the
// retrieved specification.
func ParseAcquisitionLinks(feed []byte) ([]Link, error) {
	dec := xml.NewDecoder(bytes.NewReader(feed))
	dec.Strict = true

	var links []Link
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("pse: malformed OPDS feed XML: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "link" {
			continue
		}
		attrs := map[string]string{} // pse-namespaced, keyed by local name
		std := map[string]string{}   // unprefixed attributes
		for _, a := range se.Attr {
			switch a.Name.Space {
			case NamespaceURI:
				attrs[a.Name.Local] = a.Value
			case "":
				std[a.Name.Local] = a.Value
			}
		}
		if std["rel"] != StreamRel {
			continue
		}
		l := Link{Rel: std["rel"], Type: std["type"], Href: std["href"]}
		if err := l.validate(std, attrs); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("pse: no acquisition link with rel %q found", StreamRel)
	}
	return links, nil
}

func (l *Link) validate(std, attrs map[string]string) error {
	if !IsImageType(l.Type) {
		return fmt.Errorf("pse: link type %q is not an image type allowed by OPDS-PSE (image/jpeg, image/gif, image/png)", l.Type)
	}
	if !strings.Contains(l.Href, PageNumberTemplate) {
		return fmt.Errorf("pse: href %q must contain the %s template", l.Href, PageNumberTemplate)
	}
	count, ok := attrs["count"]
	if !ok {
		return errors.New("pse: pse:count attribute is required and must be a positive integer")
	}
	n, err := strconv.Atoi(count)
	if err != nil || n < 1 {
		return fmt.Errorf("pse: pse:count attribute is required and must be a positive integer, got %q", count)
	}
	l.Count = n
	if lr, ok := attrs["lastRead"]; ok {
		v, err := strconv.Atoi(lr)
		if err != nil || v < 1 {
			return fmt.Errorf("pse: pse:lastRead must be an integer >= 1 (numbering starts at 1), got %q", lr)
		}
		l.LastRead, l.HasLastRead = v, true
	}
	if d, ok := attrs["lastReadDate"]; ok {
		if !isAtomDate(d) {
			return fmt.Errorf("pse: pse:lastReadDate %q must conform to the Atom date construct (RFC 3339 date-time or date)", d)
		}
		l.LastReadDate, l.HasLastReadDt = d, true
	}
	return nil
}

// PageURL instantiates the link href for a page, per the retrieved spec
// note "pages are numbered from 0 to N-1". maxWidth is substituted for
// {maxWidth}; because replacing the optional template is mandatory when
// present ("the client MUST replace (if it exists)"), an href containing
// {maxWidth} requires maxWidth > 0.
func (l Link) PageURL(page, maxWidth int) (string, error) {
	if page < 0 || page > l.Count-1 {
		return "", fmt.Errorf("pse: page %d out of range 0..%d", page, l.Count-1)
	}
	u := strings.ReplaceAll(l.Href, PageNumberTemplate, strconv.Itoa(page))
	if strings.Contains(u, MaxWidthTemplate) {
		if maxWidth <= 0 {
			return "", fmt.Errorf("pse: href contains %s but maxWidth %d is not positive", MaxWidthTemplate, maxWidth)
		}
		u = strings.ReplaceAll(u, MaxWidthTemplate, strconv.Itoa(maxWidth))
	}
	return u, nil
}

// isAtomDate reports whether s conforms to the Atom Date construct
// (RFC 4287 3.3), which is RFC 3339 date-time or, with no time part, date.
func isAtomDate(s string) bool {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}

// IsImageType reports whether t is one of the image media types the
// extension permits for a stream link. Producers use it for the same rule
// parsers enforce, so a feed cannot advertise a type a client must reject.
func IsImageType(t string) bool {
	for _, v := range ImageTypes {
		if t == v {
			return true
		}
	}
	return false
}
