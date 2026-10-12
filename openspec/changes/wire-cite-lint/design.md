# Design

## Context

The tool is `github.com/open-platform-model/enhancements/tools/enh`, command `cite-lint`. The reusable workflow is `cite-lint.yml` in the `.github` repo; it runs the tool at its own `enh-ref` default and lints Go and CUE files. A caller passes `paths` and `exclude` only.

## Research & Decisions

### Excludes

**Context**: The comment rule must hold in docs-kit, whose job is to read and strip citations. Its tests and extractors name old forms on purpose.
**Explored**: `cite-lint --root .` with the pinned tool: 121 findings with no exclude (66 of them in the fixture folders), 0 with the list below.
**Options considered**:
1. Exclude every `testdata/` folder - one line, but a new fixture with a decision number passes unseen.
2. Exclude the exact files and folders that hold findings - 11 lines, each with a reason; a new file is linted.
**Decision**: Option 2.
**Rationale**: The smallest set that gives 0 findings keeps the rule live everywhere else. Citations are not rewritten; the rewrite of these inputs would delete what the strip tests prove.

The excludes:
- `internal/doctext/doctext_test.go`, `internal/doctext/policy_test.go`: tests of the strip rules, old forms in strings.
- `internal/build/enhancements_test.go`, `internal/extract/enhancements/extract_test.go`, `internal/pull/sections_test.go`: tests of the reader of the old entry layout; the paths are synthetic entry paths.
- `internal/extract/enhancements/extract.go`, `internal/extract/enhancements/model.go`: the reader of the old entry layout; comments name its file names.
- `internal/render/testdata/library/`: a snapshot of library source with old comments.
- `internal/extract/cuecatalog/testdata/catalog/demo/`, `internal/extract/cuedefs/testdata/defs/src/`, `internal/extract/goapi/testdata/mod/lib/`: fixtures that keep one old-form citation, so the strip tests prove its removal.

### One place for the excludes

**Decision**: Two places, the task and the workflow input, each with a comment that names the other. A reusable-workflow input cannot read a Taskfile.
**Rationale**: Same as the other repos. The task and the CI run the same list, and the lint fails in CI if they drift in a way that matters.

### New workflow file

**Decision**: A new file `cite-lint.yml`, not a job in `ci.yml`. A call job of a reusable workflow is the whole job (`uses`, `with`, `permissions`), and `ci.yml` holds jobs with steps. A separate file keeps the call plain and the `.github` pin in one place.

### OpenSpec CLI pin

`openspec validate --all --strict` under 1.14.1 fails on existing specs (TECH-DEBT row of 2026-10-11, 17 specs with "requirement text very long"). The ruling of 2026-10-11 23:50 keeps the pin when strict fails. Not changed.

## Risks / Trade-offs

- The tool commit is set in the task and is the `enh-ref` default of the called workflow; nothing checks that they agree. Move both with the `.github` pin.
- A new citation in an excluded file is not caught. The list is short and named.

## Durable decisions

- The excludes of `cite-lint` and the reason of each: stay in the comments of `Taskfile.yml` and `cite-lint.yml`.
- `task cite-lint` is part of `task check`: lands in `AGENTS.md`.
