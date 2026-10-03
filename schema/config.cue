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
	// The exact versions of other projects this build documents against
	// (DESIGN decision 10). The command prints a pins document
	// ({"schema": "docs.opmodel.dev/pins/v1", "pins": {<project>: <version>}})
	// whose keys are exactly projects; build writes them to manifest.json pins.
	pins?: {
		command: #Command
		projects: [#Project, ...#Project]
	}
}

// A repository command (docs-kit C14): argv[0] is looked up on PATH; no
// shell, no globbing. It runs in the source tree and prints one JSON document.
#Command: [string & !="", ...string]

// What an extractor source's citations become: "strip" removes them;
// "link" links each enhancement decision citation to its decisions page.
#Citations: *"strip" | "link"

#Source: #CueCatalog | #Markdown

#CueCatalog: {
	kind:   "cue-catalog"
	module: =~"^\\./[^/]" // the CUE module root, repo-relative: "./opm"
	// The catalog doc model holds plain prose its renderer escapes, so a
	// catalog strips citations; "link" needs a doc model that carries links.
	citations: "strip"
}

#Markdown: {
	kind: "markdown"
	dir:  =~"^[^/.][^.]*$" // repo-relative directory, copied to content/ as it is
	// Patterns relative to dir, matched against each file's slash path: a
	// path.Match glob ("**" is not special), or a directory ending "/",
	// which matches every file under it. A file is copied when it matches
	// some include (or include is absent) and no exclude.
	include?: [string, ...string]
	exclude?: [string, ...string]
}
