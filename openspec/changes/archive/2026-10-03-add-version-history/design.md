# Design: add-version-history

## Context

Contracts read: C3 (`manifest.json`, its `tool`), C7 (unpack layout, the reserved `<out>/<project>/history.json`, the lock), C10 (`data/catalog.json`, `spec.fields` and the note that `type` and `default` compare only within one docs-kit minor). DESIGN.md "Phase 1b" lists the badges; DESIGN decision 12 limits the scope.

## Goals / Non-Goals

**Goals:** compute and write `history.json` in `pull`; fix its format (C13); record its digest in the lock.

**Non-Goals:** rendering badges (the site's templates do that, from the file); a diff view (DESIGN decision 12); history for docs-placed bundles (they have no segments of their own); tracking `level`, `appliesTo`, `servedBy`, `mark`, `optional` or `fulfilment` changes (not in DESIGN.md's badge table; a later additive op).

## Decisions

### D1. Inputs and order

For each tab project after unpacking, `pull` reads every segment directory it wrote this run that holds `data/catalog.json` with schema `docs.opmodel.dev/data/cue-catalog/v1`; a segment without one takes no part. Segments are ordered minors ascending (numeric MAJOR, then MINOR), `edge` last. The **previous** segment of a minor is the next lower minor pulled; the previous of `edge` is the newest minor pulled. The **floor** is the oldest minor pulled. With fewer than two such segments, no file is written and an existing one is removed.

History is recomputed on every run from the trees just unpacked, never cached across digests: a docs revision changes a segment's data after the fact.

### D2. Comparison mode per pair (the tool-minor gate)

A pair (previous, current) is compared in mode `full` when both bundles' `manifest.json` `tool` share MAJOR.MINOR, else in mode `paths`. In `paths` mode only field paths and `presence` are compared; `type`, `default`, `ref` and the spec-text fallback are skipped. Each pair and its mode are listed in `compared`.

### D3. Member changes

Members match by `fqn` (it includes the apiVersion). For a member present in both segments of a pair, its `spec.fields` are matched by `path`, and these changes are recorded, in this order per path (paths in the current segment's field order, then paths only the previous segment has, in its order):

| `op` | When | `from` / `to` | Mode |
|---|---|---|---|
| `removed` | path in previous only | previous `type` / `null` | both |
| `added` | path in current only | `null` / current `type` | both |
| `presence` | `presence` differs | the two presences | both |
| `type` | `type` differs | the two types | `full` |
| `default` | `default` differs (either may be `null`) | the two defaults | `full` |
| `ref` | `ref` differs (either may be `null`) | the two refs | `full` |
| `spec` | none of the above for the member, and the spec blocks' CUE tokens differ with comments skipped | `null` / `null`, `path` `""` | `full` |

`doc` is never compared: a doc-comment fix (a docs revision) must not read as a change. The token comparison is the one `revise` uses for `.cue` files (C3 "Docs revisions", step 5): `cue/scanner` with comments skipped, inserted commas equal written ones. It moves from `internal/gitsrc/doconly.go` to `internal/cuetok` so both callers share it.

### D4. `history.json` (C13)

```cue
package schema

#History: {
	schema:   "docs.opmodel.dev/history/v1"
	project:  #Project
	tool:     #SemVer                           // the opm-docs that computed it
	floor:    =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"
	segments: [#Segment, #Segment, ...#Segment] // ordered: minors ascending, edge last
	compared: [...{from: #Segment, to: #Segment, mode: "full" | "paths"}]
	members: [string]: #MemberHistory           // keyed by FQN
	removed: [#Segment]: [#Removed, ...#Removed] // a segment key only when something was removed in it
	lineage: [string]: [#Segment]: [string, ...string] // "<kind>/<name>": segment: apiVersions, newest first (C8 order)
}

#MemberHistory: {
	kind:         "resource" | "trait" | "blueprint"
	name:         string
	apiVersion:   string
	first:        #Segment   // the first segment that has it
	firstIsFloor: bool       // first == floor: the site writes "in <floor> or earlier", never "added in"
	in: [#Segment, ...#Segment]
	changes: [#Segment]: [#Change, ...#Change] // a segment key only when the member changed against its previous segment
}

#Change: {
	op:   "added" | "removed" | "presence" | "type" | "default" | "ref" | "spec"
	path: string             // a spec.fields path; "" for op "spec"
	from: string | null
	to:   string | null
}

#Removed: {
	fqn:        string
	kind:       "resource" | "trait" | "blueprint"
	name:       string
	apiVersion: string
	lastIn:     #Segment     // the last segment that had it: the site links <root><lastIn>/<page>/
	page:       string       // its page in lastIn, "traits/backup-v1alpha1"
}
```

Serialization: JSON, two-space indent, trailing newline, object keys in the order above for structs and sorted for maps (Go's encoder), no timestamps. The same pulled trees and tool write the same bytes. Example for `backup`, first in 4.5 (the floor) and given a required field in 4.6:

```json
"opmodel.dev/catalogs/opm/traits/backup@v1alpha1": {
  "kind": "trait", "name": "backup", "apiVersion": "v1alpha1",
  "first": "4.5", "firstIsFloor": true, "in": ["4.5", "4.6", "edge"],
  "changes": {"4.6": [{"op": "presence", "path": "retention.daily", "from": "optional", "to": "required"}]}
}
```

What the site derives (recorded in C13 so both sides agree): "Added in X" when `first` is a minor and `firstIsFloor` is false; "In <floor> or earlier" when it is true; "Unreleased" when `first` is `edge`; "Changed in X" for every key of `changes`; "Removed in X" on X's kind index from `removed`; "Newer version" from `lineage` of the page's own segment. Field-level changes render as entries of the page-end "Changes in X" list, not inline in the spec block, which is a code fence (decided in planning; DESIGN.md's "badges on each changed spec field" becomes list entries naming the field).

### D5. Where it is written, and the lock

`pull` writes `<out>/<project>/history.json` after the sweep and before the lock, for tab projects only, in every mode (registry, `--frozen`, `--offline`, `--local`); `--local` segments count like pulled ones. It validates the bytes against `#History` before writing.

`schema/lock.cue` gains an optional key after `bundles`:

```cue
#Lock: {
	schema: "docs.opmodel.dev/lock/v1"
	tool:   #SemVer
	config: =~"^sha256:[0-9a-f]{64}$"
	bundles: [...#Locked]
	history?: [...{project: #Project, digest: =~"^sha256:[0-9a-f]{64}$", path: string & !=""}] // path relative to the lock's directory
}
```

Entries sorted by project; the key is omitted when no history file was written. The schema id stays `lock/v1`, and every lock without the key still validates. `#Lock` is closed, so an `opm-docs` that predates `history` refuses a lock carrying it (`--frozen`); only the site reads its lock, with the one `opm-docs` it pins, so the site's tool bump and its first lock with `history` arrive together. `--frozen` does not compare the old digest (history is a function of the trees and the tool, and the tool may have moved); it writes the new one.

### D6. Commands

No new command or flag. Exit codes unchanged: a history file that fails `#History` is a tool bug and exits 2 naming the project ("history for catalog-opm does not validate: <error>; report it against opm-docs").

## Research & Decisions

### Field badges in the spec block (decided in planning)

**Context**: DESIGN.md puts badges "on each changed spec field".
**Explored**: the member page's spec is a `cue` fence rendered by docs-kit at build time; history exists only at pull time.
**Options considered**:
1. Re-render member pages at pull time - breaks "the site assembles, the producer builds" and the bundle digest.
2. Site injects HTML into the code block - fragile, outside the page dialect.
3. Field changes as entries of the "Changes in X" list - fits DESIGN decision 12's list, no page rewrite.
**Decision**: option 3.
**Rationale**: the only option that keeps pages as published and the dialect intact.

### Edge in the history (decided in planning)

**Context**: with the tab starting at opm 4.5 (DESIGN decision 8, amended), the only pair today is 4.5 and edge.
**Decision**: edge takes part, compared with the newest minor, and a member first seen in edge is "Unreleased".
**Rationale**: it is the one comparison readers can use now; it needs no special case beyond the order.

### Lock digest versus regenerate-on-frozen (decided in planning)

**Options considered**: 1. digest in the lock - the site can check the mounted file; 2. no record, document that `--frozen` regenerates - nothing to check.
**Decision**: option 1, as an optional key, not compared under `--frozen`.

### Edge cases found in implementation (accepted 2026-10-03)

- A member removed and later returned keeps its original `first`; it is listed under `removed` for the segment that dropped it, and the segment it returns in records no change for it.
- A spec block that does not scan as CUE (the extractor never writes one) counts as changed: the member gets one `spec` change.

## Risks / Trade-offs

- A change hidden behind `ref` (a shared schema) is not reported. Under-claiming is the safe direction; a later change can follow refs within the module.
- Mixed tool minors weaken the comparison to paths; the file says so per pair so the site can word it.

## Durable decisions

- `history.json` format and rules: `docs/contracts.md` C13 (new), with D3's table and D4's schema and derivations.
- The lock's `history` key: `docs/contracts.md` C7.
- `README.md`: one line under `pull` saying it writes `history.json`.
