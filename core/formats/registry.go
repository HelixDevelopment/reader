package formats

import (
	"path/filepath"
	"sort"
	"strings"
)

type Kind string

const (
	KindEBook      Kind = "ebook"
	KindComic      Kind = "comic"
	KindAudiobook  Kind = "audiobook"
	KindDocument   Kind = "document"
	KindPackage    Kind = "package"
	KindLicense    Kind = "license"
	KindAccessible Kind = "accessible"
)

type Strategy string

const (
	StrategyNative   Strategy = "native-hdm"
	StrategyReadium  Strategy = "readium"
	StrategyCalibre  Strategy = "calibre"
	StrategyMuPDF    Strategy = "mupdf"
	StrategyArchive  Strategy = "archive"
	StrategyAudio    Strategy = "audio"
	StrategyLicense  Strategy = "lcp-license"
	StrategyDAISY    Strategy = "daisy"
	StrategyExternal Strategy = "external"
)

// Capabilities describe implemented behavior only. Strategy names a planned or
// configured engine; it never upgrades these flags by itself.
type Capabilities struct {
	Detect     bool `json:"detect"`
	Metadata   bool `json:"metadata"`
	Stream     bool `json:"stream"`
	Render     bool `json:"render"`
	Convert    bool `json:"convert"`
	DRMFulfill bool `json:"drm_fulfill"`
	ContentAI  bool `json:"content_ai"`
}
type Descriptor struct {
	Format       Format       `json:"format"`
	Extensions   []string     `json:"extensions"`
	MIMETypes    []string     `json:"mime_types"`
	Kind         Kind         `json:"kind"`
	Strategy     Strategy     `json:"strategy"`
	Protected    bool         `json:"protected"`
	Container    bool         `json:"container"`
	Capabilities Capabilities `json:"capabilities"`
}

var parsedCaps = Capabilities{Detect: true, Metadata: true, Stream: true, Convert: true, ContentAI: true}
var streamCaps = Capabilities{Detect: true, Stream: true}
var cbzCaps = Capabilities{Detect: true, Metadata: true, Stream: true}
var detectCaps = Capabilities{Detect: true}
var registry = []Descriptor{
	{EPUB, []string{".epub", ".epub3"}, []string{"application/epub+zip"}, KindEBook, StrategyNative, false, true, parsedCaps},
	{PDF, []string{".pdf"}, []string{"application/pdf"}, KindDocument, StrategyMuPDF, false, false, streamCaps},
	{MOBI, []string{".mobi", ".prc", ".pdb"}, []string{"application/x-mobipocket-ebook"}, KindEBook, StrategyNative, false, false, parsedCaps},
	{AZW3, []string{".azw", ".azw3", ".azw4", ".kfx"}, []string{"application/vnd.amazon.ebook"}, KindEBook, StrategyCalibre, false, false, streamCaps},
	{FB2, []string{".fb2"}, []string{"application/x-fictionbook+xml"}, KindEBook, StrategyNative, false, false, parsedCaps},
	{Format("fb2_zip"), []string{".fb2.zip"}, []string{"application/x-fictionbook+zip"}, KindEBook, StrategyArchive, false, true, streamCaps},
	{Format("fb3"), []string{".fb3"}, []string{"application/x-fictionbook+zip"}, KindEBook, StrategyArchive, false, true, streamCaps},
	{DOCX, []string{".docx"}, []string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document"}, KindDocument, StrategyCalibre, false, true, streamCaps},
	{Format("doc"), []string{".doc"}, []string{"application/msword"}, KindDocument, StrategyCalibre, false, false, streamCaps},
	{Format("odt"), []string{".odt"}, []string{"application/vnd.oasis.opendocument.text"}, KindDocument, StrategyCalibre, false, true, streamCaps},
	{Format("rtf"), []string{".rtf"}, []string{"application/rtf"}, KindDocument, StrategyCalibre, false, false, streamCaps},
	{TXT, []string{".txt", ".md", ".markdown", ".textile", ".tcr"}, []string{"text/plain", "text/markdown"}, KindEBook, StrategyNative, false, false, parsedCaps},
	{HTMLZ, []string{".htmlz"}, []string{"application/html+zip"}, KindEBook, StrategyArchive, false, true, streamCaps},
	{Format("html"), []string{".html", ".htm", ".xhtml", ".phtml", ".shtml"}, []string{"text/html", "application/xhtml+xml"}, KindEBook, StrategyExternal, false, false, streamCaps},
	{CBZ, []string{".cbz"}, []string{"application/vnd.comicbook+zip"}, KindComic, StrategyArchive, false, true, cbzCaps},
	{Format("cbr"), []string{".cbr"}, []string{"application/vnd.comicbook-rar"}, KindComic, StrategyArchive, false, true, streamCaps},
	{Format("cbt"), []string{".cbt"}, []string{"application/x-cbt"}, KindComic, StrategyArchive, false, true, streamCaps},
	{Format("cb7"), []string{".cb7"}, []string{"application/x-cb7"}, KindComic, StrategyArchive, false, true, streamCaps},
	{Format("cba"), []string{".cba"}, []string{"application/x-cba"}, KindComic, StrategyArchive, false, true, streamCaps},
	{Format("djvu"), []string{".djvu", ".djv"}, []string{"image/vnd.djvu"}, KindDocument, StrategyMuPDF, false, false, streamCaps},
	{Format("chm"), []string{".chm"}, []string{"application/vnd.ms-htmlhelp"}, KindDocument, StrategyExternal, false, false, streamCaps},
	{Format("lit"), []string{".lit"}, []string{"application/x-ms-reader"}, KindEBook, StrategyCalibre, false, false, streamCaps},
	{Format("snb"), []string{".snb"}, []string{"application/x-snb-ebook"}, KindEBook, StrategyCalibre, false, false, streamCaps},
	{Format("lrf"), []string{".lrf", ".lrs"}, []string{"application/x-sony-bbeb"}, KindEBook, StrategyCalibre, false, false, streamCaps},
	{Format("pml"), []string{".pml", ".pmlz"}, []string{"application/x-pml"}, KindEBook, StrategyCalibre, false, true, streamCaps},
	{Format("rb"), []string{".rb"}, []string{"application/x-rocketbook"}, KindEBook, StrategyCalibre, false, false, streamCaps},
	{Format("webpub"), []string{".webpub"}, []string{"application/webpub+zip"}, KindPackage, StrategyReadium, false, true, streamCaps},
	{Format("audiobook"), []string{".audiobook", ".zab"}, []string{"application/audiobook+zip"}, KindAudiobook, StrategyReadium, false, true, streamCaps},
	{Format("divina"), []string{".divina"}, []string{"application/divina+zip"}, KindComic, StrategyReadium, false, true, streamCaps},
	{Format("lpf"), []string{".lpf"}, []string{"application/lpf+zip"}, KindPackage, StrategyReadium, false, true, streamCaps},
	{Format("pnld"), []string{".pnld"}, []string{"application/vnd.readium.pnld"}, KindAccessible, StrategyReadium, false, true, streamCaps},
	{Format("daisy"), []string{".daisy", ".opf", "ncc.html"}, []string{"application/x-daisy", "application/oebps-package+xml", "application/x-dtbncx+xml"}, KindAccessible, StrategyDAISY, false, true, streamCaps},
	{Format("lcpl"), []string{".lcpl"}, []string{"application/vnd.readium.lcp.license.v1.0+json"}, KindLicense, StrategyLicense, true, false, detectCaps},
	{Format("lcpa"), []string{".lcpa", ".lcpaudiobook"}, []string{"application/audiobook+lcp"}, KindAudiobook, StrategyReadium, true, true, streamCaps},
	{Format("lcpdf"), []string{".lcpdf"}, []string{"application/pdf+lcp"}, KindDocument, StrategyReadium, true, true, streamCaps},
	{Format("lcp_epub"), []string{".lcp.epub"}, []string{"application/epub+zip+lcp"}, KindEBook, StrategyReadium, true, true, streamCaps},
	{Format("zip"), []string{".zip"}, []string{"application/zip"}, KindPackage, StrategyArchive, false, true, detectCaps},
	{Format("audio"), []string{".mp3", ".m4a", ".m4b", ".aac", ".ogg", ".oga", ".opus", ".flac", ".wav", ".aax", ".aiff"}, []string{"audio/mpeg", "audio/mp4", "audio/ogg", "audio/opus", "audio/flac", "audio/wav"}, KindAudiobook, StrategyAudio, false, false, streamCaps},
}

func Detect(name string) (Descriptor, bool) {
	base := strings.ToLower(filepath.Base(name))
	for _, d := range registry {
		for _, ext := range d.Extensions {
			if len(ext) > 5 && (base == strings.TrimPrefix(ext, ".") || strings.HasSuffix(base, ext)) {
				return d, true
			}
		}
	}
	for _, d := range registry {
		for _, ext := range d.Extensions {
			if strings.HasSuffix(base, ext) {
				return d, true
			}
		}
	}
	return Descriptor{}, false
}
func ByFormat(f Format) (Descriptor, bool) {
	for _, d := range registry {
		if d.Format == f {
			return d, true
		}
	}
	return Descriptor{}, false
}
func cloneDescriptor(d Descriptor) Descriptor {
	d.Extensions = append([]string(nil), d.Extensions...)
	d.MIMETypes = append([]string(nil), d.MIMETypes...)
	return d
}
func All() []Descriptor {
	out := make([]Descriptor, len(registry))
	for i, d := range registry {
		out[i] = cloneDescriptor(d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Format < out[j].Format })
	return out
}
func Extensions() []string {
	seen := map[string]bool{}
	for _, d := range registry {
		for _, e := range d.Extensions {
			seen[e] = true
		}
	}
	out := make([]string, 0, len(seen))
	for e := range seen {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}
