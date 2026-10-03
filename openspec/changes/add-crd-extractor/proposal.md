## Why

opm-operator generates its resource reference with `hack/crdref` (about 800 lines) into a marked block of `docs/site/reference/operator-resources.md`. DESIGN.md phase 2 moves it into a bundle docs-kit builds from the controller-gen CRD YAML and the samples, so the operator's CI publishes it per release and the generator and the committed block go.

Gate: `generalize-build-assembly` is merged (registry, docs placement, completable pages, `exclude`, `citations: "link"`). Parallel with the other extractor changes and `pull-docs-placement`.

## What Changes

- **`crd` source kind**: reads every CRD in a directory of controller-gen YAML and, optionally, each kind's kubebuilder sample (hiding fixture samples and stripping scaffold labels); writes `data/crd.json` (`docs.opmodel.dev/data/crd/v1`) with names, scope, versions, printer columns, subresources, the spec and status field tables, the validation rules and the sample.
- **One completable page** at a configured path: the operator keeps its authored front matter and intro (its `docs/site` page of the same path), and the generated entries follow it.
- **"Served by" from configuration**: `reconciledBy` maps a kind to its controller's name. crdref found it by scanning `internal/controller`'s Go syntax; that scan is operator-specific, and Principle VI prefers one explicit input over inference (decided in planning).
- **Citations as links** through the shared `citations: "link"` policy, which reproduces crdref's `/enhancements/<NNNN>/decisions/` links.
- **Parity** with crdref's block at the newest operator tag; **contract C18**.

Release class: MINOR. Additive.

Scope: three sections, the last with the contract and the archive.

## Capabilities

### New Capabilities

- `crd-extractor`: the source kind, its data model and its page.

### Modified Capabilities

None.

## Impact

- Code: `internal/extract/crd` (YAML via `sigs.k8s.io/yaml` into `apiextensionsv1` types, or a minimal local struct if the dependency is too heavy; decided in section 1), `internal/render` (templates), `schema/config.cue`, `docs/contracts.md` C18.
- Consumers: **opm-operator** (sibling change `publish-crd-bundle`): `docs-kit.cue` with the `crd` source and its `docs/site` (excluding the committed page until the switch), removes the marker block, the exclude and `hack/crdref` once the site reads the bundle. **opmodel.dev**: pulls `opm-operator` per site version at the cli's pin.
