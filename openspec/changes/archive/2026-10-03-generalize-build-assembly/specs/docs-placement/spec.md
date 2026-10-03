## ADDED Requirements

### Requirement: A docs bundle declares what it owns
A bundle with `placement.kind: "docs"` SHALL have root `/docs/` and MAY list `owns`, paths under `content/` (a directory ending `/` or a page ending `.md`) it owns exclusively; two owned paths of one bundle SHALL NOT nest. `build` SHALL copy the placement into `manifest.json` and SHALL fail with exit 2 when a generated page lies outside every owned path, naming the page and the config. A docs bundle SHALL NOT hold a `cue-catalog` source.

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
