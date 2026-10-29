package theme

import (
	"strings"
	"testing"
)

const dayBrand = `{
  "helixtheme": 1,
  "name": "helix_day_brand",
  "base": "day",
  "colors": {
    "background": "#FDFBF7",
    "text": "#1A1A1A",
    "accent": "#B4E212",
    "muted": "#6B6B6B",
    "highlight": "#B8ECD7"
  },
  "typography": {"font_family": "Literata", "font_size_sp": 17, "line_height": 1.45},
  "page": {"margin_dp": 16, "two_column_min_width_dp": 600, "page_curl_animation": true, "allow_brightness_dim": true}
}`

func TestParseValid(t *testing.T) {
	th, err := Parse([]byte(dayBrand))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if th.Name != "helix_day_brand" || th.Colors.Accent != "#B4E212" {
		t.Fatalf("bad parse: %+v", th)
	}
}

func TestParseBadVersion(t *testing.T) {
	_, err := Parse([]byte(`{"helixtheme": 2}`))
	if err != ErrBadVersion {
		t.Fatalf("want ErrBadVersion, got %v", err)
	}
}

func TestParseBadColor(t *testing.T) {
	_, err := Parse([]byte(strings.Replace(dayBrand, "#B4E212", "lime!", 1)))
	if err == nil || !strings.Contains(err.Error(), "accent") {
		t.Fatalf("want accent color error, got %v", err)
	}
}

func TestContrastMath(t *testing.T) {
	th, _ := Parse([]byte(dayBrand))
	r := th.Contrast()
	// #1A1A1A on #FDFBF7 must be very high contrast.
	if !r.PassesAAA_NormalText {
		t.Fatalf("day brand text must pass AAA, got ratio %.2f", r.TextOnBackground)
	}
	if r.TextOnBackground < 15 {
		t.Fatalf("expected ratio > 15, got %.2f", r.TextOnBackground)
	}
}

func TestContrastWhiteOnBlackIs21(t *testing.T) {
	th := &Theme{HelixVer: 1, Colors: Colors{Background: "#000000", Text: "#FFFFFF",
		Accent: "#888888", Muted: "#888888", Highlight: "#888888"},
		Typo: Typo{FontFamily: "x", FontSizeSp: 16, LineHeight: 1.4}}
	if err := th.Validate(); err != nil {
		t.Fatal(err)
	}
	r := th.Contrast()
	if r.TextOnBackground < 20.9 || r.TextOnBackground > 21.1 {
		t.Fatalf("white-on-black must be 21:1, got %.3f", r.TextOnBackground)
	}
}

func TestValidateRejectsLowTextContrast(t *testing.T) {
	bad := strings.Replace(dayBrand, `"text": "#1A1A1A"`, `"text": "#FDFBF7"`, 1)
	if _, err := Parse([]byte(bad)); err == nil || !strings.Contains(err.Error(), "contrast") {
		t.Fatalf("1:1 text contrast must be rejected, got %v", err)
	}
}

func TestLineHeightBounds(t *testing.T) {
	bad := strings.Replace(dayBrand, `"line_height": 1.45`, `"line_height": 4.0`, 1)
	if _, err := Parse([]byte(bad)); err == nil {
		t.Fatal("line_height 4.0 must be rejected")
	}
}
