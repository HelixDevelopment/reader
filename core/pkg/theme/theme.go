// Package theme loads and validates .helixtheme files (docs/12, docs/33).
// The wire format is JSON (subset of the open spec) so the parser is
// dependency-free and identical on every platform. Validation enforces the
// WCAG contrast floors mandated by docs/16 (Accessibility).
package theme

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Theme is a parsed .helixtheme file.
type Theme struct {
	Name     string `json:"name"`
	Base     string `json:"base"` // "day" | "night" | "sepia" | "custom"
	Colors   Colors `json:"colors"`
	Typo     Typo   `json:"typography"`
	Page     Page   `json:"page"`
	HelixVer int    `json:"helixtheme"` // spec version, must be 1
}

type Colors struct {
	Background string `json:"background"`
	Text       string `json:"text"`
	Accent     string `json:"accent"`
	Muted      string `json:"muted"`
	Highlight  string `json:"highlight"`
}

type Typo struct {
	FontFamily string  `json:"font_family"`
	FontSizeSp float64 `json:"font_size_sp"`
	LineHeight float64 `json:"line_height"` // multiplier, 1.0-3.0
}

type Page struct {
	MarginDp      int  `json:"margin_dp"`
	TwoColumnMin  int  `json:"two_column_min_width_dp"`
	PageCurl      bool `json:"page_curl_animation"`
	BrightnessDim bool `json:"allow_brightness_dim"`
}

// ErrBadVersion for unsupported spec versions.
var ErrBadVersion = errors.New("theme: unsupported helixtheme spec version")

// Parse decodes and validates a .helixtheme document.
func Parse(data []byte) (*Theme, error) {
	var t Theme
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("theme: invalid json: %w", err)
	}
	if t.HelixVer != 1 {
		return nil, ErrBadVersion
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return &t, nil
}

// Validate checks structural and accessibility invariants.
func (t *Theme) Validate() error {
	for _, kv := range [][2]string{
		{"background", t.Colors.Background}, {"text", t.Colors.Text},
		{"accent", t.Colors.Accent}, {"muted", t.Colors.Muted},
		{"highlight", t.Colors.Highlight},
	} {
		if _, err := parseHex(kv[1]); err != nil {
			return fmt.Errorf("theme: colors.%s: %w", kv[0], err)
		}
	}
	if ratio := t.Contrast().TextOnBackground; ratio < 4.5 {
		return fmt.Errorf("theme: text/background contrast %.2f below WCAG AA normal-text floor 4.5", ratio)
	}
	if t.Typo.LineHeight < 1.0 || t.Typo.LineHeight > 3.0 {
		return fmt.Errorf("theme: line_height %.2f outside [1.0,3.0]", t.Typo.LineHeight)
	}
	if t.Typo.FontSizeSp < 6 || t.Typo.FontSizeSp > 64 {
		return fmt.Errorf("theme: font_size_sp %.1f outside [6,64]", t.Typo.FontSizeSp)
	}
	return nil
}

// ContrastReport holds WCAG contrast ratios between key color pairs.
type ContrastReport struct {
	TextOnBackground      float64
	AccentOnBackground    float64
	HighlightOnBackground float64
	PassesAA_NormalText   bool // >= 4.5
	PassesAAA_NormalText  bool // >= 7.0
}

// Contrast computes the WCAG 2.x contrast report for this theme.
func (t *Theme) Contrast() ContrastReport {
	bg, _ := parseHex(t.Colors.Background)
	fg, _ := parseHex(t.Colors.Text)
	ac, _ := parseHex(t.Colors.Accent)
	hi, _ := parseHex(t.Colors.Highlight)
	r := ContrastReport{
		TextOnBackground:      ratio(luminance(fg), luminance(bg)),
		AccentOnBackground:    ratio(luminance(ac), luminance(bg)),
		HighlightOnBackground: ratio(luminance(hi), luminance(bg)),
	}
	r.PassesAA_NormalText = r.TextOnBackground >= 4.5
	r.PassesAAA_NormalText = r.TextOnBackground >= 7.0
	return r
}

type rgb struct{ r, g, b float64 }

func parseHex(s string) (rgb, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return rgb{}, fmt.Errorf("invalid hex color %q", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return rgb{}, fmt.Errorf("invalid hex color %q", s)
	}
	return rgb{float64(v>>16&0xFF) / 255, float64(v>>8&0xFF) / 255, float64(v&0xFF) / 255}, nil
}

func channel(c float64) float64 {
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func luminance(c rgb) float64 {
	return 0.2126*channel(c.r) + 0.7152*channel(c.g) + 0.0722*channel(c.b)
}

func ratio(l1, l2 float64) float64 {
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}
