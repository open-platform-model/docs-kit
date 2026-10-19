## REMOVED Requirements

### Requirement: docs-kit.cue configures the bundles a repository builds
**Reason**: Split into four requirements that each state one behavior.
**Migration**: See "docs-kit.cue is validated before any extraction", "docs-kit.cue holds bundles with sources", "Extractor sources accept a citation policy", "A markdown source selects files by include and exclude".

### Requirement: build writes one bundle tree per project
**Reason**: Split into six requirements that each state one behavior.
**Migration**: See "build writes one bundle tree per selected project", "build --release derives the version from the tag", "A release build agrees with the catalog version", "build --source resolves sources against one directory", "A missing markdown dir depends on where the config came from", "A build from a dirty work tree is recorded and not pushed".

### Requirement: serve previews a repository's bundles
**Reason**: Split into four requirements that each state one behavior.
**Migration**: See "serve builds edge bundles into a temporary directory", "serve serves the bundles on its embedded Hugo site", "serve rebuilds a bundle when its sources change", "serve --site previews in the real site".

## ADDED Requirements

### Requirement: docs-kit.cue is validated before any extraction
`build` and `check` SHALL read `docs-kit.cue` (or `--config`) and validate it against the embedded `#Config` schema before any extraction. A `package` clause in the file SHALL be optional and ignored.

#### Scenario: Package clause ignored
- **WHEN** one `docs-kit.cue` starts with `package docs` and another has no package clause, with the same fields
- **THEN** both validate and configure the same bundles

#### Scenario: A misspelled key is refused
- **WHEN** `docs-kit.cue` holds `bundles: "catalog-opm": {placment: ...}`
- **THEN** `opm-docs build` exits 1 naming the field `placment` and the file, before loading any CUE module

### Requirement: docs-kit.cue holds bundles with sources
`docs-kit.cue` SHALL hold `bundles`, keyed by project, each with a `placement`, a `version` (`from: "tag"` and a tag `prefix`), at least one source of a kind the tool registers (`docs/contracts.md` C6 lists them, each with its options), and optionally `pins`. A bundle SHALL hold at most one source of each extractor kind.

#### Scenario: An unknown source kind
- **WHEN** a source has `kind: "javadoc"`
- **THEN** `build` exits 1 naming the kind and the kinds this `opm-docs` registers

#### Scenario: Two sources of one extractor kind
- **WHEN** a bundle holds two `cue-catalog` sources
- **THEN** the config is refused, because a bundle holds at most one source of each extractor kind

### Requirement: Extractor sources accept a citation policy
Every extractor source SHALL accept `citations` (`"strip"`, the default, or `"link"`).

#### Scenario: Citations linked
- **WHEN** a source with `citations: "link"` documents a comment that names `enhancement 0010`
- **THEN** the page holds `[enhancement 0010](/enhancements/0010/)`

### Requirement: A markdown source selects files by include and exclude
A `markdown` source SHALL accept `include` and `exclude`, lists of patterns relative to its `dir` (a glob, or a directory ending `/` matching every file under it), copying a file that matches some `include` (all files when it is absent) and no `exclude`.

#### Scenario: Include selects one page
- **WHEN** a `markdown` source has `dir: "docs/site"` and `include: ["reference/operator-resources.md"]`
- **THEN** only that page is copied from `docs/site`

#### Scenario: Exclude a committed generated directory
- **WHEN** a `markdown` source has `dir: "docs/site"` and `exclude: ["reference/cli/"]`
- **THEN** every page of `docs/site` except those under `reference/cli/` is copied

### Requirement: build writes one bundle tree per selected project
`opm-docs build` SHALL write `out/<project>/` (or under `--out`) for every project in the config, or only those named with `--project`. Without `--release` it SHALL build an edge bundle.

#### Scenario: An edge build
- **WHEN** `opm-docs build --project catalog-opm` runs without `--release`
- **THEN** it writes `out/catalog-opm/` as an edge bundle

### Requirement: build --release derives the version from the tag
With `--release <tag>` `opm-docs build` SHALL derive the version by removing the project's tag prefix and SHALL refuse a tag without that prefix or whose remainder is not SemVer.

#### Scenario: Release build from a tag
- **WHEN** `opm-docs build --project catalog-opm --release opm-v4.4.5 --source src` runs on a checkout of that tag
- **THEN** `out/catalog-opm/manifest.json` has version `4.4.5`, revision 0, `source.ref` `opm-v4.4.5` and `source.commit` the tag's commit

#### Scenario: Wrong prefix
- **WHEN** `--release v4.4.5` is passed for project `catalog-opm`, whose prefix is `opm-v`
- **THEN** `build` exits 1 naming the tag and the expected prefix

### Requirement: A release build agrees with the catalog version
With `--release <tag>` and a `cue-catalog` source, `opm-docs build` SHALL refuse with exit 2 when the derived version differs from the catalog's `metadata.version`, naming both values; this holds for a docs revision too, which `revise` builds with `--release`.

#### Scenario: Tag and catalog version differ
- **WHEN** `build --release opm-v4.6.0` runs on a tree whose catalog declares `metadata.version: "4.5.0"`
- **THEN** `build` exits 2 naming `4.6.0` and `4.5.0`, and writes no `manifest.json`

### Requirement: build --source resolves sources against one directory
With `--source <dir>` `opm-docs build` SHALL take `docs-kit.cue` from that directory when present, else from the current directory, and SHALL resolve every source (each `cue-catalog` `module` and `markdown` `dir`) and all git history against that directory, wherever the config came from.

#### Scenario: Release older than docs-kit adoption
- **WHEN** the tree at `src/` has no `docs-kit.cue` and the current directory has one
- **THEN** `build --source src` uses the current directory's `docs-kit.cue`, and extracts the `cue-catalog` module and the `markdown` dir from `src/`, not from the current directory

### Requirement: A missing markdown dir depends on where the config came from
A `markdown` dir that does not exist SHALL yield no pages when the config came from outside the `--source` tree, and SHALL fail the build otherwise.

#### Scenario: A missing markdown dir in a normal build
- **WHEN** `docs-kit.cue` names `docs/catalogs/opm` and the same tree has no such directory
- **THEN** `build` exits 2 naming the dir and the config file

#### Scenario: A missing markdown dir in a backfill
- **WHEN** the config came from outside the `--source` tree and a `markdown` dir does not exist in it
- **THEN** the build yields no pages for that dir

### Requirement: A build from a dirty work tree is recorded and not pushed
A build from a work tree with uncommitted changes SHALL record `source.dirty: true`, and `push` SHALL refuse such a bundle.

#### Scenario: A dirty work tree
- **WHEN** `opm-docs build` runs in a work tree with uncommitted changes
- **THEN** the manifest records `source.dirty: true`, and `push` refuses the bundle

### Requirement: serve builds edge bundles into a temporary directory
`opm-docs serve` SHALL build the selected projects as edge bundles (a dirty work tree allowed) into a temporary directory it removes on exit, each with `build`'s own extractors, repository commands and bundle-mode dialect lint, so a page `build` refuses is never served.

#### Scenario: A page that build refuses
- **WHEN** `build` refuses a page of a selected project
- **THEN** `opm-docs serve` does not serve that page

### Requirement: serve serves the bundles on its embedded Hugo site
By default `opm-docs serve` SHALL serve the bundles on its embedded minimal Hugo site with the host's `hugo`, listening on `127.0.0.1` only (refusing, with exit 1, a missing `hugo` or one older than 0.146.0, naming the version found), mounting each tab bundle at `<root>edge/`, each docs bundle at `/docs/` and any other placement at its root.

#### Scenario: No hugo
- **WHEN** `opm-docs serve` runs without `hugo` on `PATH`
- **THEN** it exits 1 naming `hugo` and the version it needs

### Requirement: serve rebuilds a bundle when its sources change
By default (without `--site`), while serving, `opm-docs serve` SHALL start a rebuild of a bundle within two seconds of a change to a file under its sources. A build failure while watching SHALL be printed and SHALL keep the last good bundle served.

#### Scenario: Preview an authored page
- **WHEN** an author in the cli repository runs `opm-docs serve` and edits `docs/site/start/install.md`
- **THEN** the page at `/docs/start/install/` on the printed local URL shows the edit after the rebuild

#### Scenario: A failed rebuild keeps the last good page
- **WHEN** the author saves a page that breaks the page dialect while `opm-docs serve` runs
- **THEN** the violations print as `build` prints them, and the page keeps showing the last build that passed

### Requirement: serve --site previews in the real site
With `--site <dir>`, `opm-docs serve` SHALL run `task bundles:pull` and then `task serve` in that directory with `OPM_BUNDLES_LOCAL` naming each built tree (a tab or section as `<project>@edge`, a docs bundle as `<project>@<--site-version>`), without watching.

#### Scenario: Preview in the real site
- **WHEN** `opm-docs serve --site ../opmodel.dev --site-version v1.0` runs in the cli repository
- **THEN** it runs the site's `task bundles:pull` and `task serve` with `OPM_BUNDLES_LOCAL="cli@v1.0=<built tree>"`

#### Scenario: Docs bundle without a site version
- **WHEN** `opm-docs serve --site ../opmodel.dev` runs for a docs bundle without `--site-version`
- **THEN** it exits 1 naming `--site-version`
