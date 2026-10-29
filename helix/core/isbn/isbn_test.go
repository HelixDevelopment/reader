package isbn

import "testing"

func TestValidISBN13(t *testing.T) {
	for _, s := range []string{"978-0-306-40615-7", "9780306406157", "978 0 306 40615 7"} {
		if !Valid13(s) {
			t.Errorf("expected valid ISBN-13: %q", s)
		}
	}
}
func TestInvalidISBN13(t *testing.T) {
	for _, s := range []string{"978-0-306-40615-8", "9771634552217", "978163455221", "97803064061570"} {
		if Valid13(s) {
			t.Errorf("expected invalid ISBN-13: %q", s)
		}
	}
}
func TestValidISBN10(t *testing.T) {
	if !Valid10("0-306-40615-2") || !Valid10("0306406152") {
		t.Error("valid ISBN-10 rejected")
	}
	if Valid10("0-306-40615-3") {
		t.Error("bad check digit accepted")
	}
}
func TestCanonicalISBN(t *testing.T) {
	for _, value := range []string{"0-306-40615-2", "0306406152", "978-0-306-40615-7", "9780306406157"} {
		got, ok := Canonical(value)
		if !ok || got != "9780306406157" {
			t.Errorf("Canonical(%q) = %q, %v", value, got, ok)
		}
	}
	if got, ok := Canonical("978-0-306-40615-8"); ok || got != "" {
		t.Fatalf("invalid ISBN canonicalized: %q, %v", got, ok)
	}
}

func TestExtractFromPage(t *testing.T) {
	page := "First published 2024. ISBN 978-0-306-40615-7 (hb). Also 0-306-40615-2. Bad: 978-1-63455-221-8. Repeat: 978-0-306-40615-7."
	got := Extract(page)
	if len(got) != 2 || got[0] != "9780306406157" || got[1] != "0306406152" {
		t.Fatalf("unexpected: %v", got)
	}
}
func TestExtractAdjacentSpaceSeparatedISBNs(t *testing.T) {
	got := Extract("9780123456786 9780596007126")
	if len(got) != 2 || got[0] != "9780123456786" || got[1] != "9780596007126" {
		t.Fatalf("adjacent ISBNs must both survive: %v", got)
	}
}

func TestExtractEmpty(t *testing.T) {
	if got := Extract("no identifiers here"); len(got) != 0 {
		t.Fatalf("expected none, got %v", got)
	}
}
