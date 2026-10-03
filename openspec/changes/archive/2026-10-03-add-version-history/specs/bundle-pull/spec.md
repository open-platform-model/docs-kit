## ADDED Requirements

### Requirement: Pull writes each tab's version history
After the sweep and before the lock, `pull` SHALL write `<out>/<project>/history.json` for every tab project with at least two segments holding `data/catalog.json`, in every mode (registry, `--frozen`, `--offline`, `--local`), and SHALL remove that file for a project with fewer. It SHALL write no history for a project that is not a tab.

#### Scenario: Local trees get a history
- **WHEN** `pull --local catalog-opm@4.5=a --local catalog-opm@edge=b` runs with no network
- **THEN** `<out>/catalog-opm/history.json` compares 4.5 with edge

#### Scenario: One segment left
- **WHEN** a previous run wrote `history.json` and the tab now resolves only `4.5`
- **THEN** `pull` removes `<out>/catalog-opm/history.json`

### Requirement: The lock records each history file's digest
When `pull` writes a history file, the lock SHALL carry an optional top-level `history` list after `bundles`, one entry per project sorted by project, with `project`, `digest` (`sha256:` and the file's SHA-256) and `path` relative to the lock's directory; the key SHALL be omitted when no file was written. The lock schema id SHALL stay `docs.opmodel.dev/lock/v1`. Under `--frozen` the recorded digest SHALL be the newly written file's, not compared with the frozen lock's.

#### Scenario: Digest recorded
- **WHEN** `pull` writes `catalog-opm/history.json`
- **THEN** the lock's `history` holds `{project: "catalog-opm", digest: "sha256:<hex of the file>", path: "catalog-opm/history.json"}`
