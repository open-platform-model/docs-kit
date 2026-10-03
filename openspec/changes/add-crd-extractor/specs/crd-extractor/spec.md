## ADDED Requirements

### Requirement: The crd source reads controller-gen CRDs and samples
A `crd` source SHALL read every `CustomResourceDefinition` in the YAML files of its `dir`, and, when `samples` is set, the sample of each kind there (matched by `apiVersion` and `kind`, at most one per kind; a second is an error naming both files), and SHALL write `data/crd.json` with schema `docs.opmodel.dev/data/crd/v1`: per kind its group, served versions with the storage version marked, scope, plural, short names, subresources, printer columns, the spec and status field tables (path, type, required, description cleaned by the doc-comment rules and the citation policy, default, enum), the validation rules with what enforces them, the sample text, and its `reconciledBy` controller when configured. Kinds SHALL be ordered by the configured `order`, then by kind name.

#### Scenario: A required spec field
- **WHEN** `ModuleInstance`'s schema lists `module` under `spec.required`
- **THEN** the model's spec table marks `module` required

#### Scenario: Two samples of one kind
- **WHEN** `config/samples` holds two `Platform` samples
- **THEN** `build` exits 2 naming both files

### Requirement: The crd page is one completable page
The renderer SHALL write one page at the source's `page` path, one of the bundle's owned paths: per kind a `## <Kind>` entry with `### At a glance`, the printer-column table, `### Spec`, `### Status`, `### Example` when a sample exists, `### Notes`, `### Served by` when `reconciledBy` names a controller, and `### Enforcement`, in crdref's order and wording. The page SHALL be completable, so an authored page with the same path from a `markdown` source supplies the front matter and intro. The entries SHALL match crdref's generated block for the same tree.

#### Scenario: Authored intro kept
- **WHEN** a `markdown` source over `docs/site` supplies `reference/operator-resources.md` holding front matter and an intro paragraph and no `## ModuleInstance` heading
- **THEN** the bundle's page is that front matter and intro followed by the four generated entries

#### Scenario: Citation linked
- **WHEN** a field description cites `0021:D4` and the source has `citations: "link"`
- **THEN** the description renders `[0021:D4](/enhancements/0021/decisions/)`

#### Scenario: Parity with crdref
- **WHEN** opm-operator's tree at its newest tag is built with the planned config
- **THEN** the generated entries equal crdref's block between its markers
