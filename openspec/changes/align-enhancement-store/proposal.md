# Proposal

## Why

The enhancements store replaces the `enhancements/NNNN/` layout. A decision number (`NNNN:Dn`, `NNNN Dn`) no longer names anything a reader can open, because the new entries have `requirements`, not numbered decisions. The workspace rule is now: cite an enhancement as its id beside a reason in words, never a decision number and never a lone id. docs-kit still cites the old form in its rules, comments, contracts and specs, and its `link` citation policy still links `NNNN:Dn` to a `decisions` page that new-layout entries do not have.

## What Changes

- `openspec/config.yaml` and `AGENTS.md`: add `references:` to the store `opm-enhancements`, the `Delivers` proposal rule and the comment rule; drop the `NNNN:Dn` form.
- Rewrite every enhancement citation in tracked docs, live specs, archived changes and Go comments and strings to "enhancement NNNN" beside its reason. The first pass is `enh cite-rewrite`; a hand pass clears its worklist.
- The `link` citation policy (`internal/doctext/policy.go`) keeps only the link from "enhancement NNNN" to `/enhancements/NNNN/`. A decision citation is removed under `link`, as under `strip`.
- `docs/contracts.md` (C6, C18, C20, C21, Doc-comment rules) and the live specs state the new `link` policy.
- Code that strips or reads old citation forms stays, with its tests and testdata: a docs revision builds an old release tag whose comments still hold them. The lint skips those test files and every `testdata/` folder by path.

No spec delta: the live spec edits only reword the citation examples and state the `link` policy that the contracts already fix. `.openspec.yaml` sets `skip_specs: true`.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
None. Spec text changes as wording only (no requirement changes its observable behaviour for the repos that use the default `strip` policy).

## Impact

- Consumers of a contract: `opm-operator` (`docs-kit.cue` uses `citations: "link"`; its own change drops that line, per the enhancements design). `core`, `catalog_opm`, `cli`, `library`, `opm`, `opmodel.dev` use the default `strip` policy and see no change in the pages they build.
- Release class: the change is `docs`, `test` and `chore` only, so release-please makes no release. The `link` policy change reaches a consumer with the next `feat` or `fix` release of docs-kit.

## Enhancement

None. The store entry for this alignment lives in `enhancements`; this repo holds no line of its `tasks.md`.
