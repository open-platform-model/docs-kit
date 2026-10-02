## Purpose

How `opm-docs` renders the doc model into pages in the page dialect: paths, links, marks and the member, index and landing pages.

## ADDED Requirements

### Requirement: Pages are written at fixed paths
The renderer SHALL write, for a `cue-catalog` source: a landing `_index.md` (unless a `markdown` source supplies a root `_index.md`), `blueprints/_index.md`, `resources/_index.md` and `traits/_index.md` with weights 1, 2 and 3, `<kind>/<name>.md` for the newest `apiVersion` of each name and kind, and `<kind>/<name>-<apiVersion>.md` for each older one. Any other path written by two sources SHALL fail the build naming the path and both sources.

#### Scenario: Two apiVersions of one trait
- **WHEN** a catalog holds `backup@v1alpha1` and `backup@v1beta1`
- **THEN** `traits/backup.md` is the v1beta1 page and `traits/backup-v1alpha1.md` the v1alpha1 page

#### Scenario: Authored landing wins
- **WHEN** a `markdown` source holds `_index.md`
- **THEN** the bundle's `content/_index.md` is that file, marked `generated: false` in `manifest.json`

### Requirement: Links use the bundle's own segment and the aliases
Every link the renderer writes into its own bundle SHALL be `<root><segment>/<page>/`, where the segment is the build version's `MAJOR.MINOR` or `edge`, and SHALL resolve to a page of the bundle. A link to another catalog SHALL use that catalog's major alias. The contract link of a member page SHALL point at the bundle's landing.

#### Scenario: Release bundle link
- **WHEN** the catalog-opm 4.4.5 build renders the `backup` page, which applies to `volumes`
- **THEN** the page links `/catalogs/opm/4.4/resources/volumes/` and the API-version row links `/catalogs/opm/4.4/`

#### Scenario: Edge bundle link
- **WHEN** an edge build renders the same page
- **THEN** it links `/catalogs/opm/edge/resources/volumes/`

### Requirement: Member pages keep refgen's content and order
A member page SHALL carry front matter `title`, `description` (the member's `metadata.description`) and `type: reference`, then the sections At a glance, Spec, Notes, Served by and Enforcement in that order, with the rows, alert strings and sentences `design.md` "Page renderer" fixes. An edge page's Catalog row SHALL name `main` and the 12-hex commit and say the build is unreleased. No page SHALL carry a generator marker comment.

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
