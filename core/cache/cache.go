// Package cache implements the book cache eviction policy (docs/30):
// LRU within budget, but books that are being read, annotated, or pinned
// are NEVER evicted (zero-data-loss covenant).
package cache

import "container/list"

type Entry struct {
	BookID    string
	Bytes     int64
	Protected bool // reading / has annotations / pinned
}

type Cache struct {
	budget int64
	used   int64
	ll     *list.List // front = most recently used
	items  map[string]*list.Element
}

func New(budgetBytes int64) *Cache {
	return &Cache{budget: budgetBytes, ll: list.New(), items: map[string]*list.Element{}}
}

// Touch records access; inserts if absent. Returns false if bytes > budget.
func (c *Cache) Touch(e Entry) bool {
	if e.Bytes < 0 || e.Bytes > c.budget {
		return false
	}
	if el, ok := c.items[e.BookID]; ok {
		ent := el.Value.(Entry)
		c.used += e.Bytes - ent.Bytes
		// Protection is sticky: a routine refresh must not silently make a
		// pinned/annotated/reading entry eligible for eviction.
		e.Protected = e.Protected || ent.Protected
		el.Value = e
		c.ll.MoveToFront(el)
		return true
	}
	c.items[e.BookID] = c.ll.PushFront(e)
	c.used += e.Bytes
	return true
}

// EvictTo frees space down to target, skipping protected entries.
// Returns evicted book IDs. If only protected entries remain above target,
// eviction stops (documented behaviour — user is warned instead).
func (c *Cache) EvictTo(target int64) []string {
	var evicted []string
	for c.used > target {
		// scan from back for the first non-protected entry
		var victim *list.Element
		for el := c.ll.Back(); el != nil; el = el.Prev() {
			if !el.Value.(Entry).Protected {
				victim = el
				break
			}
		}
		if victim == nil {
			break // everything left is protected
		}
		ent := victim.Value.(Entry)
		c.ll.Remove(victim)
		delete(c.items, ent.BookID)
		c.used -= ent.Bytes
		evicted = append(evicted, ent.BookID)
	}
	return evicted
}

func (c *Cache) Used() int64 { return c.used }
func (c *Cache) Len() int    { return c.ll.Len() }
