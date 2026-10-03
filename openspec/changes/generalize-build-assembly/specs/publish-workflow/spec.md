## ADDED Requirements

### Requirement: Build and publish run in separate jobs
`publish.yml` SHALL build (`check`, `build` or `revise`) in a job that declares `contents: read` and `packages: read` and no `id-token`, and SHALL push, sign and promote in a second job that needs the first, checks out nothing of the caller and takes the bundle tree from a workflow artifact. Its inputs, outputs and the caller's required permissions SHALL be unchanged, and the concurrency groups SHALL be declared at the workflow level, covering both jobs of a run, so two revisions of one release never build at once.

#### Scenario: Two revisions dispatched together
- **WHEN** two `revision` runs of `opm-v4.5.1` are dispatched a second apart
- **THEN** the second run's `build` job starts only after the first run has finished, and the two get different revision numbers

#### Scenario: Repository code never meets the signing token
- **WHEN** a cli release publish runs its dump command
- **THEN** that command runs in the build job, which cannot request an OIDC token, and the signing job runs no step from the cli tree

### Requirement: The workflow installs Go on request
`publish.yml` SHALL accept a boolean input `setup-go` (default `false`); when true, the build job SHALL install Go with a SHA-pinned `actions/setup-go` from `src/go.mod` in `release` mode and from `go.mod` otherwise, before building.

#### Scenario: cli check
- **WHEN** the cli calls `mode: check` with `setup-go: true`
- **THEN** Go from the cli's `go.mod` is on `PATH` when `opm-docs check` runs the dump command
