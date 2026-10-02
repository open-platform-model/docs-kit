## Why

Four repositories each carry their own reference generator (about 5,800 lines of Go solving the same problems four times), commit its output to git, and need a regenerate step for every doc-comment change; catalog_opm's release workflow even regenerates pages on the release PR. The site, opmodel.dev, reads every source repository through git and cannot show two versions of one repository's pages side by side, which a catalog reference with one entry per minor needs. `DESIGN.md` moves both jobs to the producer: each repository builds, lints and publishes its docs as a signed OCI bundle, and the site only chooses versions and assembles them.

This change is phase 1 of `DESIGN.md` on the docs-kit side: the tool and the reusable workflow, enough for catalog_opm to publish the opm catalog's reference and for opmodel.dev to show it in a Catalogs tab. It also fixes, in `design.md`, every contract the two sibling changes build against (`orchestration.md`).

DESIGN decision coverage:

| Decision | Here |
|---|---|
| 1, 2, 3, 5, 7, 9 | in full |
| 4 | phase 1's part (the `cue-catalog` extractor and a minimal `markdown` source); phases 2 and 3 are later changes |
| 6 | the contracts only (`revision` field and annotation, `source.patches`, tag rules); the `revise` command and the workflow's `revision` mode are the follow-up change `add-docs-revisions` |
| 8 | in full, including the backfill of a release cut before docs-kit (`release` mode reads `docs-kit.cue` from `main` when the tag has none) |
| 10 | phase 2 |
| 11 | out of scope |
| 12 | phase 1b; phase 1 extracts the structured spec it needs and reserves the history file |

Phase-1 done criterion, as this change and its siblings deliver it: the site shows opm 4.4 and edge in the Catalogs tab from pulled, verified bundles. The second half of DESIGN.md's criterion, a docs revision of 4.4.x reaching the site without a catalog release, moves to `add-docs-revisions`.

## What Changes

- **`opm-docs`**, one Go binary with `build`, `lint`, `check`, `push`, `promote`, `pull` and `version` (`design.md`, "Commands"). `promote`, and the split between `push` and `promote`, are decided in planning (design R4).
- **Extractors and renderer.** The `cue-catalog` extractor, ported from catalog_opm's `tools/refgen` with its doc-comment, citation, escaping, served-by, mark and enforcement rules, plus the structured spec fields phase 1b needs; a minimal `markdown` source for a bundle's authored landing; one embedded template set.
- **The page-dialect lint** in Go, dialect 1: the site's shell lint rule for rule, plus the `/catalogs/` link forms.
- **Contracts** (`design.md` C1 to C12), each with a CUE schema embedded in the tool where it is a file: OCI paths, media types and annotations, `manifest.json`, the tag scheme, the workflow interface, `docs-kit.cue`, the pull config, the unpack layout and lock, URLs and link forms, the signing identity, the doc model and the dialect.
- **Publishing.** `.github/workflows/publish.yml`, a reusable workflow with `check`, `edge` and `release` modes, cosign keyless signing, and the tool version tied to the workflow ref.
- **docs-kit's own CI and release**: a PR and main CI running `task check`, release-please, and a draft-first goreleaser release that attaches binaries for linux/amd64, linux/arm64, darwin/arm64 and darwin/amd64 plus `checksums.txt` before publishing. Consumers never `go run` the tool: they pin a release in `.opm-docs-version` (or, for `publish.yml`, its version literal) and verify the checksum (C12).

Out of scope: the raw Kubernetes catalog (`k8s/`), which the owner is removing from catalog_opm in a separate session; docs revisions (`add-docs-revisions`).

Scope: five implementation sections, then the archive step (Principle VII).

SemVer class: MINOR. docs-kit has no release yet; the first release PR after this change merges is `0.1.0`. The section commits are `feat` (or `docs`/`ci` where tasks.md says so), so release-please opens one release PR covering them.

## Capabilities

### New Capabilities

- `bundle-format`: the bundle tree, `manifest.json`, media types and annotations, deterministic packing and guarded unpacking.
- `tag-scheme`: full, release, minor, major and edge tags; ordering; revision numbers; when tags move.
- `opm-docs-cli`: the command set, `docs-kit.cue`, exit codes and messages.
- `cue-catalog-extractor`: the doc model from an evaluated CUE catalog, structured spec fields, doc-comment cleanup.
- `page-renderer`: page paths, links, marks and the member, index and landing pages.
- `dialect-lint`: dialect 1 and its docs and bundle modes.
- `publish-workflow`: the reusable workflow's interface, modes, signing and tool pinning.
- `bundle-pull`: the pull config, tag resolution, signature verification, unpack layout, cache and lock.
- `tool-release`: the release assets of every docs-kit release, and how consumers pin and verify them.

### Modified Capabilities

None. The repository has no specs yet.

## Impact

- Code: everything under `cmd/opm-docs/`, `internal/`, `schema/`; `.github/workflows/{publish,ci,release,spike}.yml`; `release-please-config.json`, `.release-please-manifest.json`, `.goreleaser.yml`; `Taskfile.yml`, `AGENTS.md`, `README.md`, `docs/contracts.md`.
- Consumers: catalog_opm (change `publish-docs-bundle`) and opmodel.dev (change `add-catalogs-tab`) build against the contracts; `orchestration.md` lists what each must do and in which order. cli carries one link fix (`docs/site/reference/registry-namespaces.md` links the contract page that moves). core, opm-operator, library and opm are untouched until phase 2.
- Registry: a new public package `ghcr.io/open-platform-model/docs/catalog-opm` (the owner makes it public), and a throwaway `docs/spike` for section 1.
- Workspace: the root `AGENTS.md` registry rule ("a version-named OCI tag is never overwritten") conflicts with the bundles' moving tags (`4.4.5`, `4.4`, `4`, `edge`); only full tags (`4.4.5.0`) are immutable. An owner item in `orchestration.md`, handled by a workspace PR, not here.
- Risk: GHCR's handling of OCI 1.1 artifacts and of cosign v3 signatures is unverified; section 1 is a gate, and its fallback (design C2, outcome B) changes no sibling contract.
- Risk: callers pin `publish.yml` by tag, against the org's SHA-pinning convention; a known conflict left to review (design C5).
- Risk: the site build now depends on GHCR and on the Sigstore trusted root; `pull --frozen --offline` with the cache is the mitigation.
