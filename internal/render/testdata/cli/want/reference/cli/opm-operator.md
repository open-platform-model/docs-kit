---
title: "opm operator"
description: "Install and manage the opm-operator on a cluster."
type: reference
---

Every command on this page also takes the [global flags](/docs/reference/cli/#global-flags).

## opm operator

Install and manage the opm-operator on a cluster.

```text
opm operator [command]
```

Install, upgrade, and remove the opm-operator on a Kubernetes cluster.

opm operator prepares a cluster for OPM workflows that depend on the operator: its CRDs (ModuleInstance, ModulePackage, Platform, TransformerRegistration) and, for full deployments, the running controller itself.

**Subcommands**

| Command | Summary |
| --- | --- |
| [opm operator install](/docs/reference/cli/opm-operator/#opm-operator-install) | Install the opm-operator on a cluster. |
| [opm operator uninstall](/docs/reference/cli/opm-operator/#opm-operator-uninstall) | Remove the opm-operator from a cluster. |

## opm operator install

Install the opm-operator on a cluster.

```text
opm operator install [flags]
```

Install the opm-operator from its OPM module, `opmodel.dev/modules/opm_operator`, as the CLI-owned ModuleInstance opm-operator in opm-operator-system, wait for it to roll out, and give it a Platform to reconcile.

Install pulls the module from the configured registry (`--registry`, `OPM_REGISTRY` or the config file); an air-gapped cluster installs from a mirror that serves the module and its dependencies. Without `--version` it installs the module version this CLI pins; `--version` takes a module version (0.2.0 pins, v0 floats to the newest release of that major), and install prints the operator release that version deploys. The module renders against its own dependency pins, never the cluster Platform.

Every check that can refuse runs before anything is written: the values against the module's `#config`, an operator newer than this CLI, a rendered image that disagrees with the operator version the module states, a ModuleInstance CRD below this CLI's floor, and objects that exist and are not OPM's. Then the CRDs are applied and served, then the instance is applied and recorded (every rendered object, the CRDs and the Namespace included), and install waits for the controller Deployment's rollout. Objects left terminating by a previous uninstall are waited out first; all waits share the one `--timeout` budget.

An operator installed from an earlier release manifest (by an older CLI or kubectl apply) is migrated: install takes over exactly the objects it proves that manifest created, recreates the controller Deployment once (its selector changes, so patches made to it are lost; pass them as values), and deletes the earlier role bindings the module replaces. An object that fails the proof refuses the install before anything is written, and an interrupted migration completes on the next run.

The operator's settings are instance values: `-f/--values` files are layered over the values recorded on the operator's instance, so a reinstall keeps every recorded value it does not change; `--reset-values` starts from the module's defaults instead.

A full install then creates the singleton cluster Platform, subscribed to the newest published release of the first-party catalog. The version is resolved from the registry before anything is applied. If the catalog has published no release yet, the command refuses and names `--catalog-prerelease`; an existing Platform is reported and left untouched, never rewritten.

`--crds-only` applies just the CRDs of the same render, for clusters where the CLI drives module lifecycle without a running operator.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--catalog-prerelease` |  | bool |  | Subscribe the cluster Platform to the newest catalog prerelease instead of the newest release. |
| `--context` |  | string |  | Kubernetes context to use. |
| `--crds-only` |  | bool |  | Install only the CustomResourceDefinitions of the operator module's render. |
| `--group` |  | string |  | Bind the opm-cli-user ClusterRole to this group (requires `--rbac`). |
| `--kubeconfig` |  | string |  | Path to kubeconfig file. |
| `--rbac` |  | bool |  | Also create the opm-cli-user ClusterRole. |
| `--reset-values` |  | bool |  | Ignore the operator instance's recorded values and start from the module's defaults. |
| `--skip-platform` |  | bool |  | Do not create the cluster Platform. |
| `--timeout` |  | duration | `5m0s` | How long to wait, in total, for terminating objects to clear, the CRDs to be served and the operator to roll out. |
| `--user` |  | string |  | Bind the opm-cli-user ClusterRole to this user (requires `--rbac`). |
| `--values` | `-f` | stringArray |  | Values file layered over the operator instance's recorded values (repeatable). |
| `--version` |  | string |  | Operator module version to install: X.Y.Z pins, vN floats (default: the version this CLI pins). |

**Examples**

```sh
# Install the pinned operator module and seed the cluster Platform
opm operator install

# Route the operator's own module fetches through a mirror
opm operator install -f operator-values.cue   # values: registry: "opmodel.dev=mirror.example/opm"

# Install another operator module version
opm operator install --version 0.2.0

# Start over from the module's defaults
opm operator install --reset-values

# Install the operator without creating a Platform
opm operator install --skip-platform

# Install only the CRDs, plus RBAC for a specific user
opm operator install --crds-only --rbac --user alice
```

## opm operator uninstall

Remove the opm-operator from a cluster.

```text
opm operator uninstall [flags]
```

Delete every object the operator's instance record (ModuleInstance opm-operator in opm-operator-system) lists, then the record, except the CRDs and the operator's Namespace — those remain for a deliberate, separate `kubectl delete crd` once you're sure no ModuleInstance data is still needed. An object that no longer carries the instance's identity is left behind.

With no record, uninstall deletes nothing: run `opm operator install` first, so the running operator is recorded.

Refuses to proceed while any ModuleInstance still carries the operator's cleanup finalizer: deleting the operator out from under it would orphan its workload without the operator ever running cleanup. `--remove-finalizers` overrides this by stripping just that finalizer (leaving any other finalizers intact) and proceeding — the instance's resources become unmanaged.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--context` |  | string |  | Kubernetes context to use. |
| `--kubeconfig` |  | string |  | Path to kubeconfig file. |
| `--remove-finalizers` |  | bool |  | Strip the operator's cleanup finalizer from active ModuleInstances and proceed. |

**Examples**

```sh
# Remove the operator (refuses if any ModuleInstance is still active)
opm operator uninstall

# Remove the operator, orphaning any still-active ModuleInstances
opm operator uninstall --remove-finalizers
```
