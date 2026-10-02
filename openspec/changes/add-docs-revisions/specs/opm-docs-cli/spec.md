## ADDED Requirements

### Requirement: revise builds a docs revision
`opm-docs` SHALL provide `revise --project P --tag T --fix F` with the optional flags `--out`, `--registry` and `--config`, following the steps of `design.md` D1, exiting 0 when the revision is built into `--out`, 1 on a usage error and 2 when a step refuses. It SHALL push nothing.

#### Scenario: Missing fix
- **WHEN** `opm-docs revise --project catalog-opm --tag opm-v4.4.5` runs without `--fix`
- **THEN** it exits 1 naming `--fix`

#### Scenario: Built, not pushed
- **WHEN** `revise` succeeds
- **THEN** `out/<project>/manifest.json` holds the next revision and the registry is unchanged
