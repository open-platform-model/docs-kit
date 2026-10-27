## ADDED Requirements

### Requirement: A page names a project's version with a placeholder
A page SHALL name the version of an OPM project with `{{opm:<name>.version}}`, where `<name>` is one of `cli`, `core`, `library`, `opm-controller`, `opm-controller-module` and `catalog-opm`. The value SHALL be the bare version with no `v`. The list SHALL live in docs-kit; a new name is a docs-kit change.

#### Scenario: A command with a version
- **WHEN** a page holds `opm controller install --version v{{opm:opm-controller.version}}` and the site version's `opm-controller` bundle is `1.0.0-beta.10`
- **THEN** the pulled page holds `opm controller install --version v1.0.0-beta.10`

### Requirement: A name maps to one value
A bundle name SHALL take the `version` of the site version's docs bundle of that name's project: `cli`, `core`, `library` and `opm-controller` map to the docs projects of the same name, and `catalog-opm` maps to `catalog-opm-docs`. The unit name `opm-controller-module` SHALL take the `units` entry of that name in the manifest of a docs bundle of the site version; if two bundles of one site version record it, the pull SHALL fail naming both.

#### Scenario: The module version
- **WHEN** the `opm-controller` bundle of a site version records `units: {"opm-controller-module": "0.1.1"}` and a page holds `{{opm:opm-controller-module.version}}`
- **THEN** the pulled page holds `0.1.1`

#### Scenario: Two bundles record one unit
- **WHEN** two docs bundles of `v1.0` both record `opm-controller-module`
- **THEN** the pull exits 2 naming both bundles

### Requirement: A value the site version cannot supply fails the fill
When a page names a placeholder whose value the site version cannot supply, the pull SHALL exit 2 naming the site version, the page and the placeholder. A value cannot be supplied when the site version pulls no bundle for the name, when no bundle records the unit, or when the value is `edge`.

#### Scenario: No bundle for the name
- **WHEN** a page of `v1.0` holds `{{opm:core.version}}` and `v1.0` pulls no `core` bundle
- **THEN** the pull exits 2 naming `v1.0`, the page and `{{opm:core.version}}`

#### Scenario: An edge bundle
- **WHEN** the `cli` bundle of a site version was pulled at tag `edge` and a page holds `{{opm:cli.version}}`
- **THEN** the pull exits 2 saying the value is `edge`, not a version

### Requirement: Each site version keeps its own numbers
The fill SHALL use only the versions collected for the site version being pulled. Two site versions pulled together SHALL each get the values of their own bundles.

#### Scenario: Two site versions
- **WHEN** `v1.0` collects `cli` `1.0.2` and `v1.1` collects `cli` `1.1.0`, and both carry a page holding `{{opm:cli.version}}`
- **THEN** the `v1.0` page holds `1.0.2` and the `v1.1` page holds `1.1.0`

### Requirement: A leftover placeholder fails the pull
After the fill and before any tree is swapped in, `pull` SHALL scan every page of every staged tree (tabs, sections and site versions) for `{{opm:` and SHALL exit 2 naming the page if one is left, swapping nothing in.

#### Scenario: A placeholder in a tab bundle
- **WHEN** a page of a tab bundle holds `{{opm:cli.version}}`
- **THEN** the pull exits 2 naming that page and the previous trees stay as they were

### Requirement: A local build shows the raw placeholder
`build` and `serve` SHALL NOT fill placeholders. A page built locally keeps the text as written.

#### Scenario: A preview
- **WHEN** `opm-docs build` packs a page holding `{{opm:cli.version}}`
- **THEN** the page in the bundle holds `{{opm:cli.version}}`
