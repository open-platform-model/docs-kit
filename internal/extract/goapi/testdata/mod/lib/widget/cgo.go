package widget

import "C"

// CgoOnly is in a file that imports "C", which a build without cgo leaves
// out.
func CgoOnly() {}
