// Command opm-docs builds, lints, publishes and pulls OPM docs bundles.
package main

import (
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
