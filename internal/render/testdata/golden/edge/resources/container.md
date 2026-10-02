---
title: "Container"
description: "One container image a component runs"
type: reference
---

## At a glance

| Field | Value |
| --- | --- |
| FQN | `example.com/catalogs/demo/resources/container@v1` |
| API version | `v1`, GA ([contract levels](/catalogs/demo/edge/)) |
| Module path | `example.com/catalogs/demo/resources/v1` |
| Definition | `#ContainerResource` in `demo/resources/v1/container.cue` |
| Catalog | `example.com/catalogs/demo@v1` at `main` (commit `0123456789ab`), unreleased |
| Category | `workload` |
| Fulfilment | `catalog`: the declaring catalog implements it |

## Spec

A component writes this resource's fields under `spec.container`.

```cue
spec: container: #ContainerSchema

// The container's settings.
#ContainerSchema: {
	// The container name, unique in the component.
	name!: string
	// How many replicas run.
	replicas: int | *1
	image!:   string
	// Environment variables, the vendored Kubernetes shape.
	env?: [...k8s.#EnvVar]
}
```

Defined elsewhere:

- `k8s.#EnvVar`: from `example.com/catalogs/demo/schemas/kubernetes/core/v1`, the vendored Kubernetes API types

## Notes

The deployment transformer renders it.

## Served by

These transformers in `example.com/catalogs/demo@v1` at `main` (commit `0123456789ab`), unreleased require this resource or read it when present.

| Transformer | Demand | What it does |
| --- | --- | --- |
| `deployment` | required | Renders a stateless workload as a Deployment \\| with \<pods\> |
| `service` | optional | Exposes a container as a Service |

## Enforcement

Each rule names what refuses a violation ([What enforces a rule](/docs/concepts/what-enforces-a-rule/)).

| Rule | Enforced by |
| --- | --- |
| A value under `spec.container` satisfies the schema in Spec, or it does not evaluate. | `cue` |
