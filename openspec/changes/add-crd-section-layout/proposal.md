## Why

The site's reference sidebar shows the CLI as a collapsible section with one page per command, but the operator's resources as one flat page. The owner wants the operator reference to follow the CLI's layout under `/docs/reference/operator/`: a section with one page per resource kind. The `crd` source can only write one page today (C18), so opm-operator cannot publish a section until docs-kit can render one.

## What Changes

- **`section` on `#CRD`**: a new field, mutually exclusive with `page` (exactly one is set). `page` keeps today's single completable page byte for byte, so a released operator tag built with its old `docs-kit.cue` still builds.
- **Section layout**: with `section: "reference/operator/"`, the renderer writes `<section>_index.md` (completable, heading `## Kinds`, a table linking every kind page) and one page per kind, `<section><kind lower-cased>.md`, titled by the kind, weighted by its position in model order, holding that kind's entry with its headings one level up (`## Spec`, not `### Spec`).
- **`data/crd.json`**: additive fields `layout` (`"page"` or `"section"`) and per kind `page` (its page path in section mode, `null` otherwise). The schema identifier stays `docs.opmodel.dev/data/crd/v1`.
- **Refusal**: two kinds whose lower-cased names are equal (one page path) exit `2`.
- **Contract C18** (and C6's `#CRD` copy and source table) document both layouts.

Release class: MINOR (`feat`). Additive: no field removed, renamed or tightened.

Scope: two sections: the layout end to end, then the contract and the archive.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `crd-extractor`: the crd source renders either one page or a section of kind pages.

## Impact

- Code: `schema/config.cue`, `internal/extract/crd`, `internal/build/crd.go`, `internal/render/crd.go` and `templates/crd/`, `docs/contracts.md` C6 and C18, `README.md`.
- Consumers: **opm-operator** moves its `crd` source from `page: "reference/operator-resources.md"` to `section: "reference/operator/"` after it pins the docs-kit release carrying this change: `owns` becomes `["reference/operator/"]`, its authored `docs/site/reference/operator-resources.md` moves to `docs/site/reference/operator/_index.md` (it completes the section index), and its own links to `/docs/reference/operator-resources/#<kind>` become `/docs/reference/operator/<kind>/`. **opmodel.dev** needs nothing from docs-kit: it reads the operator's bundle pages as written; the pinned `opm-docs` the site runs for `pull` does not render. Links on site pages to the old URL are the site's to update.
- Risk: none for existing callers; `page` mode keeps its output byte for byte (the crdref parity test still runs on it).
