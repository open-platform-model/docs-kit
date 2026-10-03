# Tasks: pull-docs-placement

Gate: `generalize-build-assembly` is merged on docs-kit `main`. One PR, titled `feat: pull docs bundles per site version`.

## 1. Config and lock schemas

- [x] 1.1 `schema/pull.cue` (design.md "Pull config") and `internal/config` (`Pull.Docs`, `Pull.Versions`, the pre-network checks). Verify: tests for opmodel.dev's planned file, an undeclared project, a project twice in one version, a project both tab and docs.
- [x] 1.2 `schema/lock.cue` and `internal/pull/lock.go`: the `docs` key, entry forms, key order and sort (design.md "Lock"). Verify: a lock without `docs` still validates; encode/decode round trip; sort order test.
- [x] 1.3 `task check` green, then commit `feat(pull): accept site versions in the pull config and lock`.

## 2. Resolution and layout

- [x] 2.1 `internal/pull`: anchor, pins and own tags (design.md "Resolution"); the `_versions/` layout, per-version staging and replacement, sweep (design.md "Layout and replacement"). Verify: in-process registry tests (as the tab tests) for a pinned resolution, a docs revision followed through the release tag, a missing pin, a pinned release without a bundle, an anchor outside its line, a refused version keeping the old one.
- [x] 2.2 `task check` green, then commit `feat(pull): resolve site versions from the anchor's pins`.

## 3. Cross-bundle checks, local and frozen

- [x] 3.1 The checks of design.md "Checks across one version" over the staged version. Verify: tests for a duplicate path, a page under another bundle's owned path, overlapping owned paths.
- [x] 3.2 `--local <project>@v<M>.<m>=<dir>` parsing and behavior, `--frozen` and `--offline` for docs entries (design.md "`--local`, `--frozen`, `--offline`"). Verify: an all-local version with no network; a local anchor pinning a pulled project; a local pinned tree off its pin (exit 1); a frozen lock with a changed role (exit 1); a frozen offline rebuild writing the same lock.
- [x] 3.3 `task check` green, then commit `feat(pull): check site versions for overlapping bundles`.

## 4. Contracts and archive

- [x] 4.1 `docs/contracts.md`: C16 (new; design.md "Pull config" to "`--local`, `--frozen`, `--offline`") and C7 (config, layout, lock, `--local`); `README.md` (`pull`). Verify: schema text equals `schema/pull.cue` and `schema/lock.cue`.
- [x] 4.2 `openspec archive pull-docs-placement --yes`. Verify: `task openspec:check` green.
- [x] 4.3 `task check` green, then commit `docs(pull): document site versions`.
