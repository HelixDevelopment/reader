package parseutil

import (
	"bytes"
	"testing"
)

func TestReadAllBounds(t *testing.T) {
	data, err := ReadAll(bytes.NewReader([]byte("1234")), 4)
	if err != nil || string(data) != "1234" {
		t.Fatalf("exact limit: %q, %v", data, err)
	}
	if _, err := ReadAll(bytes.NewReader([]byte("12345")), 4); err == nil {
		t.Fatal("overflow must be rejected")
	}
	if _, err := ReadAll(bytes.NewReader(nil), 0); err == nil {
		t.Fatal("non-positive limit must be rejected")
	}
}
