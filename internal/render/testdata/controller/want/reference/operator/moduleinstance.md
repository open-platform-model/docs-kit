---
title: "ModuleInstance"
description: "ModuleInstance asks the operator to render one CUE module at a pinned version with the given values and apply the result."
type: reference
weight: 1
---

ModuleInstance asks the operator to render one CUE module at a pinned version with the given values and apply the result.

## At a glance

| Property | Value |
| --- | --- |
| Group | `opmodel.dev` |
| Version | `v1alpha1` |
| Kind | `ModuleInstance` |
| Scope | Namespaced |
| Resource | `moduleinstances` |
| Short names | `mi` |
| Subresources | `status` |

The resource definition declares these `kubectl get` columns. A column with a priority above 0 appears only with `-o wide`.

| Column | Type | Value | Priority |
| --- | --- | --- | --- |
| Ready | string | `.status.conditions[?(@.type=='Ready')].status` | 0 |
| Module | string | `.spec.module.path` | 0 |
| Version | string | `.spec.module.version` | 0 |
| Retry | date | `.status.nextRetryAt` | 1 |

## Spec

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `spec.module` | `object` | Yes | Module identifies the CUE module to evaluate from the OCI registry. |
| `spec.module.path` | `string` | Yes | Path is the CUE module import path. Example: "opmodel.dev/modules/cert\_manager@v0" |
| `spec.module.version` | `string` | Yes | Version is the pinned module version to resolve from the registry. Example: "v0.2.1" |
| `spec.owner` | `string` | No | Owner identifies which actor manages this instance. An absent or "operator" value means operator-managed: the controller reconciles normally. The API server rejects any other value, including an explicit empty string. Only an explicit "cli" makes the operator skip the instance (no render/apply/prune, no finalizer) and record a single ManagedExternally acknowledgement. There is no CRD default; the reconciler carries the operator-managed default semantics. The operator never reconciles the instance that deploys the operator itself, whatever this field says: an instance named opm-operator in opm-operator-system, an instance of the opmodel.dev/modules/opm\_operator module, or one whose inventory records the operator's own CustomResourceDefinitions. With an absent or "operator" value such an instance is refused (Ready=False, Stalled=True, reason SelfManagementRefused) and never gets the cleanup finalizer, so re-running the install can always repair the operator. Keep it "cli". |
| `spec.prune` | `boolean` | No | Prune controls whether the operator deletes what an instance no longer renders. When true, the operator deletes stale resources on reconcile and every applied object when the ModuleInstance is deleted. When false or absent, it leaves them in place: deleting the ModuleInstance leaves its applied objects running. Namespaces and CustomResourceDefinitions are never deleted. |
| `spec.rollout` | `object` | No | RolloutSpec configures apply behavior for a release. |
| `spec.rollout.forceConflicts` | `boolean` | No | ForceConflicts enables SSA force ownership when desired. |
| `spec.rollout.strategy` | `string` | No | Strategy controls how apply operations are performed. |
| `spec.serviceAccountName` | `string` | No |  |
| `spec.suspend` | `boolean` | No |  |
| `spec.values` | `free-form object` | No | Values contains arbitrary release input values. |

## Status

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `status.conditions` | `[]Condition` | No | conditions represent the current state of the ModuleInstance resource. |
| `status.failureCounters` | `object` | No | FailureCounters tracks bounded reconcile failure counts by action. |
| `status.failureCounters.apply` | `integer` | No |  |
| `status.failureCounters.drift` | `integer` | No |  |
| `status.failureCounters.prune` | `integer` | No |  |
| `status.failureCounters.reconcile` | `integer` | No |  |
| `status.history` | `[]object` | No |  |
| `status.history[].action` | `string` | No |  |
| `status.history[].configDigest` | `string` | No |  |
| `status.history[].finishedAt` | `string (date-time)` | No |  |
| `status.history[].inventoryCount` | `integer` | No |  |
| `status.history[].inventoryDigest` | `string` | No |  |
| `status.history[].message` | `string` | No |  |
| `status.history[].phase` | `string` | No |  |
| `status.history[].renderDigest` | `string` | No |  |
| `status.history[].sequence` | `integer` | No |  |
| `status.history[].sourceDigest` | `string` | No |  |
| `status.history[].startedAt` | `string (date-time)` | No |  |
| `status.instanceUUID` | `string` | No | InstanceUUID is the globally-unique identity of the rendered ModuleInstance, read from the `module-instance.opmodel.dev/uuid` label on rendered resources. Populated on the first successful render; consumed by the prune ownership guard (including the deletion path where a fresh render is not available). |
| `status.inventory` | `object` | No | Inventory stores the current set of owned resources. |
| `status.inventory.count` | `integer` | No |  |
| `status.inventory.digest` | `string` | No |  |
| `status.inventory.entries` | `[]object` | No |  |
| `status.inventory.entries[].component` | `string` | No |  |
| `status.inventory.entries[].group` | `string` | No |  |
| `status.inventory.entries[].kind` | `string` | Yes |  |
| `status.inventory.entries[].name` | `string` | Yes |  |
| `status.inventory.entries[].namespace` | `string` | No |  |
| `status.inventory.entries[].v` | `string` | No |  |
| `status.inventory.revision` | `integer` | No |  |
| `status.lastAppliedAt` | `string (date-time)` | No |  |
| `status.lastAppliedConfigDigest` | `string` | No |  |
| `status.lastAppliedRenderDigest` | `string` | No |  |
| `status.lastAppliedSourceDigest` | `string` | No |  |
| `status.lastAttemptedAction` | `string` | No |  |
| `status.lastAttemptedAt` | `string (date-time)` | No |  |
| `status.lastAttemptedConfigDigest` | `string` | No |  |
| `status.lastAttemptedDuration` | `string` | No |  |
| `status.lastAttemptedRenderDigest` | `string` | No |  |
| `status.lastAttemptedSourceDigest` | `string` | No |  |
| `status.nextRetryAt` | `string (date-time)` | No | NextRetryAt indicates when the controller will next attempt reconciliation after a transient or stalled failure. Nil when the resource is healthy or no-op. |
| `status.observedGeneration` | `integer` | No |  |
| `status.requiredContracts` | `[]string` | No | RequiredContracts lists every contract FQN this instance's components declare, sorted and deduplicated, as the last successful render reported them. It is the instance side of the removal guard ([0015:D3/D16](/enhancements/0015/decisions/)): a TransformerRegistration counts its dependents by intersecting this with its own spec.provides, so a provider cannot be deleted, or shrink its provides, out from under the instances still demanding what it serves. Derived, never authored. The contracts are read off the synthesized instance's components, which is the same keyspace the render's matching compares, so nothing a module author writes into spec can raise or lower the count. A reconcile that does not render — a failed render, a suspended instance, a CLI-owned one — leaves the previous value: a stale entry over-reports demand and blocks a deletion that could have proceeded, which is the safe direction for a guard. |

## Notes

The module is resolved from an OCI registry by spec.module.path and spec.module.version. When spec.owner is "cli", the OPM CLI manages the instance and the operator leaves it alone: it renders, applies and prunes nothing, adds no finalizer, and records only a ManagedExternally acknowledgement.

## Served by

The operator's `moduleinstance` controller watches every ModuleInstance.

## Enforcement

| Field | Rule | Enforced by |
| --- | --- | --- |
| `spec` | Required | API server |
| `spec.module` | Required | API server |
| `spec.module.path` | Required | API server |
| `spec.module.path` | Must not be empty | API server |
| `spec.module.version` | Required | API server |
| `spec.module.version` | Must not be empty | API server |
| `spec.owner` | One of `cli`, `operator` | API server |
| `spec.rollout.strategy` | One of `Apply` | API server |
| `status.inventory.entries[].kind` | Required | API server |
| `status.inventory.entries[].name` | Required | API server |
