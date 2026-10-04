---
title: "Gadget"
description: "Gadget is a cluster-wide gadget."
type: reference
weight: 2
---

Gadget is a cluster-wide gadget.

## At a glance

| Property | Value |
| --- | --- |
| Group | `example.dev` |
| Version | `v1` |
| Kind | `Gadget` |
| Scope | Cluster |
| Resource | `gadgets` |

## Spec

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `spec.module` | `string` | Yes |  |

## Enforcement

| Field | Rule | Enforced by |
| --- | --- | --- |
| `spec.module` | Required | API server |
