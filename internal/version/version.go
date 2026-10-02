// Package version holds the opm-docs build identity, set at link time.
package version

// Version is the opm-docs release, without a leading "v". The release build
// sets it with -ldflags "-X github.com/open-platform-model/docs-kit/internal/version.Version=<v>".
var Version = "0.0.0-dev"

// String returns the version line opm-docs prints.
func String() string {
	return "opm-docs " + Version
}
