# crd-extractor Specification

## Purpose
How `opm-docs` turns controller-gen CustomResourceDefinitions and their kubebuilder samples into the doc model in `data/crd.json` and one completable reference page (docs-kit C18).

## Requirements

### Requirement: The crd source reads controller-gen CRDs and samples
A `crd` source SHALL read every `CustomResourceDefinition` in the YAML files of its `dir`, and, when `samples` is set, the sample of each kind there: the first document of that kind in the file kubebuilder names `<group>_<version>_<kind>.yaml` (other files are not read), shown only when it contains no `hideSamplesMatching` string, with each `stripLabels` label of a matching value removed, and SHALL write `data/crd.json` with schema `docs.opmodel.dev/data/crd/v1`: per kind its group, served versions with the storage version marked, scope, plural, short names, subresources, printer columns, the spec and status field tables (path, type, required, description cleaned by the doc-comment rules and the citation policy, default, enum), the validation rules with what enforces them, the sample text, and its `reconciledBy` controller when configured. Kinds SHALL be ordered by the configured `order`, then by kind name.

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
The renderer SHALL write one page at the source's `page` path, one of the bundle's owned paths: per kind a `## <Kind>` entry with `### At a glance`, the printer-column table, `### Spec`, `### Status`, `### Example` when a sample exists, `### Notes`, `### Served by` when `reconciledBy` names a controller, and `### Enforcement`, in crdref's order and wording. The page SHALL be completable, so an authored page with the same path from a `markdown` source supplies the front matter and intro; without one, the page takes the configured `title`, `description` and `weight`. The entries SHALL match crdref's generated block for the same tree.

#### Scenario: Authored intro kept
- **WHEN** a `markdown` source over `docs/site` supplies `reference/operator-resources.md` holding front matter and an intro paragraph and no `## ModuleInstance` heading
- **THEN** the bundle's page is that front matter and intro followed by the four generated entries

#### Scenario: Citation linked
- **WHEN** a field description cites `0021:D4` and the source has `citations: "link"`
- **THEN** the description renders `[0021:D4](/enhancements/0021/decisions/)`

#### Scenario: Parity with crdref
- **WHEN** opm-operator's tree at its newest tag is built with the planned config
- **THEN** the generated entries equal crdref's block between its markers

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
