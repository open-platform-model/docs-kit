# Tasks: add-crd-extractor

Gate: `generalize-build-assembly` is merged on docs-kit `main`. One PR, titled `feat: add the crd extractor`.

## 1. The extractor and its model

- [x] 1.1 Spike: decode the operator's four CRDs at its newest tag with `apiextensionsv1` types and with a minimal local struct; record the dependency cost (`go mod graph | wc -l` before and after) and the choice in design.md. Verify: the finding is written in design.md D2.
- [x] 1.2 `schema/config.cue`: `#CRD` (design.md D1). Verify: `go test ./schema/...`.
- [x] 1.3 `internal/extract/crd` (D2, D3) ported from opm-operator `hack/crdref`, using `internal/doctext` and the citation policy. Verify: tests on fixture CRDs and samples under `internal/extract/crd/testdata/`, golden `crd.golden.json`, the kubebuilder file-name pick with a second file of the same kind ignored, a hidden fixture sample, stripped scaffold labels, a missing sample file.
- [x] 1.4 `task check` green, then commit `feat(extract): add the crd extractor`.

## 2. The renderer and parity

- [x] 2.1 `internal/render`: the crd page (D4), completable with heading `## <first kind>`. Verify: golden pages with and without an authored page.
- [x] 2.2 `TestOperatorCRDParity` (skipped unless `OPM_OPERATOR_CHECKOUT` names a checkout at a tag): compare with crdref's block between its markers. Verify: run against the newest operator tag; record tag and result in design.md.
- [x] 2.3 `task check` green, then commit `feat(render): render the operator resource reference`.

## 3. Contract and archive

- [x] 3.1 `docs/contracts.md` C18 (D1, D3, D4, parity record); C6's source table. Verify: schema text equals `schema/config.cue`.
- [x] 3.2 `openspec archive add-crd-extractor --yes`. Verify: `task openspec:check` green.
- [x] 3.3 `task check` green, then commit `docs(extract): document the crd contract`.
