---
title: "Platform"
description: "Platform declares the catalogs the operator renders against: the operator generates a platform CUE module from it, and renders ModuleInstances and ModulePackages against that module."
type: reference
weight: 3
---

Platform declares the catalogs the operator renders against: the operator generates a platform CUE module from it, and renders ModuleInstances and ModulePackages against that module.

## At a glance

| Property | Value |
| --- | --- |
| Group | `opmodel.dev` |
| Version | `v1alpha1` |
| Kind | `Platform` |
| Scope | Cluster |
| Resource | `platforms` |
| Short names | `plat` |
| Subresources | `status` |

The resource definition declares these `kubectl get` columns. A column with a priority above 0 appears only with `-o wide`.

| Column | Type | Value | Priority |
| --- | --- | --- | --- |
| Type | string | `.spec.type` | 0 |
| Ready | string | `.status.conditions[?(@.type=='Ready')].status` | 0 |
| Reason | string | `.status.conditions[?(@.type=='Ready')].reason` | 0 |
| Operator | string | `.status.operatorVersion` | 0 |

## Spec

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `spec.registry` | `map[string]object` | No | Registry is the set of catalog subscriptions keyed by major-suffixed catalog CUE module path (e.g. "opmodel.dev/catalogs/opm@v2"), projecting core \#Platform.\#registry. The key's major must agree with the major of the subscribed version ([0010:D14](/enhancements/0010/decisions/)). |
| `spec.registry.<key>.enable` | `boolean` | No | Enable toggles the subscription. A pointer so that an omitted value defers to the schema default (true) rather than serializing as an explicit false. A disabled subscription is still pinned and imported by the generated module, with enable set to false on its entry. |
| `spec.registry.<key>.version` | `string` | Yes | Version names exactly one published catalog build as a bare SemVer string (e.g. "2.0.0-alpha.3") — the platform module IS the resolution ([0010:D14](/enhancements/0010/decisions/)); there is no range or allow/deny vocabulary. The version's major must agree with the subscription key's `@vN` suffix. The operator uses it twice: as the generated cue.mod pin and as the entry's stamped expected version, which unifies with the imported catalog's own version so wrong bytes fail the build naming the entry ([0019:D13](/enhancements/0019/decisions/)). CRD-required is safe against the stored pre-reshape singleton: API server validation ratcheting keeps status-subresource patches working against a stored object lacking the field (measured in test/integration/crdvalidation). |
| `spec.skewPolicy` | `string` | No | SkewPolicy is the operator's response to catalog version skew: a module whose cue.mod requires a newer build of an OPM-namespace path (core or a catalog) than the platform pins ([0019:D7/D18](/enhancements/0019/decisions/)). "Warn" (the default when unset) renders against the platform's build and reports the skew as a RenderWarning event on the workload; "Refuse" refuses the render before evaluation and the workload reports Ready=False with reason SkewRefused, naming the path and both versions. The policy is not part of the generated platform module; it is recorded beside it, so changing the field alone bumps the generation, regenerates and re-enqueues the workloads. |
| `spec.type` | `string` | Yes | Type is the informational discriminator for the platform (core \#Platform.type). It is required and shown in the Type column of kubectl get. The operator passes it on to the generated platform module, and nothing in OPM acts on its value. |

## Status

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `status.conditions` | `[]Condition` | No | conditions represent the current state of the Platform resource. The PlatformReconciler summarizes module generation on the Ready condition: Ready=True (reason Generated) when the generated platform module built and its contract inventory was accepted, Ready=False with reason BuildFailed (a pinned build does not exist, a key disagrees with its imported catalog, the module did not build, or its contract inventory could not be read; the message names the dependency, the entry or the field), GenerateFailed (the module could not be written to the operator's disk), ContractCollisions (the module built, but more than one enabled registry entry defines a contract key, as two majors of one catalog sharing keys do; the message names each key and the defining entries), OverSubscribedContracts (the module built, but a provider-fulfilled contract is provided by more than one enabled registry entry, counted per entry so two majors of one catalog are two, and whether or not its defining catalog is enabled; the platform cannot route it, and the message names each contract, its defining catalog when one is enabled and the providing registry entries) or ComparablePredicates (the module built, but two enabled transformers have comparable match predicates over a shared catalog-fulfilled contract, so both would render every component the narrower matches; the message names each pair and the contracts it shares). When several refusals hold, the reason is the first of ContractCollisions, OverSubscribedContracts and ComparablePredicates, and the message carries every finding. A refused module is not recorded: the last good package keeps serving renders. The ContractsFulfilled condition is a separate, non-gating report on the package renders are consuming, written whenever one is recorded: False with reason UnfulfilledContracts when the package defines provider-fulfilled contracts nothing on the platform implements (the message names each and its defining catalog), True with reason ContractsFulfilled when every defined contract has a provider, and True with reason NoContractsDefined when the enabled catalogs list no contract at all, so nothing was verified. It never moves Ready. |
| `status.observedGeneration` | `integer` | No |  |
| `status.operatorVersion` | `string` | No | operatorVersion is the version of the operator that last patched this Platform's status, stamped on every reconcile regardless of outcome ([0006:D24](/enhancements/0006/decisions/)). The CLI reads it as the version-skew ceiling; absence means no operator has reconciled the Platform (solo cluster). |
| `status.packageIdentity` | `string` | No | packageIdentity identifies the generated platform package by the two inputs it is a function of ([0015:D13](/enhancements/0015/decisions/)): this CR's generation and the sorted set of active claims' catalog coordinates. The same inputs always produce the same value and any change to either produces a different one. This, not status.active on a TransformerRegistration, is the authoritative answer to what a render is building against. The claim reconciler writes the verdict on the claim and this reconciler folds the verdict into the package, so the two are eventually consistent: an observer can see a claim active for a moment before a package containing its catalog exists. Generation is level-computed, so the next regeneration catches up; until it lands, an active claim is a promise about the next package, not a statement about the current one. |
| `status.registry` | `[]object` | No | registry is the resolved union of the registry the platform is running: the subscriptions spec.registry authored plus the catalogs active TransformerRegistrations contributed, each entry recording which. It makes the effective set enumerable from the Platform rather than only by listing every claim, and it is rewritten on every generation, so it follows the active set in both directions. |
| `status.registry[].catalog` | `string` | Yes | catalog is the major-suffixed CUE module path of the catalog. |
| `status.registry[].enabled` | `boolean` | No | enabled reports whether the catalog's transformers register. A disabled subscription is still pinned and imported by the generated module, so it belongs in the resolved registry, but it contributes no transformer; reading the list without this field would overstate what the platform runs. Every catalog an active claim contributed is enabled. |
| `status.registry[].source` | `string` | Yes | source records which of the two transformer paths put the catalog here, so an operator can tell an authored subscription from one a provider registered. |
| `status.registry[].version` | `string` | Yes | version is the bare SemVer build the generated platform module pins the catalog at. |

## Example

```yaml
apiVersion: opmodel.dev/v1alpha1
kind: Platform
metadata:
  name: cluster
spec:
  registry:
    opmodel.dev/catalogs/opm@v4:
      version: 4.4.4
  type: kubernetes
```

## Notes

It is a cluster-scoped singleton (the only permitted name is "cluster") whose spec projects the core \#Platform author surface.

## Served by

The operator's `platform` controller watches every Platform.

## Enforcement

| Field | Rule | Enforced by |
| --- | --- | --- |
| the object | CEL rule `self.metadata.name == 'cluster'`; refused with: Platform is a cluster singleton; the only permitted name is 'cluster' | API server |
| `spec` | Required | API server |
| `spec.registry.<key>.version` | Required | API server |
| `spec.registry.<key>.version` | Must not be empty | API server |
| `spec.skewPolicy` | One of `Warn`, `Refuse` | API server |
| `spec.type` | Required | API server |
| `spec.type` | Must not be empty | API server |
| `status.registry` | At most one item per `catalog` | API server |
| `status.registry[].catalog` | Required | API server |
| `status.registry[].catalog` | Must not be empty | API server |
| `status.registry[].source` | Required | API server |
| `status.registry[].source` | One of `Subscription`, `Registration` | API server |
| `status.registry[].version` | Required | API server |
| `status.registry[].version` | Must not be empty | API server |
