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
| opmodel.dev | `add-catalog-version-history` | docs-site-change | one PR | G1b |
| core | `publish-definitions-bundle` | spec-driven | one PR per section | G2-core |
| library | `publish-go-api-bundle` | spec-driven | one PR per section | G2-library |
| opm-operator | `publish-crd-bundle` | spec-driven | one PR per section | G2-operator |
| cli | `publish-cli-bundle` | spec-driven | one PR per section | G2-cli |
| opmodel.dev | `pull-reference-bundles` | docs-site-change | one PR per section | G2-site |
| catalog_opm | `publish-site-docs-bundle` | catalog-change | one PR per section | G3.0 |
| opm | none: two PRs, no OpenSpec workspace | n/a | two PRs | G3.0 |
| enhancements | none: one PR, no OpenSpec workspace | n/a | one PR | G3.0 and `add-enhancements-bundle` released |
| opmodel.dev | `serve-docs-from-bundles` | docs-site-change | one PR per section | G3.1 |
| opmodel.dev | `retire-git-pipeline` | docs-site-change | one PR | G3.4 |

A docs-kit change reaches other repositories only through a release: merging its release-please PR (owner) creates `vX.Y.Z` and its binaries. Every gate below that says "released" means that. Callers bump `.opm-docs-version` and their `publish.yml@vX.Y.Z` ref together, in one PR (C5); opmodel.dev bumps `OPM_DOCS_VERSION` and `OPM_DOCS_SHA256` in `site/Dockerfile` together (C12).

## Gates

| Gate | Holds when |
|---|---|
| G1b | `add-version-history` is released |
| G2.0 | `generalize-build-assembly` is merged on docs-kit `main` |
| G2-core | `add-cue-definitions-extractor`, `add-authored-docs` and `generalize-build-assembly` are released |
| G2-library | `add-go-api-extractor`, `add-authored-docs` and `generalize-build-assembly` are released |
| G2-operator | `add-crd-extractor`, `add-authored-docs` and `generalize-build-assembly` are released |
| G2-cli | `add-cobra-extractor`, `add-authored-docs` and `generalize-build-assembly` are released, and the tag `cobradump/v0.1.0` exists |
| G2-site | `pull-docs-placement` is released |
| G2-pins | a cli release with a bundle exists whose `manifest.json` `pins` all resolve: `opm-docs pull` with a scratch `bundles.cue` holding opmodel.dev's planned `versions."v1.0"` (C16) succeeds anonymously |
| G2-switch | opmodel.dev `pull-reference-bundles` section 2 is merged (v1.0 reads core, cli, library and opm-operator from bundles) |
| G3.0 | `add-authored-docs` is released (phase 3 starts) |
| G3.1 | the first phase-3 bundle (catalog_opm's `catalog-opm-docs`) is published |
| G3.4 | opm and enhancements bundles are published and v1.0 reads them |
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
5. Fixtures: `site/tests/fixtures/bundles/catalog-opm/{4.5,4.6,edge}/` with `data/catalog.json` carrying real `fqn`s and `spec.fields` (a `backup@v1alpha1` in 4.5 and 4.6, `backup@v1beta1` added in 4.6, a default change, a field made required, a member removed in edge), pulled with `--local`; regenerate the fixture lock; tests for each badge, the list, the digest mismatch and a `paths`-mode pair.
6. QA screenshots of a member page with badges and the list, light and dark, phone width.

Durable decision for opmodel.dev `AGENTS.md`: the site reads history, never computes it.

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
 3 OWNER: release PRs ─────────► 4 §1 adopt (each repo, own gate):       5 §1 adopt (G2-cli):           6 pull-reference-bundles §1
   (one release may carry         docs-kit.cue, docs.yml, publish-docs     hack/docskit-dump, pins,       (G2-site): pull docs bundles,
   several changes)               job; merge: edge publishes               docs.yml setup-go; merge:      side by side with git; fixtures
                                4b OWNER: package public; next release     edge publishes
                                   of each publishes its bundle, in the
                                   cascade order core → library →
                                   opm-operator ────────────────────────► 5b OWNER: cli release
                                                                            pinning those releases
                                                                            (G2-pins) ──────────────────► 7 §2 switch v1.0 (G2-pins)
                                8 §3 each: delete generators,           ◄─ 8 §3 delete cmdref,  ◄───────── merge (G2-switch)
                                   committed pages and exclude              pages and exclude
```

1. **docs-kit `generalize-build-assembly`** (one PR). Gate for everything else in phase 2.
2. **docs-kit, in parallel**: `add-cue-definitions-extractor`, `add-crd-extractor`, `add-cobra-extractor` (also creates the `cobradump` component), `add-go-api-extractor`, `pull-docs-placement`, `add-authored-docs`, and `add-serve-command` (useful to authors from step 4 on; nothing waits for it). Each is one PR with its archive.
3. **Releases.** The owner merges docs-kit's release PRs; a repository starts its step 4 at its own gate (G2-core, G2-library, G2-operator, G2-cli). `cobradump/v0.1.0` is created by its own release PR.
4. **core, library, opm-operator adopt** (each its section 1), then each **releases** in the existing release cascade (core, then library pinning that core, then opm-operator), and the release's `publish-docs` job publishes its bundle. A release cut before adoption has no bundle; a dispatched backfill (`mode: release`) is the fallback, and works for these three (the config comes from `main`, the sources from the tag, a pattern matching nothing is ignored, C5, C6), but a fresh release is preferred because it carries the doc fixes made during adoption (the library's kernel package doc in particular).
5. **cli adopts** (section 1), then **releases** (section 2) only when G2-pins holds: every version it pins has a bundle. This is the hard gate of phase 2: a cli bundle whose pins lack bundles breaks every site pull (C16 D2).
6. **opmodel.dev `pull-reference-bundles` section 1** can merge from G2-site, before any product bundle exists, tested with fixture bundles; v1.0 still reads git.
7. **opmodel.dev section 2 switches v1.0** once G2-pins holds (G2-switch on merge).
8. **core, cli, opm-operator section 3** (after G2-switch): delete the generator, the committed generated pages and the `exclude`; the library has none. These land on `main` only; released bundles are unaffected.

Done (phase 2): no repository commits generated pages; v1.0's reference and the four repositories' authored pages come from bundles; the site's two Reference placeholders are gone.

### What every sibling change must reference

- The contracts, by number: `https://github.com/open-platform-model/docs-kit/blob/main/docs/contracts.md` (`docs-kit C15`). Until a change lands, its `design.md` on docs-kit `main` (`openspec/changes/<change>/design.md`) shows the contract.
- `publish.yml` by release tag, never by SHA (C5, C9): the signer glob is `refs/tags/v[0-9]*`, and `publish.yml` reads the caller's repo-root `.opm-docs-version`. A caller's workflow comment says why the tag pin is a deliberate exception to SHA pinning.
- DESIGN decisions by number (`docs-kit DESIGN decision 10`). docs-kit has no enhancement entry, so no `enhancement.yaml` and no delivery log.
- The reference adopter: catalog_opm's `.github/workflows/docs.yml`, `release.yml` job `publish-docs`, `Taskfile.yml` tasks `tools:opm-docs`, `docs:bundle`, `docs:bundle:check` and `.tasks/opm-docs.sh`.

Common to the four product siblings' section 1 (adopt), each with its own project from C1:

1. `docs-kit.cue` exactly as the extractor change's design shows (core: `add-cue-definitions-extractor` D1; opm-operator: `add-crd-extractor` D1; cli: `add-cobra-extractor` D3; library: `add-go-api-extractor` D1), including the `markdown` source over `docs/site` with its transitional `exclude`.
2. `.github/workflows/docs.yml` as catalog_opm's: `check` on `pull_request`, `edge` on push to `main`, `workflow_dispatch` with `mode` (`release`, `revision`), `tag` and `fix`; `permissions: {}` at the top and per job as C5's table (`check`: `contents: read`, `packages: read`; the rest: `contents: read`, `packages: write`, `id-token: write`).
3. `release.yml`: a job `publish-docs` calling `publish.yml` with `mode: release` and the release's tag, gated on that package's release output, after the job that publishes the release (core: after `publish-cue`; library: after `release-please`; opm-operator: after `image-release`; cli: after `goreleaser` succeeds), with the publishing permissions. It runs on the push to `main` that merged the release PR (C5, DESIGN decision 9), never on `release: published`.
4. `.opm-docs-version` naming the docs-kit release that meets the repository's gate; `tools:opm-docs`, `docs:bundle` and `docs:bundle:check` tasks (C12: download, verify the checksum, install to `.bin/`); `docs:bundle:check` in `task check`; `.gitignore` gains `/out/` and `/.bin/`.
5. `AGENTS.md`: a "Docs bundles" paragraph (what publishes when, how to preview with `task docs:bundle` or `opm-docs serve`, how to backfill or revise a release).
6. Gate: `task check` green; the PR's `Docs / check` job green.

Section 2 (owner): verify the new GHCR package `docs/<project>` is public and linked to the repository (the phase-1 spike found new packages inherit the public repository's visibility), then cut the next release (or dispatch the backfill) and confirm the full, release, minor and major tags verify (`cosign verify` with C9's flags or an anonymous `opm-docs pull`). Record run URLs in the change.

Section 3 (after G2-switch): remove the generator, its tests, tasks and CI steps, the committed generated pages and the `exclude`.

### core: `publish-definitions-bundle`

- Section 1: move `tools/refgen/groups.go`'s `pages` and `excluded` lists verbatim into `docs-kit.cue` (`add-cue-definitions-extractor` D1); `skip: ["*_pins.cue"]`; `exclude: ["reference/definitions/"]` on the `markdown` source. `publish-docs` after `publish-cue`. Keep `tools/refgen` and `docs:reference:check` until section 3.
- Section 2: the next core release publishes `docs/core`; record it.
- Section 3: delete `tools/refgen/`, `docs/site/reference/definitions/`, the `docs:reference*` and `refgen:test` tasks, the CI steps that run them, and the `exclude`.

### library: `publish-go-api-bundle`

- Section 1: first repair `opm/kernel/doc.go`'s garbled "Surface" list and the undocumented exported symbols the docs-kit trial build lists (`add-go-api-extractor` task 2.2); then `docs-kit.cue` (`add-go-api-extractor` D1, no `exclude`: the library commits no generated pages); link `docs/site/embedding/embed-the-kernel.md` to `/docs/reference/go-api/`.
- Section 2: the next library release (in the cascade, pinning the core release of step 4) publishes `docs/library`.
- No section 3.

### opm-operator: `publish-crd-bundle`

- Section 1: `docs-kit.cue` (`add-crd-extractor` D1) with `reconciledBy` taken from what `hack/crdref`'s controller scan finds at the adopting commit, `citations: "link"`, and `exclude: ["reference/operator-resources.md"]` on the `markdown` source. Optionally a repository test comparing `reconciledBy` with the controllers.
- Section 2: the next operator release publishes `docs/opm-operator`.
- Section 3: delete `hack/crdref/`, its task (`dev:docs:reference`) and CI step; reduce `docs/site/reference/operator-resources.md` to its front matter and intro (no marker block, no `## <Kind>` heading); remove the `exclude`, so the authored intro completes the generated page (C15).

### cli: `publish-cli-bundle`

- Section 1: `hack/docskit-dump/main.go` importing `github.com/open-platform-model/docs-kit/cobradump` at `v0.1.0` or later (`add-cobra-extractor` D1): `Write(cmd.NewRootCmd(), ...)`, and with the argument `pins`, `WritePins` of `library`, `core` and `opm-operator`, each read where opmodel.dev's `resolve-versions.sh` reads it today (library from the cli's `go.mod`, core from the library's default schema module, the operator from the cli's pinned operator version); a test that the program's pins equal those sources. `docs-kit.cue` (`add-cobra-extractor` D3, with `pins` and `exclude: ["reference/cli/"]`); `docs.yml` passes `setup-go: true`; `publish-docs` after `goreleaser` succeeds.
- Section 2 (G2-pins before merging the release PR): the next cli release publishes `docs/cli`; confirm with the scratch pull of G2-pins.
- Section 3: delete `internal/cmdref/`, `hack/cmdref/`, `docs/site/reference/cli/`, the `docs:reference*` tasks and the CI "command reference" jobs (`ci.yml`, `pr.yml`), and the `exclude`.

### opmodel.dev: `pull-reference-bundles`

Proposal must state: no URL changes (the reference keeps `/docs/reference/cli/`, `/docs/reference/definitions/`, `/docs/reference/operator-resources/`, and gains `/docs/reference/go-api/`); the site gains docs bundles as an input and stops reading core, cli, library and opm-operator from git for v1.0; `Delivery: one PR per section (v1.0 switches only after cli publishes a bundle whose pins resolve)`.

Section 1 (from G2-site; v1.0 unchanged):

1. Bump `site/Dockerfile` to the release carrying `pull-docs-placement` and `add-authored-docs`.
2. `bundles.cue`: `docs` for `cli`, `core`, `library`, `opm-operator` (C16; no `versions` yet).
3. Side by side: per site version, a repository is read from bundles exactly when the version's `versions` entry names its project; otherwise from git as today. `gen-mounts.sh` mounts `site/.bundles/_versions/<v>/<project>/content` into that version's `content/docs` and drops that repository's `docs/site` git mount for that version.
4. Page data from manifests: "Last updated" from `pages[].lastmod`; "Edit this page" `https://github.com/<source.repo>/edit/main/<edit>` when `edit` is set; "View source" `source` at `source.commit`; none for generated pages beyond View source (C8 table, `add-authored-docs` D1). The build stamp records the lock's `docs` entries.
5. The site's post-build link check covers `/docs/` links across bundles (C15 leaves them to it).
6. `OPM_BUNDLES_LOCAL` accepts `<project>@v<M>.<m>=<dir>` (C16 D6) so `opm-docs serve --site` works.
7. Fixtures: `site/tests/fixtures/bundles/_versions/v1.0/{cli,core,library,opm-operator}/` (an anchor fixture with `pins`), a test of a collision (C16 D4) and a pin without a bundle; a two-version test with one version on bundles and one on git.

Section 2 (G2-pins): `bundles.cue` `versions: "v1.0": {anchor: {project: "cli", tag: "1.0"}, pinned: ["library", "core", "opm-operator"]}`; remove the placeholders `content/docs/reference/cli/_index.md` and `content/docs/reference/definitions/_index.md` and their check-pages exemption; `resolve-versions.sh` stops resolving those four repositories for v1.0 (it still resolves opm and catalog_opm until phase 3). Merge is G2-switch.

Durable decisions for opmodel.dev `AGENTS.md`: site versions come from `bundles.cue` `versions`; which repositories a version reads from git until phase 3 ends.

## Phase 3: authored pages of catalog_opm and opm, and the enhancements

```text
 docs-kit                          catalog_opm / opm / enhancements           opmodel.dev
 add-enhancements-bundle merge,
 OWNER release ─────────────────►  1 catalog_opm publish-site-docs-bundle ─► 2 serve-docs-from-bundles §1
                                   3 opm: release-please PR, then docs PR,      (catalog-opm-docs) (G3.1)
                                     OWNER first release ──────────────────► 4 §2 (opm)
                                   5 enhancements PR (spike fixes first) ──► 6 §3 (enhancements section)
                                                                            7 retire-git-pipeline (G3.4)
 8 retire-lint-conformance-binding (G3.5)
```

The site supports git and bundle docs side by side from phase 2 (`pull-reference-bundles` section 1), so each repository cuts over on its own.

### 1. catalog_opm: `publish-site-docs-bundle` (first)

Proposal must state: a second bundle, `catalog-opm-docs` (C1 naming rule: the repository's name is already the tab), placed in `/docs/`, built from the same `opm-v*` release tags as the tab; no catalog release needed (class `ci`); `Delivery: one PR per section (opmodel.dev needs the published bundle after section 2)`.

- Section 1: `docs-kit.cue` gains `"catalog-opm-docs"` as `add-authored-docs` D2 shows; `docs.yml` runs `check` and `edge` for both projects (two jobs or a matrix over `project`), and the dispatch gains a `project` choice; `release.yml` `publish-docs` calls `publish.yml` once per project; `docs:bundle:check` checks both.
- Section 2 (owner): verify `docs/catalog-opm-docs` is public; publish its first release bundle with the next opm release, or dispatch `mode: release` for the newest `opm-v4.*` tag (its `docs/site/` builds from the tag with `main`'s config). Record the runs.

### 2. opmodel.dev: `serve-docs-from-bundles` section 1 (G3.1)

`bundles.cue` `docs."catalog-opm-docs"` (`repo: "open-platform-model/catalog_opm"`) and `versions."v1.0".tags: {"catalog-opm-docs": "4"}` (the major, as `catalog-line = opm-v4` does today); catalog_opm's `docs/site` leaves the git mounts and `resolve-versions.sh`.

### 3. opm: release-please, then its bundle (no OpenSpec workspace; two PRs)

1. `chore: release with release-please` PR: `release-please-config.json` (one package `.`, `release-type: simple`, `include-v-in-tag: true`, tags `vX.Y.Z`, `draft: false`, `"initial-version": "1.0.0-beta.1"`, and prerelease versioning: `"versioning": "prerelease"`, `"prerelease": true`, `"prerelease-type": "beta"`, so later releases go `1.0.0-beta.2` and onward until the owner drops it for `1.0.0`), `.release-please-manifest.json` without an entry for `.` until the first release (`initial-version` applies only to a package with no release yet, as docs-kit's own `0.1.0` did), `.github/workflows/release.yml` with the release App token as the other repositories have it, `CHANGELOG.md`. The first version is `1.0.0-beta.1` (DESIGN decision 21), so the minor tag `1.0` follows the v1.0 site version. Never a `Release-As:` footer: squash merges use a blank body (workspace `RELEASING.md`), so a footer never reaches `main`; if `initial-version` cannot be used, a temporary `release-as` in the config, removed by the next PR, is the fallback. **Owner**: install the release App on opm, provide `vars.RELEASE_APP_CLIENT_ID` and `secrets.RELEASE_APP_PRIVATE_KEY`, and cover opm with the `tags-immutable` and `tags-create-app-only` rulesets.
2. `ci(docs): publish the opm docs bundle with docs-kit` PR: `docs-kit.cue` (`add-authored-docs` D2), `docs.yml`, `publish-docs` in `release.yml`, `.opm-docs-version`, Taskfile tasks as in phase 2's common list, `AGENTS.md` paragraph.
3. **Owner**: merge the first release PR; verify `docs/opm` is public and its tags verify.

### 4. opmodel.dev: `serve-docs-from-bundles` section 2

`docs.opm` and `versions."v1.0".tags.opm: "1.0"`; opm's `docs/site` (including the `/docs/` landing and `start/_index.md`) leaves the git mounts and `resolve-versions.sh`'s head-of-main row.

### 5. enhancements (no OpenSpec workspace; one PR, after `add-enhancements-bundle` is released)

Before it, the source fixes `add-enhancements-bundle` section 1 lists (spike findings), as ordinary PRs. Then `ci(docs): publish the enhancements bundle with docs-kit`: `docs-kit.cue` (`add-enhancements-bundle` D2), `docs.yml` with `check` on pull requests and `edge` on push to `main` only (no release or revision mode: a section is edge only), `.opm-docs-version`. **Owner**: verify `docs/enhancements` is public.

### 6. opmodel.dev: `serve-docs-from-bundles` section 3

`bundles.cue` `sections: enhancements: {repo: "open-platform-model/enhancements", root: "/enhancements/"}`; the Enhancements section built from `site/.bundles/enhancements/edge/` (pages from `content/`, header data from `data/enhancements.json`); delete `site/enhancements/_content.gotmpl`, `layouts/_partials/opm/enh-clean.html`, `layouts/enhancements/_markup/render-link.html`, the `gen-mounts.sh` enhancements block and `versions.conf`'s `[section "enhancements"]`; copy docs-kit's `link-enhancements-graph` lint fixture.

### 7. opmodel.dev: `retire-git-pipeline` (G3.4)

Delete `resolve-versions.sh`, `materialise.sh`, `gen-lastmod.sh` (site-owned pages keep Hugo's GitInfo), the source-repository rows and floors of `versions.conf` (site versions live in `bundles.cue`), `frozen.conf` (the lock's `--frozen` replaces it), `lint-sources.sh` and its contract copy (every page is linted by `opm-docs lint` on pull). Keep `site/tests/lint/` only if the site still wants it; it no longer binds docs-kit. Done (phase 3, DESIGN.md): the site build reads no git repository except its own.

### 8. docs-kit: `retire-lint-conformance-binding` (G3.5)

A small spec-driven change: REMOVE the `dialect-lint` requirement "The conformance fixture set binds both linters until phase 3" (the fixtures stay as ordinary tests of the Go lint) and rewrite C11's "Agreement with the site's shell lint until phase 3".

## Owner items

- Merge each docs-kit release PR at the gates above, and the `cobradump` release PR.
- Confirm the tag rulesets cover `cobradump/v*` tags as they cover `v*`.
- New GHCR packages, each checked public on first publish: `docs/core`, `docs/library`, `docs/opm-operator`, `docs/cli`, `docs/catalog-opm-docs`, `docs/opm`, `docs/enhancements`.
- opm: release App, secrets, rulesets; first version `1.0.0-beta.1` (DESIGN decision 21), set by `initial-version` in `release-please-config.json`, never by a commit footer.
- The release cascade order of phase 2 (core, library, opm-operator, then cli) and the G2-pins check before the cli release PR merges.

## Decisions made in planning (for the owner to confirm)

Confirmed by the owner on 2026-10-03 and recorded in DESIGN.md: one cutover per product repository (decision 20) and opm's first release `1.0.0-beta.1` (decision 21). The rest stand until the owner says otherwise:

- Version history: field changes appear in the "Changes in X" list, not inline in the spec block; edge is compared with the newest minor; type, default and ref compare only within one docs-kit minor; the lock records the history digest (`add-version-history`).
- Pins live in `manifest.json`, printed by a repository command; pinned bundles resolve through the release tag so docs revisions follow (`generalize-build-assembly`, `pull-docs-placement`).
- `publish.yml` builds in a job without the signing token and publishes in another that runs no repository code (`generalize-build-assembly`).
- Docs bundles declare `owns`; `pull` refuses overlaps per site version and replaces a version whole; cross-bundle `/docs/` links are left to the site's link check.
- `catalog-opm-docs` naming rule; `_versions/` unpack layout; `v<MAJOR>.<MINOR>` site-version names.
- `cobradump` is a nested module released as `cobradump/vX.Y.Z`; the cli's hook is a `hack/` program, not a hidden command; the dump carries raw text.
- The `crd` extractor takes `reconciledBy` from config instead of scanning Go; one page per configured path.
- `go-api` parses without type checking; anchors are Hugo's default heading anchors.
- `pages[].edit` is set only when the same path exists on `main` (no rename following).
- The enhancements section is built producer-side through mechanical transforms, never a relaxed lint; its root is fixed to `/enhancements/`.
- `opm-docs serve` has an embedded skeleton site with live rebuild, and a `--site` mode through opmodel.dev's own preview.
