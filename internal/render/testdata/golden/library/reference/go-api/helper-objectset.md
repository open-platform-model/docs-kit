---
title: "opm/helper/objectset"
description: "Package objectset finds rendered objects that share one Kubernetes apply identity, so a runtime can refuse the render instead of letting the last write silently overwrite the first."
type: reference
---

```go
import "github.com/open-platform-model/library/opm/helper/objectset"
```

Package objectset finds rendered objects that share one Kubernetes apply identity, so a runtime can refuse the render instead of letting the last write silently overwrite the first.

Two objects with the same apiVersion, kind, namespace and name reach apply as two writes to one object. Nothing in the kernel notices: kernel.Compiled deliberately carries no platform vocabulary, and Render never reads kind or metadata. This package supplies the missing check in the one place that has both the objects and their provenance, without moving Kubernetes vocabulary into the kernel.

[Duplicates](/docs/reference/go-api/helper-objectset/#duplicates) takes the render's compiled objects and returns every identity two or more of them share, each row naming the identity and every producing component and transformer in render order. It reads exactly four fields off each value — apiVersion, kind, metadata.namespace, metadata.name — and validates nothing else. A value with no kind or no metadata.name is not a Kubernetes object: it is skipped, never refused, because a frontend that requires manifests already fails at conversion with a better message. [DuplicateIdentitiesError](/docs/reference/go-api/helper-objectset/#duplicateidentitieserror) turns the rows into the refusal itself, worded once so every runtime says the same thing.

The kernel never calls this package, and a depguard rule in .golangci.yml keeps it that way; a frontend that applies to something other than Kubernetes may skip it entirely. A runtime that does apply to Kubernetes calls it between render and apply, on the \[\]\*kernel.Compiled the kernel returned and before any wrapping that would lose the provenance fields: the CLI in its render workflow, so build refuses what apply would, and the operator before it builds inventory entries.

## Types

### Duplicate

```go
type Duplicate struct {
	Identity  Identity
	Producers []Producer
}
```

Duplicate is one apply identity that two or more rendered objects share, with every producer of it in render order.

#### Duplicates

```go
func Duplicates(compiled []*kernel.Compiled) []Duplicate
```

Duplicates scans a render's compiled objects and returns every apply identity two or more of them share, in the order each identity was first rendered. A value carrying no kind or no metadata.name is not a Kubernetes object and is skipped; nothing else about the objects is validated. A render whose objects all have distinct identities returns no rows.

### DuplicateIdentitiesError

```go
type DuplicateIdentitiesError struct {
	Duplicates []Duplicate
}
```

DuplicateIdentitiesError is the refusal a runtime raises from the rows [Duplicates](/docs/reference/go-api/helper-objectset/#duplicates) returned, before apply. The kernel never returns it: it is raised by the frontend that calls the helper.

#### DuplicateIdentitiesError.Error

```go
func (e *DuplicateIdentitiesError) Error() string
```

Error names each shared identity once, on its own line, with every component and transformer that produced it, so one wording serves every runtime that applies to Kubernetes.

### Identity

```go
type Identity struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}
```

Identity is a rendered object's apply identity: the four fields a Kubernetes apply addresses an object by. Namespace is empty for a cluster-scoped object, or for one that names no namespace.

#### Identity.String

```go
func (i Identity) String() string
```

String renders the identity as "apps/v1 Deployment web-system/web", or "apps/v1 Deployment web" when there is no namespace.

### Producer

```go
type Producer struct {
	Component   string
	Transformer string
}
```

Producer is the (component, transformer) pair the kernel recorded on a rendered object.

#### Producer.String

```go
func (p Producer) String() string
```

String renders the producer as: component "web" (…/deployment@1.2.0).
