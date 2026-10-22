## MODIFIED Requirements

### Requirement: The openspec reader reads changes as entries
The source SHALL read `<changes>/<id>-<slug>/` and `<changes>/archive/<date>-<id>-<slug>/` as entries, skip every `seed-` and `correction-` change, and require `config.yaml` (its `id` and `slug` equal to the folder's), a `README.md`, `proposal.md` and `design.md`; `tasks.md` and `specs/` MAY be absent. A change folder named any other way SHALL fail the build with exit 2 naming it and the forms `<id>-<slug>`, `seed-<capability>` and `correction-<slug>`.

#### Scenario: A live and an archived entry
- **WHEN** `openspec/changes/0026-x/` and `openspec/changes/archive/2026-10-20-0027-y/` both hold the required files
- **THEN** the bundle holds entries 0026 and 0027, the second with `archived: true`

#### Scenario: A seed change
- **WHEN** `openspec/changes/archive/2026-10-14-seed-single-build-render/` exists
- **THEN** it is no entry and no page

#### Scenario: A refused name
- **WHEN** `openspec/changes/fix-things/` exists
- **THEN** `build` exits 2 with `change folder openspec/changes/fix-things is not <id>-<slug>, a seed-<capability> change or a correction-<slug> change; an enhancement is a change named by its id and slug`

#### Scenario: A missing design
- **WHEN** an entry has no `design.md`
- **THEN** `build` exits 2 naming the entry and `design.md`

### Requirement: Each capability publishes a page
When `specs/` holds a capability, the source SHALL publish `capabilities/_index.md` and `capabilities/<cap>.md` from `<specs>/<cap>/spec.md`. A capability page SHALL take its title from the name and its description from the first sentence of its Purpose, and end with a table "Entries that changed it": one row for each archived change whose deltas touch the capability, with its archive date, linked only when it is a `delivered` entry. A `seed-` change shows `seed` and a `correction-` change shows `correction`, each unlinked. Rows run by date, then entry id (none first), then `correction` before `seed`.

#### Scenario: Entries that changed a capability
- **WHEN** a delivered entry 0027 and the change `seed-registration-acceptance` both touched `registration-acceptance`
- **THEN** its page lists both with their archive dates, 0027 as a link to `/enhancements/0027/`

#### Scenario: Order of rows
- **WHEN** a seed and a correction are archived on 2026-10-21, the delivered entry 0027 on 2026-10-21 and a correction on 2026-10-23
- **THEN** the rows are `correction | 2026-10-21`, `seed | 2026-10-21`, `[0027: <title>](/enhancements/0027/) | 2026-10-21`, `correction | 2026-10-23`

#### Scenario: A rejected entry
- **WHEN** an archived entry with status `rejected` holds a delta for the capability
- **THEN** it has no row

## ADDED Requirements

### Requirement: A correction change is no entry
The openspec reader SHALL skip a live `<changes>/correction-<slug>/` and an archived `<changes>/archive/<date>-correction-<slug>/` as it skips a seed: no entry, no page, no field in `data/enhancements.json`. The `<slug>` MUST be non-empty lower-case words (letters and digits) joined by single `-`, and need not name a capability.

#### Scenario: A live correction
- **WHEN** `openspec/changes/correction-registry-paths/` exists beside entries
- **THEN** `build` succeeds and the bundle holds the same entries and pages as without it

#### Scenario: An archived correction
- **WHEN** `openspec/changes/archive/2026-10-23-correction-registry-paths/` exists
- **THEN** it is no entry and no page

#### Scenario: A malformed slug
- **WHEN** a folder is named `correction-`, `correction`, `correction-Foo_Bar` or, archived, `2026-10-23-correction-`
- **THEN** `build` exits 2 naming the folder

#### Scenario: A numbered name
- **WHEN** a folder is named `0123-correction-x`
- **THEN** it is read as entry 0123 with slug `correction-x`

### Requirement: A capability page lists a correction as an unlinked row
A capability page SHALL list each archived correction whose `specs/` holds the capability as the row `correction` with its archive date, with no link. A correction that touches several capabilities SHALL give a row on each page.

#### Scenario: A correction row
- **WHEN** the archived change `2026-10-23-correction-registry-paths` holds deltas for `registry-paths` and `tag-rules`
- **THEN** both pages list `| correction | 2026-10-23 |`
