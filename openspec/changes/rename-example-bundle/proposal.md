## Why

The Kubernetes controller repository `opm-operator` is renamed `opm-controller`: repository, image, docs bundle `docs/opm-controller` and the reference section `/docs/reference/controller/`. docs-kit names no project in its production code (a consuming site's config names each project and its signing repository), but its tests, fixtures, contracts, design record and specs use the controller's bundle as their running example. Left as they are, they describe a bundle that no longer exists and keep the old name alive in every grep.

## What Changes

- **Tests and fixtures**: the example project `opm-operator` becomes `opm-controller` in the build, config, pull, serve, extractor, dialect and render tests; the example paths `reference/operator-resources.md` and `reference/operator/` become `reference/controller-resources.md` and `reference/controller/`. The dialect conformance cases `alert-marker` and `index-md` move their repository directory to `opm-controller`, with the paths in `shell.out` and `expect`. The frozen controller copy moves from `internal/render/testdata/operator` to `internal/render/testdata/controller`; the tests over it become `TestControllerCRDParity` and `TestControllerCRDSection`, and the live-checkout variable becomes `OPM_CONTROLLER_CHECKOUT`.
- **Docs**: `docs/contracts.md` (the C1 projects table, C6, C16, C18 examples, parity and consumers), `DESIGN.md` and `docs/orchestration.md` name `opm-controller`. `openspec/config.yaml` lists `opm-controller` among the consuming repositories.
- **Specs**: example data in `crd-extractor`, `page-renderer`, `bundle-pull`, `docs-placement` and `opm-docs-cli`. Two requirements whose scenario titles named the old product are replaced under new names (REMOVED + ADDED) with the same behaviour.

Not in this change: the contents of the two render snapshots, `internal/render/testdata/controller/**` (CRDs, samples and crdref page copied at `bf8d3c5`) and `internal/render/testdata/cli/**` (the cli's dump and cmdref pages at `ebf0479b`). They are true copies of what those repositories produced, so they are regenerated from the first opm-controller and cli releases under the new names, not edited by hand. Until then the parity test keeps the copy's scaffold label and page path, marked `TODO(cutover)`.

SemVer class: none. Tests, fixtures, docs and specs only; the commits are `test` and `docs` and cut no docs-kit release. No contract changes shape: the renamed values are examples.

Consumers: none has anything to do. opmodel.dev's `bundles.cue` names the project and its signing repository itself; it moves to `opm-controller` in its own change.

Scope: two sections, then the archive step.

## Capabilities

### Modified Capabilities

- `crd-extractor`: example data; the section requirement is replaced under a new name.
- `page-renderer`: the completable-page requirement is replaced under a new name.
- `bundle-pull`, `docs-placement`, `opm-docs-cli`: example data.

## Impact

- Code: none outside `_test.go` files and comments (`internal/config/config.go`, `internal/dialect/dialect.go`, `internal/extract/crd/{extract,model}.go`, `internal/pull/versions.go`, `schema/config.cue`).
- Risk: none at runtime.
