package identify

import "testing"

func TestISBN13Valid(t *testing.T) {
	// Known-valid ISBN-13 for "The Left Hand of Darkness" (Ace).
	if !ValidateISBN("978-0-441-47812-5") {
		t.Fatal("9780441478125 must validate")
	}
	if ValidateISBN("9780441478126") {
		t.Fatal("bad check digit must fail")
	}
}

func TestISBN10Valid(t *testing.T) {
	if !ValidateISBN("0-441-47812-3") {
		t.Fatal("0441478123 must validate")
	}
	// X check digit case: "The Colour of Magic" Corgi 0-552-12475-3? use classic X example
	if !ValidateISBN("080442957X") {
		t.Fatal("ISBN-10 with X check digit must validate")
	}
}

func TestISBN10to13RoundTrip(t *testing.T) {
	v13, err := ISBN10to13("0441478123")
	if err != nil || v13 != "9780441478125" {
		t.Fatalf("got %q %v", v13, err)
	}
	back, err := ISBN13to10(v13)
	if err != nil || back != "0441478123" {
		t.Fatalf("round trip failed: %q %v", back, err)
	}
}

func TestISBN13to10Rejects979(t *testing.T) {
	if _, err := ISBN13to10("9791090636071"); err != ErrInvalidISBN {
		t.Fatalf("979 prefix must be rejected, got %v", err)
	}
}

func TestIsASIN(t *testing.T) {
	if !IsASIN("B003ZGZ9N2") {
		t.Fatal("valid ASIN shape rejected")
	}
	if IsASIN("b003zgz9n2") || IsASIN("B003") || IsASIN("0441478123") || IsASIN("A003ZGZ9N2") {
		t.Fatal("lowercase, short, ISBN-only, or non-B0 identifier must fail")
	}
}

func TestRankCandidatesDeterministic(t *testing.T) {
	in := []Candidate{
		{Scheme: "oclc", Value: "123", Confidence: 0.5},
		{Scheme: "isbn13", Value: "9780441478125", Confidence: 0.9},
		{Scheme: "isbn10", Value: "0441478123", Confidence: 0.9},
		{Scheme: "asin", Value: "B003ZGZ9N2", Confidence: 0.9},
	}
	out := RankCandidates(in)
	if out[0].Scheme != "isbn13" || out[1].Scheme != "isbn10" || out[2].Scheme != "asin" || out[3].Scheme != "oclc" {
		t.Fatalf("bad ranking: %+v", out)
	}
	// idempotent
	if again := RankCandidates(out); again[0] != out[0] {
		t.Fatal("ranking not stable")
	}
}
