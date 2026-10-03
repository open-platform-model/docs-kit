# Tasks: generalize-build-assembly

Gate: none (docs-kit `main`). One PR, titled `feat: build docs-placed bundles from registered sources`. It gates the four extractor changes and `pull-docs-placement` (`docs/orchestration.md`).

## 1. The registry, with catalog output unchanged

- [ ] 1.1 `internal/build`: the `Extractor` registry and `Input`/`Data` (design.md D1); `cue-catalog` and `markdown` behind it. Verify: `go test ./internal/build/... ./internal/render/... ./internal/extract/...` with the existing goldens and `catalog.golden.json` unchanged.
- [ ] 1.2 `internal/render`: the `Renderer` registry keyed by data schema, `Target.Kind`, the catalog renderer registered. Verify: `TestCatalogOPMParity` (when its checkout is present) and the render goldens unchanged.
- [ ] 1.3 A test that builds the catalog fixture and packs it, comparing the digest with one recorded before this section. Verify: the test passes.
- [ ] 1.4 `task check` green, then commit `refactor(build): register extractors and renderers by kind and schema`.

## 2. Docs placement, completable pages, include and exclude

- [ ] 2.1 `schema/config.cue` and `schema/manifest.cue`: `owns` and `#Owned` (D2), `include` and `exclude` (D4); `#Source` stays the union of registered kinds. Verify: `go test ./schema/...` with a docs placement, nested `owns` refused by build, a bad glob form.
- [ ] 2.2 `internal/build`: docs placement rules (D2), completable pages generalized from the catalog landing (D3). Verify: a test renderer writing a completable page with and without an authored page; the heading collision; a generated page outside `owns`.
- [ ] 2.3 `internal/dialect` and `build.Lint`: docs bundle mode (D2). Verify: fixture bundle under `internal/dialect/testdata/docs-bundle/` with a link into `owns` that misses, one into another bundle's path, and a minor catalog link.
- [ ] 2.4 `internal/extract/markdown`: `include` and `exclude` with directory patterns (D4), and alias pinning only for tab bundles. Verify: tests for a match, an exclude winning over an include, a directory pattern, no match (error), no match in a backfill (ignored).
- [ ] 2.5 `task check` green, then commit `feat(build): build and lint docs-placed bundles`.

## 3. Citations, repository commands and pins

- [ ] 3.1 `internal/doctext`: the `link` citation policy (D5). Verify: table tests for each citation form in prose, in a code span and in a spec comment, in both modes.
- [ ] 3.2 `internal/command`: the runner (D6) with timeout, output cap, one-document check, and the `check` double run. Verify: tests with a helper program (`go run ./internal/command/testdata/echo`) for success, failure, trailing output, timeout (short test timeout) and nondeterminism.
- [ ] 3.3 `pins` (D7): config, build, manifest. Verify: tests for exact keys, a missing and an extra project, a `v`-prefixed version refused.
- [ ] 3.4 `task check` green, then commit `feat(build): run repository commands and record pins`.

## 4. The workflow

- [ ] 4.1 `.github/workflows/publish.yml`: the `build` and `publish` jobs (D8), the artifact hand-off, the `setup-go` input; the concurrency groups at the workflow level (D8). Verify: `actionlint` clean; the `check` mode needs only `contents: read` and `packages: read` (read the job's `permissions`).
- [ ] 4.2 `task check` green, then commit `feat(workflow): build in a job without the signing token`.

## 5. Contracts, docs and archive

- [ ] 5.1 `docs/contracts.md`: C1 (D9 table and naming rule), C3 (`owns`, `pins`), C5 (job split, workflow-level concurrency, `setup-go`), C6 (registry, `include`, `exclude`, `citations`, `pins`), C8 (docs URL form), new C14 (D6) and C15 (D2, D3, D7), C12 (D10, the site bumps first). `README.md`: source kinds, adding an extractor. `AGENTS.md`: layout tree. Verify: the schema text in C3 and C6 equals `schema/*.cue`; links resolve.
- [ ] 5.2 `openspec archive generalize-build-assembly --yes`. Verify: `task openspec:check` green; every durable decision landed.
- [ ] 5.3 `task check` green, then commit `docs(build): document docs placement, commands and pins`.
