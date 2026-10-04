## Context

Until now two linters implemented dialect 1: `opm-docs lint` and opmodel.dev's `site/scripts/lint-sources.sh`. The site ran the shell lint over `docs/site/` trees it mounted from git and `opm-docs lint` over every pulled bundle, so C11 bound them with one conformance fixture set (`internal/dialect/testdata/conformance/`, 36 cases), each with its expected `<file>:<line>` output in `shell.out`. A rule changed in docs-kit first, and opmodel.dev copied the fixture and changed its script in the PR that bumped its pinned `opm-docs`.

opmodel.dev#47 removes the git pipeline, the shell lint and `site/tests/lint/`. After it, every page reaches the site in a bundle and is linted by `opm-docs lint` on pull.

## Goals / Non-Goals

**Goals:** make C11's rule list the definition of dialect 1; keep the conformance fixtures, and the `task test` gate over them, as tests of the Go lint; remove every instruction that tells opmodel.dev to copy a fixture or change a shell lint.

**Non-Goals:** any change to a lint rule, message, exit code or mode; renaming `shell.out` or reshaping the fixture cases; `DESIGN.md`, which records the move from the shell lint as history and stays true.

## Decisions

### D1. Dialect 1 is C11's rule list

The `dialect-lint` requirement "Dialect 1 is the site's shell lint plus catalog links" is removed and added again as "Dialect 1 is the rule set C11 lists": `opm-docs lint` enforces the rules `docs/contracts.md` C11 lists. The shell lint is named once, in the past tense, as where the rules were first ported from (as of 2026-10-02). Its scenario "The shell lint's fixtures agree" becomes "The conformance fixtures pass": the lint reports exactly the files and lines each case's committed expected output records.

OpenSpec 1.12 refuses a MODIFIED requirement that drops or renames a main-spec scenario, so the rename is a REMOVED plus an ADDED.

### D2. The conformance set tests the Go lint

The requirement "The conformance fixture set binds both linters until phase 3" is removed. An added requirement, "The conformance fixture set tests the Go lint", keeps what still holds: docs-kit ships the set, every case has its expected output, `task test` fails on any disagreement, and a rule change lands with a fixture that records it. No other repository copies the set.

### D3. Two requirements lose their opmodel.dev clause

"Docs mode allows only major catalog links" says the slashless minor or `edge` message is the one "opmodel.dev's shell lint reports"; "The enhancements graph is a link target" says opmodel.dev copies its fixture with a shell lint change. Both are MODIFIED with every scenario kept by name; only the clauses that point at the shell lint change, to the past tense or out.

### D4. `shell.out` keeps its name

Each case's expected output stays in `shell.out` (decided in planning). The file is named for how the first cases were captured; the README says so. Renaming 36 files and `TestConformance`'s glob would change no behaviour and break the provenance the README records.

## Research & Decisions

### Where the shell lint is still named

**Context**: the planned scope (orchestration step 8) named only the binding requirement and C11's agreement paragraph; review found more.
**Explored**: `grep -rn "lint-sources\|shell lint\|tests/lint"` over the repository.
**Options considered**:
1. Remove only the binding requirement and paragraph - leaves C11's opening, the spec's first requirement and the README telling opmodel.dev to keep a lint it no longer has.
2. Every present-tense binding, in specs, contracts, README and comments; leave history (`DESIGN.md`, the README's capture notes, the archived changes) as written.
**Decision**: option 2.
**Rationale**: a contract or spec that names a deleted file as the authority is wrong the day the file goes; history that says where the rules came from stays true.

## Risks / Trade-offs

- Merged before opmodel.dev#47, the site's shell lint would run unbound to the Go lint. The PR stays a draft until #47 merges (G3.5).

## Durable decisions

- Dialect 1 is defined by C11's rule list, and the conformance set tests only the Go lint: `docs/contracts.md` C11 and the `dialect-lint` main spec.
- `shell.out` keeps its name: `internal/dialect/testdata/conformance/README.md`.
