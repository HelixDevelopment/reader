HRD-067 review round 2. mutate.sh: production mutants of jobs.go (P1 drop pending bound, P2 drop semaphore, P3 scan ignores
deadline, P6 pending bound 200), all KILLED by TestSlowClientJobSurfaceHermetic (mutation.log). GREEN_hermetic.log: hermetic
tests (real in-process jobStore with no Redis journal; assertion-logic fixtures for jobs and catalog). No live legs run.
