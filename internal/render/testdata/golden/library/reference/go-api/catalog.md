---
title: "opm/catalog"
description: "Package catalog defines the Catalog type, mirroring the #Catalog definition in the OPM core schema: the contracts a catalog defines (#resources, #traits, #blueprints) beside the transformers that implement them (#transformers)."
type: reference
---

```go
import "github.com/open-platform-model/library/opm/catalog"
```

Package catalog defines the Catalog type, mirroring the #Catalog definition in the OPM core schema: the contracts a catalog defines (#resources, #traits, #blueprints) beside the transformers that implement them (#transformers).

A catalog is the kernel's fourth acquired kind, admitted by ADR-009 on the terms recorded there: the kernel reads and derives, and every verdict about what it reads stays with the caller. Nothing here refuses a value for being unwelcome — [Catalog.Provides](/docs/reference/go-api/catalog/#catalogprovides) reports the empty set for a catalog that implements no provider-fulfilled contract, because implementing none is a fact about the catalog and not a malformed value.

The package mirrors opm/module's dependency direction: it imports opm/module for the shared Source type and opm/schema for paths and metadata, and it imports opm/kernel not at all. That direction is structural rather than a convention — opm/kernel imports this package to return what its catalog verbs acquire, so the reverse edge is an import cycle the compiler refuses.

## Types

### Catalog

```go
type Catalog struct {
	// Metadata is the decoded catalog-level identity cache. Authoritative
	// data lives in Package; Metadata exists for hot-path access (logging,
	// provenance in diagnostics). May be nil when the metadata could not be
	// decoded.
	Metadata *CatalogMetadata `json:"metadata"`

	// Package is the loaded CUE value for the catalog artifact. Source of
	// truth for every field reachable via opm/schema's path vars.
	Package cue.Value `json:"-"`

	// Source is the catalog's staged source tree, populated only when the
	// catalog was acquired through a source-carrying path
	// (Kernel.AcquireCatalogFromRegistry, Kernel.AcquireCatalogFromDir):
	// always overlay mode, never on-disk. It is nil otherwise, and
	// [Catalog.Requires] is the one reader that needs it. See
	// [module.Source] for the full two-mode contract shared with every
	// artifact.
	Source *Source `json:"-"`
}
```

Catalog represents an OPM #Catalog artifact in the unified artifact shape.

Package is the source of truth: it is the loaded CUE value for the catalog, and every derived view reads it by path on demand. Nothing but metadata is decoded at construction — a caller that never asks what a catalog provides pays nothing for the fold, exactly as platform.Platform.Contracts is not paid for by a render.

Metadata is an ergonomic decoded projection of the catalog-level metadata stamped at construction. It is a cache, not a parallel source of truth — when Metadata and the corresponding subtree of Package disagree, Package wins.

#### NewCatalogFromValue

```go
func NewCatalogFromValue(v cue.Value) (*Catalog, error)
```

NewCatalogFromValue builds a \*Catalog from a raw CUE artifact value: it decodes CatalogMetadata from the value's metadata field and stores the input cue.Value unmodified in Package. Errors return a nil \*Catalog — partial values are never returned. The returned Catalog carries no Source.

#### Catalog.Provides

```go
func (c *Catalog) Provides() ([]string, error)
```

Provides derives the provider-fulfilled contracts this catalog implements: every contract required by one of the catalog's own #transformers ([schema.Transformers](/docs/reference/go-api/schema/#metadata)) whose value carries `fulfilment: "provider"`. Required demands only — `optionalResources` and `optionalTraits` are tolerance, not fulfilment, the same rule core applies when it folds a platform's inventory.

The fold reads the demand entry's own value rather than looking the contract up in this catalog's #resources / #traits. That is not an optimization: a provider catalog implements contracts ANOTHER catalog defines (that is what makes it a provider), so its own member maps do not list them and a lookup there would derive the empty set for exactly the catalogs this method exists to describe.

Nothing decodes this at construction and no kernel verb calls it: the value is already built, and it is read only when a caller asks, so acquiring a catalog pays nothing for it. That is platform.Platform.Contracts's rule, and so is the next one.

Reports, never refusals. The result is deterministically ordered and deduplicated, so two derivations of one catalog compare equal element for element without the caller sorting — the comparison a consumer makes against a claimed list is then a plain slice equality. A catalog that implements no provider-fulfilled contract, or ships no transformers at all, returns a non-nil empty slice and a nil error: implementing none is a fact about the catalog, not a malformed value.

The one error is a catalog whose demand entries do not evaluate: a `fulfilment` present but not readable as a concrete string means the catalog's contracts were built against a core that means something else by the field, and that is reported rather than silently folded away. An entry carrying no `fulfilment` at all is simply not provider-fulfilled.

#### Catalog.Requires

```go
func (c *Catalog) Requires() (map[string]string, error)
```

Requires returns the dependency requirements the catalog COMMITTED in its own cue.mod/module.cue, keyed by major-qualified module path ("opmodel.dev/core@v2") and valued with the canonical version the file records ("v2.0.0-beta.1"). Those are the two strings and nothing else: a consumer comparing a catalog's committed resolution against a platform's never sees CUE's module types, and version arithmetic within a major is string work.

It reads the committed file, not the build: the point of the answer is what the catalog's author pinned when the artifact was published, which is the side of the comparison a consumer cannot reconstruct from its own resolution. The file is read from the staged Source — from the overlay in overlay mode, from disk in on-disk mode — so a catalog acquired from a registry answers without a second fetch.

Reports, never refusals. Whether a requirement is acceptable, newer, older or incompatible is the caller's judgement; this returns what is written.

Errors: a catalog carrying no Source (one built straight from a value by [NewCatalogFromValue](/docs/reference/go-api/catalog/#newcatalogfromvalue) rather than acquired) has no committed file to read and says so; a module file that is absent or unparseable is reported with its path. A dependency a local replacement serves may carry no version in the file; its entry maps to the empty string rather than being dropped, so the path is still visible to a caller reconciling the two sides.

### CatalogMetadata

```go
type CatalogMetadata = schema.CatalogMetadata
```

CatalogMetadata is a re-export of [schema.CatalogMetadata](/docs/reference/go-api/schema/#catalogmetadata) so callers can keep working with `catalog.CatalogMetadata` without taking a transitive dependency on opm/schema at every reference site.

### Source

```go
type Source = module.Source
```

Source is a re-export of [module.Source](/docs/reference/go-api/module/#source) so callers can keep working with `catalog.Source`, mirroring platform.Source. One type describes the staged source tree of every artifact.
