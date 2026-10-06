## Context

docs-kit replaced four reference generators. Its parity tests prove the replacement: for the controller, crdref's block frozen at `bf8d3c5`; for the cli, cmdref's pages frozen at `ebf0479b`. Both generators are deleted upstream, and the controller and cli are being renamed (opm-operator to opm-controller), so a frozen copy can only grow stale. Every controller and cli release since the switch publishes a signed docs bundle built by docs-kit's `publish.yml`, which is a better reference: it is what readers actually got.

## Goals / Non-Goals

**Goals:** parity tests that compare with released bundles; a snapshot refresh that is one command and needs no hand edits, so the cutover step after the first opm-controller release and the next cli release is data only (plus deleting one rewrite); every snapshot traceable to a bundle digest and a commit.

**Non-Goals:** refreshing from releases that do not exist yet; the library Go API snapshot (library `v1.0.0-beta.1` stays as it is); a live test against a local checkout; changing what `opm-docs` writes.

## Decisions

- D1. A snapshot holds the release commit's `docs-kit.cue` and the test reads its source config, so section, title, labels and order come from the release, not from test code. Renaming the section or the scaffold label upstream is then a refresh, not a code change.
- D2. The expected pages are the bundle's pages under its one owned section, copied after `opm-docs pull` verified the signature and signer. The test compares the page set and every body byte for byte.
- D3. The inputs come from the release commit the bundle's manifest names, exported with `git archive`. A docs revision builds that commit with fix commits applied; the script refuses a revision whose fix touches an input, because the snapshot would then not hold what the bundle was built from.
- D4. The cli's input is the dump its `hack/docskit-dump` prints at the release commit, run twice in an exported tree with identical output required (C14), as the publish build runs it.
- D5. A deliberate render change that postdates the snapshot's bundle is a named rewrite of the released pages in the test. `compareWithRelease` applies every named rewrite and fails the test on one that matches nothing, so the refresh that makes a rewrite obsolete also deletes it, and deleting it is removing its definition and its name from the call. Today there is one: the served-by sentence, since opm-operator `1.0.0-beta.8` was built by opm-docs 0.7.0.
- D6. The script lives in `hack/` with a `snapshots:refresh` task. It needs network (GHCR, the Go proxy) and is never part of `task check`; the tests stay offline.

## Research & Decisions

### What the parity tests compare with

**Context**: crdref and cmdref are retired upstream; the frozen copies name the old product.
**Options considered**:
1. Keep the frozen generator copies as dated residue - they stay true but can never follow a rename, and the old names stay in docs-kit.
2. Delete the parity tests - the extractors keep unit tests and goldens, but nothing checks a real tree end to end.
3. Compare with released bundles - real trees, refreshable, traceable to a digest.
**Decision**: option 3, which the rename to opm-controller (enhancement 0032, "The operator becomes the controller") needs, since a frozen copy cannot follow it.
**Rationale**: it keeps an end-to-end check on real input and lets the snapshot follow the products.

### Seeding before the renamed releases exist

**Context**: the first opm-controller release and the next cli release publish only during the rename cutover.
**Options considered**:
1. Leave the change open until those releases exist.
2. Seed now from the newest releases (opm-operator `1.0.0-beta.8`, cli `1.0.0-beta.10`) and refresh at the cutover.
**Decision**: option 2.
**Rationale**: the procedure is exercised end to end now, the change can be archived, and the cutover step runs a tested command. The old names stay in the seeded snapshot until that refresh replaces them.

## Risks / Trade-offs

- A docs-kit render change fails the parity tests until a release built with it exists. Mitigation: a named rewrite (D5), deleted by the refresh.
- The refresh runs the cli's release code (`go run ./hack/docskit-dump`). It is the cli's own published release commit, run as its publish build ran it.

## Durable decisions

- The parity paragraphs of `docs/contracts.md` C18 and C19 state what the tests compare with and how to refresh (done in this change).
