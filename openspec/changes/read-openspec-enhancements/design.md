## Context

The `enhancements` source (docs/contracts.md C21) reads `NNNN/` and `archive/NNNN/`, each with `config.yaml`, `README.md` and seven numbered documents, and writes `data/enhancements.json` (v1) and the section's pages. The enhancements repository is moving to OpenSpec changes (its design is part two, section 5 of the swarm's design record): 17 live entries convert, ten stay at `archive/NNNN/`. A new entry is `openspec/changes/<id>-<slug>/` with `config.yaml`, a generated `README.md`, `proposal.md`, `design.md`, optional `tasks.md` and `specs/<cap>/spec.md` delta files; an archived one is `openspec/changes/archive/<date>-<id>-<slug>/`. The main capability specs live in `openspec/specs/<cap>/spec.md`. A new entry stores no `updated`; the date comes from git. The publish workflow already checks out with `fetch-depth: 0` (`.github/workflows/publish.yml`), so the history exists in the docs build.

## Goals / Non-Goals

**Goals:** read both layouts from one source; one data file for both; pages, capability pages, redirect data and dialect links for the new layout; the contract and spec say so.

**Non-Goals:** any change to the bundle format, the other extractors, the publish workflow or the site (the site's partials change later, in opmodel.dev); validating the enhancements rules (the `enh` checker's job); a `history` of requirements.

## Research & Decisions

### The legacy reader's scope

**Context**: After the migration only the ten archived entries are old layout.
**Explored**: `internal/extract/enhancements/extract.go` `entryDirs` reads the entry root and `archive/`.
**Options considered**:
1. Keep reading a live `NNNN/` too - nothing breaks for an unmigrated repository; two live layouts stay possible and a half-migrated repository builds.
2. Read only `<legacy>/NNNN/` and refuse a live `NNNN/` - the contract the design states; a repository that is not migrated fails loudly.
3. Read only `<legacy>/NNNN/` and ignore a live `NNNN/` - an entry vanishes from the site silently.
**Decision**: option 2. A directory named `NNNN` at the entry root is refused: "entry NNNN/ is a legacy live folder; convert it to openspec/changes/ or move it to archive/".
**Rationale**: Principle VI prefers refusing to guessing. The enhancements repository keeps its docs-kit pin until the migration merges, so nothing refuses in practice.

### Source config

**Decision**: three fields on `#Enhancements`, all relative to `dir`, each a repo-relative path like `dir`:

```cue
#Enhancements: {
	kind:        "enhancements"
	dir:         *"." | =~"^[^/.][^.]*$"
	changes:     *"openspec/changes" | =~"^[^/.][^.]*$" // OpenSpec changes: live <id>-<slug>/, archive/<date>-<id>-<slug>/
	specs:       *"openspec/specs" | =~"^[^/.][^.]*$"   // the capability specs
	legacy:      *"archive" | =~"^[^/.][^.]*$"          // old-layout entries NNNN/
	title:       *"Enhancements" | string & !=""
	description: string & !=""
}
```

`config.Load` refuses a path that equals or nests another of the three. A missing `changes`, `specs` or `legacy` directory is not an error: it holds nothing.

### Data schema v2

**Context**: Principle II: additive, and a breaking change bumps the identifier. The site must read both layouts from one array.
**Options considered**: (1) two arrays, `entries` and `changes`; (2) one array, entries tagged with `layout`, fields per layout; (3) the same fields for both layouts, new ones empty on a legacy entry.
**Decision**: option 2, as the design record's N8 states. A legacy entry has exactly its v1 fields plus `layout: "legacy"`. An openspec entry has this shape:

```json
{
  "id": "0026", "layout": "openspec", "slug": "module-dictated-catalog-versions",
  "title": "...", "summary": "...", "status": "accepted", "category": "schema",
  "breaking": true, "affects": ["core", "library"],
  "created": "2026-09-08", "updated": "2026-10-16", "archived": false,
  "after": ["0010"], "supersededBy": [],
  "accepted": {"date": "2026-10-16", "commit": "4be19d0"},
  "progress": {"done": 0, "total": 5},
  "capabilities": ["authored-platform", "single-build-render"],
  "page": "0026",
  "documents": [{"slug": "proposal", "title": "Proposal", "page": "0026/proposal", "file": "openspec/changes/0026-x/proposal.md"}],
  "aliases": [{"from": "0026/problem", "to": "0026/proposal"}]
}
```

`breaking` is `true`, `false` or `null` (unset in a draft). `accepted` is an object or `null`. `capabilities` are the directory names under the entry's `specs/`. `progress` counts the `- [ ]` and `- [x]` lines of `tasks.md` outside code fences (zeros when the file is absent). `supersededBy` is the `ids` of the `superseded_by` record. `documents[].file` of `requirements` is the entry's `specs` directory. A v1 reader finds every v1 field on a legacy entry; it must read `layout` before it reads the rest of an openspec entry.
**Rationale**: one array keeps the page-to-entry join the site already does. Different `supersededBy` types per layout cost the site one `reflect.IsSlice` it already has.

### `updated` from git (the open check of the design record)

**Context**: the design record leaves open how the extractor gets history: a full-depth checkout, or the bundle step.
**Explored**: `publish.yml` checks out the caller with `fetch-depth: 0` in the build job (lines 123-127 and 131-136), and `gitsrc.Repo.LastMod` already reads the last commit date of a path for every page's `lastmod`.
**Decision**: no new input. The extractor receives `Options.LastCommit(path)`, which the build wires to `gitsrc.Repo.LastMod` at the commit built; `updated` is the UTC date of the last commit at or before the commit built that touched the entry's folder. Two guards: a shallow clone (`git rev-parse --is-shallow-repository` is `true`) is refused when any openspec entry exists, because `git log` then returns the boundary commit; an entry folder with no commit (an uncommitted entry in a local preview, where `source.dirty` already stops a push) takes `created`.
**Rationale**: the workflow provides the depth; the refusal turns a silent wrong date into a message that names the fix (a checkout with `fetch-depth: 0`). What the docs build must provide: full history of the commit built. An archive move or a regenerated README that changes a file is a commit that touched the folder and moves the date; a regeneration that changes nothing is not.

### Pages of an openspec entry

| Page | From | Front matter |
|---|---|---|
| `<id>/_index.md` | `README.md` | `title` "<id>: <title>", `description` the summary, `weight` id plus 1 |
| `<id>/proposal.md` | `proposal.md` | "<id>: Proposal", "What changes, and why.", `type: explanation`, `weight: 1` |
| `<id>/design.md` | `design.md` | "<id>: Design", "How it works, and why this way.", weight 2 |
| `<id>/requirements.md` | `specs/<cap>/spec.md`, when the entry has any | "<id>: Requirements", "Exactly what must hold, by capability.", weight 3 |
| `<id>/tasks.md` | `tasks.md`, when it exists | "<id>: Tasks", "Which repository still has work.", weight 4 |

The requirements page holds one `## <cap>` heading per capability in name order, then that delta file with its text cleaned as every page is, its leading `# ` heading removed and every heading one level down (`## ADDED Requirements` becomes `###`); a heading that would pass level 6 is refused. When the capability exists in `specs/`, a line under the heading links its capability page. `source` is absent on this page and `lastmod` is the newest of the delta files (`Data.Inputs`). Links resolve as for the old layout; `specs/` of the entry and its files resolve to the requirements page, a file under `specs/` of the repository root to its capability page, anything else that exists at the commit to GitHub. `evidence/` is not a page.

### Capability pages

`capabilities/_index.md` lists every capability of `specs/` (a table of links with the summary); `capabilities/<cap>.md` is the main spec with its `# ` heading removed, `title` the name with hyphens as spaces and a capital, `description` the first sentence of its `## Purpose`, followed by `## Entries that changed it`: a row per archived change that holds a delta for the capability, with the archive date, and the entry as a link when the entry's status is `delivered`. A change that holds a `seed-` prefix is a row without a link. An archived entry that is not `delivered` (rejected, withdrawn, superseded) is no row, because its deltas never reached `specs/`. Rows run by date, then name. No `specs/` directory, no capability pages.

### Redirects

**Decision**: the map of the design record's N9 (`problem` and `questions` to `proposal`; `design`, `risks` and `operational` to `design`; `decisions` to `requirements`; `graduation` to `tasks`), as `aliases` `{from, to}` page paths, skipping `design` to `design`. When an entry has no `requirements` or `tasks` page, the alias points at the entry's own page. The ten legacy entries have none. docs-kit writes data only; the site's adapter adds Hugo `aliases` from it (Hugo `aliases` through `AddPage` is not verified).
**Trade-off**: docs-kit cannot know which ids had old pages, so an entry created after the migration also carries these aliases, and the site writes five redirect stubs for URLs that never existed.

### Dialect links and the link policy

`reEnhOK` admits `proposal`, `requirements`, `tasks`, `capabilities/` and `capabilities/<cap>/` beside the seven old slugs, and the error message names the new forms. The `link` policy (`internal/doctext`) additionally turns `enhancement NNNN` into `[enhancement NNNN](/enhancements/NNNN/)` and, in `enhancements NNNN and NNNN`, each id into a link to its page; the `NNNN:Dn` form keeps its `/decisions/` link. The `crd` model keeps plain prose and links in its renderer, so its extractor unwraps the new links as it unwraps the decision links, and its renderer links the same forms. The `strip` policy leaves `enhancement NNNN` as written, as it does today.

### Fixtures

The extractor fixture becomes a mixed repository: two legacy entries under `archive/`, the worked example's entry 0026 (config with `breaking`, generated README, proposal, design, tasks, four delta specs, one evidence file), its two seed changes and two main specs, and one archived delivered entry. The tests run on a real git repository for the `updated` cases (`internal/gittest`).

## Durable decisions

- The legacy reader reads `<legacy>/NNNN/` only and refuses a live `NNNN/`: lands in `docs/contracts.md` C21.
- Schema v2 shape, `layout`, `updated` from git and the full-depth requirement: C21 and `README.md`.
- The alias map and the stub trade-off: C21.
- The `enhancement NNNN` link policy: C21 and the Doc-comment rules of `docs/contracts.md`.
- Hugo `aliases` through `AddPage` is not verified: stays with the change.

## Risks / Trade-offs

- A shallow clone now refuses an enhancements build that holds an openspec entry. The publish workflow is full depth; a local `serve` of a shallow clone gets the message.
- The v2 identifier makes a consumer that checks `v1` exactly refuse the file. opmodel.dev reads `data/enhancements.json` without checking the identifier today; its change must still accept both before the pin moves.
- The `link` policy change reaches every source that uses `link`; the new forms are additive and a text without them is byte for byte as before.
