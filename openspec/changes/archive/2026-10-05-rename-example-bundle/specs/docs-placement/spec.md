## MODIFIED Requirements

### Requirement: A bundle can carry the versions it pins
When a bundle's config has `pins`, `build` SHALL run its command, SHALL require a `docs.opmodel.dev/pins/v1` document whose keys are exactly `pins.projects` and whose values are SemVer versions without `v`, and SHALL write them to `manifest.json` `pins`. A missing or extra project, or a malformed version, SHALL fail the build with exit 2 naming it. Source: DESIGN decision 10.

#### Scenario: cli pins
- **WHEN** the cli config pins `library`, `core` and `opm-controller` and the command prints all three
- **THEN** `manifest.json` holds `pins: {library: "1.0.0-beta.1", core: "2.0.0-beta.1", "opm-controller": "1.0.0-beta.4"}`

#### Scenario: A pin missing from the output
- **WHEN** the command prints no `core` pin
- **THEN** `build` exits 2 naming `core` and the command
