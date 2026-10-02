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
	// Phase 2 adds `versions:` for bundles placed in a site version's /docs/.
}
