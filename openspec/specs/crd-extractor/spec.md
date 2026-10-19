# crd-extractor Specification

## Purpose
How `opm-docs` turns controller-gen CustomResourceDefinitions and their kubebuilder samples into the doc model in `data/crd.json` and either one completable reference page or a section of one page per kind (docs-kit C18).

## Requirements

### Requirement: The crd source writes a model of each kind
A `crd` source SHALL read every `CustomResourceDefinition` in the YAML files of its `dir` and write `data/crd.json` with schema `docs.opmodel.dev/data/crd/v1`: per kind its group, served versions with the storage version marked, scope, plural, short names, subresources and printer columns, and its `reconciledBy` reconciler when configured. Kinds SHALL be ordered by the configured `order`, then by kind name.

#### Scenario: Kinds in order
- **WHEN** a `crd` source reads two kinds and `order` names neither
- **THEN** `data/crd.json` lists the kinds by kind name

### Requirement: The crd model holds the field tables and rules of each kind
The `crd` source SHALL write `data/crd.json` with schema `docs.opmodel.dev/data/crd/v1`: per kind the spec and status field tables (path, type, required, description cleaned by the doc-comment rules and the citation policy, default, enum) and the validation rules with what enforces them.

#### Scenario: A required spec field
- **WHEN** `ModuleInstance`'s schema lists `module` under `spec.required`
- **THEN** the model's spec table marks `module` required

### Requirement: The crd source reads the sample of each kind
When `samples` is set, the `crd` source SHALL read the sample of each kind there: the first document of that kind in the file kubebuilder names `<group>_<version>_<kind>.yaml` (other files are not read), shown only when it contains no `hideSamplesMatching` string, with each `stripLabels` label of a matching value removed. It SHALL write the sample text to `data/crd.json`.

#### Scenario: A second sample file of one kind
- **WHEN** `config/samples` holds `opmodel.dev_v1alpha1_moduleinstance.yaml` and `opmodel.dev_v1alpha1_moduleinstance_jellyfin.yaml`
- **THEN** the ModuleInstance sample comes from the first file only

#### Scenario: A fixture sample is hidden
- **WHEN** the kind's sample references `testing.opmodel.dev` and `hideSamplesMatching` is `["testing.opmodel.dev"]`
- **THEN** that kind's entry has no `### Example`

#### Scenario: Scaffold labels stripped
- **WHEN** the sample carries `app.kubernetes.io/managed-by: kustomize` and `stripLabels` names that label and value
- **THEN** the shown sample has no such label

### Requirement: The crd page holds one entry per kind
In the page layout (the source sets `page`, not `section`), the renderer SHALL write one page at the source's `page` path, one of the bundle's owned paths: per kind a `## <Kind>` entry with `### At a glance`, the printer-column table, `### Spec`, `### Status`, `### Example` when a sample exists, `### Notes`, `### Served by` when `reconciledBy` names a reconciler, and `### Enforcement`, in crdref's order and wording. The one exception is the `### Served by` sentence.

#### Scenario: An entry of the page
- **WHEN** a `crd` source sets `page` and a kind has a sample
- **THEN** the page holds a `## <Kind>` entry with `### At a glance`, the printer-column table, `### Spec`, `### Status`, `### Example`, `### Notes` and `### Enforcement`, in crdref's order

### Requirement: The crd page applies the citation policy to descriptions
A field description on the crd page SHALL follow the source's `citations` policy: under `link` an enhancement reference becomes a link to its entry and a decision citation is removed without a link.

#### Scenario: Citation linked
- **WHEN** a field description names `enhancement 0021` and the source has `citations: "link"`
- **THEN** the description renders `[enhancement 0021](/enhancements/0021/)`

#### Scenario: Decision citation removed
- **WHEN** a field description ends with a decision citation (an id, a colon and a decision number) and the source has `citations: "link"`
- **THEN** the description renders without the citation, and without a link

### Requirement: The served-by sentence names no product
The `### Served by` sentence SHALL name the reconciler without naming a product: "The `<name>` reconciler watches every <Kind>." Apart from that sentence, the entries SHALL match crdref's generated block for the same tree.

#### Scenario: Parity with crdref
- **WHEN** opm-operator's tree at its newest tag is built with the planned config
- **THEN** the generated entries equal crdref's block between its markers, once crdref's "The operator's `<name>` controller watches every <Kind>." sentences read "The `<name>` reconciler watches every <Kind>."

#### Scenario: The served-by sentence names no product
- **WHEN** `reconciledBy` maps `Widget` to `widget`
- **THEN** the Widget entry's `Served by` part reads "The `widget` reconciler watches every Widget." and names no product

### Requirement: The crd page is completable
The page SHALL be completable, so an authored page with the same path from a `markdown` source supplies the front matter and intro; without one, the page takes the configured `title`, `description` and `weight`.

#### Scenario: Authored intro kept
- **WHEN** a `markdown` source over `docs/site` supplies `reference/operator-resources.md` holding front matter and an intro paragraph and no `## ModuleInstance` heading
- **THEN** the bundle's page is that front matter and intro followed by the four generated entries

#### Scenario: No authored page
- **WHEN** no `markdown` source supplies the page at the source's `page` path
- **THEN** the page takes the configured `title`, `description` and `weight`

### Requirement: The crd source refuses what its page cannot show
The site renders raw HTML, so the crd source SHALL treat every CRD and sample string as untrusted. It SHALL refuse, exiting `2` and naming the file, a field it does not read (strict decoding), a construct its page cannot show, a kind name not matching `^[A-Z][A-Za-z0-9]*$`, a scope other than `Namespaced` or `Cluster`, a printer-column type outside `integer`, `number`, `string`, `boolean` and `date`, and a CRD or sample file that is not a regular file.

#### Scenario: An unread field is refused
- **WHEN** a CRD carries `spec.conversion`
- **THEN** the build exits `2` naming the file and the unknown field

### Requirement: The crd renderer escapes prose and links only enhancement references
The renderer SHALL escape all prose, and SHALL write as links only the enhancement references it finds under the `link` policy.

#### Scenario: A description cannot forge a link
- **WHEN** a field description holds `[enhancement 0021](/enhancements/0099/)` and `[x](javascript:alert(1))`
- **THEN** both stay text with their brackets escaped, and only a bare `enhancement 0021` becomes a link to its own enhancement

### Requirement: The crd renderer fences a sample with a longer fence
The renderer SHALL fence a sample with more backticks than any backtick or tilde run in it.

#### Scenario: A sample cannot close its fence
- **WHEN** a shown sample holds a block scalar with a "```" line followed by a `<script>` line
- **THEN** the example's fence is longer than three backticks and the `<script>` line stays inside it

### Requirement: A crd source takes exactly one of page and section
A `crd` source SHALL take exactly one of `page` and `section`; a source with both or neither SHALL fail config validation (exit `1`). With `page`, the source SHALL render the single completable page, unchanged.

#### Scenario: Both page and section
- **WHEN** a `crd` source sets both `page` and `section`
- **THEN** loading the config exits `1` saying a crd source takes exactly one of page and section

#### Scenario: Page layout unchanged
- **WHEN** a `crd` source sets `page`
- **THEN** the page's generated body still equals crdref's block for the operator's tree

### Requirement: The crd section writes an index and one page per kind
With `section`, the renderer SHALL write `<section>_index.md` and one page per kind at `<section><kind lower-cased>.md`. Two kinds whose lower-cased names are equal SHALL exit `2` naming both.

#### Scenario: Operator section
- **WHEN** opm-operator's CRDs are built with `section: "reference/operator/"`
- **THEN** the bundle holds `reference/operator/_index.md`, `moduleinstance.md`, `modulepackage.md`, `platform.md` and `transformerregistration.md`, with weights 1 to 4 in the configured order

#### Scenario: Kinds that differ only in case
- **WHEN** two kinds have lower-cased names that are equal
- **THEN** the build exits `2` naming both

### Requirement: The crd section index lists the kinds
The index SHALL carry the configured `title`, `description` and `weight`, an intro, and a `## Kinds` table linking every kind page in model order, and SHALL be completable with heading `## Kinds`.

#### Scenario: Authored index completed
- **WHEN** a `markdown` source supplies `reference/operator/_index.md` holding front matter and an intro and no `## Kinds` heading
- **THEN** the bundle's index is that front matter and intro followed by the generated `## Kinds` table

#### Scenario: The index links each kind page
- **WHEN** the section index is linted in bundle mode
- **THEN** every link in its `## Kinds` table names a page of the bundle

### Requirement: A kind page holds its entry one level up
A kind page SHALL be titled by the kind, weighted by its 1-based position in model order, typed `reference`, and SHALL hold the kind's entry without its `## <Kind>` heading and with its parts one level up (`## Spec`).

#### Scenario: A kind page of the section
- **WHEN** the section holds the kind `Platform` at position 3 in model order
- **THEN** its page is titled `Platform`, weighted 3, typed `reference`, and holds `## Spec` rather than `### Spec`

### Requirement: The crd model records the layout
`data/crd.json` SHALL record `layout` (`"page"` or `"section"`) and each kind's `page` path (`null` in page layout).

#### Scenario: The layout in the model
- **WHEN** a `crd` source sets `page`
- **THEN** `data/crd.json` records `layout` `"page"` and `null` as each kind's `page` path
