// Command opm-docs builds, lints, publishes and pulls OPM docs bundles.
//
// Nothing is built yet: this entrypoint only prints its version, so the
// repository's tasks run green while the build-opm-docs-phase-1 change is
// planned. See openspec/changes/build-opm-docs-phase-1/.
package main

import (
	"fmt"
	"os"

	"github.com/open-platform-model/docs-kit/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] != "version" && os.Args[1] != "--version" {
		fmt.Fprintf(os.Stderr, "opm-docs: %q is not built yet; only \"version\" exists\n", os.Args[1])
		os.Exit(1)
	}
	fmt.Println(version.String())
}
