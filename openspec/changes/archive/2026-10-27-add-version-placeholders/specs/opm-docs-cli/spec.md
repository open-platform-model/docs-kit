## ADDED Requirements

### Requirement: A bundle may declare units read from a release manifest
A `docs-kit.cue` bundle SHALL accept `units`: a map from a unit name (`opm-controller-module`) to `{file, key}`. `file` is a repository-relative JSON file without a `..` segment and defaults to `.release-please-manifest.json`; `key` is a non-empty key of that file's top-level object.

#### Scenario: A unit name not on the list
- **WHEN** `docs-kit.cue` declares a unit named `x`
- **THEN** the config validation exits 1 before any extraction, naming the allowed unit names

### Requirement: build records each unit and refuses a bad value
`build` SHALL read each unit's `file` from the source tree it builds, SHALL require the value to be an exact version without `v`, and SHALL record it in `manifest.json` `units`. A missing file, a missing key or a malformed value SHALL exit 2 naming the file, the key and the form required.

#### Scenario: The controller module
- **WHEN** opm-controller declares `units: "opm-controller-module": key: "modules/opm_controller"` and its `.release-please-manifest.json` holds `"modules/opm_controller": "0.1.1"`
- **THEN** `build` records `units: {"opm-controller-module": "0.1.1"}`

#### Scenario: A missing key
- **WHEN** the file has no `modules/opm_controller` key
- **THEN** `build` exits 2 naming the file and the key

### Requirement: revise can refresh the units of a release
`opm-docs revise --project P --tag T --refresh-units`, exclusive with `--fix`, SHALL build the next revision of T's version with the newest published revision's `source.patches` unchanged and `units` read from the checkout of `main`. It SHALL push nothing.

#### Scenario: A module release
- **WHEN** `main` holds module `0.2.0`, revision 0 of the controller release recorded `0.1.1`, and `revise --refresh-units` runs
- **THEN** revision 1 is built with the same pages and `units` `0.2.0`

### Requirement: revise --refresh-units refuses what cannot be refreshed
`revise --refresh-units` SHALL exit 2 when the newest published revision is not signed, when `main`'s `docs-kit.cue` declares no `units` for the project, or when the values equal the newest revision's ("units already current in V.n"). Given with `--fix`, it SHALL exit 1.

#### Scenario: Nothing to refresh
- **WHEN** the newest revision already records `0.2.0` and `main` holds `0.2.0`
- **THEN** `revise` exits 2 saying the units are already current in that revision

#### Scenario: Both a fix and a refresh
- **WHEN** `revise` gets `--fix` and `--refresh-units`
- **THEN** it exits 1 saying the two are exclusive
