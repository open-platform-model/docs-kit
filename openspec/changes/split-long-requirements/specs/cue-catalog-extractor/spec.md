## MODIFIED Requirements

### Requirement: The spec is extracted as structured fields beside its text
For each member the extractor SHALL record `spec.fields`, walked from the evaluated value: every field under the spec key with its dot path (`[]` for a list element, `[string]` for a pattern constraint), formatted type, `presence` (`regular`, `optional` or `required`), formatted default or null, cleaned doc, and `ref` naming a definition outside the member's package where the walk stops.

#### Scenario: Required and defaulted fields
- **WHEN** a fixture member's spec declares `name!: string` and `replicas: int | *1`
- **THEN** `spec.fields` holds `{path: "name", presence: "required", default: null}` and `{path: "replicas", presence: "regular", default: "1"}`

#### Scenario: A vendored type stops the walk
- **WHEN** a spec field's value is a definition from the module's vendored Kubernetes schemas
- **THEN** that field's entry carries `ref` with the definition name and no child fields are listed under it

## ADDED Requirements

### Requirement: The spec.fields walk stops at the module boundary and at a visited definition
The walk is the one of the requirement "The spec is extracted as structured fields beside its text". The walk SHALL stop at the module boundary and at a definition already visited, and SHALL not exceed depth 12.

#### Scenario: A definition already visited
- **WHEN** the walk reaches a definition it already visited
- **THEN** the walk stops there

### Requirement: The authored spec block is recorded as spec.cue
For each member the extractor SHALL record `spec.cue`, the formatted authored spec block plus the in-module definitions it references (another member's spec root linked, not repeated; out-of-module and vendored packages named only).

#### Scenario: A referenced spec root
- **WHEN** a member's spec references another member's spec root
- **THEN** `spec.cue` links that spec root and does not repeat it
