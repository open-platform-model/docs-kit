## Purpose

How the site pulls docs bundles: the pull config, tag resolution, signature verification, the unpack layout, the cache and the lock.

## ADDED Requirements

### Requirement: The pull config names tabs, owners and the trusted signer
`opm-docs pull --config <file>` SHALL validate the file against the embedded `#Pull` schema: a `registry`, a `signer` (issuer, workflow URL, allowed ref globs) and `tabs` keyed by project, each with the `repo` allowed to sign it, its URL `root`, the oldest minor `from` and whether `edge` is shown. Phase 1 SHALL refuse any other placement.

#### Scenario: Missing owner repository
- **WHEN** a tab entry has no `repo`
- **THEN** `pull` exits 1 naming the project and the missing field, before any network call

### Requirement: Tab versions resolve from the registry's tags
For each tab, `pull` SHALL list the repository's tags, ignore `sha256-*` signature tags, select every minor tag (`<MAJOR>.<MINOR>`) at or above `from`, plus `edge` when enabled, resolve each to a digest, and refuse a selected tag whose build's `version` annotation is not in that minor or whose `project` annotation differs. A new minor SHALL appear with no change to the config. Source: DESIGN decisions 8 and 9.

#### Scenario: A new minor appears
- **WHEN** catalog_opm publishes its first 4.5 build and the site pulls with an unchanged config
- **THEN** the unpack directory gains `catalog-opm/4.5/` and the lock gains its entry

#### Scenario: Older minors are not shown
- **WHEN** the registry holds `4.3` and the tab's `from` is `4.4`
- **THEN** `pull` does not fetch `4.3`

### Requirement: Nothing is unpacked before its signature verifies
`pull` SHALL verify each resolved digest's Sigstore bundle with sigstore-go against the public-good trusted root, requiring a signed certificate timestamp, a transparency-log entry and an observer timestamp, the artifact digest equal to the manifest digest, issuer `https://token.actions.githubusercontent.com`, a SAN equal to the signer workflow URL at a ref matching an allowed glob, Source Repository URI `https://github.com/<repo>` of the tab, and Source Repository Ref `refs/heads/main`. Only then SHALL it fetch the layer by digest, unpack it with the bundle-format guards and lint it in bundle mode. Any failure SHALL fail the whole pull, naming the project, tag and digest.

#### Scenario: A bundle signed by another repository
- **WHEN** a digest under `docs/catalog-opm` carries a valid signature whose Source Repository URI is `https://github.com/open-platform-model/cli`
- **THEN** `pull` exits 2 naming `catalog-opm`, the tag, the digest and the repository mismatch, and unpacks nothing for it

#### Scenario: Unsigned edge
- **WHEN** `edge` resolves to a digest with no signature
- **THEN** `pull` exits 2 naming `edge` and the digest

### Requirement: The unpack layout and lock are fixed
`pull` SHALL unpack each bundle into `<out>/<project>/<segment>/` (segment `<MAJOR>.<MINOR>` or `edge`), remove every project or segment directory under `<out>` it did not write in this run, and write the lock (`docs.opmodel.dev/lock/v1`) with the fields `design.md` C7 fixes, every entry including the tab's placement `root`, sorted, with no timestamps. The same resolution SHALL write byte-identical lock files.

#### Scenario: Stable lock
- **WHEN** `pull` runs twice and no tag moved in between
- **THEN** both runs write the same `lock.json` bytes

### Requirement: Frozen, offline and local pulls
With `--frozen <lock>`, `pull` SHALL fetch exactly the digests the lock names, still verifying and linting. With `--offline` (only beside `--frozen`) it SHALL use only the cache and fail naming the first missing digest. With `--local <project>@<segment>=<dir>`, repeatable, it SHALL take that segment of that project from a local bundle tree, refusing a segment that differs from the one the tree's manifest implies; it SHALL skip signature verification and the Sigstore trusted root for local entries, still validate, guard and lint them, skip registry resolution for every project named by `--local`, and mark each local lock entry `"local": true`.

#### Scenario: An all-local pull needs no network
- **WHEN** `pull` runs with `--local catalog-opm@4.4=a --local catalog-opm@4.5=b --local catalog-opm@edge=c` and every tab is `catalog-opm`, on a host with no network
- **THEN** it unpacks `catalog-opm/4.4/`, `catalog-opm/4.5/` and `catalog-opm/edge/` and writes a lock with three entries marked `local`, without contacting the registry or the Sigstore infrastructure

#### Scenario: Segment mismatch
- **WHEN** `--local catalog-opm@4.5=dir` names a tree whose manifest version is 4.4.5
- **THEN** `pull` exits 1 naming the segment and the version

#### Scenario: Offline rebuild from a lock
- **WHEN** a previous pull cached every blob and `pull --frozen site/.bundles/lock.json --offline` runs with no network
- **THEN** it unpacks the same trees and writes the same lock
