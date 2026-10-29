package catalog

import (
	"fmt"
	"sort"
)

// rehomeLocked applies the same source/path identity rules as PostgreSQL.
// The caller must hold m.mu for writing. Editions that become fileless because
// their last file moved are removed; pre-existing metadata-only editions stay.
func (m *Memory) rehomeLocked(incoming Work) error {
	workIDs := make([]string, 0, len(m.works))
	for workID := range m.works {
		workIDs = append(workIDs, workID)
	}
	sort.Strings(workIDs)
	for _, edition := range incoming.Editions {
		for _, workID := range workIDs {
			existing := m.works[workID]
			if workID == incoming.ID {
				continue
			}
			for _, candidate := range existing.Editions {
				if candidate.ID == edition.ID {
					return fmt.Errorf("catalog: edition %s already belongs to work %s", edition.ID, workID)
				}
			}
		}
	}
	type fileIdentity struct {
		id       string
		sourceID string
		path     string
	}
	var incomingFiles []fileIdentity
	for _, edition := range incoming.Editions {
		for _, file := range edition.Files {
			incomingFiles = append(incomingFiles, fileIdentity{id: file.ID, sourceID: file.SourceID, path: file.Path})
		}
	}
	if len(incomingFiles) == 0 {
		return nil
	}
	for _, workID := range workIDs {
		existing := m.works[workID]
		if workID == incoming.ID {
			continue
		}
		changed := false
		editions := existing.Editions[:0]
		for _, edition := range existing.Editions {
			removed := 0
			files := edition.Files[:0]
			for _, file := range edition.Files {
				move := false
				for _, incoming := range incomingFiles {
					if file.ID == incoming.id || (file.SourceID == incoming.sourceID && file.Path == incoming.path) {
						move = true
						break
					}
				}
				if move {
					removed++
					changed = true
					continue
				}
				files = append(files, file)
			}
			edition.Files = files
			if removed > 0 && len(files) == 0 {
				continue
			}
			editions = append(editions, edition)
		}
		if !changed {
			continue
		}
		existing.Editions = editions
		if len(editions) == 0 {
			delete(m.works, workID)
			continue
		}
		m.works[workID] = existing
	}
	return nil
}
