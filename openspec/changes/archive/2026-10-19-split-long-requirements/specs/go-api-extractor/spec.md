## REMOVED Requirements

### Requirement: The go-api source documents a module's exported API
**Reason**: Split into three requirements that each state one behavior.
**Migration**: See "A go-api source parses a module's exported packages", "A go-api source writes its model file" and "An exported symbol without a doc comment is listed".

### Requirement: Go API pages and anchors
**Reason**: Split into two requirements that each state one behavior.
**Migration**: See "Go API pages hold each symbol under a fixed anchor" and "A Go doc link resolves in the bundle or on pkg.go.dev".

## ADDED Requirements

### Requirement: A go-api source parses a module's exported packages
A `go-api` source SHALL parse, without building or type checking, every package matched by its `packages` patterns under the Go module at `module`, excluding `internal/` packages, `_test.go` files and files excluded by build constraints for `linux/amd64`.

#### Scenario: Internal packages are skipped
- **WHEN** `packages` is `["./opm/..."]` and `opm/internal/x` exists
- **THEN** no entry for `opm/internal/x` is written

### Requirement: A go-api source writes its model file
A `go-api` source SHALL write `data/go-api.json` with schema `docs.opmodel.dev/data/go-api/v1`: per package its import path, page, package doc, and its exported constants, variables, functions and types (with methods and constructors as `go/doc` groups them), each with its gofmt-printed declaration and its doc comment converted to Markdown and cleaned by the doc-comment rules and the citation policy.

#### Scenario: A constructor groups under its type
- **WHEN** package `kernel` declares `type Kernel struct{...}` and `func New(...) *Kernel`
- **THEN** `New` is listed among `Kernel`'s functions, not among the package's functions

### Requirement: An exported symbol without a doc comment is listed
An exported symbol without a doc comment SHALL be listed in `data/go-api.json` with its declaration and an empty doc.

#### Scenario: A symbol without a doc comment
- **WHEN** an exported symbol has no doc comment
- **THEN** it is listed with its declaration and an empty doc

### Requirement: Go API pages hold each symbol under a fixed anchor
The renderer SHALL write `<section>_index.md` listing every package with its synopsis and `<section><page>.md` per package, where `<page>` is the package's directory relative to the source's `root` with `/` replaced by `-`. Each symbol SHALL sit under a heading whose anchor is Hugo's default anchor of the heading text (a method `Kernel.Render` as `kernelrender`). Headings in a doc comment SHALL be shifted below the page's own heading levels.

#### Scenario: A method heading
- **WHEN** the renderer writes the method `Kernel.Render`
- **THEN** its heading anchor is `kernelrender`

### Requirement: A Go doc link resolves in the bundle or on pkg.go.dev
A doc link to a symbol documented in the bundle SHALL link `/docs/<section><page>/#<anchor>`; a doc link to anything else SHALL link `https://pkg.go.dev/<import path>#<Name>`.

#### Scenario: Link within the library
- **WHEN** a `module` package doc says `[kernel.Kernel]`
- **THEN** the page links `/docs/reference/go-api/kernel/#kernel`

#### Scenario: Link to the standard library
- **WHEN** a doc comment says `[context.Context]`
- **THEN** the page links `https://pkg.go.dev/context#Context`
