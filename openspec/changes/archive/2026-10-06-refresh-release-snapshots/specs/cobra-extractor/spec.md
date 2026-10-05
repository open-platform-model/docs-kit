## REMOVED Requirements

### Requirement: The cobra source renders the command reference
**Reason**: it ties the pages to the cli's `internal/cmdref` output, and its scenario "Parity with cmdref" checks frozen cmdref pages. cmdref is retired upstream and the copy can never be refreshed; OpenSpec cannot drop a scenario in place.
**Migration**: replaced, with the same pages, by "The cobra source renders a command reference section"; parity moves to "The cli's release snapshot renders to its released pages".

## ADDED Requirements

### Requirement: The cobra source renders a command reference section
A `cobra` source SHALL run its command (C14), validate the dump, write `data/cobra.json` (`docs.opmodel.dev/data/cobra/v1`), and render `<section>_index.md` with the global flags and `<section><root>-<top>.md` for each top-level command holding that command and every command under it, depth first, each under a heading whose anchor is the command path in kebab case. The section SHALL be one of the bundle's owned paths. Every command name SHALL be lower-case kebab-case. No page SHALL carry a generator marker comment.

#### Scenario: A subcommand link
- **WHEN** `opm module` lists `apply`
- **THEN** its subcommand table links `/docs/reference/cli/opm-module/#opm-module-apply`

#### Scenario: An unknown dump schema
- **WHEN** the command prints a document with schema `docs.opmodel.dev/cobradump/v2`
- **THEN** `build` exits 2 naming the schema and the one this `opm-docs` reads

### Requirement: The cli's release snapshot renders to its released pages
docs-kit SHALL keep a release snapshot of the cli under `internal/render/testdata/cli`: the pages of one released cli docs bundle under its owned section, verified by `opm-docs pull`, the release commit's `docs-kit.cue`, the dump that commit's `hack/docskit-dump` printed (two runs, identical), and a `SOURCE` naming the bundle tag, digest, docs revision, the opm-docs version that built it and the commit. Rendered with the snapshot's own cobra source config, the pages SHALL equal the released pages, page set and bodies byte for byte. `task snapshots:refresh` SHALL take the snapshot from a released version and a clone holding its commit.

#### Scenario: Parity with the released cli bundle
- **WHEN** the snapshot's dump is rendered with its `docs-kit.cue` cobra source
- **THEN** every page equals the released page of the same path, and the released section holds no other page

#### Scenario: The dump differs between runs
- **WHEN** two runs of the release's `hack/docskit-dump` print different output during a refresh
- **THEN** `task snapshots:refresh` exits non-zero and the snapshot is unchanged
