## Why

Docs pages name OPM releases by hand, and the numbers rot: the cli install page says beta.5, the opm
quickstart says beta.12. Only the opm-controller install page is stamped, by release-please, and
only with that repository's own version; the repository's release-automation spec forbids any other
project's number there. A page needs a way to name the version of the cli, core, library, the
controller, the controller module or the opm catalog, filled in per site version, so that an old
site version keeps the numbers it was built with.

`opm-docs pull` already collects the docs bundles of each site version and knows each bundle's exact
version (DESIGN decision 10). It is the step that fills the placeholders, and `opm-docs lint` is
the place that refuses a misspelt one in the author's own pull request.

## What Changes

- A placeholder form `{{opm:<name>.version}}` and a fixed list of names in docs-kit: `cli`, `core`,
  `library`, `opm-controller`, `opm-controller-module`, `catalog-opm`. The value is the bare version
  with no `v`.
- `opm-docs lint` refuses a `{{opm:` that is not a well-formed placeholder with a listed name, and
  prints the known names. It checks every line, code fences, links and front matter included.
- A docs-kit.cue bundle may declare `units`: a version read from a repository file such as
  `.release-please-manifest.json`. `opm-docs build` records it in `manifest.json` under a new
  optional field `units`. The opm-controller bundle uses it for `opm-controller-module`.
- `opm-docs pull` fills the placeholders of each site version's docs bundles by a plain text
  replace, from the versions it collected for that site version, and fails the pull when any
  `{{opm:` is left in a pulled tree, naming the page.
- `opm-docs revise --refresh-units` and a `refresh-units` input of `publish.yml` rebuild the newest
  revision of a release with the units read from `main`, so that a module release reaches the site
  (see Design, "Republishing on a module release").

## Capabilities

### New Capabilities

- `version-placeholders`: the placeholder form, the name list, how a name maps to a value, and the
  fill and the leftover check.

### Modified Capabilities

- `dialect-lint`: the placeholder check.
- `bundle-format`: the optional manifest field `units`.
- `opm-docs-cli`: the `units` entry of `docs-kit.cue`, the build that records it, `revise --refresh-units`.
- `bundle-pull`: the fill and the leftover check.
- `publish-workflow`: the `refresh-units` input of revision mode.

## Impact

- Packages: new `internal/placeholder` (names, scan, fill); `internal/dialect` (`page.go`);
  `internal/config`, `schema/config.cue`, `schema/manifest.cue`, `internal/bundle/manifest.go`;
  `internal/build` (new `units.go`); `internal/pull`; `internal/revise`; `cmd/opm-docs`;
  `.github/workflows/publish.yml`.
- Contracts that change (all additive): `docs/contracts.md` C3 (manifest `units`), C5 (`refresh-units`
  input), C6 (`units` in `docs-kit.cue`), C7 and C16 (the fill and the leftover check in `pull`),
  C11 (the placeholder rule). Dialect stays version `1`: a page that was valid stays valid, except
  a page that already holds the literal text `{{opm:`.
- A bundle that carries `units` is refused by an older `opm-docs` (the manifest schema is closed,
  C12). The site therefore bumps its pinned `opm-docs` first.
- Consumers, in order. opmodel.dev: bump `opm-docs` in `site/Dockerfile`; its pull then fills and
  checks (no script of its own). cli, opm, opm-controller: bump `.opm-docs-version` and every
  `publish.yml` ref in one PR, then write placeholders in their pages; opm-controller also adds
  `units` to its `docs-kit.cue` and calls `publish.yml` in revision mode on a module release.
  core, library, catalog_opm, opm-portal, modules, enhancements: nothing to do.
- Release class: minor bump of `opm-docs` (docs-kit is on 0.x). Not breaking. Commit type and PR
  title of the implementing PR: `feat(lint): refuse unknown version placeholders and fill them on pull`.
  Per repo rule, `feat` and `fix` release `opm-docs` through release-please; `docs`, `test`, `chore`
  do not.

## Enhancement

None.
