## ADDED Requirements

### Requirement: Renderers are registered per data schema
Each source kind SHALL write one data file under `data/` with its own schema id, and the renderer registered for that schema SHALL render it, reading the data file as written and never the source. A `cue-catalog` bundle built through the registry SHALL be byte-identical to one built before it.

#### Scenario: Catalog output unchanged
- **WHEN** the catalog fixture is built before and after the registry
- **THEN** both `out/` trees, and the packed digests, are identical

### Requirement: Completable pages take an authored page first
When a renderer marks a page completable and a `markdown` source of the same bundle supplies a page at the same path, the bundle's page SHALL be the authored front matter and body, one blank line, then the generated body without its own front matter, recorded as `generated: false` with the authored file as `source`. An authored body holding the generated body's first heading SHALL fail the build with exit 2 naming the file. Without an authored page, the generated page SHALL stand alone with generated front matter.

#### Scenario: The operator's resource page
- **WHEN** the `crd` source renders `reference/operator-resources.md` and a `markdown` source over `docs/site` supplies `reference/operator-resources.md`
- **THEN** the page holds the authored front matter and intro followed by the generated resource entries, and `manifest.json` records it with `source` `docs/site/reference/operator-resources.md`
