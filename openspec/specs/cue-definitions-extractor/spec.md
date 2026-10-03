# cue-definitions-extractor Specification

## Purpose
How the `cue-definitions` source turns a CUE package's exported definitions into a reference section: parsing without evaluation, the inclusion list that must place or exclude every definition, the doc model and the pages. The contract is `docs/contracts.md` C17.

## Requirements

### Requirement: The cue-definitions source reads a CUE package's exported definitions
A `cue-definitions` source SHALL parse every `.cue` file of its `package` directory not matched by `skip`, without evaluating it, and SHALL take every exported top-level definition with its doc comment (cleaned by the doc-comment rules and the source's citation policy), its spec block (formatted, hidden fields and maintainer comments dropped), its shape, the definitions it uses, the definitions that use it and the rules its syntax shows, writing them to `data/cue-definitions.json` with schema `docs.opmodel.dev/data/cue-definitions/v1`.

#### Scenario: Uses and used-by
- **WHEN** `#Module` references `#Component`
- **THEN** `#Module`'s entry lists `#Component` under `uses` and `#Component`'s lists `#Module` under `usedBy`

#### Scenario: Pins files skipped
- **WHEN** `skip` is `["*_pins.cue"]` and `src/core_pins.cue` declares `#X`
- **THEN** `#X` is not in the model

### Requirement: Every exported definition is placed or excluded
Each exported definition SHALL appear in exactly one configured page or in `exclude` with a reason. In a build whose config is in the source tree, an unplaced definition, a definition placed twice, or a configured name the package does not declare SHALL fail the build with exit 2 naming each. When the config came from outside the source tree (a backfill), each SHALL be a warning, an unplaced definition SHALL be left out and a page left with no definition SHALL be dropped.

#### Scenario: A new definition
- **WHEN** core adds `#Policy` and `docs-kit.cue` neither places nor excludes it
- **THEN** `opm-docs check` exits 2 naming `#Policy` and the file

#### Scenario: Backfill of an older tag
- **WHEN** a release build of `v2.0.0-beta.1` uses `main`'s config, which places `#Policy` that the tag lacks
- **THEN** the build warns naming `#Policy` and succeeds

### Requirement: Definitions render as a section index and one page per group
The renderer SHALL write `<section>_index.md` (the configured title and description, the generated "Pages" list and the "All definitions" table) and `<section><file>.md` per configured page, weighted in configuration order, each definition under a `## <name>` heading whose anchor is the name lowercased without `#`, and every cross-reference linking `/docs/<section><file>/#<anchor>`. The section SHALL be one of the bundle's owned paths. The pages SHALL match core's `tools/refgen` output for the same tree, except that no page carries a generator marker comment.

#### Scenario: Link to another group
- **WHEN** `#Component` uses `#NameType`, placed on `names-paths-and-versions`
- **THEN** the components page links `/docs/reference/definitions/names-paths-and-versions/#nametype`

#### Scenario: Parity with refgen
- **WHEN** a core commit that carries refgen's committed pages is built with core's configuration
- **THEN** every page equals refgen's committed page with the marker comments removed
