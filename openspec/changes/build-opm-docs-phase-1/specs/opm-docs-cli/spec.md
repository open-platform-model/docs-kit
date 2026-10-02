## Purpose

The `opm-docs` command set, its configuration file `docs-kit.cue`, and its exit codes and messages.

## ADDED Requirements

### Requirement: The command set
`opm-docs` SHALL provide the commands `build`, `lint`, `check`, `push`, `promote`, `pull` and `version`, with the flags `design.md` lists under "Commands". It SHALL exit 0 on success, 1 on a usage error (unknown command or flag, missing argument, unreadable or invalid config) and 2 on an execution error (lint violations, a refused build, a registry or signature failure). Every error message SHALL name what failed and the fix.

#### Scenario: Version
- **WHEN** `opm-docs version` runs
- **THEN** it prints `opm-docs <version>` and exits 0

#### Scenario: Unknown flag
- **WHEN** `opm-docs build --nope` runs
- **THEN** it exits 1 naming `--nope`

### Requirement: docs-kit.cue configures the bundles a repository builds
`build` and `check` SHALL read `docs-kit.cue` (or `--config`) and validate it against the embedded `#Config` schema before any extraction. The file SHALL hold `bundles`, keyed by project, each with a `placement`, a `version` (`from: "tag"` and a tag `prefix`) and at least one source of kind `cue-catalog` (`module`) or `markdown` (`dir`).

#### Scenario: A misspelled key is refused
- **WHEN** `docs-kit.cue` holds `bundles: "catalog-opm": {placment: ...}`
- **THEN** `opm-docs build` exits 1 naming the field `placment` and the file, before loading any CUE module

### Requirement: build writes one bundle tree per project
`opm-docs build` SHALL write `out/<project>/` (or under `--out`) for every project in the config, or only those named with `--project`. With `--release <tag>` it SHALL derive the version by removing the project's tag prefix and SHALL refuse a tag without that prefix or whose remainder is not SemVer. Without `--release` it SHALL build an edge bundle. With `--source <dir>` it SHALL read sources and git history from that directory and take `docs-kit.cue` from it when present, else from the current directory. A build from a work tree with uncommitted changes SHALL record `source.dirty: true`, and `push` SHALL refuse such a bundle.

#### Scenario: Release build from a tag
- **WHEN** `opm-docs build --project catalog-opm --release opm-v4.4.5 --source src` runs on a checkout of that tag
- **THEN** `out/catalog-opm/manifest.json` has version `4.4.5`, revision 0, `source.ref` `opm-v4.4.5` and `source.commit` the tag's commit

#### Scenario: Release older than docs-kit adoption
- **WHEN** the tree at `src/` has no `docs-kit.cue` and the current directory has one
- **THEN** `build --source src` uses the current directory's `docs-kit.cue` and builds from `src/`

#### Scenario: Wrong prefix
- **WHEN** `--release v4.4.5` is passed for project `catalog-opm`, whose prefix is `opm-v`
- **THEN** `build` exits 1 naming the tag and the expected prefix

### Requirement: check is the pull-request gate
`opm-docs check` SHALL run `build` for the configured projects into a temporary directory, including the dialect lint, and SHALL exit 2 on any failure, printing every violation, and 0 otherwise. It SHALL write nothing in the work tree.

#### Scenario: A doc comment that does not open with the description
- **WHEN** a member's doc comment does not start with its `metadata.description`
- **THEN** `check` exits 2 naming the member's FQN, its file, and both texts
