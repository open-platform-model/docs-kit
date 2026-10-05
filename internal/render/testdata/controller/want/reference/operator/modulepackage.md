---
title: "ModulePackage"
description: "ModulePackage renders the ModuleInstance package that a Flux source artifact carries."
type: reference
weight: 2
---

ModulePackage renders the ModuleInstance package that a Flux source artifact carries.

## At a glance

| Property | Value |
| --- | --- |
| Group | `opmodel.dev` |
| Version | `v1alpha1` |
| Kind | `ModulePackage` |
| Scope | Namespaced |
| Resource | `modulepackages` |
| Short names | `mpkg` |
| Subresources | `status` |

The resource definition declares these `kubectl get` columns. A column with a priority above 0 appears only with `-o wide`.

| Column | Type | Value | Priority |
| --- | --- | --- | --- |
| Ready | string | `.status.conditions[?(@.type=='Ready')].status` | 0 |
| Source | string | `.spec.sourceRef.name` | 0 |
| Path | string | `.spec.path` | 0 |
| Revision | string | `.status.source.artifactRevision` | 1 |
| Retry | date | `.status.nextRetryAt` | 1 |

## Spec

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `spec.dependsOn` | `[]object` | No | DependsOn references other ModulePackage CRs that must be Ready=True before this ModulePackage is reconciled. References are same-namespace only: an entry whose namespace is set to anything other than this ModulePackage's own is refused: the ModulePackage reports Ready=False with reason DependenciesNotReady and does not reconcile. |
| `spec.dependsOn[].name` | `string` | Yes | Name of the referent. |
| `spec.dependsOn[].namespace` | `string` | No | Namespace of the referent, when not specified it acts as LocalObjectReference. |
| `spec.interval` | `string` | No | Interval at which the reconciler runs again: it re-fetches the source artifact, re-renders the package and compares the source, render and inventory digests with the last applied ones, and skips the apply when they all match. It does not compare the live objects, so a change made directly to an applied object is not detected or reverted until the rendered output changes. Also the requeue interval after transient failures. Defaults to 5 minutes. |
| `spec.path` | `string` | Yes | Path is the directory within the artifact containing instance.cue. Example: "releases/prod/minecraft". |
| `spec.prune` | `boolean` | No | Prune enables deletion of stale resources on reconcile and of all owned resources on ModulePackage deletion. |
| `spec.rollout` | `object` | No | Rollout configures apply behavior. |
| `spec.rollout.forceConflicts` | `boolean` | No | ForceConflicts enables SSA force ownership when desired. |
| `spec.rollout.strategy` | `string` | No | Strategy controls how apply operations are performed. |
| `spec.serviceAccountName` | `string` | No | ServiceAccountName is the name of the ServiceAccount used to impersonate during apply and prune. Empty means use the controller's identity. |
| `spec.sourceRef` | `object` | Yes | SourceRef references a Flux source (OCIRepository, GitRepository, or Bucket) that provides the artifact containing the CUE package. |
| `spec.sourceRef.apiVersion` | `string` | No | API version of the referent, if not specified the Kubernetes preferred version will be used. |
| `spec.sourceRef.kind` | `string` | Yes | Kind of the referent. |
| `spec.sourceRef.name` | `string` | Yes | Name of the referent. |
| `spec.sourceRef.namespace` | `string` | No | Namespace of the referent, when not specified it acts as LocalObjectReference. |
| `spec.suspend` | `boolean` | No | Suspend halts reconciliation when true. |

## Status

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `status.conditions` | `[]Condition` | No | conditions represent the current state of the ModulePackage resource. |
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
| `status.nextRetryAt` | `string (date-time)` | No | NextRetryAt indicates when the controller will next attempt reconciliation after a transient or stalled failure. |
| `status.observedGeneration` | `integer` | No |  |
| `status.source` | `object` | No | Source is the resolved Flux source artifact metadata. |
| `status.source.artifactDigest` | `string` | No | ArtifactDigest is the digest reported by the source artifact. |
| `status.source.artifactRevision` | `string` | No | ArtifactRevision is the revision reported by the source artifact. |
| `status.source.artifactURL` | `string` | No | ArtifactURL is the fetch URL reported by the source artifact. |
| `status.source.ref` | `object` | No | Ref is the resolved source reference. |
| `status.source.ref.apiVersion` | `string` | No | API version of the referent, if not specified the Kubernetes preferred version will be used. |
| `status.source.ref.kind` | `string` | Yes | Kind of the referent. |
| `status.source.ref.name` | `string` | Yes | Name of the referent. |
| `status.source.ref.namespace` | `string` | No | Namespace of the referent, when not specified it acts as LocalObjectReference. |

## Example

```yaml
apiVersion: opmodel.dev/v1alpha1
kind: ModulePackage
metadata:
  name: modulepackage-sample
  namespace: default
spec:
  interval: 5m
  path: releases/local/hello
  prune: true
  serviceAccountName: hello-deploy
  sourceRef:
    apiVersion: source.toolkit.fluxcd.io/v1
    kind: GitRepository
    name: releases
```

## Notes

The operator fetches the artifact spec.sourceRef names, loads instance.cue from the directory at spec.path, and renders and applies it when it evaluates to \#ModuleInstance; any other kind is rejected as unsupported.

## Served by

The operator's `modulepackage` controller watches every ModulePackage.

## Enforcement

| Field | Rule | Enforced by |
| --- | --- | --- |
| `spec` | Required | API server |
| `spec.dependsOn[].name` | Required | API server |
| `spec.path` | Required | API server |
| `spec.path` | Must not be empty | API server |
| `spec.rollout.strategy` | One of `Apply` | API server |
| `spec.sourceRef` | Required | API server |
| `spec.sourceRef.kind` | Required | API server |
| `spec.sourceRef.name` | Required | API server |
| `status.inventory.entries[].kind` | Required | API server |
| `status.inventory.entries[].name` | Required | API server |
| `status.source.ref.kind` | Required | API server |
| `status.source.ref.name` | Required | API server |
