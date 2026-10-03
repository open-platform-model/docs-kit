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
