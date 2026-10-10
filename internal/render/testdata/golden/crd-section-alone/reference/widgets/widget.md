---
title: "Widget"
description: "Widget asks the controller to build one widget."
type: reference
weight: 1
---

Widget asks the controller to build one widget ([0021:D4](/enhancements/0021/decisions/)).

## At a glance

| Property | Value |
| --- | --- |
| Group | `example.dev` |
| Version | `v1` |
| Kind | `Widget` |
| Scope | Namespaced |
| Resource | `widgets` |
| Short names | `wd` |
| Categories | `toys` |
| Subresources | `status` |

The resource definition declares these `kubectl get` columns. A column with a priority above 0 appears only with `-o wide`.

| Column | Type | Value | Priority |
| --- | --- | --- | --- |
| Ready | string | `.status.conditions[?(@.type=='Ready')].status` | 0 |
| Size\|Count | integer | `.spec.size` | 1 |

## Spec

| Field | Type | Required | Default | Description |
| --- | --- | --- | --- | --- |
| `spec.color` | `string` | No | `"blue"` | Color picks the paint, a \<name\> or `a\|b`. |
| `spec.labels` | `map[string]string` | No |  | Labels are copied to every part. |
| `spec.parts` | `[]object` | No |  |  |
| `spec.parts[].name` | `string` | Yes |  |  |
| `spec.parts[].port` | `integer or string` | No |  |  |
| `spec.size` | `integer` | Yes |  | Size in units. |
| `spec.values` | `free-form object` | No |  | Values is free-form input. |

## Status

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `status.conditions` | `[]Condition` | No | Conditions of the widget. |
| `status.observedAt` | `string (date-time)` | No |  |

## Example

```yaml
apiVersion: example.dev/v1
kind: Widget
metadata:
  labels:
    app.kubernetes.io/name: demo
    tier: front
  name: widget-sample
spec:
  color: red
  size: 3
```

## Notes

It is the smallest \*unit\*.

A widget's `spec.size` never shrinks; see \#Widget and [0021:D4/D9](/enhancements/0021/decisions/).

## Served by

The `widget` reconciler watches every Widget.

## Enforcement

| Field | Rule | Enforced by |
| --- | --- | --- |
| the object | CEL rule `size(self.metadata.name) < 20`; refused with: name must be short | API server |
| `spec` | Required | API server |
| `spec` | CEL rule `self.size >= oldSelf.size`; refused with: size cannot shrink ([0021:D4](/enhancements/0021/decisions/)) | API server |
| `spec.color` | One of `blue`, `red` | API server |
| `spec.parts` | At most 8 items | API server |
| `spec.parts` | At most one item per `name` | API server |
| `spec.parts[].name` | Required | API server |
| `spec.parts[].name` | Must not be empty | API server |
| `spec.parts[].name` | Matches the pattern `^[a-z]+$` | API server |
| `spec.size` | Required | API server |
| `spec.size` | Greater than 0 | API server |
| `spec.size` | At most 10 | API server |
