// Package hlc implements a Hybrid Logical Clock (docs/10) — the ordering
// primitive for CRDT sync that never trusts wall clocks (edge case EC-S03).
package hlc

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Timestamp struct {
	Physical int64
	Logical  uint32
}

func (t Timestamp) String() string { return fmt.Sprintf("%d.%d", t.Physical, t.Logical) }

func Parse(s string) (Timestamp, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return Timestamp{}, fmt.Errorf("bad hlc %q", s)
	}
	p, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Timestamp{}, err
	}
	l, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return Timestamp{}, err
	}
	return Timestamp{Physical: p, Logical: uint32(l)}, nil
}

func (t Timestamp) Before(o Timestamp) bool {
	return t.Physical < o.Physical || (t.Physical == o.Physical && t.Logical < o.Logical)
}
func (t Timestamp) After(o Timestamp) bool { return o.Before(t) }

type Clock struct {
	mu   sync.Mutex
	last Timestamp
	now  func() int64
}

func New() *Clock { return &Clock{now: func() int64 { return time.Now().UnixMilli() }} }
func NewWithClock(now func() int64) *Clock { return &Clock{now: now} }

func (c *Clock) Now() Timestamp {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.now()
	t := Timestamp{Physical: p}
	if p <= c.last.Physical {
		t.Physical = c.last.Physical
		t.Logical = c.last.Logical + 1
	}
	c.last = t
	return t
}

func (c *Clock) Update(remote Timestamp) Timestamp {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.now()
	switch local := c.last; {
	case p > local.Physical && p > remote.Physical:
		c.last = Timestamp{Physical: p}
	case local.Physical > remote.Physical:
		c.last = Timestamp{Physical: local.Physical, Logical: local.Logical + 1}
	case remote.Physical > local.Physical:
		c.last = Timestamp{Physical: remote.Physical, Logical: remote.Logical + 1}
	default:
		l := local.Logical
		if remote.Logical > l {
			l = remote.Logical
		}
		c.last = Timestamp{Physical: local.Physical, Logical: l + 1}
	}
	return c.last
}
