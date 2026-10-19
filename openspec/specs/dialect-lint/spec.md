# dialect-lint Specification

## Purpose
The page-dialect lint in `opm-docs`: dialect 1, and its docs and bundle modes.

## Requirements

### Requirement: Dialect 1 is the rule set C11 lists
`opm-docs lint` SHALL enforce, for dialect 1, every rule `docs/contracts.md` C11 lists (file kinds and names, front matter, shortcodes on every line, Starlight and MDX forms, component tags, images, raw `href=`/`src=`, fence language tags, alert markers, link destinations), including the link forms `/catalogs/<name>/` and `/catalogs/<name>/<segment>/(<seg>/)*` with an optional fragment, where `<segment>` is a major (`4`), a minor (`4.4`) or `edge`.

#### Scenario: The conformance fixtures pass
- **WHEN** `opm-docs lint` runs over each case of the conformance set, `internal/dialect/testdata/conformance/`
- **THEN** it reports exactly the files and lines that case's committed expected output records

#### Scenario: A catalog link passes
- **WHEN** a page links `/catalogs/opm/4/traits/backup/`
- **THEN** the lint reports nothing for that line

### Requirement: Docs mode allows only major catalog links
In docs mode (the default), a `/catalogs/` link SHALL be the bare tab root (`/catalogs/<name>/`) or use a major segment; a minor or `edge` segment SHALL be a violation naming the major form. A link whose second segment is a minor or `edge` SHALL get that message even when it also lacks its trailing slash, as opmodel.dev's retired shell lint reported it; any other malformed `/catalogs/` link, the slashless bare root included, SHALL get the trailing-slash message.

#### Scenario: A docs page pins a minor
- **WHEN** a `docs/site` page links `/catalogs/opm/4.4/traits/backup/`
- **THEN** the lint reports the line, saying docs pages link catalogs through `/catalogs/opm/4/`

#### Scenario: The bare tab root
- **WHEN** a `docs/site` page links `/catalogs/opm/`
- **THEN** the lint reports nothing for that line

#### Scenario: A slashless minor or edge link
- **WHEN** a `docs/site` page links `/catalogs/opm/4.4/traits/backup` or `/catalogs/opm/edge`
- **THEN** the lint reports the line, saying docs pages link catalogs through `/catalogs/opm/4/` or `/catalogs/opm/<MAJOR>/`

#### Scenario: A slashless bare root
- **WHEN** a `docs/site` page links `/catalogs/opm`
- **THEN** the lint reports the line, saying to write the link with a trailing slash

### Requirement: Bundle mode checks links into the bundle
With `--bundle`, the lint SHALL read the bundle's `manifest.json` and SHALL report a link into the bundle's own root that uses a segment other than the bundle's own, or that names no page of the bundle, and a mismatch between `pages` and the files under `content/`. `build` and `pull` SHALL run the lint in bundle mode.

#### Scenario: A link to a missing member
- **WHEN** a 4.4 bundle page links `/catalogs/opm/4.4/traits/nope/`
- **THEN** the lint reports the line and the missing page

### Requirement: The enhancements graph is a link target
Dialect 1 SHALL accept the link destination `/enhancements/graph/` with an optional fragment, and the conformance fixture set SHALL hold a case for it.

#### Scenario: A docs page links the graph
- **WHEN** a page links `/enhancements/graph/`
- **THEN** the lint reports nothing for that line

### Requirement: The conformance fixture set tests the Go lint
docs-kit SHALL ship a conformance fixture set, `internal/dialect/testdata/conformance/`, each case with its expected `<file>:<line>` output, and `task test` SHALL fail when `opm-docs lint` disagrees with any expected output. A dialect rule change SHALL land with a case, or a changed expected output, that records it. The set binds no other repository: no linter outside docs-kit is held to it.

#### Scenario: A rule change the fixtures do not record
- **WHEN** a change makes the Go lint refuse a form the expected output of a conformance case accepts
- **THEN** `task test` fails naming the case and the line

### Requirement: C11 is the only definition of the dialect 1 rules
The rules of dialect 1 were first ported from opmodel.dev's `site/scripts/lint-sources.sh` as of 2026-10-02, since retired; C11 SHALL be their only definition.

#### Scenario: The only definition
- **WHEN** a dialect 1 rule is looked up
- **THEN** `docs/contracts.md` C11 is its only definition

### Requirement: A dialect 1 violation prints as a line and exits 2
Each violation of a rule that `opm-docs lint` enforces for dialect 1 SHALL print as `<file>:<line>: <message>` and the command SHALL exit 2 when any is found.

#### Scenario: A violation is printed
- **WHEN** `opm-docs lint` finds a dialect 1 violation
- **THEN** it prints `<file>:<line>: <message>` and exits 2

### Requirement: Bundle-mode lint runs the markup check on every listed page
Bundle-mode lint SHALL run the markup check (the one a section page gets, `enhancements-bundle`) on every page `manifest.json` lists, in every bundle kind, so `build`, `lint --bundle` and `pull` apply it, `pull` to bundles already published. Every violation SHALL print as `<file>:<line>: <message>` and the command SHALL exit 2.

#### Scenario: A published bundle with writer's notes still pulls
- **WHEN** `pull` fetches a docs bundle whose authored pages carry only well-formed HTML comments and whose generated pages carry no raw HTML
- **THEN** it unpacks the bundle and records it in the lock

### Requirement: A generated page and a section page get the full markup check
A page with `generated: true`, and every page of a section bundle, SHALL get the full markup check.

#### Scenario: Raw HTML in a generated page
- **WHEN** a bundle's `manifest.json` lists `reference/crd.md` with `generated: true` and that page holds `<details open ontoggle=alert(1)>` outside code
- **THEN** `opm-docs lint --bundle` reports the page and the line and exits 2

#### Scenario: A comment in a generated page
- **WHEN** a `generated: true` page holds `<!-- note -->`
- **THEN** the lint reports the page and the line and exits 2

### Requirement: An authored page of a tab or docs bundle gets the full markup check except for well-formed comments
A page with `generated: false` in a tab or docs bundle SHALL get the full markup check except that a raw HTML node (inline or a block) made only of HTML comments separated by whitespace SHALL pass when each comment opens `<!--`, is not closed abruptly (`<!-->`, `<!--->`) and ends, inside the node, at its first `-->` or `--!>`.

#### Scenario: A writer's note in an authored page
- **WHEN** a `generated: false` page of a docs bundle holds a line `<!-- Check against: core/src/x.cue -->` and an inline `| a | b <!-- c --> |`
- **THEN** the lint reports nothing for either line

#### Scenario: A comment a browser closes early
- **WHEN** a `generated: false` page holds `<!-- a --!> <svg onload=alert(1)> -->` or `<!--> <svg onload=alert(1)> -->`
- **THEN** the lint reports the page and the line and exits 2

#### Scenario: Markup after a comment
- **WHEN** a `generated: false` page holds `<!-- note --> <b>x</b>` on one line
- **THEN** the lint reports the page and the line and exits 2

#### Scenario: A heading attribute block in an authored page
- **WHEN** a `generated: false` page holds `## Install {.hx:fixed}`
- **THEN** the lint reports the page and the line and exits 2

### Requirement: Docs mode does not run the markup check
Docs mode SHALL not run the markup check, so the conformance fixture set is unchanged.

#### Scenario: A docs-mode tree is not checked
- **WHEN** `opm-docs lint docs/site` runs over a tree whose page holds `<!-- note -->`
- **THEN** the lint reports nothing for that line
