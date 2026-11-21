package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// getWorks loads a set of works in a fixed number of round trips: one per
// table, regardless of how many works or editions are requested. Search used to
// call GetWork per row, which cost 2 + 3*editions queries for every work on the
// page — hundreds of serialized round trips for a full catalog page.
//
// Returned works follow the order of ids. Column lists and orderings match
// getWork exactly so both paths describe the same rows identically. Each result
// set is closed before the next query starts: the connection pool is bounded, and
// holding one connection per pending page against a small MaxOpenConns would
// stall.
func (p *Postgres) getWorks(ctx context.Context, ids []string) ([]Work, error) {
	if len(ids) == 0 {
		return []Work{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	list := strings.Join(placeholders, ",")

	byID := make(map[string]*Work, len(ids))
	order := make([]string, 0, len(ids))
	workRows, err := p.db.QueryContext(ctx, "SELECT id,canonical_title,series,series_index,subjects,created_at,updated_at FROM works WHERE id IN ("+list+")", args...)
	if err != nil {
		return nil, err
	}
	for workRows.Next() {
		var w Work
		var subjects []byte
		if err := workRows.Scan(&w.ID, &w.CanonicalTitle, &w.Series, &w.SeriesIndex, &subjects, &w.CreatedAt, &w.UpdatedAt); err != nil {
			workRows.Close()
			return nil, err
		}
		if err := json.Unmarshal(subjects, &w.Subjects); err != nil {
			workRows.Close()
			return nil, err
		}
		record := w
		byID[w.ID] = &record
		order = append(order, w.ID)
	}
	if err := workRows.Err(); err != nil {
		workRows.Close()
		return nil, err
	}
	workRows.Close()

	editionIDs := []string{}
	editionRows, err := p.db.QueryContext(ctx, "SELECT work_id,id,title,subtitle,authors,publisher,published,language,description,cover_href,state,confidence,updated_at FROM editions WHERE work_id IN ("+list+") ORDER BY work_id,id", args...)
	if err != nil {
		return nil, err
	}
	for editionRows.Next() {
		var workID string
		var e Edition
		var authors []byte
		if err := editionRows.Scan(&workID, &e.ID, &e.Title, &e.Subtitle, &authors, &e.Publisher, &e.Published, &e.Language, &e.Description, &e.CoverHREF, &e.State, &e.Confidence, &e.UpdatedAt); err != nil {
			editionRows.Close()
			return nil, err
		}
		if err := json.Unmarshal(authors, &e.Authors); err != nil {
			editionRows.Close()
			return nil, err
		}
		parent := byID[workID]
		if parent == nil {
			// An edition whose work is not in the requested set belongs to a
			// concurrent delete; skipping it keeps this page self-consistent.
			continue
		}
		parent.Editions = append(parent.Editions, e)
		editionIDs = append(editionIDs, e.ID)
	}
	if err := editionRows.Err(); err != nil {
		editionRows.Close()
		return nil, err
	}
	editionRows.Close()
	if len(editionIDs) == 0 {
		return assemble(ids, byID), nil
	}

	childPlaceholders := make([]string, len(editionIDs))
	childArgs := make([]any, len(editionIDs))
	for i, id := range editionIDs {
		childPlaceholders[i] = fmt.Sprintf("$%d", i+1)
		childArgs[i] = id
	}
	childList := strings.Join(childPlaceholders, ",")
	children := make(map[string]*Edition)
	for _, id := range order {
		work := byID[id]
		for j := range work.Editions {
			children[work.Editions[j].ID] = &work.Editions[j]
		}
	}

	identifiers, err := p.db.QueryContext(ctx, "SELECT edition_id,scheme,value,source,confidence FROM identifiers WHERE edition_id IN ("+childList+") ORDER BY edition_id,scheme,value,source", childArgs...)
	if err != nil {
		return nil, err
	}
	for identifiers.Next() {
		var editionID string
		var id Identifier
		if err := identifiers.Scan(&editionID, &id.Scheme, &id.Value, &id.Source, &id.Confidence); err != nil {
			identifiers.Close()
			return nil, err
		}
		if e := children[editionID]; e != nil {
			e.Identifiers = append(e.Identifiers, id)
		}
	}
	if err := identifiers.Err(); err != nil {
		identifiers.Close()
		return nil, err
	}
	identifiers.Close()

	files, err := p.db.QueryContext(ctx, "SELECT edition_id,id,source_id,path,format,mime_type,size,sha256,protected,drm_scheme,license_href,modified_at FROM files WHERE edition_id IN ("+childList+") ORDER BY edition_id,id", childArgs...)
	if err != nil {
		return nil, err
	}
	for files.Next() {
		var editionID string
		var f File
		if err := files.Scan(&editionID, &f.ID, &f.SourceID, &f.Path, &f.Format, &f.MIMEType, &f.Size, &f.SHA256, &f.Protected, &f.DRMScheme, &f.LicenseHREF, &f.ModifiedAt); err != nil {
			files.Close()
			return nil, err
		}
		if e := children[editionID]; e != nil {
			e.Files = append(e.Files, f)
		}
	}
	if err := files.Err(); err != nil {
		files.Close()
		return nil, err
	}
	files.Close()

	evidence, err := p.db.QueryContext(ctx, "SELECT edition_id,field,provider,confidence,observed_at FROM metadata_evidence WHERE edition_id IN ("+childList+") ORDER BY edition_id,field,provider,observed_at", childArgs...)
	if err != nil {
		return nil, err
	}
	for evidence.Next() {
		var editionID string
		var ev MetadataEvidence
		if err := evidence.Scan(&editionID, &ev.Field, &ev.Provider, &ev.Confidence, &ev.ObservedAt); err != nil {
			evidence.Close()
			return nil, err
		}
		if e := children[editionID]; e != nil {
			e.Evidence = append(e.Evidence, ev)
		}
	}
	if err := evidence.Err(); err != nil {
		evidence.Close()
		return nil, err
	}
	evidence.Close()
	return assemble(ids, byID), nil
}

// assemble returns works in the requested order. A work deleted between the page
// query and this load is reported as absent rather than as an error.
func assemble(ids []string, byID map[string]*Work) []Work {
	out := make([]Work, 0, len(byID))
	for _, id := range ids {
		if w := byID[id]; w != nil {
			out = append(out, *w)
		}
	}
	return out
}
