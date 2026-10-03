## Why

opmodel.dev renders raw HTML (`markup.goldmark.renderer.unsafe = true`). docs-kit#36 added the markup check (`internal/mdsafe`: goldmark v1.8.6 as Hugo 0.167.0 parses a page, after Hugo's `DedentMarkers` step) and runs it on section bundles only. Every other page reaches the site guarded by its extractor's escaping alone, which C11, C21 and "Doc-comment rules" call best effort: docs-kit#27 showed a `<script>` from a CRD sample and an event handler from a CRD `scope` passing `build` and the lint before #24 escaped them, and docs-kit#37 showed a heading attribute block (`{.hx:fixed style="..."}`, `{id=kernel}`) passing through a heading built from source text. There is no backstop for the generated pages of `cue-catalog`, `cue-definitions`, `crd`, `cobra` and `go-api`, nor for the authored pages a `markdown` source copies.

## What Changes

- **Bundle-mode lint runs the markup check on every page of every bundle** (tab, docs and section), so `build`, `lint --bundle` and `pull` apply it to every page they touch, including bundles already published.
- **Two rule sets, chosen per page from `manifest.json`**:
  - `Generated` (every page with `generated: true`, and every page of a section bundle): the full check, as section pages have it today: raw HTML of any kind refused, comments included; destinations with a scheme other than `http`, `https`, `mailto` refused, and protocol-relative ones; heading attribute blocks refused; the `{{__hugo_ctx` marker refused anywhere.
  - `Authored` (`generated: false` in a tab or docs bundle, the pages a `markdown` source copies as written, and a completed page): the same rules, except that a raw HTML node made only of HTML comments, each of which a browser closes exactly where goldmark does, is allowed. Anything else in raw HTML, a comment a browser closes early (`--!>`, `<!-->`, `<!--->`) or text after the comment's `-->`, is refused.
- **Heading attribute blocks in renderers (docs-kit#37)**: `cue-catalog` prose (notes and summaries in tables) escapes `{` and `}` as `go-api` already does. `crd` and `cobra` already escape `#` in prose and build headings only from validated identifiers; `cue-definitions` prose is Markdown as written by contract (C17), so a doc comment that writes a heading attribute block fails the build, naming the page and line.
- **Test nits from the #36 review**: a probe whose page Hugo fails to build fails `TestProbesWithHugo` unless an expected-to-fail list names it (empty today); the live-markup detector reads the rendered HTML with an HTML5 tokenizer and flags any element goldmark does not emit, any `on*` attribute and any `href`/`src` not starting `http:`, `https:`, `mailto:`, `#` or `/` (not `//`); new probes for browser-early comment closes; the probes also render through the pinned Hextra theme and opmodel.dev's `_markup` hooks when an opmodel.dev `site/` checkout is named.
- `docs/contracts.md` C11, C21, "Doc-comment rules" and "Commands" (`build`, `lint`, `pull`) state the check as the guarantee for every bundle.

SemVer class: MINOR (`feat`, `0.6.0`). No schema, tag, workflow or config change. A page that passes today and fails after the change would need raw HTML other than comments, a script URL or a heading attribute block; the measurement in design.md found none in the six producing repositories or in any bundle published on GHCR.

## Consumers

- opmodel.dev: nothing required. The check runs in bundle mode only, which the site's shell lint does not mirror, so no conformance fixture changes (C11 re-sync rule does not trigger). When it moves its pinned `opm-docs` to the release, `pull` applies the check to every bundle it pulls; every bundle published today passes (design.md).
- catalog_opm, core, cli, opm-operator, library, opm: nothing required; their authored pages carry only HTML comments that pass the `Authored` rules. A future page with raw HTML other than a comment fails their `build` naming the page and line.
- enhancements: nothing; its section bundle keeps the full check.

## Capabilities

### Modified Capabilities

- `dialect-lint`: bundle mode checks the markup of every page, by rule set.
- `page-renderer`: rendered prose escapes `{` and `}`.

## Impact

- Code: `internal/mdsafe` (the `Authored` comment rule, tests), `internal/build` (`Lint` runs the check on every page), `internal/mdtext` (brace escaping), tests in `internal/render` and `internal/build`.
- Risk: a published bundle that fails the check stops pulling. Every bundle on GHCR was pulled and checked with the new binary before merge (tasks.md, section 4).
