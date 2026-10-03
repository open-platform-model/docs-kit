---
title: "Modules"
description: "The module and its components."
type: reference
weight: 1
---

Definitions on this page: [`#Module`](/docs/reference/definitions/modules/#module), [`#Component`](/docs/reference/definitions/modules/#component), [`#Widget`](/docs/reference/definitions/modules/#widget). Every definition is listed on the [definitions page](/docs/reference/definitions/).

## #Module

The unit an author publishes.

**At a glance**

- Source: `src/module.cue` in `example.com/defs@v1`
- Shape: struct, closed, `kind: "Module"`
- Uses: [`#Component`](/docs/reference/definitions/modules/#component), [`#LabelsType`](/docs/reference/definitions/types/#labelstype), [`#NameType`](/docs/reference/definitions/types/#nametype)

**Spec**

```cue
#Module: {
    apiVersion: "example.com/v1"
    kind:       "Module"

    metadata: {
        // The module's name, unique in its registry.
        name!:    #NameType
        version!: string

        labels?: #LabelsType
    }

    // The components, keyed by id.
    #components: #ComponentMap

    debug: bool | *false // a trailing comment

    if debug {
        trace!: string
    }
}
```

**Notes**

It holds components and the values they read.

A module names its components in `#components`; each is a [`#Component`](/docs/reference/definitions/modules/#component).

- one list item names [`#Widget`](/docs/reference/definitions/modules/#widget)
- another item

```text
an indented line
```

**Enforcement**

CUE enforces each of these rules on a value unified with `#Module`:

- A field the definition does not declare is refused: the definition is closed.
- A value is incomplete until each required field (`!`) is set, and CUE names the missing one when it needs a complete value: `metadata.name`, `metadata.version`, `trace` when `debug`.
- `_count < 10` must hold (`_ok`).

## #Component

`#Component` is one deployable part of a [`#Module`](/docs/reference/definitions/modules/#module).

**At a glance**

- Source: `src/module.cue` in `example.com/defs@v1`
- Shape: struct, closed, `kind: "Component"`
- Embeds: [`#Widget`](/docs/reference/definitions/modules/#widget)
- Uses: [`#Widget`](/docs/reference/definitions/modules/#widget)
- Used by: [`#Module`](/docs/reference/definitions/modules/#module)

**Spec**

```cue
#Component: {
    kind: "Component"
    #Widget

    spec: {...}
}
```

**Example**

```cue
"web"
"db"
```

**Enforcement**

CUE enforces each of these rules on a value unified with `#Component`:

- A field the definition does not declare is refused: the definition is closed.

## #Widget

Shared fields every component embeds.

**At a glance**

- Source: `src/module.cue` in `example.com/defs@v1`
- Shape: struct, open
- Used by: [`#Component`](/docs/reference/definitions/modules/#component)

**Spec**

```cue
#Widget: {
    // The widget's size.
    size: int & >=0
    ...
}
```
