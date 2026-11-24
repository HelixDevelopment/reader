package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/helixreader/helix/core/pkg/lcp"
	"github.com/helixreader/helix/server/internal/ingest/lcpfixture"
)

// Defect 2: a same-size byte flip must not be served under the original hash.
func TestLCPContentRejectsSameSizeTamper(t *testing.T) {
	r := newLCPRig(t, false)
	url := "/v1/files/" + r.file.ID + "/content"
	if w := r.get(t, http.MethodGet, url, nil, false); w.Code != 200 {
		t.Fatalf("baseline: %d", w.Code)
	}
	path := r.srv.Ingest.LCP.ArtifactPath(r.file.SHA256)
	b, _ := os.ReadFile(path)
	b[len(b)/2] ^= 0xff
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	w := r.get(t, http.MethodGet, url, nil, false)
	if w.Code != http.StatusConflict {
		t.Fatalf("tampered artifact served: status=%d etag=%s", w.Code, w.Header().Get("ETag"))
	}
	if bytes.Equal(w.Body.Bytes(), b) {
		t.Fatal("tampered bytes in response body")
	}
}

func TestLCPContentTamperDetectedOnFirstRequest(t *testing.T) {
	r := newLCPRig(t, false)
	path := r.srv.Ingest.LCP.ArtifactPath(r.file.SHA256)
	b, _ := os.ReadFile(path)
	b[len(b)/3] ^= 0x01
	os.WriteFile(path, b, 0o600)
	if w := r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/content", nil, false); w.Code != http.StatusConflict {
		t.Fatalf("status=%d", w.Code)
	}
}

// Defect 3: /license is the license embedded in the served artifact, even when
// the source .lcpl was replaced afterwards.
func TestLCPLicenseIsTheEmbeddedLicenseNotTheLiveSource(t *testing.T) {
	r := newLCPRig(t, false)
	other := lcpfixture.Build(t, "http://127.0.0.1:1/p.epub", nil)
	if err := os.WriteFile(filepath.Join(r.root, "Sealed Book.lcpl"), other.License, 0o644); err != nil {
		t.Fatal(err)
	}
	c, _ := r.srv.connector("src-1")
	r.srv.Ingest.ScanSource(context.Background(), "src-1", c, "") // fetch of the new license fails
	lic := r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/license", nil, false)
	cont := r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/content", nil, false)
	if cont.Code != 200 {
		t.Fatalf("content: %d", cont.Code)
	}
	ctr, err := lcp.OpenContainer(bytes.NewReader(cont.Body.Bytes()), int64(cont.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if lic.Code != 200 || !bytes.Equal(lic.Body.Bytes(), ctr.License.Raw) {
		t.Fatalf("license %d differs from the embedded license", lic.Code)
	}
	if bytes.Equal(lic.Body.Bytes(), other.License) {
		t.Fatal("served the live source license that does not match the artifact")
	}
}

func TestLCPLicenseDoesNotNeedTheSource(t *testing.T) {
	r := newLCPRig(t, false)
	os.Remove(filepath.Join(r.root, "Sealed Book.lcpl"))
	w := r.get(t, http.MethodGet, "/v1/files/"+r.file.ID+"/license", nil, false)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), r.fx.License) {
		t.Fatalf("status %d", w.Code)
	}
}
