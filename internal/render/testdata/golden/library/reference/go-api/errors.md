---
title: "opm/errors"
description: "Package errors provides the structured verdict rows and error types of OPM."
type: reference
---

```go
import "github.com/open-platform-model/library/opm/errors"
```

Package errors provides the structured verdict rows and error types of OPM.

The render path splits the two. A ROW is plain data with no Error method: [UnresolvedDemand](/docs/reference/go-api/errors/#unresolveddemand), [UnifyRefusal](/docs/reference/go-api/errors/#unifyrefusal), [UnmatchedComponent](/docs/reference/go-api/errors/#unmatchedcomponent), [CandidateVerdict](/docs/reference/go-api/errors/#candidateverdict), [OverSubscribedContract](/docs/reference/go-api/errors/#oversubscribedcontract) and [ContractCollision](/docs/reference/go-api/errors/#contractcollision) are the verdicts the render build decided, decoded as they were emitted and carried on the kernel's render diagnostics. A CAUSE is a pointer-receiver aggregate over those rows that the fail-closed gate raises: [\*ContractCollisionsError](/docs/reference/go-api/errors/#contractcollisionserror), [\*UnresolvedDemandsError](/docs/reference/go-api/errors/#unresolveddemandserror), [\*UnmatchedComponentsError](/docs/reference/go-api/errors/#unmatchedcomponentserror) and [\*OverSubscribedContractsError](/docs/reference/go-api/errors/#oversubscribedcontractserror). Each carries its rows unchanged and in the same order, and wraps nothing, so errors.As on one never yields a cause of another kind. [\*NotRoutableError](/docs/reference/go-api/errors/#notroutableerror) is the gate's rowless catch-all, raised from the decoded routable verdict alone. [\*TransformError](/docs/reference/go-api/errors/#transformerror) and [\*SkewError](/docs/reference/go-api/errors/#skewerror) are ordinary wrappers with a real cause underneath.

Configuration validation errors are CUE-native — see [cuelang.org/go/cue/errors](https://pkg.go.dev/cuelang.org/go/cue/errors) for the canonical interface and helpers (Errors, Positions, Print). The library does not wrap CUE diagnostics in custom Go-typed projections, nor does it ship a presentation-layer formatter; frontends walk the CUE error tree and render however their consumer requires.

## Variables

### ErrInvalidPackage

```go
var (
	// ErrInvalidPackage marks a structurally invalid package: the built
	// root value is not a struct, or the package load resolved other than
	// exactly one instance.
	ErrInvalidPackage = errors.New("invalid OPM package")

	// ErrWrongKind marks a package whose concrete kind does not match the
	// artifact the acquire verb was asked for.
	ErrWrongKind = errors.New("wrong OPM artifact kind")

	// ErrMissingRequiredField marks a package missing a required identity
	// field, or carrying it in non-concrete form. Concreteness is judged
	// before default finalization, so an identity field authored as a
	// defaulted disjunction is missing for this purpose.
	ErrMissingRequiredField = errors.New("missing required field")

	// ErrMissingModule marks an instance synthesis whose input carries no
	// Module, or one with no decoded modulePath/version identity to import
	// the module by.
	ErrMissingModule = errors.New("instance synthesis: Module is required")

	// ErrMissingName marks an instance synthesis whose input carries no
	// instance name.
	ErrMissingName = errors.New("instance synthesis: Name is required")

	// ErrMissingNamespace marks an instance synthesis whose input carries no
	// target namespace.
	ErrMissingNamespace = errors.New("instance synthesis: Namespace is required")

	// ErrMissingSource marks an instance synthesis whose Module carries no
	// staged source tree (HasSource reports false). Synthesis constructs the
	// instance INSIDE the module's own tree so the module's already-tidied
	// cue.mod/module.cue drives transitive resolution; it never fetches or
	// walks a tree of its own. Acquire a source-carrying module via
	// Kernel.AcquireModuleFromRegistry or Kernel.AcquireModuleFromDir.
	ErrMissingSource = errors.New("instance synthesis: Module has no staged source; acquire it via Kernel.AcquireModuleFromRegistry or Kernel.AcquireModuleFromDir")

	// ErrSchemaUnavailable marks a schema resolution that surfaces no core
	// release to derive the synthesized package's core import major from (a
	// bare-major loader whose load reports no version). A pinned loader never
	// produces it: the release is read off the pin without a load.
	ErrSchemaUnavailable = errors.New("instance synthesis: schema unavailable")
)
```

Sentinel errors every kernel acquisition and synthesis failure wraps via %w, so a frontend branches on the failure class with [errors.Is](https://pkg.go.dev/errors#Is) rather than on message text.

They live here, beside the typed render causes, because the packages that raise them (the kernel's internal loader and synthesizer) are internal: a sentinel declared there is unreachable for a consumer, and re-exporting it from opm/kernel would give one value two names. No other package under opm/ declares a sentinel of the same meaning.

## Types

### CandidateVerdict

```go
type CandidateVerdict struct {
	// Transformer is the candidate's FQN.
	Transformer string

	// Matched is the combined verdict of the unify and predicate rungs.
	Matched bool

	// MissingLabels are the required labels the predicate rung found
	// missing or divergent, in the build's order.
	MissingLabels []string
}
```

CandidateVerdict is one candidate the render build's demand walk reached for a component: whether it matched and, when the label predicate refused it, which required labels the component's matchLabels lacked or carried with a different value. A candidate the always-unify rung refused appears unmatched with no missing labels; its conflict is a [UnifyRefusal](/docs/reference/go-api/errors/#unifyrefusal) on the render diagnostics. It is data, not an error.

### ContractCollision

```go
type ContractCollision struct {
	// Key is the colliding contract key (a resource, trait or blueprint
	// FQN).
	Key string `json:"key"`

	// Catalogs are the registry keys (path@major) of the enabled entries
	// whose catalogs list Key, exactly #contracts.collidingEntries[Key].
	// Sorted, so the refusal is deterministic.
	Catalogs []string `json:"catalogs"`
}
```

ContractCollision is one row of the collision report: a contract key that the catalogs of two or more enabled registry entries list in their contract maps (two majors of one catalog sharing a key, say). Core folds only keys with exactly one enabled definer, so a colliding key is absent from the platform's definedBy, requiredBy and comparability report, and the platform is not routable. The render glue reads the rows off core's #Platform.#contracts.collisions and collidingEntries and never computes them, so they are exactly the platform inventory's CollidingEntries.

It is data, not an error; [ContractCollisionsError](/docs/reference/go-api/errors/#contractcollisionserror) is the gate cause.

### ContractCollisionsError

```go
type ContractCollisionsError struct {
	// Contracts is the collision set, key-sorted.
	Contracts []ContractCollision
}
```

ContractCollisionsError aggregates every collision row into the one typed cause the fail-closed gate joins, first among the causes: every other row is read against an inventory the collision distorts. The refusal is platform-wide, whatever the instance and whatever the caller's skip switch says. It carries the diagnostics' rows unchanged and wraps nothing.

#### ContractCollisionsError.Error

```go
func (e *ContractCollisionsError) Error() string
```

### IdentityError

```go
type IdentityError struct {
	// Field names the mismatched identity field: "path" | "version".
	Field string

	// Declared is the value the artifact's metadata claims.
	Declared string

	// Fetched is the coordinate/tag the artifact was actually fetched by.
	Fetched string

	// Coordinate is the full fetched coordinate for context,
	// e.g. "opmodel.dev/catalogs/opm@v4 v4.0.1".
	Coordinate string
}
```

IdentityError reports a mismatch between an artifact's declared identity and the coordinate it was fetched by, including the version clause: the kernel is the version label's verifier, never its source. It is emitted at the one library read site that holds both a fetched coordinate and decoded metadata: module acquire (the kernel's registry acquisition) returns it bare, so frontends route on it via [errors.As](https://pkg.go.dev/errors#As). A platform's catalog builds are verified structurally by core instead (: the registry key binds to the embedded catalog's modulePath), so no catalog read site produces it.

#### IdentityError.Error

```go
func (e IdentityError) Error() string
```

Error names both values (: "a typed error naming both"). Value receiver: the condition is a comparison, not a wrapped failure, so there is no Cause and no Unwrap.

### NotRoutableError

```go
type NotRoutableError struct{}
```

NotRoutableError is the gate's catch-all: the platform's contract inventory reads routable false, and the render decoded neither an over-subscription row nor a collision row to explain it. It fires on no platform any core release produces today; it keeps the kernel's decoded refusal in agreement with the render module's own gate (which reads routable) should a future core add a term to routable that no decoded row reports. It carries no rows and wraps nothing.

#### NotRoutableError.Error

```go
func (e *NotRoutableError) Error() string
```

### OverSubscribedContract

```go
type OverSubscribedContract struct {
	// Key is the provider-fulfilled contract key (a resource or trait FQN).
	Key string

	// Catalogs are the registry keys whose transformers require Key: the
	// catalog module paths (path@major) core binds each entry's embedded
	// catalog identity to, exactly #contracts.providedBy[Key]. Sorted, so
	// the refusal is deterministic.
	Catalogs []string
}
```

OverSubscribedContract is one row of the single-provider guard: a contract key declared `fulfilment: "provider"` on a required demand of transformers from two or more of the platform's enabled registry entries (path plus major: two majors of one catalog are two entries), whether or not an enabled catalog defines the key (as corrected by; enforced inside the render build since library-render-cutover). The count is core's #Platform.#contracts.providedBy, which the render glue reads and never recomputes, so the rows are exactly the keys the platform's contract inventory reports over-subscribed. A platform must carry exactly one provider for such a key; two is a misconfigured platform, not an arbitration. The refusal text is unchanged by where the count comes from.

It is data, not an error; [OverSubscribedContractsError](/docs/reference/go-api/errors/#oversubscribedcontractserror) is the gate cause.

### OverSubscribedContractsError

```go
type OverSubscribedContractsError struct {
	// Contracts is the over-subscription set, key-sorted.
	Contracts []OverSubscribedContract
}
```

OverSubscribedContractsError aggregates every over-subscription row into the one typed cause the fail-closed gate joins. It carries the diagnostics' rows unchanged and wraps nothing.

#### OverSubscribedContractsError.Error

```go
func (e *OverSubscribedContractsError) Error() string
```

### PlatformCoreTooOldError

```go
type PlatformCoreTooOldError struct {
	// Platform is the platform's metadata.name (empty when the value
	// carries none; the message then reads <unnamed>).
	Platform string

	// Field is the missing field: a field under #contracts such as
	// "providedBy", or "#contracts" itself when the value carries no
	// inventory at all.
	Field string

	// Since is the first core release deriving Field, without the "v"
	// prefix, e.g. "2.0.0-alpha.12".
	Since string

	// Require is the oldest core release the kernel accepts today, without
	// the "v" prefix: the release the message tells the caller to re-pin
	// to, so one re-pin clears every missing field at once. The kernel's
	// callers fill it with the floor they enforce (schema.ProvidedBySince);
	// empty, the message falls back to Since.
	Require string
}
```

PlatformCoreTooOldError reports a platform module pinning a core release older than the first one deriving a #Platform.#contracts field the kernel reads. Returned (wrapped) by Kernel.Render before staging and by Platform.Contracts. The fix is re-pinning opmodel.dev/core in the platform module; the kernel never falls back to a count or a verdict of its own.

#### PlatformCoreTooOldError.Error

```go
func (e *PlatformCoreTooOldError) Error() string
```

### SkewError

```go
type SkewError struct {
	// Path is the major-qualified module path in skew.
	Path string

	// ModuleVersion is the build the instance module requires.
	ModuleVersion string

	// PlatformVersion is the build the platform module carries.
	PlatformVersion string
}
```

SkewError is the refuse-mode diagnostic for catalog version skew: the instance module's cue.mod requires a NEWER build of an OPM-namespace path than the platform module carries, and the caller configured the render to refuse rather than warn. The render stops before evaluation; the platform's build is what would have executed.

#### SkewError.Error

```go
func (e *SkewError) Error() string
```

### TransformError

```go
type TransformError struct {
	Component   string
	Transformer string
	Cause       error
}
```

TransformError indicates transformer execution failed for one matched (component, transformer) pair. Unlike the verdict rows, it wraps a real cause: the CUE error the pair's output carried, or the kernel's own concreteness refusal.

#### TransformError.Error

```go
func (e *TransformError) Error() string
```

#### TransformError.Unwrap

```go
func (e *TransformError) Unwrap() error
```

### UnifyRefusal

```go
type UnifyRefusal struct {
	// Component is the component whose bodies diverged.
	Component string

	// Transformer is the FQN of the candidate that was disqualified.
	Transformer string

	// Conflicts are the primitive FQNs at which the bodies conflicted, in
	// the build's order.
	Conflicts []string
}
```

UnifyRefusal is one row of the always-unify rung: a candidate transformer whose required primitive bodies conflict with the component's own bodies. It is data, not an error — the rung's verdicts are carried on the render diagnostics and aggregated into a gate cause, never raised on their own.

The conflict is reported as the FQNs it occurred at, not as a CUE error tree: the glue decides the verdict inside the build and a CUE error is not exportable from there.

### UnmatchedComponent

```go
type UnmatchedComponent struct {
	// Component is the component name.
	Component string

	// Candidates are the verdicts on every transformer the build evaluated
	// for the component, in transformer order. Empty when no transformer
	// was a candidate at all (the component's demands are then on the
	// unresolved rows).
	Candidates []CandidateVerdict
}
```

UnmatchedComponent is one component no transformer matched, with every candidate the demand walk reached for it. It is data, not an error; [UnmatchedComponentsError](/docs/reference/go-api/errors/#unmatchedcomponentserror) is the gate cause.

### UnmatchedComponentsError

```go
type UnmatchedComponentsError struct {
	// Components is the unmatched set, in build order.
	Components []UnmatchedComponent
}
```

UnmatchedComponentsError is the render gate's refusal for components no transformer matched. It carries the diagnostics' rows unchanged, so a frontend can list which candidates were evaluated and why each was refused, and wraps nothing: a row is data, so there is no cause of another kind underneath. Reachable via errors.As from \*kernel.RenderError.

#### UnmatchedComponentsError.Error

```go
func (e *UnmatchedComponentsError) Error() string
```

### UnresolvedDemand

```go
type UnresolvedDemand struct {
	// Component is the component whose demand went unresolved.
	Component string

	// FQN is the demanded contract key.
	FQN string

	// Kind is "resource" or "trait".
	Kind string

	// Alternatives is the same-base contract-key set the platform does
	// implement, in contract-key order (kube-aware apiVersion ladder, applied
	// inside the build). Empty when nothing implements the contract.
	Alternatives []string

	// Disqualified carries, when candidates existed, the unify refusals that
	// disqualified them. Predicate-disqualified candidates contribute no
	// entry (there is no unify conflict to carry).
	Disqualified []UnifyRefusal

	// DefinedBy is the registry key (module path) of the enabled catalog whose
	// contract maps list the demanded key, read inside the build from the
	// platform's derived contract inventory (#contracts.definedBy) and never
	// parsed off the FQN. Empty when no enabled catalog lists it, or when more
	// than one does (Colliding). Diagnostic only: its presence or absence never
	// changes whether the demand refuses.
	DefinedBy string

	// Colliding carries, when the demanded key is a contract collision
	// (more than one enabled registry entry's catalog lists it), the
	// sorted registry keys (path@major) of those entries, read inside the
	// build from #contracts.collidingEntries. Empty otherwise, and on a
	// platform pinning a core without the collision report. Diagnostic
	// only: its presence or absence never changes whether the demand
	// refuses.
	Colliding []string

	// Unprovided is true when the demanded contract is provider-fulfilled
	// and no enabled catalog provides it (no enabled registry entry carries
	// a transformer requiring the key): exactly the demands
	// kernel.RenderInput.SkipUnprovided would skip. Computed inside the
	// build on every render, whatever the switch says.
	Unprovided bool
}
```

UnresolvedDemand is the structured diagnostic for a demanded contract key the platform does not resolve: the matcher index holds no candidate for it, or every candidate was disqualified (by unification or by predicate). Every declared resource is a required demand; a trait demand is unresolved only when its effective `optional` posture is load-bearing (a trait whose posture the catalog never stated is a build error, not a diagnostics row).

The contract-key diagnostic is carried by Alternatives: empty means nothing on this platform implements the contract at any version; non-empty means the contract base is implemented at a different apiVersion only. DefinedBy carries the arm beside it: which enabled catalog lists the demanded key in its contract maps, so the refusal can say "defined by this catalog and implemented by nothing" rather than "unknown".

It is data, not an error; [UnresolvedDemandsError](/docs/reference/go-api/errors/#unresolveddemandserror) is the gate cause.

### UnresolvedDemandsError

```go
type UnresolvedDemandsError struct {
	// Demands is the full unresolved-demand set, in build order.
	Demands []UnresolvedDemand
}
```

UnresolvedDemandsError aggregates every unresolved demand of a render into the typed cause Render fails with through the fail-closed gate. It carries the diagnostics' rows unchanged and wraps nothing: a row is data, so there is no cause of another kind underneath.

#### UnresolvedDemandsError.Error

```go
func (e *UnresolvedDemandsError) Error() string
```
