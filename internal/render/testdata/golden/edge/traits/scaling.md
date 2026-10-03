---
title: "Scaling"
description: "Replica bounds for a component"
type: reference
---

## At a glance

> [!WARNING]
> **Not implemented**
>
> This catalog defines the trait and ships no transformer that handles it. On a platform where no other catalog handles it, a component that attaches it still renders, and the render warns that the trait is not handled and ignores its values.

| Field | Value |
| --- | --- |
| FQN | `example.com/catalogs/demo/traits/scaling@v1beta1` |
| API version | `v1beta1`, beta ([contract levels](/catalogs/demo/edge/)) |
| Module path | `example.com/catalogs/demo/traits/v1beta1` |
| Definition | `#ScalingTrait` in `demo/traits/v1beta1/scaling.cue` |
| Catalog | `example.com/catalogs/demo@v1` at `main` (commit `0123456789ab`), unreleased |
| Fulfilment | `catalog`: the declaring catalog implements it |
| Optional posture | advisory: `optional` defaults to `true`, so an unhandled trait warns and the render continues; a module may override it where it attaches the trait |
| Applies to (declared) | [Container](/catalogs/demo/edge/resources/container/) |

## Spec

A component writes this trait's fields under `spec.scaling`.

```cue
spec: scaling: {
	min: int | *1
	max: int | *3
	// Per-zone overrides, by zone name.
	zones?: [string]: int
}
```

## Notes

Advisory: see [docs/scaling-notes.md](https://github.com/example/demo/blob/0123456789abcdef0123456789abcdef01234567/docs/scaling-notes.md), and a pattern like \{\{\< x \>\}\} stays escaped.

## Served by

No transformer in `example.com/catalogs/demo@v1` requires this trait or reads it.

## Enforcement

Each rule names what refuses a violation ([What enforces a rule](/docs/concepts/what-enforces-a-rule/)).

| Rule | Enforced by |
| --- | --- |
| A value under `spec.scaling` satisfies the schema in Spec, or it does not evaluate. | `cue` |
