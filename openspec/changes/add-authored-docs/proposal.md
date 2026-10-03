## Why

Phase 3 (DESIGN.md) ships every repository's authored pages (`docs/site/`) in its bundle, so the site stops reading source repositories through git: no more `materialise.sh`, git-date step or archive of release-branch heads. The `markdown` source already copies a directory with its git dates, and `generalize-build-assembly` lets it feed a docs-placed bundle. core, cli, library and opm-operator ship their authored pages already when they adopt docs-kit for their reference (DESIGN decision 20): their bundles are chosen by the cli's pins, so a second adoption later would need a second release cascade. What is missing is the one fact the site still takes from git for authored docs: where "Edit this page" goes. The owner decided it links the page's file on `main` whatever version the page shows (DESIGN decision 19); a released page's file may have moved or gone on `main` since, so the producer, which has `main` checked out, records the right path.

Gate: `generalize-build-assembly` is merged. It must ship before core, cli, library and opm-operator cut the releases whose bundles the site first shows, or those bundles carry no edit paths.

## What Changes

- **`pages[].edit`** in `manifest.json`: the repository-relative path of an authored page's file on `main` when that file exists there, written by `build` from the `main` tree it runs in (the checkout `publish.yml` makes beside the release tree), for docs-placed bundles only; absent for generated pages, for tab and section pages, and for a file `main` no longer has. A manifest with `edit` is refused by an older `opm-docs pull` (the schema is closed), so the site's pinned `opm-docs` moves to this release before any producer does (C12). The site links `https://github.com/<source.repo>/edit/main/<edit>`.
- **Authored docs in docs bundles, specified**: a `markdown` source over `docs/site` in a docs bundle, the root `_index.md` and section `_index.md` pages it may carry, figure shortcodes, git dates, and how a repository with a phase-2 reference adds its authored pages to the same bundle (one bundle per repository; the committed generated pages are gone by then, or the build refuses the collision).
- **The bundle shapes of phase 3, recorded in C1 and C15**: `catalog-opm-docs` beside the `catalog-opm` tab (both built from one release tag), `opm` released by release-please (DESIGN decision 17), and the "Edit this page" rule for tab, docs and section pages.

Release class: MINOR. Additive: an optional `#Page` field.

Scope: two sections, the last with the contracts and the archive.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `docs-placement`: authored docs pages and their edit path (requirements added).
- `bundle-format`: `pages[].edit` (requirement added).

## Impact

- Code: `schema/manifest.cue` (`#Page.edit`), `internal/build` (main-tree lookup), `internal/gitsrc` (file exists at a tree), `docs/contracts.md` C3, C8, C15.
- Consumers: **opmodel.dev** (`serve-docs-from-bundles`): "Edit this page" from `edit`, "Last updated" from `lastmod`, per-repository switch from git to bundle. **catalog_opm, core, cli, library, opm-operator, opm**: each adds a `markdown` source over `docs/site` (catalog_opm as the new project `catalog-opm-docs`; opm with its first release). All listed in `docs/orchestration.md`, phase 3.
