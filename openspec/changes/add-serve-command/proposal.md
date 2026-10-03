## Why

When authored pages and generated reference leave git (phase 3), an author can no longer preview a page by opening the site's checkout of their repository. DESIGN.md lists `opm-docs serve`: build one repository's bundles and serve them on a local Hugo. Today an author must build with `opm-docs build`, then run the site with `OPM_BUNDLES_LOCAL` by hand.

Gate: `generalize-build-assembly` is merged. Independent of the other phase-3 changes; useful from the first repository's phase-3 cutover.

## What Changes

- **`opm-docs serve`** builds the selected projects of `docs-kit.cue` as edge bundles (a dirty tree allowed) and serves them:
  - by default on a minimal Hugo site embedded in `opm-docs` (front matter, alerts, the seven figure names as placeholders, no theme), with the host's `hugo`, rebuilding a bundle when a file under its sources changes so Hugo reloads the page;
  - with `--site <opmodel.dev checkout>`, through the site's own preview (`task bundles:pull` then `task serve`, both given `OPM_BUNDLES_LOCAL`), so the author sees the real theme; no watch in this mode.
- **Contract**: the command in "Commands"; the site interface it relies on (`OPM_BUNDLES_LOCAL`, the two tasks) recorded in "Site decisions".

Release class: MINOR. A new command; no contract other repositories read changes.

Scope: three sections, the last with the docs and the archive.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `opm-docs-cli`: the command set gains `serve` (and names `revise`, added earlier); `serve`'s behavior (requirement added).

## Impact

- Code: `cmd/opm-docs/serve.go`, `internal/serve` (skeleton site embedded with `go:embed`, Hugo process, polling watcher), `README.md`, `docs/contracts.md` "Commands" and "Site decisions".
- Consumers: every repository's authors (a `task docs:serve` wrapper in each phase-3 sibling). **opmodel.dev** keeps `OPM_BUNDLES_LOCAL`, `task bundles:pull` and `task serve` working for `--site` mode, and accepts docs-project segments (`cli@v1.0=<dir>`) there once `pull-docs-placement` ships.
