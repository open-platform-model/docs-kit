## Purpose

How `opm-docs` is distributed: the release assets every docs-kit release carries, and how a consumer pins and verifies them instead of building the tool from source.

## ADDED Requirements

### Requirement: Every release publishes binaries and checksums
Each docs-kit release `vX.Y.Z` SHALL carry `opm-docs_X.Y.Z_<os>_<arch>.tar.gz` for `linux_amd64`, `linux_arm64`, `darwin_arm64` and `darwin_amd64`, each holding the `opm-docs` binary stamped with version `X.Y.Z` and `LICENSE`, and a `checksums.txt` with the SHA-256 of every archive in `sha256sum` format. The release SHALL be published only after every asset is attached (draft-first).

#### Scenario: A published release is complete
- **WHEN** the release workflow publishes `v0.1.0`
- **THEN** the release lists four archives and `checksums.txt`, and `opm-docs version` from any archive prints `opm-docs 0.1.0`

### Requirement: Consumers pin a release and verify it
A consumer that runs `opm-docs` outside `publish.yml` SHALL pin it in a repo-root `.opm-docs-version` holding one release tag, SHALL download that release's archive for its os and arch and `checksums.txt`, SHALL verify the archive against its `checksums.txt` line before extracting, and SHALL stop on a missing line or a mismatch. No consumer SHALL build the tool from source (`go run`, `go install`) to produce or pull a bundle.

#### Scenario: A tampered archive
- **WHEN** the downloaded archive's SHA-256 differs from its `checksums.txt` line
- **THEN** the install task fails naming the archive, and no `opm-docs` binary is installed
