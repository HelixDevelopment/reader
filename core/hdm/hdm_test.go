package hdm

import "testing"

func sample() *Publication {
	return &Publication{
		Meta: Metadata{Title: "T", Authors: []string{"A"}},
		Spine: []SpineItem{{
			ID: "c1",
			Blocks: []Block{
				{Kind: Heading, Level: 1, Inlines: []Inline{{Text: "Chapter One"}}},
				{Kind: Paragraph, Inlines: []Inline{{Text: "one two three four five"}}},
			},
		}},
	}
}

func TestWordCount(t *testing.T) {
	if got := sample().WordCount(); got != 7 {
		t.Fatalf("want 7 words, got %d", got)
	}
}

func TestModelRoundTripShape(t *testing.T) {
	p := sample()
	if p.Spine[0].Blocks[0].Kind != Heading || p.Spine[0].Blocks[0].Level != 1 {
		t.Fatal("model shape broken")
	}
}
