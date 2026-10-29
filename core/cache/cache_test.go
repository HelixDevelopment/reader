package cache

import "testing"

func TestLRUEviction(t *testing.T) {
	c := New(100)
	c.Touch(Entry{"a", 40, false})
	c.Touch(Entry{"b", 40, false})
	c.Touch(Entry{"c", 40, false})
	ev := c.EvictTo(80)
	if len(ev) != 1 || ev[0] != "a" { // least recently used first
		t.Fatalf("expected to evict a, got %v", ev)
	}
	if c.Used() != 80 {
		t.Fatalf("used=%d", c.Used())
	}
}

func TestProtectedNeverEvicted(t *testing.T) {
	c := New(100)
	c.Touch(Entry{"reading", 60, true}) // currently being read
	c.Touch(Entry{"other", 60, false})
	ev := c.EvictTo(50)
	if len(ev) != 1 || ev[0] != "other" {
		t.Fatalf("must evict unprotected only, got %v", ev)
	}
	// now only protected remains, over budget: eviction must stop, not violate
	c2 := New(50)
	c2.Touch(Entry{"pinned", 60, true})
	if ev := c2.EvictTo(10); len(ev) != 0 {
		t.Fatalf("protected entry evicted: %v", ev)
	}
}

func TestProtectionSurvivesRoutineTouch(t *testing.T) {
	c := New(100)
	c.Touch(Entry{BookID: "pinned", Bytes: 60, Protected: true})
	c.Touch(Entry{BookID: "pinned", Bytes: 60, Protected: false})
	c.Touch(Entry{BookID: "other", Bytes: 40})
	if evicted := c.EvictTo(10); len(evicted) != 1 || evicted[0] != "other" || c.Len() != 1 {
		t.Fatalf("sticky protection violated: evicted=%v len=%d", evicted, c.Len())
	}
}

func TestOversizedRejected(t *testing.T) {
	c := New(10)
	if c.Touch(Entry{"huge", 100, false}) {
		t.Fatal("oversized entry must be rejected")
	}
}
