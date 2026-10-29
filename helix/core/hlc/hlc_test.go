package hlc

import "testing"

func TestMonotonicWithFrozenClock(t *testing.T) {
	c := NewWithClock(func() int64 { return 1000 })
	a, b := c.Now(), c.Now()
	if !a.Before(b) || a.Physical != 1000 || b.Logical != 1 {
		t.Fatalf("frozen clock must still order: %v %v", a, b)
	}
}
func TestClockSkewBackward(t *testing.T) {
	tm := int64(5000)
	c := NewWithClock(func() int64 { return tm })
	a := c.Now()
	tm = 4000 // EC-S03: wall clock jumps backwards
	b := c.Now()
	if !a.Before(b) {
		t.Fatalf("skewed clock broke ordering: %v !< %v", a, b)
	}
}
func TestUpdateMerge(t *testing.T) {
	c := NewWithClock(func() int64 { return 100 })
	local := c.Now()
	remote := Timestamp{Physical: 200, Logical: 5}
	merged := c.Update(remote)
	if !merged.After(remote) || !merged.After(local) {
		t.Fatalf("merge must dominate both: %v", merged)
	}
}
func TestParseRoundTrip(t *testing.T) {
	x := Timestamp{Physical: 12345, Logical: 7}
	y, err := Parse(x.String())
	if err != nil || y != x {
		t.Fatalf("round trip failed: %v %v", y, err)
	}
}
