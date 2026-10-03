## MODIFIED Requirements

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
- **THEN** `build` exits 2 naming `4.6.0` and `4.5.0`, and writes no bundle
