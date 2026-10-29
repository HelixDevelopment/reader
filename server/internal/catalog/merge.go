package catalog

import (
	"sort"
	"time"
)

// MergeWork combines independently discovered files and editions without
// discarding existing evidence. Caller-provided newer metadata wins only at
// edition level; files and identifiers are unioned deterministically.
func MergeWork(existing, incoming Work) Work {
	if existing.ID == "" {
		return cloneWork(incoming)
	}
	out := cloneWork(existing)
	if out.CanonicalTitle == "" {
		out.CanonicalTitle = incoming.CanonicalTitle
	}
	out.Subjects = unionStrings(out.Subjects, incoming.Subjects)
	if incoming.UpdatedAt.After(out.UpdatedAt) {
		out.UpdatedAt = incoming.UpdatedAt
	}
	for _, in := range incoming.Editions {
		idx := -1
		for i := range out.Editions {
			if out.Editions[i].ID == in.ID {
				idx = i
				break
			}
		}
		if idx < 0 {
			out.Editions = append(out.Editions, cloneWork(Work{Editions: []Edition{in}}).Editions[0])
			continue
		}
		e := &out.Editions[idx]
		if in.UpdatedAt.After(e.UpdatedAt) || e.UpdatedAt.IsZero() {
			files, evidence, ids := e.Files, e.Evidence, e.Identifiers
			created := e.UpdatedAt
			// Deep-copy the incoming edition so stored state never aliases
			// caller-owned slices.
			*e = cloneWork(Work{Editions: []Edition{in}}).Editions[0]
			e.Files, e.Evidence, e.Identifiers = files, unionEvidence(evidence, in.Evidence), unionIdentifiers(ids, in.Identifiers)
			if !created.IsZero() && e.UpdatedAt.IsZero() {
				e.UpdatedAt = created
			}
		} else {
			e.Evidence = unionEvidence(e.Evidence, in.Evidence)
		}
		e.Identifiers = unionIdentifiers(e.Identifiers, in.Identifiers)
		e.Files = unionFiles(e.Files, in.Files)
	}
	sort.Slice(out.Editions, func(i, j int) bool { return out.Editions[i].ID < out.Editions[j].ID })
	return out
}

func unionStrings(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range [][]string{a, b} {
		for _, v := range list {
			if v != "" && !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	sort.Strings(out)
	return out
}
func unionIdentifiers(a, b []Identifier) []Identifier {
	type key struct{ scheme, value string }
	seen := map[key]bool{}
	var out []Identifier
	for _, list := range [][]Identifier{a, b} {
		for _, id := range list {
			k := key{id.Scheme, id.Value}
			if !seen[k] {
				seen[k] = true
				out = append(out, id)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Scheme != out[j].Scheme {
			return out[i].Scheme < out[j].Scheme
		}
		return out[i].Value < out[j].Value
	})
	return out
}

// unionEvidence merges provenance records, deduplicating exact duplicates by
// (field, provider, observed_at) so repeated merges and re-scans never grow
// the evidence log with identical observations. Existing confidence wins on
// collision; ordering is deterministic.
func unionEvidence(a, b []MetadataEvidence) []MetadataEvidence {
	type key struct {
		field, provider string
		observedAt      time.Time
	}
	seen := map[key]bool{}
	var out []MetadataEvidence
	for _, list := range [][]MetadataEvidence{a, b} {
		for _, ev := range list {
			k := key{ev.Field, ev.Provider, ev.ObservedAt}
			if !seen[k] {
				seen[k] = true
				out = append(out, ev)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Field != out[j].Field {
			return out[i].Field < out[j].Field
		}
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].ObservedAt.Before(out[j].ObservedAt)
	})
	return out
}

func unionFiles(a, b []File) []File {
	indices := map[string]int{}
	var out []File
	for _, f := range a {
		k := f.ID
		if k == "" {
			k = f.SHA256
		}
		if _, ok := indices[k]; !ok {
			indices[k] = len(out)
			out = append(out, f)
		}
	}
	for _, f := range b {
		k := f.ID
		if k == "" {
			k = f.SHA256
		}
		if i, ok := indices[k]; ok {
			// A rescan of the same source/path is newer byte evidence, not a
			// second file. Replace stale size/hash/format metadata atomically.
			out[i] = f
			continue
		}
		indices[k] = len(out)
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Touch ensures a valid deterministic timestamp when callers have no clock.
func Touch(w *Work, t time.Time) {
	if w.CreatedAt.IsZero() {
		w.CreatedAt = t
	}
	w.UpdatedAt = t
}
