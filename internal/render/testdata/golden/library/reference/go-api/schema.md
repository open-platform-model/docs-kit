---
title: "opm/schema"
description: "Package schema is the kernel's single source of truth for OPM schema-side knowledge: CUE paths, metadata types, and the OCI-backed schema loader."
type: reference
---

```go
import "github.com/open-platform-model/library/opm/schema"
```

Package schema is the kernel's single source of truth for OPM schema-side knowledge: CUE paths, metadata types, and the OCI-backed schema loader.

Schema is unversioned at the package level. The library consumes exactly one OPM CUE schema package, opmodel.dev/core at the release [DefaultSchemaModule](/docs/reference/go-api/schema/#defaultschemamodule) pins, resolved through CUE's module system against CUE\_REGISTRY. There is no in-tree schema mirror.

### Path inventory

CUE paths are exported as package-level cue.Path variables (Metadata, Components, Values, Config, Module, DebugValues). Callers use schema.X verbatim — there is no Paths() accessor, no struct, no lookup. The inventory is exactly what Go code reads: matching and execution happen inside the render build, in CUE, and read nothing by path from Go.

### Metadata types

ModuleMetadata, InstanceMetadata and PlatformMetadata — one per artifact the kernel accepts — are the canonical decoded metadata records. They are decoded from the artifact's metadata field by the artifact constructors (module.NewModuleFromValue, platform.NewPlatformFromValue) and by the kernel's instance processing; missing metadata is fatal there. Consumers read them through Module.Metadata, Instance.Metadata and Platform.Metadata.

### Schema loader and cache

Loader is the strategy interface for resolving the schema; OCILoader is the sole public implementation, fetching [DefaultSchemaModule](/docs/reference/go-api/schema/#defaultschemamodule) through CUE's module system, and its PinnedVersion reports the exact release the identifier names with no load. Cache memoizes a single Loader.Load per instance (sync.Once-guarded) and exposes ResolvedVersion for diagnostics.

Long-running consumers attach the Cache to a Kernel (via kernel.WithSchemaLoader) and reuse the kernel-owned cache via kernel.SchemaCache(). The library auto-applies no CUE\_REGISTRY default; callers opt in by setting CUE\_REGISTRY to schema.PublicRegistry (or to a private mirror) before the first Cache.Get.

## Constants

### CollisionsSince

```go
const CollisionsSince = "2.0.0-alpha.13"
```

CollisionsSince is the first core release reporting contract collisions (#Platform.#contracts.collisions and collidingEntries, [ContractsCollisions](/docs/reference/go-api/schema/#metadata)), without the "v" prefix. It documents the report and is used by tests; it is never a floor. Every older core carrying #contracts fails to evaluate a platform whose enabled entries share a contract key (the definedBy fold conflicts), so a platform pinning a core between [ProvidedBySince](/docs/reference/go-api/schema/#providedbysince) and this release decodes an absent report as no collision.

### DefaultSchemaModule

```go
const DefaultSchemaModule = "opmodel.dev/core@v2.0.0-beta.1"
```

DefaultSchemaModule is the module identifier used by [OCILoader.Load](/docs/reference/go-api/schema/#ociloaderload) when [OCILoader.Module](/docs/reference/go-api/schema/#ociloader) is empty.

It names an exact core release, never the floating "opmodel.dev/core@v2" major: the release the kernel's render glue, fixtures and parity oracle were verified against. 2.0.0-beta.1 is that release; it is core's first beta and carries the 2.0.0-alpha.13 schema unchanged. 2.0.0-alpha.13 is the first release reporting contract collisions on the derived #Platform.#contracts inventory (`collisions` and `collidingEntries`, with `routable` false while any exist, and `defined` and `definedBy` folding only keys with exactly one enabled definer), on top of the per-registry-entry provider count (`providedBy`, with `overSubscribed`, `unfulfilled` and `routable` recounted from it), the comparable-predicate report (`comparable` and `discriminated`), the inventory, the registry shape (a #Platform.#registry entry embeds its catalog by import and derives `version` from it) and the transformer-context projection.

The default is not the render floor: Kernel.Render and Platform.Contracts accept every core from [ProvidedBySince](/docs/reference/go-api/schema/#providedbysince) on, and a platform pinning a release between the floor and [CollisionsSince](/docs/reference/go-api/schema/#collisionssince) decodes an absent collision report as no collision (such a core cannot evaluate a colliding platform at all). The constant advances only by a deliberate change that re-verifies the glue and the fixtures against the new release; a default that floats ahead of the glue breaks every synthesized artifact on a cold cache.

### ProvidedBySince

```go
const ProvidedBySince = "2.0.0-alpha.12"
```

ProvidedBySince is the first core release deriving #Platform.#contracts.providedBy ([ContractsProvidedBy](/docs/reference/go-api/schema/#metadata)), without the "v" prefix: the oldest core a platform module may pin for Kernel.Render and Platform.Contracts, named in their PlatformCoreTooOldError.

### PublicRegistry

```go
const PublicRegistry = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
```

PublicRegistry is the documented CUE\_REGISTRY mapping for resolving the OPM core schema from its canonical GHCR location with a fallback to registry.cue.works. The library does NOT auto-apply this value as a default; callers opt in by setting CUE\_REGISTRY=schema.PublicRegistry (or by passing it via [OCILoader.Registry](/docs/reference/go-api/schema/#ociloader)).

Operators in restricted environments may set CUE\_REGISTRY to a mirror or to an inline configuration without touching this constant.

## Variables

### Metadata

```go
var (
	// Artifact root.
	Metadata = cue.ParsePath("metadata")

	// Module instance.
	Components = cue.ParsePath("components")
	Values     = cue.ParsePath("values")
	Config     = cue.MakePath(cue.Def("config"))
	Module     = cue.MakePath(cue.Def("module")) // instance's reference to its source #Module

	// Platform. Contracts is #Platform.#contracts, the contract inventory core
	// derives from the enabled registry entries' contract maps and the
	// transformers' required demands. (*platform.Platform).Contracts decodes
	// its eleven data fields on demand; `defined` (member schemas, not data) is
	// not decoded. Never the loader gate, never platform construction.
	Contracts = cue.MakePath(cue.Def("contracts"))

	// ContractsProvidedBy is #Platform.#contracts.providedBy: every
	// provider-fulfilled contract FQN an enabled transformer requires, to
	// the sorted registry keys of the entries supplying it. It is the one
	// provider count: the render glue reads it in CUE, Contracts() decodes
	// it, and Kernel.Render checks, before staging, only that the
	// platform's Package carries it (a presence test, nothing more), so a
	// platform pinning a core older than [ProvidedBySince] is refused.
	ContractsProvidedBy = cue.MakePath(cue.Def("contracts"), cue.Str("providedBy"))

	// ContractsCollisions is #Platform.#contracts.collisions: every
	// contract FQN more than one enabled registry entry's catalog lists,
	// ascending. Such a key is folded into none of definedBy, requiredBy,
	// unfulfilled or comparable, and routable is false while any exists.
	// The render glue reads it and Contracts() decodes it, both guarded on
	// presence: a core without it cannot evaluate a colliding platform, so
	// absence means no collision ([CollisionsSince]).
	ContractsCollisions = cue.MakePath(cue.Def("contracts"), cue.Str("collisions"))

	// ContractsCollidingEntries is #Platform.#contracts.collidingEntries:
	// each [ContractsCollisions] key to the ascending registry keys (path
	// plus major) of the enabled entries whose catalogs list it.
	ContractsCollidingEntries = cue.MakePath(cue.Def("contracts"), cue.Str("collidingEntries"))

	// Catalog. Transformers is #Catalog.#transformers, the implementations
	// a catalog ships. RequiredResources, RequiredTraits and Fulfilment are
	// read RELATIVE to a transformer and to one of its demand entries, not
	// from an artifact root: the provider-fulfilled set a catalog implements
	// is the fold of every contract those two demand maps require whose
	// value carries fulfilment "provider" (ADR-009). Their one reader is
	// (*catalog.Catalog).Provides, on demand.
	Transformers      = cue.MakePath(cue.Def("transformers"))
	RequiredResources = cue.ParsePath("requiredResources")
	RequiredTraits    = cue.ParsePath("requiredTraits")
	Fulfilment        = cue.ParsePath("fulfilment")

	// Module-internal field. DebugValues is a Module field — NOT a separate
	// kernel artifact. Frontends that want a debug overlay read it from
	// Module.Package and decide whether to layer it into the values stack;
	// the kernel never receives debugValues as a parameter.
	DebugValues = cue.ParsePath("debugValues")
)
```

CUE paths the kernel's Go code reads or writes on an OPM artifact: metadata decoding, instance processing, the loaders' identity reads, the instance's components and #config accessors, the platform's on-demand contract inventory, the render's core floor and the catalog's on-demand provider-set derivation. This is the whole inventory. Matching and execution read nothing by path from Go: the render build imports the instance and the platform as packages and the generated glue reads `components`, `#composedTransformers` and `#contracts` in CUE. A path with no reader is removed, not kept for a possible consumer.

Definition fields (those starting with "#" in CUE) use cue.MakePath with cue.Def selectors; concrete fields use cue.ParsePath. The two forms are not interchangeable — definition paths constructed with ParsePath do not resolve on closed structs.

## Functions

### DefaultSchemaVersion

```go
func DefaultSchemaVersion() string
```

DefaultSchemaVersion returns the exact core release [DefaultSchemaModule](/docs/reference/go-api/schema/#defaultschemamodule) pins, in the canonical "v"-prefixed form a cue.mod dependency carries ("v2.0.0-beta.1"). It is the version a generated platform module pins core at by default (opm/helper/platformmodule): the release the render glue was verified against is the release a generated platform must embed.

## Types

### Cache

```go
type Cache struct {
	// Loader is the strategy used to resolve and build the schema value.
	// Required.
	Loader Loader
	// contains filtered or unexported fields
}
```

Cache memoizes a single [Loader.Load](/docs/reference/go-api/schema/#loader) invocation per instance and exposes the resolved schema module version for diagnostics. It owns no goroutines and no I/O of its own; the Loader carries those concerns.

The first [Cache.Get](/docs/reference/go-api/schema/#cacheget) invocation creates a private [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) and runs Loader.Load into it through sync.Once; every subsequent Get (including the one that loses the race) returns the same cached value or the same cached error. Errors are cached too — the load is never retried. To force a re-fetch, construct a fresh Cache with a fresh Loader.

Each Cache instance owns its own memoization and its own context. The context is the one long-lived evaluation state a Kernel holds, and no accessor exposes it: a caller that must compile a value against the schema takes the returned value's Context. The library MUST NOT expose a package-level Cache singleton; long-running consumers attach the Cache to a Kernel (or equivalent lifetime anchor) and keep that anchor alive across operations.

#### Cache.Get

```go
func (c *Cache) Get() (cue.Value, error)
```

Get returns the schema [cue.Value](https://pkg.go.dev/cuelang.org/go/cue#Value), invoking the underlying Loader at most once per Cache instance. Concurrent first-call invocations are serialized via sync.Once; the call that wins creates the cache's private context, runs Loader.Load with it, and the rest observe the cached result. The context lives as long as the cached value does and is reachable only through that value.

Returns the zero cue.Value and a non-nil error if Loader.Load fails; the error is cached and subsequent calls return it without re-invoking the Loader.

#### Cache.ResolvedVersion

```go
func (c *Cache) ResolvedVersion() string
```

ResolvedVersion returns the schema module version that the underlying Loader resolved during the first successful [Cache.Get](/docs/reference/go-api/schema/#cacheget) (e.g. "v2.0.0-beta.1" when the default identifier resolved to that instance).

Returns the empty string before the first successful Get, after a failed Get, or when the Loader does not surface a resolved version. The value is diagnostic-only: callers SHOULD log it but MUST NOT branch behavior on it.

### CatalogMetadata

```go
type CatalogMetadata struct {
	// ModulePath is the CUE registry module path the catalog is published
	// under, major suffix included. Example:
	// "opmodel.dev/catalogs/opm@v4".
	ModulePath string `json:"modulePath"`

	// Version is the catalog build version (semver), the value stamped onto
	// every member's metadata.catalogVersion.
	Version string `json:"version"`

	// FQN is the catalog's fully qualified name. Core derives it as the
	// module path itself; the version does not join it.
	FQN string `json:"fqn,omitempty"`

	// Description is a brief description of the catalog.
	Description string `json:"description,omitempty"`

	// Labels from the catalog definition.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the catalog definition.
	Annotations map[string]string `json:"annotations,omitempty"`
}
```

CatalogMetadata is the canonical decoded catalog-level metadata. A catalog carries no name: its identity is the module path it is published under and the version stamped on every member it ships, so ModulePath and Version are the two fields core declares required with no default. FQN is core's derivation and equals ModulePath; it is decoded so a caller reading provenance off the artifact does not have to know that.

### InstanceMetadata

```go
type InstanceMetadata struct {
	// Name is the instance name (from --name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// FQN is the instance's OWN fully qualified name
	// (registryPath:name:namespace), defined by core v2 on
	// #ModuleInstance.metadata and decoded with the rest of the metadata.
	// Distinct from the source module's FQN, which lives on the instance's
	// Package at the module-metadata path.
	FQN string `json:"fqn,omitempty"`

	// UUID is the instance identity UUID.
	// Computed by CUE as SHA1(OPMNamespace, moduleUUID:name:namespace).
	UUID string `json:"uuid"`

	// Labels are the merged instance labels (module labels + standard opm labels).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged instance annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}
```

InstanceMetadata contains instance-level identity information. Used for inventory tracking, resource labeling, and CLI output.

### Loader

```go
type Loader interface {
	Load(ctx *cue.Context) (cue.Value, error)
}
```

Loader resolves the OPM core CUE schema and returns it as a built [cue.Value](https://pkg.go.dev/cuelang.org/go/cue#Value). Implementations MUST return a value whose definitions (#Module, #ModuleInstance, #Platform, #Resource, #Trait, #ComponentTransformer, …) are reachable via LookupPath.

The library exposes exactly one Loader implementation: [OCILoader](/docs/reference/go-api/schema/#ociloader). Any other Loader satisfying the interface is internal-only and MUST NOT appear in the public API surface.

### ModuleMetadata

```go
type ModuleMetadata struct {
	// Name is the canonical module name from module.metadata.name (kebab-case).
	Name string `json:"name"`

	// Description is a brief description of the module.
	Description string `json:"description,omitempty"`

	// ModulePath is the CUE registry module path from metadata.modulePath.
	// This is the registry path (e.g., "opmodel.dev/modules"), NOT a filesystem path.
	ModulePath string `json:"modulePath"`

	// Version is the module version (semver).
	Version string `json:"version"`

	// FQN is the fully qualified module name (modulePath/name:version).
	// Example: "opmodel.dev/modules/my-app:1.0.0"
	FQN string `json:"fqn"`

	// UUID is the module identity UUID (from #Module.metadata.identity).
	UUID string `json:"uuid"`

	// Labels from the module definition (pre-build, author-declared).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module definition.
	Annotations map[string]string `json:"annotations,omitempty"`
}
```

ModuleMetadata contains module-level identity and version information. This is the module's canonical metadata, distinct from the instance it is deployed as. Populated by module.NewModuleFromValue.

### OCILoader

```go
type OCILoader struct {
	// Module is the schema module identifier. Empty means
	// [DefaultSchemaModule], the pinned core release.
	//
	// A bare major form ("…@v0") is automatically expanded to "…@v0.latest"
	// before calling [load.Instances]; CUE's standalone-package loader
	// requires either a fully qualified version, "@latest", or
	// "<major>.latest" outside a module context.
	Module string

	// Registry overrides CUE_REGISTRY for this load. Empty inherits from
	// the process environment.
	Registry string

	// CacheDir overrides CUE_CACHE_DIR for this load. Empty inherits from
	// the process environment (or CUE's default ~/.cache/cuelang/).
	CacheDir string
}
```

OCILoader resolves the OPM core schema through CUE's module system. It is the canonical and only public [Loader](/docs/reference/go-api/schema/#loader) implementation.

The zero value is a valid Loader: empty fields resolve via process environment (CUE\_REGISTRY, CUE\_CACHE\_DIR) and the [DefaultSchemaModule](/docs/reference/go-api/schema/#defaultschemamodule) identifier. Explicit field values override environment values.

OCILoader.Load does not mutate process state (no os.Setenv); env overrides are plumbed into [load.Config.Env](https://pkg.go.dev/cuelang.org/go/cue/load#Config.Env) for the single load call.

#### OCILoader.Load

```go
func (l OCILoader) Load(ctx *cue.Context) (cue.Value, error)
```

Load implements [Loader](/docs/reference/go-api/schema/#loader).

#### OCILoader.PinnedVersion

```go
func (l OCILoader) PinnedVersion() (string, bool)
```

PinnedVersion reports, without any I/O, the exact core release the loader's module identifier names: the version suffix of [OCILoader.Module](/docs/reference/go-api/schema/#ociloader) (or of [DefaultSchemaModule](/docs/reference/go-api/schema/#defaultschemamodule) when Module is empty) and true when that suffix is a full release ("v2.0.0-beta.1"), or ("", false) when the identifier names a bare major ("opmodel.dev/core@v2", resolved to ".latest" only by a load) or is not a module identifier at all.

A kernel whose loader pins a release reads the core release its synthesized instances import from here instead of loading the schema; a bare-major loader resolves it through the schema cache.

### PlatformMetadata

```go
type PlatformMetadata struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
```

PlatformMetadata is the canonical decoded platform-level metadata. Type is the top-level #Platform.type field hoisted into the metadata projection so callers see one Go-level identity record per Platform artifact.
