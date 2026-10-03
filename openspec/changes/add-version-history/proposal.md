## Why

The Catalogs tab shows each opm minor on its own, so a reader cannot see what a minor added, changed or removed. `DESIGN.md` phase 1b fixes the scope (DESIGN decision 12: badges and a per-member "Changes in X" list, no side-by-side diff), and `docs/contracts.md` "Site decisions" fixes where the data comes from: a file `opm-docs pull` writes at `<out>/<project>/history.json` from the `data/` of every segment it pulled, which the site reads and never computes. Phase 1 reserved that path. With opm 4.5 and edge published, there are two segments to compare today.

## What Changes

- **`internal/history`**: a pure package that compares the `data/catalog.json` (C10) of a tab's segments in order (minors ascending, then `edge`) and computes, per member FQN, where it first appears (with the floor rule: a member of the oldest pulled minor reads "in <floor> or earlier"), the segments it is in, its field changes against the previous segment, the removed members per segment with the last segment that had them, and the apiVersion lineage of each name and kind (for "Newer version" links).
- **The tool-minor gate**: `type`, `default` and `ref` strings are stable only within one docs-kit minor (C10), so two segments built by different tool minors are compared by field paths and presence only, and the file says so per pair.
- **The spec-text fallback**: when the field list shows no change but the spec block's CUE tokens (comments skipped) differ, the member gets one `spec` change, so a `matchN` or `if`-guard change the walk cannot express still shows.
- **`opm-docs pull` writes `history.json`** for every tab project with two or more segments, every run (pulled, frozen, offline or local), and removes it when the project has fewer. `schema/history.cue` validates it before it is written.
- **The lock records each history file's SHA-256** in an optional top-level `history` list, so the site can check the file it mounts is the one this pull wrote.
- **`docs/contracts.md` C13** "Version history": the file, its schema and the comparison rules.

Release class: MINOR (`0.3.0`). Additive: a new file under `<out>/`, an optional lock key. A 0.2.x site keeps working until it bumps.

Scope: three sections, then the archive.

## Capabilities

### New Capabilities

- `version-history`: what `history.json` holds and how it is computed.

### Modified Capabilities

- `bundle-pull`: writes `history.json` and records its digest in the lock (requirements added; none modified).

## Impact

- Code: new `internal/history`, `schema/history.cue`; `internal/pull` (write after the sweep, lock entry), `schema/lock.cue` (optional `history`); the CUE token scanner moves from `internal/gitsrc/doconly.go` into a shared unexported-API package `internal/cuetok` so `history` and `gitsrc` use one scanner.
- Contracts: C7 (lock gains `history`), C13 (new). `pull`'s sweep already keeps `history.json` (C7).
- Consumers: **opmodel.dev** (change `add-catalog-version-history`): bumps its pinned `opm-docs` to the release carrying this, mounts `*/history.json`, checks its digest against the lock, renders the badges and the "Changes in <segment>" list, extends its fixture bundles. **catalog_opm**: nothing; its bundles already carry `spec.fields`.
- Risk: a member whose shared schema (`ref`) changed shows no change, because the walk stops at `ref`; the badge then under-claims, which is the safe direction.
