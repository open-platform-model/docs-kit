---
title: "opm/platform"
description: "Package platform defines the Platform and PlatformMetadata types, mirroring the #Platform definition of the OPM core schema."
type: reference
---

```go
import "github.com/open-platform-model/library/opm/platform"
```

Package platform defines the Platform and PlatformMetadata types, mirroring the #Platform definition of the OPM core schema. A Platform represents a deployment target's identity, type, and the catalogs it carries, in the unified (Metadata, Package, Source) artifact shape used elsewhere in the kernel.

A platform is a CUE module on disk that imports its catalogs: every #registry entry embeds a catalog by import and core derives the entry's version and the platform's #composedTransformers from it. The kernel acquires it with Kernel.AcquirePlatformFromDir, which stamps Source, and renders against it with Kernel.Render, which imports the platform package into the render build. The composed transformers are read by the render glue, in CUE, inside the build; the one derived view Go reads by path is the contract inventory (#Platform.#contracts, enhancement 0015), on demand through Platform.Contracts and never at construction.

See:

- enhancements/0019 (workspace root) — single-build render
- adr/006-single-build-artifact-construction.md (one CUE build per artifact)
- adr/005-shares-nothing-renders.md (the build's lifetime and concurrency)

## Types

### ComparablePredicates

```go
type ComparablePredicates struct {
	// Broader is the implementation FQN of the transformer whose predicate
	// contains the other's: it matches every component Narrower matches.
	Broader string `json:"broader"`

	// Narrower is the implementation FQN of the contained transformer.
	Narrower string `json:"narrower"`

	// Contracts lists the catalog-fulfilled contract FQNs both predicates
	// require, the bucket the pair is undiscriminated within.
	Contracts []string `json:"contracts"`
}
```

ComparablePredicates is one row of the comparable-predicate report: two enabled transformers whose match predicates are comparable over at least one shared catalog-fulfilled contract. A transformer's predicate is every required demand it declares — resources, traits and label key-value pairs together — and a pair is comparable when one predicate contains the other, so Broader matches every component Narrower matches and the two are never told apart by a component's shape. A report row, never a refusal: the generation step refuses.

### ContractInventory

```go
type ContractInventory struct {
	// DefinedBy maps each contract FQN exactly one enabled catalog lists
	// to the registry key (the catalog's module path) of the catalog
	// listing it: the value every diagnostic prints beside the contract. A
	// colliding key (Collisions) is absent.
	DefinedBy map[string]string `json:"definedBy"`

	// RequiredBy maps each defined contract FQN to the implementation FQNs of
	// every enabled transformer whose requiredResources or requiredTraits name
	// it, in the build's order. Required demands only (: optional consumption
	// is tolerance, not fulfilment); a defined contract nothing requires maps
	// to an empty list. A colliding key is not defined, so it is absent however
	// many transformers require it.
	RequiredBy map[string][]string `json:"requiredBy"`

	// ProvidedBy maps every provider-fulfilled contract FQN some enabled
	// transformer requires (defined by an enabled catalog or not) to the
	// sorted registry keys (path@major) of the enabled entries whose
	// transformers require it. OverSubscribed is exactly its keys with two
	// or more entries; a key a defined provider contract lacks is
	// Unfulfilled. It is the field a caller prints to name the supplying
	// entries, including when DefinedBy and RequiredBy lack the key (the
	// defining catalog is disabled or absent).
	ProvidedBy map[string][]string `json:"providedBy"`

	// Unfulfilled lists the provider-fulfilled resources and traits an enabled
	// catalog defines and no enabled registry entry provides (no ProvidedBy
	// key). A report the operator surfaces as a non-gating condition, never a
	// refusal. A colliding key is not defined, so it is never listed here.
	Unfulfilled []string `json:"unfulfilled"`

	// OverSubscribed lists the provider-fulfilled resources and traits required
	// by transformers of two or more enabled registry entries (path plus
	// major), whether or not an enabled catalog defines them: the ProvidedBy
	// keys with two or more entries, and what the generation step refuses on.
	// ProvidedBy names the entries.
	OverSubscribed []string `json:"overSubscribed"`

	// Comparable lists every pair of enabled transformers whose match
	// predicates are comparable over at least one shared catalog-fulfilled
	// contract, in the build's comprehension order. The accessor does not sort;
	// a caller that needs a stable order sorts. Only defined contracts are
	// compared, so no row shares a colliding key, even when two transformers
	// require one under equal predicates.
	Comparable []ComparablePredicates `json:"comparable"`

	// Fulfilled is true exactly when Unfulfilled is empty. It is blind to
	// Collisions: it can read true while Collisions is non-empty.
	Fulfilled bool `json:"fulfilled"`

	// Routable is true exactly when OverSubscribed and Collisions are both
	// empty.
	Routable bool `json:"routable"`

	// Discriminated is true exactly when Comparable is empty. It is blind
	// to Collisions: it can read true while Collisions is non-empty.
	Discriminated bool `json:"discriminated"`

	// Collisions lists, ascending, every contract FQN that two or more
	// enabled registry entries' catalogs list in their contract maps. Such
	// a key is in none of DefinedBy, RequiredBy, Unfulfilled or
	// Comparable, and Routable is false while any exists. A disabled entry
	// never counts as a definer. Empty on a platform pinning a core older
	// than [schema.CollisionsSince], which cannot evaluate a colliding
	// platform at all.
	Collisions []string `json:"collisions"`

	// CollidingEntries maps each Collisions key to the ascending registry
	// keys (path@major) of the enabled entries whose catalogs list it.
	CollidingEntries map[string][]string `json:"collidingEntries"`
}
```

ContractInventory is the decoded view of #Platform.#contracts, the inventory core derives from the enabled registry entries' contract maps and the required demands of #composedTransformers. Providers are counted per registry entry (path plus major), the one count the render's single-provider guard reads too: two majors of one catalog are two providers, two transformers of one entry are one, and a provider counts whether or not an enabled catalog defines the contract. Every field is a report: an inventory that is not Fulfilled, not Routable or not Discriminated, or that carries Collisions, is still a healthy value, and whether a generation step withholds a platform package on Routable or Discriminated false is that step's decision, outside this type.

A contract key more than one enabled registry entry's catalog lists (two majors of one catalog sharing a key, say) is a collision: core folds only keys with exactly one enabled definer, so a colliding key is in Collisions and CollidingEntries and in none of DefinedBy, RequiredBy, Unfulfilled or Comparable. Fulfilled and Discriminated are computed without it and can therefore read true while Collisions is non-empty; a caller never reads either as safe while Collisions is non-empty. Routable reads false.

`defined` (the listed members themselves) is not part of this view: its values are the catalogs' member schemas, non-concrete by construction, so there is no Go value a caller could use. A caller that wants one reads it off Platform.Package under #contracts.defined; DefinedBy carries the same key set.

### Platform

```go
type Platform struct {
	// Metadata is the decoded platform-level metadata cache. Authoritative
	// data lives in Package; Metadata exists for hot-path access (logging,
	// name lookups). May be nil when the metadata could not be decoded.
	Metadata *PlatformMetadata `json:"metadata"`

	// Package is the loaded CUE value for the platform artifact, the
	// evaluated form of the package Source points at.
	Package cue.Value `json:"-"`

	// Source is the staged source tree the platform package was loaded from,
	// so a follow-on build can import the platform as a package. It is
	// stamped by Kernel.AcquirePlatformFromDir (on-disk mode, the loaded
	// directory) and is nil for platforms constructed from a bare value
	// (NewPlatformFromValue). Source is the render input: Kernel.Render
	// imports the platform package from it, so a platform without Source
	// cannot be rendered against. No other kernel operation reads it.
	Source *Source `json:"-"`
}
```

Platform represents an OPM #Platform artifact in the unified artifact shape: \{ Metadata, Package \}.

Package is the source of truth: it is the loaded CUE value for the platform, and metadata decoding reads it. The derived CUE views (#composedTransformers, #contracts) are NOT decoded into Go fields at construction: the render build imports the platform package and the glue reads #composedTransformers in CUE, and the contract inventory is read off Package on demand through [Platform.Contracts](/docs/reference/go-api/platform/#platformcontracts) (enhancement 0015).

Metadata is an ergonomic decoded projection of the platform-level metadata stamped at construction. It is a cache, not a parallel source of truth — when Metadata and the corresponding subtree of Package disagree, Package wins.

#### NewPlatformFromValue

```go
func NewPlatformFromValue(v cue.Value) (*Platform, error)
```

NewPlatformFromValue builds a \*Platform from a raw CUE artifact value: it decodes PlatformMetadata from the value's metadata field (with the top-level type hoisted in) and stores the input cue.Value unmodified in Package. Errors return a nil \*Platform — partial values are never returned. The returned Platform carries no Source.

#### Platform.Contracts

```go
func (p *Platform) Contracts() (*ContractInventory, error)
```

Contracts decodes the contract inventory off Package on demand (#Platform.#contracts, [schema.Contracts](/docs/reference/go-api/schema/#metadata)). Nothing decodes it at construction and no kernel verb calls it: the value is already built, and it is read only when a caller asks (the operator's readiness loop, a platform check), so a render pays nothing for it.

The eleven data fields are read by path, so the decoded set is exactly this type's field list; `defined` stays on Package (see [ContractInventory](/docs/reference/go-api/platform/#contractinventory)). Reports, never refusals: an over-subscribed, colliding or undiscriminated platform returns Routable or Discriminated false and a nil error.

A report the value does not carry is never defaulted, in either direction: a platform whose #contracts predates a report field is refused with a [\*oerrors.PlatformCoreTooOldError](/docs/reference/go-api/errors/#platformcoretooolderror) naming the missing field, the first core release carrying it and the floor to re-pin to ([schema.ProvidedBySince](/docs/reference/go-api/schema/#providedbysince)), never returned as a partial inventory whose missing verdict would read as a pass. The same refusal covers a platform carrying no #contracts at all (Field "#contracts": a value built against a core release before 2.0.0-alpha.9, or one that is not a #Platform). A missing providedBy is the refusal Kernel.Render returns for the same platform.

The one exception is the collision report: an absent `collisions` or `collidingEntries` decodes as empty. Every core release carrying #contracts before [schema.CollisionsSince](/docs/reference/go-api/schema/#collisionssince) folds definedBy over every enabled entry, and registry keys are distinct strings, so two enabled definers of one key always conflict there: such a core fails to evaluate a colliding platform, and a value that evaluated without the report provably has no collision. A collision field that is present but fails to decode is still an error.

### PlatformMetadata

```go
type PlatformMetadata = schema.PlatformMetadata
```

PlatformMetadata is a re-export of [schema.PlatformMetadata](/docs/reference/go-api/schema/#platformmetadata) so callers can keep working with `platform.PlatformMetadata`.

### Source

```go
type Source = module.Source
```

Source is a re-export of [module.Source](/docs/reference/go-api/module/#source) so callers can keep working with `platform.Source`, mirroring the [PlatformMetadata](/docs/reference/go-api/platform/#platformmetadata) re-export. One type describes the staged source tree of every artifact; see [module.Source](/docs/reference/go-api/module/#source) for the overlay and on-disk modes.
