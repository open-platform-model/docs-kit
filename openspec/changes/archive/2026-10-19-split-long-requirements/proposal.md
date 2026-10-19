## Why

OpenSpec 1.14.1 raises "Requirement text is very long (>500 characters)" from INFO to WARNING.
`openspec validate --strict` fails on any warning, and no config key or flag turns it off: the
limit is a constant in the validator. The warning is right on the merits. A long requirement
usually holds several behaviors, and one behavior per requirement makes each one testable by its
own scenarios.

The repo bumped the CLI to 1.14.1 and switched `--strict` off in `openspec:check` as a stopgap
(`Taskfile.yml`, `AGENTS.md`, `openspec/config.yaml` all say "`--strict` is off until the
long-requirement rewrite lands"). Today `openspec validate --all --strict` reports 47 such
warnings in the main specs, in 17 of the 18 specs. This change rewrites them so that `--strict`
passes again with no rule lost or changed.

## What Changes

The rewrite follows the approved guide of the swarm `openspec-requirement-split` (forms a, b,
c2 and d, the scenario ledger, the gate and the review checklist). Every rewrite is a split of a
requirement or a move of detail into scenarios. No rule, value, default, exit code or citation
changes. No code changes.

- Batch docs-kit-B1, 28 requirements: `bundle-pull` (9), `enhancements-bundle` (6),
  `crd-extractor` (4), `cue-definitions-extractor` (3), `opm-docs-cli` (3),
  `publish-workflow` (3).
- Batch docs-kit-B2, 19 requirements: `bundle-format` (2), `cobra-extractor` (2),
  `dialect-lint` (2), `docs-placement` (2), `docs-revision` (2), `go-api-extractor` (2),
  `page-renderer` (2), `tool-release` (2), `cue-catalog-extractor` (1),
  `repository-commands` (1), `version-history` (1).
- Repo extra: `--strict` returns to `openspec:check` in `Taskfile.yml`, and the text that says
  it is off is corrected in `Taskfile.yml`, `AGENTS.md` and `openspec/config.yaml`.

The spec deltas are written batch by batch during apply, not in this proposal step. The
`tag-scheme` spec has no long requirement and is not touched.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

All 17 specs of the two batches above. No new scenario changes a rule; every added scenario
uses only words and values of the old requirement body.

## Impact

- Affected: `openspec/specs/` of this repo through deltas, and three text lines plus one
  command in `Taskfile.yml`, `AGENTS.md` and `openspec/config.yaml`.
- Contracts: none changes. The bundle format, tag scheme, workflow interface, `docs-kit.cue`,
  pull config and lock keep their meaning, so no consuming repository has anything to do.
- Open changes: `openspec/changes/` holds no active change, so no rewrite collides with one.
- Release class: none. Commit types are `docs` for spec text and `ci` or `chore` for the gate;
  none of them releases `opm-docs`.
- Risk: a rewrite that loses a scenario or changes its text passes OpenSpec without an error.
  The guide's checklist (item 3, scenario text compared byte for byte) and the ledger in
  `design.md` guard it.

## Enhancement

None.
