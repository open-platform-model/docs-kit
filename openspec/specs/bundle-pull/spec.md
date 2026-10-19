# bundle-pull Specification

## Purpose
How the site pulls docs bundles: the pull config, tag resolution, signature verification, the unpack layout, the cache and the lock.

## Requirements

### Requirement: Tab versions resolve from the registry's tags
For each tab, `pull` SHALL list the repository's tags, ignore `sha256-*` signature tags, select every minor tag (`<MAJOR>.<MINOR>`) at or above `from`, plus `edge` when enabled, resolve each to a digest, and refuse a selected tag whose build's `version` annotation is not in that minor or whose `project` annotation differs. A new minor SHALL appear with no change to the config. Source: DESIGN decisions 8 and 9.

#### Scenario: A new minor appears
- **WHEN** catalog_opm publishes its first 4.5 build and the site pulls with an unchanged config
- **THEN** the unpack directory gains `catalog-opm/4.5/` and the lock gains its entry

#### Scenario: Older minors are not shown
- **WHEN** the registry holds `4.3` and the tab's `from` is `4.4`
- **THEN** `pull` does not fetch `4.3`

### Requirement: The unpack layout and lock are fixed
`pull` SHALL unpack each bundle into `<out>/<project>/<segment>/` (segment `<MAJOR>.<MINOR>` or `edge`), remove every project or segment directory under `<out>` it did not write in this run, and write the lock (`docs.opmodel.dev/lock/v1`) with the fields and key order `docs/contracts.md` C7 fixes (validated against `schema/lock.cue`), every entry including the tab's placement `root`. The same resolution SHALL write byte-identical lock files.

#### Scenario: Stable lock
- **WHEN** `pull` runs twice and no tag moved in between
- **THEN** both runs write the same `lock.json` bytes

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

### Requirement: A site version unpacks and is replaced whole
`pull` SHALL unpack each docs bundle of site version `v` into `<out>/_versions/v/<project>/`, staging the whole version beside it, and SHALL replace `<out>/_versions/v/` only after every bundle of the version has verified, linted and passed the cross-bundle checks; on any failure the previous `<out>/_versions/v/` SHALL stay. The sweep SHALL remove every `_versions/` entry the config does not name.

#### Scenario: A refused version keeps the old one
- **WHEN** a pull of `v1.0` fails because two bundles write one page
- **THEN** `<out>/_versions/v1.0/` holds the bundles of the previous successful pull and the pull exits 2

### Requirement: One version's docs bundles never overlap
Across the docs bundles of one site version, `pull` SHALL fail with exit 2, naming the version, the path and both projects with their versions, when two pages of different bundles serve one URL, when a bundle has a page under a path another bundle owns, or when two bundles own overlapping paths.

#### Scenario: One page in two bundles
- **WHEN** cli and core both hold `start/_index.md` and neither owns it
- **THEN** `pull` exits 2 naming `v1.0`, `start/_index.md`, cli and core with their versions

#### Scenario: Two pages that serve one URL
- **WHEN** cli holds `guides/cli.md` and core holds `guides/cli/_index.md`
- **THEN** `pull` exits 2 naming `v1.0`, both paths, cli and core with their versions, and the URL `/docs/guides/cli/`

#### Scenario: A page under another bundle's owned directory
- **WHEN** core's bundle holds `reference/cli/extra.md` and cli owns `reference/cli/`
- **THEN** `pull` exits 2 naming `reference/cli/extra.md`, core and cli

### Requirement: The lock records each site version's bundles
The lock SHALL carry an optional `docs` list after `bundles` (and after `history` when present), one entry per docs bundle with `site`, `project`, `role`, `tag`, `repository`, `digest`, `version`, `revision`, `commit`, `dialect`, `builtBy`, `signer`, the anchor's `pins` and `dir`, in that key order; a local entry SHALL carry `local: true` and no `tag`, `repository`, `digest` or `signer`.

#### Scenario: Stable docs lock
- **WHEN** `pull` runs twice and no tag moved
- **THEN** both locks are byte-identical, including their `docs` entries

### Requirement: Pull takes a section bundle from its edge tag
For each section, `pull` SHALL resolve only the `edge` tag, verify it (C9, with the section's `repo`), require placement `kind: "section"` with the configured root, unpack and lint it into `<out>/<project>/edge/` as a tab segment, and lock it in `bundles` with `root` the section's root and `segment` `edge`. A section with no `edge` tag SHALL fail the pull naming the project, since the section would be empty.

#### Scenario: The enhancements section
- **WHEN** `sections: enhancements: {repo: "open-platform-model/enhancements", root: "/enhancements/"}` and `docs/enhancements` has `edge`
- **THEN** `pull` unpacks `<out>/enhancements/edge/` and the lock has an entry with root `/enhancements/` and segment `edge`

### Requirement: The pull config is validated against the embedded schema
`opm-docs pull --config <file>` SHALL validate the file against the embedded `#Pull` schema: a `registry` and a `signer` (issuer, workflow URL, allowed ref globs), beside `tabs`, `docs` and `sections`, each keyed by project, and `versions` keyed by site version (`v<MAJOR>.<MINOR>`).

#### Scenario: A config with a registry and a signer
- **WHEN** `opm-docs pull --config <file>` runs on a file that holds a `registry`, a `signer` (issuer, workflow URL, allowed ref globs) and `tabs`, `docs` and `sections` keyed by project
- **THEN** the file is validated against the embedded `#Pull` schema

### Requirement: A tab entry names its signing repository, root and oldest minor
`tabs` SHALL be keyed by project, each with the `repo` allowed to sign it, its URL `root`, the oldest minor `from` and whether `edge` is shown. A tab bundle SHALL have placement `kind: "tab"`.

#### Scenario: Missing owner repository
- **WHEN** a tab entry has no `repo`
- **THEN** `pull` exits 1 naming the project and the missing field, before any network call

### Requirement: A docs entry names its signing repository and its site versions
`docs` SHALL be keyed by project, each with the `repo` allowed to sign it. `versions` SHALL be keyed by site version (`v<MAJOR>.<MINOR>`), each with an `anchor` (project and tag), the `pinned` projects and the projects pulled by their own `tags`. A docs bundle SHALL have placement `kind: "docs"`.

#### Scenario: A site version with an anchor
- **WHEN** `versions` holds `v1.0` with an `anchor` (project and tag), `pinned` projects and projects pulled by their own `tags`
- **THEN** each of those projects is a key of `docs`, whose entry has the `repo` allowed to sign it

### Requirement: A section entry names its signing repository and root
`sections` SHALL be keyed by project, each with the `repo` allowed to sign it and its `root`. A section bundle SHALL have placement `kind: "section"` with the configured root.

#### Scenario: A section entry
- **WHEN** `sections` holds `enhancements` with a `repo` and a `root`
- **THEN** a section bundle of `enhancements` has placement `kind: "section"` with that `root`

### Requirement: Pull refuses an inconsistent config before any network call
Before any network call `pull` SHALL refuse, with exit 1 naming the version and project, a project in a version that is not a key of `docs`, a project named twice in one version, and a project that appears under more than one of `tabs`, `docs` and `sections`.

#### Scenario: A version names an undeclared project
- **WHEN** `versions."v1.0".pinned` lists `core` and `docs` has no `core`
- **THEN** `pull` exits 1 naming `v1.0` and `core`, before any network call

#### Scenario: A project in two roles
- **WHEN** `enhancements` is both under `docs` and under `sections`
- **THEN** `pull` exits 1 naming `enhancements`, before any network call

#### Scenario: A project named twice in a version
- **WHEN** `versions."v1.0"` names `core` twice
- **THEN** `pull` exits 1 naming `v1.0` and `core`, before any network call

### Requirement: A bundle is fetched and unpacked only after its signature verifies
`pull` SHALL verify each resolved digest's Sigstore bundle. Only then SHALL it fetch the layer by digest, unpack it with the bundle-format guards and lint it in bundle mode. Any failure SHALL fail the whole pull, naming the project, tag and digest.

#### Scenario: Unsigned edge
- **WHEN** `edge` resolves to a digest with no signature
- **THEN** `pull` exits 2 naming `edge` and the digest

### Requirement: The Sigstore bundle is verified against the public-good trusted root
`pull` SHALL verify each resolved digest's Sigstore bundle with sigstore-go against the public-good trusted root, requiring a signed certificate timestamp, a transparency-log entry and an observer timestamp, and the artifact digest equal to the manifest digest.

#### Scenario: A bundle whose digest differs from the manifest
- **WHEN** a Sigstore bundle names an artifact digest that differs from the manifest digest
- **THEN** `pull` fails the whole pull, naming the project, tag and digest

### Requirement: The signer identity must match the config
`pull` SHALL verify each resolved digest's Sigstore bundle, requiring issuer `https://token.actions.githubusercontent.com`, a SAN equal to the signer workflow URL at a ref matching an allowed glob, Source Repository URI `https://github.com/<repo>` of the tab, and Source Repository Ref `refs/heads/main`.

#### Scenario: A bundle signed by another repository
- **WHEN** a digest under `docs/catalog-opm` carries a valid signature whose Source Repository URI is `https://github.com/open-platform-model/cli`
- **THEN** `pull` exits 2 naming `catalog-opm`, the tag, the digest and the repository mismatch, and unpacks nothing for it

#### Scenario: A signer workflow at a ref no glob allows
- **WHEN** the certificate SAN is the signer workflow URL at a ref that matches no allowed glob
- **THEN** `pull` fails the whole pull, naming the project, tag and digest

### Requirement: A local lock entry carries only local data
A local entry of the lock SHALL carry `local: true` and no `tag`, `repository`, `digest` or `signer`.

#### Scenario: A local entry
- **WHEN** `pull --local catalog-opm@4.5=a` writes the lock
- **THEN** the `catalog-opm` entry carries `local: true` and no `tag`, `repository`, `digest` or `signer`

### Requirement: The lock is serialized in one form
`pull` SHALL serialize the lock as JSON with two-space indent and a trailing newline, entries sorted by project then segment (minors ascending, `edge` last), with no timestamps.

#### Scenario: Entries sorted by segment
- **WHEN** a project has the segments `edge`, `4.5` and `4.4`
- **THEN** its lock entries read `4.4`, `4.5`, `edge`, and the lock holds no timestamp

### Requirement: A frozen pull fetches exactly the locked digests
With `--frozen <lock>`, `pull` SHALL fetch exactly the digests the lock names, still verifying and linting.

#### Scenario: Frozen pull of a lock
- **WHEN** `pull --frozen <lock>` runs
- **THEN** it fetches exactly the digests the lock names, and still verifies and lints them

### Requirement: An offline pull uses only the cache
With `--offline` (only beside `--frozen`), `pull` SHALL use only the cache and fail naming the first missing digest.

#### Scenario: Offline rebuild from a lock
- **WHEN** a previous pull cached every blob and `pull --frozen site/.bundles/lock.json --offline` runs with no network
- **THEN** it unpacks the same trees and writes the same lock

#### Scenario: A digest missing from the cache
- **WHEN** `pull --frozen <lock> --offline` runs and the cache lacks a digest the lock names
- **THEN** `pull` fails naming the first missing digest

### Requirement: A local pull takes a segment from a local tree
With `--local <project>@<segment>=<dir>`, repeatable, `pull` SHALL take that segment of that project from a local bundle tree, refusing a segment that differs from the one the tree's manifest implies. It SHALL skip registry resolution for every project named by `--local`, and mark each local lock entry `"local": true`.

#### Scenario: An all-local pull needs no network
- **WHEN** `pull` runs with `--local catalog-opm@4.4=a --local catalog-opm@4.5=b --local catalog-opm@edge=c` and every tab is `catalog-opm`, on a host with no network
- **THEN** it unpacks `catalog-opm/4.4/`, `catalog-opm/4.5/` and `catalog-opm/edge/` and writes a lock with three entries marked `local`, without contacting the registry or the Sigstore infrastructure

#### Scenario: Segment mismatch
- **WHEN** `--local catalog-opm@4.5=dir` names a tree whose manifest version is 4.4.5
- **THEN** `pull` exits 1 naming the segment and the version

### Requirement: A local entry skips signature verification
For local entries `pull` SHALL skip signature verification and the Sigstore trusted root, and still validate, guard and lint them.

#### Scenario: A local tree is still linted
- **WHEN** `pull --local catalog-opm@4.5=<dir>` names a tree
- **THEN** `pull` validates, guards and lints it, and neither verifies a signature nor loads the Sigstore trusted root

### Requirement: A tab without an edge tag is skipped and a tab without a build fails
`pull` SHALL skip a tab's `edge` segment with a warning when no `edge` tag exists, and SHALL fail naming the project when a tab gets no minor at or above `from` and no `edge` build.

#### Scenario: A project before its first edge push
- **WHEN** a tab with `edge: true` has `4.4` but no `edge` tag
- **THEN** `pull` unpacks `4.4`, warns that `edge` is missing, and writes a lock with no edge entry

#### Scenario: A tab with no build
- **WHEN** a tab gets no minor at or above `from` and no `edge` build
- **THEN** `pull` fails naming the project

### Requirement: A tab bundle must declare the tab placement
`pull` SHALL refuse a bundle whose `manifest.json` placement is not `kind: "tab"` with the tab's configured `root`.

#### Scenario: Placement mismatch
- **WHEN** a bundle under `docs/catalog-opm` declares `placement.root` `/catalogs/other/`
- **THEN** `pull` exits 2 naming the project, the digest and both roots

### Requirement: A frozen pull refuses a lock from another config
`pull` SHALL refuse `--frozen` with a lock whose `config` digest differs from the current config's.

#### Scenario: Frozen lock from another config
- **WHEN** `pull --frozen lock.json` runs after `bundles.cue` changed a tab's `repo`
- **THEN** `pull` exits 1 naming both config digests

### Requirement: An offline pull verifies with the cached trusted root
Under `--offline`, `pull` SHALL verify with the cached trusted root without refreshing it, warning when its TUF metadata has expired and failing when none is cached.

#### Scenario: Expired trusted root offline
- **WHEN** `pull --frozen <lock> --offline` runs and the cached trusted root's TUF metadata has expired
- **THEN** `pull` verifies with the cached trusted root without refreshing it, and warns

#### Scenario: No cached trusted root offline
- **WHEN** `pull --frozen <lock> --offline` runs and no trusted root is cached
- **THEN** `pull` fails

### Requirement: A layer over 32 MiB is refused
`pull` SHALL refuse a layer whose descriptor size exceeds 32 MiB before fetching it.

#### Scenario: An oversized layer
- **WHEN** a layer descriptor has a size over 32 MiB
- **THEN** `pull` refuses the layer before fetching it

### Requirement: The anchor of a site version is resolved first
For each site version, `pull` SHALL resolve, verify (C9, with the docs project's `repo`), unpack and lint the anchor first, requiring the resolved build to be in the anchor tag's line. Source: DESIGN decisions 9 and 10.

#### Scenario: An anchor outside its tag's line
- **WHEN** the anchor tag resolves to a build that is not in the anchor tag's line
- **THEN** `pull` refuses that build

### Requirement: Pinned projects resolve at the anchor's pins
After the anchor, `pull` SHALL resolve each pinned project at the release tag equal to the anchor manifest's pin for it, requiring the resolved build's version to equal the pin, and SHALL resolve each `tags` project at its tag. Source: DESIGN decisions 9 and 10.

#### Scenario: Pins choose the docs
- **WHEN** `v1.0` anchors on `cli` tag `1.0`, which resolves to 1.0.0-beta.6 pinning core `2.0.0-beta.1`
- **THEN** `pull` resolves `docs/core` tag `2.0.0-beta.1` and unpacks it under `_versions/v1.0/core/`

#### Scenario: A docs revision of a pinned release
- **WHEN** core publishes `2.0.0-beta.1.1` and `2.0.0-beta.1` moves to it
- **THEN** the next pull of `v1.0` takes revision 1 of core with no config change

### Requirement: A missing pin or pinned release fails the pull
A pinned project with no pin in the anchor's manifest, or a pin whose release tag does not exist, SHALL fail the pull with exit 2 naming the anchor's version, the project, the pinned version and the release to publish. Source: DESIGN decisions 9 and 10.

#### Scenario: A pinned release without a bundle
- **WHEN** the anchor pins `opm-operator` `1.0.0-beta.5` and `docs/opm-operator` has no tag `1.0.0-beta.5`
- **THEN** `pull` exits 2 naming the cli version, `opm-operator`, `1.0.0-beta.5` and the release to publish

#### Scenario: A pinned project with no pin
- **WHEN** the anchor's manifest has no pin for a `pinned` project
- **THEN** `pull` exits 2 naming the anchor's version, the project, the pinned version and the release to publish

### Requirement: Docs bundle paths compare by the URL Hugo serves
`pull` SHALL compare paths by the URL Hugo serves: `.md` dropped, then a trailing `_index` or `index`, so `cli.md` and `cli/_index.md` are one page, and an owned page `x.md` holds everything under `x/` as an owned directory `x/` does.

#### Scenario: An owned page holds its directory
- **WHEN** core owns the page `reference/cli.md` and cli's bundle holds `reference/cli/extra.md`
- **THEN** `pull` exits 2 naming `reference/cli/extra.md`, core and cli

### Requirement: The docs list of the lock is sorted
The `docs` entries SHALL be sorted by site version, then role (`anchor`, `pinned`, `tag`), then project.

#### Scenario: Docs entries in order
- **WHEN** a lock holds `docs` entries for `v1.0` with the roles `tag`, `pinned` and `anchor`
- **THEN** the entries read `anchor`, `pinned`, `tag`

### Requirement: The docs list is omitted without versions
The `docs` key SHALL be omitted when the config has no `versions`, and the schema id SHALL stay `docs.opmodel.dev/lock/v1`.

#### Scenario: A config without versions
- **WHEN** the config has no `versions`
- **THEN** the lock has no `docs` key and its schema id is `docs.opmodel.dev/lock/v1`

### Requirement: A local docs bundle replaces a project of a site version
`--local <project>@<site-version>=<dir>` SHALL take that docs project of that site version from a local tree without a signature check, applying every other check but the tag's line (a local tree may be an edge build); a local anchor's pins SHALL choose the pinned projects, and a local pinned tree's version SHALL equal its pin or the pull exits 1.

#### Scenario: Author preview of the cli reference
- **WHEN** `pull --local cli@v1.0=out/cli` runs and the tree pins core `2.0.0-beta.1`
- **THEN** `_versions/v1.0/cli/` is the local tree, core is pulled at `2.0.0-beta.1`, and the lock marks cli `local`

### Requirement: A local docs bundle must name a pulled project
A `--local` naming a site version the config lacks, or a project the version does not pull, SHALL exit 1.

#### Scenario: A local bundle for an unknown site version
- **WHEN** `pull --local cli@v9.9=out/cli` runs and the config has no `v9.9`
- **THEN** `pull` exits 1

### Requirement: A frozen pull fetches each docs entry by its digest
Under `--frozen`, each `docs` entry SHALL be fetched by its digest and SHALL be refused with exit 1 when its site version, project or role is not in the config with that role, when its repository is not `<registry>/<project>`, or when it is local.

#### Scenario: Frozen lock with a changed role
- **WHEN** the config pulls `core` as `pinned` in `v1.0` and the frozen lock, written for the same config, names it with role `tag`
- **THEN** `pull --frozen` exits 1 naming `v1.0`, `core` and both roles

#### Scenario: Frozen offline rebuild
- **WHEN** `pull --frozen <lock> --offline` runs with every blob in the cache, after the anchor's tag has moved
- **THEN** it unpacks the locked digests and writes a lock byte-identical to the frozen one

### Requirement: A frozen pull refuses a docs entry whose tag or pin changed
Under `--frozen`, a `docs` entry SHALL be refused with exit 1 when an anchor or `tags` entry's tag differs from the config's, or when a pinned entry's version or tag differs from the locked anchor's pin.

#### Scenario: A frozen anchor with another tag
- **WHEN** the frozen lock names an anchor entry whose tag differs from the config's
- **THEN** `pull --frozen` exits 1

### Requirement: A frozen pull needs every configured docs project
A configured project of a site version that the frozen lock does not name (and no `--local` supplies) SHALL exit 1.

#### Scenario: A configured project missing from the lock
- **WHEN** the config pulls `core` in `v1.0`, the frozen lock does not name it and no `--local` supplies it
- **THEN** `pull --frozen` exits 1
