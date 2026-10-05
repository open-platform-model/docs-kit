## Why

Every CRD reference page docs-kit renders says who serves a kind: "The operator's `moduleinstance` controller watches every ModuleInstance." The sentence names a product, and the product is being renamed from opm-operator to opm-controller. Renamed in place it would read "The controller's `moduleinstance` controller ...". The workspace now uses one vocabulary: the controller is the product, and its per-kind loops are reconcilers. The renderer serves any repository's CRDs, so its sentence should name no product at all.

## What Changes

- **Template** `internal/render/templates/crd/entries.md.tmpl`: the `Served by` sentence becomes "The `<name>` reconciler watches every <Kind>." in both layouts (page and section).
- **Goldens and tests**: the three crd goldens and the `Served by` expectations in `crd_test.go` and `crd_section_test.go` take the new sentence. `TestOperatorCRDParity` rewrites crdref's old sentence in its frozen block before comparing, the one place the entries depart from crdref on purpose.
- **`docs/contracts.md` C18**: the page layout quotes the new sentence, and the parity paragraph lists it as the one visible deliberate difference from crdref. The `reconciledBy` comment in C6 and `schema/config.cue` says it names the reconciler.
- **`crd-extractor` spec**: the page requirement allows the new wording; the parity scenario allows the rewritten sentence.

SemVer class: patch (`fix(render)`). No config key, data field, command or exit code changes; only the text of one rendered sentence.

Consumers: opm-operator (soon opm-controller) gets the new sentence on its kind pages when it moves its `.opm-docs-version` and `publish.yml` ref to this release. opmodel.dev bumps its pinned `opm-docs` first, so that the site never builds pages from a docs-kit newer than its own (docs-kit C12 ordering). No other repository uses a `crd` source.

Scope: one section, then the archive step.

## Capabilities

### Modified Capabilities

- `crd-extractor`: the `Served by` sentence names the reconciler without a product.

## Impact

- Code: one template line; test expectations and goldens.
- Docs: `docs/contracts.md` C6 comment and C18.
- Risk: none at runtime. A bundle built before this release keeps the old sentence until its repository rebuilds it.
