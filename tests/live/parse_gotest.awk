# parse_gotest.awk - HRD-050: turn `go test -v` text logs into "KIND<TAB>package<TAB>name" records.
# KIND: PASS SKIP FAIL (per test, from `--- X: Name`), BUILDFAIL / PKGFAIL / PANIC / TIMEOUT (per package).
# Package attribution: test lines are buffered until the package terminator line `ok|FAIL <pkg> ...`;
# `# PKG <pkg>` marker lines (written by run_gated_tests.sh before in-container/worker binaries, which
# print no package line) flush the buffer and set the default package for leftovers; `# PKG-END` (written after every such run)
# flushes the buffer to that package and resets it, so a later go-test package line cannot claim the records.
# ANSI escapes and CR are stripped first.
function flush(p, i) { for (i = 1; i <= nb; i++) print bk[i] "\t" p "\t" bn[i]; nb = 0 }
BEGIN { def = "(unlabeled)"; nb = 0 }
{ gsub(/\033\[[0-9;]*[A-Za-z]/, ""); sub(/\r$/, "") }
/^# PKG-END/ { if (nb > 0 && !closed) print "INCOMPLETE\t" def "\t-"; flush(def); def = "(unlabeled)"; closed = 1; next }
/^# PKG / { flush(def); def = $3; closed = 1; next }
/^(PASS|FAIL)[ \t]*\r?$/ { closed = 1; next }
/^[ \t]*--- (PASS|SKIP|FAIL): / {
  k = $2; sub(/:$/, "", k); n = $3; nb++; closed = 0; bk[nb] = k; bn[nb] = n; seen++; next }
/^(ok|FAIL)[ \t]+[^ \t]+/ {
  p = $2
  if ($0 ~ /\[build failed\]|\[setup failed\]/) print "BUILDFAIL\t" p "\t-"
  else if ($1 == "FAIL") { print "PKGFAIL\t" p "\t-"; if (nb == 0) print "NORESULTS\t" p "\t-" }
  flush(p); next }
/^panic: test timed out/ { print "TIMEOUT\t" def "\t-"; next }
/^panic: / { print "PANIC\t" def "\t" substr($0, 8, 80); next }
END { if (nb > 0 && !closed) print "INCOMPLETE\t" def "\t-"; flush(def); print "SEEN\t-\t" seen + 0 }
