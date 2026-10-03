# Tasks: pull-docs-placement

Gate: `generalize-build-assembly` is merged on docs-kit `main`. One PR, titled `feat: pull docs bundles per site version`.

## 1. Config and lock schemas

- [ ] 1.1 `schema/pull.cue` (design.md D1) and `internal/config` (`Pull.Docs`, `Pull.Versions`, the pre-network checks). Verify: tests for opmodel.dev's planned file, an undeclared project, a project twice in one version, a project both tab and docs.
- [ ] 1.2 `schema/lock.cue` and `internal/pull/lock.go`: the `docs` key, entry forms, key order and sort (D5). Verify: a lock without `docs` still validates; encode/decode round trip; sort order test.
- [ ] 1.3 `task check` green, then commit `feat(pull): accept site versions in the pull config and lock`.

## 2. Resolution and layout

- [ ] 2.1 `internal/pull`: anchor, pins and own tags (D2); the `_versions/` layout, per-version staging and replacement, sweep (D3). Verify: in-process registry tests (as the tab tests) for a pinned resolution, a docs revision followed through the release tag, a missing pin, a pinned release without a bundle, an anchor outside its line, a refused version keeping the old one.
- [ ] 2.2 `task check` green, then commit `feat(pull): resolve site versions from the anchor's pins`.

## 3. Cross-bundle checks, local and frozen

- [ ] 3.1 The D4 checks over the staged version. Verify: tests for a duplicate path, a page under another bundle's owned path, overlapping owned paths.
- [ ] 3.2 `--local <project>@v<M>.<m>=<dir>` parsing and behavior, `--frozen` and `--offline` for docs entries (D6). Verify: an all-local version with no network; a local anchor pinning a pulled project; a local pinned tree off its pin (exit 1); a frozen lock with a changed role (exit 1); a frozen offline rebuild writing the same lock.
- [ ] 3.3 `task check` green, then commit `feat(pull): check site versions for overlapping bundles`.

## 4. Contracts and archive

- [ ] 4.1 `docs/contracts.md`: C16 (new; D1 to D6) and C7 (config, layout, lock, `--local`); `README.md` (`pull`). Verify: schema text equals `schema/pull.cue` and `schema/lock.cue`.
- [ ] 4.2 `openspec archive pull-docs-placement --yes`. Verify: `task openspec:check` green.
- [ ] 4.3 `task check` green, then commit `docs(pull): document site versions`.
