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
`pull` SHALL unpack each bundle into `<out>/<project>/<segment>/` (segment `<MAJOR>.<MINOR>` or `edge`), remove every project or segment directory under `<out>` it did not write in this run, and write the lock (`docs.opmodel.dev/lock/v1`) with the fields and key order `docs/contracts.md` C7 fixes (validated against `schema/lock.cue`), every entry including the tab's placement `root`, a local entry carrying `local: true` and no `tag`, `repository`, `digest` or `signer`; serialized as JSON with two-space indent and a trailing newline, entries sorted by project then segment (minors ascending, `edge` last), with no timestamps. The same resolution SHALL write byte-identical lock files.

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

### Requirement: Pull handles missing and mismatched versions
`pull` SHALL skip a tab's `edge` segment with a warning when no `edge` tag exists; SHALL fail naming the project when a tab gets no minor at or above `from` and no `edge` build; SHALL refuse a bundle whose `manifest.json` placement is not `kind: "tab"` with the tab's configured `root`; SHALL refuse `--frozen` with a lock whose `config` digest differs from the current config's; under `--offline` SHALL verify with the cached trusted root without refreshing it, warning when its TUF metadata has expired and failing when none is cached; and SHALL refuse a layer whose descriptor size exceeds 32 MiB before fetching it.

#### Scenario: A project before its first edge push
- **WHEN** a tab with `edge: true` has `4.4` but no `edge` tag
- **THEN** `pull` unpacks `4.4`, warns that `edge` is missing, and writes a lock with no edge entry

#### Scenario: Placement mismatch
- **WHEN** a bundle under `docs/catalog-opm` declares `placement.root` `/catalogs/other/`
- **THEN** `pull` exits 2 naming the project, the digest and both roots

#### Scenario: Frozen lock from another config
- **WHEN** `pull --frozen lock.json` runs after `bundles.cue` changed a tab's `repo`
- **THEN** `pull` exits 1 naming both config digests
