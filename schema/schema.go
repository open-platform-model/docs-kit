// Package schema embeds the CUE schemas of the docs bundle contracts:
// manifest.json, docs-kit.cue, the pull config and the lock.
package schema

import "embed"

// Files holds every schema file, all in CUE package schema.
//
//go:embed *.cue
var Files embed.FS
