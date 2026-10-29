package formats

import (
	"reflect"
	"testing"
)

func TestMandatedThoriumAndLCPExtensions(t *testing.T) {
	names := []string{
		"license.lcpl", "book.epub", "book.epub3", "book.pnld", "book.audiobook",
		"book.webpub", "book.lcpa", "book.lcpaudiobook", "book.lcpdf", "book.pdf",
		"book.lpf", "book.divina", "book.daisy", "book.zip", "book.opf", "NCC.HTML",
	}
	for _, name := range names {
		if _, ok := Detect(name); !ok {
			t.Errorf("mandated name %q is not detected", name)
		}
	}
}

func TestDetectCompoundBeforeGeneric(t *testing.T) {
	cases := map[string]Format{
		"BOOK.FB2.ZIP":  Format("fb2_zip"),
		"book.lcp.epub": Format("lcp_epub"),
		"NCC.HTML":      Format("daisy"),
		"book.epub3":    EPUB,
	}
	for name, want := range cases {
		got, ok := Detect(name)
		if !ok || got.Format != want {
			t.Errorf("%s: got %+v, %v; want %s", name, got, ok, want)
		}
	}
}

func TestProtectedFormatsDoNotClaimConversion(t *testing.T) {
	for _, f := range []Format{"lcpl", "lcpa", "lcpdf", "lcp_epub"} {
		d, ok := ByFormat(f)
		if !ok || !d.Protected {
			t.Errorf("%s has incorrect LCP contract: %+v", f, d)
		}
		if d.Capabilities.DRMFulfill {
			t.Errorf("%s cannot claim unimplemented DRM fulfillment", f)
		}
		if d.Capabilities.Convert {
			t.Errorf("%s must not silently permit DRM-removing conversion", f)
		}
	}
}

func TestRegistrySnapshotIsSortedAndIndependent(t *testing.T) {
	a := All()
	if len(a) < 25 {
		t.Fatalf("registry unexpectedly small: %d", len(a))
	}
	for i := 1; i < len(a); i++ {
		if a[i-1].Format >= a[i].Format {
			t.Fatal("formats not strictly sorted")
		}
	}
	a[0].Format = "mutation"
	b := All()
	if b[0].Format == "mutation" {
		t.Fatal("All leaked mutable registry storage")
	}
	if reflect.DeepEqual(a, b) {
		t.Fatal("mutation test failed to change snapshot")
	}
}

func TestNoDescriptorClaimsUnimplementedRenderOrDRM(t *testing.T) {
	for _, d := range All() {
		if d.Capabilities.Render {
			t.Errorf("%s claims rendering without an integrated renderer", d.Format)
		}
		if d.Capabilities.DRMFulfill {
			t.Errorf("%s claims DRM fulfillment without an integrated LCP client", d.Format)
		}
	}
}

func TestEveryDescriptorHasDetectionAndMIME(t *testing.T) {
	for _, d := range All() {
		if !d.Capabilities.Detect {
			t.Errorf("%s cannot be detected", d.Format)
		}
		if len(d.Extensions) == 0 || len(d.MIMETypes) == 0 {
			t.Errorf("%s lacks extension or MIME metadata", d.Format)
		}
	}
}
