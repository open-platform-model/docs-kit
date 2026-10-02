# docs-kit

Builds each Open Platform Model repository's documentation into a versioned, signed OCI artifact (a docs bundle) in that repository's CI, and lets the opmodel.dev site pull and assemble those bundles.

It holds one Go program, `opm-docs`, and the reusable workflow that runs it, `.github/workflows/publish.yml`. [DESIGN.md](DESIGN.md) is the design; [docs/contracts.md](docs/contracts.md) fixes everything another repository reads (the bundle format, tags, the workflow interface, `docs-kit.cue`, the pull config and lock, the signing identity, the doc model and the page dialect).

Status: phase 1. `opm-docs` builds the opm catalog's reference from a CUE catalog module (`cue-catalog`) plus a directory of authored pages (`markdown`), publishes it as a signed bundle, and pulls bundles for the site. Docs revisions follow in the change `add-docs-revisions`.

## Commands

`opm-docs <command> [flags]`. Exit codes: `0` success, `1` usage error (unknown flag, missing argument, unreadable or invalid config), `2` execution error (lint violations, a refused build, a registry or signature failure).

| Command | Flags | Does |
|---|---|---|
| `build` | `--config` (default `docs-kit.cue` in `--source`, else the current directory), `--project` (repeatable), `--out` (`out`), `--source` (`.`), `--edge` (default) or `--release <tag>` | Extract, render and lint; write `out/<project>/`. |
| `check` | `--config`, `--project` | Build into a temporary directory and fail on any problem. The pull-request gate. |
| `lint` | `--bundle`, `--dialect` (`1`) | Lint page directories (or bundle directories) against the page dialect. |
| `push` | `--dir` (required), `--registry` (`ghcr.io/open-platform-model/docs`) | Pack deterministically and push under the full tag (by digest for edge); print `{"digest","tag"}`. |
| `promote` | `--project`, `--digest` (required), `--registry` | Verify the digest's signature, then move the moving tags of its line. Runs in GitHub Actions (`GITHUB_REPOSITORY`). |
| `pull` | `--config` (`bundles.cue`), `--out` (`.bundles`), `--lock` (`<out>/lock.json`), `--frozen <lock>`, `--offline`, `--local <project>@<segment>=<dir>` (repeatable) | Resolve, verify, unpack, lint and lock the bundles the site shows. |
| `version` | | Print `opm-docs <version>`. |

A local preview needs no registry: `opm-docs build` in the source repository, then point the site at the output with `opm-docs pull --local catalog-opm@edge=<repo>/out/catalog-opm`.

## Installing

Consumers never `go run` or `go install` `opm-docs`. Every release `vX.Y.Z` carries `opm-docs_X.Y.Z_<os>_<arch>.tar.gz` for linux and darwin on amd64 and arm64, each holding `opm-docs` and `LICENSE`, and `checksums.txt`. Pin a release and verify it (docs/contracts.md C12):

- **On a host**: a repo-root `.opm-docs-version` holding the tag (`v0.1.0`); download the archive for the host and `checksums.txt`, check with `grep ' <archive>$' checksums.txt | sha256sum -c -` (refusing a missing line), extract only `opm-docs` into a gitignored `.bin/`.
- **In a build image**: pin the `linux_amd64` archive by the SHA-256 on its `checksums.txt` line and check it in the image build.

## Using the workflow

Reference `publish.yml` by docs-kit release tag, never a branch or a SHA: the signing certificate names the workflow at that ref, and the site trusts only `refs/tags/v*` (docs/contracts.md C5, C9). This is a deliberate exception to the org's SHA-pinning convention, safe because docs-kit's tags are immutable; say so in a comment beside the `uses:` line. `publish.yml@vX.Y.Z` runs `opm-docs` X.Y.Z, so one ref bump upgrades both.

| Mode | Run it on | Caller job permissions |
|---|---|---|
| `check` | `pull_request` | `contents: read`, `packages: read` |
| `edge` | `push` to `main` | `contents: read`, `packages: write`, `id-token: write` |
| `release` | the release-please job, gated on that package's release; or `workflow_dispatch` to backfill a release | `contents: read`, `packages: write`, `id-token: write` |

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

Every publishing mode runs only on `refs/heads/main`. A new package on GHCR is linked to the calling repository through `org.opencontainers.image.source`; check that it is public before the site pulls it.

## Development

`task check` runs every gate (format, vet, golangci-lint, OpenSpec, tests). Tests are offline. `TestCatalogOPMParity` compares the rendered opm catalog with catalog_opm's former generator when `OPM_DOCS_CATALOG_OPM` names a catalog_opm checkout at `opm-v4.4.5` (it reads CUE modules from GHCR: set `CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'`).

Licensed under the Apache License, Version 2.0; see [LICENSE](LICENSE).
