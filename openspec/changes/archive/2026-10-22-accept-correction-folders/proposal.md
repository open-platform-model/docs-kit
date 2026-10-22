## Why

The enhancements store is adding a route to fix a delivered capability without a new entry: an
unnumbered change `correction-<slug>` that modifies, removes, adds or renames requirements of one
or more capabilities, and is archived as `<date>-correction-<slug>`. The `enhancements` source of
`opm-docs` refuses every change folder that is not an entry or a `seed-` change (exit 2). The first
correction on the store's `main` would therefore break its docs build. The source must treat a
correction like a seed: no entry, no page.

## What Changes

- The openspec reader skips a live `correction-<slug>` folder and an archived
  `<date>-correction-<slug>` folder, as it skips a seed. They are never entries.
- The refusal message for any other folder name names the correction form beside the seed form.
- A capability page lists a correction in "Entries that changed it" as an unlinked row `correction`
  with its archive date, for each capability whose `specs/` the correction touches. A correction
  is never listed as an entry and has no link. This mirrors the row `seed` and is the simplest
  option: no new column, no new data.
- `docs/contracts.md` (the openspec-reader paragraph and the capability page paragraph of C21)
  and the `enhancements-bundle` spec say so.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `enhancements-bundle`: the openspec reader skips correction folders; the capability page lists
  them as unlinked `correction` rows.

## Impact

- Package `internal/extract/enhancements` (`openspec.go`, `capabilities.go`, tests).
- Contract: `docs/contracts.md` C21, the enhancements source. The change only accepts a folder
  name that was refused before, so it is additive. `data/enhancements.json` is unchanged: a
  correction is in no entry list. Consumer: `enhancements`, the only repository with this source.
  It must bump `.opm-docs-version` and every `publish.yml` ref to the release that carries this
  change, in one PR, before its first correction folder lands. No other consumer
  (catalog_opm, core, cli, opm-controller, library, opm, opmodel.dev) reads this source.
- Not breaking. Release class: minor bump of `opm-docs` (docs-kit is on 0.x; `feat` releases).
- Commit type and PR title of the implementing PR: `feat(extract): accept correction folders in the enhancements bundle`.

## Enhancement

None.
