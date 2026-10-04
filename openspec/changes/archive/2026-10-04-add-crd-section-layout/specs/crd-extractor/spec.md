## ADDED Requirements

### Requirement: The crd source renders a section of kind pages
A `crd` source SHALL take exactly one of `page` and `section`; a source with both or neither SHALL fail config validation (exit `1`). With `page`, the source SHALL render the single completable page, unchanged. With `section`, the renderer SHALL write `<section>_index.md` and one page per kind at `<section><kind lower-cased>.md`. The index SHALL carry the configured `title`, `description` and `weight`, an intro, and a `## Kinds` table linking every kind page in model order, and SHALL be completable with heading `## Kinds`. A kind page SHALL be titled by the kind, weighted by its 1-based position in model order, typed `reference`, and SHALL hold the kind's entry without its `## <Kind>` heading and with its parts one level up (`## Spec`). `data/crd.json` SHALL record `layout` (`"page"` or `"section"`) and each kind's `page` path (`null` in page layout). Two kinds whose lower-cased names are equal SHALL exit `2` naming both.

#### Scenario: Operator section
- **WHEN** opm-operator's CRDs are built with `section: "reference/operator/"`
- **THEN** the bundle holds `reference/operator/_index.md`, `moduleinstance.md`, `modulepackage.md`, `platform.md` and `transformerregistration.md`, with weights 1 to 4 in the configured order

#### Scenario: Authored index completed
- **WHEN** a `markdown` source supplies `reference/operator/_index.md` holding front matter and an intro and no `## Kinds` heading
- **THEN** the bundle's index is that front matter and intro followed by the generated `## Kinds` table

#### Scenario: The index links each kind page
- **WHEN** the section index is linted in bundle mode
- **THEN** every link in its `## Kinds` table names a page of the bundle

#### Scenario: Both page and section
- **WHEN** a `crd` source sets both `page` and `section`
- **THEN** loading the config exits `1` saying a crd source takes exactly one of page and section

#### Scenario: Page layout unchanged
- **WHEN** a `crd` source sets `page`
- **THEN** the page's generated body still equals crdref's block for the operator's tree
