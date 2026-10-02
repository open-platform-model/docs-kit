## Why

Four repositories each carry their own reference generator (about 5,800 lines of Go solving the same problems four times), commit its output to git, and need a regenerate step for every doc-comment change; catalog_opm's release workflow even regenerates pages on the release PR. The site, opmodel.dev, reads every source repository through git and cannot show two versions of one repository's pages side by side, which a catalog reference with one entry per minor needs. `DESIGN.md` moves both jobs to the producer: each repository builds, lints and publishes its docs as a signed OCI bundle, and the site only chooses versions and assembles them.

This change is phase 1 of `DESIGN.md` on the docs-kit side: the tool and the reusable workflow, enough for catalog_opm to publish the opm catalog's reference and for opmodel.dev to show it in a Catalogs tab (DESIGN decisions 1 to 9 and 12). It also fixes, in `design.md`, every contract the two sibling changes build against (`orchestration.md`).

## What Changes

- **`opm-docs`**, one Go binary with `build`, `lint`, `check`, `push`, `promote`, `pull`, `revise` and `version` (`design.md`, "Commands"). `promote` and the split between `push` and `promote` are decided in planning (design R4); `revise` is in phase 1 because DESIGN.md's phase-1 done criterion needs a docs revision to reach the site.
- **Extractors and renderer.** The `cue-catalog` extractor, ported from catalog_opm's `tools/refgen` with its doc-comment, citation, escaping, served-by, mark and enforcement rules, plus the structured spec fields phase 1b needs; a minimal `markdown` source for a bundle's authored landing; one embedded template set.
- **The page-dialect lint** in Go, dialect 1: the site's shell lint rule for rule, plus the `/catalogs/` link forms.
- **Contracts** (`design.md` C1 to C11), each with a CUE schema embedded in the tool where it is a file: OCI paths, media types and annotations, `manifest.json`, the tag scheme, the workflow interface, `docs-kit.cue`, the pull config, the unpack layout and lock, URLs and link forms, the signing identity, the doc model and the dialect.
- **Publishing.** `.github/workflows/publish.yml`, a reusable workflow with `check`, `edge`, `release` and `revision` modes, cosign keyless signing, and the tool version tied to the workflow ref.
- **docs-kit's own CI and release**: a PR and main CI running `task check`, release-please, and a draft-first release that attaches the binaries and checksums before publishing.

Scope: six sections, one more than the constitution's "about five" (Principle VII). The owner asked for one change; every section ends green on its own. If an implementer's Execution Gate needs a split, section 5 (`revise`) becomes its own change, `add-docs-revisions`, and the phase-1 done criterion waits for it.

SemVer class: MINOR. docs-kit has no release yet; the first release PR after this change merges is `0.1.0`. Each section's commit is `feat` (or `docs`/`ci` where it says so), so release-please opens one release PR covering them.

## Capabilities

### New Capabilities

- `bundle-format`: the bundle tree, `manifest.json`, media types and annotations, deterministic packing and guarded unpacking.
- `tag-scheme`: full, release, minor, major and edge tags; ordering; revision numbers; when tags move.
- `opm-docs-cli`: the command set, `docs-kit.cue`, exit codes and messages.
- `cue-catalog-extractor`: the doc model from an evaluated CUE catalog, structured spec fields, doc-comment cleanup.
- `page-renderer`: page paths, links, marks and the member, index, landing and table pages.
- `dialect-lint`: dialect 1 and its docs and bundle modes.
- `publish-workflow`: the reusable workflow's interface, modes, signing and tool pinning.
- `bundle-pull`: the pull config, tag resolution, signature verification, unpack layout, cache and lock.
- `docs-revision`: the documentation-only check and how a revision is built.

### Modified Capabilities

None. The repository has no specs yet.

## Impact

- Code: everything under `cmd/opm-docs/`, `internal/`, `schema/`; `.github/workflows/{publish,ci,release,spike}.yml`; `release-please-config.json`, `.release-please-manifest.json`; `Taskfile.yml`, `AGENTS.md`, `README.md`, `docs/contracts.md`.
- Consumers: catalog_opm (change `publish-docs-bundle`) and opmodel.dev (change `add-catalogs-tab`) build against the contracts; `orchestration.md` lists what each must do and in which order. cli carries one link fix (`docs/site/reference/registry-namespaces.md` links the contract page that moves). core, opm-operator, library and opm are untouched until phase 2.
- Registry: new public packages `ghcr.io/open-platform-model/docs/catalog-opm` and `docs/catalog-k8s` (owner makes them public), and a throwaway `docs/spike` for section 1.
- Workspace: the root `AGENTS.md` registry rule ("a version-named OCI tag is never overwritten") needs one line naming the docs bundles' moving tags (`4.4.5`, `4.4`, `4`, `edge`) as mutable by design; only full tags (`4.4.5.0`) are immutable. An owner follow-up, not an edit here.
- Risk: GHCR's handling of OCI 1.1 artifacts and of cosign v3 signatures is unverified; section 1 is a gate, and its fallback (design C2, outcome B) changes no sibling contract.
- Risk: the site build now depends on GHCR and on the Sigstore trusted root; `pull --frozen --offline` with the cache is the mitigation.
