package sync

import "testing"

func TestClockCompareDominance(t *testing.T) {
	a := New()
	a.Increment("phone")
	b := a.Clone()
	b.Increment("tablet")
	if b.Compare(a) != After || a.Compare(b) != Before {
		t.Fatal("dominance broken")
	}
	if a.Compare(a.Clone()) != Equal {
		t.Fatal("equal clocks must compare Equal")
	}
}

func TestClockConcurrent(t *testing.T) {
	a, b := New(), New()
	a.Increment("phone")
	b.Increment("tablet")
	if a.Compare(b) != Concurrent || b.Compare(a) != Concurrent {
		t.Fatal("disjoint clocks must be Concurrent")
	}
}

func TestMergeIdempotentCommutative(t *testing.T) {
	a, b := New(), New()
	a.Increment("x")
	a.Increment("x")
	b.Increment("y")
	m1 := a.Clone()
	m1.Merge(b)
	m2 := b.Clone()
	m2.Merge(a)
	if m1.Compare(m2) != Equal {
		t.Fatal("merge not commutative")
	}
	m3 := m1.Clone()
	m3.Merge(m1)
	if m3.Compare(m1) != Equal {
		t.Fatal("merge not idempotent")
	}
}

func TestResolveDominant(t *testing.T) {
	c1 := New()
	c1.Increment("phone")
	c2 := c1.Clone()
	c2.Increment("phone")
	l := ProgressRecord{BookID: "b1", DeviceID: "phone", Pos: Position{1, 2, 3}, Clock: c1}
	r := ProgressRecord{BookID: "b1", DeviceID: "phone", Pos: Position{5, 0, 0}, Clock: c2}
	w, err := ResolveProgress(l, r)
	if err != nil || w.Pos.Chapter != 5 {
		t.Fatalf("dominant remote must win: %+v %v", w, err)
	}
	if w.Clock.Compare(c2) != Equal {
		t.Fatal("merged clock must equal union")
	}
}

func TestResolveConcurrentFurthestWins(t *testing.T) {
	c1, c2 := New(), New()
	c1.Increment("phone")
	c2.Increment("tablet")
	l := ProgressRecord{BookID: "b1", DeviceID: "phone", Pos: Position{9, 0, 0}, Clock: c1}
	r := ProgressRecord{BookID: "b1", DeviceID: "tablet", Pos: Position{3, 0, 0}, Clock: c2}
	w, _ := ResolveProgress(l, r)
	if w.Pos.Chapter != 9 {
		t.Fatal("furthest position must win under concurrency")
	}
	// and symmetric
	w2, _ := ResolveProgress(r, l)
	if w2.Pos.Chapter != 9 {
		t.Fatal("resolution must be order-independent")
	}
}

func TestResolveConcurrentTieDeterministic(t *testing.T) {
	c1, c2 := New(), New()
	c1.Increment("aaa")
	c2.Increment("zzz")
	p := Position{1, 1, 1}
	l := ProgressRecord{BookID: "b1", DeviceID: "aaa", Pos: p, Clock: c1}
	r := ProgressRecord{BookID: "b1", DeviceID: "zzz", Pos: p, Clock: c2}
	w1, _ := ResolveProgress(l, r)
	w2, _ := ResolveProgress(r, l)
	if w1.DeviceID != "zzz" || w2.DeviceID != "zzz" {
		t.Fatal("tie-break must deterministically pick larger device id")
	}
}

func TestResolveEqualClocksFurthestAndCommutative(t *testing.T) {
	clock := New()
	clock.Increment("phone")
	l := ProgressRecord{BookID: "b1", DeviceID: "phone", Pos: Position{Chapter: 2}, Clock: clock.Clone()}
	r := ProgressRecord{BookID: "b1", DeviceID: "phone", Pos: Position{Chapter: 8}, Clock: clock.Clone()}
	forward, err := ResolveProgress(l, r)
	if err != nil || forward.Pos.Chapter != 8 {
		t.Fatalf("furthest equal-clock position must win: %+v %v", forward, err)
	}
	reverse, err := ResolveProgress(r, l)
	if err != nil || reverse.Pos.Chapter != 8 || reverse.DeviceID != forward.DeviceID {
		t.Fatalf("equal-clock resolution must be commutative: %+v %v", reverse, err)
	}
}

func TestResolveDifferentBooks(t *testing.T) {
	_, err := ResolveProgress(
		ProgressRecord{BookID: "a", Clock: New()},
		ProgressRecord{BookID: "b", Clock: New()},
	)
	if err == nil {
		t.Fatal("must reject different books")
	}
}
