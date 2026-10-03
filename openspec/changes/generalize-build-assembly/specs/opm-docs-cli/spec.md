## MODIFIED Requirements

### Requirement: docs-kit.cue configures the bundles a repository builds
`build` and `check` SHALL read `docs-kit.cue` (or `--config`) and validate it against the embedded `#Config` schema before any extraction. A `package` clause in the file SHALL be optional and ignored. The file SHALL hold `bundles`, keyed by project, each with a `placement`, a `version` (`from: "tag"` and a tag `prefix`), at least one source of a kind the tool registers (`docs/contracts.md` C6 lists them, each with its options), and optionally `pins`. Every extractor source SHALL accept `citations` (`"strip"`, the default, or `"link"`); a `markdown` source SHALL accept `include`, a list of globs relative to its `dir`. A bundle SHALL hold at most one source of each extractor kind.

#### Scenario: Package clause ignored
- **WHEN** one `docs-kit.cue` starts with `package docs` and another has no package clause, with the same fields
- **THEN** both validate and configure the same bundles

#### Scenario: A misspelled key is refused
- **WHEN** `docs-kit.cue` holds `bundles: "catalog-opm": {placment: ...}`
- **THEN** `opm-docs build` exits 1 naming the field `placment` and the file, before loading any CUE module

#### Scenario: An unknown source kind
- **WHEN** a source has `kind: "javadoc"`
- **THEN** `build` exits 1 naming the kind and the kinds this `opm-docs` registers

#### Scenario: Include selects one page
- **WHEN** a `markdown` source has `dir: "docs/site"` and `include: ["reference/operator-resources.md"]`
- **THEN** only that page is copied from `docs/site`

#### Scenario: Citations linked
- **WHEN** a source with `citations: "link"` documents a comment citing `0010:D28`
- **THEN** the page holds `[0010:D28](/enhancements/0010/decisions/)`
