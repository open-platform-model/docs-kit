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

1. `F` is 40 hex, a single-parent commit, and an ancestor of `origin/main` ("the fix must land on `main` first"), and not already in `T` ("already in the release"; implementation addition).
2. `T` carries the project's tag prefix (from `--config`, else the checkout's `docs-kit.cue`); its version `V` has a published revision 0 in the registry ("publish the release first: dispatch `mode: release`").
3. Read the `source.patches` of the newest published revision `n` of `V` (empty for revision 0), from its layer unpacked with every check `pull` applies; its manifest must name `V`, `n` and `T`'s commit as `source.commit` (implementation addition). The new list is that list plus `F`, and every earlier fix is checked again as in step 1 (implementation addition). A fix already in the list is refused ("already applied in `V.<n>`"), except a re-run: when `F` is the last fix of revision `n` and the release tag `V` does not name revision `n` (it was pushed but never promoted), revision `n` is built again with the same list instead (supervisor decision, 2026-10-03). The build is deterministic, so `push` finds the same digest and writes nothing, and the workflow signs and promotes it.
4. `git worktree add --detach <tmp> T`; `git cherry-pick --no-commit` each patch in order; a conflict is refused naming the files. The worktree is removed on every path.
5. The documentation-only check (D2) between `T`'s tree and the worktree.
6. `build --release T --source <tmp> --revision <n+1> --patches <list>` into `--out`, with `--config` when given, else the release tree's `docs-kit.cue`, else the checkout's, as a release build picks it (implementation finding; D1 first named only `docs-kit.cue`). `source.commit` is `T`'s commit; `lastmod` of a patched file is the committer date of the newest patch that touched it.

`revise` pushes nothing (decided in planning for phase 1, kept here): the workflow runs `push`, `cosign sign` and `promote` after it, so a tag never points at an unsigned build.

### D2. Documentation only

| Path | Allowed |
|---|---|
| `*.md` | added, changed, renamed, removed |
| `*.cue` | changed only; both versions scanned by `cue/scanner` with comments skipped (interpolations resumed as the parser does), token sequences equal; a comma the scanner inserts at a line end equals a written one |
| `*.go` | changed only; both versions scanned by `go/scanner` with comments skipped, token sequences equal (an inserted semicolon equals a written one); the directive comments (`//go:build`, `//go:embed`, `//line`, `//export`, any `//word:word`, `// +build`) equal in order; never a file that imports `"C"` |
| a symlink or submodule, at any path | refused |
| anything else, or an added, renamed or removed `.cue`/`.go` | refused, naming the file |

The trees compared are the release tag's and the index of the temporary worktree after the cherry-picks (`git write-tree`), with `git diff-tree -r -M`. Every refusal names the file and why, and ends "a change to code needs a patch release".

A change to `metadata.description` is a value change and is refused: the member's summary is part of its contract, so it needs a patch release.

## Research & Decisions

### Comparing code with comments removed (implementation finding)

**Context**: D2 as planned compared formatted output: CUE parsed, comments removed, printed with `cue/format`; Go parsed without comments, printed with `go/printer`; bytes equal.
**Explored**: a test of both printers on a doc comment added between two struct fields (`a: int` / `// b is b.` / `b: int`, and the Go equivalent). Both printers keep the line gap the removed comment leaves as a blank line, so the outputs differ and the commonest documentation fix, adding a field's doc comment, would be refused. Rewording a comment of the same length passes; adding or removing a comment line next to code does not.
**Options considered**:
1. Formatted-bytes equality as planned - false refusals of ordinary doc-comment fixes.
2. Formatted bytes with blank lines removed - hides a change to a multi-line string's blank lines, a value change accepted.
3. Token sequences with comments skipped - layout-blind and exact on every literal, identifier, operator and attribute; line ends that the language reads (Go's inserted semicolons, CUE's inserted commas) are tokens, so they are still compared.
**Decision**: option 3, plus Go's directive comments compared separately and cgo files refused, since those comments are code that no syntax tree without comments sees (the planned check missed them too).
**Rationale**: token equality means the same program, which is what "documentation only" needs; the only freedom it adds over option 1 is layout, which changes no value.

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

Permissions as `release` (`contents: read`, `packages: write`, `id-token: write`); the `refs/heads/main` guard and the GHCR read login apply. The job runs in the release's concurrency group, `docs-release-${{ inputs.project }}-${{ inputs.tag }}` with `cancel-in-progress: false` (`docs/contracts.md` C5), so revisions of one release are serialized, never race for a revision number, and never cancel the release publish.

### Re-running a failed revision (review finding)

**Context**: a revision job that fails after `push` (at signing or promote) leaves revision `n` pushed with `F` as its last fix. Re-running the job ran `revise` again, which refused `F` as already applied, so revision `n` could never be signed or promoted.
**Options considered**:
1. Always refuse - a failed run strands its revision; recovery needs another fix.
2. Rebuild revision `n` when `F` is its last fix and the release tag does not name it - same digest, so `push` is a no-op and signing and promote complete the run.
**Decision**: option 2, accepted by the supervisor 2026-10-03. A promoted revision, or an `F` that is not the last fix, is still refused.
**Rationale**: the build is deterministic (Principle V), so a rebuild cannot publish anything new; it only lets a failed run finish.

## Risks / Trade-offs

- [A cherry-pick of an older fix conflicts with a newer one] → refused naming the files; the author lands a combined fix on `main` and revises with that.
- [Comment-only comparison misses a semantic change hidden in a comment-like construct] → CUE and Go scanners read comments as comments only; the comparison is on token streams with comments skipped, never on text. Comments that are code (Go directives, the cgo preamble) are compared, with their place, or refused.

## Review amendments (PR review, 2026-10-03)

- Go directives are compared with their place (the number of code tokens before each), so a moved `//go:embed`, `//go:noinline` or `//go:build` is refused.
- Patched paths and their dates come from the index after each pick, not from the fix commits, so a file main renamed after the release is dated under its release-tree name; `revise` refuses a work tree that differs from its index, and a fix that leaves the tree unchanged.
- A revision counts as promoted only when every tag `promote` would move to it names it; a partly promoted one is rebuilt.
- The re-run rebuild is refused when the newest revision was built by another `opm-docs` version.
- The newest revision's signature is verified before its `source.patches` is trusted; an unsigned one is accepted only for a re-run, when the signed revision before it plus the fix gives its list.
- A `.md` change is refused when the release's CUE declares `@extern(embed)`; a path that was a symlink or submodule is refused; `NextRevision` needs revision 0 itself; the 40-hex (SHA-1) limit is stated.
- A cancelled pending revision run must be dispatched again (README, contracts).

## Durable decisions

| Decision | Lands in |
|---|---|
| A docs revision is the only way to change a published release's pages, and how to run one | `README.md` |
| D1's steps and D2's table (the specs cite them), and the token comparison | `docs/contracts.md`, a new "Docs revisions" section |
