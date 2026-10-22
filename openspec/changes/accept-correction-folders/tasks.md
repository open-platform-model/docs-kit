## 1. Accept correction folders in the enhancements source

- [ ] 1.1 `internal/extract/enhancements/openspec.go`: add `reCorrectionDate`; make `classifyChange` return "not an entry" for `correction-<slug>` (live) and `<date>-correction-<slug>` (archived); add the correction form to the refusal message
- [ ] 1.2 `internal/extract/enhancements/capabilities.go`: add the unlinked `correction` row in `capabilityHistory` and the page table
- [ ] 1.3 Tests in `internal/extract/enhancements/extract_test.go`: a live and an archived correction build with no entry and no page; the capability page shows `correction` rows on two capabilities; `correction/` still exits 2 with the new message; update the refusal-message assertion
- [ ] 1.4 `docs/contracts.md` C21: the openspec-reader paragraph and the capability page paragraph say so; sync the `enhancements-bundle` spec from this change's delta
- [ ] 1.5 `task check` green and `go test ./internal/extract/enhancements/...` passes; then commit `feat(extract): accept correction folders in the enhancements bundle`
