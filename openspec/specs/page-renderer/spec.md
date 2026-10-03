# page-renderer Specification

## Purpose
How `opm-docs` renders the doc model into pages in the page dialect: paths, links, marks and the member, index and landing pages.

## Requirements

### Requirement: Pages are written at fixed paths
The renderer SHALL write, for a `cue-catalog` source: a landing `_index.md`, `blueprints/_index.md`, `resources/_index.md` and `traits/_index.md` with weights 1, 2 and 3, `<kind>/<name>.md` for the newest `apiVersion` of each name and kind, and `<kind>/<name>-<apiVersion>.md` for each older one. Any other path written by two sources SHALL fail the build naming the path and both sources.

#### Scenario: Two apiVersions of one trait
- **WHEN** a catalog holds `backup@v1alpha1` and `backup@v1beta1`
- **THEN** `traits/backup.md` is the v1beta1 page and `traits/backup-v1alpha1.md` the v1alpha1 page

#### Scenario: Authored landing gets the generated block
- **WHEN** a `markdown` source holds `_index.md`
- **THEN** the bundle's `content/_index.md` is that file followed by the generated `## Catalog members` block, recorded as `generated: false` with that file as `source`

### Requirement: Links use the bundle's own segment and the aliases
Every link the renderer writes into its own bundle SHALL be `<root><segment>/<page>/`, where the segment is the build version's `MAJOR.MINOR` or `edge`, and SHALL resolve to a page of the bundle. A link to another catalog SHALL use that catalog's major alias. The contract link of a member page SHALL point at the bundle's landing.

#### Scenario: Release bundle link
- **WHEN** the catalog-opm 4.4.5 build renders the `backup` page, which applies to `volumes`
- **THEN** the page links `/catalogs/opm/4.4/resources/volumes/` and the API-version row links `/catalogs/opm/4.4/`

#### Scenario: Edge bundle link
- **WHEN** an edge build renders the same page
- **THEN** it links `/catalogs/opm/edge/resources/volumes/`

### Requirement: Member pages keep refgen's content and order
A member page SHALL carry front matter `title`, `description` (the member's `metadata.description`) and `type: reference`, then the sections At a glance, Spec, Notes, Served by and Enforcement in that order, with the rows, alert strings and sentences `docs/contracts.md` "Page renderer" fixes. An edge page's Catalog row SHALL name `main` and the 12-hex commit and say the build is unreleased. No page SHALL carry a generator marker comment.

#### Scenario: Provided-by-platform alert
- **WHEN** a member's `mark` is `provided-by-platform` and it is a load-bearing trait
- **THEN** its At a glance section opens with the `> [!IMPORTANT]` alert titled `**Provided by your platform**` whose body ends "Without one, rendering a component that attaches it fails; with two, the kernel refuses every render on that platform."

### Requirement: Text is escaped for Hugo and Markdown
Rendered prose SHALL escape `\ < > * _ [ ] |` outside code spans and write `{{` as `{\{`; table cells SHALL escape every `|`; a rendered page that still contains `{{<` or `{{%` SHALL fail the build naming the page.

#### Scenario: A shortcode-looking doc comment
- **WHEN** a member's note contains `{{< foo >}}` outside a code span
- **THEN** the page holds `{\{\< foo \>}}` and passes the dialect lint

### Requirement: Authored pages pin own-catalog alias links to the build
The `markdown` source SHALL rewrite every link of the form `/catalogs/<name>/<MAJOR>/<path>/` in an authored page, where `<name>` is the bundle's own catalog and `<MAJOR>` is the build version's major (or any major, for an edge build), to `/catalogs/<name>/<segment>/<path>/` with the build's own segment, and the bundle-mode lint SHALL then require it to name a page of the bundle.

#### Scenario: The authored landing links a kind index
- **WHEN** `docs/catalogs/opm/_index.md` links `/catalogs/opm/4/traits/` and the build is 4.4.5
- **THEN** the bundle's `content/_index.md` links `/catalogs/opm/4.4/traits/`

#### Scenario: A link to another catalog is left alone
- **WHEN** the same page links `/catalogs/acme/1/`, another catalog
- **THEN** the link is unchanged

### Requirement: The landing always carries the generated members block
The landing SHALL end with a generated `## Catalog members` section stating the catalog's module path and version (for edge, `main`, the 12-hex commit and "unreleased") and linking each kind index, with its member count, in the build's own segment. When a `markdown` source supplies a root `_index.md`, the renderer SHALL append the block after the authored body instead of replacing it, and SHALL refuse an authored body that already holds a `## Catalog members` heading. Without an authored landing, it SHALL write a landing holding generated front matter and the block alone.

#### Scenario: Generated landing for a backfilled release
- **WHEN** a 4.4.5 build has no authored landing and 5 blueprints, 12 resources and 28 traits
- **THEN** `content/_index.md` holds the block with links `/catalogs/opm/4.4/blueprints/`, `/catalogs/opm/4.4/resources/` and `/catalogs/opm/4.4/traits/` and the counts 5, 12 and 28

#### Scenario: Heading collision
- **WHEN** the authored `_index.md` already has a `## Catalog members` heading
- **THEN** `build` exits 2 naming the file

### Requirement: Renderers are registered per data schema
Each source kind SHALL write one data file under `data/` with its own schema id, and the renderer registered for that schema SHALL render it, reading the data file as written and never the source. A `cue-catalog` bundle built through the registry SHALL be byte-identical to one built before it.

#### Scenario: Catalog output unchanged
- **WHEN** the catalog fixture is built before and after the registry
- **THEN** both `out/` trees, and the packed digests, are identical

### Requirement: Completable pages take an authored page first
When a renderer marks a page completable and a `markdown` source of the same bundle supplies a page at the same path, the bundle's page SHALL be the authored front matter and body, one blank line, then the generated body without its own front matter, recorded as `generated: false` with the authored file as `source`. An authored body holding the generated body's first heading SHALL fail the build with exit 2 naming the file. Without an authored page, the generated page SHALL stand alone with generated front matter.

#### Scenario: The operator's resource page
- **WHEN** the `crd` source renders `reference/operator-resources.md` and a `markdown` source over `docs/site` supplies `reference/operator-resources.md`
- **THEN** the page holds the authored front matter and intro followed by the generated resource entries, and `manifest.json` records it with `source` `docs/site/reference/operator-resources.md`
