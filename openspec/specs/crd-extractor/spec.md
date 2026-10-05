# crd-extractor Specification

## Purpose
How `opm-docs` turns controller-gen CustomResourceDefinitions and their kubebuilder samples into the doc model in `data/crd.json` and either one completable reference page or a section of one page per kind (docs-kit C18).

## Requirements

### Requirement: The crd source reads controller-gen CRDs and samples
A `crd` source SHALL read every `CustomResourceDefinition` in the YAML files of its `dir`, and, when `samples` is set, the sample of each kind there: the first document of that kind in the file kubebuilder names `<group>_<version>_<kind>.yaml` (other files are not read), shown only when it contains no `hideSamplesMatching` string, with each `stripLabels` label of a matching value removed, and SHALL write `data/crd.json` with schema `docs.opmodel.dev/data/crd/v1`: per kind its group, served versions with the storage version marked, scope, plural, short names, subresources, printer columns, the spec and status field tables (path, type, required, description cleaned by the doc-comment rules and the citation policy, default, enum), the validation rules with what enforces them, the sample text, and its `reconciledBy` reconciler when configured. Kinds SHALL be ordered by the configured `order`, then by kind name.

#### Scenario: A required spec field
- **WHEN** `ModuleInstance`'s schema lists `module` under `spec.required`
- **THEN** the model's spec table marks `module` required

#### Scenario: A second sample file of one kind
- **WHEN** `config/samples` holds `opmodel.dev_v1alpha1_moduleinstance.yaml` and `opmodel.dev_v1alpha1_moduleinstance_jellyfin.yaml`
- **THEN** the ModuleInstance sample comes from the first file only

#### Scenario: A fixture sample is hidden
- **WHEN** the kind's sample references `testing.opmodel.dev` and `hideSamplesMatching` is `["testing.opmodel.dev"]`
- **THEN** that kind's entry has no `### Example`

#### Scenario: Scaffold labels stripped
- **WHEN** the sample carries `app.kubernetes.io/managed-by: kustomize` and `stripLabels` names that label and value
- **THEN** the shown sample has no such label

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

### Requirement: CRD text never reaches the page as markup
The site renders raw HTML, so the crd source SHALL treat every CRD and sample string as untrusted. It SHALL refuse, exiting `2` and naming the file, a field it does not read (strict decoding), a construct its page cannot show, a kind name not matching `^[A-Z][A-Za-z0-9]*$`, a scope other than `Namespaced` or `Cluster`, a printer-column type outside `integer`, `number`, `string`, `boolean` and `date`, and a CRD or sample file that is not a regular file. The renderer SHALL escape all prose, SHALL write as links only the decision citations it finds under the `link` policy, and SHALL fence a sample with more backticks than any backtick or tilde run in it.

#### Scenario: A sample cannot close its fence
- **WHEN** a shown sample holds a block scalar with a "```" line followed by a `<script>` line
- **THEN** the example's fence is longer than three backticks and the `<script>` line stays inside it

#### Scenario: A description cannot forge a link
- **WHEN** a field description holds `[0021:D4](/enhancements/0099/decisions/)` and `[x](javascript:alert(1))`
- **THEN** both stay text with their brackets escaped, and only a bare citation becomes a link to its own enhancement

#### Scenario: An unread field is refused
- **WHEN** a CRD carries `spec.conversion`
- **THEN** the build exits `2` naming the file and the unknown field

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
