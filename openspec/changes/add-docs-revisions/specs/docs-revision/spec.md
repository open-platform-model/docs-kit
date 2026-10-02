## Purpose

How a documentation fix reaches a published release as a docs revision, without a code release.

## ADDED Requirements

### Requirement: A docs revision applies fixes from main to a release tree
`opm-docs revise --project P --tag T --fix F` SHALL refuse unless F is a single-parent commit that is an ancestor of `origin/main`, and unless revision 0 of T's version is published. It SHALL read the `source.patches` of the newest published revision of that version, check out T in a temporary worktree, apply those patches and then F with `git cherry-pick --no-commit` in order, and refuse on a conflict naming the files. Source: DESIGN decision 6.

#### Scenario: Fix not on main
- **WHEN** F is not an ancestor of `origin/main`
- **THEN** `revise` exits 2 saying the fix must land on `main` first

#### Scenario: Second revision carries the first fix
- **WHEN** 4.4.5.1 was built with patches `[A]` and `revise --fix B` runs
- **THEN** the new build applies A then B and records `source.patches: [A, B]` and revision 2

### Requirement: Only documentation may change
Between the release tree and the patched tree, `revise` SHALL allow any added, changed, renamed or removed `.md` file; a changed `.cue` file only when both versions, parsed with every comment removed and formatted with `cue/format`, are byte-identical; a changed `.go` file only when both versions, parsed without comments and printed with `go/printer`, are byte-identical. It SHALL refuse every other change, naming each file and why.

#### Scenario: A comment fix in CUE
- **WHEN** F changes only a doc comment in `opm/traits/v1alpha1/backup.cue`
- **THEN** the check passes and the revision builds

#### Scenario: A description string change
- **WHEN** F changes the value of `metadata.description` in a member file
- **THEN** `revise` exits 2 naming the file and that a change to CUE values needs a patch release

### Requirement: The revision is built, not pushed
After the check passes, `revise` SHALL build the project at the release version with the next revision number into `--out`, with `source.commit` the release commit, `source.ref` the tag and `source.patches` every applied fix. Pushing, signing and tag moves SHALL be left to `push`, `cosign sign` and `promote`.

#### Scenario: Revision output
- **WHEN** `revise --project catalog-opm --tag opm-v4.4.5 --fix B` succeeds with 4.4.5.0 as the newest published revision
- **THEN** `out/catalog-opm/manifest.json` has version `4.4.5`, revision 1, `source.ref` `opm-v4.4.5` and `source.patches: [B]`, and nothing was pushed
