## ADDED Requirements

### Requirement: The release key is read only on main
The release workflow SHALL read `RELEASE_APP_PRIVATE_KEY` in exactly one job, `release-please`, and that job SHALL run in the `release` environment, whose deployment branch policy admits `main` only. Every job of `release.yml` and `ci.yml` SHALL declare the token permissions it uses, so no job depends on the repository's default `GITHUB_TOKEN` permissions; the `release-please` job, which acts only through the App token, SHALL declare none. The `goreleaser` job SHALL NOT restore an Actions cache.

#### Scenario: A push to main
- **WHEN** a commit is pushed to `main`
- **THEN** the `release-please` job runs in the `release` environment and mints the App token from the key

#### Scenario: A workflow on another branch asks for the key
- **WHEN** a workflow run on any branch other than `main` names the `release` environment in a job
- **THEN** GitHub refuses to run that job, and no job outside the environment reads the key

#### Scenario: Release binaries build without a cache
- **WHEN** the `goreleaser` job builds a release
- **THEN** `actions/setup-go` runs with `cache: false`, and no step restores an Actions cache
