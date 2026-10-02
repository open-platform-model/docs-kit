---
title: "Backup"
description: "Scheduled backup policy for a component's state"
type: reference
---

## At a glance

> [!IMPORTANT]
> **Provided by your platform**
>
> This catalog defines the contract and ships no transformer for it. Your platform needs exactly one catalog that implements it. Without one, rendering a component that attaches it fails; with two, the kernel refuses every render on that platform.

| Field | Value |
| --- | --- |
| FQN | `example.com/catalogs/demo/traits/backup@v1alpha1` |
| API version | `v1alpha1`, alpha ([contract levels](/catalogs/demo/1.2/)) |
| Module path | `example.com/catalogs/demo/traits/v1alpha1` |
| Definition | `#BackupTrait` in `demo/traits/v1alpha1/backup.cue` |
| Component wrapper | `#Backup` |
| Catalog | `example.com/catalogs/demo@v1` version `1.2.3` |
| Category | `storage` |
| Fulfilment | `provider`: a catalog on the platform implements it, never this one |
| Optional posture | load-bearing: `optional` defaults to `false`, so an unhandled trait fails the render; a module may override it where it attaches the trait |
| Applies to (declared) | [Container](/catalogs/demo/1.2/resources/container/) |

## Spec

A component writes this trait's fields under `spec.backup`.

```cue
spec: backup: #BackupSchema

#BackupSchema: {
	schedule!: string
	keep?:     int & >0
}
```

## Notes

A platform's backup adapter reads it.

Exactly one provider serves it.

## Served by

No transformer in `example.com/catalogs/demo@v1` requires this trait or reads it.

## Enforcement

Each rule names what refuses a violation ([What enforces a rule](/docs/concepts/what-enforces-a-rule/)).

| Rule | Enforced by |
| --- | --- |
| A value under `spec.backup` satisfies the schema in Spec, or it does not evaluate. | `cue` |
| A platform carries exactly one catalog whose transformers require this contract; with two, every render on that platform is refused ([0010:D32](/enhancements/0010/decisions/)). | `kernel` |
| When no transformer matched to the component handles this trait, the render is refused, unless the attachment sets `optional: true` ([0010:D28](/enhancements/0010/decisions/)). | `kernel` |
