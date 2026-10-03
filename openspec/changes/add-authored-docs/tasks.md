# Tasks: add-authored-docs

Gate: `generalize-build-assembly` and `pull-docs-placement` are merged on docs-kit `main`. One PR, titled `feat: record the edit path of authored pages`.

## 1. The edit path

- [ ] 1.1 `schema/manifest.cue`: `#Page.edit?` (design.md D1). Verify: `go test ./schema/...` with and without it.
- [ ] 1.2 `internal/gitsrc`: whether a path is a regular file in a tree's `HEAD`; `internal/build`: the main tree (D1) and `edit` for authored pages. Verify: tests with a temporary repository for edge, a release with the file on main, a release whose file `main` renamed (no `edit`), a revision.
- [ ] 1.3 A trial build of catalog_opm's `docs/site` as `catalog-opm-docs` and of opm's `docs/site` as `opm` with the planned configs (D2), outside the test suite. Verify: both lint in bundle mode; record page counts and problems in design.md (fixed in those repositories, not here).
- [ ] 1.4 `task check` green, then commit `feat(build): record where each authored page is edited on main`.

## 2. Contracts and archive

- [ ] 2.1 `docs/contracts.md`: C3 (`edit`), C8 (Edit and source links: tab pages none, docs pages `edit` on main, section pages none), C15 (authored docs in docs bundles, one bundle per repository, D2's configurations as examples). `README.md`: phase 3 adoption steps. Verify: links resolve.
- [ ] 2.2 `openspec archive add-authored-docs --yes`. Verify: `task openspec:check` green.
- [ ] 2.3 `task check` green, then commit `docs(build): document authored docs bundles`.
