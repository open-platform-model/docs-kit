## Why

The owner decided the enhancements repository is bundled too, as one edge-only, unversioned bundle at `/enhancements/` (DESIGN decision 18), so the site build reads no git repository except its own, the done criterion of phase 3. Today the site mounts the enhancements repository's tree and builds its section with a content adapter (`site/enhancements/_content.gotmpl`), a cleaning partial and its own link hook: page generation and link resolution on the site, which Principle I gives to the producer.

Gate: `generalize-build-assembly` and `pull-docs-placement` are merged (the registry, and the pull config this change extends). Independent of `add-authored-docs` and `add-serve-command`.

## What Changes

- **A third placement kind, `section`**, root `/enhancements/`: one bundle outside every site version and outside the version switcher, built from `main` only. Its pages publish at `<root><page URL>` with no segment. A section bundle has no release and no revision: `build --release` and `revise` refuse it; `version` is optional in its config and `manifest.json` always says `edge`.
- **The `enhancements` source kind**: reads every entry (`NNNN/` and `archive/NNNN/`, not `0000`), its `config.yaml`, `README.md` and the seven numbered documents, plus `INDEX.md` and `GRAPH.md`; writes `data/enhancements.json` (`docs.opmodel.dev/data/enhancements/v1`) with each entry's header data, and pages in the page dialect. The site adapter's transforms move here: HTML comments and the first heading removed, relative links resolved to `/enhancements/...` or to GitHub at the commit built (a dangling or escaping link fails the build), link text that is a file name replaced by the page title, an untagged code fence tagged `text`.
- **The dialect accepts `/enhancements/graph/`** (a new fixture in the conformance set, so the site's shell lint learns it too).
- **`pull` gains `sections`**: each section project and the repository allowed to sign it; `pull` resolves its `edge` tag only and unpacks it like a tab segment; the lock records it in `bundles` with its root.
- **Contract C21** "Sections and the enhancements bundle", and C1, C3, C7, C8, C11 updated.

Release class: MINOR. Additive: a new placement kind, source kind, config key and link form.

Scope: five sections; section 1 is a spike because the enhancement documents have never passed the page dialect (an estimate finds 81 untagged fences and about 190 lines with angle brackets in 224 files).

## Capabilities

### New Capabilities

- `enhancements-bundle`: the section placement, the `enhancements` source and its pages.

### Modified Capabilities

- `bundle-pull`: the pull config admits `sections`; pulling a section bundle (requirement modified and requirements added).
- `dialect-lint`: the `/enhancements/graph/` link form (requirement added).

## Impact

- Code: `schema/config.cue`, `schema/manifest.cue`, `schema/pull.cue`, `schema/lock.cue`, `internal/extract/enhancements`, `internal/render`, `internal/build` (edge-only placement), `internal/pull` (sections), `internal/dialect` (link form, fixture), `docs/contracts.md`.
- Consumers: **enhancements** (sibling change `publish-enhancements-bundle`): `docs-kit.cue`, `docs.yml` (check on PRs, edge on `main`), `.opm-docs-version`, and the source fixes the spike lists. **opmodel.dev** (`serve-enhancements-from-bundle`): `bundles.cue` `sections`, the section built from the bundle, its adapter, cleaning partial and link hook removed, `[section "enhancements"]` removed from `versions.conf`, the shell lint fixture copied.
- Risk: the enhancements repository's own link check and the bundle's link resolution must agree; the spike compares them.
