## ADDED Requirements

### Requirement: cobradump releases as its own component
release-please SHALL release the `cobradump/` module as a second component with tags `cobradump/vX.Y.Z`, starting at `0.1.0`, published without a draft step, and SHALL keep the `opm-docs` release unchanged. The release workflow SHALL run goreleaser only for an `opm-docs` release. A `cobradump/` tag SHALL NOT match the default signer glob `refs/tags/v[0-9]*`.

#### Scenario: A cobradump-only change
- **WHEN** a `feat(cobradump)` commit merges and its release PR merges
- **THEN** the tag `cobradump/v0.2.0` exists, the Go module proxy serves that version, and no `opm-docs` archive is built
