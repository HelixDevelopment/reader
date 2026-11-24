# parse_gotest.awk - HRD-050: turn `go test -v` text logs into "KIND<TAB>package<TAB>name" records.
# KIND: PASS SKIP FAIL (per test, from `--- X: Name`), BUILDFAIL / PKGFAIL / PANIC / TIMEOUT / NORESULTS /
# INCOMPLETE (per package or marker).
# Package attribution: test lines are buffered until the package terminator line `ok|FAIL <pkg> ...`.
# `# PKG <import path>` ... `# PKG-END <import path>` marker pairs (written by run_gated_tests.sh around in-container
# test binaries and B3 workers, which print no package line) attribute the records between them to that package.
# Marker discipline is checked: a `# PKG` while another is open, a `# PKG-END` with no open marker or with a different
# path, a marker still open at EOF, a package terminator line while a marker is open, and buffered records with no closing
# PASS/FAIL all yield INCOMPLETE (never silently attributed). ANSI escapes and CR are stripped first.
# KNOWN LIMITS (documented, not detectable in text mode): a column-0 `ok <pkg>` line printed by test code is
# indistinguishable from a go test package line; a test printing an unindented `--- PASS: X` can forge a record.
function flush(p, i) { for (i = 1; i <= nb; i++) print bk[i] "\t" p "\t" bn[i]; nb = 0 }
BEGIN { def = "(unlabeled)"; nb = 0; open = 0 }
{ gsub(/\033\[[0-9;]*[A-Za-z]/, ""); sub(/\r$/, "") }
/^# PKG-END/ {
  if (!open) print "INCOMPLETE\t" $3 "\tPKG-END-without-open-marker"
  else if ($3 != def) print "INCOMPLETE\t" def "\tPKG-END-path-mismatch:" $3
  if (nb > 0 && !closed) print "INCOMPLETE\t" def "\t-"
  flush(def); def = "(unlabeled)"; open = 0; closed = 1; next }
/^# PKG / {
  if (open) print "INCOMPLETE\t" def "\tnested-or-unclosed-marker"
  flush(def); def = $3; open = 1; closed = 1; next }
/^(PASS|FAIL)[ \t]*$/ { closed = 1; next }
/^[ \t]*--- (PASS|SKIP|FAIL): / {
  k = $2; sub(/:$/, "", k); n = $3; nb++; closed = 0; bk[nb] = k; bn[nb] = n; seen++; next }
/^(ok|FAIL)[ \t]+[^ \t]+/ {
  p = $2
  if (open) { print "INCOMPLETE\t" def "\tpackage-line-while-marker-open"; open = 0; def = "(unlabeled)" }
  if ($0 ~ /\[build failed\]|\[setup failed\]/) print "BUILDFAIL\t" p "\t-"
  else if ($1 == "FAIL") { print "PKGFAIL\t" p "\t-"; if (nb == 0) print "NORESULTS\t" p "\t-" }
  flush(p); next }
/^panic: test timed out/ { print "TIMEOUT\t" def "\t-"; next }
/^panic: / { print "PANIC\t" def "\t" substr($0, 8, 80); next }
END {
  if (open) print "INCOMPLETE\t" def "\tmarker-open-at-EOF"
  if (nb > 0 && !closed) print "INCOMPLETE\t" def "\t-"
  flush(def); print "SEEN\t-\t" seen + 0 }
