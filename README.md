# docs-kit

Builds each Open Platform Model repository's documentation into a versioned, signed OCI artifact (a docs bundle) in that repository's CI, and lets the opmodel.dev site pull and assemble those bundles.

It holds one Go program, `opm-docs`, and the reusable workflow that runs it, `.github/workflows/publish.yml`. [DESIGN.md](DESIGN.md) is the design; [docs/contracts.md](docs/contracts.md) fixes everything another repository reads (the bundle format, tags, the workflow interface, `docs-kit.cue`, the pull config and lock, the signing identity, the doc model and the page dialect).

Status: phase 1 is built, and phase 2's shared ground. `opm-docs` builds the opm catalog's reference from a CUE catalog module (`cue-catalog`) plus a directory of authored pages (`markdown`), publishes it as a signed bundle, pulls bundles for the site, and builds docs revisions of published releases. It also builds bundles placed in a site version's `/docs/` tree, runs repository commands, records the versions a bundle pins and pulls each site version's docs bundles from the anchor's pins; the extractors that use them come in their own changes.

## Source kinds

`docs-kit.cue` (docs/contracts.md C6) lists each bundle's sources. Each extractor kind writes one data file under `data/`, and the renderer registered for that file's schema turns it into pages; the `markdown` kind copies authored pages.

| Kind | Writes | Options |
|---|---|---|
| `cue-catalog` | `data/catalog.json` and the catalog's pages; a tab bundle only | `module` |
| `markdown` | the authored pages of `dir` | `dir`, `include`, `exclude` (globs, or a directory ending `/`) |
| `cue-definitions` | `data/cue-definitions.json` and a reference section of a CUE package's exported definitions, grouped into pages by the config (C17); a docs bundle only | `package`, `skip`, `section`, `title`, `description`, `weight`, `intro`, `pages`, `exclude` (definition: reason) |
| `cobra` | `data/cobra.json` and a command reference: a section index with the global flags, one page per top-level command | `command`, `section`, `title`, `description`, `weight`, `citations` |
| `crd` | `data/crd.json` (C18) and one completable reference page, an entry per controller-gen CRD; a docs bundle only | `dir`, `samples`, `hideSamplesMatching`, `stripLabels`, `page`, `title`, `description`, `weight`, `order`, `reconciledBy`, `citations` |

A bundle placed in `/docs/` (`placement: {kind: "docs", root: "/docs/", owns: [...]}`) lists the content paths it owns; every generated page lies under one (C15). A bundle's `pins: {command, projects}` runs a repository command (C14) and records the exact versions it documents against in `manifest.json`.

**Documenting a cobra CLI** (C19). Require the nested module `github.com/open-platform-model/docs-kit/cobradump` (tags `cobradump/vX.Y.Z`; it pulls in cobra and pflag only) from a small program that the shipped binary does not contain, and point a `cobra` source at it:

```go
// hack/docskit-dump/main.go
func main() {
	if len(os.Args) > 1 && os.Args[1] == "pins" {
		check(cobradump.WritePins(os.Stdout, map[string]string{"core": coreVersion()}))
		return
	}
	check(cobradump.Write(cmd.NewRootCmd(), os.Stdout, cobradump.Options{}))
}
```

```cue
bundles: cli: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/cli/"]}
	version: {from: "tag", prefix: "v"}
	pins: {command: ["go", "run", "./hack/docskit-dump", "pins"], projects: ["core"]}
	sources: [{
		kind:        "cobra"
		command:     ["go", "run", "./hack/docskit-dump"]
		section:     "reference/cli/"
		title:       "CLI Reference"
		description: "Every opm command and flag, generated from the CLI's cobra commands."
	}]
}
```

The caller passes `setup-go: true` to `publish.yml`, so `go run` works in the build job. The dump carries help text raw; docs-kit parses it, so a presentation fix arrives with a docs-kit bump, not a CLI release.

**Adding an extractor.** Write an `Extractor` in `internal/build` (`Kind`, `Extract(ctx, Input) (Data, error)`; `Input` carries the source tree, the source's own config entry, the build identity, the command runner and the citation policy) and add it to the `extractors` table; write a `Renderer` in `internal/render` (`Schema`, `Render(data, Target) ([]Page, error)`, reading only the data file) and add it to `renderers` under its data schema; add the kind to `#Source` in `schema/config.cue` with `citations?: #Citations` and its options. A test asserts that `#Source` and the registry name the same kinds. A renderer page may be completable: an authored page at its path comes first and the generated body follows it (C15).

## Commands

`opm-docs <command> [flags]`. Exit codes: `0` success, `1` usage error (unknown flag, missing argument, unreadable or invalid config), `2` execution error (lint violations, a refused build, a registry or signature failure).

| Command | Flags | Does |
|---|---|---|
| `build` | `--config` (default `docs-kit.cue` in `--source`, else the current directory), `--project` (repeatable), `--out` (`out`), `--source` (`.`), `--edge` (default) or `--release <tag>` | Extract, render and lint; write `out/<project>/`. |
| `check` | `--config`, `--project` | Build into a temporary directory and fail on any problem; every repository command runs twice and must print the same output. The pull-request gate. |
| `lint` | `--bundle`, `--dialect` (`1`) | Lint page directories (or bundle directories) against the page dialect. |
| `push` | `--dir` (required), `--registry` (`ghcr.io/open-platform-model/docs`) | Pack deterministically and push under the full tag (by digest for edge); print `{"digest","tag"}`. |
| `promote` | `--project`, `--digest` (required), `--registry` | Verify the digest's signature, then move the moving tags of its line. Runs in GitHub Actions (`GITHUB_REPOSITORY`). |
| `pull` | `--config` (`bundles.cue`), `--out` (`.bundles`), `--lock` (`<out>/lock.json`), `--frozen <lock>`, `--offline`, `--local <project>@<segment>=<dir>` (repeatable; the segment a tab's `MAJOR.MINOR` or `edge`, or a site version `v<MAJOR>.<MINOR>`) | Resolve, verify, unpack, lint and lock the bundles the site shows; write each tab's version history to `<out>/<project>/history.json` (docs/contracts.md C13); pull each site version's docs bundles into `<out>/_versions/<site-version>/<project>/` (C16). |
| `revise` | `--project`, `--tag`, `--fix` (required), `--out` (`out`), `--registry`, `--config` (default `docs-kit.cue` in the release tree, else the current directory) | Build the next docs revision of a published release with a documentation fix from `main` into `out/<project>/`; push nothing ([docs revisions](docs/contracts.md#docs-revisions)). |
| `serve` | `--config`, `--project` (repeatable), `--source` (`.`), `--port` (`1313`), `--site <dir>` (not with `--port`), `--site-version v<MAJOR>.<MINOR>` (with `--site`, for a docs bundle) | Build edge bundles and preview them on a local Hugo, rebuilding on change; or through an opmodel.dev checkout ([preview your pages](#preview-your-pages)). |
| `version` | | Print `opm-docs <version>`. |

A local preview needs no registry: `opm-docs serve` (below), or `opm-docs build` in the source repository and then point the site at the output with `opm-docs pull --local catalog-opm@edge=<repo>/out/catalog-opm`.

**Site versions.** A site version's `/docs/` comes from docs bundles chosen by `bundles.cue` `versions` (docs/contracts.md C16): an anchor bundle (the cli at a tag such as `1.0`), the projects pulled at the exact versions the anchor's `manifest.json` `pins` names (through each release's tag, so docs revisions follow), and projects pulled by their own tag. A version is replaced whole or not at all: its bundles must not serve one URL twice or put a page under another bundle's owned path. `--local cli@v1.0=<cli>/out/cli` previews a cli tree with the releases it pins.

## Installing

Consumers never `go run` or `go install` `opm-docs`. Every release `vX.Y.Z` carries `opm-docs_X.Y.Z_<os>_<arch>.tar.gz` for linux and darwin on amd64 and arm64, each holding `opm-docs` and `LICENSE`, and `checksums.txt`. Pin a release and verify it (docs/contracts.md C12):

- **On a host, and for `publish.yml`**: a repo-root `.opm-docs-version` holding the tag (`v0.1.0`); download the archive for the host and `checksums.txt`, check with `grep ' <archive>$' checksums.txt | sha256sum -c -` (refusing a missing line), extract only `opm-docs` into a gitignored `.bin/`.
- **In a build image**: pin the `linux_amd64` archive by the SHA-256 on its `checksums.txt` line and check it in the image build.

## Using the workflow

Reference `publish.yml` by docs-kit release tag, never a branch or a SHA: the signing certificate names the workflow at that ref, and the site trusts only `refs/tags/v[0-9]*` (docs/contracts.md C5, C9). This is a deliberate exception to the org's SHA-pinning convention, safe because docs-kit's tags are immutable; say so in a comment beside the `uses:` line. The tool it runs is the caller's: the tag in the repository's `.opm-docs-version` (one line, `v0.1.0`), checked against that release's `checksums.txt`. Bump `.opm-docs-version` and the `@vX.Y.Z` ref together, in one PR.

| Mode | Run it on | Caller job permissions |
|---|---|---|
| `check` | `pull_request` | `contents: read`, `packages: read` |
| `edge` | `push` to `main` | `contents: read`, `packages: write`, `id-token: write` |
| `release` | the release-please job, gated on that package's release; or `workflow_dispatch` to backfill a release | `contents: read`, `packages: write`, `id-token: write` |
| `revision` | `workflow_dispatch`, with the inputs `tag` and `fix` (needs docs-kit `v0.2.0` or later) | `contents: read`, `packages: write`, `id-token: write` |

```yaml
# Pinned by tag, not SHA: docs-kit tags are immutable and the signing
# identity names the tag (docs-kit docs/contracts.md C5).
jobs:
  check:
    if: github.event_name == 'pull_request'
    permissions: {contents: read, packages: read}
    uses: open-platform-model/docs-kit/.github/workflows/publish.yml@v0.1.0
    with: {project: catalog-opm, mode: check}

  edge:
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    permissions: {contents: read, packages: write, id-token: write}
    uses: open-platform-model/docs-kit/.github/workflows/publish.yml@v0.1.0
    with: {project: catalog-opm, mode: edge}

  # In release.yml, after release-please; or on workflow_dispatch with a tag
  # input to backfill a release cut before the repository adopted docs-kit.
  publish-docs:
    needs: release-please
    if: needs.release-please.outputs.opm_tag_name != ''
    permissions: {contents: read, packages: write, id-token: write}
    uses: open-platform-model/docs-kit/.github/workflows/publish.yml@v0.1.0
    with:
      project: catalog-opm
      mode: release
      tag: ${{ needs.release-please.outputs.opm_tag_name }}
```

The workflow builds in one job, which declares only `contents: read` and `packages: read`, and pushes, signs and promotes in a second job that checks out nothing of the caller, so a repository command never runs beside the signing token. Pass `setup-go: true` when a repository command needs Go.

Every publishing mode runs only on `refs/heads/main`. A new package on GHCR is linked to the calling repository through `org.opencontainers.image.source`; check that it is public before the site pulls it.

## Shipping authored docs

A repository's authored pages (`docs/site/`) ship in its one docs-placed bundle, so the site reads no source repository for them (docs/contracts.md C15, "Authored docs"):

1. Add `{kind: "markdown", dir: "docs/site"}` to the repository's docs bundle in `docs-kit.cue`, beside its extractor sources; a repository with only authored pages (opm) or a tab already (catalog_opm, as `catalog-opm-docs`) gets a bundle with `placement: {kind: "docs", root: "/docs/"}` holding that source alone. While committed generated pages still sit under `docs/site/`, `exclude` them (`exclude: ["reference/definitions/"]`), or the build fails naming the page and both sources.
2. Run `opm-docs build` and read its lint output: pages are copied as written, and their `/catalogs/` links must use the bare root or a major (C8). `manifest.json` records each page's `source`, `lastmod` and, for a file `main` has, `edit`: the site's "Edit this page" links that path on `main`, whatever version the page shows.
3. Move `.opm-docs-version` and the `publish.yml@` ref to a docs-kit release that writes `edit` only after opmodel.dev's pinned `opm-docs` is at least that release (C12): an older `opm-docs` refuses a manifest carrying it.
4. Publish as for any bundle (above). Backfill only a tag whose `docs/site/` holds no page the site serves from elsewhere: a backfill takes `main`'s config and the tag's pages.

## Preview your pages

`opm-docs serve` in a repository with a `docs-kit.cue` builds its bundles as edge bundles (uncommitted changes included) and serves them on `http://127.0.0.1:1313/` with the `hugo` on your `PATH` (0.146.0 or later). The site it serves is a minimal one built into `opm-docs`: it shows each page's title, description, body and alerts, a sidebar of its section, and a labelled box for each figure, not the site's design. A docs bundle's pages are at `/docs/`, a tab's at its root's edge segment (`/catalogs/opm/edge/`). Save a page and its bundle rebuilds within a couple of seconds; Hugo reloads the browser. Every build is `opm-docs build`'s, lint included, so a page that would fail the PR check prints its violations and the preview keeps the last build that passed. Repository commands (a `cobra` dump, `pins`) rerun on every rebuild, as `build` runs them. Ctrl-C, SIGTERM or a closed terminal stops it and removes the build; on Linux, hugo also stops when `opm-docs` is killed, and the next `serve` removes a killed run's build. A page added or removed restarts hugo, so the sidebar and the removed page's URL follow.

To see the real theme, preview through an opmodel.dev checkout: `opm-docs serve --site ../opmodel.dev --site-version v1.0` builds once and runs the site's `task bundles:pull` and `task serve` there with `OPM_BUNDLES_LOCAL` naming each built tree (a tab as `<project>@edge`, a docs bundle as `<project>@<site version>`; your own `OPM_BUNDLES_FROZEN` and `OPM_BUNDLES` are not passed). That needs `task`, the checkout's `Taskfile.yml` and `site/bundles.cue`, and the site's own build image; the site picks its port (`SITE_PORT`), so `--port` is refused, and nothing rebuilds on change. Two limits:

- **A docs bundle** previews only once the checkout's `site/bundles.cue` names site versions (`docs` and `versions`, docs/contracts.md C16), which opmodel.dev's `pull-reference-bundles` change adds; before that the pull has no site version to put it in.
- **A tab project** given locally skips the registry for every segment, so docs links to its major alias (`/catalogs/opm/4/`) fail the site build there; use the embedded preview for a tab's edge pages.

Afterwards, run `task bundles:pull` once more in the checkout: its lock (`site/.bundles/lock.json`) still records your local trees, and a later `task build` would use them.

## Fixing a release's docs

A published release's pages change only through a docs revision: its full tag (`4.4.5.0`) is never overwritten, so a documentation fix is published as `4.4.5.1`, `4.4.5.2` and so on, and `4.4.5`, `4.4` and `4` move to it. A fix that changes code needs a patch release instead.

1. Land the fix on `main` as one single-parent commit that changes only Markdown files, or only comments in `.cue` and `.go` files. Markdown counts as code in a release whose CUE embeds files (`@extern(embed)`). `edge` shows it on the next push.
2. Dispatch the `revision` mode with the release tag and the fix's full 40-hex hash. The workflow applies every fix the newest revision of that release already carries, then this one, to the release tree; it refuses a fix that is not on `main`, one already applied, a conflict, and any change other than documentation, naming each file. Then it pushes `<version>.<next>`, signs it and moves the moving tags. A run that failed after its push can simply be re-run: the unpromoted revision is built again to the same digest and finished. Revisions of one release run one at a time, and GitHub keeps only one waiting run per release: a revision dispatched while another waits cancels the waiting one, which shows as cancelled and must be dispatched again. The fix is a full 40-hex (SHA-1) hash; SHA-256 repositories are not supported.

The steps and the documentation-only rules are in [docs/contracts.md, "Docs revisions"](docs/contracts.md#docs-revisions). catalog_opm's dispatch, beside its other `docs.yml` jobs:

```yaml
on:
  workflow_dispatch:
    inputs:
      mode: {type: choice, options: [release, revision], default: release}
      tag: {description: "the release's git tag (opm-v4.4.5)", type: string, required: true}
      fix: {description: "revision: the 40-hex commit on main to apply", type: string, default: ""}

jobs:
  docs-dispatch:
    if: github.event_name == 'workflow_dispatch'
    permissions: {contents: read, packages: write, id-token: write}
    uses: open-platform-model/docs-kit/.github/workflows/publish.yml@v0.2.0
    with:
      project: catalog-opm
      mode: ${{ inputs.mode }}
      tag: ${{ inputs.tag }}
      fix: ${{ inputs.fix }}
```

Or from a terminal: `gh workflow run docs.yml -f mode=revision -f tag=opm-v4.4.5 -f fix=<sha>`.

## Development

`task check` runs every gate (format, vet, golangci-lint, OpenSpec, tests). Tests are offline. `TestCatalogOPMParity` compares the rendered opm catalog with catalog_opm's former generator when `OPM_DOCS_CATALOG_OPM` names a catalog_opm checkout at `opm-v4.4.5` (it reads CUE modules from GHCR: set `CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'`).

Licensed under the Apache License, Version 2.0; see [LICENSE](LICENSE).
