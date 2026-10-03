## Context

`internal/mdsafe.Check` parses a page as opmodel.dev's Hugo 0.167.0 does (goldmark v1.8.6 with Hugo's default extensions, after Hugo's `DedentMarkers` step) and refuses raw HTML, a destination with a scheme other than `http`, `https` or `mailto` or a protocol-relative one, a heading attribute block and Hugo's `{{__hugo_ctx` marker. `build.Lint`, the bundle-mode lint behind `build`, `lint --bundle` and `pull`, runs it only when `placement.kind` is `section`. Its `Mode` argument (`Authored`, `Generated`) is accepted and ignored.

Every bundle page is listed in `manifest.json` with `generated` (C3): `true` for a page a renderer wrote, `false` for a page a `markdown` source copied as written and for a completed page (C15: an authored page with a renderer's tail).

## Goals / Non-Goals

**Goals:** run the check on every page of every bundle in bundle mode; fix the rules for authored pages from what the repositories hold today; keep every published bundle pulling; close the #36 review's test nits; close docs-kit#37 for every renderer.

**Non-Goals:** running the check in docs mode (`opm-docs lint <dir>` over a `docs/site` tree), which the site's shell lint would then have to mirror (C11); stripping HTML comments from authored pages (they are copied as written, C15); turning off heading attributes in the site's Hugo config.

## Decisions

### D1. Bundle-mode lint checks every page, rule set by page

`build.Lint(dir)` SHALL run `mdsafe.Check` on every page `manifest.json` lists, after the dialect lint, with:

| Page | Mode |
|---|---|
| any page of a `section` bundle | `Generated` |
| `generated: true` | `Generated` |
| `generated: false` in a `tab` or `docs` bundle | `Authored` |

Violations print as today, `<content path>:<line>: <message>`, and fail `build` and `pull` with exit `2`. `lint --bundle` prints them and exits `2`. Nothing changes in docs mode.

A completed page is `generated: false`, so it takes `Authored` rules. Its tail is a renderer's text, which escapes `<`, so the tail cannot hold a comment; the allowance reaches only the authored part.

### D2. The `Authored` rule: HTML comments a browser closes where goldmark does

`Generated` keeps every rule of the check. `Authored` keeps every rule except one: a raw HTML node (an inline `RawHTML` or an `HTMLBlock`, its closure line included) is allowed when its text is a sequence of HTML comments separated by whitespace, each read as an HTML5 tokenizer reads it:

- it opens `<!--`;
- it is not closed abruptly: the text after `<!--` does not start with `>` or `->` (`<!-->`, `<!--->`);
- it ends at the first `-->` or `--!>` after `<!--`, whichever comes first, and that end exists in the node.

Anything else in the node (text after the last comment, `<!-- x --!> <svg onload=...> -->`, an unclosed comment, any element) refuses the whole node with:

```text
content/how-to/x.md:12: raw HTML other than an HTML comment a browser closes where goldmark does; the site would render it, so write it as Markdown or in a code span
```

(`an HTML block other than ...` for a block). A node goldmark reads as a comment but a browser closes earlier is exactly the case where the allowance would put live markup on the page, so it is refused, not trusted.

The comments themselves reach the published HTML (goldmark's unsafe renderer writes them out), so they are visible in the page source. That is a content choice for each repository, not a safety one, and stays as it is (Non-Goals).

### D3. Braces in rendered prose (docs-kit#37)

`mdtext.Text`, the prose escaper `cue-catalog` uses for notes and table descriptions, SHALL escape `{` and `}` as `\{` and `\}` outside code spans (this subsumes the `{{` rule: `{{<` becomes `\{\{\<`). Goldmark renders `\{` as `{`, so the page reads the same. Per renderer:

| Renderer | Heading built from source | Prose can open a heading | Attribute block possible before | After |
|---|---|---|---|---|
| `cue-catalog` | no (member names are CUE identifiers) | yes (`#` not escaped) | yes, in a note | no: braces escaped |
| `cue-definitions` | definition names (identifiers) | yes, Markdown as written (C17) | yes | the build refuses it (exit `2`, page and line) |
| `crd` | kinds (`^[A-Z][A-Za-z0-9]*$`, checked) | no: `#` escaped | no | no |
| `cobra` | command paths (kebab-case, checked) | no: `#` escaped | no | no |
| `go-api` | escaped (C20) | escaped | no | no |
| `enhancements` | authored, section | yes | refused by the check (C21) | unchanged |

`cue-definitions` prose stays Markdown as written: escaping braces there would change what a core author writes on purpose (a code-like `{a: 1}` in prose reads the same either way, but a contract of "as written" is simpler than a list of exceptions). The check is the guarantee, and a heading attribute block in a doc comment now fails core's build instead of styling the site.

### D4. Test changes in `internal/mdsafe`

- `TestProbesWithHugo` fails when Hugo refuses to build a probe's page, unless the probe is in `hugoBuildFails` (empty: all 33 probes build with Hugo 0.167.0).
- The live-markup detector (`active`) tokenizes the rendered HTML with `golang.org/x/net/html` (already in the module graph) and reports: a start tag outside the set goldmark and Hugo's heading renderer emit (`p h1-h6 a em strong code pre ul ol li blockquote hr br img table thead tbody tr th td del input dl dt dd sup div section`); any attribute whose name starts `on`; an `href`, `src`, `action`, `formaction` or `xlink:href` value not starting `http:`, `https:`, `mailto:`, `#` or `/` (or starting `//`), compared after trimming and lower-casing. A comment is a token of its own, so a comment holding `<svg onload=...>` is inert and one a browser closes early is not.
- Each probe runs under both modes; new probes cover the comment forms of D2 (`--!>`, `<!-->`, `<!--->`, text after `-->`, an unclosed comment, a well-formed comment).
- `TestProbesWithSite` renders each probe again through the pinned Hextra theme and opmodel.dev's `_markup` hooks, when `OPM_DOCS_SITE_DIR` names an opmodel.dev `site/` directory; it skips otherwise. The site's link hook fails the build on an internal link it cannot resolve; such a probe counts as refused when the check refuses it too, and otherwise fails the test.

## Research & Decisions

### What authored pages hold today

**Context**: the `Authored` rules had to keep every producing repository building.
**Explored**: `mdsafe.Check` (strict) over every `.md` under `docs/site` of catalog_opm, opm, core, cli, library and opm-operator, and catalog_opm's `docs/catalogs/opm`, at `origin/main` on 2026-10-03 (catalog_opm f47ccd0, opm 4d97186, core b9f047b, cli 9febef5f, library 130b2d1, opm-operator f41b54f): 67 pages. Every node was classified by goldmark node type.
**Findings**: 609 violations on 57 pages, all raw HTML, all HTML comments: 474 HTML blocks of type 2 (comment) in all six repositories (writer's notes such as "Check against: ..."), and 135 inline comments in one page, opm-operator `docs/site/diagnostics/operator-conditions.md` (table cells annotated `<!-- ModuleInstance, ModulePackage -->`). No element, no unsafe scheme, no heading attribute block, no context marker.
**Options considered**:
1. Same rules as generated pages - breaks 57 pages in six repositories and every published docs bundle.
2. Skip the check on authored pages - leaves the gap #27 names for the pages most likely to be edited by hand.
3. Allow only HTML comments a browser closes where goldmark does - keeps every page building, refuses everything else.
**Decision**: option 3.
**Rationale**: the narrowest rule that keeps them building. A comment renders nothing; the only way a comment carries markup is a browser closing it earlier than goldmark, which D2 refuses.

### What published bundles hold

**Context**: `pull` now applies the check to bundles already on GHCR.
**Explored**: every non-signature tag of every docs package on GHCR on 2026-10-03, fetched anonymously: catalog-opm `4.5.1.0` and `edge`, core `2.0.0-beta.1.0` and `edge`, opm-operator `1.0.0-beta.4.0` and `edge`, cli `edge` (library, catalog-opm-docs, opm and enhancements have none).
**Findings**: every `generated: true` page passes the `Generated` rules. Authored pages hold HTML comments only (cli edge 54 blocks; core 191 and 164; opm-operator 34 blocks and 135 inline comments in each). Section 4 pulls them again with the built binary.

### Heading attributes off on the site instead

**Context**: #37 offered `markup.goldmark.parser.attribute.title = false` on the site.
**Options considered**:
1. Site config - one line, but leaves the check and Hugo disagreeing on what a heading is, and does not reach a theme or a later site.
2. Refuse in the check and escape in renderers - nothing changes on the site.
**Decision**: option 2. The site may still turn them off; the check stays.

## Risks / Trade-offs

- A producer adds raw HTML to an authored page: its `build` now fails, naming the page and line. That is the intent.
- A published bundle that fails can no longer be pulled, and its tab or site version fails the site build. None does today (above); a docs revision (C14) is the fix for a future one.
- `golang.org/x/net` moves from indirect to direct in `go.mod` for a test.

## Durable decisions

- D1 and D2: `docs/contracts.md` C11 (bundle mode), C21 (the check's rules and modes) and "Commands" (`build`, `lint`, `pull`); the `dialect-lint` main spec.
- D3: `docs/contracts.md` "Doc-comment rules" (Escaping) and C17 (a heading attribute block in a doc comment fails the build); the `page-renderer` main spec.
- D4: stays with the change and the test files.
