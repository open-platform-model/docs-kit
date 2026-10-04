# Tasks: add-crd-section-layout

One PR, titled `feat(render): render a crd source as a section of kind pages`.

## 1. The section layout

- [x] 1.1 `schema/config.cue`: `#CRD` gains `section`, `page` becomes optional, `matchN` makes them exclusive (design.md D1); `internal/config` refuses both or neither with a plain message. Verify: `go test ./internal/config/... ./schema/...` with cases for both, neither, page alone and section alone.
- [x] 1.2 `internal/extract/crd`: `layout`, `kinds[].page`, per-kind reads, the lower-case collision refusal (D2). Verify: the golden `crd.golden.json` gains `"layout": "page"` and `"page": null` only; a section-layout extraction test; a collision test.
- [x] 1.3 `internal/build/crd.go`: sources and inputs per page in section layout (D3). Verify: a build test of a section-layout bundle checks the page paths, `source` and `lastmod` in `manifest.json`.
- [x] 1.4 `internal/render`: the `crd-kind` template shared by both layouts (D5), the section index and kind pages (D3). Verify: `TestOperatorCRDParity` and the page-layout goldens unchanged; new goldens under `testdata/golden/crd-section` (standalone and completed index, kind pages) pass bundle-mode lint.
- [x] 1.5 `task check` green, then commit `feat(render): render a crd source as a section of kind pages`.

## 2. Contract and archive

- [x] 2.1 `docs/contracts.md` C18 (both layouts, data fields, consumers) and C6 (`#CRD` copy equal to `schema/config.cue`, the source table row); `README.md`'s source table. Verify: the C6 `#CRD` text equals `schema/config.cue`'s.
- [x] 2.2 `openspec archive add-crd-section-layout --yes`. Verify: `task openspec:check` green.
- [x] 2.3 `task check` green, then commit `docs(render): document the crd section layout`.
