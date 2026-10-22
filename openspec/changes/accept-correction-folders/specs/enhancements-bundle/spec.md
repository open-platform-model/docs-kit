## MODIFIED Requirements

### Requirement: The openspec reader reads changes as entries
The source SHALL read `<changes>/<id>-<slug>/` and `<changes>/archive/<date>-<id>-<slug>/` as entries, skip every `seed-` change, and require `config.yaml` (its `id` and `slug` equal to the folder's), a `README.md`, `proposal.md` and `design.md`; `tasks.md` and `specs/` MAY be absent. A change folder named any other way SHALL fail the build with exit 2 naming it.

#### Scenario: A live and an archived entry
- **WHEN** `openspec/changes/0026-x/` and `openspec/changes/archive/2026-10-20-0027-y/` both hold the required files
- **THEN** the bundle holds entries 0026 and 0027, the second with `archived: true`

#### Scenario: A seed change
- **WHEN** `openspec/changes/archive/2026-10-14-seed-single-build-render/` exists
- **THEN** it is no entry and no page

#### Scenario: A missing design
- **WHEN** an entry has no `design.md`
- **THEN** `build` exits 2 naming the entry and `design.md`

### Requirement: Each capability publishes a page
When `specs/` holds a capability, the source SHALL publish `capabilities/_index.md` and `capabilities/<cap>.md` from `<specs>/<cap>/spec.md`. A capability page SHALL take its title from the name and its description from the first sentence of its Purpose, and end with a table "Entries that changed it": one row for each archived change whose deltas touch the capability, with its archive date, linked only when it is a `delivered` entry.

#### Scenario: Entries that changed a capability
- **WHEN** a delivered entry 0027 and the change `seed-registration-acceptance` both touched `registration-acceptance`
- **THEN** its page lists both with their archive dates, 0027 as a link to `/enhancements/0027/`

#### Scenario: A rejected entry
- **WHEN** an archived entry with status `rejected` holds a delta for the capability
- **THEN** it has no row

## ADDED Requirements

### Requirement: A correction change is no entry
The openspec reader SHALL skip a live `<changes>/correction-<slug>/` and an archived `<changes>/archive/<date>-correction-<slug>/` as it skips a seed: no entry, no page, no field in `data/enhancements.json`. The `<slug>` is any lower-case words joined by `-`, and need not name a capability.

#### Scenario: A live correction
- **WHEN** `openspec/changes/correction-registry-paths/` exists beside entries
- **THEN** `build` succeeds and the bundle holds the same entries and pages as without it

#### Scenario: An archived correction
- **WHEN** `openspec/changes/archive/2026-10-23-correction-registry-paths/` exists
- **THEN** it is no entry and no page

#### Scenario: A malformed name
- **WHEN** `openspec/changes/correction/` exists
- **THEN** `build` exits 2 naming the folder and listing the `correction-<slug>` and `seed-<capability>` forms

### Requirement: A capability page lists a correction as an unlinked row
A capability page SHALL list each archived correction whose `specs/` holds the capability as a row with the text `correction` and its archive date, with no link and no entry id. A correction that touches several capabilities SHALL give a row on each page.

#### Scenario: A correction row
- **WHEN** the archived change `2026-10-23-correction-registry-paths` holds deltas for `registry-paths` and `tag-rules`
- **THEN** both pages list `correction` with `2026-10-23`, unlinked

#### Scenario: Order with other rows
- **WHEN** a seed, a delivered entry and a correction touch one capability
- **THEN** the rows run by date, then id, and the correction row is no link
