---
title: "TransformerRegistration"
description: "TransformerRegistration is a provider module's claim that its catalog implements platform contracts."
type: reference
weight: 4
---

TransformerRegistration is a provider module's claim that its catalog implements platform contracts.

## At a glance

| Property | Value |
| --- | --- |
| Group | `opmodel.dev` |
| Version | `v1alpha1` |
| Kind | `TransformerRegistration` |
| Scope | Cluster |
| Resource | `transformerregistrations` |
| Short names | `treg` |
| Subresources | `status` |

The resource definition declares these `kubectl get` columns. A column with a priority above 0 appears only with `-o wide`.

| Column | Type | Value | Priority |
| --- | --- | --- | --- |
| Catalog | string | `.spec.catalog` | 0 |
| Version | string | `.spec.version` | 0 |
| Accepted | string | `.status.accepted` | 0 |
| Active | string | `.status.active` | 0 |

## Spec

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `spec.catalog` | `string` | Yes | Catalog is the provider catalog's major-suffixed CUE module path (e.g. "opmodel.dev/catalogs/k8up@v1"). It names a catalog, never a module: a catalog is what carries transformers, so a claim naming a module names nothing that could implement a contract. Acceptance refuses a non-catalog artifact structurally (enhancement 0015). |
| `spec.providerRef` | `object` | Yes | ProviderRef identifies the ModuleInstance that rendered this claim. It is stamped from the rendering instance and never authored, so a module cannot claim to be another provider (enhancement 0015). |
| `spec.providerRef.name` | `string` | Yes | Name of the claiming ModuleInstance. |
| `spec.providerRef.namespace` | `string` | Yes | Namespace of the claiming ModuleInstance. |
| `spec.provides` | `[]string` | Yes | Provides lists every provider-fulfilled contract FQN the named catalog claims to implement. Required but MAY be empty: the CRD sets no minimum on its length. Acceptance compares the list with the contracts re-derived from the catalog, for exact equality. An empty list therefore passes for a catalog that implements no provider-fulfilled contract, and is refused with reason ProvidesMismatch, naming both lists, for any other catalog. The CRD is the wrong place to encode a rule the reconciler states better. |
| `spec.version` | `string` | Yes | Version is the released build of the named catalog, a bare SemVer string (e.g. "1.0.0"). It pins the claim to one build, which is what lets acceptance re-derive Provides from the artifact this names. |

## Status

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `status.accepted` | `boolean` | No | accepted reports whether the claim passed acceptance: the catalog resolved, its Provides re-derived equal, and no other instance holds the same provider. |
| `status.active` | `boolean` | No | active reports whether an accepted claim's provider is serving. An accepted claim stays inactive until its provider ModuleInstance (named by spec.providerRef) reports Ready, and once active it stays active. |
| `status.conditions` | `[]Condition` | No | conditions represent the current state of the TransformerRegistration resource. |
| `status.observedGeneration` | `integer` | No | observedGeneration is the .metadata.generation this claim was last reconciled for, whether or not that reconcile reached a verdict: a claim waiting on the platform or on its provider's inventory records the generation it observed rather than reading as un-reconciled. A claim whose generation is ahead of this has been edited since, so whatever conditions report is stale. |

## Notes

It is the cluster-scoped claim a provider module ships among its rendered resources, the second path by which transformers reach a platform (enhancement 0015); the first is a subscription in Platform.spec.registry.

metadata.name is the dot-joined "\<namespace\>.\<name\>" of the claiming instance (enhancement 0015). A namespace cannot contain a dot, so the join is collision-free and two instances of one provider module produce two distinct claims, the second refused at acceptance naming the claimant, rather than contending for one object.

Creating one requires platform-admin RBAC. The controller ships that role unbound (config/rbac/transformerregistration\_admin\_role.yaml) and ships no tenant-facing role granting create, so a module applied under an impersonated tenant ServiceAccount cannot register a transformer unless a cluster administrator deliberately bound it.

## Served by

The `transformerregistration` reconciler watches every TransformerRegistration.

## Enforcement

| Field | Rule | Enforced by |
| --- | --- | --- |
| the object | CEL rule `self.metadata.name.contains('.')`; refused with: name must be the dot-joined \<namespace\>.\<name\> of the claiming instance | API server |
| `spec` | Required | API server |
| `spec.catalog` | Required | API server |
| `spec.catalog` | Must not be empty | API server |
| `spec.providerRef` | Required | API server |
| `spec.providerRef.name` | Required | API server |
| `spec.providerRef.name` | Must not be empty | API server |
| `spec.providerRef.namespace` | Required | API server |
| `spec.providerRef.namespace` | Must not be empty | API server |
| `spec.provides` | Required | API server |
| `spec.version` | Required | API server |
| `spec.version` | Must not be empty | API server |
