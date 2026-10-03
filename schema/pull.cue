package schema

#Pull: {
	registry: *"ghcr.io/open-platform-model/docs" | string
	signer: {
		issuer:                                       *"https://token.actions.githubusercontent.com" | string
		workflow:                                     *"https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml" | string
		refs: *["refs/tags/v[0-9]*"] | [string, ...string] // glob over the workflow ref in the certificate
	}
	tabs: [#Project]: {
		repo: =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$" // the only repository allowed to sign this project
		root: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"
		from: =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$" // the oldest minor shown
		edge: *true | bool
	}
	// Projects that may be placed in a site version's /docs/, each with the
	// only repository allowed to sign it.
	docs: [#Project]: {
		repo: =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$"
	}
	versions: [#SiteVersion]: {
		// The bundle that chooses the others: its manifest's pins.
		anchor: {project: #Project, tag: #DocsTag}
		// Pulled at the exact version the anchor's manifest.json pins.
		pinned: *[] | [...#Project]
		// Pulled by their own tag, not by a pin.
		tags: [#Project]: #DocsTag
	}
}

// A site version: "v1.0".
#SiteVersion: =~"^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"

// A minor tag ("1.0"), a major tag ("4"), an exact release ("2.0.0-beta.1")
// or "edge".
#DocsTag: =~"^((0|[1-9][0-9]*)(\\.(0|[1-9][0-9]*))?|edge)$" | #SemVer
