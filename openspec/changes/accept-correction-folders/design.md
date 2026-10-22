## Context

`classifyChange` in `internal/extract/enhancements/openspec.go` returns "not an entry" for a
`seed-` folder (live) and `<date>-seed-` (archived), and an error for every other name that is not
`<id>-<slug>` or `<date>-<id>-<slug>`. `capabilityHistory` in `capabilities.go` walks
`archive/`, gives a seed a row with no id, and skips any other non-entry name.

## Goals / Non-Goals

**Goals:** a correction folder breaks nothing; a capability page shows that a correction touched it.

**Non-Goals:** a page for a correction; any data field; reading a correction's content.

## Decisions

### Name patterns

**Decided in planning.** The folder name is a slug, not a capability, because one correction may
touch several capabilities. A live folder matches the prefix `correction-`. An archived folder
matches `^\d{4}-\d{2}-\d{2}-correction-`. Both are the same shape as the seed checks, so a
second regexp `reCorrectionDate` sits beside `reSeedDate`.

A folder `correction-` with no slug, or `0123-correction-x`, is not a correction and is still
refused (the entry pattern needs a four-digit id, so `0123-correction-x` is an entry).

### The history row

**Context**: `changeRow` has `id == ""` for a seed, rendered as `seed`.
**Options considered**:
1. Row text `correction`, unlinked, one row for each touched capability. Simple: one string.
2. Omit corrections from the table. The page would hide a change to its own text.
3. A row with the slug and a link. There is no page to link to.
**Decision**: option 1. `changeRow` gains a `kind` string (`seed` or `correction`) for the unlinked
cases, replacing the empty-id test; rows still sort by date, then id, then kind.
**Rationale**: a reader of a capability sees that its text changed after delivery, with the date.

### Refusal message

The message for an unknown name lists `seed-<capability>` today. It adds
`correction-<slug>` so the fix is clear. Exit code stays 2.

## Risks / Trade-offs

A correction folder with a missing `specs/` adds no row (as for a seed). A name that
looks like a correction but is malformed fails the build with the existing message.

## Durable decisions

- The correction folder pattern and the `correction` row: land in `docs/contracts.md` C21 and the
  `enhancements-bundle` spec at apply.
