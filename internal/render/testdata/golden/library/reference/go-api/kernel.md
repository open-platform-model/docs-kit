---
title: "opm/kernel"
description: "Package kernel exposes the OPM runtime as a single struct, Kernel."
type: reference
---

```go
import "github.com/open-platform-model/library/opm/kernel"
```

Package kernel exposes the OPM runtime as a single struct, [Kernel](/docs/reference/go-api/kernel/#kernel).

Kernel owns its [\*schema.Cache](/docs/reference/go-api/schema/#cache) for its entire lifetime and no build context: every operation that evaluates CUE creates its own [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context), builds in it, and lets it go when it returns. Construction is [New](/docs/reference/go-api/kernel/#new) plus two options, [WithSchemaLoader](/docs/reference/go-api/kernel/#withschemaloader) and [WithRegistry](/docs/reference/go-api/kernel/#withregistry); the kernel exposes no injection slot that no kernel operation reads. Downstream binaries (CLI, controller, Crossplane function) construct one Kernel per process and call methods on it instead of importing the individual loader / module / validate packages.

### Surface

One tier: every artifact a frontend can hold comes from an acquire verb, or from the package constructor it already has a value for.

\- [Kernel.AcquireModuleFromRegistry](/docs/reference/go-api/kernel/#kernelacquiremodulefromregistry) and [Kernel.AcquireModuleFromDir](/docs/reference/go-api/kernel/#kernelacquiremodulefromdir) return a source-carrying [\*module.Module](/docs/reference/go-api/module/#module); - [Kernel.AcquireCatalogFromRegistry](/docs/reference/go-api/kernel/#kernelacquirecatalogfromregistry) and [Kernel.AcquireCatalogFromDir](/docs/reference/go-api/kernel/#kernelacquirecatalogfromdir) return a source-carrying [\*catalog.Catalog](/docs/reference/go-api/catalog/#catalog), the kind admitted by ADR-009: the kernel reads it and derives from it ([catalog.Catalog.Provides](/docs/reference/go-api/catalog/#catalogprovides), [catalog.Catalog.Requires](/docs/reference/go-api/catalog/#catalogrequires)) and judges nothing beyond its shape; - [Kernel.AcquirePlatformFromDir](/docs/reference/go-api/kernel/#kernelacquireplatformfromdir) returns a [\*platform.Platform](/docs/reference/go-api/platform/#platform); - [Kernel.AcquireInstanceFromDir](/docs/reference/go-api/kernel/#kernelacquireinstancefromdir) returns a validated [\*module.Instance](/docs/reference/go-api/module/#instance), with optional values as trailing [Source](/docs/reference/go-api/kernel/#source) values; - [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance) builds one from typed inputs ([InstanceInput](/docs/reference/go-api/kernel/#instanceinput)); the module it takes comes from the two module acquire verbs, and the core release the synthesized package imports is the kernel's pinned schema release, read from the configured [schema.OCILoader](/docs/reference/go-api/schema/#ociloader) with no schema load when it pins an exact release (the default) and resolved through the schema cache otherwise; - [Kernel.ValidateConfigDetailed](/docs/reference/go-api/kernel/#kernelvalidateconfigdetailed) validates layered values; - [Kernel.Render](/docs/reference/go-api/kernel/#kernelrender) renders an instance against a platform.

There is no second, value-only tier: a caller that wants the raw value of an acquired artifact reads its Package field, and a caller holding a value it built itself calls [module.NewModuleFromValue](/docs/reference/go-api/module/#newmodulefromvalue) or [platform.NewPlatformFromValue](/docs/reference/go-api/platform/#newplatformfromvalue) directly. The registry mapping is [WithRegistry](/docs/reference/go-api/kernel/#withregistry) for every one of these operations, the schema cache and the compilation of file-backed values sources (a values file that imports a registry module) included; no verb takes a per-call override. Absent the option, every operation inherits the process CUE\_REGISTRY and applies no default; the mapping is plumbed into the operation's load configuration and never written back to the environment.

### Every operation shares nothing

The Kernel holds no [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) (ADR-007). Each acquire verb, synthesis, validation and render creates a context for the call, builds in it, and returns; the values an operation returns (an artifact's Package, a validated value) keep that operation's runtime alive for exactly as long as the caller holds them, and the Kernel retains nothing. Memory held by a long-lived Kernel is therefore bounded by the artifacts its caller holds, not by the number of operations it has run. The cross-artifact verbs read only Metadata and Source from their inputs, with one exception: Render reads whether the platform's Package carries #contracts.providedBy (the core floor), a read-only path lookup and presence test with no unification and no fill. Nothing is built into an input's context, so a module acquired by one Kernel synthesizes on another and an instance from either renders on a third, and one acquired platform may be shared by concurrent renders. No method returns or accepts a [\*cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context); a caller that must compile a value against the schema takes the context of the value [schema.Cache.Get](/docs/reference/go-api/schema/#cacheget) returns.

### Goroutine safety

A single Kernel is safe for concurrent use across its own method calls: no operation shares evaluation state with another, and the schema cache is memoized under synchronization into a private context of its own. A consumer that needs concurrent operations shares one Kernel across its goroutines; there is nothing to gain from constructing more than one. Concurrency is across operations, never within one.

[Kernel.Render](/docs/reference/go-api/kernel/#kernelrender) shares nothing between renders (ADR-005). Each render is its own CUE build in a fresh cue.Context created for that call and dropped when Render returns; no built value is retained between calls, and a caller cannot obtain one to hold. A consumer rendering from several goroutines calls Render on one Kernel with no mutex, and may share one acquired platform across them: each render builds the platform from its Source in its own context, and reads the shared Package only for the core floor (a read-only lookup of #contracts.providedBy, no unification, no fill), so concurrent renders never write to it. There is no materialized platform to share and no serialised render path; the earlier shared-platform contract (ADR-002, renders filling one shared platform value) is superseded, not supported.

A render is single-threaded and its working set grows with the module, so a render pool is sized by memory rather than by core count: about 61 MB plus 7.75 MB per component per concurrent render, and throughput saturates at roughly physical cores divided by 1.6 renders in flight. Size against the largest module the pool will see.

### One-Kernel-per-process example

func renderAll(ctx context.Context, k \*kernel.Kernel, platformDir string, instanceDirs \[\]string) error \{ plat, err := k.AcquirePlatformFromDir(ctx, platformDir) // once; the platform is shared as data if err \!= nil \{ return err \} var wg sync.WaitGroup errs := make(chan error, len(instanceDirs)) for \_, dir := range instanceDirs \{ wg.Add(1) go func(dir string) \{ defer wg.Done() inst, err := k.AcquireInstanceFromDir(ctx, dir) // the one Kernel, concurrently if err \!= nil \{ errs \<- err return \} if \_, err := k.Render(ctx, kernel.RenderInput\{Instance: inst, Platform: plat, RuntimeName: "opm-cli"\}); err \!= nil \{ errs \<- err \} \}(dir) \} wg.Wait() close(errs) for err := range errs \{ if err \!= nil \{ return err \} \} return nil \}

### Rendering

[Kernel.Render](/docs/reference/go-api/kernel/#kernelrender) is the kernel's single render verb. It takes a source-carrying instance ([Kernel.AcquireInstanceFromDir](/docs/reference/go-api/kernel/#kernelacquireinstancefromdir) or [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance)) and a source-carrying platform ([Kernel.AcquirePlatformFromDir](/docs/reference/go-api/kernel/#kernelacquireplatformfromdir): a platform is a CUE module on disk that imports its catalogs), stages one generated render module that imports both (an on-disk input in place, an overlay-mode input served from memory; the per-render staging directory holds only the generated module), builds it once, and decodes the matching verdicts ([RenderDiagnostics](/docs/reference/go-api/kernel/#renderdiagnostics)) and the rendered output ([RenderResult.Compiled](/docs/reference/go-api/kernel/#renderresult), one entry per rendered object as a [\*Compiled](/docs/reference/go-api/kernel/#compiled) carrying instance, component and transformer provenance). Matching and transformer execution are CUE inside the build, not Go; the build reports its verdicts as data and the kernel's fail-closed gate turns them into a [\*RenderError](/docs/reference/go-api/kernel/#rendererror) that carries the full diagnostics, with the typed causes reachable through errors.As and joined in this order: a contract collision (a key more than one enabled registry entry defines, read from core's #contracts.collisions and collidingEntries; the opm/errors ContractCollisionsError, first because the other rows are read against the inventory a collision distorts, platform-wide and standing under the skip switch), an unresolved demand, an over-subscribed provider-fulfilled contract (read from core's #contracts.overSubscribed and providedBy), an unmatched component, and last the NotRoutableError catch-all, raised only when core's decoded #contracts.routable reads false and no collision or over-subscription row explains it. The rows are the ones the platform's Contracts() reads, so the render refuses on a collision or an over-subscription exactly when the inventory reads not routable. Catalog version skew (the instance module requiring a newer OPM-namespace build than the platform carries) marks a resolved-versions row Newer by default ([SkewWarn](/docs/reference/go-api/kernel/#skewwarn)) or refuses before evaluation ([SkewRefuse](/docs/reference/go-api/kernel/#skewwarn)).

The render module's own gate field agrees with the kernel: it errors exactly when the kernel refuses on a decoded verdict, so a staged module is self-refusing under a plain cue eval. The kernel decides from the decoded rows and the decoded routable verdict only, never by reading the gate. Each typed cause carries its diagnostics rows unchanged and wraps nothing. Inputs are never mutated, and the staging directory is removed on return, success or failure; refusals before evaluation (a missing Source, a platform whose core predates #contracts.providedBy, an uncovered OPM-namespace path, skew under [SkewRefuse](/docs/reference/go-api/kernel/#skewwarn), a local replacement without the opt-in) are plain errors. The core floor runs before anything is staged: a platform module pinning core older than [schema.ProvidedBySince](/docs/reference/go-api/schema/#providedbysince) is refused with an error wrapping the opm/errors PlatformCoreTooOldError, and the render never falls back to a provider count of its own.

A render result carries no presentation strings. The three advisory facts a render can report are rows on the diagnostics: an unhandled optional trait on RenderDiagnostics.UnhandledTraits, a module requiring a newer build than the platform carries on a RenderDiagnostics.ResolvedVersions row with Newer set, and a demand skipped under [RenderInput.SkipUnprovided](/docs/reference/go-api/kernel/#renderinput) on a RenderDiagnostics.Skipped row. A frontend words all three:

for comp, traits := range result.Diagnostics.UnhandledTraits \{ for \_, fqn := range traits \{ log.Printf("component %q: trait %q is unhandled", comp, fqn) \} \} for \_, r := range result.Diagnostics.ResolvedVersions \{ if r.Newer \{ log.Printf("%s: module requires %s, platform carries %s", r.Path, r.ModuleVersion, r.PlatformVersion) \} \} for \_, s := range result.Diagnostics.Skipped \{ log.Printf("component %q: skipped %s %q, no provider on this platform (component rendered: %t)", s.Component, s.Kind, s.FQN, \!s.ComponentOmitted) \}

A dry run is Render with the output discarded: the build evaluates every matched pair regardless, and RenderDiagnostics carries the pairing diagnosis (Pairs, Unmatched, Unresolved, Skipped, Unify, UnhandledTraits, OverSubscribed, Collisions, Routable, ResolvedVersions). There is no separate match verb.

A demand is unprovided when its contract declares fulfilment "provider" and no enabled registry entry carries a transformer requiring the key: the key is absent from core's #contracts.providedBy, the count the single-provider guard reads. The build marks every unresolved row with this fact on every render (UnresolvedDemand.Unprovided in opm/errors, and a "provider-fulfilled, no provider on this platform" suffix on its message), so a frontend can offer its skip switch without re-deriving fulfilment. Under [RenderInput.SkipUnprovided](/docs/reference/go-api/kernel/#renderinput) the build moves exactly those demands out of the refusal: a skipped trait demand leaves its component rendering every pair it matched, and a skipped resource demand omits the whole component (no pair of it renders, it is not reported unmatched, and every skipped row of it carries ComponentOmitted). Every other refusal stands under the switch: a catalog-fulfilled unresolved demand, a provider that exists but did not match, a contract collision, an over-subscribed contract and an unmatched component. The switch is the caller's, per render; the kernel never sets it, and a frontend names its own flag.

An input's own cue.mod/local-module.cue (a developer redirecting a dependency to a directory or another module) reaches the render only under [RenderInput.LocalReplacements](/docs/reference/go-api/kernel/#renderinput). Off, the default, Render refuses before staging an input whose file carries a replacement rather than silently rendering against the published pin. On, the replacements are promoted into the render module under the precedence dependencies get (the platform's whole, the instance's only for paths the platform's dependency list does not name) and each honoured one is a [RenderDiagnostics.Replacements](/docs/reference/go-api/kernel/#renderdiagnostics) row naming the path, the target and the input that supplied it; a replaced path keeps its pinned versions on ResolvedVersions. A relative directory target is resolved against the input's own module root; a replaced path the promoted list lacks is listed with a placeholder version of its major so coverage holds; a version-less dependency no promoted replacement covers is refused naming the path and the input; the rows are path-sorted, and an input without the file renders identically under either setting. The switch is a security boundary, not an extension point: it is the one place a render may read a directory an artifact names, so a frontend sets it for a developer's checkout and never for an artifact it did not author (the operator never sets it). The frontend words the rows (an instance replacement the platform made inert is not a row, so the frontend computes the inert set from the file it read):

for \_, r := range result.Diagnostics.Replacements \{ log.Printf("%s: served from %s (%s local-module.cue)", r.Path, r.Target, r.By) \}

Render consumes the instance as processed: values are validated where they are applied. [Kernel.AcquireInstanceFromDir](/docs/reference/go-api/kernel/#kernelacquireinstancefromdir) unifies its trailing [Source](/docs/reference/go-api/kernel/#source) values inside the package build and checks them against the module's `#config` at their own positions; [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance) does the same for [InstanceInput.Values](/docs/reference/go-api/kernel/#instanceinput), rendering them into the synthesized package; both then assert concreteness on the whole built spec. Render performs no validation pass of its own.

### Configuration validation

One primitive forms the validation surface: [Kernel.ValidateConfigDetailed](/docs/reference/go-api/kernel/#kernelvalidateconfigdetailed) accepts an ordered slice of [Source](/docs/reference/go-api/kernel/#source), compiles each in the schema's own context, unifies in stack order, then validates the merged value against the schema with concreteness enforced. A single value is a one-element slice. A [Source](/docs/reference/go-api/kernel/#source) is CUE source bytes plus their origin and is bound to no context; per-source attribution flows through [token.Pos.Filename](https://pkg.go.dev/cuelang.org/go/cue/token#Pos.Filename), populated from [cue.Filename](https://pkg.go.dev/cuelang.org/go/cue#Filename)(Origin) when the kernel compiles the source where it is used. Use [Kernel.LoadSourceFromFile](/docs/reference/go-api/kernel/#kernelloadsourcefromfile) or [Kernel.LoadSourceFromBytes](/docs/reference/go-api/kernel/#kernelloadsourcefrombytes) to construct sources that are checked for syntax up front; a frontend needs no [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) of its own. A file-backed source is loaded at its file's directory, so its imports resolve through the kernel's [WithRegistry](/docs/reference/go-api/kernel/#withregistry) mapping on every path that compiles it. There is no partial-mode entry: partial validation is an internal attribution pass under AcquireInstanceFromDir with extra values, not a public contract.

Because the sources are compiled into the schema value's own context, validating against one acquired artifact from several goroutines at once shares that artifact's context; a consumer that needs that gives each goroutine its own acquired artifact. The kernel's own verbs never share a context this way.

The primitive returns CUE-native errors. Walk them via [cuelang.org/go/cue/errors.Errors](https://pkg.go.dev/cuelang.org/go/cue/errors#Errors) / [cuelang.org/go/cue/errors.Positions](https://pkg.go.dev/cuelang.org/go/cue/errors#Positions), or print via [cuelang.org/go/cue/errors.Print](https://pkg.go.dev/cuelang.org/go/cue/errors#Print). Presentation belongs to the frontend — the kernel does not ship a formatter.

A caller holding a \*module.Module or \*module.Instance composes its ConfigSchema() accessor with the primitive, e.g. k.ValidateConfigDetailed(m.ConfigSchema(), \[\]kernel.Source\{src\}).

## Types

### Compiled

```go
type Compiled struct {
	// Value is the CUE value produced by the transformer. Concrete and
	// fully evaluated — safe to encode directly to YAML or JSON.
	Value cue.Value

	// Instance is the name of the ModuleInstance that produced this resource.
	Instance string

	// Component is the source component name within the instance.
	Component string

	// Transformer is the FQN of the transformer that produced this resource.
	Transformer string
}
```

Compiled is the terminal output of an OPM render: [Kernel.Render](/docs/reference/go-api/kernel/#kernelrender) emits \*Compiled values carrying the rendered CUE value plus OPM provenance. It carries no platform-native fields — keeping platform vocabulary out of the kernel keeps it platform-neutral, and each consumer wraps \*Compiled in its own resource type.

### InstanceInput

```go
type InstanceInput struct {
	// Module is the source #Module the instance deploys. Required, and it
	// MUST carry its staged source — acquire it with
	// [Kernel.AcquireModuleFromRegistry] or [Kernel.AcquireModuleFromDir].
	// Its metadata.modulePath / metadata.version identify the module the
	// synthesized package imports; because that import resolves to the
	// module's ROOT package, a module acquired from a subdirectory is
	// refused.
	Module *module.Module

	// Name is the instance name (metadata.name). Required. It must satisfy
	// the schema's #NameType regex; a violation surfaces as a CUE
	// unification error from the synthesized build.
	Name string

	// Namespace is the target namespace (metadata.namespace). Required.
	Namespace string

	// Values are the configuration sources, in stack order — the same
	// [Source] type [Kernel.ValidateConfigDetailed] and
	// [Kernel.AcquireInstanceFromDir] take. They are compiled in the call's
	// own context, unified, rendered into the synthesized package's values
	// file so the merge is the schema's own values unification in CUE, and
	// checked against the module's #config at their own positions after the
	// build.
	//
	// Empty means "no values supplied": the values path is left unfilled and
	// the concreteness check then fails unless every #config field has a
	// default. Synthesis NEVER falls back to Module.debugValues; layering a
	// debug-values overlay is frontend policy.
	Values []Source

	// Labels and Annotations layer over the schema's stamped
	// module-instance.opmodel.dev/{name,uuid} labels. CUE unification merges
	// caller-supplied entries with schema-stamped ones; caller-supplied keys
	// MUST NOT collide with the schema's reserved keys.
	Labels      map[string]string
	Annotations map[string]string
}
```

InstanceInput is the typed input [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance) takes. Module, Name and Namespace are required; the rest are optional and are filled into the instance only when present, so an empty field never displaces a schema-derived one.

It carries no schema cache and no [\*cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context): the Kernel owns the schema cache, synthesis builds in a context it creates for the call, and the registry mapping is the Kernel's [WithRegistry](/docs/reference/go-api/kernel/#withregistry).

### Kernel

```go
type Kernel struct {
	// contains filtered or unexported fields
}
```

Kernel is the public anchor type for the OPM runtime. It owns a [\*schema.Cache](/docs/reference/go-api/schema/#cache) for its lifetime and no build context: every operation that evaluates CUE creates its own [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context), builds in it, and lets it go when it returns, so the values an operation returns (an artifact's Package, a validated value) keep that operation's runtime alive for exactly as long as the caller holds them, and nothing else does.

A single Kernel is safe for concurrent use across its method calls — see the package documentation.

The Kernel owns exactly one [\*schema.Cache](/docs/reference/go-api/schema/#cache) for its lifetime. Long- running consumers (operator, server) keep one Kernel alive for the process to reuse the in-process schema cache; constructing a fresh Kernel per request pays the schema-fetch cost on every cold disk cache. The CUE module cache on disk is shared across Kernels.

#### New

```go
func New(opts ...Option) *Kernel
```

New constructs a [Kernel](/docs/reference/go-api/kernel/#kernel) with default dependencies and applies the supplied options. The one default is the schema cache: a fresh [\*schema.Cache](/docs/reference/go-api/schema/#cache) backed by a [schema.OCILoader](/docs/reference/go-api/schema/#ociloader) whose Registry is the kernel's [WithRegistry](/docs/reference/go-api/kernel/#withregistry) value (empty when the option is absent, which resolves [schema.DefaultSchemaModule](/docs/reference/go-api/schema/#defaultschemamodule) against CUE\_REGISTRY / CUE\_CACHE\_DIR from the process environment).

Seeding the loader from the registry option is what makes [WithRegistry](/docs/reference/go-api/kernel/#withregistry) the ONE mapping every kernel operation resolves through — schema fetch included — so a consumer cannot end up rendering against an explicit mapping while its schema silently resolves from the process environment. An explicit [WithSchemaLoader](/docs/reference/go-api/kernel/#withschemaloader) still wins, whatever order the options are given in: the loader is chosen after every option has been applied.

New never returns nil, creates no [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) and evaluates nothing. The returned Kernel is safe for concurrent use across method calls.

New does NOT trigger a schema load, and on a pinned loader (the default) no Kernel method does either: only a bare-major loader's instance synthesis, or a caller's own [schema.Cache.Get](/docs/reference/go-api/schema/#cacheget), runs the lazy fetch.

#### Kernel.AcquireCatalogFromDir

```go
func (k *Kernel) AcquireCatalogFromDir(_ context.Context, dirPath string) (*catalog.Catalog, error)
```

AcquireCatalogFromDir loads a #Catalog CUE package from a directory and returns it as a typed, source-carrying [\*catalog.Catalog](/docs/reference/go-api/catalog/#catalog). It is the directory peer of [Kernel.AcquireCatalogFromRegistry](/docs/reference/go-api/kernel/#kernelacquirecatalogfromregistry), and it mirrors [Kernel.AcquireModuleFromDir](/docs/reference/go-api/kernel/#kernelacquiremodulefromdir) exactly: the package is evaluated and shape-gated as the registry path gates a fetched catalog, [catalog.NewCatalogFromValue](/docs/reference/go-api/catalog/#newcatalogfromvalue) constructs the typed artifact, and [catalog.Source](/docs/reference/go-api/catalog/#source) is stamped in OVERLAY mode — Root the enclosing module root (the nearest ancestor holding cue.mod/module.cue, the directory itself when it is the root or when no ancestor holds one), Pkg the package directory relative to it, and Overlay every .cue file under Root (the module's own cue.mod/module.cue included) keyed by its absolute path.

Overlay mode is what makes [catalog.Catalog.Requires](/docs/reference/go-api/catalog/#catalogrequires) answer identically whichever route acquired the catalog: the committed cue.mod/module.cue is carried on the artifact either way.

The package is built in a [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) created for the call; Catalog.Package keeps it alive for as long as the caller holds the catalog. The registry mapping for the catalog's own imports is the kernel's ([WithRegistry](/docs/reference/go-api/kernel/#withregistry)), applied via the load configuration's environment and never os.Setenv. The caller's directory is never written to.

Shape-gate failures propagate unchanged (missing directory, no package, or a sentinel such as [oerrors.ErrWrongKind](/docs/reference/go-api/errors/#errinvalidpackage)); no partial catalog is returned.

#### Kernel.AcquireCatalogFromRegistry

```go
func (k *Kernel) AcquireCatalogFromRegistry(ctx context.Context, modPath, version string) (*catalog.Catalog, error)
```

AcquireCatalogFromRegistry loads a #Catalog published in an OCI registry by its major-qualified path (e.g. "opmodel.dev/catalogs/opm@v4") and version (e.g. "v4.3.0"), in a [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) created for the call, through the kernel's configured registry (set via [WithRegistry](/docs/reference/go-api/kernel/#withregistry), inheriting CUE\_REGISTRY from the process environment when unset). It is the registry peer of [Kernel.AcquireCatalogFromDir](/docs/reference/go-api/kernel/#kernelacquirecatalogfromdir) and the exact counterpart of [Kernel.AcquireModuleFromRegistry](/docs/reference/go-api/kernel/#kernelacquiremodulefromregistry): one fetch routine serves both, and the only thing that differs is the shape it gates to (ADR-009).

It returns a decoded [\*catalog.Catalog](/docs/reference/go-api/catalog/#catalog) whose staged source ([catalog.Source](/docs/reference/go-api/catalog/#source)) is populated in overlay mode, so [catalog.Catalog.Requires](/docs/reference/go-api/catalog/#catalogrequires) reads the catalog's committed cue.mod/module.cue without a second fetch. A caller that wants the raw value reads Catalog.Package, which keeps the call's runtime alive for as long as the caller holds the catalog.

The catalog is evaluated and shape-gated (concrete kind == "Catalog"; concrete metadata.modulePath and metadata.version), never fully schema-validated, which remains the kernel's contract. A gate failure propagates unchanged, wrapping the shared sentinels ([oerrors.ErrWrongKind](/docs/reference/go-api/errors/#errinvalidpackage) for an artifact of another kind or of none); no partial catalog is returned. Unlike the module path, the fetched coordinate is not compared against the catalog's declared one — see loader.FetchModule for why that check is the module's alone.

The kernel judges nothing beyond the shape: what a catalog provides and what it requires are reported by [catalog.Catalog.Provides](/docs/reference/go-api/catalog/#catalogprovides) and [catalog.Catalog.Requires](/docs/reference/go-api/catalog/#catalogrequires) on demand, and what either means is the caller's.

#### Kernel.AcquireInstanceFromDir

```go
func (k *Kernel) AcquireInstanceFromDir(_ context.Context, dirPath string, values ...Source) (*module.Instance, error)
```

AcquireInstanceFromDir loads a #ModuleInstance CUE package from a directory and returns it as a validated, source-carrying [\*module.Instance](/docs/reference/go-api/module/#instance). The package is evaluated, run through the instance shape gate and then the kernel's instance processing — concreteness on the whole built spec, so the package must already be fully concrete, as an authored instance package is, and metadata decoding — and [module.Instance.Source](/docs/reference/go-api/module/#instance) is stamped in on-disk mode: Overlay is nil, Root is the enclosing module root (the nearest ancestor holding cue.mod/module.cue, the directory itself when it is the root) and Pkg the package directory relative to it, so a package in a subdirectory of its module imports correctly from a follow-on build.

The trailing values sources — the same [Source](/docs/reference/go-api/kernel/#source) type [Kernel.ValidateConfigDetailed](/docs/reference/go-api/kernel/#kernelvalidateconfigdetailed) takes, in stack order — layer extra values onto the package: the sources are compiled in the call's own context, the on-disk files under the module root are read into an in-memory overlay, the unified sources are rendered as a package file declaring `values` (opm-values.cue) beside the package's own files, and the package is built in one pass through the same instance shape gate, so the merge is the schema's own values unification in CUE. Nothing is filled from Go and nothing is written into the caller's directory. The returned Source is then overlay mode: the same Root and Pkg, with Overlay carrying every on-disk .cue file plus the rendered values file, exactly as load.Config.Overlay expects, so [Kernel.Render](/docs/reference/go-api/kernel/#kernelrender) imports the layered package by source. A source conflicting with the package's own values or the module's #config fails acquisition with the conflict attributed to the source (its Origin), exactly as layered validation reports it.

Passing no sources is the "no values supplied" path: the package is built from disk as authored and the Source stays on-disk mode.

The build runs in a [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) created for the call; Instance.Package keeps it alive for as long as the caller holds the instance. This is the same bar [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance) output meets. Loader failures propagate unchanged (missing directory, no package, or a shape-gate sentinel); a non-concrete package surfaces the concreteness error, framed `instance "<name>": …`. No partial instance is returned.

#### Kernel.AcquireModuleFromDir

```go
func (k *Kernel) AcquireModuleFromDir(_ context.Context, dirPath string) (*module.Module, error)
```

AcquireModuleFromDir loads a #Module CUE package from a directory and returns it as a typed, source-carrying [\*module.Module](/docs/reference/go-api/module/#module). It is the directory peer of [Kernel.AcquireModuleFromRegistry](/docs/reference/go-api/kernel/#kernelacquiremodulefromregistry): the package is evaluated and shape-gated exactly as the registry path gates a fetched module, [module.NewModuleFromValue](/docs/reference/go-api/module/#newmodulefromvalue) constructs the typed artifact, and [module.Source](/docs/reference/go-api/module/#source) is stamped in OVERLAY mode — Root the enclosing module root (the nearest ancestor holding cue.mod/module.cue, the directory itself when it is the root or when no ancestor holds one), Pkg the package directory relative to it, and Overlay every .cue file under Root (the module's own cue.mod/module.cue included) keyed by its absolute path.

Stamping the overlay is what makes the acquired module a valid [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance) input ([module.Module.HasSource](/docs/reference/go-api/module/#modulehassource) reports true): synthesis stages the instance package inside the module's own tree, so a frontend rendering from a module directory no longer walks that tree itself. Because the synthesized package imports the module by its module path — which resolves to the module's ROOT package — synthesis refuses a module whose Source.Pkg is non-empty; acquiring a subdirectory package is still valid for reading its value and metadata.

The package is built in a [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) created for the call; Module.Package keeps it alive for as long as the caller holds the module. The registry mapping is the kernel's ([WithRegistry](/docs/reference/go-api/kernel/#withregistry)), applied via the load configuration's environment and never os.Setenv. The caller's directory is never written to.

Shape-gate failures propagate unchanged (missing directory, no package, or a sentinel such as [oerrors.ErrWrongKind](/docs/reference/go-api/errors/#errinvalidpackage)); no partial module is returned.

#### Kernel.AcquireModuleFromRegistry

```go
func (k *Kernel) AcquireModuleFromRegistry(ctx context.Context, modPath, version string) (*module.Module, error)
```

AcquireModuleFromRegistry loads a #Module published in an OCI registry by its major-qualified path (e.g. "example.com/modules/hello@v0") and version (e.g. "v0.0.2"), in a [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) created for the call, through the kernel's configured registry (set via [WithRegistry](/docs/reference/go-api/kernel/#withregistry), inheriting CUE\_REGISTRY from the process environment when unset). It returns a decoded [\*module.Module](/docs/reference/go-api/module/#module) whose staged source ([module.Source](/docs/reference/go-api/module/#source)) is populated, so the module can be reused as the main module of a follow-on build — notably by [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance), which stages the instance inside the module's own root so the module's already-tidied cue.mod/module.cue drives transitive dependency resolution. A caller that wants only the raw module value reads Module.Package, which keeps the call's runtime alive for as long as the caller holds the module.

#### Kernel.AcquirePlatformFromDir

```go
func (k *Kernel) AcquirePlatformFromDir(_ context.Context, dirPath string) (*platform.Platform, error)
```

AcquirePlatformFromDir loads a #Platform CUE package from a directory and returns it as a typed, source-carrying [\*platform.Platform](/docs/reference/go-api/platform/#platform). The package is evaluated and run through the platform shape gate, then [platform.NewPlatformFromValue](/docs/reference/go-api/platform/#newplatformfromvalue) constructs the typed artifact and [platform.Platform.Source](/docs/reference/go-api/platform/#platform) is stamped in on-disk mode: Root is the enclosing module root (the nearest ancestor holding cue.mod/module.cue, the directory itself when it is the root), Pkg the package directory relative to it, and Overlay nil.

It is the directory peer of [Kernel.AcquireModuleFromRegistry](/docs/reference/go-api/kernel/#kernelacquiremodulefromregistry) ("Acquire" returns a typed artifact that knows where its source lives) and the only way to obtain a platform: a caller that wants the raw value reads Platform.Package, which keeps the call's [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) alive for as long as the caller holds the platform; the kernel retains nothing. The registry mapping used for the platform's catalog imports is the kernel's ([WithRegistry](/docs/reference/go-api/kernel/#withregistry)), applied via the load configuration's environment and never os.Setenv; the verb takes no per-call override.

Loader failures propagate unchanged (missing directory, no package, or a shape-gate sentinel such as [oerrors.ErrWrongKind](/docs/reference/go-api/errors/#errinvalidpackage)); no partial platform is returned.

#### Kernel.LoadSourceFromBytes

```go
func (k *Kernel) LoadSourceFromBytes(origin string, b []byte) (Source, error)
```

LoadSourceFromBytes wraps b as a [Source](/docs/reference/go-api/kernel/#source) with Origin origin after checking that it parses as CUE. Nothing is evaluated: the kernel compiles the source with [cue.Filename](https://pkg.go.dev/cuelang.org/go/cue#Filename)(origin) in the context of the operation that uses it, so any validation error positions carry origin via [token.Pos.Filename](https://pkg.go.dev/cuelang.org/go/cue/token#Pos.Filename). A caller holding a string passes \[\]byte(s).

Returns an error positioned at origin if b does not parse (the returned [Source](/docs/reference/go-api/kernel/#source) is the zero value in that case). A payload that parses but does not evaluate, or violates the schema it meets, fails in the operation that applies it, positioned at origin.

#### Kernel.LoadSourceFromFile

```go
func (k *Kernel) LoadSourceFromFile(path string) (Source, error)
```

LoadSourceFromFile reads a values file from disk and returns a [Source](/docs/reference/go-api/kernel/#source) whose Origin is the file's absolute path and whose Data is the file's bytes, after checking that they parse as CUE. Nothing is evaluated here: the operation that uses the source loads it through [cuelang.org/go/cue/load.Instances](https://pkg.go.dev/cuelang.org/go/cue/load#Instances) at the file's directory (so imports resolve as they do for any CUE file, through the kernel's [WithRegistry](/docs/reference/go-api/kernel/#withregistry) mapping like every other kernel load, and never through a mutated process environment), builds it in that operation's own context, and unwraps a top-level `values:` field that exists without error (OPM values files conventionally wrap their payload in one), so the value carried into validation is the inner object.

Returns an error if the file cannot be read, or one positioned at the file's path if it does not parse.

#### Kernel.Render

```go
func (k *Kernel) Render(ctx context.Context, in RenderInput) (*RenderResult, error)
```

Render renders an instance against a platform as ONE CUE build: it stages a generated render module in a per-render temporary directory (the promoted cue.mod; directory replacements bringing both inputs in, an on-disk input in place and an overlay-mode input from memory, so the directory holds only the generated module; the embedded matching and execution glue), verifies the promoted list covers every OPM-namespace path either input requires, applies the skew policy, builds the module once in a fresh cue.Context that is dropped when Render returns, and decodes `diagnostics` and `rendered` off the built value.

The Kernel holds no context of its own, and no built value survives the call except the returned output; repeated renders share nothing. The staging directory is removed on return, success or failure. Registry resolution for the platform's catalog imports uses [WithRegistry](/docs/reference/go-api/kernel/#withregistry) when set, else the process CUE\_REGISTRY, plumbed through the load configuration only.

Refusals before evaluation (missing Source, a platform whose core predates the provider count, uncovered OPM path, skew under [SkewRefuse](/docs/reference/go-api/kernel/#skewwarn)) return plain errors; refusals after evaluation return a [\*RenderError](/docs/reference/go-api/kernel/#rendererror) carrying the decoded diagnostics. A platform whose Package carries no #contracts.providedBy (a platform module pinning core older than [schema.ProvidedBySince](/docs/reference/go-api/schema/#providedbysince)) is refused before staging, with no staging directory created, by an error wrapping [\*oerrors.PlatformCoreTooOldError](/docs/reference/go-api/errors/#platformcoretooolderror): the render never falls back to a provider count of its own.

#### Kernel.SchemaCache

```go
func (k *Kernel) SchemaCache() *schema.Cache
```

SchemaCache returns the [\*schema.Cache](/docs/reference/go-api/schema/#cache) owned by this Kernel. The same pointer is returned for the lifetime of the Kernel; callers MAY hold it across operations to ensure cache reuse.

Calling SchemaCache does NOT trigger a schema load. Only the first [schema.Cache.Get](/docs/reference/go-api/schema/#cacheget) invocation contacts CUE; the load is lazy and memoized into a context the cache owns and never exposes. A caller that must compile a value against the schema (the cli's publish gate does) takes the returned value's Context.

Typical use: read [schema.Cache.ResolvedVersion](/docs/reference/go-api/schema/#cacheresolvedversion) for diagnostics after a load has run. Nothing needs to be passed back in: a kernel whose loader names a bare major resolves the core release for [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance) through this cache on its own, and a pinned kernel (the default) runs no load at all, so a consumer that wants the diagnostic calls [schema.Cache.Get](/docs/reference/go-api/schema/#cacheget) itself.

#### Kernel.SynthesizeInstance

```go
func (k *Kernel) SynthesizeInstance(_ context.Context, in InstanceInput) (*module.Instance, error)
```

SynthesizeInstance builds a [\*module.Instance](/docs/reference/go-api/module/#instance) from typed in-memory inputs. It is the entry point for a caller that holds a Module and needs a fully validated instance, mirroring [Kernel.AcquireInstanceFromDir](/docs/reference/go-api/kernel/#kernelacquireinstancefromdir) for a directory-based CUE package; the module it takes comes from [Kernel.AcquireModuleFromRegistry](/docs/reference/go-api/kernel/#kernelacquiremodulefromregistry) or [Kernel.AcquireModuleFromDir](/docs/reference/go-api/kernel/#kernelacquiremodulefromdir).

The method stages a package importing the module inside the module's own staged source tree, renders the unified in.Values into it, and builds it once so CUE derives uuid, components, auto-secrets and standard labels and performs the values merge against the module's #config. It then checks the values sources against #config at their own positions — so a violation is reported with the source's Origin rather than the rendered values file — asserts concreteness on the whole built spec and decodes instance metadata. No additional values source is consulted.

The synthesized package imports core at the major of the kernel's schema release: read from the configured [schema.OCILoader](/docs/reference/go-api/schema/#ociloader) when it pins an exact release (the default), with no schema load, and resolved through the kernel's schema cache when it names a bare major. The release the import resolves to inside the build is the one the module's own cue.mod pins.

The build runs in a [cue.Context](https://pkg.go.dev/cuelang.org/go/cue#Context) created for the call and reads only the module's Metadata and Source, never its Package, so a module acquired by another Kernel is a valid input. The returned instance carries [module.Instance.Source](/docs/reference/go-api/module/#instance): the staged tree the build evaluated, in overlay mode, with Pkg naming the reserved instance subdirectory inside the module's staged root; Instance.Package keeps the call's context alive for as long as the caller holds the instance.

A missing required input fails before any build runs, wrapping the matching sentinel from opm/errors ([oerrors.ErrMissingModule](/docs/reference/go-api/errors/#errinvalidpackage), [oerrors.ErrMissingName](/docs/reference/go-api/errors/#errinvalidpackage), [oerrors.ErrMissingNamespace](/docs/reference/go-api/errors/#errinvalidpackage)); a module with no staged source wraps [oerrors.ErrMissingSource](/docs/reference/go-api/errors/#errinvalidpackage); a module acquired from a subdirectory of its CUE module fails stating the root-package requirement.

#### Kernel.ValidateConfigDetailed

```go
func (k *Kernel) ValidateConfigDetailed(schema cue.Value, sources []Source) (cue.Value, error)
```

ValidateConfigDetailed is the kernel's one validation entry: it compiles an ordered slice of [Source](/docs/reference/go-api/kernel/#source) values in the schema's own context, unifies them in stack order, runs the closed-schema disallowed-field walk, and asserts concreteness on the merged value via [cue.Concrete](https://pkg.go.dev/cuelang.org/go/cue#Concrete)(true). A single value is a one-element slice.

Per-source attribution flows through [token.Pos.Filename](https://pkg.go.dev/cuelang.org/go/cue/token#Pos.Filename): each source is compiled with [cue.Filename](https://pkg.go.dev/cuelang.org/go/cue#Filename)(Origin) here, where it meets the schema, so every position in the returned error names the source's Origin — see [Kernel.LoadSourceFromFile](/docs/reference/go-api/kernel/#kernelloadsourcefromfile) and [Kernel.LoadSourceFromBytes](/docs/reference/go-api/kernel/#kernelloadsourcefrombytes) for the constructors, and [Source](/docs/reference/go-api/kernel/#source) for what a file-backed origin means.

Returns the merged [cue.Value](https://pkg.go.dev/cuelang.org/go/cue#Value) on success and the zero value on failure. The returned error is the raw CUE error tree (a source that fails to compile included); walk it via [cuelang.org/go/cue/errors.Errors](https://pkg.go.dev/cuelang.org/go/cue/errors#Errors) and [cuelang.org/go/cue/errors.Positions](https://pkg.go.dev/cuelang.org/go/cue/errors#Positions), or print it via [cuelang.org/go/cue/errors.Print](https://pkg.go.dev/cuelang.org/go/cue/errors#Print). Presentation is outside the kernel's contract — frontends own their own formatting. Module-name framing is the caller's responsibility — wrap with [fmt.Errorf](https://pkg.go.dev/fmt#Errorf) if a context prefix is required.

Empty sources, a zero schema, or a merged value that does not exist all short-circuit to (zero, nil) — the "no values supplied" path documented across the kernel's validation surface.

### Option

```go
type Option func(*Kernel)
```

Option configures a [Kernel](/docs/reference/go-api/kernel/#kernel) at construction time. Options compose via the functional-options pattern; new options can be added in MINOR instances without breaking existing call sites. The provided options are [WithSchemaLoader](/docs/reference/go-api/kernel/#withschemaloader) and [WithRegistry](/docs/reference/go-api/kernel/#withregistry); the Kernel exposes no injection slot that no kernel operation reads.

#### WithRegistry

```go
func WithRegistry(registry string) Option
```

WithRegistry sets the ONE OCI registry mapping (CUE\_REGISTRY syntax, e.g. "opmodel.dev=ghcr.io/open-platform-model") every kernel operation uses for catalog, module and schema resolution:

- the render build's catalog imports ([Kernel.Render](/docs/reference/go-api/kernel/#kernelrender));
- registry module acquisition ([Kernel.AcquireModuleFromRegistry](/docs/reference/go-api/kernel/#kernelacquiremodulefromregistry));
- directory acquisition ([Kernel.AcquireModuleFromDir](/docs/reference/go-api/kernel/#kernelacquiremodulefromdir), [Kernel.AcquirePlatformFromDir](/docs/reference/go-api/kernel/#kernelacquireplatformfromdir), [Kernel.AcquireInstanceFromDir](/docs/reference/go-api/kernel/#kernelacquireinstancefromdir));
- instance synthesis ([Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance));
- the compilation of file-backed values sources on every path that accepts [Source](/docs/reference/go-api/kernel/#source) values ([Kernel.ValidateConfigDetailed](/docs/reference/go-api/kernel/#kernelvalidateconfigdetailed), [Kernel.AcquireInstanceFromDir](/docs/reference/go-api/kernel/#kernelacquireinstancefromdir) with trailing values, [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance)), so a values file importing a registry module resolves it through this mapping;
- the default schema cache (absent [WithSchemaLoader](/docs/reference/go-api/kernel/#withschemaloader)).

No acquire verb takes a per-call registry override.

Omitting this option (or passing an empty string) inherits CUE\_REGISTRY from the process environment; the kernel applies no built-in default registry — the same stance as the schema loader. The mapping is never written back to the process environment; it is plumbed into the load configuration for the operation only.

#### WithSchemaLoader

```go
func WithSchemaLoader(l schema.Loader) Option
```

WithSchemaLoader configures the [schema.Loader](/docs/reference/go-api/schema/#loader) used to populate the kernel's [\*schema.Cache](/docs/reference/go-api/schema/#cache). Omitting this option defaults to a [schema.OCILoader](/docs/reference/go-api/schema/#ociloader) carrying the kernel's [WithRegistry](/docs/reference/go-api/kernel/#withregistry) mapping, which resolves [schema.DefaultSchemaModule](/docs/reference/go-api/schema/#defaultschemamodule) through it (and through CUE\_REGISTRY / CUE\_CACHE\_DIR from the process environment when no mapping was given). The option wins over that default regardless of the order the options are passed in.

The Kernel wraps the supplied Loader in a fresh Cache; callers cannot inject a pre-built Cache. This guarantees one Kernel = one Cache, so no two Kernels accidentally share memoization. Multi-Kernel cache sharing is intentionally not exposed and may be added later as a non-breaking addition.

A nil Loader is ignored (the default OCILoader applies).

### RenderDiagnostics

```go
type RenderDiagnostics struct {
	// Pairs is the matched pair set in build order.
	Pairs []RenderPair

	// Unmatched lists components no transformer matched, each carrying
	// every candidate the demand walk reached for it.
	Unmatched []oerrors.UnmatchedComponent

	// Unresolved is every demand the platform failed to resolve: an empty
	// bucket (Disqualified empty, Alternatives naming same-base keys the
	// platform does implement) or every candidate disqualified.
	Unresolved []oerrors.UnresolvedDemand

	// Skipped is every demand the render skipped under
	// [RenderInput.SkipUnprovided], in build order (every component's
	// resource rows, then every component's trait rows). Always empty when
	// the switch is off. Advisory: a skipped demand never refuses, and a
	// frontend words it.
	Skipped []SkippedDemand

	// Unify is every candidate the always-unify rung disqualified, one row per
	// (component, transformer) carrying the FQNs it conflicted at. The verbatim
	// CUE cause is not recoverable from inside the build.
	Unify []oerrors.UnifyRefusal

	// UnhandledTraits maps a component to the effectively-optional traits
	// no matched transformer handles. Advisory: a frontend formats it.
	UnhandledTraits map[string][]string

	// FailedPairs names matched pairs whose transformer output errored.
	FailedPairs []RenderPair

	// OverSubscribed is every provider-fulfilled contract key that transformers
	// of two or more enabled registry entries (path plus major) require (the
	// single-provider guard), key-sorted, each row naming the registry keys
	// core's #contracts.providedBy holds for it. The count is core's, the one
	// the platform's Contracts reads, so the rows are exactly its
	// OverSubscribed. Any row refuses the render through the gate.
	OverSubscribed []oerrors.OverSubscribedContract

	// Collisions is every contract key two or more enabled registry
	// entries' catalogs list (core's #contracts.collisions), key-sorted,
	// each row naming the registry keys core's #contracts.collidingEntries
	// holds for it: exactly the platform inventory's CollidingEntries. Any
	// row refuses the render through the gate, first among the causes,
	// whatever the instance and whatever [RenderInput.SkipUnprovided] says.
	// Empty on a platform pinning a core without the report, which cannot
	// evaluate a colliding platform at all.
	Collisions []oerrors.ContractCollision

	// Routable is core's #contracts.routable as decoded: false when the
	// platform carries an over-subscribed or colliding contract. A false
	// verdict with neither row refuses the render with
	// [*oerrors.NotRoutableError].
	Routable bool

	// ResolvedVersions holds the per-path version rows, in path order.
	ResolvedVersions []ResolvedVersion

	// Replacements holds one row per local replacement the render honoured
	// under [RenderInput.LocalReplacements], in path order; nil otherwise.
	// A replaced path keeps its pinned versions on ResolvedVersions; the
	// row says where its bytes were served from. Advisory data a frontend
	// words (an instance replacement the platform's list made inert is not
	// here, so a frontend computes the inert set from its own file).
	Replacements []Replacement
}
```

RenderDiagnostics is everything the build reports as data, decoded into the kernel's structured types. It is populated on success and carried by [\*RenderError](/docs/reference/go-api/kernel/#rendererror) on a refusal, so a caller can always read the full verdict set. Every field is a row the build emitted, in the build's order; the kernel derives, joins and re-sorts nothing.

It also holds the three advisory facts a render can report, as rows rather than as messages: an unhandled optional trait is on UnhandledTraits, a module requiring a newer build than the platform carries is a ResolvedVersions row with Newer set, and a demand skipped under [RenderInput.SkipUnprovided](/docs/reference/go-api/kernel/#renderinput) is a Skipped row. A frontend words all three.

### RenderError

```go
type RenderError struct {
	Diagnostics RenderDiagnostics
	Err         error
}
```

RenderError is a refusal after the build: the fail-closed gate (a contract collision, an unresolved demand, an over-subscribed provider-fulfilled contract, an unmatched component, or a not-routable platform no row explains), a failed pair, or a non-concrete pair output. Diagnostics carries everything the build reported; Err carries the typed causes ([\*oerrors.ContractCollisionsError](/docs/reference/go-api/errors/#contractcollisionserror), [\*oerrors.UnresolvedDemandsError](/docs/reference/go-api/errors/#unresolveddemandserror), [\*oerrors.OverSubscribedContractsError](/docs/reference/go-api/errors/#oversubscribedcontractserror), [\*oerrors.UnmatchedComponentsError](/docs/reference/go-api/errors/#unmatchedcomponentserror), [\*oerrors.NotRoutableError](/docs/reference/go-api/errors/#notroutableerror), [\*oerrors.TransformError](/docs/reference/go-api/errors/#transformerror)), reachable through errors.As; the gate causes are joined in that order.

#### RenderError.Error

```go
func (e *RenderError) Error() string
```

#### RenderError.Unwrap

```go
func (e *RenderError) Unwrap() error
```

### RenderInput

```go
type RenderInput struct {
	// Instance is the validated instance to render. It MUST carry a Source
	// (Kernel.SynthesizeInstance, Kernel.AcquireInstanceFromDir): the render
	// build imports the instance as a package, so an evaluated value alone
	// is never sufficient.
	Instance *module.Instance

	// Platform is the platform to render against, in the shape (registry
	// entries carrying their catalog by import). It MUST carry a Source
	// (Kernel.AcquirePlatformFromDir).
	Platform *platform.Platform

	// RuntimeName identifies the executing runtime; it enters the build as
	// #context.#runtimeName and is stamped on every rendered object.
	RuntimeName string

	// Skew is the response to catalog version skew. Zero is [SkewWarn].
	Skew SkewPolicy

	// LocalReplacements enables an input's own cue.mod/local-module.cue
	// replacements (a developer redirecting a dependency to a directory or
	// another module) for this render. Off, the default, refuses an input
	// whose file carries a replacement rather than silently rendering
	// against the published pin. On, the replacements are promoted into
	// the render module under the precedence dependencies get (the
	// platform's whole, the instance's only for paths the platform does
	// not name) and reported on [RenderDiagnostics.Replacements]. This is
	// the one switch that lets a render read a directory an artifact names;
	// a frontend sets it for a developer's checkout, never for an artifact
	// it did not author.
	LocalReplacements bool

	// SkipUnprovided renders what the platform can when a component demands
	// a provider-fulfilled contract that no enabled catalog provides (zero
	// providers). Off, the default, refuses such a render as always. On, a
	// skipped trait demand leaves its component rendering every pair it
	// matched; a skipped resource demand omits the whole component (a partly
	// satisfied component never renders) without reporting it unmatched.
	// Every skipped demand is a row on [RenderDiagnostics.Skipped]. Every
	// other refusal stands: a catalog-fulfilled unresolved demand, a
	// provider that exists but did not match, an over-subscribed contract,
	// an unmatched component. The decision is made inside the build, so the
	// render module's own gate agrees with the kernel under both values.
	SkipUnprovided bool
}
```

RenderInput is the input of [Kernel.Render](/docs/reference/go-api/kernel/#kernelrender).

### RenderPair

```go
type RenderPair struct {
	Component   string
	Transformer string
}
```

RenderPair names one matched (component, transformer) pair.

### RenderResult

```go
type RenderResult struct {
	// Compiled is the rendered output, one entry per rendered object, in
	// the build's deterministic pair order, each carrying instance,
	// component and transformer provenance.
	Compiled []*Compiled

	// Diagnostics are the matching verdicts and version rows decoded from
	// the build.
	Diagnostics RenderDiagnostics
}
```

RenderResult is the output of a successful [Kernel.Render](/docs/reference/go-api/kernel/#kernelrender).

### Replacement

```go
type Replacement struct {
	Path   string
	Target string
	By     string
}
```

Replacement is one honoured local replacement: the replaced major-qualified module path, its target (an absolute directory, or a module path for a module replacement) and the input whose cue.mod/local-module.cue supplied it, "platform" or "instance".

### ResolvedVersion

```go
type ResolvedVersion struct {
	// Path is the major-qualified module path.
	Path string

	// ModuleVersion is the build the instance module's cue.mod requires.
	ModuleVersion string

	// PlatformVersion is the build the platform module's cue.mod carries;
	// empty when the platform does not list the path (the instance's own
	// entry then resolves).
	PlatformVersion string

	// Newer is true when the instance requires a newer build than the
	// platform carries.
	Newer bool
}
```

ResolvedVersion is one resolved-versions row: for an OPM-namespace path the instance module requires, what it asked for and what the platform carries. Plain data with no severity; Newer marks the skew case the policy decided.

### SkewPolicy

```go
type SkewPolicy int
```

SkewPolicy is the caller's response to catalog version skew: the instance module's cue.mod requiring a NEWER build of an OPM-namespace path than the platform module carries. Exactly two responses exist; the zero value is the default.

#### SkewWarn

```go
const (
	// SkewWarn renders against the platform's build and marks that path's
	// row on [RenderDiagnostics.ResolvedVersions] as Newer. The default;
	// the wording of any advisory is the frontend's.
	SkewWarn SkewPolicy = iota

	// SkewRefuse fails the render before evaluation with an
	// [*oerrors.SkewError] per skewed path.
	SkewRefuse
)
```

### SkippedDemand

```go
type SkippedDemand struct {
	// Component is the demanding component.
	Component string

	// FQN is the demanded contract key.
	FQN string

	// Kind is "resource" or "trait".
	Kind string

	// DefinedBy is the registry key of the enabled catalog listing the key
	// in its contract maps; empty when none does.
	DefinedBy string

	// Alternatives is the same-base contract-key set the platform does
	// implement at another apiVersion, in the build's ladder order.
	Alternatives []string

	// ComponentOmitted is true on every skipped row of a component that
	// rendered nothing because it has a skipped resource demand, trait rows
	// included, so a frontend can say "not rendered" once per component.
	ComponentOmitted bool
}
```

SkippedDemand is one provider-fulfilled demand skipped under [RenderInput.SkipUnprovided](/docs/reference/go-api/kernel/#renderinput) because nothing on the platform provides it. Data, like every diagnostics row.

### Source

```go
type Source struct {
	// Origin is the stable identifier for machine-readable correlation (file
	// path, K8s object reference, composition input key). It is the filename
	// the kernel compiles Data under, so error positions report Origin via
	// [token.Pos.Filename]. An absolute path naming an existing file marks a
	// file-backed source: the kernel loads it through cue/load at that file's
	// directory, so its imports resolve as they do for any CUE file, through
	// the kernel's [WithRegistry] mapping like every other kernel load and
	// never through a mutated process environment.
	Origin string

	// Data is the values payload as CUE source. It is compiled where it is
	// used, never ahead of use; an empty payload is "no values supplied".
	Data []byte
}
```

Source is one values input for every values-taking kernel entry: [Kernel.ValidateConfigDetailed](/docs/reference/go-api/kernel/#kernelvalidateconfigdetailed), the trailing values sources of [Kernel.AcquireInstanceFromDir](/docs/reference/go-api/kernel/#kernelacquireinstancefromdir) and [InstanceInput.Values](/docs/reference/go-api/kernel/#instanceinput) on [Kernel.SynthesizeInstance](/docs/reference/go-api/kernel/#kernelsynthesizeinstance).

A Source pairs a values payload, as CUE source bytes, with its stable origin. It carries no [cue.Value](https://pkg.go.dev/cuelang.org/go/cue#Value): a value is bound to the context that built it, and every kernel operation builds in a context of its own, so the kernel compiles Data with [cue.Filename](https://pkg.go.dev/cuelang.org/go/cue#Filename)(Origin) in the context of the operation that uses it, where the source meets the schema it is checked against. Per-position diagnostics then carry Origin through [token.Pos.Filename](https://pkg.go.dev/cuelang.org/go/cue/token#Pos.Filename) without a Go-typed wrapper around CUE's error attribution. It carries no display label either: presentation is outside the kernel's contract, and Origin is what CUE positions report.

[Kernel.LoadSourceFromFile](/docs/reference/go-api/kernel/#kernelloadsourcefromfile) and [Kernel.LoadSourceFromBytes](/docs/reference/go-api/kernel/#kernelloadsourcefrombytes) construct a Source after checking that the payload parses; a hand-built Source is compiled the same way and reports a syntax error in the operation that uses it.
