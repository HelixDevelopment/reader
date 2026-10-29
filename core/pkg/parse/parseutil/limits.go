// Package parseutil supplies shared resource bounds for untrusted book parsers.
package parseutil

import (
	"fmt"
	"io"
)

const (
	MaxEntryBytes     int64 = 16 << 20
	MaxImageBytes     int64 = 32 << 20
	MaxTextBytes      int64 = 64 << 20
	MaxAggregateBytes int64 = 128 << 20
	MaxPages                = 10000
	MaxArchiveEntries       = 100000
	MaxXMLDepth             = 256
)

// ReadAll reads at most limit bytes and rejects, rather than truncates, an
// entry that exceeds the limit. This prevents compressed input from causing
// unbounded allocation or silently changing parser output.
func ReadAll(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("parse: invalid read limit")
	}
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("parse: entry exceeds %d byte limit", limit)
	}
	return data, nil
}
