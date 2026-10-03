// The library's planned docs-kit.cue (add-go-api-extractor design D1).
bundles: library: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/go-api/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:   "go-api"
		module: "./"
		root:   "./opm"
		packages: ["./opm/..."]
		section:     "reference/go-api/"
		title:       "Go API"
		description: "Every exported package of the OPM library, from its doc comments."
	}, {
		kind: "markdown", dir: "docs/site" // the library commits no generated pages
	}]
}
