# Design: add-docs-revisions

## Context

The contracts are `build-opm-docs-phase-1`'s (after its archive, `docs/contracts.md`): C3 `manifest.json` (`revision`, `source.patches`), C4 tag rules 1 to 5, C5 the workflow. This change adds behavior only. `DESIGN.md`, "Docs revisions", steps 1 to 4, is the process; this change builds steps 3 and 4.

## Goals / Non-Goals

**Goals:** `opm-docs revise`; `build --revision`; the `revision` mode of `publish.yml`; the end-to-end proof on catalog_opm and the site.

**Non-Goals:** any contract change; revisions of edge (edge is rebuilt on every push); a revision that changes code (that is a patch release).

## Decisions

### D1. The command

```text
opm-docs revise --project P --tag T --fix F [--out out] [--registry ghcr.io/open-platform-model/docs] [--config docs-kit.cue]
```

Steps, each failing with exit 2 and a message naming the fix:

1. `F` is 40 hex, a single-parent commit, and an ancestor of `origin/main` ("the fix must land on `main` first").
2. `T` carries the project's tag prefix; its version `V` has a published revision 0 in the registry ("publish the release first: dispatch `mode: release`").
3. Read the `source.patches` of the newest published revision of `V` (empty for revision 0); the new list is that list plus `F`. A fix already in the list is refused ("already applied in `V.<n>`").
4. `git worktree add --detach <tmp> T`; `git cherry-pick --no-commit` each patch in order; a conflict is refused naming the files. The worktree is removed on every path.
5. The documentation-only check (D2) between `T`'s tree and the worktree.
6. `build --release T --source <tmp> --revision <n+1> --patches <list>` into `--out`. `source.commit` is `T`'s commit; `lastmod` of a patched file is the committer date of the newest patch that touched it.

`revise` pushes nothing (decided in planning for phase 1, kept here): the workflow runs `push`, `cosign sign` and `promote` after it, so a tag never points at an unsigned build.

### D2. Documentation only

| Path | Allowed |
|---|---|
| `*.md` | added, changed, renamed, removed |
| `*.cue` | changed only; both versions parsed with comments, every comment removed (`ast.Walk` clearing comment groups), formatted with `cue/format`, bytes equal |
| `*.go` | changed only; both versions parsed by `go/parser` without comments, printed with `go/printer`, bytes equal |
| anything else, or an added, renamed or removed `.cue`/`.go` | refused, naming the file |

A change to `metadata.description` is a value change and is refused: the member's summary is part of its contract, so it needs a patch release.

### D3. The workflow mode

`publish.yml` gains:

```yaml
      mode:
        description: "check | edge | release | revision"
      fix:
        description: revision mode, the 40-hex commit on main whose documentation change to apply
        type: string
        default: ""
```

| Mode | Caller runs it on | Steps |
|---|---|---|
| `revision` | `workflow_dispatch` | checkout `main` with full history; `revise --tag <tag> --fix <sha>`; `push`; `cosign sign`; `promote` |

Permissions as `release` (`contents: read`, `packages: write`, `id-token: write`); the `refs/heads/main` guard and the concurrency group apply, so two revisions of one project never race for a revision number.

## Risks / Trade-offs

- [A cherry-pick of an older fix conflicts with a newer one] → refused naming the files; the author lands a combined fix on `main` and revises with that.
- [Comment-only comparison misses a semantic change hidden in a comment-like construct] → CUE and Go parse comments as comments only; the comparison is on formatted syntax trees, not text.

## Durable decisions

| Decision | Lands in |
|---|---|
| A docs revision is the only way to change a published release's pages, and how to run one | `README.md` |
| D2's table | `docs/contracts.md`, a new "Docs revisions" section |
