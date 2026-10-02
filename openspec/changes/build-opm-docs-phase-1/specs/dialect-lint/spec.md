## Purpose

The page-dialect lint in `opm-docs`: dialect 1, and its docs and bundle modes.

## ADDED Requirements

### Requirement: Dialect 1 is the site's shell lint plus catalog links
`opm-docs lint` SHALL enforce, for dialect 1, every rule of `opmodel.dev/site/scripts/lint-sources.sh` as of 2026-10-02 (file kinds and names, front matter, shortcodes on every line, Starlight and MDX forms, component tags, images, raw `href=`/`src=`, fence language tags, alert markers, link destinations), and SHALL additionally accept the link forms `/catalogs/<name>/` and `/catalogs/<name>/<segment>/(<seg>/)*` with an optional fragment, where `<segment>` is a major (`4`), a minor (`4.4`) or `edge`. Each violation SHALL print as `<file>:<line>: <message>` and the command SHALL exit 2 when any is found.

#### Scenario: The shell lint's fixtures agree
- **WHEN** `opm-docs lint` runs over each fixture under `opmodel.dev/site/tests/lint/` (copied into `internal/dialect/testdata/`)
- **THEN** it reports the same files and lines the shell lint reports

#### Scenario: A catalog link passes
- **WHEN** a page links `/catalogs/opm/4/traits/backup/`
- **THEN** the lint reports nothing for that line

### Requirement: Docs mode allows only major catalog links
In docs mode (the default), a `/catalogs/` link SHALL use a major segment; a minor or `edge` segment SHALL be a violation naming the major form.

#### Scenario: A docs page pins a minor
- **WHEN** a `docs/site` page links `/catalogs/opm/4.4/traits/backup/`
- **THEN** the lint reports the line, saying docs pages link catalogs through `/catalogs/opm/4/`

### Requirement: Bundle mode checks links into the bundle
With `--bundle`, the lint SHALL read the bundle's `manifest.json` and SHALL report a link into the bundle's own root that uses a segment other than the bundle's own, or that names no page of the bundle, and a mismatch between `pages` and the files under `content/`. `build` and `pull` SHALL run the lint in bundle mode.

#### Scenario: A link to a missing member
- **WHEN** a 4.4 bundle page links `/catalogs/opm/4.4/traits/nope/`
- **THEN** the lint reports the line and the missing page
