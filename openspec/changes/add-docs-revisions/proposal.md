## Why

DESIGN decision 6 says a fix to released docs ships as a docs-only revision of that release, never by overwriting it, and `DESIGN.md`'s phase-1 done criterion asks for a docs revision of opm 4.4.x to reach the site without a catalog release. `build-opm-docs-phase-1` ships every contract a revision needs (the `revision` field and annotation, `source.patches`, the tag rules, and `promote` and `pull` handling any revision) but not the command that builds one. This change adds it, split out of phase 1 to keep that change within five sections (supervisor decision, 2026-10-02).

Gate: starts only after `build-opm-docs-phase-1` is archived on docs-kit `main` and released as `v0.1.0`. Its delta specs extend capabilities that change creates.

## What Changes

- **`opm-docs revise`**: checks the fix is a single-parent commit on `origin/main`, applies every earlier revision's fixes and the new one to the release tree in a temporary worktree, refuses anything but a documentation-only change, and builds the next revision. It pushes nothing; `push`, `cosign sign` and `promote` follow in the workflow, as for a release.
- **`build --revision`** (an internal flag `revise` uses) writes `revision` and `source.patches` into `manifest.json`.
- **The workflow's `revision` mode** in `publish.yml`: `workflow_dispatch` only, inputs `tag` and `fix`, the same permissions as `release`.
- **Done criterion** for phase 1's second half: a doc-comment fix on catalog_opm `main` reaches `/catalogs/opm/4.4/` as `4.4.<n>.1` with no catalog release.

SemVer class: MINOR (`0.2.0`): a new command and a new workflow mode; no contract changes. Bundles it builds are readable by a 0.1.0 `pull`, because phase 1's schema already accepts `source.patches` and any revision.

Scope: three implementation sections, then the archive step.

## Capabilities

### New Capabilities

- `docs-revision`: the documentation-only check, patch accumulation and how a revision is built.

### Modified Capabilities

- `publish-workflow`: adds the `revision` mode.
- `opm-docs-cli`: adds the `revise` command.

## Impact

- Code: `internal/gitsrc` (worktrees, cherry-pick, the documentation-only check), `cmd/opm-docs/revise.go`, `.github/workflows/publish.yml`, `README.md`.
- Consumers: catalog_opm moves `docs.yml` and its Taskfile to `v0.2.0` and adds `revision` (with a `fix` input) to its dispatch choices: a `ci` PR, no OpenSpec change. opmodel.dev: nothing; `pull` already handles revisions.
- Risk: a fix that touches code and docs is refused whole; the author splits it, or ships a patch release.
