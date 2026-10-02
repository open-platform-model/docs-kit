---
title: "Backup"
description: "Scheduled backup policy for a component's state"
type: reference
---

## At a glance

| Field | Value |
| --- | --- |
| FQN | `example.com/catalogs/demo/traits/backup@v1beta1` |
| API version | `v1beta1`, beta ([contract levels](/catalogs/demo/edge/)) |
| Module path | `example.com/catalogs/demo/traits/v1beta1` |
| Definition | `#BackupTrait` in `demo/traits/v1beta1/backup.cue` |
| Catalog | `example.com/catalogs/demo@v1` at `main` (commit `0123456789ab`), unreleased |
| Fulfilment | `catalog`: the declaring catalog implements it |
| Optional posture | advisory: `optional` defaults to `true`, so an unhandled trait warns and the render continues; a module may override it where it attaches the trait |
| Applies to (declared) | [Container](/catalogs/demo/edge/resources/container/) |

## Spec

A component writes this trait's fields under `spec.backup`.

```cue
spec: backup: {
	schedule!: string
	// Copies kept, under the keep-more rule.
	keep: int & >0 | *7
}
```

## Notes

The deployment transformer reads it when present.

## Served by

These transformers in `example.com/catalogs/demo@v1` at `main` (commit `0123456789ab`), unreleased require this trait or read it when present.

| Transformer | Demand | What it does |
| --- | --- | --- |
| `deployment` | optional | Renders a stateless workload as a Deployment \\| with \<pods\> |

## Enforcement

Each rule names what refuses a violation ([What enforces a rule](/docs/concepts/what-enforces-a-rule/)).

| Rule | Enforced by |
| --- | --- |
| A value under `spec.backup` satisfies the schema in Spec, or it does not evaluate. | `cue` |
