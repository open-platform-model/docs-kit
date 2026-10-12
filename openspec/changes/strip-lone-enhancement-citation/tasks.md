## 1. The strip rule

- [x] 1.1 `internal/doctext/policy.go`: `Strip.Clean` removes a lone `(enhancement NNNN)` outside code spans
- [x] 1.2 `internal/extract/cuecatalog`: call `Strip.CleanParagraphs` in place of the package function
- [x] 1.3 State the rule in `docs/contracts.md` "Citation policy"; regenerate the library golden page
- [x] 1.4 `task check` green, then commit feat(doctext): strip a lone enhancement citation

## 2. Tests

- [x] 2.1 Table tests in `internal/doctext/policy_test.go`: end, middle, start of a line, two in one sentence, plural list, code span, other words, year, fewer than four digits, link destination, the `link` policy, `CleanCode`
- [x] 2.2 `go test ./internal/doctext` passes, then commit test: lone enhancement citation under strip

## 3. Archive

- [x] 3.1 `openspec validate strip-lone-enhancement-citation --strict` passes with the pinned CLI 1.12.0
- [x] 3.2 Archive the change on the branch, then commit chore(openspec): archive strip-lone-enhancement-citation
