---
title: "Web"
description: "A stateless web workload"
type: reference
---

## At a glance

| Field | Value |
| --- | --- |
| FQN | `example.com/catalogs/demo/blueprints/web@v1` |
| API version | `v1`, GA ([contract levels](/catalogs/demo/edge/)) |
| Module path | `example.com/catalogs/demo/blueprints/v1` |
| Definition | `#WebBlueprint` in `demo/blueprints/v1/web.cue` |
| Catalog | `example.com/catalogs/demo@v1` at `main` (commit `0123456789ab`), unreleased |
| Composed resources | [Container](/catalogs/demo/edge/resources/container/) |
| Composed traits | not declared |
| Match label | `"workload-type"!: "stateless"` (required) |

## Spec

A component writes this blueprint's fields under `spec.web`.

```cue
spec: web: #WebSchema

#WebSchema: {
	container: res.#ContainerSchema
	// The port the web server listens on.
	port: int | *8080
}
```

Defined elsewhere:

- `res.#ContainerSchema`: the spec of [Container](/catalogs/demo/edge/resources/container/)

## Notes

One container, matched to the deployment transformer by its label.

## Served by

These transformers in `example.com/catalogs/demo@v1` require only what this blueprint supplies: its match labels answer their required labels, and it composes every resource and trait they require. A listed transformer can still emit nothing for a component that leaves out the optional fields it renders from.

| Transformer | What it does |
| --- | --- |
| `deployment` | Renders a stateless workload as a Deployment \\| with \<pods\> |

## Enforcement

Each rule names what refuses a violation ([What enforces a rule](/docs/concepts/what-enforces-a-rule/)).

| Rule | Enforced by |
| --- | --- |
| A value under `spec.web` satisfies the schema in Spec, or it does not evaluate. | `cue` |
| A component carrying this blueprint answers its required match label `workload-type`. | `cue` |
