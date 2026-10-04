## REMOVED Requirements

### Requirement: Dialect 1 is the site's shell lint plus catalog links
**Reason**: opmodel.dev retired `site/scripts/lint-sources.sh`; a deleted script cannot define the dialect.
**Migration**: replaced by "Dialect 1 is the rule set C11 lists", which enforces the same rules with the same messages; its scenario "The shell lint's fixtures agree" becomes "The conformance fixtures pass".

### Requirement: The conformance fixture set binds both linters until phase 3
**Reason**: phase 3 retired the site's shell lint and its fixture copy `site/tests/lint/`, so there is no second linter to bind.
**Migration**: the fixture set stays as tests of the Go lint under "The conformance fixture set tests the Go lint"; opmodel.dev copies nothing.

## ADDED Requirements

### Requirement: Dialect 1 is the rule set C11 lists
`opm-docs lint` SHALL enforce, for dialect 1, every rule `docs/contracts.md` C11 lists (file kinds and names, front matter, shortcodes on every line, Starlight and MDX forms, component tags, images, raw `href=`/`src=`, fence language tags, alert markers, link destinations), including the link forms `/catalogs/<name>/` and `/catalogs/<name>/<segment>/(<seg>/)*` with an optional fragment, where `<segment>` is a major (`4`), a minor (`4.4`) or `edge`. The rules were first ported from opmodel.dev's `site/scripts/lint-sources.sh` as of 2026-10-02, since retired; C11 is their only definition. Each violation SHALL print as `<file>:<line>: <message>` and the command SHALL exit 2 when any is found.

#### Scenario: The conformance fixtures pass
- **WHEN** `opm-docs lint` runs over each case of the conformance set, `internal/dialect/testdata/conformance/`
- **THEN** it reports exactly the files and lines that case's committed expected output records

#### Scenario: A catalog link passes
- **WHEN** a page links `/catalogs/opm/4/traits/backup/`
- **THEN** the lint reports nothing for that line

### Requirement: The conformance fixture set tests the Go lint
docs-kit SHALL ship a conformance fixture set, `internal/dialect/testdata/conformance/`, each case with its expected `<file>:<line>` output, and `task test` SHALL fail when `opm-docs lint` disagrees with any expected output. A dialect rule change SHALL land with a case, or a changed expected output, that records it. The set binds no other repository: no linter outside docs-kit is held to it.

#### Scenario: A rule change the fixtures do not record
- **WHEN** a change makes the Go lint refuse a form the expected output of a conformance case accepts
- **THEN** `task test` fails naming the case and the line

## MODIFIED Requirements

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

### Requirement: The enhancements graph is a link target
Dialect 1 SHALL accept the link destination `/enhancements/graph/` with an optional fragment, and the conformance fixture set SHALL hold a case for it.

#### Scenario: A docs page links the graph
- **WHEN** a page links `/enhancements/graph/`
- **THEN** the lint reports nothing for that line
