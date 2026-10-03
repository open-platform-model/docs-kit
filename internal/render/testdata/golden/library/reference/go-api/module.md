---
title: "opm/module"
description: "Package module defines the Module type, mirroring the #Module definition in the OPM core schema."
type: reference
---

```go
import "github.com/open-platform-model/library/opm/module"
```

Package module defines the Module type, mirroring the #Module definition in the OPM core schema. A Module represents the parsed module definition before it is built into an instance.

Debug overlays. The CUE schema includes a `debugValues` field on every `#Module` for author-supplied example values used by build/validation tooling. `debugValues` is a Module field — NOT a separate kernel artifact — and it is read off Module.Package via schema.DebugValues. Whether a frontend layers debugValues into the values stack is a policy decision that lives in the helper layer; the kernel itself never observes the distinction.

## Types

### Instance

```go
type Instance struct {
	// Metadata is the decoded instance-level metadata cache. May be nil when
	// the metadata could not be decoded.
	Metadata *InstanceMetadata

	// Package is the loaded, concrete CUE value for the instance artifact.
	// Source of truth for every field reachable via opm/schema, including
	// the embedded #module reference at schema.Module.
	Package cue.Value

	// Source is the staged source tree the instance package was built from,
	// so a follow-on build can import the instance as a package. Instances
	// are constructed only by the kernel, which stamps it at exactly two
	// sites: Kernel.SynthesizeInstance (overlay mode, the synthesized package
	// inside the module's staged root) and Kernel.AcquireInstanceFromDir
	// (on-disk mode, the loaded directory; overlay mode when values sources are layered on).
	Source *Source
}
```

Instance is an OPM #ModuleInstance artifact in the unified artifact shape.

Package is the source of truth: it is the concrete, values-filled CUE value for the instance and every kernel-internal read (components subtree, source module, transformer match data) goes through Package.LookupPath with paths from opm/schema.

Metadata is an ergonomic decoded projection of the instance-level metadata stamped at construction. It is a cache, not a parallel source of truth — when Metadata and the corresponding subtree of Package disagree, Package wins.

#### Instance.Components

```go
func (r *Instance) Components() cue.Value
```

Components returns the instance's components value as evaluated, definition fields (#resources, #traits, #blueprints, #names) included. It is a read for frontends and tests: the render build reads the same field in CUE, inside the generated glue, and never through this accessor.

#### Instance.ConfigSchema

```go
func (r *Instance) ConfigSchema() cue.Value
```

ConfigSchema returns the embedded source module's #config schema reachable via schema.Module followed by schema.Config on r.Package.

All failure modes return the zero cue.Value (not an error): a nil receiver, a missing #module reference, or a missing #config definition on the embedded module.

### InstanceMetadata

```go
type InstanceMetadata = schema.InstanceMetadata
```

InstanceMetadata is a re-export of [schema.InstanceMetadata](/docs/reference/go-api/schema/#instancemetadata) so callers can keep working with `module.InstanceMetadata`.

### Module

```go
type Module struct {
	// Metadata is the decoded module-level metadata cache. Authoritative data
	// lives in Package; Metadata exists for hot-path access (logging, name
	// lookups). May be nil when the metadata could not be decoded.
	Metadata *ModuleMetadata `json:"metadata"`

	// Package is the loaded CUE value for the module artifact. Source of
	// truth for every field reachable via opm/schema's path vars.
	Package cue.Value `json:"-"`

	// Source is the module's staged source tree, populated only when the
	// module was acquired through a source-carrying path
	// (Kernel.AcquireModuleFromRegistry, Kernel.AcquireModuleFromDir):
	// always overlay mode, never on-disk. It is nil otherwise. Consumers
	// that must build inside the module's own root — Kernel.SynthesizeInstance
	// — gate on HasSource(). See [Source] for the
	// full two-mode contract shared with Instance and Platform.
	Source *Source `json:"-"`
}
```

Module represents an OPM #Module artifact in the unified artifact shape.

Package is the source of truth: it is the loaded CUE value for the module and every kernel-internal read (the #config schema, components subtree) goes through Package.LookupPath with paths from opm/schema.

Metadata is an ergonomic decoded projection of the module-level metadata stamped at construction. It is a cache, not a parallel source of truth — when Metadata and the corresponding subtree of Package disagree, Package wins.

#### NewModuleFromValue

```go
func NewModuleFromValue(v cue.Value) (*Module, error)
```

NewModuleFromValue builds a \*Module from a raw CUE artifact value: it decodes ModuleMetadata from the value's metadata field and stores the input cue.Value unmodified in Package. Errors return a nil \*Module — partial values are never returned. The returned Module carries no Source.

#### Module.ConfigSchema

```go
func (m *Module) ConfigSchema() cue.Value
```

ConfigSchema returns the module's #config schema reachable via schema.Config on m.Package.

All failure modes return the zero cue.Value (not an error): a nil receiver or a missing #config definition on the module package. Callers detect failure via the returned value's Exists() method.

#### Module.HasSource

```go
func (m *Module) HasSource() bool
```

HasSource reports whether the module carries a staged source tree (non-nil Source with a populated overlay). Consumers that must build inside the module's own root — Kernel.SynthesizeInstance — gate on this and return a deterministic error when it is false, rather than silently fetching.

### ModuleMetadata

```go
type ModuleMetadata = schema.ModuleMetadata
```

ModuleMetadata is the decoded module-level identity record. It is a re-export of [schema.ModuleMetadata](/docs/reference/go-api/schema/#modulemetadata) so callers can keep working with `module.ModuleMetadata` without taking a transitive dependency on opm/schema at every reference site.

### Source

```go
type Source struct {
	// Root is the absolute module root of the tree: the load.Config.ModuleRoot
	// a consumer builds against. In overlay mode it is the synthetic root every
	// Overlay key sits under; in on-disk mode it is a real directory.
	Root string

	// Pkg is the package directory relative to Root that holds the artifact's
	// CUE package. Empty means the root package (".").
	Pkg string

	// Overlay maps absolute paths under Root to their file contents as bytes,
	// cue.mod/module.cue included. Nil selects on-disk mode: the tree is read
	// from Root on the filesystem.
	//
	// Bytes, not cue/load's opaque source interface: every overlay the library
	// builds starts as bytes, and a consumer that materializes the tree — or
	// hands it to cue/load — should not have to recover them by reflection.
	// The kernel wraps them with load.FromBytes at the one place it calls
	// cue/load with an overlay.
	Overlay map[string][]byte
}
```

Source is the staged source tree an artifact was loaded or synthesized from: the module root the tree is keyed under, the package directory inside it, and (for in-memory trees) the overlay carrying the files as bytes.

A Source is in one of two modes:

- Overlay mode (Overlay non-empty): the tree lives in memory, keyed under the deterministic synthetic Root. This is how a module fetched from a registry is staged (Kernel.AcquireModuleFromRegistry) and how a synthesized instance is staged inside its module's tree (Kernel.SynthesizeInstance), and how a module acquired from a directory is staged (Kernel.AcquireModuleFromDir).
- On-disk mode (Overlay nil): the tree lives at Root on the real filesystem. This is how an artifact acquired from a directory is described (Kernel.AcquirePlatformFromDir, Kernel.AcquireInstanceFromDir).

It exists so an artifact can be RE-USED as the input of a follow-on build: a module acquired from the registry becomes the main module of the synth build (so its already-tidied cue.mod/module.cue drives transitive dependency resolution), and an instance or platform carrying its tree can be imported as a package by a later render build. Carrying the staged source on the artifact avoids a second fetch or a second directory load.

Source is carried by Module (registry path only), Instance (synthesis and directory acquire) and Platform (directory acquire; platform.Source is an alias of this type). It is nil for artifacts constructed from a bare value (e.g. a unit-test CompileString).

#### Source.WriteTo

```go
func (s *Source) WriteTo(dir string) ([]string, error)
```

WriteTo materializes an overlay-mode source under dir: every overlay entry is written at its path relative to Root — parent directories created as needed — so a build served from dir sees the tree it would see from Root. It returns the dir-relative paths it wrote, sorted, so a caller never has to iterate the overlay itself.

The whole overlay is validated before anything is written, so a refusal leaves the directory untouched: a nil receiver, an on-disk source (Overlay nil — Root already IS the directory) and any entry whose path is not under Root are refused with a plain error.

This is the library's one overlay writer, for a frontend that needs a fetched module's tree on disk (scaffolding from a published template): it calls this instead of fetching the module a second time and walking it. The kernel itself never writes an overlay: the render stage serves an overlay-mode input to its build from memory.
