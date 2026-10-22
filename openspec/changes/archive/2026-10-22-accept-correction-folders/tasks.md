## 1. Accept correction folders in the enhancements source

- [x] 1.1 `internal/extract/enhancements/openspec.go`: add the two anchored patterns of design.md (`reCorrection`, `reCorrectionDate`); make `classifyChange` return "not an entry" for a name that matches; add the correction form to the refusal message
- [x] 1.2 `internal/extract/enhancements/capabilities.go`: add the unlinked `correction` row in `capabilityHistory` and the page table; sort by date, id, then `correction` before `seed`
- [x] 1.3 Tests in `internal/extract/enhancements/extract_test.go`: a live and an archived correction build with no entry and no page; the capability page shows `correction` rows on two capabilities; `correction/`, `correction-`, `correction-Foo_Bar` and `2026-10-23-correction-` exit 2 with the new message; `0123-correction-x` is entry 0123; row order matches the spec scenario; update the refusal-message assertion
- [x] 1.4 `docs/contracts.md` C21: the openspec-reader paragraph and the capability page paragraph say so, including the row sort key; sync the `enhancements-bundle` spec from this change's delta
- [x] 1.5 `task check` green and `go test ./internal/extract/enhancements/...` passes; then commit `feat(extract): accept correction folders in the enhancements bundle`
