# Tasks: add-cue-definitions-extractor

Gate: `generalize-build-assembly` is merged on docs-kit `main`. One PR, titled `feat: add the cue-definitions extractor`.

## 1. The extractor and its model

- [x] 1.1 `schema/config.cue`: `#CueDefinitions` (design.md D1) in `#Source`. Verify: `go test ./schema/...` accepts core's planned config (D1) and refuses a page without `definitions`.
- [x] 1.2 `internal/extract/cuedefs`: parse, collect, uses/used-by, shape, rules, spec text (D2), placement checks with the backfill leniency (D3), `data/cue-definitions.json` (D4). Port from core `tools/refgen` (`defs.go`, `spec.go`, `text.go`) at the newest core tag, reusing `internal/doctext` and `internal/mdtext`. Verify: tests on a fixture package under `internal/extract/cuedefs/testdata/` and a golden `cue-definitions.golden.json`.
- [x] 1.3 `task check` green, then commit `feat(extract): add the cue-definitions extractor`.

## 2. The renderer and parity

- [x] 2.1 `internal/render`: the definitions renderer and templates (D5). Verify: golden pages for the fixture.
- [x] 2.2 `TestCoreDefinitionsParity` (skipped unless `OPM_CORE_CHECKOUT` names a core checkout at a tag): build with core's planned config and compare every page with refgen's committed page, marker comments removed. Verify: run it against core's newest tag and record the tag and result in design.md.
- [x] 2.3 `task check` green, then commit `feat(render): render CUE definitions reference pages`.

## 3. Contract and archive

- [x] 3.1 `docs/contracts.md` C17 (D1, D4, D5, and the parity record); C6's source table gains the kind. Verify: schema text equals `schema/config.cue`.
- [ ] 3.2 `openspec archive add-cue-definitions-extractor --yes`. Verify: `task openspec:check` green.
- [ ] 3.3 `task check` green, then commit `docs(extract): document the cue-definitions contract`.
