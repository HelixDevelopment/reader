package catalog

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Cursor is a position in the catalog's deterministic ordering: the pair
// (updated_at DESC, id) of the last work a client has seen. It exists because
// LIMIT/OFFSET has to walk and discard every row before the page, so the cost of
// page N grows with N; a cursor asks for the rows after a position and costs the
// same at every depth.
//
// The encoding is opaque but not secret: it is a transport detail, and a client
// that constructs one by hand must still satisfy the same predicate, so nothing
// is gained by pretending it is signed.
type Cursor struct {
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
}

const cursorSeparator = "|"

// Encode renders the cursor for use in a URL query parameter.
//
// Microseconds are the smallest unit PostgreSQL stores in a timestamptz, so the
// instant survives a round trip exactly and the ordering predicate cannot
// straddle rows that differ below the stored precision.
func (c Cursor) Encode() string {
	payload := strconv.FormatInt(c.UpdatedAt.UnixMicro(), 10) + cursorSeparator + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

// ParseCursor decodes a cursor produced by Encode.
func ParseCursor(value string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, fmt.Errorf("catalog: cursor is not base64url: %w", err)
	}
	microseconds, id, found := strings.Cut(string(raw), cursorSeparator)
	if !found || id == "" {
		return Cursor{}, errors.New("catalog: cursor is missing its identifier")
	}
	micro, err := strconv.ParseInt(microseconds, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("catalog: cursor timestamp: %w", err)
	}
	return Cursor{UpdatedAt: time.UnixMicro(micro).UTC(), ID: id}, nil
}

// CursorAfter returns the cursor for the last work on a page, or nil when the
// page is empty.
//
// The timestamp is truncated to microseconds because that is all PostgreSQL
// stores. A cursor carrying a finer value could not be matched against the rows
// the database actually holds, and the row it names would be skipped.
func CursorAfter(works []Work) *Cursor {
	if len(works) == 0 {
		return nil
	}
	last := works[len(works)-1]
	return &Cursor{UpdatedAt: last.UpdatedAt.Truncate(time.Microsecond), ID: last.ID}
}
