//go:build !unix

package serve

import "os"

// lockOwner has no portable lock here; the owner file only records the pid.
func lockOwner(*os.File) error { return nil }

// sweepStale cannot tell a live run from a dead one here, so it removes
// nothing.
func sweepStale(string) {}
