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
	dir:      =~"^\\./"            // controller-gen output: "./config/crd/bases"
	samples?: =~"^\\./"            // one sample per kind: "./config/samples"
	page:     =~"^([a-z0-9]+(-[a-z0-9]+)*/)*[a-z0-9]+(-[a-z0-9]+)*\\.md$" // under content/, an owned path
	title:       string & !=""     // front matter when no authored page completes it
	description: string & !=""
	order?: [string, ...string]    // kinds first, in this order; the rest by name
	reconciledBy?: [string]: string & !="" // kind: the controller that reconciles it
	citations: *"strip" | "link"
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

### D3. Data model (C18)

```json
{
  "schema": "docs.opmodel.dev/data/crd/v1",
  "kinds": [
    {
      "kind": "ModuleInstance", "group": "opmodel.dev", "plural": "moduleinstances",
      "scope": "Namespaced", "shortNames": ["mi"], "subresources": ["status"],
      "versions": [{"name": "v1alpha1", "served": true, "storage": true}],
      "summary": "<first paragraph of the schema description>", "notes": ["<paragraph>"],
      "columns": [{"name": "Ready", "type": "string", "jsonPath": ".status...", "priority": 0}],
      "spec":   [{"path": "module", "type": "string", "required": true, "description": "...", "default": null, "enum": []}],
      "status": [{"path": "conditions[]", "type": "array", "required": false, "description": "...", "default": null, "enum": []}],
      "rules":  [{"field": "spec.module", "rule": "<sentence>", "by": "api-server"}],
      "sample": {"file": "config/samples/...yaml", "yaml": "<text>"},
      "reconciledBy": "moduleinstance"
    }
  ],
  "files": {"ModuleInstance": "config/crd/bases/opmodel.dev_moduleinstances.yaml"}
}
```

Field paths are dot-separated from `spec`/`status`, `[]` for array items, `{}` for `additionalProperties`, depth-first in schema property order (controller-gen sorts properties). `rules` come from `required`, `enum`, `minimum`/`maximum`, `pattern`, `x-kubernetes-validations` (CEL, with its `message`) and `x-kubernetes-immutable`-style markers crdref recognizes; `by` is `api-server`, the only enforcer the CRD proves. `sample` and `reconciledBy` are `null` when absent. The implementer adds fields crdref's entries need (additive, recorded in C18).

### D4. The page

One completable page at `page` (C15): per kind, in D3 order, crdref's entry (`## <Kind>`, summary, `### At a glance` table, the columns table, `### Spec` and `### Status` tables, `### Example` with a `yaml` fence, `### Notes`, `### Served by` "The operator's `<controller>` controller watches every <Kind>." only when `reconciledBy` names one, `### Enforcement` table). Its generated body's first heading is `## <first kind>`. Without an authored page it has front matter `title`, `description`, `type: reference`. `manifest.json` `pages[].source` is the authored page when completed, else the first CRD file.

### D5. Commands

No new command or flag. Messages name the file: "config/samples/x.yaml: a second Platform sample (the first is config/samples/y.yaml); keep one per kind" (exit 2).

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
