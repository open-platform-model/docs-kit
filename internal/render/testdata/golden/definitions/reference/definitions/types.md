---
title: "Types"
description: "Constraint types."
type: reference
weight: 2
---

Definitions on this page: [`#NameType`](/docs/reference/definitions/types/#nametype), [`#LabelsType`](/docs/reference/definitions/types/#labelstype), [`#Mode`](/docs/reference/definitions/types/#mode). Every definition is listed on the [definitions page](/docs/reference/definitions/).

## #NameType

An RFC 1123 DNS label.

**At a glance**

- Source: `src/types.cue` in `example.com/defs@v1`
- Shape: string constraint
- Used by: [`#Module`](/docs/reference/definitions/modules/#module)

**Spec**

```cue
#NameType: string & =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" & strings.MinRunes(1) & strings.MaxRunes(63)
```

**Example**

```cue
"web" // valid
```

**Enforcement**

CUE enforces each of these rules on a value unified with `#NameType`:

- The string must match this regular expression:

  ```text
  ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
  ```

- The string must be 1 to 63 runes long.

## #LabelsType

`#LabelsType` is a map of label keys to values.

**At a glance**

- Source: `src/types.cue` in `example.com/defs@v1`
- Shape: map
- Used by: [`#Module`](/docs/reference/definitions/modules/#module)

**Spec**

```cue
#LabelsType: [string]: string
```

## #Mode

`#Mode` is either fast or safe.

**At a glance**

- Source: `src/types.cue` in `example.com/defs@v1`
- Shape: disjunction

**Spec**

```cue
#Mode: "fast" | "safe"
```
