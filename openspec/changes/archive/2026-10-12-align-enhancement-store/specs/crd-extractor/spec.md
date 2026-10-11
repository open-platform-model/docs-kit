## MODIFIED Requirements

### Requirement: The crd page is one completable page
In the page layout (the source sets `page`, not `section`), the renderer SHALL write one page at the source's `page` path, one of the bundle's owned paths: per kind a `## <Kind>` entry with `### At a glance`, the printer-column table, `### Spec`, `### Status`, `### Example` when a sample exists, `### Notes`, `### Served by` when `reconciledBy` names a reconciler, and `### Enforcement`, in crdref's order and wording. The one exception is the `### Served by` sentence, which SHALL name the reconciler without naming a product: "The `<name>` reconciler watches every <Kind>." The page SHALL be completable, so an authored page with the same path from a `markdown` source supplies the front matter and intro; without one, the page takes the configured `title`, `description` and `weight`. Apart from that sentence, the entries SHALL match crdref's generated block for the same tree.

#### Scenario: Authored intro kept
- **WHEN** a `markdown` source over `docs/site` supplies `reference/operator-resources.md` holding front matter and an intro paragraph and no `## ModuleInstance` heading
- **THEN** the bundle's page is that front matter and intro followed by the four generated entries

#### Scenario: Citation linked
- **WHEN** a field description names `enhancement 0021` and the source has `citations: "link"`
- **THEN** the description renders `[enhancement 0021](/enhancements/0021/)`

#### Scenario: Decision citation removed
- **WHEN** a field description ends with a decision citation (an id, a colon and a decision number) and the source has `citations: "link"`
- **THEN** the description renders without the citation, and without a link

#### Scenario: Parity with crdref
- **WHEN** opm-operator's tree at its newest tag is built with the planned config
- **THEN** the generated entries equal crdref's block between its markers, once crdref's "The operator's `<name>` controller watches every <Kind>." sentences read "The `<name>` reconciler watches every <Kind>."

#### Scenario: The served-by sentence names no product
- **WHEN** `reconciledBy` maps `Widget` to `widget`
- **THEN** the Widget entry's `Served by` part reads "The `widget` reconciler watches every Widget." and names no product

### Requirement: CRD text never reaches the page as markup
The site renders raw HTML, so the crd source SHALL treat every CRD and sample string as untrusted. It SHALL refuse, exiting `2` and naming the file, a field it does not read (strict decoding), a construct its page cannot show, a kind name not matching `^[A-Z][A-Za-z0-9]*$`, a scope other than `Namespaced` or `Cluster`, a printer-column type outside `integer`, `number`, `string`, `boolean` and `date`, and a CRD or sample file that is not a regular file. The renderer SHALL escape all prose, SHALL write as links only the enhancement references it finds under the `link` policy, and SHALL fence a sample with more backticks than any backtick or tilde run in it.

#### Scenario: A sample cannot close its fence
- **WHEN** a shown sample holds a block scalar with a "```" line followed by a `<script>` line
- **THEN** the example's fence is longer than three backticks and the `<script>` line stays inside it

#### Scenario: A description cannot forge a link
- **WHEN** a field description holds `[enhancement 0021](/enhancements/0099/)` and `[x](javascript:alert(1))`
- **THEN** both stay text with their brackets escaped, and only a bare `enhancement 0021` becomes a link to its own enhancement

#### Scenario: An unread field is refused
- **WHEN** a CRD carries `spec.conversion`
- **THEN** the build exits `2` naming the file and the unknown field
