## Why

The cli generates its command reference (`docs/site/reference/cli/`, ten pages) with `internal/cmdref` and `hack/cmdref` (about 1,200 lines). A program's command tree exists only inside the program, so docs-kit cannot read it from source or a binary (DESIGN.md "The cobra hook"): the cli must print it. This change ships the hook as a tiny nested Go module, `cobradump`, the only docs-kit code a repository imports, and the `cobra` extractor that reads its output. The same program prints the cli's pins (DESIGN decision 10), which `generalize-build-assembly` records in the manifest.

Gate: `generalize-build-assembly` is merged (registry, docs placement, repository commands, pins, `setup-go`). Parallel with the other extractor changes and `pull-docs-placement`.

## What Changes

- **`cobradump`**, a nested module `github.com/open-platform-model/docs-kit/cobradump` with its own `go.mod` (cobra and pflag only), released by release-please as its own component with tags `cobradump/vX.Y.Z`: `Write(root, w, opts)` prints the command-tree dump (`docs.opmodel.dev/cobradump/v1`), `WritePins(w, pins)` prints a pins document (`docs.opmodel.dev/pins/v1`).
- **The dump contract (C19)**: which commands and flags it includes (hidden, deprecated and the help command skipped; the default completion command added; inherited flags resolved), home directories in defaults written as `~`, everything sorted, raw `Long` and `Example` strings (docs-kit parses them, so a text-rule fix needs no cli release).
- **`cobra` source kind** running the repository's dump command (C14), writing `data/cobra.json`, and its renderer: `<section>_index.md` with the global flags and one page per top-level command holding it and every command below it, as cmdref writes them.
- **Release plumbing**: release-please's second package, its tag form, and `release.yml` running goreleaser only for the `opm-docs` release. The `cobradump/` tag does not match the signer glob `refs/tags/v[0-9]*` (C9), so it can never act as a trusted `publish.yml` ref.
- **Parity** with cmdref for the same tree; **contract C19**.

Release class: MINOR for `opm-docs`; `cobradump` starts at `0.1.0`.

Scope: four sections, the last with the contract and the archive.

## Capabilities

### New Capabilities

- `cobra-extractor`: the dump format, `cobradump`'s API, the source kind and its pages.

### Modified Capabilities

- `tool-release`: a second release component, `cobradump` (requirement added).

## Impact

- Code: new `cobradump/` (own `go.mod`, tests), `internal/extract/cobra`, `internal/render` (templates and the Long/Example parser ported from cli `internal/cmdref/text.go`), `schema/config.cue`, `release-please-config.json`, `.release-please-manifest.json`, `.github/workflows/release.yml` and `ci.yml` (test, vet and lint the nested module), `Taskfile.yml` (`test`, `vet`, `lint` cover `cobradump/`), `docs/contracts.md` C12 (release assets unchanged; the component noted) and C19.
- Consumers: **cli** (sibling change `publish-cli-bundle`): `hack/docskit-dump` importing `cobradump`, `docs-kit.cue` with the `cobra` source and `pins`, the workflows; the first cli release with the hook is the first with a bundle. **opmodel.dev**: pulls `cli` as each site version's anchor (`pull-docs-placement`).
- Risk: a cli release cut before the hook has no cli bundle and cannot be backfilled (the hook must exist in the tagged tree); a docs revision of such a release is refused too.
