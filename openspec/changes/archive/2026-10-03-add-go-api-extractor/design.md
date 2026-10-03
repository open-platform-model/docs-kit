# Design: add-go-api-extractor

## Context

Builds on `generalize-build-assembly` (registry, docs placement, citations; C6, C15). No generator exists to port: the library has nine packages under `github.com/open-platform-model/library/opm/` (about 118 exported symbols; the `kernel` package doc is long, with `#` headings and doc links), and a reader today has only `go doc`.

## Goals / Non-Goals

**Goals:** the `go-api` kind, its model, its pages, stable anchors and links.

**Non-Goals:** type checking or building (no toolchain needed, unlike `cobra`); examples (`Example*` functions; a later additive field); unexported or `internal/` API; a source-code view.

## Decisions

### D1. Configuration (C6, C20)

```cue
#GoAPI: {
	kind:     "go-api"
	module:   =~"^\\./"                 // the directory holding go.mod: "./"
	root:     =~"^\\./"                 // page names are relative to it: "./opm"
	packages: [string, ...string]       // patterns relative to module: "./opm/..."; "..." matches any depth
	section:  =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$" // an owned directory: "reference/go-api/"
	title:       string & !=""
	description: string & !=""
	weight?:     int & >=1
	citations:   *"strip" | "link"
}
```

The library's file, as its sibling change writes it:

```cue
bundles: library: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/go-api/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:        "go-api"
		module:      "./"
		root:        "./opm"
		packages:    ["./opm/..."]
		section:     "reference/go-api/"
		title:       "Go API"
		description: "Every exported package of the OPM library, from its doc comments."
	}, {
		kind: "markdown", dir: "docs/site" // the library commits no generated pages
	}]
}
```

### D2. Extraction

The import path prefix is `go.mod`'s `module` line. Packages: every directory matched by a pattern that holds non-test `.go` files selected by `go/build.Context{GOOS: "linux", GOARCH: "amd64", CgoEnabled: false}`; a directory with an `internal` path element is skipped; a pattern matching nothing fails a normal build (warning in a backfill, as C5). Parsing: `go/parser.ParseFile(..., parser.ParseComments)`, then `doc.NewFromFiles(fset, files, importPath)` (exported only, `doc.AllDecls` off). Declarations are printed with `go/printer` (gofmt settings) without their doc comments; a type's body keeps field comments. Doc comments: `comment.Parser` with a `LookupPackage` that knows the module's packages, printed by a `comment.Printer` to Markdown (`HeadingLevel` set by D4, `DocLinkURL` per D4), then cleaned by "Doc-comment rules" and the citation policy. No package or symbol is invented: an undocumented exported symbol keeps its declaration and an empty `doc`.

### D3. Data model (C20)

```json
{
  "schema": "docs.opmodel.dev/data/go-api/v1",
  "modulePath": "github.com/open-platform-model/library",
  "version": "1.0.0-beta.1",
  "section": "reference/go-api/", "title": "Go API", "description": "...", "weight": null,
  "packages": [
    {
      "importPath": "github.com/open-platform-model/library/opm/kernel",
      "name": "kernel", "page": "kernel", "synopsis": "<first sentence>",
      "doc": "<markdown>",
      "consts": [#Value], "vars": [#Value],
      "funcs":  [#Func],
      "types":  [{"name": "Kernel", "anchor": "kernel", "decl": "...", "doc": "...",
                  "consts": [#Value], "vars": [#Value], "funcs": [#Func], "methods": [#Func]}],
      "files": ["opm/kernel/doc.go", "opm/kernel/kernel.go"]
    }
  ]
}
```

with `#Value: {names: [string], anchor, decl, doc}` and `#Func: {name, recv: string | null, anchor, decl, doc}`. Packages sort by import path; symbols in `go/doc` order (by name). Every list is present.

### D4. Pages and links

| Path | Page |
|---|---|
| `<section>_index.md` | front matter `title`, `description`, `weight` when set; one line per package: link and synopsis. Completable, first heading `## Packages`. |
| `<section><page>.md` | front matter `title` = the import path relative to the module (`opm/kernel`), `description` = synopsis (or "Package <name>." when empty), `type: reference`; the package doc with its headings shifted to start at `###`; `## Constants`, `## Variables`, `## Functions`, `## Types` (each type `### <Name>` with its declaration in a `go` fence, doc, then its constructors and methods as `#### <Name>` / `#### <Type>.<Method>`) |

Page name: the package directory relative to `root`, `/` as `-` (`helper/objectset` is `helper-objectset`); a package at `root` itself is refused (no page name). Anchors: Hugo's default (`github`-style) anchor of the heading text, which the renderer computes the same way: lower case, every character other than a letter, digit, space, `-` or `_` dropped, spaces as `-` (`Kernel` is `kernel`, `Kernel.Render` is `kernelrender`); a constant or variable group is headed by its first name. Doc links: a target documented in this bundle links `/docs/<section><page>/#<anchor>`; anything else `https://pkg.go.dev/<import path>#<Name>` (a URL form dialect 1 allows). `manifest.json` `pages[].source` is the package's first file by name.

### D5. Commands

No new command or flag. A pattern that matches nothing: "`packages` `./opm/nope/...` matches no package under `./`" (exit 2).

## Research & Decisions

### Parse versus type-check (decided in planning)

**Options considered**: 1. `go/parser` + `go/doc` - no toolchain, no module download, deterministic; 2. `golang.org/x/tools/go/packages` - resolves types across packages but needs the toolchain and the module cache in the build job.
**Decision**: option 1.
**Rationale**: `go/doc` already groups constructors and methods; doc links resolve by name through `LookupPackage`; nothing on the page needs types.

### Anchor scheme (decided in planning)

**Decision**: Hugo's default heading anchors for the heading text, so the page needs no attribute syntax the dialect does not know, and `pkg.go.dev`-style `#Name` fragments are not used inside the site.

## Risks / Trade-offs

- A package doc written for `go doc` can hold forms that read oddly as Markdown; the library fixes its own comments (its sibling change), docs-kit does not special-case them.

## Durable decisions

- C20 (new): config (D1), model (D3), pages, anchors and links (D4).
- C6: the source-kind table gains `go-api`.

## Implementation notes

- **Markdown printer.** `go/doc/comment`'s own `Printer.Markdown` writes `{#hdr-...}` heading IDs the dialect does not know, indents code instead of fencing it and leaves `{{` alone, so the extractor prints the same parsed tree with a printer of its own under docs-kit's rules: prose escaped (``\ ` * _ [ ] < > | !``, `{{` as `{\{`, a block marker at the start of a line), backtick code spans kept and resized, code blocks in `text` fences one backtick longer than their longest run, link text escaped, and a link whose scheme is not `http` or `https` refused (the comment parser makes links of `file`, `ftp`, `gopher`, `mailto`, `nntp`, `http` and `https` URLs only; any other text stays escaped prose). Doc links resolve as D4 says; in addition a name of a constant or variable group links its group's heading, and a field (`[T.Field]`) of a documented type links the type's heading, whose declaration shows it.
- **Anchors.** The extractor walks a package page's headings in the order the renderer writes them (`goapi.PageSections`), the package doc's and every symbol doc's headings included, and makes repeats unique as Hugo does (`-1`, `-2`): a function `WidgetSpin` and a method `Widget.Spin` get `widgetspin` and `widgetspin-1`.
- **Pages.** A package page opens with its import line in a `go` fence. A type's constants and variables are `####` entries headed by their first name, before its constructors and methods. A command (`package main`) is not documented.

## Trial build (task 2.2, 2026-10-03)

`opm-docs build --release v1.0.0-beta.1` over a clone of the library at `v1.0.0-beta.1` (`02344e5`) with D1's config (from outside the tree, as the backfill runs it, and again from inside with `opm-docs check`): green, bundle-mode lint clean, two builds byte-identical. 18 pages: the section index and 9 package pages (`catalog`, `errors`, `helper`, `helper-objectset`, `helper-platformmodule`, `kernel`, `module`, `platform`, `schema`), plus the 8 authored pages of `docs/site`. 119 exported entries.

- **Undocumented at the tag** (13): `opm/errors` `ContractCollisionsError.Error`, `NotRoutableError.Error`, `OverSubscribedContractsError.Error`, `PlatformCoreTooOldError.Error`, `SkewError.Error`, `TransformError.Error`, `TransformError.Unwrap`, `UnmatchedComponentsError.Error`, `UnresolvedDemandsError.Error`; `opm/kernel` `RenderError.Error`, `RenderError.Unwrap`, the `SkewWarn`/`SkewRefuse` group (no group comment; each constant has its own); `opm/helper/platformmodule` the `CorePath`/`LanguageVersion`/`ModuleFileName`/`PlatformFileName` group (likewise).
- **Garbled at the tag**: `opm/kernel/doc.go`'s "Surface" list and its three code examples ("One-Kernel-per-process example", the advisory-facts loop and the replacements loop) are re-wrapped into running text at `v1.0.0-beta.1`, so they render as paragraphs. No page-dialect problem otherwise.
- **Library `main`** (`cd31684`): the same build is green with every exported symbol documented, the list a list and the examples `text` fences; so the fix is already on `main` and reaches the backfilled bundle as a docs revision or with the next library release (docs/orchestration.md step 4b).
