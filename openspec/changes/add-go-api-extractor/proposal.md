## Why

The library is embedded by the cli and the operator and by anyone building on the kernel, yet its Go API has no reference on the site (DESIGN.md phase 2: "library: an API reference for the first time"). This change adds a `go-api` extractor that documents a module's exported packages from their doc comments, so the library publishes `reference/go-api/` in its bundle.

Gate: `generalize-build-assembly` is merged. Parallel with the other extractor changes and `pull-docs-placement`.

## What Changes

- **`go-api` source kind**: parses the selected packages of a Go module with `go/parser` and `go/doc` (no type checking, no build), skipping `internal/` and test files, and writes `data/go-api.json` (`docs.opmodel.dev/data/go-api/v1`): per package its import path, doc, constants, variables, functions, types with their methods and constructors, each with its declaration (gofmt-printed) and doc comment converted from Go doc syntax to Markdown with `go/doc/comment`.
- **Pages**: `<section>_index.md` listing the packages, one page per package, stable symbol anchors, `[Name]` and `[pkg.Name]` doc links resolved to those anchors when the target is documented in the bundle, and to `pkg.go.dev` otherwise.
- **Contract C20**.

Release class: MINOR. Additive.

Scope: three sections, the last with the contract and the archive.

## Capabilities

### New Capabilities

- `go-api-extractor`: the source kind, its data model and its pages.

### Modified Capabilities

None.

## Impact

- Code: `internal/extract/goapi`, `internal/render` (templates), `schema/config.cue`, `docs/contracts.md` C20.
- Consumers: **library** (sibling change `publish-go-api-bundle`): first repairs the garbled "Surface" list in `opm/kernel/doc.go`, adds `docs-kit.cue`, the workflows and a link from `docs/site/embedding/embed-the-kernel.md` to `/docs/reference/go-api/`. **opmodel.dev**: pulls `library` per site version at the cli's pin.
- Risk: a package doc written for `go doc` (its `# Heading` and lists) renders as Markdown through `go/doc/comment`; headings are shifted under the page's own heading levels so the page outline stays valid.
