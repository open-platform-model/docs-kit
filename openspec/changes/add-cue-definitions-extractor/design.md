# Design: add-cue-definitions-extractor

## Context

The registry, docs placement, completable pages and the citation policy come from `generalize-build-assembly` (C6, C14, C15). The behavior to reproduce is core's `tools/refgen` (`groups.go` inclusion list, `defs.go` collection and uses, `spec.go` spec text, shape and rules, `render.go` pages), whose committed output is `core/docs/site/reference/definitions/` (nine pages). The site links these pages at `/docs/reference/definitions/<page>/#<anchor>` today; those URLs stay.

## Goals / Non-Goals

**Goals:** the `cue-definitions` kind, its data model, its pages, parity with refgen.

**Non-Goals:** evaluating the package (refgen parses; evaluation would change what a spec block shows); a generic grouping heuristic (the inclusion list is the author's).

## Decisions

### D1. Configuration (C6, C17)

```cue
#CueDefinitions: {
	kind:    "cue-definitions"
	package: =~"^\\./"              // the package directory, repo-relative: "./src"
	skip:    *[] | [...string]       // path.Match globs on file base names: ["*_pins.cue"]
	section: =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$" // a directory, one of the bundle's owned paths: "reference/definitions/"
	title:       string & !=""       // the section index's front matter
	description: string & !=""
	weight?:     int & >=1          // the section index's weight among its siblings
	pages: [#DefPage, ...#DefPage]
	exclude: [=~"^#"]: string & !="" // definition: the reason it is left out
	citations: *"strip" | "link"
}

#DefPage: {
	file:        =~"^[a-z0-9]+(-[a-z0-9]+)*$"
	title:       string & !=""
	description: string & !=""
	definitions: [=~"^#", ...=~"^#"]
}
```

The module path shown on pages is read from the enclosing `cue.mod/module.cue` (`module:`), never configured. core's file, as its sibling change writes it (the lists moved verbatim from `tools/refgen/groups.go`):

```cue
bundles: core: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/definitions/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:        "cue-definitions"
		package:     "./src"
		skip:        ["*_pins.cue"]
		section:     "reference/definitions/"
		title:       "Definitions"
		description: "Every OPM definition type, generated from the CUE schema in core."
		weight:      1
		pages: [
			{file: "modules-and-instances", title: "Modules and instances", description: "...", definitions: ["#Module", "#ModuleInstance", "#InstanceIdentity"]},
			// ... the other seven pages of groups.go, in order
		]
		exclude: {"#BlueprintMap": "map shorthand", /* ... the rest of groups.go's excluded map */}
	}, {
		// core's authored pages ship in the same bundle (docs/orchestration.md);
		// the exclude goes when the committed generated pages are deleted.
		kind: "markdown", dir: "docs/site", exclude: ["reference/definitions/"]
	}]
}
```

### D2. Extraction

As refgen: `cue/parser` with comments; exported top-level definitions only (`#Name`, not `_#`); doc comment = the comment group attached before the field, cleaned by "Doc-comment rules" and the citation policy; summary = the first paragraph (no `metadata.description` exists for a definition, so the catalog summary rule does not apply); spec text = the field printed with `cue/format` after refgen's transforms (hidden fields dropped, maintainer comments dropped, nested sections and elisions as `spec.go` does); shape = struct/closed/literal kind or the constraint (refgen's `atAGlance`); uses = definitions referenced by identifier in the field, used-by its inverse, both limited to placed or excluded names and followed through excluded definitions (refgen's rule for map shorthands); rules = refgen's `enforcement` (required fields, assertions, string constraints), tagged `cue`.

### D3. Placement checks

Every exported definition is in exactly one `pages[].definitions` or in `exclude`. In a normal build, an unplaced, a twice-placed or an unknown name is exit 2, listing all of them. With the config from outside the source tree (`Input.Outside`), each is a warning on stderr, unplaced definitions are left out, unknown names are skipped, and a page left empty is dropped (and with it its index row), so a backfill of an older tag builds.

### D4. Data model (C17)

```json
{
  "schema": "docs.opmodel.dev/data/cue-definitions/v1",
  "modulePath": "opmodel.dev/core@v2",
  "version": "2.0.0-beta.1",
  "section": "reference/definitions/",
  "title": "Definitions",
  "description": "Every OPM definition type, generated from the CUE schema in core.",
  "weight": 1,
  "pages": [
    {"file": "components", "title": "Components", "description": "...", "weight": 2, "definitions": ["#Component", "#ComponentNames"]}
  ],
  "definitions": [
    {
      "name": "#Component", "anchor": "component", "page": "components",
      "file": "src/component.cue",
      "summary": "<first paragraph>", "notes": ["<paragraph>"],
      "shape": "struct, closed, `kind: \"Component\"`",
      "cue": "<the formatted spec block>",
      "uses": ["#Blueprint"], "usedBy": ["#Module"],
      "rules": [{"rule": "<sentence>", "by": "cue"}]
    }
  ],
  "excluded": [{"name": "#BlueprintMap", "reason": "map shorthand"}]
}
```

Definitions are listed in page order, then in each page's configured order; `excluded` by name. Every list is present, empty or not. The implementer may add fields refgen's pages need (additive; recorded in C17), never remove one.

### D5. Pages

| Path | Page |
|---|---|
| `<section>_index.md` | front matter `title`, `description`, `weight` when configured, no `type`; refgen's intro sentence with the module path; `## Pages` (one line per page); `## All definitions` (table: definition, page, summary). Completable (D3 of `generalize-build-assembly`), first heading `## Pages`, so core may author the intro instead. |
| `<section><file>.md` | front matter `title`, `description`, `type: reference`, `weight` = the page's position (1-based); refgen's "Definitions on this page" line; per definition `## <name>`, summary, `**At a glance**` list (Source with file and module, Shape, Uses, Used by), `**Spec**` cue fence, notes, rules |

Anchors are the name lowercased without `#` (`#ComponentNames` is `componentnames`), as refgen writes them; links are `/docs/<section><file>/#<anchor>`. No marker comment (C8 rule for generated pages).

### D6. Commands

No new command or flag. Messages: "`#Policy` (src/policy.cue) is exported but neither placed in a page nor excluded in docs-kit.cue; add it to a page's `definitions` or to `exclude` with a reason" (exit 2).

## Research & Decisions

### Parse, not evaluate (decided in planning)

**Context**: `cue-catalog` evaluates; refgen parses.
**Options considered**: 1. parse, as refgen - the spec block shows the authored text and uses are syntactic; 2. evaluate - would expand references and change what readers see.
**Decision**: option 1.
**Rationale**: parity with the committed pages, and a definitions package is documented by what it says, not what it evaluates to.

### Strictness on backfills (decided in planning)

**Context**: a release built with `main`'s config against an older tag (C5) sees a different set of definitions.
**Decision**: warnings and omission, as a missing `markdown` dir is tolerated in the same situation (C5).

### Parity record (2026-10-03)

**Context**: task 2.2 runs `TestCoreDefinitionsParity` against core's newest tag. That tag, `v2.0.0-beta.1`, predates `tools/refgen` and the committed pages (both landed after it), so no tag carries pages to compare.
**Decision**: parity is measured at core `main`, commit `c5a6076` (`git describe`: `v2.0.0-beta.1-15-gc5a6076`), where `refgen -check` reports the nine committed pages current. Result: all nine pages equal refgen's byte for byte once the marker comments are removed. No planned difference. The test needs only `OPM_CORE_CHECKOUT`; it does not require a tag.
**Rationale**: the pages to match exist only on `main`; the first core tag that carries them becomes the next measuring point.

## Risks / Trade-offs

- Parity is measured at one core tag; a later refgen change in core before the cutover must be ported or accepted as a planned difference in C17's parity record.

## Durable decisions

- C17 (new): config (D1), model (D4), pages and anchors (D5), the parity record.
- C6: the source-kind table gains `cue-definitions`.
