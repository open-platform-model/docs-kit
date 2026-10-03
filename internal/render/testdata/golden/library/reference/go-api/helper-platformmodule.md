---
title: "opm/helper/platformmodule"
description: "Package platformmodule generates a platform CUE module from catalog coordinates."
type: reference
---

```go
import "github.com/open-platform-model/library/opm/helper/platformmodule"
```

Package platformmodule generates a platform CUE module from catalog coordinates. A platform is a CUE module that imports its catalogs, and the kernel's only platform input is such a module on disk ([kernel.Kernel.AcquirePlatformFromDir](/docs/reference/go-api/kernel/#kernelacquireplatformfromdir)); a frontend that starts from typed coordinates (a Platform CR, a seeded local default) turns them into that module here instead of writing its own generator.

Three seams, each independently testable:

- [Generate](/docs/reference/go-api/helper-platformmodule/#generate) is pure: typed input plus the resolved dependency closure in, deterministic file bytes out. The same input always yields byte-identical files (cue.mod/module.cue in modfile canonical format, platform.cue embedding core.#Platform and importing every catalog under a positional alias).
- [Closure](/docs/reference/go-api/helper-platformmodule/#closure) derives the module's full dependency list from the pinned modules' published module files: the roots ([Roots](/docs/reference/go-api/helper-platformmodule/#roots): core and every subscribed catalog) plus everything they transitively require, at the maximum version any requirement names. It is the tidied list a `cue mod tidy` would write, computed once at generation. Module files are read through a caller-configured [ModFileSource](/docs/reference/go-api/helper-platformmodule/#modfilesource) ([NewRegistry](/docs/reference/go-api/helper-platformmodule/#newregistry)); tests supply a fixture graph.
- [Files.WriteTo](/docs/reference/go-api/helper-platformmodule/#fileswriteto) places the generated files under a caller-owned directory. Directory lifecycle (generations, staging swaps, retention) stays with the frontend.

The core pin is the release the kernel was verified against ([schema.DefaultSchemaVersion](/docs/reference/go-api/schema/#defaultschemaversion)); a caller that needs another core build assembles its [Dep](/docs/reference/go-api/helper-platformmodule/#dep) roots directly. The generated module's own path is caller input ([Input.ModulePath](/docs/reference/go-api/helper-platformmodule/#input)) and lives under the reserved, never-published platforms namespace.

This package is opt-in helper convenience (see package opm/helper): a frontend MAY write its platform module by hand instead.

## Constants

### CorePath

```go
const (
	// CorePath is the major-qualified module path of the core schema the
	// generated module embeds.
	CorePath = "opmodel.dev/core@v2"

	// LanguageVersion is the generated module's declared CUE language
	// version: the floor every published first-party module declares and the
	// render build requires for cue.mod/local-module.cue.
	LanguageVersion = "v0.17.0"

	// ModuleFileName and PlatformFileName are the two files a generated
	// module consists of, relative to the module directory.
	ModuleFileName   = "cue.mod/module.cue"
	PlatformFileName = "platform.cue"
)
```

## Types

### Dep

```go
type Dep struct {
	Path    string
	Version string
}
```

Dep is one pinned dependency of the generated cue.mod: a major-qualified module path and a canonical "v"-prefixed version.

#### Closure

```go
func Closure(ctx context.Context, src ModFileSource, roots []Dep) ([]Dep, error)
```

Closure derives the generated module's full dependency list from roots: a breadth-first walk over each reachable module version's published module file, selecting the maximum version per major-qualified path, the roots participating in the maximum. This is minimum version selection computed the way `cue mod tidy` computes it (: tidying happens once, at platform-module generation), minus the prune of modules no import reaches, which pins a path nothing evaluates and is harmless. Derived entries carry no default-major marker; `cue mod tidy` writes none for a platform either, because the platform imports nothing unqualified. Local replacements (cue.mod/local-module.cue, the "local" path) are skipped.

A root or transitive requirement naming an unpublished build fails with an error naming the module path and version, the same wording the CUE resolver uses for a missing pin. The walk honours ctx cancellation.

#### Roots

```go
func Roots(entries []Entry) []Dep
```

Roots returns the dependency roots the closure is derived from: the core pin plus every entry's catalog, disabled entries included (a disabled entry still imports its catalog). Core is pinned at [schema.DefaultSchemaVersion](/docs/reference/go-api/schema/#defaultschemaversion), the release the kernel was verified against; a caller that needs a different core build assembles its \[\]Dep roots directly. Versions are canonicalised with the "v" prefix cue.mod requires; subscriptions carry bare SemVer.

### Entry

```go
type Entry struct {
	Path    string
	Version string
	Enable  bool
}
```

Entry is one catalog subscription in the shape Generate consumes: the major-qualified catalog path (the registry key), the bare SemVer build the subscription names and whether the subscription is enabled.

### Files

```go
type Files map[string][]byte
```

Files maps a path relative to the module directory to the file's bytes.

#### Generate

```go
func Generate(in Input) (Files, error)
```

Generate renders the module's two files from in. It is pure and deterministic: entries and dependencies are emitted in sorted path order whatever order they arrive in, so the same input always produces byte-identical content. Each registry entry stamps the subscription's version as the entry's expected `version`, which unifies with the schema's readout of the imported catalog so wrong bytes are a build conflict naming the entry (tripwire).

#### Files.WriteTo

```go
func (f Files) WriteTo(dir string) error
```

WriteTo places the generated files under dir, creating parent directories as needed. A file name that would resolve outside dir is refused before anything is written. The helper owns no directory lifecycle: whether dir is a fresh generation directory, a staging directory later swapped into place, or a cache entry is the frontend's policy.

### Input

```go
type Input struct {
	Name       string
	Type       string
	ModulePath string
	Entries    []Entry
	Deps       []Dep
}
```

Input is everything Generate needs. Name and Type are the platform's metadata.name and type; ModulePath is the generated module's own identity (a reserved, never-published platforms path such as "opmodel.dev/platforms/cluster@v0"); Entries are the catalog subscriptions; Deps is the resolved dependency closure (see Closure), which MUST contain a pin for core and for every entry's catalog.

### ModFileSource

```go
type ModFileSource interface {
	ModFile(ctx context.Context, mv module.Version) (*modfile.File, error)
}
```

ModFileSource yields a published module's cue.mod/module.cue. It is the one method of [modconfig.Registry](https://pkg.go.dev/cuelang.org/go/mod/modconfig#Registry) the closure needs, kept as a narrow interface so tests can supply a fixture graph without a registry.

#### NewRegistry

```go
func NewRegistry(cfg RegistryConfig) (ModFileSource, error)
```

NewRegistry returns a module-file source resolving through cfg. Module files it fetches are the same artifacts a build fetches, so a closure derivation never adds an artifact class to the frontend's registry path.

### RegistryConfig

```go
type RegistryConfig struct {
	// Registry is the CUE registry mapping (CUE_REGISTRY syntax) module files
	// resolve through. Empty falls back to CUE_REGISTRY in Env.
	Registry string

	// ClientType is reported to registries in the User-Agent header. Empty
	// falls back to modconfig's own default ("cuelang.org/go").
	ClientType string

	// Env is the environment the CUE module cache location (CUE_CACHE_DIR)
	// and, when Registry is empty, CUE_REGISTRY are read from. Nil selects
	// modconfig's default, the current process environment; passing nil is
	// the caller's explicit choice, never a hidden lookup by the helper.
	Env []string
}
```

RegistryConfig is the caller-supplied configuration a registry-backed [ModFileSource](/docs/reference/go-api/helper-platformmodule/#modfilesource) resolves through. Nothing here is read from the process by the helper itself (kernel neutrality): the frontend states every value.
