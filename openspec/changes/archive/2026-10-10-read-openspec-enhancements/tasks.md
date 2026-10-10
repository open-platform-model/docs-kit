# Tasks: read-openspec-enhancements

One PR, titled `feat(extract): read OpenSpec-layout enhancement entries`. Run `task check` before each commit task; `task openspec:check` runs openspec 1.14.1 locally, whose strict mode also fails on the long requirements of unrelated main specs, so check this change with `openspec validate read-openspec-enhancements --strict` as well.

## 1. Source config and the legacy reader

- [x] 1.1 `schema/config.cue` `#Enhancements` gains `changes`, `specs`, `legacy` (design.md, Source config); `internal/config` refuses overlapping paths. Verify: `go test ./internal/config/...` with defaults, a custom value, an absolute path, a path with `..` and overlapping paths.
- [x] 1.2 `internal/extract/enhancements` `Options` takes the three directories; the legacy reader reads `<legacy>/NNNN/` only and refuses a live `NNNN/`; `internal/build/enhancements.go` passes the config through. The fixture's live and template entries move under `archive/`. Verify: `go test ./internal/extract/enhancements/... ./internal/build/...` including a live-folder refusal.
- [x] 1.3 Gates observed at the branch head: `task fmt:check`, `task vet` and `task lint` passed, and `task test` failed only in `internal/serve` `TestRunSite`, a known environment failure that also fails on `main`; `task check` itself was never observed green. Then commit `feat(extract): give the enhancements source changes, specs and legacy directories`.

## 2. The openspec reader and data schema v2

- [x] 2.1 `model.go`: schema `v2`, `layout`, the two entry shapes, encode and decode (design.md, Data schema v2). Verify: the legacy golden gains `layout` and the new schema id only.
- [x] 2.2 `internal/gitsrc`: `Shallow`. `internal/build/enhancements.go` passes `LastMod` as `Options.LastCommit`. Verify: a `gitsrc` test on a shallow clone and a full one.
- [x] 2.3 The openspec reader: entry discovery, `config.yaml` checks, pages (`_index`, `proposal`, `design`, `requirements`, `tasks`), `progress`, `capabilities`, `aliases`, `updated` (design.md, Pages of an openspec entry, Redirects, `updated` from git). Verify: tests on the worked example's entry (golden data file and pages), each refusal in the spec, and `updated` on a real git repository.
- [x] 2.4 `internal/dialect/links.go` `reEnhOK` admits `proposal`, `requirements`, `tasks` and `capabilities/<cap>/`, and its message names the capability form (the entry pages link the new slugs, so the dialect moves with the reader). Verify: `go test ./internal/dialect/...` with a new conformance case of accepted and refused forms.
- [x] 2.5 A build test of the mixed fixture: manifest pages, sources, lastmod, and bundle-mode lint pass. Verify: `go test ./internal/build/...`.
- [x] 2.6 Gates observed at the branch head: `task fmt:check`, `task vet` and `task lint` passed, and `task test` failed only in `internal/serve` `TestRunSite`, a known environment failure that also fails on `main`; `task check` itself was never observed green. Then commit `feat(extract): read OpenSpec-layout enhancement entries`.

## 3. Capability pages

- [x] 3.1 `capabilities/_index.md` and `capabilities/<cap>.md` with the "Entries that changed it" table (design.md, Capability pages). Verify: tests with a seed row, a delivered entry row and a rejected entry that has no row; a build test passes bundle-mode lint.
- [x] 3.2 Gates observed at the branch head: `task fmt:check`, `task vet` and `task lint` passed, and `task test` failed only in `internal/serve` `TestRunSite`, a known environment failure that also fails on `main`; `task check` itself was never observed green. Then commit `feat(extract): publish a page for each capability`.

## 4. The link policy

- [x] 4.1 `internal/doctext` `Link` links `enhancement NNNN` and `enhancements NNNN and NNNN`; the `crd` extractor unwraps and its renderer links the same forms. Verify: `go test ./internal/doctext/... ./internal/extract/crd/... ./internal/render/... ./internal/build/...`; existing goldens unchanged.
- [x] 4.2 Gates observed at the branch head: `task fmt:check`, `task vet` and `task lint` passed, and `task test` failed only in `internal/serve` `TestRunSite`, a known environment failure that also fails on `main`; `task check` itself was never observed green. Then commit `feat(extract): link enhancement references under the link policy`.

## 5. Contract and archive

- [x] 5.1 `docs/contracts.md` C6 (`#Enhancements` equal to `schema/config.cue`), C21 (both readers, v2 file, pages, redirects, the full-depth requirement, consumers) and the Doc-comment rules (link policy); `README.md` and `AGENTS.md` (layout). Verify: the C6 copy equals the schema.
- [x] 5.2 `openspec archive read-openspec-enhancements --yes`, then `openspec validate enhancements-bundle --strict`. Verify: the main spec holds the new requirements.
- [x] 5.3 Gates observed at the branch head: `task fmt:check`, `task vet` and `task lint` passed, and `task test` failed only in `internal/serve` `TestRunSite`, a known environment failure that also fails on `main`; `task check` itself was never observed green. Then commit `docs(extract): document reading both enhancement layouts`.
