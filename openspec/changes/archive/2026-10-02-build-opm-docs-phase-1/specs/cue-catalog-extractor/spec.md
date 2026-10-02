## Purpose

How `opm-docs` turns an evaluated CUE catalog module into the doc model in `data/catalog.json`, including the structured spec and the doc-comment cleanup.

## ADDED Requirements

### Requirement: Members come from the evaluated catalog
The `cue-catalog` extractor SHALL load the module at `module` with the CUE Go API (registry from `CUE_REGISTRY`), read `metadata.modulePath` and `metadata.version`, and enumerate members from the catalog's `#resources`, `#traits` and `#blueprints` maps and transformers from `#transformers`, sorted by key. It SHALL refuse a duplicate FQN, a member whose `spec` does not hold exactly one field, a trait whose `optional` has no default, and a blank `metadata.description`, naming the member.

#### Scenario: Catalog fixture enumerates
- **WHEN** the extractor runs on the test fixture catalog with two resources, one trait, one blueprint and two transformers
- **THEN** `data/catalog.json` lists four members and two transformers in key order

### Requirement: The doc model is written to data/catalog.json
The extractor SHALL write `data/catalog.json` with schema id `docs.opmodel.dev/data/cue-catalog/v1` and the fields `docs/contracts.md` C10 lists for the catalog, each member and each transformer. The renderer SHALL read only this file and `manifest.json`.

#### Scenario: A member's facts are present
- **WHEN** the extractor runs on catalog_opm at `opm-v4.4.5`
- **THEN** the `backup` trait's entry has `apiVersion` `v1alpha1`, `fulfilment` `provider`, `optional` false, `mark` `provided-by-platform` and an empty `servedBy`

### Requirement: The spec is extracted as structured fields beside its text
For each member the extractor SHALL record `spec.cue`, the formatted authored spec block plus the in-module definitions it references (another member's spec root linked, not repeated; out-of-module and vendored packages named only), and `spec.fields`, walked from the evaluated value: every field under the spec key with its dot path (`[]` for a list element, `[string]` for a pattern constraint), formatted type, `presence` (`regular`, `optional` or `required`), formatted default or null, cleaned doc, and `ref` naming a definition outside the member's package where the walk stops. The walk SHALL stop at the module boundary and at a definition already visited, and SHALL not exceed depth 12.

#### Scenario: Required and defaulted fields
- **WHEN** a fixture member's spec declares `name!: string` and `replicas: int | *1`
- **THEN** `spec.fields` holds `{path: "name", presence: "required", default: null}` and `{path: "replicas", presence: "regular", default: "1"}`

#### Scenario: A vendored type stops the walk
- **WHEN** a spec field's value is a definition from the module's vendored Kubernetes schemas
- **THEN** that field's entry carries `ref` with the definition name and no child fields are listed under it

### Requirement: Doc comments are cleaned before they reach the model
The extractor SHALL drop maintainer comments (a comment group starting with `WHY` or a `////` banner, and a `WHY` line inside a doc comment), strip enhancement citations and `SPEC.md` and experiment references by the rules `docs/contracts.md` "Doc-comment rules" lists, re-wrap a changed comment paragraph in a spec block, and require each member's doc comment to open with its `metadata.description` followed by a period. The remaining paragraphs SHALL be the member's `notes`.

#### Scenario: Citation removed from prose
- **WHEN** a doc paragraph reads `Exactly one provider serves it (0010:D32).`
- **THEN** the note reads `Exactly one provider serves it.`

#### Scenario: WHY block dropped
- **WHEN** a definition holds a comment group starting `// WHY: 0010 D28 ...`
- **THEN** no text of that group appears in `notes`, `spec.cue` or any field's `doc`

### Requirement: Served-by, marks and enforcement are derived
The extractor SHALL compute `servedBy` from the transformers of the same catalog (required, else optional, by FQN; a blueprint by its required labels and composed members), `mark` (`provided-by-platform` or `not-implemented` for an unserved resource or trait by its `fulfilment`; never for a blueprint) and `enforcement` (the spec schema and each required match label by `cue`; a provider-fulfilled contract's single provider and a load-bearing trait's refused render by `kernel`; nothing else).

#### Scenario: Unserved catalog-fulfilled resource
- **WHEN** no transformer in the catalog requires or optionally reads a resource whose `fulfilment` is `catalog`
- **THEN** its `mark` is `not-implemented` and its `servedBy` is empty

### Requirement: The structured spec is ordered and formatted stably
`spec.fields` SHALL list siblings in the order CUE's `Fields` iterator yields them for the evaluated value, each field followed by its descendants, and SHALL print `type` and `default` with `cue/format` (simplified) on one line with single spaces, so the same source and tool version always produce the same list.

#### Scenario: Two runs agree
- **WHEN** the extractor runs twice on the same catalog with the same binary
- **THEN** both `data/catalog.json` files are byte-identical
