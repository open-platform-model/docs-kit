# Proposal

## Why

The comment rule of the enhancement store says a comment names an enhancement by its id beside a reason in words, never by a decision number and never by a lone id. The rule has a shared tool and a shared reusable workflow, and every other repo of the workspace runs both. docs-kit runs neither, so a later citation by decision number passes its review. Now the repo runs the same check locally and in CI, so such a citation fails the PR that adds it.

## What Changes

- A local task `cite-lint` runs the tool at one pinned enhancements commit, with the smallest set of excludes.
- `task check` runs `cite-lint`.
- A new small workflow calls the reusable `cite-lint.yml` of the `.github` repo, pinned to a full commit, on `pull_request` and on `push` to `main`.
- `AGENTS.md` lists the task and the CI check.
- The OpenSpec CLI pin stays at 1.12.0: `openspec validate --all --strict` fails under 1.14.1 on existing specs (see design.md).

## Capabilities

### New Capabilities
None.

### Modified Capabilities
None. The change touches the repo's own CI and task files; no contract that another repo reads.

## Impact

- Affects `Taskfile.yml`, one new file under `.github/workflows/`, `AGENTS.md`.
- No Go code, no bundle, no workflow interface of docs-kit changes.
- Release class: none. Every commit is `ci`, `build`, `docs` or `chore`.

## Enhancement

None. No line of an enhancement `tasks.md` is delivered here; the wiring is part of the cross-repo roll-out of the comment rule.
