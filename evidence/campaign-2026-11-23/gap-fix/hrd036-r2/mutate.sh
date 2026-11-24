#!/bin/bash
# HRD-036 mutation check: each mutant must make the test set FAIL. Restores sources afterwards.
cd "$(dirname "$0")/../../../../server" || exit 1
export GIN_MODE=release; T=$(mktemp -d)
cp internal/sources/rclone.go "$T"/_rclone.orig || exit 1
cp internal/sources/webdav.go "$T"/_webdav.orig; cp internal/httpapi/router.go "$T"/_router.orig
run() { go test ./internal/sources ./internal/httpapi -run 'TestRemoteSourcesHonourRange|TestWebDAV|TestSingleRange|TestOpenRange|TestRclone' -count=1 >"$T"/_mut.out 2>&1; }
mut() { # name file python-expr
  python3 - "$2" "$3" "$4" <<'P'
import sys
f,a,b=sys.argv[1:4]
s=open(f).read(); assert a in s,(f,a); open(f,'w').write(s.replace(a,b,1))
P
  if run; then echo "MUTANT $1: SURVIVED (bad)"; else echo "MUTANT $1: KILLED"; fi
  cp "$T"/_rclone.orig internal/sources/rclone.go; cp "$T"/_webdav.orig internal/sources/webdav.go; cp "$T"/_router.orig internal/httpapi/router.go
}
run && echo "BASELINE: GREEN" || echo "BASELINE: RED (bad)"
mut M1_rclone_no_revalidate internal/sources/rclone.go 'func (r *Rclone) Revalidate(' 'func (r *Rclone) RevalidateOff('
mut M2_webdav_size_off_by_one internal/sources/webdav.go 'return n, mod, nil' 'return n + 1, mod, nil'
mut M3_webdav_unbounded_range internal/sources/webdav.go 'if err != nil || length < 0 {' 'if true {'
mut M4_rclone_notfound_swallowed internal/sources/rclone.go 'return -1, time.Time{}, ErrNotFound
		}
		if strings.Contains' 'return -1, time.Time{}, nil
		}
		if strings.Contains'
mut M5_webdav_collection_accepted internal/sources/webdav.go 'if pr.collection {
			return -1, time.Time{}, fmt.Errorf("source: not a regular file")' 'if false {
			return -1, time.Time{}, fmt.Errorf("source: not a regular file")'
mut M6_webdav_missing_mtime_trusted internal/sources/webdav.go 'if pr.length == "" || pr.modified == "" || err != nil {' 'if pr.length == "" {'
mut M7_list_collection_ignored internal/sources/webdav.go 'if pr.collection {
			continue' 'if false {
			continue'
mut M8_nonnumeric_length_is_zero internal/sources/webdav.go '		n, ok := parseContentLength(pr.length)
		if !ok {' '		n, _ := parseContentLength(pr.length)
		if false {'
mut M14_list_invented_size internal/sources/webdav.go '		size, ok := parseContentLength(pr.length)
		if !ok {' '		size, ok := parseContentLength(pr.length)
		if !ok && false {'
mut M15_contentlength_accepts_sign internal/sources/webdav.go "if v[i] < '0' || v[i] > '9' {
			return 0, false" "if v[i] < '+' || v[i] > '9' {
			return 0, false"
mut M9_first_propstat_only internal/sources/webdav.go 'if st := strings.Fields(p.Status); len(st) >= 2 && !strings.HasPrefix(st[1], "2") {
			continue
		}' 'if st := strings.Fields(p.Status); len(st) >= 2 && !strings.HasPrefix(st[1], "2") {
			return m
		}'
mut M10_lastbyte_overflow_416 internal/httpapi/router.go 'if !okEnd || (!overEnd && last < start) {' 'if !okEnd || overEnd || last < start {'
mut M11_firstbyte_overflow_clamped internal/httpapi/router.go 'if !ok || over || start >= size {' 'if !ok || start >= size {'
mut M12_suffix_overflow_416 internal/httpapi/router.go 'if over || n > size {' 'if n > size {'
mut M13_signed_numbers_accepted internal/httpapi/router.go 'if s[i] < '"'"'0'"'"' || s[i] > '"'"'9'"'"' {' 'if s[i] < '"'"'+'"'"' || s[i] > '"'"'9'"'"' {'
mut O7_rclone_isdir_unchecked internal/sources/rclone.go '	if row.IsDir {
		return -1, time.Time{}, fmt.Errorf("source: not a regular file")' '	if false {
		return -1, time.Time{}, fmt.Errorf("source: not a regular file")'
mut O9b_propstat_status_filter_off internal/sources/webdav.go 'len(st) >= 2 && !strings.HasPrefix(st[1], "2")' 'false'
mut O10_end_clamp_off_by_one internal/httpapi/router.go 'if overEnd || end >= size {' 'if overEnd || end > size {'
mut O11_rclone_revalidate_traversal_unchecked internal/sources/rclone.go 'Revalidate(ctx context.Context, name string) (int64, time.Time, error) {
	if strings.HasPrefix(name, "..")' 'Revalidate(ctx context.Context, name string) (int64, time.Time, error) {
	if false && strings.HasPrefix(name, "..")'
mut O12_rclone_bad_modtime_is_error internal/sources/rclone.go '	if err != nil || row.Size < 0 {
		return -1, time.Time{}, nil' '	if err != nil {
		return -1, time.Time{}, err
	}
	if row.Size < 0 {
		return -1, time.Time{}, nil'
mut M16_openrange_failure_headers_leak internal/httpapi/router.go '			c.Writer.Header().Del("Content-Range")' '			_ = 0'
