## MODIFIED Requirements

### Requirement: The cue-definitions source reads a CUE package's exported definitions
A `cue-definitions` source SHALL parse every `.cue` file of its `package` directory not matched by `skip`, without evaluating it, and SHALL take every exported top-level definition with its doc comment (cleaned by the doc-comment rules and the source's citation policy), its spec block (formatted, hidden fields and maintainer comments dropped), its shape, the definitions it uses, the definitions that use it and the rules its syntax shows.

#### Scenario: Uses and used-by
- **WHEN** `#Module` references `#Component`
- **THEN** `#Module`'s entry lists `#Component` under `uses` and `#Component`'s lists `#Module` under `usedBy`

#### Scenario: Pins files skipped
- **WHEN** `skip` is `["*_pins.cue"]` and `src/core_pins.cue` declares `#X`
- **THEN** `#X` is not in the model

## REMOVED Requirements

### Requirement: Every exported definition is placed or excluded
**Reason**: Split into four requirements that each state one behavior.
**Migration**: See "Each exported definition is placed once or excluded with a reason", "A config in the source tree must place or exclude every definition", "A config in the source tree must give each definition a summary and an anchor", "A config from outside the source tree gives warnings".

### Requirement: Definitions render as a section index and one page per group
**Reason**: Split into three requirements that each state one behavior.
**Migration**: See "The renderer writes a section index and one page per group", "The definitions section lies under an owned path", "The definition pages match core's refgen output".

## ADDED Requirements

### Requirement: The cue-definitions source writes its model file
The `cue-definitions` source SHALL write the definitions it takes to `data/cue-definitions.json` with schema `docs.opmodel.dev/data/cue-definitions/v1`.

#### Scenario: The model file
- **WHEN** a `cue-definitions` source runs
- **THEN** it writes the definitions to `data/cue-definitions.json` with schema `docs.opmodel.dev/data/cue-definitions/v1`

### Requirement: Each exported definition is placed once or excluded with a reason
Each exported definition SHALL appear in exactly one configured page or in `exclude` with a reason.

#### Scenario: A placed and an excluded definition
- **WHEN** one exported definition is on one configured page and another is in `exclude` with a reason
- **THEN** both are accepted

### Requirement: A config in the source tree must place or exclude every definition
In a build whose config is in the source tree, each of these SHALL fail the build with exit 2, all of them named in one message: an unplaced definition (with its file), a definition placed on two pages, a placed name the package does not declare, an excluded name the package does not declare, and a name both placed and excluded.

#### Scenario: A new definition
- **WHEN** core adds `#Policy` and `docs-kit.cue` neither places nor excludes it
- **THEN** `opm-docs check` exits 2 naming `#Policy` and the file

### Requirement: A config in the source tree must give each definition a summary and an anchor
In a build whose config is in the source tree, each of these SHALL also fail the build with exit 2, all of them named in one message: a placed definition whose doc comment yields no summary (missing, or holding only rationale or citations), and two definitions on one page whose anchors collide (names that differ only in case).

#### Scenario: Colliding anchors
- **WHEN** two definitions on one page have names that differ only in case, and the config is in the source tree
- **THEN** the build exits 2 naming both

### Requirement: A config from outside the source tree gives warnings
When the config came from outside the source tree (a backfill), each of these SHALL be a warning on stderr: an unplaced definition SHALL be left out, an undeclared name skipped, a second placement and a colliding name ignored, a definition without a summary kept with an empty one, and a page left with no definition SHALL be dropped.

#### Scenario: Backfill of an older tag
- **WHEN** a release build of `v2.0.0-beta.1` uses `main`'s config, which places `#Policy` that the tag lacks
- **THEN** the build warns naming `#Policy` and succeeds

#### Scenario: A definition without a doc comment in a backfill
- **WHEN** a backfill's tag declares a placed `#Component` with no doc comment
- **THEN** the build warns naming `#Component` and its file, and `#Component`'s entry has an empty summary and no summary paragraph

### Requirement: The renderer writes a section index and one page per group
The renderer SHALL write `<section>_index.md` (the configured title and description, the generated "Pages" list and the "All definitions" table) and `<section><file>.md` per configured page, weighted in configuration order, each definition under a `## <name>` heading whose anchor is the name lowercased without `#`, and every cross-reference linking `/docs/<section><file>/#<anchor>`.

#### Scenario: Link to another group
- **WHEN** `#Component` uses `#NameType`, placed on `names-paths-and-versions`
- **THEN** the components page links `/docs/reference/definitions/names-paths-and-versions/#nametype`

### Requirement: The definitions section lies under an owned path
The section SHALL lie under one of the bundle's owned paths, and the source SHALL be refused (exit 1) in a bundle whose placement is not `docs`.

#### Scenario: A source in a bundle that is not docs
- **WHEN** a `cue-definitions` source is configured in a bundle whose placement is not `docs`
- **THEN** the source is refused with exit 1

### Requirement: The definition pages match core's refgen output
The pages SHALL match core's `tools/refgen` output for the same tree, except that no page carries a generator marker comment.

#### Scenario: Parity with refgen
- **WHEN** core at commit `c5a6076`, the commit the parity was measured at, is built with core's configuration (CI checks it out and runs the parity test)
- **THEN** every page equals refgen's committed page at that commit with the marker comments removed
