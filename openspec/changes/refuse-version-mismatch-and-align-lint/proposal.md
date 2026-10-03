## Why

Two defects found by consumers of docs-kit `v0.2.0`.

- docs-kit#8, found reviewing catalog_opm#121: `opm-docs build --release opm-v4.6.0` on a tree whose catalog declares `metadata.version: "4.5.0"` builds. The bundle is published under segment `4.6` while its landing and `data/catalog.json` say `4.5.0`. In a real release the identity advance on the release PR keeps the two equal, so this is defence in depth: the version a bundle is tagged with must be the version its source declares (Principle III, a generated entry states only what its source proves).
- docs-kit#10, found by opmodel.dev#26's lint parity test: both linters refuse a slashless minor or `edge` catalog link (`/catalogs/opm/4.4/traits/backup`, `/catalogs/opm/edge`) on the same line, but the site's shell lint says "docs pages link catalogs through /catalogs/opm/<MAJOR>/" and `opm-docs lint` says "write ... with a trailing slash". C11 requires one message, and the conformance set has no case for it.

## What Changes

- **`build` in release and revision mode** (and so `revise`) refuses, exit 2, when the release tag's version (prefix removed) differs from the `metadata.version` the `cue-catalog` extractor reads, naming both values. Edge builds and bundles without a `cue-catalog` source are unchanged.
- **Docs-mode lint**: a slashless `/catalogs/` link whose second segment is a minor (`4.4`) or `edge` reports the shell lint's segment message (`docs pages link catalogs through /catalogs/opm/4/`, `.../<MAJOR>/`); every other malformed `/catalogs/` link, the bare slashless root `/catalogs/opm` included, keeps the trailing-slash message. Bundle mode is unchanged.
- **Conformance fixtures** `link-slashless-minor`, `link-slashless-edge` and `link-slashless-root` with the shared expected output.
- `docs/contracts.md` C11 and "Commands" record both rules.

SemVer class: PATCH (`0.2.1`). No schema, tag or workflow interface changes. A release whose tag and catalog version disagree no longer builds, which is the defect being fixed.

Consumers: opmodel.dev copies the three fixtures into `site/tests/lint/` and records them in `link-catalogs/SOURCE` in the PR that bumps its pinned `opm-docs` to `v0.2.1`; its shell lint already prints these messages. catalog_opm: nothing, beyond moving `.opm-docs-version` when it wants the check.

Scope: two implementation sections, then the archive step.

## Capabilities

### Modified Capabilities

- `opm-docs-cli`: release builds check the catalog's declared version.
- `dialect-lint`: docs-mode message for slashless minor and edge catalog links.

## Impact

- Code: `internal/build/build.go`, `internal/dialect/links.go`, fixtures under `internal/dialect/testdata/conformance/`.
- Risk: none for a release cut by release-please; a hand-made tag on the wrong commit now fails instead of publishing a mislabelled bundle.
