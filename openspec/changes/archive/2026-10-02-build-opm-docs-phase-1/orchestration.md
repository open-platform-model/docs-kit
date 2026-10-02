# Orchestration: phase 1 across docs-kit, catalog_opm and opmodel.dev

Phase 1, as planned here, is done when the site shows opm 4.4 and edge in the Catalogs tab from pulled, verified bundles. The other half of `DESIGN.md`'s criterion, a docs revision of 4.4.x reaching the site without a catalog release, belongs to the docs-kit follow-up change `add-docs-revisions` (see the end of this file). Three repositories carry phase 1, plus a two-link cli docs fix. This file is enough to plan each sibling change without reading docs-kit's code: the contracts are `design.md` C1 to C12, which move to `docs/contracts.md` (same numbers) before this change is archived, and each sibling's list below is exhaustive for phase 1.

## Changes

| Repo | Change | Schema | Delivery |
|---|---|---|---|
| docs-kit | `build-opm-docs-phase-1` (this change) | spec-driven | one PR |
| catalog_opm | `publish-docs-bundle` | catalog-change (no specs) | one PR per section: `Delivery: one PR per section (opmodel.dev needs the published bundles after section 2)` |
| opmodel.dev | `add-catalogs-tab` | docs-site-change (no specs) | one PR |
| cli | none (a `docs:` PR, no OpenSpec change) | n/a | one PR |
| docs-kit | `add-docs-revisions` (follow-up; starts after `v0.1.0`) | spec-driven | one PR |

## Sequence and merge order

```text
 docs-kit                     catalog_opm                    opmodel.dev                 cli
 ───────────────────────────  ─────────────────────────────  ──────────────────────────  ──────────────────
 1 build-opm-docs-phase-1 PR
   (spike: push to spike/ghcr)
   merge
 2 OWNER: hard gate, then
   optional v0.1.0-rc.N ─────► (draft §1 PR, check mode)
   release PR 0.1.0 merge ───► 3 publish-docs-bundle §1 PR
                                 (publish.yml@v0.1.0) merge:
                                 edge publishes
                              4 publish-docs-bundle §2
                                 (owner go-live: package
                                  verify public, backfill) ─────────► 5 add-catalogs-tab PR
                                                                  (pulls real bundles;
                                                                   stops publishing the
                                                                   old Reference pages)
                                                                  merge ──────────┬──────► 6 link fix PR
                                                                                  │          merge
                              7 publish-docs-bundle §3 PR ◄───────────────────────┘
                                 (refgen retired; waits for
                                  the k8s-catalog removal on
                                  catalog_opm main) merge
                              (6 and 7 in either order)
```

1. **docs-kit `build-opm-docs-phase-1`.** One PR. Section 1 is a gate: the spike commit is pushed to a `spike/ghcr` branch, which triggers `spike.yml` (a workflow on a non-default branch cannot be dispatched), and nothing past section 1 starts until its findings are recorded (tasks.md 1.3 to 1.6). Merge when every section is done and archived.
2. **docs-kit release `v0.1.0`.** First the **hard gate** (owner, before any docs-kit tag exists): docs-kit is covered by the org rulesets `tags-immutable` and `tags-create-app-only`, and immutable releases are enabled; check read-only with `gh api "repos/open-platform-model/docs-kit/rulesets?includes_parents=true"` and `gh api repos/open-platform-model/docs-kit/immutable-releases`. Callers reference `publish.yml` by tag (owner decision, 2026-10-02), which is safe only while docs-kit tags cannot move. Then, optionally, a **prerelease dry run**: the owner cuts `v0.1.0-rc.N` (a one-shot `Release-As: 0.1.0-rc.1` footer on a docs-kit commit), and catalog_opm's section-1 PR, still a draft, pins `publish.yml@v0.1.0-rc.N` and runs only the `check` job; nothing publishes from an rc. Then release-please's `0.1.0` release PR is merged; the draft-first release attaches the binaries and `checksums.txt` and publishes. Confirm four archives and `checksums.txt` on the release. Nothing downstream merges a reference to docs-kit before `v0.1.0` exists.
3. **catalog_opm `publish-docs-bundle`, section 1 (publish).** Pins `publish.yml@v0.1.0`; its `check` job on the PR is the first end-to-end run of `publish.yml` at a release. On merge, the `Docs` workflow publishes `edge`; the first push creates the GHCR package `docs/catalog-opm`, linked to catalog_opm through `org.opencontainers.image.source` and, per the spike, public like the repository it links to.
4. **catalog_opm `publish-docs-bundle`, section 2 (owner go-live).** The owner verifies `docs/catalog-opm` is public (the phase-1 spike found a package created from Actions with `org.opencontainers.image.source` takes the visibility of the public repository it links to, so it should already be public; if not, make it public in Package settings) and dispatches the backfill for the current release, so the tab has its first minor (DESIGN decision 8; `release` mode reads `main`'s `docs-kit.cue` because the tag has none, but extracts the `opm/` module and the `docs/catalogs/opm` dir from the tag; the tag has no such dir, so `4.4.5.0` gets the generated landing only, C5): `gh workflow run docs.yml -R open-platform-model/catalog_opm -f mode=release -f tag=opm-v4.4.5` (or the newest `opm-v4.4.*` tag by then). Check: anonymous `opm-docs pull` with a scratch `bundles.cue` (C7) resolves `4.4` and `edge` and verifies both. The section's commit records the run URLs and the result in the change.
5. **opmodel.dev `add-catalogs-tab`.** Built and tested against fixture bundles, merged once step 4 holds, so its first `main` build pulls real ones. No double publish (supervisor decision, 2026-10-02): from this merge the site stops publishing catalog_opm's `docs/site/reference/catalog-members/**` and `docs/site/reference/catalog-contract.md` (excluded from the mount), and its link hook maps exactly two legacy targets, `/docs/reference/catalog-members/` and `/docs/reference/catalog-contract/`, to `/catalogs/opm/4/`. Deep links to individual member pages are not mapped; none exist outside the excluded pages.
6. **cli docs link fix.** Needs step 5 (the site lint accepts `/catalogs/` links). Step 5 already maps the contract URL, so it no longer has to land before step 7; steps 6 and 7 merge in either order.
7. **catalog_opm `publish-docs-bundle`, section 3 (retire refgen).** External dependency: it starts only after the removal of the k8s catalog (`k8s/`, owned by another session) has landed on catalog_opm `main`, because until then refgen still writes the k8s table and no bundle replaces it. It also waits for step 5 (not for step 6). It removes the generator and the committed pages the site already ignores. Phase 1 is done when step 5's build shows the Catalogs tab and no catalog pages under Reference; steps 6 and 7 clean up the sources.

If catalog_opm releases a new opm version after step 3, its `publish-docs` job publishes `<version>.0` by itself.

## What every sibling change must reference

- The contracts: `https://github.com/open-platform-model/docs-kit/blob/main/docs/contracts.md` once step 1 has merged (until then, this change's `design.md` on the PR branch), cited by contract number (`docs-kit C5`).
- The pinned release: `open-platform-model/docs-kit/.github/workflows/publish.yml@v0.1.0` and `opm-docs` 0.1.0. Referenced by release tag, never by SHA (owner decision, 2026-10-02; C5, C9): a deliberate exception to the org's SHA-pinning convention, so a caller's workflow comment should say so.
- DESIGN decisions by number (`docs-kit DESIGN decision 9`); docs-kit has no enhancement entry, so no `enhancement.yaml` and no delivery log.

## catalog_opm: `publish-docs-bundle`

Proposal must state: three sections, `Delivery: one PR per section (opmodel.dev needs the published bundles after section 2)`; release classes `ci` (section 1), `docs(openspec)` (section 2, the owner go-live record) and `ci` (section 3), so no catalog release (no published member changes); no member moves `apiVersion`; downstream consumers are opmodel.dev (pulls the bundles; already ignores the old pages, step 5) and cli (two docs links, step 6, in either order with section 3); section 3 depends on the k8s-catalog removal having landed on `main` (an external dependency owned by another session); Before/After shows `docs-kit.cue` (Before: none) and the removed `tools/refgen` task block. Only the opm catalog gets a bundle; nothing is planned for `k8s/`.

### Section 1: publish the bundle with docs-kit

1. Add `docs-kit.cue` at the repository root exactly as `design.md` C6 shows (one project, `catalog-opm`).
2. Add `docs/catalogs/opm/_index.md`: the content of `docs/site/reference/catalog-contract.md`, as a section landing (the renderer appends the generated `## Catalog members` block after it, so the page must not hold that heading; C8): front matter `title`, `description`, no `type`, no `weight`; body unchanged except links written in the C8 forms. Its links to `/docs/reference/cli/` and `/docs/reference/registry-namespaces/` stay as they are (the site resolves `/docs/` from a tab page in its default version). A link into the catalog's own pages is written as the major alias (`/catalogs/opm/4/traits/`); the `markdown` source pins it to the build's segment (C8).
3. Add `.github/workflows/docs.yml` (every job calls `open-platform-model/docs-kit/.github/workflows/publish.yml@v0.1.0`; `permissions: {}` at the top, per job as C5's table):
   - `check`: on `pull_request`, `project: catalog-opm`, `mode: check`.
   - `edge`: on `push` to `main`, `project: catalog-opm`, `mode: edge`.
   - `dispatch`: on `workflow_dispatch` with inputs `mode` (choice: `release` only, until `add-docs-revisions` adds `revision`) and `tag` (string, required); passes them through with `project: catalog-opm`.
4. `release.yml`: add a job `publish-docs` with `needs: [release-please, publish-cue]`, `if: needs.release-please.outputs.opm_tag_name != ''` (only an opm release publishes docs), calling `publish.yml@v0.1.0` with `project: catalog-opm`, `mode: release`, `tag: ${{ needs.release-please.outputs.opm_tag_name }}`, and job permissions `contents: read`, `packages: write`, `id-token: write`. It runs on the push to `main` that merged the release PR, which C9 and DESIGN decision 9 need. Do not rely on `release: published`.
5. Tool pin and Taskfile (C12; never `go run` the tool): a repo-root `.opm-docs-version` (`v0.1.0`); `tools:opm-docs`, which downloads that release's archive for the host's os and arch and `checksums.txt`, verifies the line with `sha256sum -c` and installs `.bin/opm-docs`; `docs:bundle` (`.bin/opm-docs build`, output `out/`) and `docs:bundle:check` (`.bin/opm-docs check`), both depending on `tools:opm-docs`; add `docs:bundle:check` to `task check`. Keep `generate:reference*` and `test:refgen` until section 3. `.opm-docs-version` and the `publish.yml@` ref name the same docs-kit release and move together in one PR.
6. `.gitignore`: `/out/` and `/.bin/`.
7. `AGENTS.md`: a "Docs bundles" paragraph (what publishes, when, how to preview with `task docs:bundle`, how to backfill a release with the dispatch, that `docs/catalogs/` holds bundle-only pages and `docs/site/` holds site-version pages).
8. Gate: `task check` green; the PR's `check` job green (the first real run of `publish.yml`). Commit `ci(docs): publish the opm catalog docs bundle with docs-kit`.

### Section 2: owner go-live (after section 1 merges)

1. **OWNER**: verify `ghcr.io/open-platform-model/docs/catalog-opm` is public and linked to `catalog_opm` (spike finding, design.md C2: a new package inherits the public repository's visibility; make it public in Package settings, Danger zone, only if it is not).
2. **OWNER**: dispatch the backfill (orchestration step 4) and confirm `4.4.<n>.0`, `4.4.<n>`, `4.4` and `4` exist and verify (`cosign verify` with the C9 flags, or an anonymous `opm-docs pull` with a scratch `bundles.cue`).
3. Record the run URLs and results in the change (`design.md` or a note beside `tasks.md`). Gate: `task check` green. Commit `docs(openspec): record the docs bundle go-live`.

### Section 3: retire tools/refgen (after the k8s-catalog removal has landed on catalog_opm `main`, and after opmodel.dev `add-catalogs-tab` has merged; the cli link fix may land before or after)

Precondition, checked by the planner and again before the PR: catalog_opm `main` has no `k8s/` module (the removal change has merged). If it has not, stop; this section does not plan around the k8s catalog.

1. Delete `tools/refgen/`, `docs/site/reference/catalog-members/` and `docs/site/reference/catalog-contract.md` (and `docs/site/reference/kubernetes-resources.md` if the k8s removal left it).
2. Taskfile: remove `generate:reference`, `generate:reference:check`, `test:refgen` and their entries in `check`, and the "Site reference" comments.
3. `ci.yml`: remove "Test the reference generator", "Verify the generated site reference is up to date" and "Setup Go" (Go is no longer needed: `docs:bundle:check` is not in `ci.yml`; the `Docs` workflow's `check` job covers PRs). `branch-publish.yml`: remove its "Setup Go".
4. `release.yml`, job `release-please`: in "Advance identity.Version on the release PRs" (its module loop as the k8s removal left it) remove `task generate:reference`, `docs/site/reference` from `git add` and the `git status --porcelain -- docs/site/reference` test; remove "Setup Go" and "Install Task" if nothing else in the job uses them; keep the GHCR login if `opm catalog version set` needs it.
5. `AGENTS.md` (the layout tree, the Dependencies Go bullet, the `tools/refgen` paragraph, the check list) and `openspec/config.yaml` context ("The only Go code is `tools/refgen/`", Principle II's `task generate:reference` sentence): rewrite to say the reference is published as docs bundles by docs-kit.
6. `vet:descriptions` stays; reword its message if it names the site reference.
7. Gate: `task check` green. Commit `ci(docs): retire tools/refgen and its committed pages`.

Durable decisions to land in catalog_opm `AGENTS.md`: where bundle-only pages live (`docs/catalogs/`), how to preview and how to backfill a release.

## opmodel.dev: `add-catalogs-tab`

Proposal must state: published URLs added (`/catalogs/opm/<MAJOR.MINOR>/`, `/catalogs/opm/edge/`, the aliases); the site gains an input (bundles pulled from GHCR, a pinned `opm-docs`); no source repo's `docs/site` changes except the cli follow-up (step 6) and catalog_opm section 3, both named as follow-ups. Pattern: the Enhancements section (unversioned, adapter-built, mounted into the default version only).

Must do:

1. **Tool pin** (C12, image pattern; supervisor decision, 2026-10-02). `site/Dockerfile` downloads the `opm-docs_<version>_linux_amd64.tar.gz` of the pinned release and checks it against a SHA-256 written in the Dockerfile, as it pins Hugo and Pagefind; that SHA-256 is the archive's line in the release's `checksums.txt`. `opm-docs pull` runs inside the build image, with network on for that step only; every other step keeps `--network none`. No host install and no `.opm-docs-version` are needed; bumping the tool is a Dockerfile edit (version and SHA-256 together).
2. **Pull config.** `site/bundles.cue` exactly as C7 shows (one tab, `catalog-opm`, owned by `open-platform-model/catalog_opm`, `from: "4.4"`).
3. **The network step.** `versions:prepare` "never fetches", so add `task bundles:pull` (`opm-docs pull --config site/bundles.cue --out site/.bundles --lock site/.bundles/lock.json`, in the build image with network on for this step only), run by `task versions:fetch` and by CI before the build; `versions:prepare` then fails, naming `task bundles:pull`, when `site/.bundles/lock.json` is missing or its `config` digest does not match `site/bundles.cue`. The lock and config check runs in the site's build image (`site/scripts/sections.sh`), like the other section checks, not on the host. A local preview of unpublished catalog pages passes `--local catalog-opm@edge=<catalog_opm>/out/catalog-opm` (repeatable, one per segment; C7) through an `OPM_BUNDLES_LOCAL` variable; an all-local pull needs no network. `site/.bundles/` is gitignored and `task clean` removes it.
4. **Build stamp.** `gen-stamp.sh` adds `sections.catalogs`: per lock entry, project, segment, version, revision, digest, commit (C7 fields). A pull or lint failure fails the build naming project, tag and digest.
5. **Mounts and adapter.** One filtered mount of `site/.bundles` at `assets/bundles` into the default version only (`files` limited to `*/*/manifest.json` and `*/*/content/**`), whatever the number of segments; a content adapter `site/catalogs/_content.gotmpl` adds every page with `url` `<root><segment>/<page URL>/` (C8), `lastmod` and the source link from `manifest.json` `pages[]` (GitHub blob at `source.commit`, or "View source" only for generated pages), `sitemap.disable` per the indexing rule, `params.catalog` (project, segment, version, edge). A data file `data/opm/catalogs.json`, written from the lock (minors newest first, newest per major, edge), feeds the adapter, the switcher, redirects, noindex and checks. Factor the Enhancements plumbing in `build-all.sh` and `serve.sh` rather than copy it a third time.
6. **Navigation.** A Catalogs menu entry (only in a build with bundles), a `/catalogs/` section page listing the catalogs (one in phase 1), a sidebar root per catalog minor, `layouts/catalogs/` copies of the Enhancements wrappers, the navbar active rule.
7. **Switcher.** A version menu on every catalog page: newest minor first, `edge` labelled "main (unreleased)"; the target is the same page path in the chosen minor when it exists, else its nearest existing parent, else the minor's landing.
8. **Aliases.** `/catalogs/<name>/` and `/catalogs/<name>/<MAJOR>/` to the newest minor (of that major), and `/catalogs/<name>/<MAJOR>/<path>/` to the same path there: `_redirects` lines plus meta-refresh stub pages for hosts that ignore `_redirects` (as `/latest/` does today). New `_redirects` lines must not match the `^/latest/\*` pattern the QA scripts parse. `edge` gets no alias stubs (supervisor decision, design.md "Site decisions").
9. **Indexing.** Only the newest minor of each major is indexed, in `llms.txt` and in the sitemap; older minors and `edge` carry `noindex` and are left out of `llms.txt` and the sitemap (supervisor decision). Pagefind indexes each minor separately, and `edge` gets its own index too; search from a catalog page stays in that minor.
10. **Links.** The global link hook accepts `/catalogs/` links (C8), resolves a `/docs/` link written on a catalog page in the default version, and the link checks know the alias stubs.
11. **Dialect.** `site/scripts/lint-sources.sh` accepts the C11 `/catalogs/` forms in docs mode (the bare tab root and major segments only); update its byte-identical contract copy and copy docs-kit's `/catalogs/` fixtures into `site/tests/lint/`. Until phase 3 retires the shell lint, it and `opm-docs lint` agree through one conformance fixture set (C11): its initial content is docs-kit's copy of `site/tests/lint/`; docs-kit is the source of every later fixture; a rule change lands in docs-kit first, and this repo re-syncs the fixtures and the shell lint in the PR that bumps its pinned `opm-docs` to that release.
12. **Reference.** Reference stops publishing catalog pages at this change's merge: the catalog_opm mount excludes `reference/catalog-members/**` and `reference/catalog-contract.md`, and `content/docs/reference/_index.md` stops describing catalog members. The link hook keeps a legacy map of exactly two URLs, `/docs/reference/catalog-members/` and `/docs/reference/catalog-contract/`, both to `/catalogs/opm/4/`; no deep member links are mapped. No redirects are served from the old `/<version>/docs/reference/catalog-*` URLs (supervisor decision, 2026-10-02): the site is still interim and `noindex`, so no reader or search index depends on them.
13. **Checks and tests.** Fixture bundles under `site/tests/fixtures/bundles/catalog-opm/{4.4,4.5,edge}/` (plain trees with `manifest.json`; the tests pass them with `--local catalog-opm@4.4=... --local catalog-opm@4.5=... --local catalog-opm@edge=...`, never the registry, and run with no network), the lint conformance set kept in agreement with docs-kit's (design.md C11), and failing cases for: a catalog page with no index (Q2), a stray file, a broken catalog link, catalogs without bundles, the redirects and stubs, a lock that disagrees with `bundles.cue`. QA screenshots of a catalog page, the switcher open and phone width.

14. **Phase 1b reservation.** Read nothing else from `site/.bundles/`; `<project>/history.json` will be written by `opm-docs pull` in phase 1b (design.md "Site decisions"), and the site will read it, never compute history itself.

Durable decisions to land in opmodel.dev `AGENTS.md`/`README.md`: the Catalogs section and its pull step, the indexing and sitemap rule, the alias rule, where the tool pin lives.

## cli: link fix (no OpenSpec change)

`docs/site/reference/registry-namespaces.md` links the contract page twice: line 19 (the `opmodel.dev/catalogs/<name>` table row) and line 38 (`[The Catalog Contract](/docs/reference/catalog-contract/)`). Both become `/catalogs/opm/4/`. Commit `docs(reference): link the catalog contract in the Catalogs tab`. Only after opmodel.dev `add-catalogs-tab` has merged; in either order with catalog_opm section 3.

## Owner setup

Before docs-kit's implementation PR:

- Install the release App (`opm-release-please`) on docs-kit and make `vars.RELEASE_APP_CLIENT_ID` and `secrets.RELEASE_APP_PRIVATE_KEY` available to it, as for the other released repos.
- **Hard gate before any docs-kit tag (step 2):** add docs-kit to the org rulesets `tags-immutable` and `tags-create-app-only` and enable immutable releases on docs-kit. Callers reference `publish.yml` by tag (owner decision, 2026-10-02), which is safe only while docs-kit tags cannot move. Also protect `main` (PRs only, required CI checks) and set squash merges to the PR title with a blank body, as in the other repos.
- Allow the spike: the change's spike commit is pushed to `spike/ghcr`, which runs `spike.yml` (tasks.md 1.3); delete the `docs/spike` and `docs/spike-edge-first` packages afterwards (1.7); the implementer removes the `spike/ghcr` branch.

After catalog_opm section 1 merges: catalog_opm section 2 (step 4) is the owner's go-live. Optionally make catalog_opm's `Docs / check` jobs required on `main`.

Workspace follow-ups (the supervisor handles the first in a workspace PR; the rest are owner items):

- **Owner item, tag-rule conflict.** Root `AGENTS.md`, "Release Tags Are Immutable", Registry tags bullet: name the docs bundles' moving tags (`<version>`, `<MAJOR.MINOR>`, `<MAJOR>`, `edge` under `ghcr.io/open-platform-model/docs/*`) as mutable by design; full tags (`<version>.<revision>`) are never overwritten.
- Root `AGENTS.md` Repos table: docs-kit's row and commands once the tool exists; the scope list of "Release Tags Are Immutable" gains docs-kit.
- `RELEASING.md`: whether a docs-kit release cascades into the callers' `publish.yml@vX` refs (and catalog_opm's `.opm-docs-version`) is undecided; until then callers bump by hand, both pins in one PR.

## Decisions recorded for the owner to confirm

- Sitemap lists only the newest minor of each major; `edge` gets its own Pagefind index but no alias stubs; phase-1b history is a file written by `opm-docs pull` (supervisor decisions, 2026-10-02; design.md "Site decisions").
- opmodel.dev pins the tool by SHA-256 in its build image and pulls inside it; `pull --local` takes several segments per project and needs no network when every tab is local; lock entries carry `root`; no redirects from the old reference URLs; the site lint and `opm-docs lint` agree through one conformance fixture set until phase 3 (supervisor decisions, 2026-10-02).
- Callers reference `publish.yml` by release tag, decided by the owner on 2026-10-02 (design.md C5); SHA pinning with a SHA allowlist was rejected.

Departures from the original `DESIGN.md`, **approved by the owner on 2026-10-02** and recorded there as decisions 13 to 16:

- **Done criterion split** (DESIGN decision 13). Phase 1 here ends with opm 4.4 and edge on the site; the docs-revision half of DESIGN.md's criterion moves to `add-docs-revisions`.
- **Pull inside the site's build image** (DESIGN decision 14), with network on for that step only, instead of DESIGN.md's host step.
- **`org.opencontainers.image.created` is the source commit's time** (DESIGN decision 15), not the build time, so a rebuild gives the same digest.
- **`push` and `promote` are separate commands** (DESIGN decision 16): `push` writes only the full tag, cosign signs, then `promote` verifies and moves the tags; DESIGN.md's `push` moved the tags itself.

## Follow-up: docs-kit `add-docs-revisions`

Planned in `openspec/changes/add-docs-revisions/`. Gated on `build-opm-docs-phase-1` being archived and released as `v0.1.0`. It adds `opm-docs revise`, the documentation-only check and the workflow's `revision` mode, and releases as `v0.2.0`. Then: the owner merges the `0.2.0` release PR; in catalog_opm (no OpenSpec change needed: a `ci` PR), `docs.yml` (`publish.yml@v0.2.0`) and `.opm-docs-version` move to `v0.2.0` together, and the dispatch gains `mode: revision` and a `fix` input. The proof that completes DESIGN.md's phase-1 criterion moves there: land a doc-comment fix on catalog_opm `main`, dispatch `mode=revision tag=opm-v4.4.<n> fix=<sha>`, and the next site build shows the fix under `/catalogs/opm/4.4/` with no catalog release, its build stamp naming revision 1. The site needs no change for it (`pull` already handles any revision).

## Amended in review (owner decision 2026-10-02)

`publish.yml` carries no version literal: it installs the `opm-docs` release named by the caller's repo-root `.opm-docs-version`. catalog_opm's section 1 therefore must add `.opm-docs-version` (already planned for its local tasks) before its first `publish.yml` run, and every later bump moves `.opm-docs-version` and the `publish.yml@vX.Y.Z` ref together in one PR. release-please no longer rewrites `publish.yml`. The default signer glob is `refs/tags/v[0-9]*`.
