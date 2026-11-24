//go:build !linux

package ingest

import "os"

// fileCtime is unavailable here: the verdict cache is disabled and every
// OpenVerified rehashes the artifact.
func fileCtime(os.FileInfo) (int64, bool) { return 0, false }
