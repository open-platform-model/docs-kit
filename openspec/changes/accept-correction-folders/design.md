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
touch several capabilities. The slug rule is the store's: non-empty kebab-case. Both patterns are
anchored, unlike the bare `seed-` prefix test:

- live: `^correction-[a-z0-9]+(?:-[a-z0-9]+)*$`
- archived: `^\d{4}-\d{2}-\d{2}-correction-[a-z0-9]+(?:-[a-z0-9]+)*$`

`correction-`, `correction-Foo_Bar` and `2026-10-23-correction-` match neither and are refused
(exit 2). `0123-correction-x` matches the entry pattern and is entry 0123 with slug
`correction-x`; no special case. The seed test stays as it is.

### The history row

**Context**: `changeRow` has `id == ""` for a seed, rendered as `seed`.
**Options considered**:
1. Row text `correction`, unlinked, one row for each touched capability. Simple: one string.
2. Omit corrections from the table. The page would hide a change to its own text.
3. A row with the slug and a link. There is no page to link to.
**Decision**: option 1. `changeRow` gains a `kind` string (`seed` or `correction`) for the unlinked
cases, replacing the empty-id test; rows sort by date, then id (empty first), then kind, so `correction` comes before `seed` on one date. C21 and the spec state this key.
**Rationale**: a reader of a capability sees that its text changed after delivery, with the date.

### Refusal message

The message becomes `change folder %s is not %s, a seed-<capability> change or a correction-<slug> change; an enhancement is a change named by its id and slug` (the `%s` forms as today). Exit code stays 2. A rendered row is `| correction | 2026-10-23 |`.

### Earlier entries

An earlier entry's requirements page gets no marker of a later correction. Present behaviour stays.

## Risks / Trade-offs

A correction folder with a missing `specs/` adds no row (as for a seed). A name that
looks like a correction but is malformed fails the build with the existing message.

## Durable decisions

- The correction folder pattern and the `correction` row: land in `docs/contracts.md` C21 and the
  `enhancements-bundle` spec at apply.
