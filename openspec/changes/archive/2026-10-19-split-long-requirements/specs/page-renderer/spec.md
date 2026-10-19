## MODIFIED Requirements

### Requirement: The landing always carries the generated members block
The landing SHALL end with a generated `## Catalog members` section stating the catalog's module path and version (for edge, `main`, the 12-hex commit and "unreleased") and linking each kind index, with its member count, in the build's own segment. When a `markdown` source supplies a root `_index.md`, the renderer SHALL append the block after the authored body instead of replacing it, and SHALL refuse an authored body that already holds a `## Catalog members` heading.

#### Scenario: Generated landing for a backfilled release
- **WHEN** a 4.4.5 build has no authored landing and 5 blueprints, 12 resources and 28 traits
- **THEN** `content/_index.md` holds the block with links `/catalogs/opm/4.4/blueprints/`, `/catalogs/opm/4.4/resources/` and `/catalogs/opm/4.4/traits/` and the counts 5, 12 and 28

#### Scenario: Heading collision
- **WHEN** the authored `_index.md` already has a `## Catalog members` heading
- **THEN** `build` exits 2 naming the file

### Requirement: Completable pages take an authored page first
When a renderer marks a page completable and a `markdown` source of the same bundle supplies a page at the same path, the bundle's page SHALL be the authored front matter and body, one blank line, then the generated body without its own front matter, recorded as `generated: false` with the authored file as `source`. An authored body holding the generated body's first heading SHALL fail the build with exit 2 naming the file.

#### Scenario: The operator's resource page
- **WHEN** the `crd` source renders `reference/operator-resources.md` and a `markdown` source over `docs/site` supplies `reference/operator-resources.md`
- **THEN** the page holds the authored front matter and intro followed by the generated resource entries, and `manifest.json` records it with `source` `docs/site/reference/operator-resources.md`

## ADDED Requirements

### Requirement: A landing without an authored landing holds the generated block alone
The block is the generated `## Catalog members` section of "The landing always carries the generated members block". Without an authored landing, the renderer SHALL write a landing holding generated front matter and the block alone.

#### Scenario: No authored landing
- **WHEN** a build has no authored landing
- **THEN** the renderer writes a landing holding generated front matter and the block alone

### Requirement: A completable page without an authored page stands alone
When a renderer marks a page completable, without an authored page, the generated page SHALL stand alone with generated front matter.

#### Scenario: No authored page
- **WHEN** a renderer marks a page completable and no `markdown` source of the same bundle supplies a page at the same path
- **THEN** the generated page stands alone with generated front matter
