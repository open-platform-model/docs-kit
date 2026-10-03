## Why

Phase 2 (DESIGN.md) moves every generated reference into bundles: core's definitions, the operator's resources, the cli's commands and, for the first time, the library's Go API, placed in a site version's `/docs/` tree. `opm-docs` is catalog-shaped today: `internal/build` knows two source kinds and calls `render.Catalog` directly, `#Source` is a closed union of `cue-catalog` and `markdown`, a docs placement is accepted by the manifest schema but nothing builds or lints one, and a build cannot run repository code (the cobra hook) or record what a cli pins (DESIGN decision 10). This change builds the shared ground the four extractor changes and `pull-docs-placement` stand on, so those five can then be built in parallel.

Gate: none in docs-kit. It is the first phase-2 change; `docs/orchestration.md` sequences the rest.

## What Changes

- **Source and renderer registry.** Each source kind is an extractor that writes one data file (`data/<kind>.json`, its own schema id) and names the renderer of that schema; the renderer turns the data file into pages. `cue-catalog` moves behind the registry with byte-identical output (its goldens and parity test unchanged).
- **Docs placement in build.** `placement: {kind: "docs", root: "/docs/", owns: [...]}`: the directories or pages a bundle owns exclusively. Generated pages of a docs bundle must lie under what it owns; bundle-mode lint for a docs bundle applies docs-mode link rules and requires every link into an owned path to name a page of the bundle.
- **Completable pages, generalized.** A renderer may mark a page completable (the catalog landing is the first): an authored page at the same path from a `markdown` source supplies front matter and intro, and the generated body follows it. The operator's resource page needs this.
- **`markdown` `include`.** Globs selecting which pages of `dir` a bundle copies, so a phase-2 bundle can take one authored page (the operator's intro) while the rest of `docs/site/` still reaches the site through git until phase 3.
- **Per-source citation policy.** `citations: "strip"` (today's behavior, the default) or `"link"` (an enhancement decision citation becomes a link to `/enhancements/<NNNN>/decisions/`), so the operator keeps the links its reference has today.
- **Repository commands.** A config value that runs a repository program (argv, no shell) and reads one JSON document from its stdout, with fixed rules for directory, environment, timeout and determinism (C14). The `cobra` extractor and pins use it.
- **Pins.** `pins: {command, projects}` in a bundle's config writes `manifest.json` `pins` (project to exact version), which `pull` uses to choose the docs of what a cli release pins (DESIGN decision 10).
- **`publish.yml`.** The build runs in its own job with no `id-token` and only `packages: read`, and hands the bundle tree to the publishing job, which runs no repository code; a `setup-go` input installs Go for repositories whose commands need it.
- **Contracts.** C1 reserves every phase-2 and phase-3 project name; C3 gains `placement.owns` and `pins`; C6 gains the source registry, `include`, `citations` and `pins`; new C14 (repository commands) and C15 (docs placement).

Release class: MINOR (`0.4.0`, or the next minor at merge). Additive: new optional manifest fields, a widened placement; a tab bundle built by this release is byte-identical to one built by the previous.

Scope: five sections, the last carrying the contracts and the archive.

## Capabilities

### New Capabilities

- `repository-commands`: how a build runs a repository program and reads its output.
- `docs-placement`: building and linting a bundle placed in a site version's `/docs/`, and the pins it carries.

### Modified Capabilities

- `opm-docs-cli`: `docs-kit.cue` takes the source kinds the tool registers, `include`, `citations` and `pins`.
- `page-renderer`: renderers are registered per data schema; completable pages.
- `publish-workflow`: build and publish run in separate jobs; the `setup-go` input.

## Impact

- Code: `internal/build` (registry, docs placement, completable pages), `internal/render` (registry, docs targets), `internal/extract/markdown` (`include`), `internal/doctext` (citation links), new `internal/command`, `schema/config.cue`, `schema/manifest.cue`, `internal/bundle`, `internal/dialect` (docs bundle mode), `.github/workflows/publish.yml`, `docs/contracts.md`, `README.md`.
- Consumers: **catalog_opm** nothing (its tab bundle is byte-identical; it may bump at leisure). **opmodel.dev** nothing until `pull-docs-placement`. **core, opm-operator, cli, library** gain the ground their phase-2 sibling changes build on (`docs/orchestration.md`); none acts on this release alone. The `publish.yml` job split changes no input or output: a caller moves its pin when it next bumps.
- Risk: a repository command runs the caller's own code during its docs build; the job split keeps it away from the signing credential.
