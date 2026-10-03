# Orchestration: docs-kit phases 1b, 2 and 3

This file sequences the work that finishes `DESIGN.md` across docs-kit, the five product repositories, opm, enhancements and opmodel.dev. It is enough to plan each sibling change without reading docs-kit's code: the docs-kit changes are under `openspec/changes/` (their `design.md` files show every contract concretely), and each sibling's list below is what that sibling must do. When this plan is done, delete this file; until then, a change that alters the sequence updates it.

Written 2026-10-03 against docs-kit `v0.2.1`. Owner decisions of that day are DESIGN decisions 17 to 21 (20 and 21 confirm two choices this plan proposed); choices this plan made are marked "decided in planning" in the change that owns them and listed at the end.

## Contract numbers

Reserved now, so the changes that run in parallel do not collide. Each lands in `docs/contracts.md` with its change.

| Contract | Title | Change |
|---|---|---|
| C13 | Version history (`history.json`) | `add-version-history` |
| C14 | Repository commands | `generalize-build-assembly` |
| C15 | Docs placement, `owns`, completable pages, pins | `generalize-build-assembly` (extended by `add-authored-docs`) |
| C16 | Site versions (pull config `docs`/`versions`, `_versions/` layout, lock `docs`) | `pull-docs-placement` |
| C17 | `cue-definitions` | `add-cue-definitions-extractor` |
| C18 | `crd` | `add-crd-extractor` |
| C19 | `cobra` and `cobradump` | `add-cobra-extractor` |
| C20 | `go-api` | `add-go-api-extractor` |
| C21 | Sections and the enhancements bundle | `add-enhancements-bundle` |

Existing contracts these changes amend: C1 (project table, `generalize-build-assembly`), C3, C5, C6, C7, C8, C11, C12.

## Changes

| Repo | Change | Schema | Delivery | Gate to start |
|---|---|---|---|---|
| docs-kit | `add-version-history` | spec-driven | one PR | none |
| docs-kit | `generalize-build-assembly` | spec-driven | one PR | none |
| docs-kit | `add-cue-definitions-extractor` | spec-driven | one PR | G2.0 |
| docs-kit | `add-crd-extractor` | spec-driven | one PR | G2.0 |
| docs-kit | `add-cobra-extractor` | spec-driven | one PR | G2.0 |
| docs-kit | `add-go-api-extractor` | spec-driven | one PR | G2.0 |
| docs-kit | `pull-docs-placement` | spec-driven | one PR | G2.0 |
| docs-kit | `add-authored-docs` | spec-driven | one PR | G2.0 |
| docs-kit | `add-serve-command` | spec-driven | one PR | G2.0 |
| docs-kit | `add-enhancements-bundle` | spec-driven | one PR | G2.0 and `pull-docs-placement` merged |
| docs-kit | `retire-lint-conformance-binding` | spec-driven | one PR | G3.5 |
| opmodel.dev | `add-catalog-version-history` | docs-site-change | two PRs | G1b |
| core | `publish-definitions-bundle` | spec-driven | one PR per section | G2-core |
| library | `publish-go-api-bundle` | spec-driven | one PR per section | G2-library |
| opm-operator | `publish-crd-bundle` | spec-driven | one PR per section | G2-operator |
| cli | `publish-cli-bundle` | spec-driven | one PR per section | G2-cli |
| opmodel.dev | `pull-reference-bundles` | docs-site-change | one PR per section | G2-site |
| catalog_opm | `publish-site-docs-bundle` | catalog-change | one PR per section | G3.0 |
| opm | none: two PRs, no OpenSpec workspace | n/a | two PRs | release-please PR: none; docs PR: G3.0 |
| enhancements | none: one PR, no OpenSpec workspace | n/a | one PR | G3.0 and `add-enhancements-bundle` released |
| opmodel.dev | `serve-docs-from-bundles` | docs-site-change | one PR per section | G3.1 |
| opmodel.dev | `retire-git-pipeline` | docs-site-change | one PR | G3.4 |

A docs-kit change reaches other repositories only through a release: merging its release-please PR (owner) creates `vX.Y.Z` and its binaries. Every gate below that says "released" means that. Callers bump `.opm-docs-version` and their `publish.yml@vX.Y.Z` ref together, in one PR (C5); opmodel.dev bumps `OPM_DOCS_VERSION` and `OPM_DOCS_SHA256` in `site/Dockerfile` together (C12).

**The site bumps first** (C12, `generalize-build-assembly` D10). `#Manifest` is closed, so an `opm-docs` older than a bundle's tool refuses a manifest with a field it does not know (`owns`, `pins`, `edit`, a `section` placement). opmodel.dev's pinned `opm-docs` is never older than any producer's `.opm-docs-version`: each docs-kit release reaches `site/Dockerfile` before any producer moves to it.

Sibling plans cite the gates below by name, never by another repository's section number.

## Gates

| Gate | Holds when |
|---|---|
| G1b | `add-version-history` is released |
| G2.0 | `generalize-build-assembly` is merged on docs-kit `main` |
| G2-core | `add-cue-definitions-extractor`, `add-authored-docs` and `generalize-build-assembly` are released |
| G2-library | `add-go-api-extractor`, `add-authored-docs` and `generalize-build-assembly` are released |
| G2-operator | `add-crd-extractor`, `add-authored-docs` and `generalize-build-assembly` are released |
| G2-cli | `add-cobra-extractor`, `add-authored-docs`, `pull-docs-placement` and `generalize-build-assembly` are released (the cli's G2-pins check runs `opm-docs pull`), and the tag `cobradump/v0.1.0` exists |
| G2-site | `pull-docs-placement` and `add-authored-docs` are released |
| G2-pins | release-mode backfills have published the bundles of core `v2.0.0-beta.1`, library `v1.0.0-beta.1` and opm-operator `v1.0.0-beta.4` (exactly what cli `main` pins), and then the cli release that first carries the hook (`v1.0.0-beta.6`) has published its bundle with those pins. Before the cli release PR merges: `opm-docs pull --local cli@v1.0=<cli out>` with a scratch `bundles.cue` holding opmodel.dev's planned `versions."v1.0"` (C16) resolves every pin; after: the same pull without `--local` succeeds anonymously (owner decision, 2026-10-03) |
| G2-switch | opmodel.dev `pull-reference-bundles`' v1.0 switch is merged (v1.0 reads core, cli, library and opm-operator from bundles) |
| G3.0 | `add-authored-docs` is released (phase 3 starts) |
| G3.1 | the first `catalog-opm-docs` release bundle is published, by the `publish-docs` job of the next opm release after catalog_opm's adoption (no dispatch: an older tag's tree holds a `docs-kit.cue` without `catalog-opm-docs`, and C5 reads the tree's config first) |
| G3.2 | `docs/opm` holds the `1.0.0-beta.1` bundle (opm's first release) |
| G3.3 | `docs/enhancements` has an `edge` bundle |
| G3.4 | opmodel.dev `serve-docs-from-bundles` is merged: v1.0 reads `catalog-opm-docs` and `opm` from bundles and the Enhancements section from its bundle |
| G3.5 | opmodel.dev `retire-git-pipeline` is merged (the shell lint is gone) |

## Follow-ups now (before or beside phase 1b)

These finish docs-kit `v0.2.1` (`fix: refuse a tag and catalog version mismatch, align the slashless catalog link message`):

1. **catalog_opm** (a `ci` PR, no OpenSpec change): `docs.yml` every `publish.yml@v0.2.0` to `publish.yml@v0.2.1`, `.opm-docs-version` to `v0.2.1`, in one PR. `task docs:bundle:check` refuses a mismatch.
2. **opmodel.dev** (a `chore(site)` PR, no OpenSpec change): `site/Dockerfile` `OPM_DOCS_VERSION=0.2.1` and `OPM_DOCS_SHA256` = the `opm-docs_0.2.1_linux_amd64.tar.gz` line of that release's `checksums.txt`; copy docs-kit's conformance fixtures `link-slashless-minor`, `link-slashless-edge` and `link-slashless-root` (`internal/dialect/testdata/conformance/`) into `site/tests/lint/` and align `site/scripts/lint-sources.sh` and its byte-identical contract copy to their expected output (C11 re-sync rule).

## Phase 1b: version history in the Catalogs tab

```text
 docs-kit                                opmodel.dev
 add-version-history PR, merge
 OWNER: release PR merge (0.3.0) ─────► add-catalog-version-history PR, merge
```

### opmodel.dev: `add-catalog-version-history`

Proposal must state: no new URL; the site gains one input file per catalog (`site/.bundles/<project>/history.json`, C13) and reads nothing else new; DESIGN decision 12 limits it to badges and a "Changes in X" list.

1. Bump `site/Dockerfile` to the release carrying `add-version-history` (version and SHA-256 together).
2. `gen-mounts.sh`: mount `*/history.json` beside `*/*/manifest.json`; `gen-catalogs.sh`: when the lock has a `history` entry for a project, check the file's SHA-256 equals it, and fail naming both otherwise; with no entry, no badges.
3. Adapter (`site/catalogs/_content.gotmpl`): put the page's member entry (by FQN) and its kind's `removed` and `lineage` slices into `params.catalog`.
4. Partials under `layouts/_partials/opm/`: under the member title, one badge from C13's derivations ("Added in X", "In <floor> or earlier", "Unreleased", "Changed in X" for the page's own segment, "Newer version" linking the newest apiVersion's page in the same segment); at page end, a "Changes in <segment>" list (every change of that segment, one line per field: added, removed, made required, default changed, type changed, "the spec changed in a way the field list does not show" for `spec`), worded more cautiously for a pair compared in `paths` mode; on a kind index, "Removed in X" entries linking `<root><lastIn>/<page>/`. Nothing inline in the spec code block (decided in planning).
5. Fixtures: the site's existing fixture segments (`site/tests/fixtures/bundles/catalog-opm/{4.4,4.5,edge}/`), with `data/catalog.json` enriched to carry real `fqn`s and `spec.fields` (an apiVersion pair, a member added later, a default change, a field made required, a member removed in edge), pulled with `--local`; regenerate the fixture lock; tests for each badge, the list, the digest mismatch and a `paths`-mode pair.
6. QA screenshots of a member page with badges and the list, light and dark, phone width.

Durable decision for opmodel.dev `AGENTS.md`: the site reads history, never computes it. The change ships as two PRs.

## Phase 2: every generated reference, and the product repos' authored pages

### One cutover per product repository (DESIGN decision 20, owner-confirmed 2026-10-03)

core, cli, library and opm-operator ship their whole `docs/site/` in the same bundle as their generated reference, from adoption on. A site version chooses those four bundles through the cli's pins, so shipping only the reference now and the authored pages in phase 3 would need a second release cascade and a second site switch. While the site still builds a repository from git, its committed generated pages stay in git and are left out of the bundle with the `markdown` source's `exclude` (C6); after the switch they are deleted with the `exclude`. Phase 3 then covers catalog_opm's `docs/site/`, opm and enhancements.


### Sequence

```text
 docs-kit                       core / library / opm-operator           cli                            opmodel.dev
 ─────────────────────────────  ──────────────────────────────────────  ─────────────────────────────  ─────────────────────────────
 1 generalize-build-assembly
   merge (G2.0)
 2 in parallel: four extractors,
   pull-docs-placement,
   add-authored-docs,
   add-serve-command; merge each
 3 OWNER: release PRs ────────────────────────────────────────────────────────────────────────────►  site bumps opm-docs first
   (one release may carry       ─► 4 adopt (each repo, own gate):        5 adopt (G2-cli):             6 pull-reference-bundles,
   several changes)                docs-kit.cue, docs.yml, publish-docs    hack/docskit-dump, pins,      support (G2-site): pull docs
                                   job, local backfill dry run; merge:     docs.yml setup-go; merge:     bundles side by side with git
                                   edge publishes                          edge publishes
                                4b OWNER: package public; release-mode
                                   backfills core v2.0.0-beta.1,
                                   library v1.0.0-beta.1,
                                   opm-operator v1.0.0-beta.4 ──────────► 5b OWNER: merge cli#276
                                                                            (v1.0.0-beta.6) after the
                                                                            G2-pins check ──────────────► 7 v1.0 switch (G2-pins)
                                8 retire: generators, committed        ◄── 8 retire: cmdref, pages ◄──── merge (G2-switch)
                                  pages and exclude                         and exclude                  9 workspace docs (G2-switch)
```

1. **docs-kit `generalize-build-assembly`** (one PR). Gate for everything else in phase 2.
2. **docs-kit, in parallel**: `add-cue-definitions-extractor`, `add-crd-extractor`, `add-cobra-extractor` (also creates the `cobradump` component), `add-go-api-extractor`, `pull-docs-placement`, `add-authored-docs`, and `add-serve-command` (useful to authors from step 4 on; nothing waits for it). Each is one PR with its archive.
3. **Releases.** The owner merges docs-kit's release PRs; `cobradump/v0.1.0` comes from its own release PR. opmodel.dev bumps its pinned `opm-docs` to each release before any producer moves to it (the site bumps first). A repository starts step 4 at its own gate (G2-core, G2-library, G2-operator, G2-cli).
4. **core, library, opm-operator adopt.** Each adoption runs a **backfill dry run** before merging: in a worktree of the tag to backfill, `opm-docs build --release <tag> --source <tag worktree> --config docs-kit.cue` with `main`'s config, linted green, so the dispatch cannot fail on the tree. After merge, `main` publishes `edge`.
   - **4b. Backfills (owner decision, 2026-10-03).** The owner dispatches `mode: release` for core `v2.0.0-beta.1`, library `v1.0.0-beta.1` and opm-operator `v1.0.0-beta.4`, exactly the versions cli `main` pins (library from its `go.mod`, core through the library's `DefaultSchemaModule`, the operator from the cli's pinned operator version). The config comes from `main`, the sources from the tag, and a pattern matching nothing is ignored (C5, C6). The library's garbled kernel package doc is in `v1.0.0-beta.1`'s tree: its fix is a comment-only change, so it reaches the backfilled bundle as a docs revision (`mode: revision`) or with the next library release.
   - Open release PRs at the time of planning: library#155 (`v1.0.0-beta.2`) and opm-operator#178 (`v1.0.0-beta.5`). One that merges after its repository adopts publishes its own bundle through `publish-docs`; one that merges before needs a backfill too. Either way, if the cascade then moves the cli's pins to it, G2-pins checks that version instead.
5. **cli adopts**, then the owner merges its release PR (cli#276, `v1.0.0-beta.6`, the first cli release with the hook) only when the cli's adoption has merged and the G2-pins check passes. This is the hard gate of phase 2: a cli bundle whose pins lack bundles breaks every site pull (C16 D2).
6. **opmodel.dev `pull-reference-bundles`, support** can merge from G2-site, before any product bundle exists, tested with fixture bundles; v1.0 still reads git.
7. **opmodel.dev `pull-reference-bundles`, v1.0 switch** once G2-pins holds; its merge is G2-switch.
8. **core, cli and opm-operator retire** (after G2-switch): delete the generator, the committed generated pages and the `exclude`; the library has none. These land on `main` only; released bundles are unaffected.
9. **Workspace docs** (a workspace PR at G2-switch): root `AGENTS.md`'s release-branch bullet ("A docs-only fix in `core` or `catalog_opm` cuts no release: `opmodel.dev` builds their docs from the release branch head ...") becomes: a docs fix reaches a released version through a docs revision (`mode: revision` in the repository's `Docs` workflow), never through a branch push. `STYLE.md`'s "Reference facts are generated in the owning repository" bullet drops "its output is committed under `docs/site/reference/` with a check that fails when it is stale" and the marker-comment sentence: generated reference is built in the owning repository's CI and published in its docs bundle, never committed.

Done (phase 2): no repository commits generated pages; v1.0's reference and the four repositories' authored pages come from bundles; the site's two Reference placeholders are gone.

### Docs revisions stay manual

A docs fix to a released version is a `workflow_dispatch` of the repository's `Docs` workflow with `mode: revision`, `tag` and `fix` (C3 "Docs revisions"). Nothing dispatches it automatically; automating it is tracked in docs-kit#16, with one issue per caller: catalog_opm#130, core#101, library#164, opm-operator#188, cli#282, opm#22.

### What every sibling change must reference

- The contracts, by number: `https://github.com/open-platform-model/docs-kit/blob/main/docs/contracts.md` (`docs-kit C15`). Until a change lands, its `design.md` on docs-kit `main` (`openspec/changes/<change>/design.md`) shows the contract.
- `publish.yml` by release tag, never by SHA (C5, C9): the signer glob is `refs/tags/v[0-9]*`, and `publish.yml` reads the caller's repo-root `.opm-docs-version`. A caller's workflow comment says why the tag pin is a deliberate exception to SHA pinning.
- DESIGN decisions by number (`docs-kit DESIGN decision 10`). docs-kit has no enhancement entry, so no `enhancement.yaml` and no delivery log.
- Gates by name (this file), never another repository's section number.
- The reference adopter: catalog_opm's `.github/workflows/docs.yml`, `release.yml` job `publish-docs`, `Taskfile.yml` tasks `tools:opm-docs`, `docs:pins:check`, `docs:bundle`, `docs:bundle:check` and `.tasks/opm-docs.sh`.

Common to the four product siblings' adoption, each with its own project from C1:

1. `docs-kit.cue` exactly as the extractor change's design shows (core: `add-cue-definitions-extractor` D1; opm-operator: `add-crd-extractor` D1; cli: `add-cobra-extractor` D3; library: `add-go-api-extractor` D1), including the `markdown` source over `docs/site` with its transitional `exclude`.
2. `.github/workflows/docs.yml` as catalog_opm's: `check` on `pull_request`, `edge` on push to `main`, `workflow_dispatch` with `mode` (`release`, `revision`), `tag` and `fix`; `permissions: {}` at the top and per job as C5's table (`check`: `contents: read`, `packages: read`; the rest: `contents: read`, `packages: write`, `id-token: write`).
3. `release.yml`: a job `publish-docs` calling `publish.yml` with `mode: release` and the release's tag, gated on that package's release output, after the job that publishes the release (core: after `publish-cue`; library: after `release-please`; opm-operator: after `image-release`; cli: after `goreleaser` succeeds), with the publishing permissions. It runs on the push to `main` that merged the release PR (C5, DESIGN decision 9), never on `release: published`.
4. `.opm-docs-version` naming the docs-kit release that meets the repository's gate (and that the site already runs); `tools:opm-docs`, `docs:pins:check` (refuses a `publish.yml@` ref that names another release than `.opm-docs-version`), `docs:bundle` and `docs:bundle:check` (runs `docs:pins:check` first) tasks (C12: download, verify the checksum, install to `.bin/`); `docs:bundle:check` in `task check`; `.gitignore` gains `/out/` and `/.bin/`.
5. The backfill dry run of step 4 (core, library, opm-operator), its command and result recorded in the change.
6. `AGENTS.md`: a "Docs bundles" paragraph (what publishes when, how to preview with `task docs:bundle` or `opm-docs serve`, how to backfill or revise a release, that revisions are dispatched by hand).
7. Gate: `task check` green; the PR's `Docs / check` job green.

Then (owner): verify the new GHCR package `docs/<project>` is public and linked to the repository (the phase-1 spike found new packages inherit the public repository's visibility), dispatch the backfill (or, for the cli, merge the release PR) and confirm the full, release, minor and major tags verify (`cosign verify` with C9's flags or an anonymous `opm-docs pull`). Record run URLs in the change.

Retire (after G2-switch): remove the generator, its tests, tasks and CI steps, the committed generated pages and the `exclude`.

### core: `publish-definitions-bundle`

- Adopt: move `tools/refgen/groups.go`'s `pages` and `excluded` lists verbatim into `docs-kit.cue` (`add-cue-definitions-extractor` D1); `skip: ["*_pins.cue"]`; `exclude: ["reference/definitions/"]` on the `markdown` source. `publish-docs` after `publish-cue`. Backfill dry run of `v2.0.0-beta.1`. Keep `tools/refgen` and `docs:reference:check` until retire.
- Publish: the owner's backfill of `v2.0.0-beta.1` publishes `docs/core`; record it.
- Retire: delete `tools/refgen/`, `docs/site/reference/definitions/`, the `docs:reference*` and `refgen:test` tasks, the CI steps that run them, and the `exclude`.

### library: `publish-go-api-bundle`

- Adopt: repair `opm/kernel/doc.go`'s garbled "Surface" list and the undocumented exported symbols the docs-kit trial build lists (`add-go-api-extractor` task 2.2); `docs-kit.cue` (`add-go-api-extractor` D1, no `exclude`: the library commits no generated pages); link `docs/site/embedding/embed-the-kernel.md` to `/docs/reference/go-api/`. Backfill dry run of `v1.0.0-beta.1`.
- Publish: the owner's backfill of `v1.0.0-beta.1` publishes `docs/library`; then a docs revision with the kernel doc fix (comment-only), unless a new library release that the cli pins replaces it first.
- No retire step.

### opm-operator: `publish-crd-bundle`

- Adopt: `docs-kit.cue` (`add-crd-extractor` D1: `hideSamplesMatching`, `stripLabels`, `weight: 7`) with `reconciledBy` taken from what `hack/crdref`'s controller scan finds at the adopting commit, `citations: "link"`, and `exclude: ["reference/operator-resources.md"]` on the `markdown` source. Optionally a repository test comparing `reconciledBy` with the controllers. Backfill dry run of `v1.0.0-beta.4`.
- Publish: the owner's backfill of `v1.0.0-beta.4` publishes `docs/opm-operator`.
- Retire: delete `hack/crdref/`, its task (`dev:docs:reference`) and CI step; reduce `docs/site/reference/operator-resources.md` to its front matter and intro (no marker block, no `## <Kind>` heading); remove the `exclude`, so the authored intro completes the generated page (C15).

### cli: `publish-cli-bundle`

- Adopt: `hack/docskit-dump/main.go` importing `github.com/open-platform-model/docs-kit/cobradump` at `v0.1.0` or later (`add-cobra-extractor` D1): `Write(cmd.NewRootCmd(), ...)`, and with the argument `pins`, `WritePins` of `library`, `core` and `opm-operator`, each read where opmodel.dev's `resolve-versions.sh` reads it today (library from the cli's `go.mod`, core from the library's default schema module, the operator from the cli's pinned operator version); a test that the program's pins equal those sources. `docs-kit.cue` (`add-cobra-extractor` D3, with `pins` and `exclude: ["reference/cli/"]`); `docs.yml` passes `setup-go: true`; `publish-docs` after `goreleaser` succeeds.
- Publish (G2-pins check before cli#276 merges): `v1.0.0-beta.6` publishes `docs/cli`; confirm with the anonymous pull of G2-pins.
- Retire: delete `internal/cmdref/`, `hack/cmdref/`, `docs/site/reference/cli/`, the `docs:reference*` tasks and the CI "command reference" jobs (`ci.yml`, `pr.yml`), and the `exclude`.

### opmodel.dev: `pull-reference-bundles`

Proposal must state: no URL changes (the reference keeps `/docs/reference/cli/`, `/docs/reference/definitions/`, `/docs/reference/operator-resources/`, and gains `/docs/reference/go-api/`); the site gains docs bundles as an input and stops reading core, cli, library and opm-operator from git for v1.0; the v1.0 switch merges only at G2-pins.

Support (from G2-site; v1.0 unchanged):

1. Bump `site/Dockerfile` to the release carrying `pull-docs-placement` and `add-authored-docs` (before any producer moves to it).
2. `bundles.cue`: `docs` for `cli`, `core`, `library`, `opm-operator` (C16; no `versions` yet).
3. Side by side: per site version, a repository is read from bundles exactly when the version's `versions` entry names its project; otherwise from git as today. `gen-mounts.sh` mounts `site/.bundles/_versions/<v>/<project>/content` into that version's `content/docs` and drops that repository's `docs/site` git mount for that version. `versions.conf` keeps each version's `label`, `weight` and `default` until `retire-git-pipeline` moves them.
4. Page data from manifests: "Last updated" from `pages[].lastmod`; "Edit this page" `https://github.com/<source.repo>/edit/main/<edit>` when `edit` is set; "View source" `source` at `source.commit`; none for generated pages beyond View source (C8 table, `add-authored-docs` D1). The build stamp records the lock's `docs` entries.
5. The site's post-build link check covers `/docs/` links across bundles (C15 leaves them to it).
6. `OPM_BUNDLES_LOCAL` accepts `<project>@v<M>.<m>=<dir>` (C16 D6) so `opm-docs serve --site` works.
7. Fixtures: `site/tests/fixtures/bundles/_versions/v1.0/{cli,core,library,opm-operator}/` (an anchor fixture with `pins`), a test of a collision (C16 D4) and a pin without a bundle; a two-version test with one version on bundles and one on git.

v1.0 switch (G2-pins): `bundles.cue` `versions: "v1.0": {anchor: {project: "cli", tag: "1.0"}, pinned: ["library", "core", "opm-operator"]}`; remove the placeholders `content/docs/reference/cli/_index.md` and `content/docs/reference/definitions/_index.md` and their check-pages exemption; `resolve-versions.sh` stops resolving those four repositories for v1.0 (it still resolves opm and catalog_opm until phase 3). Its merge is G2-switch.

Durable decisions for opmodel.dev `AGENTS.md`: site versions come from `bundles.cue` `versions`; which repositories a version reads from git until phase 3 ends; the site bumps `opm-docs` before any producer.

## Phase 3: authored pages of catalog_opm and opm, and the enhancements

```text
 docs-kit                          catalog_opm / opm / enhancements           opmodel.dev
 add-enhancements-bundle merge,
 OWNER release ─────────────────►  (site bumps first)
                                   1 catalog_opm publish-site-docs-bundle;
                                     next opm release publishes it (G3.1) ─► 2 serve-docs-from-bundles: catalog-opm-docs
                                   3 opm: release-please PR (no gate),
                                     docs PR (G3.0), OWNER first
                                     release (G3.2) ───────────────────────► 4 serve-docs-from-bundles: opm
                                   5 enhancements PR, spike fixes first
                                     (G3.3) ───────────────────────────────► 6 serve-docs-from-bundles: enhancements
                                                                            7 retire-git-pipeline (G3.4)
 8 retire-lint-conformance-binding (G3.5)
```

The site supports git and bundle docs side by side from phase 2 (`pull-reference-bundles` support), so each repository cuts over on its own.

### 1. catalog_opm: `publish-site-docs-bundle` (first)

Proposal must state: a second bundle, `catalog-opm-docs` (C1 naming rule: the repository's name is already the tab), placed in `/docs/`, built from the same `opm-v*` release tags as the tab; no catalog release needed for the adoption itself (class `ci`); its first published bundle comes from the next opm release (G3.1).

- Adopt: `docs-kit.cue` gains `"catalog-opm-docs"` as `add-authored-docs` D2 shows; `docs.yml` runs `check` and `edge` for both projects (two jobs or a matrix over `project`), and the dispatch gains a `project` choice; `release.yml` `publish-docs` calls `publish.yml` once per project; `docs:bundle:check` checks both.
- Publish (owner): verify `docs/catalog-opm-docs` is public; its first release bundle is published by the `publish-docs` job of the next opm release. No dispatch for an older `opm-v4.*` tag: that tree's `docs-kit.cue` has no `catalog-opm-docs`, and C5 reads the tree's config before `main`'s. Record the run.

### 2. opmodel.dev: `serve-docs-from-bundles`, catalog_opm (G3.1)

`bundles.cue` `docs."catalog-opm-docs"` (`repo: "open-platform-model/catalog_opm"`) and `versions."v1.0".tags: {"catalog-opm-docs": "4"}` (the major, as `catalog-line = opm-v4` does today); catalog_opm's `docs/site` leaves the git mounts and `resolve-versions.sh`.

### 3. opm: release-please, then its bundle (no OpenSpec workspace; two PRs)

1. `chore: release with release-please` PR (no gate): `release-please-config.json` (one package `.`, `release-type: simple`, `include-v-in-tag: true`, tags `vX.Y.Z`, `draft: false`, `"initial-version": "1.0.0-beta.1"`, and prerelease versioning: `"versioning": "prerelease"`, `"prerelease": true`, `"prerelease-type": "beta"`, so later releases go `1.0.0-beta.2` and onward until the owner drops it for `1.0.0`), with `changelog-sections` that make `docs` a **visible** section (`{"type": "docs", "section": "Documentation", "hidden": false}`, beside `feat` and `fix`; decided in planning, recorded in opm#21): opm's commits are almost all `docs:`, and release-please opens no release PR when every commit since the last release falls in hidden sections. `.release-please-manifest.json` without an entry for `.` until the first release (`initial-version` applies only to a package with no release yet, as docs-kit's own `0.1.0` did), `.github/workflows/release.yml` with the release App token as the other repositories have it, `CHANGELOG.md`. The first version is `1.0.0-beta.1` (DESIGN decision 21), so the minor tag `1.0` follows the v1.0 site version. Never a `Release-As:` footer: squash merges use a blank body (workspace `RELEASING.md`), so a footer never reaches `main`; if `initial-version` cannot be used, a temporary `release-as` in the config, removed by the next PR, is the fallback.
2. `ci(docs): publish the opm docs bundle with docs-kit` PR (G3.0): `docs-kit.cue` (`add-authored-docs` D2), `docs.yml`, `publish-docs` in `release.yml`, `.opm-docs-version`, Taskfile tasks as in phase 2's common list (`docs:pins:check` included), `AGENTS.md` paragraph.
3. **Owner**: merge the first release PR; verify `docs/opm` is public and its tags verify (G3.2).

### 4. opmodel.dev: `serve-docs-from-bundles`, opm (G3.2)

`docs.opm` and `versions."v1.0".tags.opm: "1.0"`; opm's `docs/site` (including the `/docs/` landing and `start/_index.md`) leaves the git mounts and `resolve-versions.sh`'s head-of-main row.

### 5. enhancements (no OpenSpec workspace; enhancements#86, after `add-enhancements-bundle` is released)

Before it, the source fixes `add-enhancements-bundle` section 1 lists (spike findings), as ordinary PRs. Then `ci(docs): publish the enhancements bundle with docs-kit`: `docs-kit.cue` (`add-enhancements-bundle` D2), `docs.yml` with `check` on pull requests and `edge` on push to `main` only (no release or revision mode: a section is edge only), `.opm-docs-version` with a `docs:pins:check`-style check that the `publish.yml@` ref names the same release. **Owner**: verify `docs/enhancements` is public (G3.3).

### 6. opmodel.dev: `serve-docs-from-bundles`, enhancements (G3.3)

`bundles.cue` `sections: enhancements: {repo: "open-platform-model/enhancements", root: "/enhancements/"}`; the Enhancements section built from `site/.bundles/enhancements/edge/` (pages from `content/`, header data from `data/enhancements.json`). `site/enhancements/_content.gotmpl` is kept but reduced to what Hugo still needs (each page's `url` outside every version, from the bundle); delete `layouts/_partials/opm/enh-clean.html`, `layouts/enhancements/_markup/render-link.html`, the `gen-mounts.sh` enhancements repository mount and `versions.conf`'s `[section "enhancements"]`; copy docs-kit's `link-enhancements-graph` lint fixture.

### 7. opmodel.dev: `retire-git-pipeline` (G3.4)

Delete `resolve-versions.sh`, `materialise.sh`, `gen-lastmod.sh`, the source-repository rows and floors of `versions.conf` (site versions live in `bundles.cue`; each version's `label`, `weight` and `default` move there or to site config), `frozen.conf` (the lock's `--frozen` replaces it), `lint-sources.sh` and its contract copy (every page is linted by `opm-docs lint` on pull). Site-owned pages keep "Last updated" through Hugo's GitInfo only if the build image sees the site's own `.git`; an opmodel.dev spike settles that before `gen-lastmod.sh` goes. Keep `site/tests/lint/` only if the site still wants it; it no longer binds docs-kit. Done (phase 3, DESIGN.md): the site build reads no git repository except its own.

### 8. docs-kit: `retire-lint-conformance-binding` (G3.5)

A small spec-driven change: REMOVE the `dialect-lint` requirement "The conformance fixture set binds both linters until phase 3" (the fixtures stay as ordinary tests of the Go lint) and rewrite C11's "Agreement with the site's shell lint until phase 3".

## Owner items

- Merge each docs-kit release PR at the gates above, and the `cobradump` release PR; let opmodel.dev's `opm-docs` bump land before any producer's.
- Confirm the tag rulesets cover `cobradump/v*` tags as they cover `v*`.
- New GHCR packages, each checked public on first publish: `docs/core`, `docs/library`, `docs/opm-operator`, `docs/cli`, `docs/catalog-opm-docs`, `docs/opm`, `docs/enhancements`.
- Phase 2 backfills: dispatch `mode: release` for core `v2.0.0-beta.1`, library `v1.0.0-beta.1`, opm-operator `v1.0.0-beta.4` after each adoption merges.
- Hold cli#276 (`v1.0.0-beta.6`) until the cli's adoption has merged and the G2-pins check passes. library#155 and opm-operator#178 may merge at any time; one merged before its repository adopts needs a backfill too, and if the cascade bumps the cli's pins to it, G2-pins checks that version.
- opm: install the release App, provide `vars.RELEASE_APP_CLIENT_ID` and `secrets.RELEASE_APP_PRIVATE_KEY`, cover opm with the `tags-immutable` and `tags-create-app-only` rulesets and enable immutable releases; first version `1.0.0-beta.1` (DESIGN decision 21), set by `initial-version` in `release-please-config.json`, never by a commit footer, with `docs` a visible changelog section so `docs:` commits open release PRs. Workspace edits for a newly released repository: add opm to the release-tag scope of root `AGENTS.md` ("Release Tags Are Immutable"), to the hooks and guards that list released repositories, and to `RELEASING.md`.
- Docs revisions are dispatched by hand (docs-kit#16 tracks automating them).

## Decisions made in planning (for the owner to confirm)

Confirmed by the owner on 2026-10-03 and recorded in DESIGN.md: one cutover per product repository (decision 20) and opm's first release `1.0.0-beta.1` (decision 21). Also decided by the owner that day: the phase-2 backfills of exactly the versions cli `main` pins (G2-pins). The rest stand until the owner says otherwise:

- Version history: field changes appear in the "Changes in X" list, not inline in the spec block; edge is compared with the newest minor; type, default and ref compare only within one docs-kit minor; the lock records the history digest (`add-version-history`).
- Pins live in `manifest.json`, printed by a repository command; pinned bundles resolve through the release tag so docs revisions follow (`generalize-build-assembly`, `pull-docs-placement`).
- `publish.yml` builds in a job without the signing token and publishes in another that runs no repository code; its concurrency groups sit at the workflow level (`generalize-build-assembly`).
- The site bumps `opm-docs` first; `edit` is written only for docs placements, so tab bundles stay readable by an older site.
- Docs bundles declare `owns`; `pull` refuses overlaps per site version and replaces a version whole; cross-bundle `/docs/` links are left to the site's link check.
- `catalog-opm-docs` naming rule; `_versions/` unpack layout; `v<MAJOR>.<MINOR>` site-version names.
- `cobradump` is a nested module released as `cobradump/vX.Y.Z`, excluded from the root package's paths; the cli's hook is a `hack/` program, not a hidden command; the dump carries raw text.
- The `crd` extractor takes `reconciledBy` from config instead of scanning Go, picks samples by kubebuilder file name, hides fixture samples and strips scaffold labels; one page per configured path.
- `go-api` parses without type checking; anchors are Hugo's default heading anchors.
- `pages[].edit` is set only when the same path exists on `main` (no rename following).
- The enhancements section is built producer-side through mechanical transforms, never a relaxed lint; its root is fixed to `/enhancements/`.
- `opm-docs serve` has an embedded skeleton site with live rebuild, and a `--site` mode through opmodel.dev's own preview.
- opm's release-please config shows `docs` as a visible changelog section (recorded in opm#21).
