# Tasks: add-docs-revisions

Gate: `build-opm-docs-phase-1` is archived on docs-kit `main` and `v0.1.0` is released. Do not start before. One PR, titled `feat: add docs revisions`.

## 1. Git source handling and the documentation-only check

- [ ] 1.1 `internal/gitsrc`: ancestor and single-parent checks, temporary worktree at a tag, ordered `cherry-pick --no-commit`, conflict reporting, worktree cleanup on every path. Verify: tests with a temporary repository, including a conflict and a merge commit as the fix.
- [ ] 1.2 The documentation-only check (design.md D2). Verify: table tests for each row, including a `metadata.description` value change (refused) and a comment change (allowed) in `.cue`, and a comment change in `.go`.
- [ ] 1.3 `task check` green, then commit `feat(revise): add the documentation-only check`.

## 2. The revise command

- [ ] 2.1 `build --revision` and `--patches` (hidden flags) writing `revision` and `source.patches`. Verify: the manifest validates and a 0.1.0-era `pull` test fixture reads it.
- [ ] 2.2 `cmd/opm-docs/revise.go` per design.md D1. Verify: in-process registry tests for the docs-revision scenarios: fix not on main, no revision 0, first revision, second revision carrying the first fix, a fix already applied.
- [ ] 2.3 `task check` green, then commit `feat(revise): build docs revisions of a published release`.

## 3. The workflow mode and docs

- [ ] 3.1 `publish.yml`: the `revision` mode and `fix` input (design.md D3). Verify: `actionlint` clean.
- [ ] 3.2 Durable decisions: `README.md` (how to run a revision, with the catalog_opm dispatch as the example) and `docs/contracts.md` ("Docs revisions": D1's steps and D2's table, which the specs cite). Verify: links resolve.
- [ ] 3.3 `task check` green, then commit `feat(workflow): add the revision mode`.

## 4. Archive (rides this PR)

- [ ] 4.1 `openspec archive add-docs-revisions --yes`. Verify: `task openspec:check` green; every durable decision landed.
- [ ] 4.2 `task check` green, then commit `chore(openspec): archive add-docs-revisions`.

## After merge

The `0.2.0` release, catalog_opm's bump and the revision proof are cross-repo steps, listed in `build-opm-docs-phase-1/orchestration.md`, "Follow-up: docs-kit `add-docs-revisions`".
