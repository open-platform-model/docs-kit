package schema

#Config: {
	bundles: [#Project]: #Bundle
}

#Bundle: {
	placement: #Placement
	version: {
		from:   "tag"         // phase 1: the release version comes from the git tag
		prefix: string & !="" // "opm-v": tag "opm-v4.4.5" is version "4.4.5"
	}
	sources: [#Source, ...#Source]
}

#Source: #CueCatalog | #Markdown

#CueCatalog: {
	kind:   "cue-catalog"
	module: =~"^\\./[^/]" // the CUE module root, repo-relative: "./opm"
}

#Markdown: {
	kind: "markdown"
	dir:  =~"^[^/.][^.]*$" // repo-relative directory, copied to content/ as it is
}
