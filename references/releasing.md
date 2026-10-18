# Releasing docs-kit

Covers how docs-kit releases: the two release components, what a commit type does, the contract
that callers and consumers pin, and how to test a release build locally.

## How a release happens

release-please (as the release App) opens a release PR from `feat` and `fix` commits on `main`.
Merging it creates the tag `vX.Y.Z` and a draft release. The release workflow then runs goreleaser
(pinned), which attaches `opm-docs_X.Y.Z_<os>_<arch>.tar.gz` for linux and darwin on amd64 and
arm64 plus `checksums.txt`, and publishes the release.

## Two components

| Component | Path | Tags | Release form |
| --- | --- | --- | --- |
| `opm-docs` | `.` | `vX.Y.Z` | Draft first, goreleaser assets |
| `cobradump` | `cobradump/` | `cobradump/vX.Y.Z` | Published at once, no assets; the Go module proxy serves it from the tag |

Each has its own release PR. The root package excludes `cobradump/`, so a commit there (scope
`cobradump`) releases only `cobradump`. The release workflow runs goreleaser only for an `opm-docs`
release (`docs/contracts.md` C12).

## The publish workflow carries no version

`publish.yml` installs the release named by the caller's `.opm-docs-version`. Callers move that file
together with their `publish.yml` tag ref (`docs/contracts.md` C5). Never add a version literal to
the workflow.

## Commit types

- A dependency bump that can change how CUE constraints are formatted, `cuelang.org/go` above all,
  releases as a minor, never a patch. Commit it as `feat(deps): ...` (pre-1.0, release-please makes
  `feat` a minor), never `fix` or `chore`, and say so in the PR body. Reason: the version history
  compares `type` and `default` strings only within one docs-kit minor (`docs/contracts.md` C10,
  C13), so a patch must never change them.

## Contract: archive names

The archive names, `checksums.txt` and the `linux_amd64` archive are a contract
(`docs/contracts.md` C12). Consumers pin them by name and SHA-256. Do not rename one.

## Test a release build locally

- `goreleaser release --snapshot --clean` builds the four archives into `dist/` (gitignored).
- `goreleaser check` validates the goreleaser config at the root.

## Pitfalls

- A `fix:` or `chore:` commit for a CUE bump ships a patch that changes version-history output.

## Check before you finish

- [ ] A CUE bump is typed `feat(deps)` and the PR body says why.
- [ ] No version literal was added to `publish.yml`.
- [ ] No archive name or `checksums.txt` form changed, or `docs/contracts.md` C12 changed with it.
- [ ] `goreleaser check` passes after a change to the goreleaser config.
