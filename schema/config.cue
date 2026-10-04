package schema

#Config: {
	bundles: [#Project]: #Bundle
}

#Bundle: {
	placement: #Placement
	// Where a release version comes from. A section bundle builds from main
	// only (edge), so its version may be left out and is ignored when given.
	version?: {
		from:   "tag"         // phase 1: the release version comes from the git tag
		prefix: string & !="" // "opm-v": tag "opm-v4.4.5" is version "4.4.5"
	}
	if placement.kind != "section" {version: _}
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

#Source: #CueCatalog | #Markdown | #Cobra | #CueDefinitions | #CRD | #GoAPI | #Enhancements

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

// A CUE package's exported definitions, parsed (never evaluated) and grouped
// into reference pages by the author's inclusion list (docs-kit C17).
#CueDefinitions: {
	kind:    "cue-definitions"
	package: =~"^\\./" & !~"(^|/)\\.\\.(/|$)" // the package directory, repo-relative: "./src"
	skip:    *[] | [...string] // path.Match globs on file base names: ["*_pins.cue"]
	// The directory the pages are written to, one of the bundle's owned
	// paths: "reference/definitions/".
	section:     =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$"
	title:       string & !="" // the section index's front matter
	description: string & !=""
	weight?:     int & >=1 // the section index's weight among its siblings
	// The section index's opening paragraph, Markdown; without it the
	// index names the package's module path.
	intro?: string & !=""
	pages: [#DefPage, ...#DefPage]
	// Exported definitions left out of the pages, each with the reason.
	exclude: [=~"^#"]: string & !=""
	citations?: #Citations
}

// One reference page: its file under section and its definitions in order.
#DefPage: {
	file:        =~"^[a-z0-9]+(-[a-z0-9]+)*$"
	title:       string & !=""
	description: string & !=""
	definitions: [=~"^#", ...=~"^#"]
}

// A cobra command tree, printed by the repository's own program through the
// cobradump module (docs-kit C19), rendered as a command reference.
#Cobra: {
	kind:        "cobra"
	command:     #Command                         // prints the dump: ["go", "run", "./hack/docskit-dump"]
	section:     =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$" // the directory the pages go in: "reference/cli/"
	title:       string & !=""                    // the section index's title
	description: string & !=""                    // the section index's description
	weight?:     int & >=1                        // the section index's weight
	citations?:  #Citations
}

// The crd source (docs-kit C18): controller-gen CRDs and their kubebuilder
// samples, rendered as one completable reference page (page) or as a
// section of one page per kind (section).
#CRD: {
	kind:     "crd"
	dir:      =~"^\\./[^/]"  // controller-gen output: "./config/crd/bases"
	samples?: =~"^\\./[^/]"  // kubebuilder samples: "./config/samples"
	// A sample document containing any of these strings is never shown (a
	// dev or e2e fixture, not something a reader can apply).
	hideSamplesMatching: *[] | [...string & !=""]
	// Labels removed from a shown sample when they carry exactly this value
	// (kubebuilder's scaffold labels say how the repository applies it).
	stripLabels: [string & !=""]: string
	// Exactly one of page and section. page: one completable page holding
	// every kind. section: the section index and one page per kind.
	page?:    =~"^([a-z0-9]+(-[a-z0-9]+)*/)*[a-z0-9]+(-[a-z0-9]+)*\\.md$" // under content/, an owned path
	section?: =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$"                           // the directory the pages go in: "reference/operator/"
	matchN(1, [{page!: _, ...}, {section!: _, ...}])
	title:       string & !="" // the page's or the section index's front matter when no authored page completes it
	description: string & !=""
	weight?:     int & >=1 // its weight among its siblings when no authored page completes it
	order?: [string & !="", ...string & !=""] // these kinds first, in this order; the rest by name
	reconciledBy?: [string & !=""]: string & !="" // kind: the controller that reconciles it
	citations?: #Citations
}

// The go-api source (docs-kit C20): a Go module's exported packages, parsed
// (never built or type-checked) and rendered from their doc comments.
#GoAPI: {
	kind:   "go-api"
	module: #GoPath // the directory holding go.mod, repo-relative: "./"
	// Page names are package directories relative to root, which is
	// relative to module: "./opm" makes opm/helper/objectset the page
	// helper-objectset.
	root: #GoPath
	// Package patterns relative to module: "./opm/..."; "..." matches any
	// string, and a trailing "/..." also matches the directory itself.
	packages: [#GoPath, ...#GoPath]
	// The directory the pages go in, one of the bundle's owned paths:
	// "reference/go-api/".
	section:     =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$"
	title:       string & !="" // the section index's front matter
	description: string & !=""
	weight?:     int & >=1 // the section index's weight among its siblings
	citations?:  #Citations
}

// A slash path that starts "./" and holds no ".." segment.
#GoPath: =~"^\\./" & !~"(^|/)\\.\\.(/|$)"

// The enhancements repository's entries, INDEX.md and GRAPH.md, as the
// /enhancements/ section (docs-kit C21); a section bundle only. Its text is
// authored, so it takes no citations policy: citations stay as written.
#Enhancements: {
	kind:        "enhancements"
	dir:         *"." | =~"^[^/.][^.]*$" // the entry root, repo-relative
	title:       *"Enhancements" | string & !="" // the section page's title
	description: string & !=""                   // the section page's description
}
