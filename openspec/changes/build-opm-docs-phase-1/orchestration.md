# Orchestration: phase 1 across docs-kit, catalog_opm and opmodel.dev

Phase 1 is done when the site shows opm 4.4 and edge in the Catalogs tab from pulled bundles, and a docs revision of 4.4.x reaches the site without a catalog release (`DESIGN.md`, "Phase 1"). Three repositories carry it, plus a one-line cli docs fix. This file is enough to plan each sibling change without reading docs-kit's code: the contracts are `design.md` C1 to C11 (after archive, `docs/contracts.md`), and each sibling's list below is exhaustive for phase 1.

## Changes

| Repo | Change | Schema | Delivery |
|---|---|---|---|
| docs-kit | `build-opm-docs-phase-1` (this change) | spec-driven | one PR |
| catalog_opm | `publish-docs-bundle` | catalog-change (no specs) | one PR per section: `Delivery: one PR per section (opmodel.dev needs the published bundles after section 1)` |
| opmodel.dev | `add-catalogs-tab` | docs-site-change (no specs) | one PR |
| cli | none (a `docs:` PR, no OpenSpec change) | n/a | one PR |

## Sequence and merge order

```text
 docs-kit                     catalog_opm                    opmodel.dev                 cli
 ───────────────────────────  ─────────────────────────────  ──────────────────────────  ──────────────────
 1 build-opm-docs-phase-1 PR
   (owner runs spike.yml
    mid-PR, section 1)
   merge
 2 release PR 0.1.0 merge ──► 3 publish-docs-bundle §1 PR
                                 (uses publish.yml@v0.1.0)
                                 merge: edge publishes
                              4 OWNER: make packages public,
                                 dispatch the backfills ─────► 5 add-catalogs-tab PR
                                                                  (pulls real bundles)
                                                                  merge ─────────────────► 6 link fix PR
                                                                                              merge
                              7 publish-docs-bundle §2 PR ◄──────────────────────────────────┘
                                 (refgen retired) merge
                              8 docs revision test ─────────► 9 site shows 4.4.5.1
```

1. **docs-kit `build-opm-docs-phase-1`.** One PR. Section 1 is a gate: the owner runs `spike.yml` from the PR branch (tasks.md 1.3) and nothing past section 1 starts until its findings are recorded. Merge when every section is done and archived.
2. **docs-kit release `v0.1.0`.** release-please opens the release PR after step 1's merge; the owner merges it; the draft-first release publishes the binaries. Nothing downstream may reference docs-kit before this tag exists.
3. **catalog_opm `publish-docs-bundle`, section 1 (adopt).** Its `check` jobs on the PR are the first end-to-end run of `publish.yml`. On merge, the `Docs` workflow publishes `edge` for both projects; the first push creates the two GHCR packages, private.
4. **Owner, in catalog_opm.** Make `docs/catalog-opm` and `docs/catalog-k8s` public. Dispatch the backfills for the current releases, so the tab has its first minor (DESIGN decision 8): `gh workflow run docs.yml -R open-platform-model/catalog_opm -f project=catalog-opm -f mode=release -f tag=opm-v4.4.5` and the same for `catalog-k8s` with its newest `k8s-v*` tag (today `k8s-v1.0.0-beta.2`). Check: anonymous `opm-docs pull` with a scratch `bundles.cue` (C7) resolves `4.4`, `edge`, `1.0` and verifies all four.
5. **opmodel.dev `add-catalogs-tab`.** Built and tested against fixture bundles, merged once step 4 holds, so its first `main` build pulls real ones. From this merge until step 7 the catalog member pages exist twice (Reference, from git, and the Catalogs tab); keep the window short.
6. **cli docs link fix.** Needs step 5 (the site lint accepts `/catalogs/` links) and must merge before step 7 (which deletes the page it links today).
7. **catalog_opm `publish-docs-bundle`, section 2 (retire refgen).** Removes the generator and the committed pages; the Reference copies disappear from the next site build.
8. **Done check.** Land a doc-comment fix on catalog_opm `main` (any member), then `gh workflow run docs.yml -R open-platform-model/catalog_opm -f project=catalog-opm -f mode=revision -f tag=opm-v4.4.5 -f fix=<sha>`. Expected: `4.4.5.1` published, `4.4.5` and (when 4.4.5 is still the newest 4.4) `4.4` moved.
9. **Done.** The next site build shows the fix under `/catalogs/opm/4.4/` with no catalog release, and the build stamp names revision 1.

If catalog_opm releases a new opm version between steps 3 and 9, its `publish-docs` job publishes `<version>.0` by itself; step 8 then revises that newest 4.4.x instead.

## What every sibling change must reference

- The contracts: `https://github.com/open-platform-model/docs-kit/blob/main/docs/contracts.md` once step 1 has merged (until then, this change's `design.md` on the PR branch), cited by contract number (`docs-kit C5`).
- The pinned release: `open-platform-model/docs-kit/.github/workflows/publish.yml@v0.1.0` and `opm-docs` 0.1.0. A tag, never a SHA (C5, C9).
- DESIGN decisions by number (`docs-kit DESIGN decision 9`); docs-kit has no enhancement entry, so no `enhancement.yaml` and no delivery log.

## catalog_opm: `publish-docs-bundle`

Proposal must state: two sections, `Delivery: one PR per section (opmodel.dev needs the published bundles after section 1)`; release class `ci` for both commits (no catalog release: no published member changes); no member moves `apiVersion`; downstream consumers are opmodel.dev (pulls the bundles) and cli (one docs link, step 6); Before/After shows `docs-kit.cue` (Before: none) and the removed `tools/refgen` task block.

### Section 1: publish bundles with docs-kit

1. Add `docs-kit.cue` at the repository root exactly as `design.md` C6 shows (projects `catalog-opm` and `catalog-k8s`).
2. Add `docs/catalogs/opm/_index.md`: the content of `docs/site/reference/catalog-contract.md`, as a section landing: front matter `title`, `description`, no `type`, no `weight`; body unchanged except links written in the C8 forms. Its links to `/docs/reference/cli/` and `/docs/reference/registry-namespaces/` stay as they are (the site resolves `/docs/` from a tab page in its default version).
3. Add `docs/catalogs/k8s/_index.md`: title "Raw Kubernetes resources", the description and the authored intro of `docs/site/reference/kubernetes-resources.md`, ending with a link to the table, written as the major alias `/catalogs/k8s/1/resources/`. The `markdown` source pins an alias link into its own catalog to the build's segment (C8), so the 1.0 bundle links `/catalogs/k8s/1.0/resources/`. The opm landing writes its own-catalog links the same way.
4. Add `.github/workflows/docs.yml` (every job calls `open-platform-model/docs-kit/.github/workflows/publish.yml@v0.1.0`; `permissions: {}` at the top, per job as C5's table):
   - `check`: on `pull_request`, matrix `project: [catalog-opm, catalog-k8s]`, `mode: check`.
   - `edge`: on `push` to `main`, same matrix, `mode: edge`.
   - `dispatch`: on `workflow_dispatch` with inputs `project` (choice of the two), `mode` (choice `release`, `revision`), `tag` (string, required), `fix` (string, default empty); passes them through.
5. `release.yml`: add a job `publish-docs` with `needs: [release-please, publish-cue]`, `if: needs.release-please.outputs.releases_created == 'true'`, matrix `module: ${{ fromJSON(needs.release-please.outputs.paths_released) }}`, calling `publish.yml@v0.1.0` with `project: catalog-${{ matrix.module }}`, `mode: release`, `tag: ${{ needs.release-please.outputs[format('{0}_tag_name', matrix.module)] }}`, and job permissions `contents: read`, `packages: write`, `id-token: write`. It runs on the push to `main` that merged the release PR, which C9 and DESIGN decision 9 need. Do not rely on `release: published`.
6. Taskfile: `docs:bundle` (`go run github.com/open-platform-model/docs-kit/cmd/opm-docs@v0.1.0 build`, output `out/`, gitignored) and `docs:bundle:check` (the same with `check`); add `docs:bundle:check` to `task check`. Keep `generate:reference*` and `test:refgen` until section 2. The Taskfile version and the workflow ref name the same docs-kit release and move together in one PR.
7. `.gitignore`: `/out/`.
8. `AGENTS.md`: a "Docs bundles" paragraph (what publishes, when, how to preview with `task docs:bundle`, how to run a docs revision, that `docs/catalogs/` holds bundle-only pages and `docs/site/` holds site-version pages).
9. Gate: `task check` green; the PR's `check` jobs green (the first real run of `publish.yml`). Commit `ci(docs): publish the catalog docs bundles with docs-kit`.

### Section 2: retire tools/refgen (after opmodel.dev `add-catalogs-tab` and the cli link fix have merged)

1. Delete `tools/refgen/`, `docs/site/reference/catalog-members/`, `docs/site/reference/kubernetes-resources.md` and `docs/site/reference/catalog-contract.md`.
2. Taskfile: remove `generate:reference`, `generate:reference:check`, `test:refgen` and their entries in `check`, and the "Site reference" comments.
3. `ci.yml`: remove "Test the reference generator", "Verify the generated site reference is up to date" and "Setup Go" (Go is no longer needed: `docs:bundle:check` is not in `ci.yml`; the `Docs` workflow's `check` job covers PRs). `branch-publish.yml`: remove its "Setup Go".
4. `release.yml`, job `release-please`: in "Advance identity.Version on the release PRs" remove `task generate:reference`, `docs/site/reference` from `git add` and the `git status --porcelain -- docs/site/reference` test; remove "Setup Go" and "Install Task" if nothing else in the job uses them; keep the GHCR login if `opm catalog version set` needs it.
5. `AGENTS.md` (the layout tree, the Dependencies Go bullet, the `tools/refgen` paragraph, the check list) and `openspec/config.yaml` context ("The only Go code is `tools/refgen/`", Principle II's `task generate:reference` sentence): rewrite to say the reference is published as docs bundles by docs-kit.
6. `vet:descriptions` stays; reword its message if it names the site reference.
7. Gate: `task check` green. Commit `ci(docs): retire tools/refgen and its committed pages`.

Durable decisions to land in catalog_opm `AGENTS.md`: where bundle-only pages live (`docs/catalogs/`), how to preview and how to run a docs revision.

## opmodel.dev: `add-catalogs-tab`

Proposal must state: published URLs added (`/catalogs/opm/<MAJOR.MINOR>/`, `/catalogs/opm/edge/`, `/catalogs/k8s/...`, the aliases); the site gains an input (bundles pulled from GHCR, a pinned `opm-docs`); no source repo's `docs/site` changes except the cli follow-up (step 6) and catalog_opm section 2, both named as follow-ups. Pattern: the Enhancements section (unversioned, adapter-built, mounted into the default version only).

Must do:

1. **Tool pin.** A repo-root `.opm-docs-version` (one line, `v0.1.0`), and `task tools:opm-docs`, which downloads that release's `opm-docs_<v>_<os>_<arch>.tar.gz` and `checksums.txt`, checks the SHA-256 and installs to `site/.bin/opm-docs` (gitignored). Host only; the Docker images do not need it.
2. **Pull config.** `site/bundles.cue` exactly as C7 shows (two tabs, both owned by `open-platform-model/catalog_opm`, `from: "4.4"` and `from: "1.0"`).
3. **The network step.** `versions:prepare` "never fetches", so add `task bundles:pull` (`opm-docs pull --config site/bundles.cue --out site/.bundles --lock site/.bundles/lock.json`), run by `task versions:fetch` and by CI before the build; `versions:prepare` then fails, naming `task bundles:pull`, when `site/.bundles/lock.json` is missing or its `config` digest does not match `site/bundles.cue`. A local preview of unpublished catalog pages passes `--local catalog-opm=<catalog_opm>/out/catalog-opm` through an `OPM_BUNDLES_LOCAL` variable. `site/.bundles/` is gitignored and `task clean` removes it.
4. **Build stamp.** `gen-stamp.sh` adds `sections.catalogs`: per lock entry, project, segment, version, revision, digest, commit (C7 fields). A pull or lint failure fails the build naming project, tag and digest.
5. **Mounts and adapter.** For each lock entry, mount `site/.bundles/<dir>/content` at `assets/catalogs/<project>/<segment>` into the default version only; a content adapter `site/catalogs/_content.gotmpl` adds every page with `url` `<root><segment>/<page URL>/` (C8), `lastmod` and the source link from `manifest.json` `pages[]` (GitHub blob at `source.commit`, or "View source" only for generated pages), `sitemap.disable` per the indexing rule, `params.catalog` (project, segment, version, edge). A data file `data/opm/catalogs.json`, written from the lock (minors newest first, newest per major, edge), feeds the adapter, the switcher, redirects, noindex and checks. Factor the Enhancements plumbing in `build-all.sh` and `serve.sh` rather than copy it a third time.
6. **Navigation.** A Catalogs menu entry (only in a build with bundles), a `/catalogs/` section page listing the catalogs, a sidebar root per catalog minor, `layouts/catalogs/` copies of the Enhancements wrappers, the navbar active rule.
7. **Switcher.** A version menu on every catalog page: newest minor first, `edge` labelled "main (unreleased)"; the target is the same page path in the chosen minor when it exists, else its nearest existing parent, else the minor's landing.
8. **Aliases.** `/catalogs/<name>/` and `/catalogs/<name>/<MAJOR>/` to the newest minor (of that major), and `/catalogs/<name>/<MAJOR>/<path>/` to the same path there: `_redirects` lines plus meta-refresh stub pages for hosts that ignore `_redirects` (as `/latest/` does today). New `_redirects` lines must not match the `^/latest/\*` pattern the QA scripts parse.
9. **Indexing.** Only the newest minor of each major is indexed and in `llms.txt`; older minors and `edge` carry `noindex` and are left out of `llms.txt`. Pagefind indexes each minor separately; search from a catalog page stays in that minor.
10. **Links.** The global link hook accepts `/catalogs/` links (C8), resolves a `/docs/` link written on a catalog page in the default version, and the link checks know the alias stubs.
11. **Dialect.** `site/scripts/lint-sources.sh` accepts the C11 `/catalogs/` forms in docs mode (major segment only); update its byte-identical contract copy and add lint fixtures. The Go lint in `opm-docs` is the reference: the site's fixtures and docs-kit's `internal/dialect/testdata` must agree.
12. **Reference.** `content/docs/reference/_index.md` stops describing catalog members; Reference loses them when catalog_opm section 2 lands. Whether old URLs (`/v1.0/docs/reference/catalog-members/...`, `catalog-contract`, `kubernetes-resources`) redirect to the alias forms is an owner decision; if yes it is a small follow-up after catalog_opm section 2, because before it the old pages still exist.
13. **Checks and tests.** Fixture bundles under `site/tests/fixtures/bundles/catalog-opm/{4.4,4.5,edge}/` and `catalog-k8s/{1.0,edge}/` (plain trees with `manifest.json`; the tests use `--local`, never the registry), and failing cases for: a catalog page with no index (Q2), a stray file, a broken catalog link, catalogs without bundles, the redirects and stubs, a lock that disagrees with `bundles.cue`. QA screenshots of a catalog page, the switcher open and phone width.

Durable decisions to land in opmodel.dev `AGENTS.md`/`README.md`: the Catalogs section and its pull step, the indexing rule, the alias rule, where the tool pin lives.

## cli: link fix (no OpenSpec change)

`docs/site/reference/registry-namespaces.md`: the link `[The Catalog Contract](/docs/reference/catalog-contract/)` becomes `[The Catalog Contract](/catalogs/opm/4/)`. Commit `docs(reference): link the catalog contract in the Catalogs tab`. Only after opmodel.dev `add-catalogs-tab` has merged; before catalog_opm section 2.

## Owner setup

Before docs-kit's implementation PR:

- Install the release App (`opm-release-please`) on docs-kit and make `vars.RELEASE_APP_CLIENT_ID` and `secrets.RELEASE_APP_PRIVATE_KEY` available to it, as for the other released repos.
- Add docs-kit to the org rulesets `tags-immutable` and `tags-create-app-only` (C5 relies on immutable docs-kit tags), protect `main` (PRs only, required CI checks), enable immutable releases, and set squash merges to the PR title with a blank body, as in the other repos.
- Run `spike.yml` from the PR branch (tasks.md 1.3); delete or keep private the `docs/spike` package afterwards (1.7).

After catalog_opm section 1 merges:

- Make the packages `ghcr.io/open-platform-model/docs/catalog-opm` and `docs/catalog-k8s` public (Package settings, Danger zone). Confirm each is linked to `catalog_opm` (it is, through `org.opencontainers.image.source`).
- Dispatch the two backfills (step 4).
- Optionally make catalog_opm's `Docs / check` jobs required on `main`.

Workspace follow-ups (workspace repo, owner or a workspace change):

- Root `AGENTS.md`, "Release Tags Are Immutable", Registry tags bullet: name the docs bundles' moving tags (`<version>`, `<MAJOR.MINOR>`, `<MAJOR>`, `edge` under `ghcr.io/open-platform-model/docs/*`) as mutable by design; full tags (`<version>.<revision>`) are never overwritten.
- Root `AGENTS.md` Repos table: docs-kit's row and commands once the tool exists; the scope list of "Release Tags Are Immutable" gains docs-kit.
- `RELEASING.md`: whether a docs-kit release cascades into the callers' `publish.yml@vX` refs (and catalog_opm's Taskfile version) is undecided; until then callers bump by hand, both pins in one PR.

## Open points for the owner

1. **The k8s bundle** (design R2): publish `catalog-k8s` in phase 1 (this plan), or keep refgen for the k8s table until phase 2.
2. **Sitemap and edge search** (opmodel.dev): does the sitemap list the newest minors (no enhancement is listed today), and does `edge` get a Pagefind index and alias stubs?
