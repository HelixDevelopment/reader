package catalog

// Hermetic tests of the catalog slow-client assertion logic (catalogSlowAsserts):
// golden-good evidence passes, each bad-evidence fixture violates the named
// assert. Proves the asserts can fail; says nothing about the live database.

import "testing"

func goodCatalogEvidence() catalogSlowClientEvidence {
	slow := func() []catalogSlowClientRecord {
		return []catalogSlowClientRecord{{Client: "s0", Op: "get", ElapsedMS: float64(slowClientDeadline.Milliseconds()), ErrClass: "context_deadline_exceeded"}}
	}
	fast := func() []catalogSlowLatency { return []catalogSlowLatency{{Client: "f0", Op: "get", LatencyMS: 2}} }
	phase := func() catalogSlowPhaseEvidence {
		return catalogSlowPhaseEvidence{Slow: slow(), FastReaders: fast(), FastWriters: fast()}
	}
	return catalogSlowClientEvidence{Arms: []catalogSlowArmEvidence{{Arm: "a", ReadPhase: phase(), WritePhase: phase()}}}
}

func catalogViolated(ev catalogSlowClientEvidence) map[string]bool {
	out := map[string]bool{}
	for _, a := range catalogSlowAsserts(ev) {
		if a.Violated {
			out[a.Name] = true
		}
	}
	return out
}

func TestCatalogSlowAssertsGoldenGoodPasses(t *testing.T) {
	if v := catalogViolated(goodCatalogEvidence()); len(v) != 0 {
		t.Fatalf("golden-good violates %v", v)
	}
}

func TestCatalogSlowAssertsDetectBadEvidence(t *testing.T) {
	cases := []struct {
		name, want string
		mut        func(*catalogSlowClientEvidence)
	}{
		{"slow_read_returns_early", "a_read_slow_client_timeout_bounded", func(e *catalogSlowClientEvidence) { e.Arms[0].ReadPhase.Slow[0].ElapsedMS = 10 }},
		{"slow_read_overshoots_deadline", "a_read_slow_client_timeout_bounded", func(e *catalogSlowClientEvidence) { e.Arms[0].ReadPhase.Slow[0].ElapsedMS = 99999 }},
		{"slow_write_succeeds", "a_write_slow_client_timeout_bounded", func(e *catalogSlowClientEvidence) { e.Arms[0].WritePhase.Slow[0].ErrClass = "" }},
		{"fast_read_errors", "a_read_fast_client_error_storm_absent", func(e *catalogSlowClientEvidence) { e.Arms[0].ReadPhase.FastErrors = 3 }},
		{"fast_write_errors", "a_write_fast_client_error_storm_absent", func(e *catalogSlowClientEvidence) { e.Arms[0].WritePhase.FastErrors = 1 }},
		{"fast_reader_slow", "a_read_fast_reader_latency_within_budget", func(e *catalogSlowClientEvidence) { e.Arms[0].ReadPhase.FastReaders[0].LatencyMS = 1e6 }},
		{"fast_writer_slow", "a_write_fast_writer_latency_within_budget", func(e *catalogSlowClientEvidence) { e.Arms[0].WritePhase.FastWriters[0].LatencyMS = 1e6 }},
		{"fast_reader_err_in_write_phase", "a_write_fast_reader_latency_within_budget", func(e *catalogSlowClientEvidence) { e.Arms[0].WritePhase.FastReaders[0].ErrClass = "x" }},
	}
	for _, c := range cases {
		ev := goodCatalogEvidence()
		c.mut(&ev)
		if v := catalogViolated(ev); !v[c.want] {
			t.Errorf("%s: want %s violated, got %v", c.name, c.want, v)
		}
	}
}
