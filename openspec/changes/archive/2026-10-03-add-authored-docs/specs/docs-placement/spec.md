## ADDED Requirements

### Requirement: Authored docs pages ship in docs bundles
A `markdown` source in a docs bundle SHALL copy its `dir` (all of it, or the pages `include` selects) into `content/` as written, with no link rewriting, recording each page as `generated: false` with its `source` and `lastmod`. It MAY supply a root `_index.md` and section `_index.md` pages; their uniqueness across a site version is `pull`'s check. A page path written both by an extractor and by the `markdown` source SHALL fail the build naming both, unless the extractor's page is completable.

#### Scenario: core adds its authored pages
- **WHEN** core's bundle has its `cue-definitions` source and a `markdown` source over `docs/site`, which no longer holds `reference/definitions/`
- **THEN** the bundle holds the generated definitions pages and every authored page of `docs/site`

#### Scenario: A committed generated page left behind
- **WHEN** core's `docs/site/reference/definitions/components.md` still exists
- **THEN** `build` exits 2 naming `content/reference/definitions/components.md`, the `cue-definitions` source and the `markdown` source

### Requirement: An authored page records its file on main
In a bundle with `placement.kind: "docs"`, for every page with `generated: false`, `build` SHALL write `pages[].edit`, the repository-relative path of its source file, when that file is a regular file in the `HEAD` of the main tree: the current directory when it is a git work tree of the repository built (`publish.yml` runs every mode in the caller's checkout of `main`), else the source tree; `revise` names its checkout of `main`. When the file does not exist in the main tree, `edit` SHALL be absent. A generated page, and every page of a tab or section bundle, SHALL have no `edit`. Source: DESIGN decision 19.

#### Scenario: A tab bundle's authored landing
- **WHEN** catalog_opm's tab bundle copies `docs/catalogs/opm/_index.md`
- **THEN** that page has no `edit`, so a site pinned to an older opm-docs still accepts the catalog bundle

#### Scenario: A page moved on main after the release
- **WHEN** `docs/site/start/install.md` exists at tag `v1.0.0` and `main` renamed it
- **THEN** the release bundle's page has `source` `docs/site/start/install.md` and no `edit`

#### Scenario: Edge
- **WHEN** an edge build copies `docs/site/start/install.md`
- **THEN** its `edit` is `docs/site/start/install.md`

#### Scenario: A revision adds a page
- **WHEN** a docs revision's fix adds `docs/site/start/upgrade.md`, which `main` still has
- **THEN** the revision's page has `edit` `docs/site/start/upgrade.md`
