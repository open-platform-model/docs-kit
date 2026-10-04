## Context

The `crd` source (C18) renders every kind into one completable page at `page`. The `cobra` source (C19) and the `go-api` source (C20) render a section: `<section>_index.md` plus one page per command or package, which the site's sidebar shows as a collapsible section. opm-operator wants the same for its resource kinds under `reference/operator/`. Released operator tags carry a `docs-kit.cue` with `page:`, and a release build reads the tag's own config (C5), so `page` must keep working.

## Goals / Non-Goals

**Goals:** a `section` layout for the `crd` source; `page` unchanged byte for byte.

**Non-Goals:** a per-kind authored intro (only the index is completable); changing what a kind's entry shows; changing any URL other than the new section's.

## Decisions

### D1. Config: `section`, exclusive with `page`

```cue
#CRD: {
	kind:     "crd"
	dir:      =~"^\\./[^/]"
	samples?: =~"^\\./[^/]"
	hideSamplesMatching: *[] | [...string & !=""]
	stripLabels: [string & !=""]: string
	// Exactly one of page and section. page: one completable page holding
	// every kind. section: the section index and one page per kind.
	page?:    =~"^([a-z0-9]+(-[a-z0-9]+)*/)*[a-z0-9]+(-[a-z0-9]+)*\\.md$"
	section?: =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$" // "reference/operator/"
	matchN(1, [{page!: _, ...}, {section!: _, ...}])
	title:       string & !="" // the page's or the section index's front matter
	description: string & !=""
	weight?:     int & >=1
	order?: [string & !="", ...string & !=""]
	reconciledBy?: [string & !=""]: string & !=""
	citations?: #Citations
}
```

`matchN` (CUE v0.17) states "exactly one" in the schema. A config with both or neither fails `#Config` (exit `1`); `checkBundle` repeats the rule with a plain message first, since a failure inside the `#Source` disjunction reads poorly ("a crd source takes exactly one of page (one page holding every kind) and section (an index and a page per kind)").

### D2. Data model, additive

```json
{
  "schema": "docs.opmodel.dev/data/crd/v1",
  "citations": "link",
  "layout": "section",
  "page": {"path": "reference/operator/_index.md", "title": "Operator Reference", "description": "...", "weight": 7},
  "kinds": [{"kind": "ModuleInstance", "page": "reference/operator/moduleinstance.md", "...": "..."}]
}
```

- `layout` is `"page"` or `"section"`. A data file without it (written by an older opm-docs) decodes as `"page"`.
- `page` is always the completable page: the single page in page layout, the section index in section layout.
- `kinds[].page` is the kind's page path in section layout, `null` in page layout. The extractor derives it (`<section><kind lower-cased>.md`) so the renderer reads it rather than recomputes it.

### D3. Pages in section layout

| Path | Page |
|---|---|
| `<section>_index.md` | front matter `title`, `description`, `weight` when set, no `type` (as the `cobra` and `go-api` indexes); the intro "Each page in this section covers one resource kind, generated from its CustomResourceDefinition."; `## Kinds` and a table `\| Kind \| Scope \| Summary \|`, one row per kind in model order, the kind linking its page (`/docs/<section><kind lower-cased>/`). Completable (C15) with heading `## Kinds`. No `source`; `lastmod` the newest of every file read |
| `<section><kind lower-cased>.md` | front matter `title` the kind, `description` its summary with every citation removed (or "The <Kind> resource." when it has none), `type: reference`, `weight` its 1-based position in model order; then the kind's entry without its `## <Kind>` heading and with every other heading one level up (`## At a glance`, `## Spec`, `## Status`, `## Example`, `## Notes`, `## Served by`, `## Enforcement`). `source` the kind's CRD file, `lastmod` the newest of that file and the sample file read for it; no `edit` |

### D4. Cross-kind links

The renderer writes no link between kinds today: the only links it writes are decision citations (C18), so section layout has no in-page anchor to rewrite. Links into the old page from authored pages (`/docs/reference/operator-resources/#moduleinstance`) are the consuming repository's to change.

### D5. One template for both layouts

`templates/crd/entries.md.tmpl` defines `crd-kind`, the body of one kind, taking the heading prefix of its parts (`###` in page layout, `##` in section layout). Page layout writes `## <Kind>` then `crd-kind` with `###`; section layout writes `crd-kind` with `##` alone. Page output is unchanged: the crdref parity test and the existing goldens guard it.

## Research & Decisions

### Where the kind page's heading level comes from
**Context**: a kind page's title is the kind, which the site shows as the page's `h1`.
**Options considered**:
1. Keep `## <Kind>` and the `###` parts - identical entry text, but a redundant heading and `#spec`-style anchors nested under a heading that repeats the title.
2. Drop `## <Kind>`, lift the parts to `##` - a page outline like the go-api and cobra pages, and anchors (`#spec`, `#status`) no longer need the `-1`, `-2` suffixes the single page gives later kinds.
**Decision**: 2.
**Rationale**: the page's title already names the kind; one kind per page makes the suffixes C18 warns about disappear.

### Completable kind pages
**Context**: today the authored page completes the single page.
**Decision**: only the section index is completable; kind pages are generated only.
**Rationale**: the operator's authored text is an intro to all four kinds, which belongs on the index; a per-kind intro has no author today (Principle VI).

### Description of a kind page
**Context**: front matter is plain text; the summary may hold a decision citation under `"link"`.
**Decision**: the summary cleaned by `doctext.Clean` (every citation removed), fallback "The <Kind> resource.".
**Rationale**: front matter cannot carry a link, and a bare `0015:D3` in a meta description says nothing to a reader.

## Risks / Trade-offs

- Two kinds differing only in case would share a page: refused, exit `2`, naming both.
- An `opm-docs` older than this change refuses a `docs-kit.cue` with `section` on a `crd` source (closed `#CRD`): opm-operator must move its `.opm-docs-version` and `publish.yml` ref to the release carrying this change in the same PR that switches to `section`.

## Durable decisions

- D1 (config shape), D2 (data model) and D3 (pages) land in `docs/contracts.md` C18 and C6, and `README.md`'s source table.
- D4 (no cross-kind links, consumers move their own links) lands in C18's Consumers paragraph.
- D5 stays with the change.
