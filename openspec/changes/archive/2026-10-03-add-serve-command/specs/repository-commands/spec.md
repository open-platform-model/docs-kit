## MODIFIED Requirements

### Requirement: A repository command runs without a shell in the source tree
Where `docs-kit.cue` names a command (an argv list), `build`, `check`, `revise` and `serve` SHALL run it with `argv[0]` looked up on `PATH` and no shell, in the source tree, in a process group of its own that a timeout or cancellation kills whole, with an allowlisted environment (`PATH`, `HOME`, `TMPDIR`, `USER`, `LANG`, `LC_*`, the proxy variables, Go's `GO<CAPITALS>` variables, `CGO_*`, `CUE_*`, `OPM_DOCS*`) plus `OPM_DOCS=1`, `OPM_DOCS_PROJECT` and `OPM_DOCS_VERSION`, empty stdin and stderr passed through. No other variable of the build's environment, a token included, SHALL reach it. `push`, `promote` and `pull` SHALL never run a repository command.

#### Scenario: Environment of a dump
- **WHEN** a cli edge build runs the command `["go", "run", "./hack/docskit-dump"]`
- **THEN** the program runs in the cli tree with `OPM_DOCS_PROJECT=cli` and `OPM_DOCS_VERSION=edge` set

#### Scenario: A token in the build's environment
- **WHEN** `GITHUB_TOKEN` is set where `opm-docs build` runs a command
- **THEN** the command's environment holds no `GITHUB_TOKEN`
