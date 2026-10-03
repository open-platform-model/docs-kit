//go:build !unix

package serve

// alive cannot tell here, so no directory is swept.
func alive(int) bool { return true }
