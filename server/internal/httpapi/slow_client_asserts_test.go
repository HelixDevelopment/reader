package httpapi

// Hermetic tests of the slow-client assertion logic itself (jobsSlowAsserts):
// a golden-good evidence fixture must pass every assert, and each bad-evidence
// fixture must violate exactly the assert named for it. This proves the
// asserts can fail (§11.4.107(10)); it says nothing about the live system.

import "testing"

func goodJobsEvidence() jobsSlowClientEvidence {
	lat := func(n int) []jobsLatencyRecord {
		out := make([]jobsLatencyRecord, n)
		for i := range out {
			out[i] = jobsLatencyRecord{Client: "c", Op: "get_and_list", LatencyMS: 0.1}
		}
		return out
	}
	bp := func(maxRunning int64) map[string]any {
		return map[string]any{
			"queue_full_count": 1, "other_errors": 0, "max_running": maxRunning,
			"samples": int64(40), "nonterminal_at_rejection": docPendingBound,
		}
	}
	return jobsSlowClientEvidence{
		Arms: []jobsSlowArmEvidence{
			{Arm: "a", SlowClients: 1, FastClients: 1,
				SlowScans:        []jobsSlowScanRecord{{Client: "s0", State: "succeeded", TimeToTerminalMS: 605}},
				SlowTrickleReads: lat(5), FastReads: lat(5), FastReadsWhileFull: lat(5), Backpressure: bp(1)},
			{Arm: "b", SlowClients: 3, FastClients: 3,
				SlowScans: []jobsSlowScanRecord{
					{Client: "s0", State: "succeeded", TimeToTerminalMS: 605},
					{Client: "s1", State: "succeeded", TimeToTerminalMS: 606},
					{Client: "s2", State: "succeeded", TimeToTerminalMS: 1206}},
				SlowTrickleReads: lat(5), FastReads: lat(5), FastReadsWhileFull: lat(5), Backpressure: bp(2)},
		},
		SlowWithDeadline:    jobsSlowDeadlineRecord{State: "canceled", ElapsedMS: 505, BudgetMS: 500},
		Shutdown:            jobsShutdownRecord{ElapsedMS: 3, BudgetMS: 5000},
		CreateAfterShutdown: jobsCreateAfterShutdown{Rejected: true, ErrClass: "scan_store_closed"},
	}
}

func violatedNames(ev jobsSlowClientEvidence) map[string]bool {
	out := map[string]bool{}
	for _, a := range jobsSlowAsserts(ev) {
		if a.Violated {
			out[a.Name] = true
		}
	}
	return out
}

func TestJobsSlowAssertsGoldenGoodPasses(t *testing.T) {
	if v := violatedNames(goodJobsEvidence()); len(v) != 0 {
		t.Fatalf("golden-good evidence violates %v", v)
	}
}

func TestJobsSlowAssertsDetectBadEvidence(t *testing.T) {
	cases := []struct {
		name string
		want string
		mut  func(*jobsSlowClientEvidence)
	}{
		{"deadline_ignored_runs_to_scan_end", "slow_scan_with_deadline_timeout_honored", func(e *jobsSlowClientEvidence) { e.SlowWithDeadline.ElapsedMS = 2500 }},
		{"deadline_state_not_canceled", "slow_scan_with_deadline_timeout_honored", func(e *jobsSlowClientEvidence) { e.SlowWithDeadline.State = "succeeded" }},
		{"no_queue_full", "a_documented_backpressure_holds", func(e *jobsSlowClientEvidence) { e.Arms[0].Backpressure["queue_full_count"] = 0 }},
		{"queue_full_at_retention_cap_not_pending_bound", "a_documented_backpressure_holds", func(e *jobsSlowClientEvidence) { e.Arms[0].Backpressure["nonterminal_at_rejection"] = 200 }},
		{"pending_bound_too_low", "b_documented_backpressure_holds", func(e *jobsSlowClientEvidence) { e.Arms[1].Backpressure["nonterminal_at_rejection"] = 10 }},
		{"running_zero_vacuous_arm_a", "a_documented_backpressure_holds", func(e *jobsSlowClientEvidence) { e.Arms[0].Backpressure["max_running"] = int64(0) }},
		{"sampler_blind_no_samples", "a_documented_backpressure_holds", func(e *jobsSlowClientEvidence) { e.Arms[0].Backpressure["samples"] = int64(0) }},
		{"running_exceeds_bound", "b_documented_backpressure_holds", func(e *jobsSlowClientEvidence) { e.Arms[1].Backpressure["max_running"] = int64(3) }},
		{"bound_never_reached_arm_b", "b_documented_backpressure_holds", func(e *jobsSlowClientEvidence) { e.Arms[1].Backpressure["max_running"] = int64(1) }},
		{"no_serialization_arm_b", "b_documented_backpressure_holds", func(e *jobsSlowClientEvidence) { e.Arms[1].SlowScans[2].TimeToTerminalMS = 610 }},
		{"slow_scan_failed", "a_slow_scans_terminal_within_budget", func(e *jobsSlowClientEvidence) { e.Arms[0].SlowScans[0].State = "failed" }},
		{"fast_read_slow", "a_fast_read_latency_within_budget", func(e *jobsSlowClientEvidence) { e.Arms[0].FastReads[0].LatencyMS = 900 }},
		{"fast_read_error", "a_fast_read_latency_within_budget", func(e *jobsSlowClientEvidence) { e.Arms[0].FastReads[0].ErrClass = "other" }},
		{"slow_trickle_unbounded", "b_slow_trickle_reads_bounded", func(e *jobsSlowClientEvidence) { e.Arms[1].SlowTrickleReads[0].LatencyMS = 900 }},
		{"reads_while_full_slow", "b_fast_reads_unaffected_while_queue_full", func(e *jobsSlowClientEvidence) { e.Arms[1].FastReadsWhileFull[0].LatencyMS = 900 }},
		{"shutdown_error", "shutdown_bounded", func(e *jobsSlowClientEvidence) { e.Shutdown.ErrClass = "context_deadline_exceeded" }},
		{"shutdown_unbounded", "shutdown_bounded", func(e *jobsSlowClientEvidence) { e.Shutdown.ElapsedMS = 9000 }},
		{"create_after_shutdown_accepted", "create_after_shutdown_rejected", func(e *jobsSlowClientEvidence) { e.CreateAfterShutdown = jobsCreateAfterShutdown{} }},
	}
	for _, c := range cases {
		ev := goodJobsEvidence()
		c.mut(&ev)
		if v := violatedNames(ev); !v[c.want] {
			t.Errorf("%s: want assert %s violated, got %v", c.name, c.want, v)
		}
	}
}
