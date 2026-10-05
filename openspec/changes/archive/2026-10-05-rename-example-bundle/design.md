## Context

The controller repository is renamed `opm-controller` with no aliases. docs-kit uses it only as example data. Two snapshots are copies of real output: the controller's CRDs, samples and crdref page at `bf8d3c5`, and the cli's dump and cmdref pages at `ebf0479b`.

## Goals / Non-Goals

**Goals:** every example and fixture name the new project; the specs read the same.

**Non-Goals:** editing the snapshot contents (a hand edit would claim an output no release produced); changing any contract's shape.

## Decisions

- D1. Exact rename patterns only: `opm-operator`, `reference/operator-resources`, `reference/operator/`, the titles "Operator resources" and "Operator Reference", the test and variable names. No bare `operator` rule: shell operators in `helptext_test.go` and "operators" in the library fixture are other words.
- D2. The snapshot directory is renamed (fixture naming); its files are not. The parity test's scaffold label (`app.kubernetes.io/name: opm-operator`) and the crdref page path match the copy, so they stay until the copy is refreshed and carry a `TODO(cutover)` comment.
- D3. A scenario title that named the old product is retitled by REMOVED + ADDED under a new requirement name, because OpenSpec 1.12 refuses a MODIFIED that drops a scenario.

## Research & Decisions

### The render snapshots

**Context**: both snapshots name the old project inside their contents.
**Explored**: hand-renaming them; freezing them as history; regenerating them.
**Options considered**:
1. Hand-rename - claims an output no release produced.
2. Freeze - leaves the old name in the tree for good.
3. Regenerate from the first opm-controller and cli releases - true snapshots, no old name.
**Decision**: option 3, after those releases exist (decided in planning by the owner).
**Rationale**: the snapshots stay copies of real output. The generators they were compared with (crdref, cmdref) are retired, so the refresh also changes what the parity tests compare with; that is recorded with the refresh.

## Durable decisions

None.
