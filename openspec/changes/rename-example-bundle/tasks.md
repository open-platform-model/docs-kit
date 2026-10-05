# Tasks: rename-example-bundle

One PR. Every commit is `test` or `docs`: no docs-kit release.

## 1. Rename the example bundle in tests and fixtures

- [x] 1.1 Build, config, pull, serve, extractor and render tests, and the comments in `internal/config/config.go`, `internal/dialect/dialect.go`, `internal/extract/crd/{extract,model}.go`, `internal/pull/versions.go` and `schema/config.cue`: `opm-operator` becomes `opm-controller`, `reference/operator-resources.md` becomes `reference/controller-resources.md`, `reference/operator/` becomes `reference/controller/` (design.md D1). Verify: `go test ./internal/build/ ./internal/config/ ./internal/pull/ ./internal/serve/ ./internal/extract/... ./schema/` passes.
- [x] 1.2 Dialect conformance cases `alert-marker` and `index-md`: the repository directory moves to `opm-controller`, with the paths in `shell.out` and `expect`; `repoOrder` names `opm-controller`; the conformance README states the rename rule. Verify: `go test ./internal/dialect/ -run TestConformance` passes.
- [x] 1.3 `internal/render/testdata/operator` moves to `internal/render/testdata/controller`; `TestControllerCRDParity`, `TestControllerCRDSection`, `controllerOptions` and `OPM_CONTROLLER_CHECKOUT` replace the old names; the copy's scaffold label and page path stay with a `TODO(cutover)` comment (design.md D2). Verify: `go test ./internal/render/` passes.
- [x] 1.4 `task check` green, then commit `test: rename the example bundle to opm-controller`.

## 2. Rename the example bundle in the docs

- [ ] 2.1 `docs/contracts.md`, `DESIGN.md`, `docs/orchestration.md` and `openspec/config.yaml` name `opm-controller`, `reference/controller/` and `reference/controller-resources.md`. Verify: `git grep -nIi operator -- docs DESIGN.md openspec/config.yaml` prints nothing.
- [ ] 2.2 `task check` green, then commit `docs: rename the example bundle to opm-controller`.
