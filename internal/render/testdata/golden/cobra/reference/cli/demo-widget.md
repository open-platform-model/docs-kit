---
title: "demo widget"
description: "Work with widgets."
type: reference
---

Every command on this page also takes the [global flags](/docs/reference/cli/#global-flags).

## demo widget

Work with widgets.

```text
demo widget [command]
```

Aliases: `w`.

Work with widgets.

Use this group to create and list widgets. Each widget lives in a `demo.cue` file under `./widgets/`.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--context` |  | string |  | Kubernetes context \| cluster. |
| `--namespace` | `-n` | string | `default` | Target namespace. |

**Subcommands**

| Command | Summary |
| --- | --- |
| [demo widget create](/docs/reference/cli/demo-widget/#demo-widget-create) | Create a widget. |
| [demo widget list](/docs/reference/cli/demo-widget/#demo-widget-list) | List widgets. |

## demo widget create

Create a widget.

```text
demo widget create <name> [flags]
```

Create a widget from a template.

The template is read from `--template <dir>`, or the built-in one.

Arguments:

```text
name   The widget name, e.g. web_app
       (kebab-case).
```

Steps:

- render the template
- write the widget

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--cache` |  | string | `~/.cache/demo` | Cache directory. |
| `--context` |  | string |  | Kubernetes context \| cluster. |
| `--label` |  | stringSlice |  | Labels, `key=value`. |
| `--namespace` | `-n` | string | `default` | Target namespace. |
| `--replicas` |  | int | `1` | Replica count. |
| `--rootfs` |  | string | `/home/uu/rootfs` | Not under the home directory. |
| `--template` |  | string |  | Template directory. |
| `--timeout` |  | duration |  | How long to wait. |

**Examples**

```sh
# Create from the built-in template
demo widget create web

demo widget create web --template ./tpl

demo widget create api
demo w create {{</* x */>}}
```

## demo widget list

List widgets.

```text
demo widget list [flags]
```

Aliases: `ls`.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--context` |  | string |  | Kubernetes context \| cluster. |
| `--namespace` | `-n` | string | `default` | Target namespace. |
