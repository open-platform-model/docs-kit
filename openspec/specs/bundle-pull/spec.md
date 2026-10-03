# bundle-pull Specification

## Purpose
How the site pulls docs bundles: the pull config, tag resolution, signature verification, the unpack layout, the cache and the lock.

## Requirements

### Requirement: The pull config names tabs, owners and the trusted signer
`opm-docs pull --config <file>` SHALL validate the file against the embedded `#Pull` schema: a `registry`, a `signer` (issuer, workflow URL, allowed ref globs), `tabs` keyed by project, each with the `repo` allowed to sign it, its URL `root`, the oldest minor `from` and whether `edge` is shown, `docs` keyed by project, each with the `repo` allowed to sign it, and `versions` keyed by site version (`v<MAJOR>.<MINOR>`), each with an `anchor` (project and tag), the `pinned` projects and the projects pulled by their own `tags`. Before any network call it SHALL refuse, with exit 1 naming the version and project, a project in a version that is not a key of `docs`, a project named twice in one version, and a project that is both a tab and a docs project. A tab bundle SHALL have placement `kind: "tab"` and a docs bundle `kind: "docs"`.

#### Scenario: Missing owner repository
- **WHEN** a tab entry has no `repo`
- **THEN** `pull` exits 1 naming the project and the missing field, before any network call

#### Scenario: A version names an undeclared project
- **WHEN** `versions."v1.0".pinned` lists `core` and `docs` has no `core`
- **THEN** `pull` exits 1 naming `v1.0` and `core`, before any network call

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

### Requirement: A refused history or lock swaps nothing in
`pull` SHALL stage every unpacked and linted segment, compute and encode every tab's history and encode the lock before it swaps any segment into place, so that a refusal at any of them leaves the previous segment trees, `history.json` files and lock unchanged.

#### Scenario: A bundle's data refused by the history
- **WHEN** a previous pull succeeded and the next pull's 4.6 bundle carries a member page that is not `<kind>s/<name>`
- **THEN** the pull exits 2 naming `catalog-opm 4.6 data/catalog.json`, and every file under `<out>/` is as the previous pull left it

### Requirement: A site version resolves from its anchor and the anchor's pins
For each site version, `pull` SHALL resolve, verify (C9, with the docs project's `repo`), unpack and lint the anchor first, requiring the resolved build to be in the anchor tag's line; SHALL then resolve each pinned project at the release tag equal to the anchor manifest's pin for it, requiring the resolved build's version to equal the pin; and SHALL resolve each `tags` project at its tag. A pinned project with no pin in the anchor's manifest, or a pin whose release tag does not exist, SHALL fail the pull with exit 2 naming the anchor's version, the project, the pinned version and the release to publish. Source: DESIGN decisions 9 and 10.

#### Scenario: Pins choose the docs
- **WHEN** `v1.0` anchors on `cli` tag `1.0`, which resolves to 1.0.0-beta.6 pinning core `2.0.0-beta.1`
- **THEN** `pull` resolves `docs/core` tag `2.0.0-beta.1` and unpacks it under `_versions/v1.0/core/`

#### Scenario: A docs revision of a pinned release
- **WHEN** core publishes `2.0.0-beta.1.1` and `2.0.0-beta.1` moves to it
- **THEN** the next pull of `v1.0` takes revision 1 of core with no config change

#### Scenario: A pinned release without a bundle
- **WHEN** the anchor pins `opm-operator` `1.0.0-beta.5` and `docs/opm-operator` has no tag `1.0.0-beta.5`
- **THEN** `pull` exits 2 naming the cli version, `opm-operator`, `1.0.0-beta.5` and the release to publish

### Requirement: A site version unpacks and is replaced whole
`pull` SHALL unpack each docs bundle of site version `v` into `<out>/_versions/v/<project>/`, staging the whole version beside it, and SHALL replace `<out>/_versions/v/` only after every bundle of the version has verified, linted and passed the cross-bundle checks; on any failure the previous `<out>/_versions/v/` SHALL stay. The sweep SHALL remove every `_versions/` entry the config does not name.

#### Scenario: A refused version keeps the old one
- **WHEN** a pull of `v1.0` fails because two bundles write one page
- **THEN** `<out>/_versions/v1.0/` holds the bundles of the previous successful pull and the pull exits 2

### Requirement: One version's docs bundles never overlap
Across the docs bundles of one site version, `pull` SHALL fail with exit 2, naming the version, the path and both projects with their versions, when a `content/` path is in two bundles, when a bundle has a page under a path another bundle owns, or when two bundles own overlapping paths.

#### Scenario: One page in two bundles
- **WHEN** cli and core both hold `start/_index.md` and neither owns it
- **THEN** `pull` exits 2 naming `v1.0`, `start/_index.md`, cli and core with their versions

#### Scenario: A page under another bundle's owned directory
- **WHEN** core's bundle holds `reference/cli/extra.md` and cli owns `reference/cli/`
- **THEN** `pull` exits 2 naming `reference/cli/extra.md`, core and cli

### Requirement: The lock records each site version's bundles
The lock SHALL carry an optional `docs` list after `bundles` (and after `history` when present), one entry per docs bundle with `site`, `project`, `role`, `tag`, `repository`, `digest`, `version`, `revision`, `commit`, `dialect`, `builtBy`, `signer`, the anchor's `pins` and `dir`, in that key order, sorted by site version, then role (`anchor`, `pinned`, `tag`), then project; a local entry SHALL carry `local: true` and no `tag`, `repository`, `digest` or `signer`. The key SHALL be omitted when the config has no `versions`, and the schema id SHALL stay `docs.opmodel.dev/lock/v1`.

#### Scenario: Stable docs lock
- **WHEN** `pull` runs twice and no tag moved
- **THEN** both locks are byte-identical, including their `docs` entries

### Requirement: Local and frozen pulls of docs bundles
`--local <project>@<site-version>=<dir>` SHALL take that docs project of that site version from a local tree without a signature check, applying every other check but the tag's line (a local tree may be an edge build); a local anchor's pins SHALL choose the pinned projects, and a local pinned tree's version SHALL equal its pin or the pull exits 1. A `--local` naming a site version the config lacks, or a project the version does not pull, SHALL exit 1. Under `--frozen`, each `docs` entry SHALL be fetched by its digest and SHALL be refused with exit 1 when its site version, project or role is not in the config with that role, when its repository is not `<registry>/<project>`, when it is local, when an anchor or `tags` entry's tag differs from the config's, or when a pinned entry's version or tag differs from the locked anchor's pin. A configured project of a site version that the frozen lock does not name (and no `--local` supplies) SHALL exit 1.

#### Scenario: Author preview of the cli reference
- **WHEN** `pull --local cli@v1.0=out/cli` runs and the tree pins core `2.0.0-beta.1`
- **THEN** `_versions/v1.0/cli/` is the local tree, core is pulled at `2.0.0-beta.1`, and the lock marks cli `local`

#### Scenario: Frozen lock with a changed role
- **WHEN** the config pulls `core` as `pinned` in `v1.0` and the frozen lock, written for the same config, names it with role `tag`
- **THEN** `pull --frozen` exits 1 naming `v1.0`, `core` and both roles

#### Scenario: Frozen offline rebuild
- **WHEN** `pull --frozen <lock> --offline` runs with every blob in the cache, after the anchor's tag has moved
- **THEN** it unpacks the locked digests and writes a lock byte-identical to the frozen one
