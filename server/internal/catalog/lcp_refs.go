package catalog

import "context"

// ReferencedLCPArtifacts returns the DISTINCT SHA-256 of every fulfilled-LCP
// file (drm_scheme 'lcp' with a license href) in one consistent read, so the
// cache collector never depends on paging order while works are being updated.
func (m *Memory) ReferencedLCPArtifacts(ctx context.Context) (map[string]bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	refs := map[string]bool{}
	for _, w := range m.works {
		for _, e := range w.Editions {
			for _, f := range e.Files {
				if f.DRMScheme == "lcp" && f.LicenseHREF != "" {
					refs[f.SHA256] = true
				}
			}
		}
	}
	return refs, nil
}

// ReferencedLCPArtifacts is a single SELECT DISTINCT: one statement, one
// snapshot, immune to concurrent updated_at movement.
func (p *Postgres) ReferencedLCPArtifacts(ctx context.Context) (map[string]bool, error) {
	rows, err := p.db.QueryContext(ctx, "SELECT DISTINCT sha256 FROM files WHERE drm_scheme='lcp' AND license_href<>''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	refs := map[string]bool{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		refs[s] = true
	}
	return refs, rows.Err()
}
