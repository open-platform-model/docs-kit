# version-history Specification

## Purpose
`opm-docs pull` computes a tab's version history, `<out>/<project>/history.json` (docs/contracts.md C13), from the doc model of every segment it unpacked, so the site can show what each minor added, changed or removed without computing anything itself.

## Requirements

### Requirement: History compares a tab's segments in order
`opm-docs pull` SHALL compute a tab project's history from the `data/catalog.json` of every segment it unpacked in the run, ordered minors ascending then `edge`, comparing each segment with the previous one (for `edge`, the newest minor). The oldest minor SHALL be the floor, and a member present in it SHALL be marked `firstIsFloor: true`. History SHALL be recomputed on every run and never cached across digests. Source: DESIGN decision 12.

#### Scenario: A member of the floor
- **WHEN** the tab pulls 4.5 and 4.6 and `backup@v1alpha1` is in both
- **THEN** its entry has `first` `4.5` and `firstIsFloor` `true`

#### Scenario: A member added later
- **WHEN** `backup@v1beta1` is in 4.6 and not in 4.5
- **THEN** its entry has `first` `4.6` and `firstIsFloor` `false`

#### Scenario: A member only in edge
- **WHEN** a member appears only in `edge`
- **THEN** its entry has `first` `edge`

### Requirement: Field changes are compared by path, gated on the tool minor
For a member in both segments of a pair, `pull` SHALL record `added`, `removed` and `presence` changes by field path, and, only when both bundles' `tool` share MAJOR.MINOR, `type`, `default` and `ref` changes and the `spec` fallback (the spec blocks' CUE tokens differ with comments skipped while no other change was found). Each pair SHALL be listed in `compared` with mode `full` or `paths`. A field's `doc` SHALL never be compared.

#### Scenario: A default changed
- **WHEN** 4.5 and 4.6 were built by opm-docs 0.3.x and `schedule`'s default moved from `"daily"` to `"hourly"`
- **THEN** the member's `changes["4.6"]` holds `{op: "default", path: "schedule", from: "\"daily\"", to: "\"hourly\""}`

#### Scenario: Different tool minors
- **WHEN** 4.5 was built by 0.3.0 and 4.6 by 0.4.0, and only a type string differs
- **THEN** no `type` change is recorded and `compared` lists the pair with mode `paths`

#### Scenario: A validator the walk cannot express
- **WHEN** a member's `matchN` constraint changed between two segments of one tool minor and its fields did not
- **THEN** the member's changes for the later segment hold one `{op: "spec", path: ""}`

#### Scenario: A doc-comment fix
- **WHEN** only a field's doc comment differs between two segments
- **THEN** the member has no change for the later segment

### Requirement: Removed members name the last segment that had them
A member present in the previous segment and absent from the current one SHALL be listed under `removed[<current>]` with its `fqn`, `kind`, `name`, `apiVersion`, `lastIn` (the previous segment) and its `page` there.

#### Scenario: A trait removed in 4.7
- **WHEN** `backup@v1alpha1` is in 4.6 at page `traits/backup-v1alpha1` and absent from 4.7
- **THEN** `removed["4.7"]` lists it with `lastIn` `4.6` and `page` `traits/backup-v1alpha1`

### Requirement: Lineage lists every apiVersion of a name per segment
`history.json` SHALL map each `<kind>/<name>` to, per segment, the apiVersions present there, newest first in the order C8 fixes for page paths.

#### Scenario: Two apiVersions
- **WHEN** 4.6 holds `backup@v1alpha1` and `backup@v1beta1`
- **THEN** `lineage["trait/backup"]["4.6"]` is `["v1beta1", "v1alpha1"]`

### Requirement: The history file is validated and deterministic
`pull` SHALL validate the file against the embedded `#History` schema (`docs.opmodel.dev/history/v1`) before writing it, and SHALL serialize it as JSON with two-space indent, a trailing newline, struct keys in schema order, map keys sorted and no timestamps. A file that fails validation SHALL fail the pull with exit 2 naming the project.

#### Scenario: Two pulls agree
- **WHEN** `pull` runs twice over the same digests with the same tool
- **THEN** both runs write the same `history.json` bytes
