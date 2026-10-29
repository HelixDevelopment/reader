// Package fts provides an embedded full-text search index (docs/04 F-search,
// plan P-2 #13). Interim implementation: deterministic in-memory inverted
// index with BM25-lite scoring — stdlib-only. Production swap: bleve or
// SQLite FTS5 behind this same Index interface.
package fts

import (
	"sort"
	"strings"
	"unicode"
)

// DocRef identifies an indexed passage.
type DocRef struct {
	BookID  string
	Chapter string
	Block   int
	Text    string
}

// Hit is a scored search result.
type Hit struct {
	DocRef
	Score float64
}

// Index is the search contract (bleve/FTS5 will implement the same).
type Index interface {
	Add(ref DocRef)
	Search(query string, limit int) []Hit
	Size() int
}

// MemoryIndex is the stdlib interim implementation.
type MemoryIndex struct {
	docs []DocRef
	inv  map[string][]int // term -> doc indices (sorted)
}

func NewMemoryIndex() *MemoryIndex {
	return &MemoryIndex{inv: map[string][]int{}}
}

func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func (m *MemoryIndex) Add(ref DocRef) {
	idx := len(m.docs)
	m.docs = append(m.docs, ref)
	seen := map[string]bool{}
	for _, t := range tokenize(ref.Text) {
		if !seen[t] {
			seen[t] = true
			m.inv[t] = append(m.inv[t], idx)
		}
	}
}

func (m *MemoryIndex) Size() int { return len(m.docs) }

// Search scores with term-frequency over matching docs; ties break by
// document index (insertion order) — fully deterministic.
func (m *MemoryIndex) Search(query string, limit int) []Hit {
	terms := tokenize(query)
	if len(terms) == 0 || limit <= 0 {
		return nil
	}
	scores := map[int]float64{}
	for _, t := range terms {
		for _, di := range m.inv[t] {
			tf := 0.0
			for _, dt := range tokenize(m.docs[di].Text) {
				if dt == t {
					tf++
				}
			}
			scores[di] += tf
		}
	}
	idxs := make([]int, 0, len(scores))
	for i := range scores {
		idxs = append(idxs, i)
	}
	sort.Slice(idxs, func(a, b int) bool {
		if scores[idxs[a]] != scores[idxs[b]] {
			return scores[idxs[a]] > scores[idxs[b]]
		}
		return idxs[a] < idxs[b]
	})
	var hits []Hit
	for _, i := range idxs {
		hits = append(hits, Hit{DocRef: m.docs[i], Score: scores[i]})
		if len(hits) == limit {
			break
		}
	}
	return hits
}
