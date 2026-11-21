// CBZ page-packaging exporter: HDM image blocks -> comic archive zip
// (docs/31 pivot). Entry names are the HDM resource ids plus a sniffed image
// extension, so re-parsing with pkg/parse/cbz reproduces the page order and
// resource payloads exactly. Only PNG and JPEG payloads are accepted: the
// exporter refuses rather than packaging bytes it cannot name honestly.
package convert

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"

	"github.com/helixreader/helix/core/pkg/hdm"
)

// toCBZ packs every image block (reading order) into a CBZ archive.
func toCBZ(doc *hdm.Document, _ Options) ([]byte, error) {
	var order []string
	seen := map[string]bool{}
	for _, ch := range doc.FlatTOC() {
		for _, b := range ch.Blocks {
			if b.Kind != hdm.BlockImage {
				continue
			}
			if b.ImageRef == "" || strings.ContainsAny(b.ImageRef, `/%\`) || seen[b.ImageRef] {
				if b.ImageRef == "" || strings.ContainsAny(b.ImageRef, `/%\`) {
					return nil, fmt.Errorf("convert: cbz: invalid image resource id %q", b.ImageRef)
				}
				continue // same image referenced twice: package once, first order wins
			}
			seen[b.ImageRef] = true
			order = append(order, b.ImageRef)
		}
	}
	if len(order) == 0 {
		return nil, fmt.Errorf("convert: cbz: document has no image blocks")
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, id := range order {
		data, ok := doc.Resources[id]
		if !ok {
			return nil, fmt.Errorf("convert: cbz: missing image resource %q", id)
		}
		ext, _, ok := imageExt(data)
		if !ok {
			return nil, fmt.Errorf("convert: cbz: image %q is neither PNG nor JPEG; refusing to package unknown encoding", id)
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: id + ext, Method: zip.Store, Modified: epubDate})
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
