## Why

`TestControllerCRDParity` and `TestCLICommandParity` compare docs-kit's render with what the retired generators wrote: a frozen crdref block from opm-controller at `bf8d3c5` and frozen cmdref pages from the cli at `ebf0479b`. Both generators are gone from their repositories, so those copies can never be refreshed: the next controller and cli releases carry renamed CRD labels, page paths and commands, and a frozen copy would keep the old names in docs-kit forever. The opt-in live tests (`OPM_CONTROLLER_CHECKOUT`, `OPM_CLI_CHECKOUT`) read the committed generator output, which no longer exists on either `main`.

What still exists is the released docs bundle each repository publishes. Comparing with it turns the parity tests into regression checks: docs-kit at HEAD, given the inputs a release was built from, must write the pages that release shipped.

## What Changes

- **Release snapshots.** `internal/render/testdata/controller` and `internal/render/testdata/cli` each hold one released bundle's pages under its owned section (`want/`), the release commit's `docs-kit.cue`, and the inputs those pages were built from: the CRDs, samples and authored section index for the controller, the `hack/docskit-dump` output for the cli. A `SOURCE` file records the bundle tag, digest, docs revision, the opm-docs version that built it and the commit.
- **`hack/refresh-snapshots.sh`** (`task snapshots:refresh`) takes a snapshot: it pulls and verifies the bundle with this checkout's `opm-docs pull`, exports the inputs from the release commit with `git archive`, runs the cli's dump twice and refuses differing output (C14), and refuses a docs revision whose fix commit touches an input.
- **Tests.** `TestControllerCRDParity` renders the snapshot with its own `docs-kit.cue` crd source, completes the section index with the authored one and compares page set and bodies with `want/`. `TestCLICommandParity` does the same with the cobra source. `TestControllerCRDSection` reads the same config and ties the page layout to the section pages. `TestCLICommandParityLive`, `OPM_CONTROLLER_CHECKOUT`, the crdref and cmdref marker handling and both `TODO(cutover)` comments go.
- **Seed.** The snapshots are taken now from the newest releases: opm-operator `1.0.0-beta.8` (docs revision 1, built by opm-docs 0.7.0) and cli `1.0.0-beta.10` (built by opm-docs 0.4.0). The controller bundle predates the reconciler wording of the served-by sentence, so the test rewrites that sentence in the released pages and fails once a refresh makes the rewrite match nothing.
- **Docs.** `docs/contracts.md` C18 and C19 parity paragraphs, `README.md` and the `AGENTS.md` layout.
- **Specs.** `crd-extractor`: the page requirement is restated without crdref parity, the section requirement's "Page layout unchanged" scenario ties the two layouts, the source requirement says "reconciler", and a new requirement states release-snapshot parity. `cobra-extractor`: the command reference requirement is restated without cmdref parity, and a new requirement states release-snapshot parity.

SemVer class: none (`test` and `docs` commits). No config key, data field, command, flag or exit code of `opm-docs` changes; the new script is a maintainer tool.

Consumers: none. No other repository reads the tests or the snapshots.

Not in this change:

- The refresh from the first opm-controller release and the next cli release. It is a data-only run of `task snapshots:refresh` after those releases publish their bundles, plus deleting the served-by rewrite.
- The library Go API snapshot (`internal/render/testdata/library`, library `v1.0.0-beta.1`). It has no generator to compare with and no rename-bearing name the residue check matches; it stays as it is.

## Capabilities

### Modified Capabilities

- `crd-extractor`: parity is with the newest released controller bundle, not with crdref.
- `cobra-extractor`: parity is with the newest released cli bundle, not with cmdref.

## Impact

- Code: tests under `internal/render/`, their test data, a new `hack/refresh-snapshots.sh` and a Taskfile task.
- Docs: `docs/contracts.md` C18, C19; `README.md`; `AGENTS.md`.
- Risk: a deliberate render change now fails the parity tests until the snapshot is refreshed from a release built with it, or a named rewrite records the difference. That is the point of the check.
