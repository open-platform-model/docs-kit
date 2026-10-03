## MODIFIED Requirements

### Requirement: The pull config names tabs, owners and the trusted signer
`opm-docs pull --config <file>` SHALL validate the file against the embedded `#Pull` schema: a `registry`, a `signer` (issuer, workflow URL, allowed ref globs), `tabs` keyed by project, each with the `repo` allowed to sign it, its URL `root`, the oldest minor `from` and whether `edge` is shown, `docs` keyed by project, each with the `repo` allowed to sign it, and `versions` keyed by site version (`v<MAJOR>.<MINOR>`), each with an `anchor` (project and tag), the `pinned` projects and the projects pulled by their own `tags`. Before any network call it SHALL refuse, with exit 1 naming the version and project, a project in a version that is not a key of `docs`, a project named twice in one version, and a project that is both a tab and a docs project. A tab bundle SHALL have placement `kind: "tab"` and a docs bundle `kind: "docs"`.

#### Scenario: Missing owner repository
- **WHEN** a tab entry has no `repo`
- **THEN** `pull` exits 1 naming the project and the missing field, before any network call

#### Scenario: A version names an undeclared project
- **WHEN** `versions."v1.0".pinned` lists `core` and `docs` has no `core`
- **THEN** `pull` exits 1 naming `v1.0` and `core`, before any network call

## ADDED Requirements

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
