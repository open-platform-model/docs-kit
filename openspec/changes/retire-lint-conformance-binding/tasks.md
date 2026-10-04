# Tasks: retire-lint-conformance-binding

One PR, after opmodel.dev#47 merges (G3.5). Every commit is `docs`: no docs-kit release.

## 1. Rebind the conformance set to the Go lint

- [x] 1.1 `docs/contracts.md` C11: the opening paragraph defines dialect 1 by its rule list and names `lint-sources.sh` only as the rules' origin; "Agreement with the site's shell lint until phase 3" becomes "Conformance fixtures" (docs-kit owns the set; `task test` gates it; a rule change lands with its case); the bundle-mode and docs-mode bullets and the closing format line name the shell lint only in the past tense (design.md D1, D2). Verify: `grep -n "keeps its shell lint\|until phase 3\|lint-sources.sh" docs/contracts.md` prints only the origin sentence.
- [x] 1.2 `internal/dialect/testdata/conformance/README.md`: the opening paragraph says the fixtures test `opm-docs lint`; the re-sync rule becomes a rule for docs-kit alone; the capture notes stay as provenance, and `shell.out` keeps its name (design.md D4). Verify: `grep -n "opmodel.dev copies\|teaches\|re-sync" internal/dialect/testdata/conformance/README.md` prints nothing.
- [x] 1.3 `internal/dialect/dialect.go` package comment and `dialect_test.go` `TestConformance` comment: dialect 1 is C11's rule list, ported from the retired shell lint; the expected output is the case's own. Verify: `go test -race ./internal/dialect/` passes.
- [x] 1.4 `task check` green, then commit `docs(lint): retire the conformance binding to the site's shell lint`.

## 2. Close phase 3 in the orchestration plan

- [ ] 2.1 `docs/orchestration.md`: G3.5 and step 8 marked done (opmodel.dev#47 merged), and phase 3 marked complete. Verify: the gate row, step 8 and the phase 3 section each carry the done line.
- [ ] 2.2 `task check` green, then commit `docs: mark phase 3 of the orchestration plan done`.
