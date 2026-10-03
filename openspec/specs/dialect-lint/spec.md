# dialect-lint Specification

## Purpose
The page-dialect lint in `opm-docs`: dialect 1, and its docs and bundle modes.

## Requirements

### Requirement: Dialect 1 is the site's shell lint plus catalog links
`opm-docs lint` SHALL enforce, for dialect 1, every rule of `opmodel.dev/site/scripts/lint-sources.sh` as of 2026-10-02 (file kinds and names, front matter, shortcodes on every line, Starlight and MDX forms, component tags, images, raw `href=`/`src=`, fence language tags, alert markers, link destinations), and SHALL additionally accept the link forms `/catalogs/<name>/` and `/catalogs/<name>/<segment>/(<seg>/)*` with an optional fragment, where `<segment>` is a major (`4`), a minor (`4.4`) or `edge`. Each violation SHALL print as `<file>:<line>: <message>` and the command SHALL exit 2 when any is found.

#### Scenario: The shell lint's fixtures agree
- **WHEN** `opm-docs lint` runs over each fixture of the conformance set, `internal/dialect/testdata/conformance/` (a copy of opmodel.dev `site/tests/lint/`)
- **THEN** it reports the same files and lines the shell lint reports, which the committed expected output records

#### Scenario: A catalog link passes
- **WHEN** a page links `/catalogs/opm/4/traits/backup/`
- **THEN** the lint reports nothing for that line

### Requirement: Docs mode allows only major catalog links
In docs mode (the default), a `/catalogs/` link SHALL be the bare tab root (`/catalogs/<name>/`) or use a major segment; a minor or `edge` segment SHALL be a violation naming the major form. A link whose second segment is a minor or `edge` SHALL get that message even when it also lacks its trailing slash, as opmodel.dev's shell lint reports it; any other malformed `/catalogs/` link, the slashless bare root included, SHALL get the trailing-slash message.

#### Scenario: A docs page pins a minor
- **WHEN** a `docs/site` page links `/catalogs/opm/4.4/traits/backup/`
- **THEN** the lint reports the line, saying docs pages link catalogs through `/catalogs/opm/4/`

#### Scenario: The bare tab root
- **WHEN** a `docs/site` page links `/catalogs/opm/`
- **THEN** the lint reports nothing for that line

#### Scenario: A slashless minor or edge link
- **WHEN** a `docs/site` page links `/catalogs/opm/4.4/traits/backup` or `/catalogs/opm/edge`
- **THEN** the lint reports the line, saying docs pages link catalogs through `/catalogs/opm/4/` or `/catalogs/opm/<MAJOR>/`, exactly as the site's shell lint does

#### Scenario: A slashless bare root
- **WHEN** a `docs/site` page links `/catalogs/opm`
- **THEN** the lint reports the line, saying to write the link with a trailing slash

### Requirement: The conformance fixture set binds both linters until phase 3
Until phase 3 retires the site's shell lint, docs-kit SHALL ship a conformance fixture set whose initial content is a copy of exactly the fixture directories under opmodel.dev `site/tests/lint/` (not `site/tests/dialect/`), with each fixture's expected `<file>:<line>` output and the source commit, and `task test` SHALL fail when `opm-docs lint` disagrees with any expected output. docs-kit SHALL be the source of every fixture added after the copy, the `/catalogs/` link fixtures first. A dialect rule change SHALL land in docs-kit first, with its fixture, and opmodel.dev SHALL copy that fixture and update its shell lint in the PR that bumps its pinned `opm-docs` to the release carrying the change.

#### Scenario: A rule added on one side only
- **WHEN** a change makes the Go lint refuse a form the expected output of a conformance fixture accepts
- **THEN** `task test` fails naming the fixture and the line

### Requirement: Bundle mode checks links into the bundle
With `--bundle`, the lint SHALL read the bundle's `manifest.json` and SHALL report a link into the bundle's own root that uses a segment other than the bundle's own, or that names no page of the bundle, and a mismatch between `pages` and the files under `content/`. `build` and `pull` SHALL run the lint in bundle mode.

#### Scenario: A link to a missing member
- **WHEN** a 4.4 bundle page links `/catalogs/opm/4.4/traits/nope/`
- **THEN** the lint reports the line and the missing page

### Requirement: The enhancements graph is a link target
Dialect 1 SHALL accept the link destination `/enhancements/graph/` with an optional fragment, and the conformance fixture set SHALL gain a fixture for it, which opmodel.dev copies with its shell lint change in the PR that bumps its pinned `opm-docs`.

#### Scenario: A docs page links the graph
- **WHEN** a page links `/enhancements/graph/`
- **THEN** the lint reports nothing for that line

### Requirement: Bundle mode checks every page's markup as the site parses it
Bundle-mode lint SHALL run the markup check (the one a section page gets, `enhancements-bundle`) on every page `manifest.json` lists, in every bundle kind, so `build`, `lint --bundle` and `pull` apply it, `pull` to bundles already published. A page with `generated: true`, and every page of a section bundle, SHALL get the full check. A page with `generated: false` in a tab or docs bundle SHALL get the full check except that a raw HTML node (inline or a block) made only of HTML comments separated by whitespace SHALL pass when each comment opens `<!--`, is not closed abruptly (`<!-->`, `<!--->`) and ends, inside the node, at its first `-->` or `--!>`. Every violation SHALL print as `<file>:<line>: <message>` and the command SHALL exit 2. Docs mode SHALL not run the check, so the conformance fixture set is unchanged.

#### Scenario: Raw HTML in a generated page
- **WHEN** a bundle's `manifest.json` lists `reference/crd.md` with `generated: true` and that page holds `<details open ontoggle=alert(1)>` outside code
- **THEN** `opm-docs lint --bundle` reports the page and the line and exits 2

#### Scenario: A comment in a generated page
- **WHEN** a `generated: true` page holds `<!-- note -->`
- **THEN** the lint reports the page and the line and exits 2

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

#### Scenario: A published bundle with writer's notes still pulls
- **WHEN** `pull` fetches a docs bundle whose authored pages carry only well-formed HTML comments and whose generated pages carry no raw HTML
- **THEN** it unpacks the bundle and records it in the lock

#### Scenario: A docs-mode tree is not checked
- **WHEN** `opm-docs lint docs/site` runs over a tree whose page holds `<!-- note -->`
- **THEN** the lint reports nothing for that line
