package httpapi

// Each conjunct of the soak leak gate (slope, growth, R^2 incl. the step
// relaxation, warm-up discard, goroutine slop) is pinned by a synthetic series
// that ONLY that conjunct rejects (or accepts). The detector is an AND, so the
// other fixtures cannot cover a weakened single conjunct (review of PR #16, N1).
// Each fixture states its precondition and checks it with fit(), so a future
// change of the constants cannot silently turn a fixture into a no-op.

import "testing"

func soakSeries(ys []uint64, iterations int) soakResult {
	res := soakResult{Iterations: iterations, HeapBytes: ys}
	res.Goroutines = make([]int, len(ys))
	for i := range res.Goroutines {
		res.Goroutines[i] = 10
	}
	step := iterations / 20
	analyzeSoak(&res, step)
	return res
}

func ramp(base uint64, perSample int64, n int) []uint64 {
	out := make([]uint64, n)
	for i := range out {
		out[i] = uint64(int64(base) + perSample*int64(i))
	}
	return out
}

func TestSoakGateRSquaredRejectsNoisyGrowth(t *testing.T) {
	ys := make([]uint64, 20)
	for i := range ys {
		noise := int64(-150000)
		if i%2 == 1 {
			noise = 150000
		}
		ys[i] = uint64(20_000_000 + 11000*int64(i) + noise)
	}
	steady := ys[6:]
	slope, r2 := fit(steady)
	growth := int64(steady[len(steady)-1]) - int64(steady[0])
	if slope/300 <= soakSlopeLimit || growth <= soakGrowthLimit || r2 > soakStepR2 {
		t.Fatalf("fixture precondition broken: slope/iter=%.1f growth=%d r2=%.3f", slope/300, growth, r2)
	}
	if res := soakSeries(ys, 6000); res.Leaked {
		t.Fatalf("noisy non-linear growth flagged: %v", res.Reasons)
	}
}

func TestSoakGateWarmupDiscard(t *testing.T) {
	ys := make([]uint64, 20)
	for i := range ys {
		if i < 6 {
			ys[i] = uint64(20_000_000 + 1_500_000*i)
		} else {
			ys[i] = uint64(29_000_000 + (i%2)*1000) // flat after warm-up, tiny jitter
		}
	}
	// Precondition: WITHOUT the discard the same series is a leak by the gate.
	slope, r2 := fit(ys)
	growth := int64(ys[19]) - int64(ys[0])
	if slope/300 <= soakSlopeLimit || growth <= soakStepGrowth || r2 <= soakStepR2 {
		t.Fatalf("fixture precondition broken: slope/iter=%.1f growth=%d r2=%.3f", slope/300, growth, r2)
	}
	if res := soakSeries(ys, 6000); res.Leaked {
		t.Fatalf("warm-up ramp flagged although the first 30%% is discarded: %v", res.Reasons)
	}
}

func TestSoakGateGrowthLimitRejectsSmallAbsoluteGrowth(t *testing.T) {
	// 40 B/iteration at 200 iterations: slope above the limit, perfectly linear,
	// but only ~5 KiB of growth in total.
	ys := ramp(20_000_000, 400, 20)
	steady := ys[6:]
	slope, r2 := fit(steady)
	growth := int64(steady[len(steady)-1]) - int64(steady[0])
	if slope/10 <= soakSlopeLimit || r2 <= soakMinR2 || growth >= soakGrowthLimit {
		t.Fatalf("fixture precondition broken: slope/iter=%.1f r2=%.3f growth=%d", slope/10, r2, growth)
	}
	if res := soakSeries(ys, 200); res.Leaked {
		t.Fatalf("tiny absolute growth flagged: %v", res.Reasons)
	}
}

func TestSoakGateSlopeLimitRejectsShallowLinearGrowth(t *testing.T) {
	// 16 B/iteration at 6000 iterations: linear, growth above 64 KiB (16*300*13
	// = 62 KiB is below it, so use 20 B/iter), slope below the 32 B/iter floor.
	ys := ramp(20_000_000, 6000, 20) // 20 B/iter
	steady := ys[6:]
	slope, r2 := fit(steady)
	growth := int64(steady[len(steady)-1]) - int64(steady[0])
	if slope/300 >= soakSlopeLimit || r2 <= soakMinR2 || growth <= soakGrowthLimit {
		t.Fatalf("fixture precondition broken: slope/iter=%.1f r2=%.3f growth=%d", slope/300, r2, growth)
	}
	if res := soakSeries(ys, 6000); res.Leaked {
		t.Fatalf("slope below the floor flagged: %v", res.Reasons)
	}
}

func TestSoakGatePositiveControlLinearLeakIsFlagged(t *testing.T) {
	// Same shape as the fixtures above but passing every conjunct: must flag.
	res := soakSeries(ramp(20_000_000, 12000, 20), 6000) // 40 B/iter, ~156 KiB growth
	if !res.Leaked {
		t.Fatalf("linear 40 B/iter leak not flagged: slope=%.1f r2=%.3f growth=%d", res.SlopeBytesIter, res.R2, res.SteadyGrowthB)
	}
}

func TestSoakGateGoroutineSlop(t *testing.T) {
	flat := ramp(20_000_000, 0, 20)
	mk := func(delta int) soakResult {
		res := soakResult{Iterations: 6000, HeapBytes: flat}
		res.Goroutines = make([]int, 20)
		for i := range res.Goroutines {
			res.Goroutines[i] = 10
		}
		res.Goroutines[19] = 10 + delta
		analyzeSoak(&res, 300)
		return res
	}
	if mk(soakGoroutineSlop).Leaked {
		t.Fatal("goroutine growth at the slop flagged")
	}
	if !mk(soakGoroutineSlop + 1).Leaked {
		t.Fatal("goroutine growth above the slop not flagged")
	}
}
