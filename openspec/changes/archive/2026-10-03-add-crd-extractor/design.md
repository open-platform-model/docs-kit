# Design: add-crd-extractor

## Context

Builds on `generalize-build-assembly` (registry, docs placement, completable pages, `exclude`, the `link` citation policy; C6, C15). The behavior to reproduce is opm-operator `hack/crdref`: it reads `config/crd/bases` (four CRDs), `config/samples`, and scans `internal/controller` for the controller of each kind, and writes the block between `<!-- BEGIN GENERATED ... -->` and `<!-- END GENERATED ... -->` in `docs/site/reference/operator-resources.md` (authored front matter and intro above it). The site serves it at `/docs/reference/operator-resources/`.

## Goals / Non-Goals

**Goals:** the `crd` kind, its data model, its completable page, parity with crdref's block.

**Non-Goals:** reading Go source (the controller scan); CRDs not produced by controller-gen; one page per kind (the operator has one page today; a `pages: "per-kind"` option waits for a consumer).

## Decisions

### D1. Configuration (C6, C18)

```cue
#CRD: {
	kind:     "crd"
	dir:      =~"^\\./[^/]"  // controller-gen output: "./config/crd/bases"
	samples?: =~"^\\./[^/]"  // kubebuilder samples: "./config/samples"
	// A sample document containing any of these strings is never shown (a
	// dev or e2e fixture, not something a reader can apply).
	hideSamplesMatching: *[] | [...string & !=""]
	// Labels removed from a shown sample when they carry exactly this value
	// (kubebuilder's scaffold labels say how the repository applies it).
	stripLabels: [string & !=""]: string
	page:        =~"^([a-z0-9]+(-[a-z0-9]+)*/)*[a-z0-9]+(-[a-z0-9]+)*\\.md$" // under content/, an owned path
	title:       string & !="" // front matter when no authored page completes it
	description: string & !=""
	weight?:     int & >=1 // the page's weight when no authored page completes it
	order?: [string & !="", ...string & !=""] // these kinds first, in this order; the rest by name
	reconciledBy?: [string & !=""]: string & !="" // kind: the controller that reconciles it
	citations: #Citations
}
```

opm-operator's file, as its sibling change writes it:

```cue
bundles: "opm-operator": {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/operator-resources.md"]}
	version: {from: "tag", prefix: "v"}
	sources: [
		// The authored pages ship in the same bundle (docs/orchestration.md).
		// While the site still builds the operator from git, the committed page
		// holds crdref's block, which would collide with the generated entries,
		// so it is excluded and the crd page stands alone with title and
		// description. When the site reads the bundle, the operator deletes the
		// block and this exclude, and the authored intro completes the page.
		{kind: "markdown", dir: "docs/site", exclude: ["reference/operator-resources.md"]},
		{
			kind:        "crd"
			dir:         "./config/crd/bases"
			samples:     "./config/samples"
			hideSamplesMatching: ["testing.opmodel.dev"]
			stripLabels: {"app.kubernetes.io/name": "opm-operator", "app.kubernetes.io/managed-by": "kustomize"}
			weight:      7
			page:        "reference/operator-resources.md"
			title:       "Operator resources"
			description: "One generated entry per operator resource kind: ModuleInstance, ModulePackage, Platform and TransformerRegistration."
			order:       ["ModuleInstance", "ModulePackage", "Platform", "TransformerRegistration"]
			reconciledBy: {ModuleInstance: "<name>", /* as crdref's scan finds them at the tag */}
			citations:   "link"
		},
	]
}
```

### D2. Decoding (spike in section 1)

YAML is decoded into the `apiextensions.k8s.io/v1` CRD types when the dependency cost is acceptable, else into a local struct holding only the fields D3 reads; the spike records the module-graph growth and picks. Either way the extractor reads only `CustomResourceDefinition` documents and refuses another kind in `dir` naming the file.

**Spike result (section 1).** `go mod graph | wc -l` on `main` (0.3.0) is 1541. Decoding the operator's four CRDs (v1.0.0-beta.4 and `main`) with `k8s.io/apiextensions-apiserver` v0.36.4 plus `sigs.k8s.io/yaml` v1.6.0 grows it to 1773 (+232 edges, +16 `go.mod` lines: `k8s.io/api`, `k8s.io/apimachinery`, gogo protobuf, json-iterator and their kin). A local struct plus `sigs.k8s.io/yaml` alone grows it to 1548 (+7; `go.mod` gains `sigs.k8s.io/yaml` and the indirect `go.yaml.in/yaml/v2`). **Decision: the local struct** (`internal/extract/crd/schema.go`), with `sigs.k8s.io/yaml` kept because a shown sample is re-encoded exactly as crdref does (YAML to JSON to sorted YAML). Documents are split at `---` lines as the Kubernetes YAML reader splits them. The struct decodes non-strictly; an `items` list or a boolean `additionalProperties` decodes to "no schema", as the Kubernetes types do.

### D3. Data model (C18)

As built (section 1; the planned shape, refined where crdref's entries needed it):

```json
{
  "schema": "docs.opmodel.dev/data/crd/v1",
  "page": {"path": "reference/operator-resources.md", "title": "Operator resources", "description": "...", "weight": 7},
  "kinds": [
    {
      "kind": "ModuleInstance", "group": "opmodel.dev", "plural": "moduleinstances",
      "scope": "Namespaced", "shortNames": ["mi"], "categories": [], "subresources": ["status"],
      "versions": [{"name": "v1alpha1", "served": true, "storage": true}],
      "file": "config/crd/bases/opmodel.dev_moduleinstances.yaml",
      "summary": "<first sentence of the schema description>", "notes": ["<paragraph>"],
      "columns": [{"name": "Ready", "type": "string", "jsonPath": ".status...", "priority": 0}],
      "spec":   [{"path": "spec.module", "type": "object", "required": true, "description": "...", "default": null, "enum": []}],
      "status": [{"path": "status.conditions", "type": "[]Condition", "required": false, "description": "...", "default": null, "enum": []}],
      "rules":  [{"field": "spec.module", "rule": "Required", "values": [], "message": "", "by": "api-server"},
                 {"field": "", "rule": "CEL rule", "values": ["<CEL>"], "message": "<message>", "by": "api-server"}],
      "sample": {"file": "config/samples/opmodel.dev_v1alpha1_modulepackage.yaml", "yaml": "<text>"},
      "reconciledBy": "moduleinstance"
    }
  ]
}
```

Refinements over the planned shape, each one a fact crdref's page shows or the renderer needs without reading source (Principle I):

- `page` carries the configured `path`, `title`, `description` and `weight` (null when unset), so the renderer writes the standalone page from the data file alone.
- Each kind carries its own `file` (the planned top-level `files` map, folded into the kind) and `categories` (crdref shows them).
- Field paths are written from `spec`/`status` as the page shows them (`spec.module.path`, `status.conditions`, `spec.parts[].name`), with `.<key>` for a map value, crdref's form, rather than the planned `{}`.
- A rule is `rule` (the sentence's fixed words: `Required`, `One of`, `Matches the pattern`, `CEL rule`, `At most 8 items`), `values` (each shown as code: the enum values, the pattern, the list-map keys, the CEL expression) and `message` (a CEL rule's message, prose). `field` is `""` for the object itself. The renderer writes `<rule> <values as code, comma-joined>; refused with: <message>`, crdref's sentence, without the model holding Markdown.
- `versions` lists the CRD's one version: a CRD with several versions is refused, as crdref refuses it ("want exactly one version with a schema").
- Prose (`summary`, `notes`, `description`, `message`) is the source text with whitespace folded and the citation policy applied: code spans stay in backticks, a linked decision citation is the Markdown link `[0015:D3](/enhancements/0015/decisions/)`, and nothing is escaped. The renderer escapes everything else and keeps exactly that link form, which is how this model "carries links" (C6, Doc-comment rules) while `cue-catalog`'s does not.

**Samples**, as crdref picks them: for each CRD, only the file kubebuilder names `<group>_<version>_<kind>.yaml` (the first served version, the kind lower-cased) in `samples`, and in it the first YAML document whose `apiVersion` is `<group>/<version>` and whose `kind` is the kind; other files (`..._moduleinstance_jellyfin.yaml`, `source_v1_ocirepository.yaml`) are never read. When that document contains a `hideSamplesMatching` string, the kind has no sample. Otherwise it is re-encoded without comments, with each `stripLabels` label removed when its value matches (and `labels` removed when it becomes empty). A missing file means no sample, not an error.

Fields are listed depth-first, properties by name. `rules` come, in that walk, from the object's own CEL rules and `required`, then per value `required`, `minLength`/`maxLength`, `pattern`, `enum`, `minimum`/`maximum` (exclusive bounds read "Greater than"/"Less than"), `minItems`/`maxItems`, `minProperties`/`maxProperties`, `uniqueItems` or a `set` list type, a `map` list type's keys, and `x-kubernetes-validations` (CEL, with its `message`): crdref's set. `spec` and `status` record their own rules too, which crdref skipped (no operator CRD has one, so parity holds). The standard `metav1.Condition` list is a `[]Condition` row whose own rules are not listed. `allOf`, `anyOf`, `oneOf`, `not` and `nullable` anywhere are refused naming the file, the kind and the path, as crdref refuses them (so controller-gen's `anyOf` for an int-or-string field is refused too, with a message naming the field and saying int-or-string is not supported yet; the operator has none; support is docs-kit#23). `by` is `api-server`, the only enforcer the CRD proves. `sample` and `reconciledBy` are `null` when absent; `reconciledBy` or `order` naming a kind no CRD defines is refused.

### D4. The page

One completable page at `page` (C15): per kind, in D3 order, crdref's entry (`## <Kind>`, summary, `### At a glance` table, the columns table, `### Spec` and `### Status` tables, `### Example` with a `yaml` fence, `### Notes`, `### Served by` "The operator's `<controller>` controller watches every <Kind>." only when `reconciledBy` names one, `### Enforcement` table). Its generated body's first heading is `## <first kind>`. Without an authored page it has front matter `title`, `description`, `type: reference` and `weight` when configured (the operator sets 7, the page's weight today). `manifest.json` `pages[].source` is the authored page when completed, else the first CRD file.

The entries are a `text/template` (`internal/render/templates/crd/entries.md.tmpl`, its own template set so the catalog's helpers stay apart) with runs of blank lines folded to one, crdref's rule. Prose is escaped as crdref escapes it: the common set (`\ < > * _ [ ] |`, `{{` as `{\{`) plus `~` and `#`, which schema descriptions use (`#Platform`) and which would otherwise start a strike-through or a heading; an unpaired backtick makes the whole text prose with the backtick escaped; in a table cell a pipe is escaped inside code spans too.

**Parity record (section 2).** `TestOperatorCRDParity` (`internal/render/crd_parity_test.go`, skipped unless `OPM_OPERATOR_CHECKOUT` is set) extracts with opm-operator's planned config (D1) and compares the page's generated body with crdref's block between its markers, line by line. Results, 2026-10-03:

| Operator tree | crdref block | Result |
|---|---|---|
| `main` at `bf8d3c5` | committed (`go run ./hack/crdref -check` passes) | equal |
| `v1.0.0-beta.4` (`0d3532b`) | the tag predates crdref, so `main`'s crdref was run with `-root` over the tag's `config/` and `internal/controller` and `main`'s page | equal |

### D5. Commands

No new command or flag. Messages name the file: "config/samples/opmodel.dev_v1alpha1_platform.yaml: does not parse as YAML: <error>" (exit 2).

## Research & Decisions

### The controller scan (decided in planning)

**Context**: crdref finds each kind's controller by scanning `internal/controller` for a builder chain naming the kind.
**Options considered**:
1. Port the scan - operator-specific Go syntax inference in a generic extractor.
2. `reconciledBy` in config - one explicit line per kind; a renamed controller is a config edit.
3. Drop "Served by" - loses a fact the page has today.
**Decision**: option 2.
**Rationale**: Principle VI (one explicit input over inference); the fact is still the author's statement, as before.

### Citation links (decided in planning)

crdref links decision citations; docs-kit strips them by default. The shared `citations: "link"` policy (`generalize-build-assembly` D5) reproduces crdref's form, so no crd-specific rule exists.

## Risks / Trade-offs

- `reconciledBy` can go stale silently. The operator's sibling change adds a test in its own repo comparing it with the controllers, if it wants the old guarantee.

## Durable decisions

- C18 (new): config (D1), model (D3), page (D4), parity record.
- C6: the source-kind table gains `crd`.
