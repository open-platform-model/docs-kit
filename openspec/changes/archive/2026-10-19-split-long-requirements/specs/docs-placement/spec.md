## MODIFIED Requirements

### Requirement: Authored docs pages ship in docs bundles
A `markdown` source in a docs bundle SHALL copy its `dir` (all of it, or what `include` and `exclude` select) into `content/` as written, with no link rewriting, recording each page as `generated: false` with its `source` and `lastmod`. It MAY supply a root `_index.md` and section `_index.md` pages; their uniqueness across a site version is `pull`'s check.

#### Scenario: core adds its authored pages
- **WHEN** core's bundle has its `cue-definitions` source and a `markdown` source over `docs/site`, which no longer holds `reference/definitions/`
- **THEN** the bundle holds the generated definitions pages and every authored page of `docs/site`

#### Scenario: A committed generated page left behind
- **WHEN** core's `docs/site/reference/definitions/components.md` still exists
- **THEN** `build` exits 2 naming `content/reference/definitions/components.md`, the `cue-definitions` source and the `markdown` source

## REMOVED Requirements

### Requirement: An authored page records its file on main
**Reason**: Split into six requirements that each state one behavior.
**Migration**: See "An authored page records its file in the main tree", "revise names its checkout of main as the main tree", "A release build with no main tree writes no edit", "An authored page missing from the main tree has no edit", "A docs revision rebuilt by revise keeps its edit values" and "A generated page and a tab or section page have no edit".

## ADDED Requirements

### Requirement: A page path written by an extractor and by a markdown source fails the build
A page path written both by an extractor and by the `markdown` source SHALL fail the build naming both, unless the extractor's page is completable.

#### Scenario: A path written twice
- **WHEN** an extractor and the `markdown` source both write a page path and the extractor's page is not completable
- **THEN** `build` fails naming both

### Requirement: An authored page records its file in the main tree
In a bundle with `placement.kind: "docs"`, for every page with `generated: false`, `build` SHALL write `pages[].edit`, the repository-relative path of its source file, when that path, relative to the repository root, is a regular file in the `HEAD` of the main tree: the current directory when it is a git work tree of the repository built (`publish.yml` runs every publishing mode in the caller's checkout of `main`), else, for an edge build, the source tree. Source: DESIGN decision 19.

#### Scenario: Edge
- **WHEN** an edge build copies `docs/site/start/install.md`
- **THEN** its `edit` is `docs/site/start/install.md`

### Requirement: revise names its checkout of main as the main tree
The main tree is the one of the requirement "An authored page records its file in the main tree". `revise` SHALL name its checkout of `main` as the main tree. Source: DESIGN decision 19.

#### Scenario: A revision adds a page
- **WHEN** a docs revision's fix adds `docs/site/start/upgrade.md`, which `main` still has
- **THEN** the revision's page has `edit` `docs/site/start/upgrade.md`

### Requirement: A release build with no main tree writes no edit
The main tree is the one of the requirement "An authored page records its file in the main tree". A release build with no such main tree SHALL write no `edit` and SHALL print a note on stderr; it SHALL NOT read the tag's tree as `main`. Source: DESIGN decision 19.

#### Scenario: A release built away from main
- **WHEN** `build --release v1.0.0 --source src` runs in a directory that is not a work tree of the repository
- **THEN** no page has `edit`, and stderr says the pages of `v1.0.0` get no edit path

### Requirement: An authored page missing from the main tree has no edit
The main tree is the one of the requirement "An authored page records its file in the main tree". In a bundle with `placement.kind: "docs"`, for every page with `generated: false`, when the file does not exist in the main tree, `edit` SHALL be absent. Source: DESIGN decision 19.

#### Scenario: A page moved on main after the release
- **WHEN** `docs/site/start/install.md` exists at tag `v1.0.0` and `main` renamed it
- **THEN** the release bundle's page has `source` `docs/site/start/install.md` and no `edit`

### Requirement: A docs revision rebuilt by revise keeps its edit values
A docs revision built again by `revise` SHALL keep the `edit` values the pushed revision recorded, so the rebuild gives the pushed bytes. Source: DESIGN decision 19.

#### Scenario: A revision rebuilt after main moved
- **WHEN** an unpromoted revision whose page recorded `edit` `docs/site/start/install.md` is built again after `main` renamed that file
- **THEN** the page keeps that `edit` and `push` finds the same digest

### Requirement: A generated page and a tab or section page have no edit
A generated page, and every page of a tab or section bundle, SHALL have no `edit`. Source: DESIGN decision 19.

#### Scenario: A tab bundle's authored landing
- **WHEN** catalog_opm's tab bundle copies `docs/catalogs/opm/_index.md`
- **THEN** that page has no `edit`, so a site pinned to an older opm-docs still accepts the catalog bundle
