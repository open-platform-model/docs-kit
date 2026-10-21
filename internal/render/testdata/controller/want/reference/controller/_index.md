---
title: "Controller Reference"
description: "One generated page per resource kind the controller defines: ModuleInstance, ModulePackage, Platform and TransformerRegistration."
weight: 7
---

The controller defines four resource kinds: ModuleInstance, ModulePackage, Platform and TransformerRegistration. Each has its own page in this section, with its fields, its status, an example where there is one, the reconciler that reconciles it and the rules the API server enforces.

To set the controller up, follow [Install the controller](/docs/start/install-the-controller/). To remove an instance without losing what it applied, follow [Delete an instance safely](/docs/operating/delete-an-instance-safely/). The conditions each kind reports are explained in [Controller conditions](/docs/diagnostics/controller-conditions/).

## Kinds

| Kind | Scope | Summary |
| --- | --- | --- |
| [ModuleInstance](/docs/reference/controller/moduleinstance/) | Namespaced | ModuleInstance asks the controller to render one CUE module at a pinned version with the given values and apply the result. |
| [ModulePackage](/docs/reference/controller/modulepackage/) | Namespaced | ModulePackage renders the ModuleInstance package that a Flux source artifact carries. |
| [Platform](/docs/reference/controller/platform/) | Cluster | Platform declares the catalogs the controller renders against: the controller generates a platform CUE module from it, and renders ModuleInstances and ModulePackages against that module. |
| [TransformerRegistration](/docs/reference/controller/transformerregistration/) | Cluster | TransformerRegistration is a provider module's claim that its catalog implements platform contracts. |
