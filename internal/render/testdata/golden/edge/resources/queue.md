---
title: "Queue"
description: "A message queue a component declares"
type: reference
---

## At a glance

> [!WARNING]
> **Not implemented**
>
> This catalog defines the resource and ships no transformer that handles it. On a platform where no other catalog handles it, rendering a component that declares it fails.

| Field | Value |
| --- | --- |
| FQN | `example.com/catalogs/demo/resources/queue@v1` |
| API version | `v1`, GA ([contract levels](/catalogs/demo/edge/)) |
| Module path | `example.com/catalogs/demo/resources/v1` |
| Definition | `#QueueResource` in `demo/resources/v1/queue.cue` |
| Catalog | `example.com/catalogs/demo@v1` at `main` (commit `0123456789ab`), unreleased |
| Fulfilment | `catalog`: the declaring catalog implements it |

## Spec

A component writes this resource's fields under `spec.queue`.

```cue
spec: queue: {
	// Messages kept at most.
	depth: int & >0 | *100
}
```

## Notes

Nothing in this catalog renders it.

## Served by

No transformer in `example.com/catalogs/demo@v1` requires this resource or reads it.

## Enforcement

Each rule names what refuses a violation ([What enforces a rule](/docs/concepts/what-enforces-a-rule/)).

| Rule | Enforced by |
| --- | --- |
| A value under `spec.queue` satisfies the schema in Spec, or it does not evaluate. | `cue` |
