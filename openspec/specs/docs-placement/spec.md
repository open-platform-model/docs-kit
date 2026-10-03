# docs-placement Specification

## Purpose
Building and linting a bundle placed in a site version's `/docs/` tree: the content paths it owns, the lint of its links, completable pages and the versions it pins. The contract is `docs/contracts.md` C15.

## Requirements

### Requirement: A docs bundle declares what it owns
A bundle with `placement.kind: "docs"` SHALL have root `/docs/` and MAY list `owns`, paths under `content/` (a directory ending `/` or a page ending `.md`) it owns exclusively; two owned paths of one bundle SHALL NOT nest. `build` SHALL copy the placement into `manifest.json` and SHALL fail with exit 2 when a page a renderer writes lies outside every owned path, a completed page included, naming the page and the config. A docs bundle SHALL NOT hold a `cue-catalog` source.

#### Scenario: A generated page outside owns
- **WHEN** the cli bundle owns `reference/cli/` and a renderer writes `reference/commands/opm.md`
- **THEN** `build` exits 2 naming `reference/commands/opm.md` and `placement.owns`

#### Scenario: Nested owned paths
- **WHEN** `owns` lists `reference/` and `reference/cli/`
- **THEN** `build` exits 1 naming both paths

### Requirement: Bundle-mode lint of a docs bundle
For a docs bundle, bundle-mode lint SHALL apply the docs-mode link rules and SHALL report a `/docs/` link into an owned path that names no page of the bundle. It SHALL NOT check a `/docs/` link outside the owned paths. The dialect version SHALL stay 1.

#### Scenario: A link to a missing command page
- **WHEN** a cli bundle page links `/docs/reference/cli/opm-nope/`
- **THEN** the lint reports the line and the missing page

#### Scenario: A link into another bundle
- **WHEN** a cli bundle page links `/docs/reference/definitions/components/`
- **THEN** the lint reports nothing for that line

#### Scenario: A minor catalog link in a docs bundle
- **WHEN** a docs bundle page links `/catalogs/opm/4.5/traits/backup/`
- **THEN** the lint reports the line, saying docs pages link catalogs through `/catalogs/opm/4/`

### Requirement: A bundle can carry the versions it pins
When a bundle's config has `pins`, `build` SHALL run its command, SHALL require a `docs.opmodel.dev/pins/v1` document whose keys are exactly `pins.projects` and whose values are SemVer versions without `v`, and SHALL write them to `manifest.json` `pins`. A missing or extra project, or a malformed version, SHALL fail the build with exit 2 naming it. Source: DESIGN decision 10.

#### Scenario: cli pins
- **WHEN** the cli config pins `library`, `core` and `opm-operator` and the command prints all three
- **THEN** `manifest.json` holds `pins: {library: "1.0.0-beta.1", core: "2.0.0-beta.1", "opm-operator": "1.0.0-beta.4"}`

#### Scenario: A pin missing from the output
- **WHEN** the command prints no `core` pin
- **THEN** `build` exits 2 naming `core` and the command

### Requirement: Docs bundles have reserved project names
Project names SHALL follow the C1 table: a repository's docs-placed bundle is named after the repository with `_` replaced by `-`, suffixed `-docs` when that name is already a tab project of the same repository.

#### Scenario: catalog_opm's docs bundle
- **WHEN** catalog_opm configures a docs-placed bundle for its `docs/site/`
- **THEN** its project is `catalog-opm-docs`, beside the tab project `catalog-opm`

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
