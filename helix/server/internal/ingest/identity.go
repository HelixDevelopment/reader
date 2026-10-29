package ingest

import (
	"sort"
	"strings"

	"github.com/helixreader/helix/core/isbn"
	"github.com/helixreader/helix/server/internal/catalog"
)

// isbnIdentity selects deterministically across multiple identifiers and
// canonicalizes spelling, hyphenation, scheme labels, and ISBN-10/13 pairs.
// Invalid ISBN-like values are not allowed to displace content identity.
func isbnIdentity(e catalog.Edition) (string, bool) {
	values := make([]string, 0, len(e.Identifiers))
	seen := map[string]struct{}{}
	for _, id := range e.Identifiers {
		if !strings.HasPrefix(strings.ToLower(id.Scheme), "isbn") {
			continue
		}
		value, ok := isbn.Canonical(id.Value)
		if !ok {
			continue
		}
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return "", false
	}
	sort.Strings(values)
	return values[0], true
}

// workIdentityID groups manifestations of an ISBN-keyed edition under one
// bibliographic work while retaining each file's independent content hash.
// Without an ISBN, content identity is the only stable work key available from
// the bytes; mutable display metadata must not fork either identity.
func workIdentityID(e catalog.Edition, contentHash string) string {
	if value, ok := isbnIdentity(e); ok {
		return stableID("work", "isbn", value)
	}
	return stableID("work", contentHash)
}
