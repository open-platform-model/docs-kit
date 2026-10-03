package schema

#Lock: {
	schema: "docs.opmodel.dev/lock/v1"
	tool:   #SemVer                   // the opm-docs that wrote the lock
	config: =~"^sha256:[0-9a-f]{64}$" // SHA-256 of the bundles.cue bytes
	bundles: [...#Locked]
	history?: [...{project: #Project, digest: =~"^sha256:[0-9a-f]{64}$", path: =~"^[a-z0-9]+(-[a-z0-9]+)*/history\\.json$"}] // path relative to the lock's directory
	docs?: [...#DocsLocked] // the bundles of each site version; left out when the config has no versions
}

#Locked: #Pulled | #Local

#Pulled: {
	project:    #Project
	root:       #LockRoot
	segment:    #Segment
	tag:        #Segment      // the tag resolved: the segment itself
	repository: string & !="" // "ghcr.io/open-platform-model/docs/catalog-opm"
	digest:     =~"^sha256:[0-9a-f]{64}$"
	version:    #Version
	revision:   int & >=0
	commit:     #SHA
	dialect:    int & >=1
	builtBy:    #SemVer // the bundle's dev.opmodel.docs.tool
	signer: {
		workflow:   string & !="" // the certificate SAN
		repository: string & !="" // Source Repository URI
		ref:        "refs/heads/main"
	}
	dir: string & !="" // "<project>/<segment>", relative to the lock's directory
}

#Local: {
	project:  #Project
	root:     #LockRoot
	segment:  #Segment
	local:    true
	version:  #Version
	revision: int & >=0
	commit:   #SHA
	dialect:  int & >=1
	builtBy:  #SemVer
	dir:      string & !=""
}

// A tab's root, or the enhancements section's.
#LockRoot: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$|^/enhancements/$"

#Segment: =~"^((0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)|edge)$"

#DocsLocked: #DocsPulled | #DocsLocal

#DocsRole: "anchor" | "pinned" | "tag"

#DocsPulled: {
	site:       #SiteVersion
	project:    #Project
	role:       #DocsRole
	tag:        #DocsTag      // the tag resolved: "1.0", "2.0.0-beta.1", "4", "edge"
	repository: string & !="" // "ghcr.io/open-platform-model/docs/cli"
	digest:     =~"^sha256:[0-9a-f]{64}$"
	version:    #Version
	revision:   int & >=0
	commit:     #SHA
	dialect:    int & >=1
	builtBy:    #SemVer
	signer: {
		workflow:   string & !=""
		repository: string & !=""
		ref:        "refs/heads/main"
	}
	pins?: [#Project]: #SemVer // the anchor only: its manifest's pins as read
	dir: string & !="" // "_versions/v1.0/cli", relative to the lock's directory
}

#DocsLocal: {
	site:     #SiteVersion
	project:  #Project
	role:     #DocsRole
	local:    true
	version:  #Version
	revision: int & >=0
	commit:   #SHA
	dialect:  int & >=1
	builtBy:  #SemVer
	pins?: [#Project]: #SemVer
	dir: string & !=""
}
