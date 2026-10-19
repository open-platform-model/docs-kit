## MODIFIED Requirements

### Requirement: The crd page is one completable page
In the page layout (the source sets `page`, not `section`), the renderer SHALL write one page at the source's `page` path, one of the bundle's owned paths: per kind a `## <Kind>` entry with `### At a glance`, the printer-column table, `### Spec`, `### Status`, `### Example` when a sample exists, `### Notes`, `### Served by` when `reconciledBy` names a reconciler, and `### Enforcement`, in crdref's order and wording. The one exception is the `### Served by` sentence, which SHALL name the reconciler without naming a product: "The `<name>` reconciler watches every <Kind>." The page SHALL be completable, so an authored page with the same path from a `markdown` source supplies the front matter and intro; without one, the page takes the configured `title`, `description` and `weight`. Apart from that sentence, the entries SHALL match crdref's generated block for the same tree.

#### Scenario: Authored intro kept
- **WHEN** a `markdown` source over `docs/site` supplies `reference/controller-resources.md` holding front matter and an intro paragraph and no `## ModuleInstance` heading
- **THEN** the bundle's page is that front matter and intro followed by the four generated entries

#### Scenario: Citation linked
- **WHEN** a field description names `enhancement 0021` and the source has `citations: "link"`
- **THEN** the description renders `[enhancement 0021](/enhancements/0021/)`

#### Scenario: Parity with crdref
- **WHEN** docs-kit's frozen copy of opm-controller's CRDs, samples and crdref page is built with the planned config
- **THEN** the generated entries equal crdref's block between its markers, once crdref's served-by sentences, which named the product, read "The `<name>` reconciler watches every <Kind>."

#### Scenario: The served-by sentence names no product
- **WHEN** `reconciledBy` maps `Widget` to `widget`
- **THEN** the Widget entry's `Served by` part reads "The `widget` reconciler watches every Widget." and names no product

## REMOVED Requirements

### Requirement: The crd source renders a section of kind pages
**Reason**: its first scenario names the controller by its old product name; OpenSpec cannot retitle a scenario in place.
**Migration**: replaced, with the same behaviour, by "The crd section layout writes an index and one page per kind", whose first scenario is "Controller section".

## ADDED Requirements

### Requirement: The crd section layout writes an index and one page per kind
A `crd` source SHALL take exactly one of `page` and `section`; a source with both or neither SHALL fail config validation (exit `1`). With `page`, the source SHALL render the single completable page, unchanged. With `section`, the renderer SHALL write `<section>_index.md` and one page per kind at `<section><kind lower-cased>.md`. The index SHALL carry the configured `title`, `description` and `weight`, an intro, and a `## Kinds` table linking every kind page in model order, and SHALL be completable with heading `## Kinds`. A kind page SHALL be titled by the kind, weighted by its 1-based position in model order, typed `reference`, and SHALL hold the kind's entry without its `## <Kind>` heading and with its parts one level up (`## Spec`). `data/crd.json` SHALL record `layout` (`"page"` or `"section"`) and each kind's `page` path (`null` in page layout). Two kinds whose lower-cased names are equal SHALL exit `2` naming both.

#### Scenario: Controller section
- **WHEN** opm-controller's CRDs are built with `section: "reference/controller/"`
- **THEN** the bundle holds `reference/controller/_index.md`, `moduleinstance.md`, `modulepackage.md`, `platform.md` and `transformerregistration.md`, with weights 1 to 4 in the configured order

#### Scenario: Authored index completed
- **WHEN** a `markdown` source supplies `reference/controller/_index.md` holding front matter and an intro and no `## Kinds` heading
- **THEN** the bundle's index is that front matter and intro followed by the generated `## Kinds` table

#### Scenario: The index links each kind page
- **WHEN** the section index is linted in bundle mode
- **THEN** every link in its `## Kinds` table names a page of the bundle

#### Scenario: Both page and section
- **WHEN** a `crd` source sets both `page` and `section`
- **THEN** loading the config exits `1` saying a crd source takes exactly one of page and section

#### Scenario: Page layout unchanged
- **WHEN** a `crd` source sets `page`
- **THEN** the page's generated body still equals crdref's block for the controller's tree, with the served-by exception the page requirement states
