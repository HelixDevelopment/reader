package lcp

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

// rewriteZipEntry replaces one member's bytes and rebuilds the archive so every
// CRC stays true. Editing stored bytes in place would corrupt the container for
// reasons unrelated to what a test is about.
func rewriteZipEntry(tb testing.TB, raw []byte, name, content string) []byte {
	tb.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		tb.Fatal(err)
	}
	found := false
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, zf := range zr.File {
		h := &zip.FileHeader{Name: zf.Name, Method: zf.Method}
		if zf.Name == name {
			found = true
			// Producers write this metadata uncompressed; inheriting the original
			// method would bury the edit inside a deflater.
			h.Method = zip.Store
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			tb.Fatal(err)
		}
		if h.Name == name {
			if _, err := w.Write([]byte(content)); err != nil {
				tb.Fatal(err)
			}
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			tb.Fatal(err)
		}
		_, cpErr := io.Copy(w, rc)
		rc.Close()
		if cpErr != nil {
			tb.Fatal(cpErr)
		}
	}
	if !found {
		tb.Fatalf("container has no %s; the rewrite tested nothing", name)
	}
	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	return out.Bytes()
}

// encXML wraps EncryptedData bodies in the §2.2 document element.
func encXML(bodies ...string) string {
	return `<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"` +
		` xmlns:enc="http://www.w3.org/2001/04/xmlenc#"` +
		` xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
		strings.Join(bodies, "") + `</encryption>`
}

// encData is one §2.2 declaration as a producer writes it.
func encData(id, algorithm, keyType, keyURI, target string) string {
	return `<enc:EncryptedData Id="` + id + `"><enc:EncryptionMethod Algorithm="` + algorithm + `"/>` +
		`<ds:KeyInfo><ds:RetrievalMethod URI="` + keyURI + `" Type="` + keyType + `"/></ds:KeyInfo>` +
		`<enc:CipherData><enc:CipherReference URI="` + target + `"/></enc:CipherData></enc:EncryptedData>`
}

// unprotected returns the same entries with nothing protected, for a container
// that must carry no LCP metadata at all.
func unprotected(entries []zipEntry) []zipEntry {
	out := make([]zipEntry, len(entries))
	for i, e := range entries {
		e.protected = false
		out[i] = e
	}
	return out
}

// countProtected reports how many entries the fixture marked protected, so a
// test that clears the flag cannot silently assert against nothing.
func countProtected(entries []zipEntry) int {
	n := 0
	for _, e := range entries {
		if e.protected {
			n++
		}
	}
	return n
}
