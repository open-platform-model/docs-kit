package schema

#Lock: {
	schema: "docs.opmodel.dev/lock/v1"
	tool:   #SemVer                   // the opm-docs that wrote the lock
	config: =~"^sha256:[0-9a-f]{64}$" // SHA-256 of the bundles.cue bytes
	bundles: [...#Locked]
	history?: [...{project: #Project, digest: =~"^sha256:[0-9a-f]{64}$", path: =~"^[a-z0-9]+(-[a-z0-9]+)*/history\\.json$"}] // path relative to the lock's directory
}

#Locked: #Pulled | #Local

#Pulled: {
	project:    #Project
	root:       =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"
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
	root:     =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"
	segment:  #Segment
	local:    true
	version:  #Version
	revision: int & >=0
	commit:   #SHA
	dialect:  int & >=1
	builtBy:  #SemVer
	dir:      string & !=""
}

#Segment: =~"^((0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)|edge)$"
