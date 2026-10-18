# docs-kit: where do I go?

Router only. Facts and rules are in `AGENTS.md`.

| Task | Read first | Then |
| --- | --- | --- |
| Add or change a command, a flag or an exit code | `cmd/opm-docs/` | `docs/contracts.md` |
| Add or change an extractor or a renderer | `internal/extract/` | `internal/render/`, `docs/contracts.md` |
| Change the bundle format, tag scheme, `docs-kit.cue`, pull config or lock | `docs/contracts.md` | `schema/`, `DESIGN.md` |
| Change the page dialect, the markup check or the goldmark pin | `internal/dialect/` | `internal/mdsafe/`, `docs/contracts.md` |
| Change `publish.yml`, CI or another workflow | `.github/workflows/` | `references/releasing.md` |
| Choose a commit type, type a dependency bump, fix a bad release, or change the release pipeline | `references/releasing.md` | `release-please-config.json` |
| Understand a design decision or the phase plan | `DESIGN.md` | `docs/orchestration.md` |
| Start a feature change | `openspec/config.yaml` | `DESIGN.md` |
| Deliver or cite an enhancement | `references/enhancements.md` at the workspace root | `AGENTS.md` |
| Look up a registry mapping or a CUE resolution problem | `references/registry.md` at the workspace root | |
| Commit, open a PR or write GitHub text | `references/git-text.md` at the workspace root | |
| Tag, publish or recover a release | `references/releases.md` at the workspace root | `references/releasing.md` |
