package schema

import (
	"list"
	"time"
)

#Manifest: {
	schema:   "docs.opmodel.dev/bundle/v1"
	project:  #Project
	version:  #Version
	revision: int & >=0
	if version == "edge" {revision: 0}
	source: {
		repo:   =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$" // "open-platform-model/catalog_opm"
		commit: #SHA                                  // the commit built (a revision: the release commit)
		ref:    string & !=""                         // the release tag "opm-v4.4.5", or the branch built ("main" for edge)
		dirty?: true                                  // built from a work tree with uncommitted changes; push refuses it
		// A docs revision only: the fix commits applied to the release tree, oldest first,
		// every earlier revision's fixes included. revise writes it; a pull of 0.1.0, which
		// never wrote it, already accepts it.
		patches?: [#SHA, ...#SHA]
	}
	// The source commit's committer time, RFC 3339 UTC: the tar entries' time and the
	// org.opencontainers.image.created annotation. build always writes it and push
	// requires it; pull accepts a bundle without it.
	created?:  time.Time
	tool:      #SemVer // the opm-docs version, without "v"
	dialect:   int & >=1
	placement: #Placement
	pages: list.MinItems(1) & [...#Page]
	data: [...#DataFile]
	// The exact versions of other projects this build documents against
	// (DESIGN decision 10), from the config's pins command.
	pins?: [#Project]: #SemVer
}

#Project: =~"^[a-z0-9]+(-[a-z0-9]+)*$"
#SHA:     =~"^[0-9a-f]{40}$"
#SemVer:  =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
#Version: #SemVer | "edge"

#Placement: {
	// "tab": its own section with its own versions, /catalogs/<name>/<MAJOR.MINOR>/.
	// "docs": merged into a site version's /docs/ tree (refused by a pull that predates it).
	kind: "tab" | "docs"
	if kind == "tab" {root: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"}
	if kind == "docs" {
		root: "/docs/"
		// Paths under content/ this bundle owns exclusively: a directory
		// ending "/" or a page ending ".md". Every generated page of the
		// bundle lies under one; two of them never nest.
		owns: *[] | [...#Owned]
	}
}

#Owned: =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$|^([a-z0-9]+(-[a-z0-9]+)*/)*[a-z0-9]+(-[a-z0-9]+)*\\.md$"

#Page: {
	path:      =~"^([a-z0-9]+(-[a-z0-9]+)*/)*(_index|[a-z0-9]+(-[a-z0-9]+)*)\\.md$" // under content/
	source?:   string & !=""                                                        // repo-relative file the page came from ("Edit this page", "View source")
	lastmod?:  time.Time                                                            // that file's last commit date at the commit built, RFC 3339
	generated: bool                                                                 // generated reference, or an authored page
}

#DataFile: {
	path:   =~"^[a-z0-9-]+\\.json$" // under data/
	schema: string & !=""           // the file's own schema id, e.g. "docs.opmodel.dev/data/cue-catalog/v1"
}
