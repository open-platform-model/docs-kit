## Context

The crd renderer writes a `Served by` part per kind when the source's `reconciledBy` names it. The text came from opm-operator's retired `hack/crdref`, which wrote "The operator's `<name>` controller watches every <Kind>." docs-kit froze that block in `internal/render/testdata/operator` and checks parity with it.

## Goals / Non-Goals

**Goals:** a sentence that names no product and uses the reconciler vocabulary; parity with crdref kept for every other byte.

**Non-Goals:** renaming the `reconciledBy` key or the `data/crd.json` field (both are contracts other repositories write and read); refreshing the frozen crdref copy (the rename change `rename-example-bundle` handles fixtures).

## Decisions

- D1. The sentence is "The `<name>` reconciler watches every <Kind>." The `<name>` is the configured `reconciledBy` value, as before. The page renders the same in the page and the section layout, because both use one template.
- D2. The frozen crdref block stays byte for byte as crdref wrote it. `TestOperatorCRDParity` rewrites only the `Served by` sentence (one anchored regular expression) before it compares, so a real drift anywhere else still fails the test.

## Research & Decisions

### Where the sentence departs from crdref

**Context**: the parity test compares with a frozen copy of crdref's output.
**Explored**: editing the frozen copy; rewriting it in the test; dropping the parity test.
**Options considered**:
1. Edit the frozen copy - the copy would claim an output crdref never produced.
2. Rewrite the sentence in the test - keeps the copy true and the departure visible in one function.
3. Drop the test - loses parity for everything else.
**Decision**: option 2.
**Rationale**: the departure is deliberate and documented in C18; every other line still matches crdref.

## Durable decisions

- The `Served by` wording and the deliberate difference from crdref land in `docs/contracts.md` C18 (done in this change).
