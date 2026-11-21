package catalog

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Memory is a deterministic, concurrency-safe development and test store.
// It performs deep copies at its boundary so callers cannot mutate stored
// catalog state without a write transaction.
type Memory struct {
	mu    sync.RWMutex
	works map[string]Work
}

func NewMemory() *Memory { return &Memory{works: map[string]Work{}} }

func cloneEdition(e Edition) Edition {
	out := e
	out.Authors = append([]string(nil), e.Authors...)
	out.Identifiers = append([]Identifier(nil), e.Identifiers...)
	out.Files = append([]File(nil), e.Files...)
	out.Evidence = append([]MetadataEvidence(nil), e.Evidence...)
	return out
}

func cloneWork(w Work) Work {
	out := w
	out.Subjects = append([]string(nil), w.Subjects...)
	out.Editions = make([]Edition, len(w.Editions))
	for i, e := range w.Editions {
		out.Editions[i] = cloneEdition(e)
	}
	return out
}

func (m *Memory) UpsertWork(_ context.Context, w Work) error {
	if err := validateWork(w); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.rehomeLocked(w); err != nil {
		return err
	}
	m.works[w.ID] = cloneWork(w)
	return nil
}

func (m *Memory) MergeWork(_ context.Context, w Work) error {
	if err := validateWork(w); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.rehomeLocked(w); err != nil {
		return err
	}
	if existing, ok := m.works[w.ID]; ok {
		w = MergeWork(existing, w)
	}
	m.works[w.ID] = cloneWork(w)
	return nil
}

func (m *Memory) ReconcileSource(ctx context.Context, sourceID string, present []string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if strings.TrimSpace(sourceID) == "" {
		return 0, fmt.Errorf("catalog: source id is required for reconciliation")
	}
	retained := make(map[string]struct{}, len(present))
	for _, path := range present {
		retained[path] = struct{}{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	removed := 0
	workIDs := make([]string, 0, len(m.works))
	for workID := range m.works {
		workIDs = append(workIDs, workID)
	}
	sort.Strings(workIDs)
	for _, workID := range workIDs {
		work := m.works[workID]
		editions := work.Editions[:0]
		for _, edition := range work.Editions {
			removedFromEdition := 0
			files := edition.Files[:0]
			for _, file := range edition.Files {
				_, stillPresent := retained[file.Path]
				if file.SourceID == sourceID && !stillPresent {
					removed++
					removedFromEdition++
					continue
				}
				files = append(files, file)
			}
			edition.Files = files
			if removedFromEdition > 0 && len(files) == 0 {
				continue
			}
			editions = append(editions, edition)
		}
		work.Editions = editions
		if len(editions) == 0 {
			delete(m.works, workID)
			continue
		}
		m.works[workID] = work
	}
	return removed, nil
}

func (m *Memory) GetWork(_ context.Context, id string) (Work, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	w, ok := m.works[id]
	if !ok {
		return Work{}, ErrNotFound
	}
	return cloneWork(w), nil
}

func (m *Memory) FindEdition(ctx context.Context, id string) (Work, Edition, error) {
	if err := ctx.Err(); err != nil {
		return Work{}, Edition{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.works))
	for workID := range m.works {
		ids = append(ids, workID)
	}
	sort.Strings(ids)
	for _, workID := range ids {
		w := m.works[workID]
		for _, e := range w.Editions {
			if e.ID == id {
				return cloneWork(w), cloneEdition(e), nil
			}
		}
	}
	return Work{}, Edition{}, ErrNotFound
}

func (m *Memory) FindFile(ctx context.Context, id string) (Work, Edition, File, error) {
	if err := ctx.Err(); err != nil {
		return Work{}, Edition{}, File{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.works))
	for workID := range m.works {
		ids = append(ids, workID)
	}
	sort.Strings(ids)
	for _, workID := range ids {
		w := m.works[workID]
		for _, e := range w.Editions {
			for _, f := range e.Files {
				if f.ID == id {
					return cloneWork(w), cloneEdition(e), f, nil
				}
			}
		}
	}
	return Work{}, Edition{}, File{}, ErrNotFound
}

func (m *Memory) Search(_ context.Context, q Query) (Page, error) {
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	text := strings.ToLower(strings.TrimSpace(q.Text))
	author := strings.ToLower(strings.TrimSpace(q.Author))
	m.mu.RLock()
	all := make([]Work, 0, len(m.works))
	for _, w := range m.works {
		if q.State != "" {
			matched := false
			for _, e := range w.Editions {
				if e.State == q.State {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		haystack := strings.ToLower(w.CanonicalTitle + " " + w.Series)
		authorMatch := author == ""
		for _, e := range w.Editions {
			haystack += " " + strings.ToLower(e.Title+" "+e.Publisher+" "+e.Description)
			for _, a := range e.Authors {
				haystack += " " + strings.ToLower(a)
				if author != "" && strings.Contains(strings.ToLower(a), author) {
					authorMatch = true
				}
			}
			for _, id := range e.Identifiers {
				haystack += " " + strings.ToLower(id.Value)
			}
		}
		if text != "" && !strings.Contains(haystack, text) {
			continue
		}
		if !authorMatch {
			continue
		}
		all = append(all, cloneWork(w))
	}
	m.mu.RUnlock()
	// Ordering and cursor comparisons happen at microsecond precision, the unit
	// PostgreSQL stores, so the same page boundaries fall in this implementation
	// as in the database-backed one.
	key := func(t time.Time) time.Time { return t.Truncate(time.Microsecond) }
	sort.Slice(all, func(i, j int) bool {
		if !key(all[i].UpdatedAt).Equal(key(all[j].UpdatedAt)) {
			return key(all[i].UpdatedAt).After(key(all[j].UpdatedAt))
		}
		return all[i].ID < all[j].ID
	})
	total := len(all)
	if q.After != nil {
		after := *q.After
		afterAt := key(after.UpdatedAt)
		kept := all[:0]
		for _, w := range all {
			at := key(w.UpdatedAt)
			if at.Before(afterAt) || (at.Equal(afterAt) && w.ID > after.ID) {
				kept = append(kept, w)
			}
		}
		all = kept
	}
	// HasNext is decided by looking one row past the page, matching the Postgres
	// implementation, so a cursor client sees identical metadata from either.
	start := 0
	if q.After == nil {
		start = q.Offset
		if start > total {
			start = total
		}
	}
	end := start + q.Limit
	if end > len(all) {
		end = len(all)
	}
	return Page{Works: all[start:end], Total: total, Offset: q.Offset, Limit: q.Limit, HasNext: len(all) > end}, nil
}

func (m *Memory) DeleteWork(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.works[id]; !ok {
		return ErrNotFound
	}
	delete(m.works, id)
	return nil
}

func (m *Memory) Close() error { return nil }
