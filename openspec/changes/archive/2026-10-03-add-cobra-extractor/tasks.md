# Tasks: add-cobra-extractor

Gate: `generalize-build-assembly` is merged on docs-kit `main`. One PR, titled `feat: add the cobra extractor and cobradump`.

## 1. cobradump

- [x] 1.1 `cobradump/go.mod` (module `github.com/open-platform-model/docs-kit/cobradump`, `go 1.26.0`, cobra and pflag), `Write` and `WritePins` (design.md D1, D2). Verify: `go -C cobradump test ./...` with a fixture tree covering hidden, deprecated, help, completion, inherited and persistent flags, a home default, and a golden `cobradump/testdata/dump.golden.json` written twice identically.
- [x] 1.2 `Taskfile.yml` and `.github/workflows/ci.yml`: `test`, `vet` and `lint` cover `cobradump/`; the constitution's commit scopes gain `cobradump`. Verify: `task check` runs the nested module's tests.
- [x] 1.3 `task check` green, then commit `feat(cobradump): print a cobra command tree and pins as JSON`.

## 2. The extractor and renderer

- [x] 2.1 `schema/config.cue`: `#Cobra` (D3). Verify: `go test ./schema/...`.
- [x] 2.2 `internal/extract/cobra`: run the command (C14), validate, write `data/cobra.json` (D4); its test reads `cobradump/testdata/dump.golden.json` through a stub command. Verify: `go test ./internal/extract/cobra/...`.
- [x] 2.3 `internal/render`: the cobra pages and the Long/Example parser ported from cli `internal/cmdref/text.go` with its tests (D4). Verify: golden pages for the fixture tree.
- [x] 2.4 `TestCLICommandParity` (skipped unless `OPM_CLI_CHECKOUT` names a cli checkout with `hack/docskit-dump`; until the cli has one, run against a scratch branch of the cli adding it): compare with cmdref's pages for the same tree. Verify: run it; record the cli commit and the result in design.md.
- [x] 2.5 `task check` green, then commit `feat(extract): add the cobra extractor`.

## 3. Release plumbing

- [x] 3.1 `release-please-config.json`: the `cobradump` package, `"exclude-paths": ["cobradump"]` on the root package and `"separate-pull-requests": true` (D5); `.release-please-manifest.json` gets no `cobradump` entry, so `initial-version` 0.1.0 applies. `.github/workflows/release.yml`: goreleaser gated on the root package's release only. Verify: `actionlint` clean; `release-please` config validates against its schema (`npx release-please` dry run or the JSON schema).
- [x] 3.2 `task check` green, then commit `ci(release): release cobradump as its own component`.

## 4. Contract and archive

- [x] 4.1 `docs/contracts.md` C19 (D1 to D4, the parity record) and C12 (the component); `AGENTS.md` layout and releasing; `README.md` (how a CLI adopts the hook). Verify: links resolve.
- [x] 4.2 `openspec archive add-cobra-extractor --yes`. Verify: `task openspec:check` green.
- [x] 4.3 `task check` green, then commit `docs(extract): document the cobra contract`.
