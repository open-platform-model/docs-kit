## ADDED Requirements

### Requirement: A manifest may record the versions of units without a bundle
`manifest.json` MAY carry `units`: a map from a unit name to an exact version without `v`, for a unit of the repository that is released on its own and has no docs bundle (`opm-controller-module`). The field SHALL be optional, SHALL be absent when the build declares no `units`, and SHALL NOT rename or change any existing field. A reader that predates it SHALL refuse the bundle, as it refuses any unknown field of the closed manifest.

#### Scenario: A controller bundle records the module
- **WHEN** the opm-controller bundle is built with `units` declared
- **THEN** `manifest.json` holds `"units": {"opm-controller-module": "0.1.1"}` beside `version` and `pins`

#### Scenario: No units declared
- **WHEN** a bundle is built from a `docs-kit.cue` with no `units`
- **THEN** its `manifest.json` has no `units` key and its bytes equal those of the same build before this field existed

#### Scenario: A malformed unit version
- **WHEN** a manifest holds `"units": {"opm-controller-module": "v0.1.1"}`
- **THEN** the manifest schema refuses it
