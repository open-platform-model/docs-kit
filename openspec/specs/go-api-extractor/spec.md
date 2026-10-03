# go-api-extractor Specification

## Purpose
The `go-api` source kind: a Go module's exported packages, parsed without building or type checking, written as `data/go-api.json` and rendered as a Go API reference with stable heading anchors and resolved doc links (docs-kit C20).

## Requirements

### Requirement: The go-api source documents a module's exported API
A `go-api` source SHALL parse, without building or type checking, every package matched by its `packages` patterns under the Go module at `module`, excluding `internal/` packages, `_test.go` files and files excluded by build constraints for `linux/amd64`, and SHALL write `data/go-api.json` with schema `docs.opmodel.dev/data/go-api/v1`: per package its import path, page, package doc, and its exported constants, variables, functions and types (with methods and constructors as `go/doc` groups them), each with its gofmt-printed declaration and its doc comment converted to Markdown and cleaned by the doc-comment rules and the citation policy. An exported symbol without a doc comment SHALL be listed with its declaration and an empty doc.

#### Scenario: A constructor groups under its type
- **WHEN** package `kernel` declares `type Kernel struct{...}` and `func New(...) *Kernel`
- **THEN** `New` is listed among `Kernel`'s functions, not among the package's functions

#### Scenario: Internal packages are skipped
- **WHEN** `packages` is `["./opm/..."]` and `opm/internal/x` exists
- **THEN** no entry for `opm/internal/x` is written

### Requirement: Go API pages and anchors
The renderer SHALL write `<section>_index.md` listing every package with its synopsis and `<section><page>.md` per package, where `<page>` is the package's directory relative to the source's `root` with `/` replaced by `-`. Each symbol SHALL sit under a heading whose anchor is Hugo's default anchor of the heading text (a method `Kernel.Render` as `kernelrender`). A doc link to a symbol documented in the bundle SHALL link `/docs/<section><page>/#<anchor>`; a doc link to anything else SHALL link `https://pkg.go.dev/<import path>#<Name>`. Headings in a doc comment SHALL be shifted below the page's own heading levels.

#### Scenario: Link within the library
- **WHEN** a `module` package doc says `[kernel.Kernel]`
- **THEN** the page links `/docs/reference/go-api/kernel/#kernel`

#### Scenario: Link to the standard library
- **WHEN** a doc comment says `[context.Context]`
- **THEN** the page links `https://pkg.go.dev/context#Context`
