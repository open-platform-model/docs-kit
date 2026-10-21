---
title: "opm controller"
description: "Install and manage the opm-controller on a cluster."
type: reference
---

Every command on this page also takes the [global flags](/docs/reference/cli/#global-flags).

## opm controller

Install and manage the opm-controller on a cluster.

```text
opm controller [command]
```

Install, upgrade, and remove the opm-controller on a Kubernetes cluster.

opm controller prepares a cluster for OPM workflows that depend on the controller: its CRDs (ModuleInstance, ModulePackage, Platform, TransformerRegistration) and, for full deployments, the running controller itself.

**Subcommands**

| Command | Summary |
| --- | --- |
| [opm controller install](/docs/reference/cli/opm-controller/#opm-controller-install) | Install the opm-controller on a cluster. |
| [opm controller uninstall](/docs/reference/cli/opm-controller/#opm-controller-uninstall) | Remove the opm-controller from a cluster. |

## opm controller install

Install the opm-controller on a cluster.

```text
opm controller install [flags]
```

Install the opm-controller from its OPM module, `opmodel.dev/modules/opm_controller`, as the CLI-owned ModuleInstance opm-controller in opm-system, wait for it to roll out, and give it a Platform to reconcile.

Install pulls the module from the configured registry (`--registry`, `OPM_REGISTRY` or the config file); an air-gapped cluster installs from a mirror that serves the module and its dependencies. Without `--version` it installs the module version this CLI pins; `--version` takes a module version (0.2.0 pins, v0 floats to the newest release of that major), and install prints the controller release that version deploys. The module renders against its own dependency pins, never the cluster Platform.

Every check that can refuse runs before anything is written: the values against the module's `#config`, a controller newer than this CLI, a rendered image that disagrees with the controller version the module states, a ModuleInstance CRD below this CLI's floor, and objects that exist and are not OPM's. Then the CRDs are applied and served, then the instance is applied and recorded (every rendered object, the CRDs and the Namespace included), and install waits for the controller Deployment's rollout. Objects left terminating by a previous uninstall are waited out first; all waits share the one `--timeout` budget.

The controller's settings are instance values: `-f/--values` files are layered over the values recorded on the controller's instance, so a reinstall keeps every recorded value it does not change; `--reset-values` starts from the module's defaults instead.

A full install then creates the singleton cluster Platform, subscribed to the newest published release of the first-party catalog. The version is resolved from the registry before anything is applied. If the catalog has published no release yet, the command refuses and names `--catalog-prerelease`; an existing Platform is reported and left untouched, never rewritten.

`--crds-only` applies just the CRDs, for clusters where the CLI drives module lifecycle without a running controller. They are the CRDs the pinned module version renders, carried in this CLI, so `--crds-only` needs no registry and talks only to the cluster: it resolves and renders no module, writes no ModuleInstance, and takes no `--version`, `-f` or `--reset-values`. A CRD that exists is judged as on a full install: one the controller's instance record lists is applied (when the record cannot be read because access is denied, the CRDs are judged without it and a warning says so); one without OPM's managed-by label, or with another instance's identity, is refused, and the refusal prints the annotation to set on the CRD so that this instance takes it over; `--crds-only` adopts nothing by itself. It also refuses CRDs a newer module version installed: run `--crds-only` from a CLI that pins that version or newer, or install that version in full.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--catalog-prerelease` |  | bool |  | Subscribe the cluster Platform to the newest catalog prerelease instead of the newest release. |
| `--context` |  | string |  | Kubernetes context to use. |
| `--crds-only` |  | bool |  | Install only the CustomResourceDefinitions of the pinned controller module, carried in this CLI (no registry). |
| `--group` |  | string |  | Bind the opm-cli-user ClusterRole to this group (requires `--rbac`). |
| `--kubeconfig` |  | string |  | Path to kubeconfig file. |
| `--rbac` |  | bool |  | Also create the opm-cli-user ClusterRole. |
| `--reset-values` |  | bool |  | Ignore the controller instance's recorded values and start from the module's defaults. |
| `--skip-platform` |  | bool |  | Do not create the cluster Platform. |
| `--timeout` |  | duration | `5m0s` | How long to wait, in total, for terminating objects to clear, the CRDs to be served and the controller to roll out. |
| `--user` |  | string |  | Bind the opm-cli-user ClusterRole to this user (requires `--rbac`). |
| `--values` | `-f` | stringArray |  | Values file layered over the controller instance's recorded values (repeatable). |
| `--version` |  | string |  | Controller module version to install: X.Y.Z pins, vN floats (default: the version this CLI pins). |

**Examples**

```sh
# Install the pinned controller module and seed the cluster Platform
opm controller install

# Route the controller's own module fetches through a mirror
opm controller install -f controller-values.cue   # values: registry: "opmodel.dev=mirror.example/opm"

# Install another controller module version
opm controller install --version 0.2.0

# Start over from the module's defaults
opm controller install --reset-values

# Install the controller without creating a Platform
opm controller install --skip-platform

# Install only the CRDs this CLI carries (no registry), plus RBAC for a user
opm controller install --crds-only --rbac --user alice
```

## opm controller uninstall

Remove the opm-controller from a cluster.

```text
opm controller uninstall [flags]
```

Delete every object the controller's instance record (ModuleInstance opm-controller in opm-system) lists, then the record, except the CRDs and the controller's Namespace — those remain for a deliberate, separate `kubectl delete crd` once you're sure no ModuleInstance data is still needed. An object that no longer carries the instance's identity, or whose `opmodel.dev/adopt` annotation names another instance, is left behind.

With no record, uninstall deletes nothing: run `opm controller install` first, so the running controller is recorded.

Refuses to proceed while any ModuleInstance still carries the controller's cleanup finalizer: deleting the controller out from under it would orphan its workload without the controller ever running cleanup. `--remove-finalizers` overrides this by stripping just that finalizer (leaving any other finalizers intact) and proceeding — the instance's resources become unmanaged.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--context` |  | string |  | Kubernetes context to use. |
| `--kubeconfig` |  | string |  | Path to kubeconfig file. |
| `--remove-finalizers` |  | bool |  | Strip the controller's cleanup finalizer from active ModuleInstances and proceed. |

**Examples**

```sh
# Remove the controller (refuses if any ModuleInstance is still active)
opm controller uninstall

# Remove the controller, orphaning any still-active ModuleInstances
opm controller uninstall --remove-finalizers
```
