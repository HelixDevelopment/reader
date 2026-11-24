//go:build linux

package ingest

import (
	"os"
	"syscall"
)

// fileCtime returns the inode change time in nanoseconds. Unlike mtime it
// cannot be set by userspace, so an in-place rewrite that restores mtime still
// moves it.
func fileCtime(info os.FileInfo) (int64, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int64(st.Ctim.Sec)*1_000_000_000 + int64(st.Ctim.Nsec), true
}
