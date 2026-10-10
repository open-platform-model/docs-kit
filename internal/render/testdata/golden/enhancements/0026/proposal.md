---
title: "0026: Proposal"
description: "What changes, and why."
type: explanation
weight: 1
---

A platform pins one exact version of each catalog, and every module renders against it, even if the author tested another. This entry splits the pin in two: the platform sets a range, and the module picks inside it.

## Why

One number admits a catalog and picks every instance's version. A module that needs a newer definition waits for a platform edit, and a bump re-renders every instance at once.

## What Changes

- A module renders against the catalog version it was tested with, when the platform admits it.
- The platform names the catalogs it admits, each with a floor, an optional ceiling and a prerelease opt-in.
- A pin outside the range is refused by name, never moved.
- A provider states a window for its own catalog. The platform may override it.
- Several majors of one catalog run side by side, one major per render.

## What this is not

- Not a change to matching, to the transformer set, or to who admits a catalog.
- Not provider routing, bridge transformers, or floating an instance forward without its owner.
- Not module-hosted transformer code.

## Example

```text
platform: catalogs/opm@v4  floor 4.0.0, no ceiling
module gotify   pins 4.1.0   before: renders at 4.3.0   after: renders at 4.1.0
module fileflows pins 4.6.0  before: refused             after: renders at 4.6.0
platform sets floor 4.2.0    gotify is refused by name; the others do not move
```

## Capabilities

New: `catalog-version-selection`, `authored-platform`. Changed: `single-build-render`, `registration-acceptance`.

## Impact

Breaking, pre-GA: a shipped core definition is renamed, and the Platform resource spec changes shape. Five repos change: core, library, cli, opm-operator, catalog.

## Open questions

- Not blocking: Which platform faults fail at acceptance, and does a bad platform fail whole or per entry?
- Not blocking: How many distinct resolutions does a real fleet produce, and what do they cost?
- Not blocking: Is the per-entry registry override needed?
- Not blocking: Does the provider window need a prerelease flag?
- Not blocking: Must a provider's own major upgrade run side by side?
- Not blocking: May a new catalog major ship a bridge to the old one?
- Not blocking: How does an operator see that a major is drained?
- Not blocking: May a provider build that depends on two majors enter both resolutions?
