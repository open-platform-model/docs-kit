---
title: "Operator Reference"
description: "One generated page per operator resource kind: ModuleInstance, ModulePackage, Platform and TransformerRegistration."
weight: 7
---

The operator defines four resource kinds: ModuleInstance, ModulePackage, Platform and TransformerRegistration. Each has its own page in this section, with its fields, its status, an example where there is one, the controller that reconciles it and the rules the API server enforces.

To set the operator up, follow [Install the operator](/docs/start/install-the-operator/). To remove an instance without losing what it applied, follow [Delete an instance safely](/docs/operating/delete-an-instance-safely/). The conditions each kind reports are explained in [Operator conditions](/docs/diagnostics/operator-conditions/).

## Kinds

| Kind | Scope | Summary |
| --- | --- | --- |
| [ModuleInstance](/docs/reference/operator/moduleinstance/) | Namespaced | ModuleInstance asks the operator to render one CUE module at a pinned version with the given values and apply the result. |
| [ModulePackage](/docs/reference/operator/modulepackage/) | Namespaced | ModulePackage renders the ModuleInstance package that a Flux source artifact carries. |
| [Platform](/docs/reference/operator/platform/) | Cluster | Platform declares the catalogs the operator renders against: the operator generates a platform CUE module from it, and renders ModuleInstances and ModulePackages against that module. |
| [TransformerRegistration](/docs/reference/operator/transformerregistration/) | Cluster | TransformerRegistration is a provider module's claim that its catalog implements platform contracts. |
