## MODIFIED Requirements

### Requirement: The crd page is one completable page
In the page layout (the source sets `page`, not `section`), the renderer SHALL write one page at the source's `page` path, one of the bundle's owned paths: per kind a `## <Kind>` entry with `### At a glance`, the printer-column table, `### Spec`, `### Status`, `### Example` when a sample exists, `### Notes`, `### Served by` when `reconciledBy` names a reconciler, and `### Enforcement`, in crdref's order and wording. The one exception is the `### Served by` sentence, which SHALL name the reconciler without naming a product: "The `<name>` reconciler watches every <Kind>." The page SHALL be completable, so an authored page with the same path from a `markdown` source supplies the front matter and intro; without one, the page takes the configured `title`, `description` and `weight`. Apart from that sentence, the entries SHALL match crdref's generated block for the same tree.

#### Scenario: Authored intro kept
- **WHEN** a `markdown` source over `docs/site` supplies `reference/operator-resources.md` holding front matter and an intro paragraph and no `## ModuleInstance` heading
- **THEN** the bundle's page is that front matter and intro followed by the four generated entries

#### Scenario: Citation linked
- **WHEN** a field description cites `0021:D4` and the source has `citations: "link"`
- **THEN** the description renders `[0021:D4](/enhancements/0021/decisions/)`

#### Scenario: Parity with crdref
- **WHEN** opm-operator's tree at its newest tag is built with the planned config
- **THEN** the generated entries equal crdref's block between its markers, once crdref's "The operator's `<name>` controller watches every <Kind>." sentences read "The `<name>` reconciler watches every <Kind>."

#### Scenario: The served-by sentence names no product
- **WHEN** `reconciledBy` maps `Widget` to `widget`
- **THEN** the Widget entry's `Served by` part reads "The `widget` reconciler watches every Widget." and names no product
