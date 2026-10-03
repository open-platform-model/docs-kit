# Design: add-enhancements-bundle

## Context

Contracts read: C3 (`#Placement`), C7 (pull config, layout, lock), C8 (URLs; the `/enhancements/` link forms), C9, C11 (dialect 1 allows `/enhancements/`, `/enhancements/<NNNN>/`, `/enhancements/<NNNN>/<document>/`), C15 and C16 (from `generalize-build-assembly` and `pull-docs-placement`). DESIGN decision 18. The behavior to move: opmodel.dev `site/enhancements/_content.gotmpl` (pages and header data from `config.yaml`), `layouts/_partials/opm/enh-clean.html` (comments and first heading removed, shortcode delimiters refused), `layouts/enhancements/_markup/render-link.html` (relative links resolved), `site/scripts/gen-mounts.sh` (the section page from `INDEX.md`, `data/opm/enhancements.json`, the path list for GitHub links), and `versions.conf`'s `[section "enhancements"]`.

## Goals / Non-Goals

**Goals:** the `section` placement; the `enhancements` source and its pages; `pull` of sections; the graph link form.

**Non-Goals:** versioning the enhancements (DESIGN decision 18: edge only); rewriting enhancement prose for the dialect beyond the mechanical transforms (the enhancements sibling fixes its sources); the site's layouts for the entry header (they read `data/enhancements.json`).

## Decisions

### D1. The `section` placement (C3, C21)

```cue
#Placement: {
	kind: "tab" | "docs" | "section"
	if kind == "tab" {root: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"}
	if kind == "docs" {root: "/docs/", owns: *[] | [...#Owned]}
	if kind == "section" {root: "/enhancements/"}
}
```

The root is fixed to `/enhancements/` until a second section has a consumer (Principle VI). In `docs-kit.cue`, a section bundle's `version` is optional and, when present, ignored; `build --release` and `revise` refuse a section project (exit 1: "enhancements is a section bundle; it builds from main only (edge)"). URLs: `content/<path>` publishes at `/enhancements/<page URL>` (C8 gains the row). Bundle-mode lint: a link into `/enhancements/` must name a page of the bundle (the tab rule with an empty segment); other links follow docs-mode rules.

### D2. The `enhancements` source (C21)

```cue
#Enhancements: {
	kind:  "enhancements"
	dir:   *"." | =~"^[^/.][^.]*$"   // the repository's entry root
	title: *"Enhancements" | string & !=""
	description: string & !=""       // the section page's description
}
```

The enhancements repository's file, as its sibling change writes it:

```cue
bundles: enhancements: {
	placement: {kind: "section", root: "/enhancements/"}
	sources: [{kind: "enhancements", description: "OPM's design record: every proposal, its decisions and its status."}]
}
```

Reading (YAML through CUE's `encoding/yaml`): every `NNNN/config.yaml` and `archive/NNNN/config.yaml`, skipping `0000`; `id` must equal the directory; `README.md` and exactly one `0<n>-*.md` per `n` in 1..7 must exist. Pages:

| Path | From | Front matter |
|---|---|---|
| `_index.md` | `INDEX.md` | `title` (config), `description` (config) |
| `graph.md` | `GRAPH.md` | `title: "Relationship graph"`, `description: "How the enhancements depend on and amend one another, by category."`, `type: explanation`, `weight: 1` |
| `<NNNN>/_index.md` | `README.md` | `title: "<NNNN>: <config title>"`, `description` = config `summary` (whitespace collapsed), `weight` = id + 1 |
| `<NNNN>/<slug>.md` | `0<n>-*.md` | `title: "<NNNN>: <document title>"`, `description` = the document's fixed description, `type: explanation`, `weight` = n |

Slugs, titles and descriptions per document are the site adapter's today (`problem` "Problem statement" "What is wrong today, and for whom.", `design`, `decisions`, `graduation`, `risks`, `operational`, `questions`). `manifest.json` `pages[].source` is the file's repository path; `lastmod` is its git date (C3); the entry's own `created` and `updated` are in the data file, and the site shows those.

### D3. Transforms

Applied to each Markdown file, in order: (1) refuse `{{<` or `{{%` (exit 2 naming the file); (2) remove HTML comments (`<!--` to `-->`, across lines); (3) remove the first `# ` heading line and the blank lines after it; (4) tag every opening code fence without a language as `text`; (5) resolve links, inline and reference definitions, outside code fences and code spans, by the site link hook's rules as the spec lists them, with the repository path list taken from `git ls-tree -r` at the commit built; (6) replace a link text that equals the target's file name by the target page's title (`INDEX.md` "the index", `GRAPH.md` "the relationship graph", others the target page's `title`). Root-absolute `/docs/` links pass through (docs-mode rules apply, and the site's link check resolves them).

What these transforms do not fix (raw HTML lines, component-tag-like lines, images) is a lint violation that fails the build; the spike (section 1) lists them, and the enhancements sibling fixes them in its sources. The lint is never relaxed for a section (Principle III).

### D4. Data model (C21)

```json
{
  "schema": "docs.opmodel.dev/data/enhancements/v1",
  "repo": "open-platform-model/enhancements",
  "entries": [
    {
      "id": "0025", "slug": "self-describing-modules", "title": "Self-Describing Modules",
      "summary": "...", "status": "draft", "category": "schema", "affects": ["core", "library"],
      "created": "2026-09-08", "updated": "2026-09-29", "archived": false,
      "dependsOn": ["0010", "0015", "0019"], "amends": [], "supersedes": [], "revives": [], "supersededBy": null,
      "page": "0025",
      "documents": [{"slug": "problem", "title": "Problem statement", "page": "0025/problem", "file": "0025/01-problem.md"}]
    }
  ]
}
```

Entries by id. `authors` and `history` are not copied (the site does not publish them today). Fields the site's layouts need later are additive.

### D5. Pull of sections (C7, C21)

```cue
#Pull: {
	...
	sections: [#Project]: {
		repo: =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$"
		root: "/enhancements/"
	}
}
```

`pull` resolves `edge` only, verifies it with the section's `repo`, checks placement `kind: "section"` and the root, unpacks to `<out>/<project>/edge/` and locks it in `bundles` (`root: "/enhancements/"`, `segment: "edge"`, `tag: "edge"`); `#Pulled.root` and `#Local.root` in `schema/lock.cue` widen to `^/catalogs/...$|^/enhancements/$`. No `edge` tag fails the pull ("enhancements has no edge build; the section would be empty"). `--local enhancements@edge=<dir>` works as for a tab. No history is computed for a section.

### D6. The graph link form

Dialect 1 gains the destination `/enhancements/graph/` (optional fragment). It lands with a conformance fixture `link-enhancements-graph`, which opmodel.dev copies with its shell-lint update (C11's re-sync rule). The dialect version stays `1`: the change only accepts a form.

### D7. Commands

No new command or flag. Exit codes as C6 and C7.

## Research & Decisions

### Producer-side pages versus raw files in the bundle (decided in planning)

**Context**: the site builds the section from the raw repository today.
**Options considered**:
1. Ship the raw tree as `data/` and keep the site adapter - `data/` is JSON only (C3), and the site would keep generating pages (against Principle I).
2. A lint-relaxed profile for sections - against Principle III.
3. Producer-side pages through mechanical transforms, sources fixed for what remains - one rule set for every page on the site.
**Decision**: option 3, with a spike to size the source fixes first.

### Section root fixed to `/enhancements/` (decided in planning)

**Decision**: a generic section root waits for a second section (Principle VI); the schema names the one root.

## Risks / Trade-offs

- An enhancement link that the repository's own `check-links.sh` accepts but the bundle refuses (or the reverse) blocks the enhancements repository's `main`. The spike runs both over `main` and the sibling aligns them.
- The site loses "View source at the commit built" from its stamp; it now reads `source.commit` from the manifest, as for every bundle.

## Durable decisions

- C21 (new): D1 to D6. C1: the `enhancements` row (already reserved). C3: `#Placement` `section`. C7: `sections`, the lock root. C8: the `/enhancements/` URL row. C11: the graph form.
- `README.md`: placements table.
