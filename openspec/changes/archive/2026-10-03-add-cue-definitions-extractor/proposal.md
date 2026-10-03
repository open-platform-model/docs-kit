## Why

core commits its definitions reference (`docs/site/reference/definitions/`, nine pages) from its own generator, `tools/refgen` (about 1,800 lines). DESIGN.md phase 2 moves it into a bundle built by docs-kit, so core's CI publishes it with each release and the generator and its committed pages go. This change adds the `cue-definitions` extractor and its renderer, with output that matches core's refgen.

Gate: `generalize-build-assembly` is merged (it provides the registry, docs placement and `citations`). Parallel with the other extractor changes and `pull-docs-placement`.

## What Changes

- **`cue-definitions` source kind**: parses a CUE package's files (no evaluation, as core's refgen does), takes every exported top-level definition with its doc comment, its formatted spec (hidden fields and maintainer comments dropped, citations handled per policy), its shape, the definitions it uses and is used by, and the rules derivable from its syntax; groups them into pages by an inclusion list in `docs-kit.cue` that must place or exclude every exported definition.
- **`data/cue-definitions.json`** (`docs.opmodel.dev/data/cue-definitions/v1`) and its renderer: the section index (`<section>_index.md`) and one page per group, with stable anchors.
- **Parity** with core's committed pages at the newest core tag, as `TestCatalogOPMParity` did for the catalog.
- **Contract C17** "`cue-definitions`".

Release class: MINOR. Additive: a new source kind.

Scope: three sections, the last with the contract and the archive.

## Capabilities

### New Capabilities

- `cue-definitions-extractor`: the source kind, its data model and its pages.

### Modified Capabilities

None (the source registry in `opm-docs-cli` already admits registered kinds).

## Impact

- Code: `internal/extract/cuedefs`, `internal/render` (templates `defs-index.md.tmpl`, `defs-page.md.tmpl`), `schema/config.cue` (`#CueDefinitions` in `#Source`), `docs/contracts.md` C17.
- Consumers: **core** (sibling change `publish-definitions-bundle`, `docs/orchestration.md`): moves `tools/refgen/groups.go`'s lists into `docs-kit.cue`, publishes the bundle, then deletes `tools/refgen` and the committed pages once the site reads the bundle. **opmodel.dev**: pulls `core` per site version (`pull-docs-placement` and its sibling).
