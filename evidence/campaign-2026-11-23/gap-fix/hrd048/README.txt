HRD-048 / GAP-013: memory + goroutine soak (server/internal/httpapi/soak_test.go)

What it measures
- Fixed request mix (works list, work detail, content full, ranged, HEAD, OPDS feed) against a real listening server,
  default 6000 iterations (HELIX_SOAK_ITERS), 20 post-GC samples; first 30% discarded as warm-up.
- Heap and goroutine gates: see "Detection floor" below.
- A flagged run is repeated once on the warm system (runSoakConfirmed); only a leak that persists is reported.
- Remote variant: WebDAV connector (3000 iterations) and rclone connector (300 iterations) against a real
  `rclone serve webdav` loopback target, full and ranged bodies compared byte for byte; additionally counts
  lingering rclone helper processes (/proc, children of the test process) and requires zero.
- Concurrency probe: 50 concurrent ranged requests to an rclone source. Observed peak rclone helper processes: 7 and 29 in two runs
  (see soak_GREEN.log), 0 lingering. UNCONFIRMED whether the connector or route imposes a cap: peak < 50 may reflect
  scheduling and request duration, not a limit. No cap was found in server/internal/sources/rclone.go.

Detection floor (honest, SHAPE-QUALIFIED; detection_floor_by_shape.log, 10 confirmed runs per cell, default 6000 iterations)
- Per-iteration allocations retained in a [][]byte: flagged 10/10 down to 16 B/iteration (the measured cost includes the
  24 B slice header).
- A single slice appended to every iteration (capacity-doubling "staircase" heap growth): flagged 10/10 at 32 B/iteration
  and above, 0/10 at 16 B/iteration. Staircase growth fits a line with R^2 about 0.77-0.80, so the R^2 gate relaxes to
  0.50 when steady growth exceeds 256 KiB (RED_staircase.log: the previous detector missed 100 B/iteration).
- Gates: slope > 32 B/iteration AND growth > 64 KiB AND (R^2 > 0.80 OR (R^2 > 0.50 AND growth > 256 KiB)).
  Goroutine leak: floor grows by more than 3 (samples wait up to 300 ms for short-lived goroutines to finish);
  1 goroutine per 200 iterations flagged; rarer than about 1 per 1000 is NOT detected.
- False positives: bounded churning workload 5 runs; and GC-heavy bursty, random-evict cache, cache filling over two
  thirds of the run, goroutine bursts, sawtooth release: 10 confirmed runs each, 0 flagged (GREEN_shapes.log).
- Real-server mutants (mutation.log): 256 B/iteration heap leak KILLED; 1 goroutine per 200 content requests KILLED;
  missing cmd.Wait() in the rclone connector (zombie helpers) KILLED by the concurrency probe.
- The rclone helper counter now counts zombies (state Z, command name rclone) as well as live helpers; its self-test
  (TestRcloneWorkersCounterSeesZombiesAndLiveHelpers) spawns a never-waited child. RED_zombie_counter.log.
- NOT detected: leaks below the floors above; a cache still filling during the whole steady window (the second pass
  reduces but cannot eliminate that class); leaks that only appear under conditions this request mix never creates.
- Rclone soak variant: 300 iterations, so its heap sensitivity is about 300 B/iteration, coarser than the main soak.
  Concurrency probe peak is not bounded by anything found in the code (peak 7 and 29 of 50 in two runs; UNCONFIRMED cap).

Scope limits: in-process, single host; no 24h run; no pprof heap diff; no web heap snapshot; no Android dumpsys meminfo.
