# opm-docs-cli Specification

## Purpose
The `opm-docs` command set, its configuration file `docs-kit.cue`, and its exit codes and messages.

## Requirements

### Requirement: The command set
`opm-docs` SHALL provide the commands `build`, `lint`, `check`, `push`, `promote`, `pull`, `revise`, `serve` and `version`, with the flags `docs/contracts.md` lists under "Commands". It SHALL exit 0 on success, 1 on a usage error (unknown command or flag, missing argument, unreadable or invalid config) and 2 on an execution error (lint violations, a refused build, a registry or signature failure). Every error message SHALL name what failed and the fix.

#### Scenario: Version
- **WHEN** `opm-docs version` runs
- **THEN** it prints `opm-docs <version>` and exits 0

#### Scenario: Unknown flag
- **WHEN** `opm-docs build --nope` runs
- **THEN** it exits 1 naming `--nope`

### Requirement: docs-kit.cue configures the bundles a repository builds
`build` and `check` SHALL read `docs-kit.cue` (or `--config`) and validate it against the embedded `#Config` schema before any extraction. A `package` clause in the file SHALL be optional and ignored. The file SHALL hold `bundles`, keyed by project, each with a `placement`, a `version` (`from: "tag"` and a tag `prefix`), at least one source of a kind the tool registers (`docs/contracts.md` C6 lists them, each with its options), and optionally `pins`. Every extractor source SHALL accept `citations` (`"strip"`, the default, or `"link"`); a `markdown` source SHALL accept `include` and `exclude`, lists of patterns relative to its `dir` (a glob, or a directory ending `/` matching every file under it), copying a file that matches some `include` (all files when it is absent) and no `exclude`. A bundle SHALL hold at most one source of each extractor kind.

#### Scenario: Package clause ignored
- **WHEN** one `docs-kit.cue` starts with `package docs` and another has no package clause, with the same fields
- **THEN** both validate and configure the same bundles

#### Scenario: A misspelled key is refused
- **WHEN** `docs-kit.cue` holds `bundles: "catalog-opm": {placment: ...}`
- **THEN** `opm-docs build` exits 1 naming the field `placment` and the file, before loading any CUE module

#### Scenario: An unknown source kind
- **WHEN** a source has `kind: "javadoc"`
- **THEN** `build` exits 1 naming the kind and the kinds this `opm-docs` registers

#### Scenario: Include selects one page
- **WHEN** a `markdown` source has `dir: "docs/site"` and `include: ["reference/operator-resources.md"]`
- **THEN** only that page is copied from `docs/site`

#### Scenario: Exclude a committed generated directory
- **WHEN** a `markdown` source has `dir: "docs/site"` and `exclude: ["reference/cli/"]`
- **THEN** every page of `docs/site` except those under `reference/cli/` is copied

#### Scenario: Citations linked
- **WHEN** a source with `citations: "link"` documents a comment citing `0010:D28`
- **THEN** the page holds `[0010:D28](/enhancements/0010/decisions/)`

### Requirement: build writes one bundle tree per project
`opm-docs build` SHALL write `out/<project>/` (or under `--out`) for every project in the config, or only those named with `--project`. With `--release <tag>` it SHALL derive the version by removing the project's tag prefix and SHALL refuse a tag without that prefix or whose remainder is not SemVer. With `--release <tag>` and a `cue-catalog` source, it SHALL refuse with exit 2 when the derived version differs from the catalog's `metadata.version`, naming both values; this holds for a docs revision too, which `revise` builds with `--release`. Without `--release` it SHALL build an edge bundle. With `--source <dir>` it SHALL take `docs-kit.cue` from that directory when present, else from the current directory, and SHALL resolve every source (each `cue-catalog` `module` and `markdown` `dir`) and all git history against that directory, wherever the config came from. A `markdown` dir that does not exist SHALL yield no pages when the config came from outside the `--source` tree, and SHALL fail the build otherwise. A build from a work tree with uncommitted changes SHALL record `source.dirty: true`, and `push` SHALL refuse such a bundle.

#### Scenario: Release build from a tag
- **WHEN** `opm-docs build --project catalog-opm --release opm-v4.4.5 --source src` runs on a checkout of that tag
- **THEN** `out/catalog-opm/manifest.json` has version `4.4.5`, revision 0, `source.ref` `opm-v4.4.5` and `source.commit` the tag's commit

#### Scenario: Release older than docs-kit adoption
- **WHEN** the tree at `src/` has no `docs-kit.cue` and the current directory has one
- **THEN** `build --source src` uses the current directory's `docs-kit.cue`, and extracts the `cue-catalog` module and the `markdown` dir from `src/`, not from the current directory

#### Scenario: A missing markdown dir in a normal build
- **WHEN** `docs-kit.cue` names `docs/catalogs/opm` and the same tree has no such directory
- **THEN** `build` exits 2 naming the dir and the config file

#### Scenario: Wrong prefix
- **WHEN** `--release v4.4.5` is passed for project `catalog-opm`, whose prefix is `opm-v`
- **THEN** `build` exits 1 naming the tag and the expected prefix

#### Scenario: Tag and catalog version differ
- **WHEN** `build --release opm-v4.6.0` runs on a tree whose catalog declares `metadata.version: "4.5.0"`
- **THEN** `build` exits 2 naming `4.6.0` and `4.5.0`, and writes no `manifest.json`

### Requirement: check is the pull-request gate
`opm-docs check` SHALL run `build` for the configured projects into a temporary directory, including the dialect lint, and SHALL exit 2 on any failure, printing every violation, and 0 otherwise. It SHALL write nothing in the work tree.

#### Scenario: A doc comment that does not open with the description
- **WHEN** a member's doc comment does not start with its `metadata.description`
- **THEN** `check` exits 2 naming the member's FQN, its file, and both texts

### Requirement: revise builds a docs revision
`opm-docs` SHALL provide `revise --project P --tag T --fix F` with the optional flags `--out`, `--registry` and `--config`, following the steps `docs/contracts.md` "Docs revisions" lists, exiting 0 when the revision is built into `--out`, 1 on a usage error and 2 when a step refuses. It SHALL push nothing.

#### Scenario: Missing fix
- **WHEN** `opm-docs revise --project catalog-opm --tag opm-v4.4.5` runs without `--fix`
- **THEN** it exits 1 naming `--fix`

#### Scenario: Built, not pushed
- **WHEN** `revise` succeeds
- **THEN** `out/<project>/manifest.json` holds the next revision and the registry is unchanged

### Requirement: serve previews a repository's bundles
`opm-docs serve` SHALL build the selected projects as edge bundles (a dirty work tree allowed) into a temporary directory it removes on exit, each with `build`'s own extractors, repository commands and bundle-mode dialect lint, so a page `build` refuses is never served, and SHALL serve them: by default on its embedded minimal Hugo site with the host's `hugo`, listening on `127.0.0.1` only (refusing, with exit 1, a missing `hugo` or one older than 0.146.0, naming the version found), mounting each tab bundle at `<root>edge/`, each docs bundle at `/docs/` and any other placement at its root, and starting a rebuild of a bundle within two seconds of a change to a file under its sources; with `--site <dir>`, by running `task bundles:pull` and then `task serve` in that directory with `OPM_BUNDLES_LOCAL` naming each built tree (a tab or section as `<project>@edge`, a docs bundle as `<project>@<--site-version>`), without watching. A build failure while watching SHALL be printed and SHALL keep the last good bundle served.

#### Scenario: Preview an authored page
- **WHEN** an author in the cli repository runs `opm-docs serve` and edits `docs/site/start/install.md`
- **THEN** the page at `/docs/start/install/` on the printed local URL shows the edit after the rebuild

#### Scenario: A failed rebuild keeps the last good page
- **WHEN** the author saves a page that breaks the page dialect while `opm-docs serve` runs
- **THEN** the violations print as `build` prints them, and the page keeps showing the last build that passed

#### Scenario: No hugo
- **WHEN** `opm-docs serve` runs without `hugo` on `PATH`
- **THEN** it exits 1 naming `hugo` and the version it needs

#### Scenario: Preview in the real site
- **WHEN** `opm-docs serve --site ../opmodel.dev --site-version v1.0` runs in the cli repository
- **THEN** it runs the site's `task bundles:pull` and `task serve` with `OPM_BUNDLES_LOCAL="cli@v1.0=<built tree>"`

#### Scenario: Docs bundle without a site version
- **WHEN** `opm-docs serve --site ../opmodel.dev` runs for a docs bundle without `--site-version`
- **THEN** it exits 1 naming `--site-version`
