# Tasks: refuse-version-mismatch-and-align-lint

One PR closing docs-kit#8 and docs-kit#10; its `fix` commits release `v0.2.1`.

## 1. Refuse a release whose tag and catalog versions differ

- [x] 1.1 `internal/build`: after the `cue-catalog` source is extracted in a non-edge build, refuse a tag version that differs from `metadata.version` (design.md D1). Verify: a `TestRefusals` case `demo-v1.2.4` on a catalog declaring `1.2.3` fails with a non-usage error naming both; `TestConfigFromOutside` advances the catalog with its tag.
- [x] 1.2 `docs/contracts.md` "Commands" `build` row records the check.
- [x] 1.3 `task check` green, then commit `fix(build): refuse a release whose tag and catalog versions differ`.

## 2. Align the slashless catalog link message

- [x] 2.1 `internal/dialect`: classify a malformed docs-mode `/catalogs/` link by its second segment (design.md D2). Verify: `TestConformance` passes the new fixtures.
- [x] 2.2 Conformance fixtures `link-slashless-minor`, `link-slashless-edge`, `link-slashless-root`, each with `expect` and `shell.out`; the outputs match opmodel.dev's `lint-sources.sh` run over them. README lists them.
- [x] 2.3 `docs/contracts.md` C11 records the rule.
- [x] 2.4 `task check` green, then commit `fix(lint): report a slashless minor or edge catalog link as the site lint does`.
