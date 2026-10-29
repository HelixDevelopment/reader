// Package sync implements the deterministic offline-first synchronisation
// primitives from docs/10 (Sync Protocol) and docs/30 (Caching & Real-Time
// Sync Lifecycle): vector clocks for causality, furthest-position tracking
// for reading progress, and last-writer-wins-with-tiebreak conflict policy
// for annotations/settings.
package sync

import (
	"errors"
	"sort"
)

// Ordering is the causal relation between two vector clocks.
type Ordering int

const (
	Equal      Ordering = iota
	Before              // receiver happened-before other
	After               // receiver happened-after other
	Concurrent          // neither: conflict
)

// Clock is a vector clock keyed by device id.
type Clock struct {
	V map[string]uint64
}

func New() *Clock { return &Clock{V: map[string]uint64{}} }

func (c *Clock) Clone() *Clock {
	n := New()
	for k, v := range c.V {
		n.V[k] = v
	}
	return n
}

// Increment advances the entry for device and returns the new value.
func (c *Clock) Increment(device string) uint64 {
	c.V[device]++
	return c.V[device]
}

// Merge takes the element-wise maximum of both clocks.
func (c *Clock) Merge(o *Clock) {
	for k, v := range o.V {
		if c.V[k] < v {
			c.V[k] = v
		}
	}
}

// Compare returns the causal ordering of c relative to o.
func (c *Clock) Compare(o *Clock) Ordering {
	less, greater := false, false
	keys := map[string]bool{}
	for k := range c.V {
		keys[k] = true
	}
	for k := range o.V {
		keys[k] = true
	}
	for k := range keys {
		a, b := c.V[k], o.V[k]
		if a < b {
			less = true
		} else if a > b {
			greater = true
		}
	}
	switch {
	case less && greater:
		return Concurrent
	case less:
		return Before
	case greater:
		return After
	}
	return Equal
}

// Position is a reading position: spine index + block + char offset.
// Ordered lexicographically; furthest-position rule (docs/30): sync keeps
// the furthest valid position, never rewinds a reader.
type Position struct {
	Chapter int
	Block   int
	Offset  int
}

// Furthest returns the lexicographically larger position.
func Furthest(a, b Position) Position {
	if b.Chapter > a.Chapter ||
		(b.Chapter == a.Chapter && b.Block > a.Block) ||
		(b.Chapter == a.Chapter && b.Block == a.Block && b.Offset > a.Offset) {
		return b
	}
	return a
}

// ProgressRecord carries a position with its causal stamp.
type ProgressRecord struct {
	BookID   string
	DeviceID string
	Pos      Position
	Clock    *Clock
}

// ResolveProgress merges local and remote progress records per docs/30:
//   - if one clock dominates, the dominant record wins entirely;
//   - if clocks are equal or concurrent, the furthest position wins; ties
//     break on the lexicographically larger device id (fully deterministic).
func ResolveProgress(local, remote ProgressRecord) (ProgressRecord, error) {
	if local.BookID != remote.BookID {
		return ProgressRecord{}, errors.New("sync: progress records for different books")
	}
	winner := local
	switch local.Clock.Compare(remote.Clock) {
	case After:
		winner = local
	case Before:
		winner = remote
	case Equal, Concurrent:
		fp := Furthest(local.Pos, remote.Pos)
		if fp == remote.Pos && fp != local.Pos {
			winner = remote
		} else if fp == local.Pos && fp != remote.Pos {
			winner = local
		} else if remote.DeviceID > local.DeviceID {
			winner = remote
		}
	}
	out := winner
	out.Clock = local.Clock.Clone()
	out.Clock.Merge(remote.Clock)
	return out, nil
}

// SortedDeviceIDs provides deterministic iteration over clock keys.
func (c *Clock) SortedDeviceIDs() []string {
	ids := make([]string, 0, len(c.V))
	for k := range c.V {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	return ids
}
