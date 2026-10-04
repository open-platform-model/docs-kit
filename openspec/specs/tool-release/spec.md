# tool-release Specification

## Purpose
How `opm-docs` is distributed: the release assets every docs-kit release carries, and how a consumer pins and verifies them instead of building the tool from source.

## Requirements

### Requirement: Every release publishes binaries and checksums
Each docs-kit release `vX.Y.Z` SHALL carry `opm-docs_X.Y.Z_<os>_<arch>.tar.gz` for `linux_amd64`, `linux_arm64`, `darwin_arm64` and `darwin_amd64`, each holding the `opm-docs` binary stamped with version `X.Y.Z` and the repository's Apache-2.0 `LICENSE`, and a `checksums.txt` with the SHA-256 of every archive in `sha256sum` format. The release SHALL be published only after every asset is attached (draft-first).

#### Scenario: A published release is complete
- **WHEN** the release workflow publishes `v0.1.0`
- **THEN** the release lists four archives and `checksums.txt`, and `opm-docs version` from any archive prints `opm-docs 0.1.0`

### Requirement: Consumers pin a release and verify it
Every consumer, `publish.yml` included (it reads its caller's file), SHALL pin one release in one of two ways: a repo-root `.opm-docs-version` holding the release tag, with the archive for its os and arch verified against that release's `checksums.txt` line before extracting; or, in its own build image, the `linux_amd64` archive's URL with its SHA-256 written beside it (the value of its `checksums.txt` line), verified before extracting. Either way it SHALL stop on a missing line or a mismatch. Every release SHALL keep shipping the `linux_amd64` archive and `checksums.txt` under the names `docs/contracts.md` C12 fixes so both patterns work. No consumer SHALL build the tool from source (`go run`, `go install`) to produce or pull a bundle.

#### Scenario: Pinned in a build image
- **WHEN** a consumer's Dockerfile pins `opm-docs_0.1.0_linux_amd64.tar.gz` by the SHA-256 from `v0.1.0`'s `checksums.txt`
- **THEN** the image build verifies the archive against that SHA-256 and installs `opm-docs` 0.1.0, with no `.opm-docs-version`

#### Scenario: A tampered archive
- **WHEN** the downloaded archive's SHA-256 differs from its `checksums.txt` line
- **THEN** the install task fails naming the archive, and no `opm-docs` binary is installed

### Requirement: cobradump releases as its own component
release-please SHALL release the `cobradump/` module as a second component with tags `cobradump/vX.Y.Z`, starting at `0.1.0`, published without a draft step, and SHALL keep the `opm-docs` release unchanged; a commit touching only `cobradump/` SHALL NOT release `opm-docs` (the root package excludes that path). The release workflow SHALL run goreleaser only for an `opm-docs` release. A `cobradump/` tag SHALL NOT match the default signer glob `refs/tags/v[0-9]*`.

#### Scenario: A cobradump-only change
- **WHEN** a `feat(cobradump)` commit merges and its release PR merges
- **THEN** the tag `cobradump/v0.2.0` exists, the Go module proxy serves that version, and no `opm-docs` release or archive is built

### Requirement: The release key is read only on main
The release workflow SHALL read `RELEASE_APP_PRIVATE_KEY` in exactly one job, `release-please`, and that job SHALL run in the `release` environment, whose deployment branch policy admits `main` only. `release.yml` and `ci.yml` SHALL declare their token permissions explicitly, at workflow or job level, so no job depends on the repository's default `GITHUB_TOKEN` permissions; the `release-please` job, which acts only through the App token, SHALL declare none, and SHALL mint that token with only `contents`, `pull-requests` and `issues` write. The `goreleaser` job SHALL NOT restore an Actions cache and SHALL check out without persisting the job token.

#### Scenario: A push to main
- **WHEN** a commit is pushed to `main`
- **THEN** the `release-please` job runs in the `release` environment and mints the App token from the key

#### Scenario: A workflow on another branch asks for the key
- **WHEN** a workflow run on any branch other than `main` names the `release` environment in a job
- **THEN** GitHub refuses to run that job, and no job outside the environment reads the key

#### Scenario: Release binaries build without a cache
- **WHEN** the `goreleaser` job builds a release
- **THEN** `actions/setup-go` runs with `cache: false`, no step restores an Actions cache, and `.git/config` holds no token
