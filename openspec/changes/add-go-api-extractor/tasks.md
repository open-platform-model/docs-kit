# Tasks: add-go-api-extractor

Gate: `generalize-build-assembly` is merged on docs-kit `main`. One PR, titled `feat: add the go-api extractor`.

## 1. The extractor and its model

- [ ] 1.1 `schema/config.cue`: `#GoAPI` (design.md D1). Verify: `go test ./schema/...`.
- [ ] 1.2 `internal/extract/goapi` (D2, D3): `go/build` context for `linux/amd64` file selection, `go/parser` with comments, `go/doc` (`doc.NewFromFiles`), `go/doc/comment` to Markdown, gofmt-printed declarations, citations per policy. Verify: tests on a fixture module under `internal/extract/goapi/testdata/` (a constructor, a method, an undocumented symbol, a build-tagged file, an internal package), golden `go-api.golden.json`.
- [ ] 1.3 `task check` green, then commit `feat(extract): add the go-api extractor`.

## 2. The renderer

- [ ] 2.1 `internal/render`: index and package pages (D4), anchors, doc-link resolution, heading shift. Verify: golden pages; the bundle passes docs bundle-mode lint.
- [ ] 2.2 A trial build of the library at its newest tag with the planned config (D1), outside the test suite. Verify: lint passes; record the page count, the undocumented symbols and any page-dialect problem in design.md (the library sibling fixes its doc comments, not docs-kit).
- [ ] 2.3 `task check` green, then commit `feat(render): render Go API reference pages`.

## 3. Contract and archive

- [ ] 3.1 `docs/contracts.md` C20 (D1, D3, D4); C6's source table. Verify: schema text equals `schema/config.cue`.
- [ ] 3.2 `openspec archive add-go-api-extractor --yes`. Verify: `task openspec:check` green.
- [ ] 3.3 `task check` green, then commit `docs(extract): document the go-api contract`.
