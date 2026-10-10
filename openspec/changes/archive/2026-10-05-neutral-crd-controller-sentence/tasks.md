# Tasks: neutral-crd-controller-sentence

One PR. The commit is `fix(render)`, a docs-kit patch release.

## 1. Name the reconciler in the Served by sentence

- [x] 1.1 `internal/render/templates/crd/entries.md.tmpl`: the `Served by` sentence becomes "The {{code .}} reconciler watches every {{$k.Kind}}." Verify: `grep -n "reconciler watches every" internal/render/templates/crd/entries.md.tmpl` prints the line.
- [x] 1.2 Goldens `internal/render/testdata/golden/{crd-alone,crd-completed}/reference/widgets.md` and `crd-section-alone/reference/widgets/widget.md`, and the `served by` expectations in `crd_test.go` and `crd_section_test.go`, take the new sentence. Verify: `go test ./internal/render/ -run 'CRD|Golden'` passes.
- [x] 1.3 `crd_parity_test.go`: `servedByWording` rewrites crdref's `Served by` sentences in the frozen block before comparing (design.md D2). Verify: `go test ./internal/render/ -run TestOperatorCRDParity` passes.
- [x] 1.4 `docs/contracts.md` C18 quotes the new sentence in the page layout and lists it under the parity paragraph's deliberate differences; the `reconciledBy` comment in C6 and `schema/config.cue` names the reconciler. Verify: `grep -n "operator's" docs/contracts.md` prints only the crdref quotation.
- [x] 1.5 `task check` green, then commit `fix(render): name the reconciler, not the product, in the crd served-by sentence`.
