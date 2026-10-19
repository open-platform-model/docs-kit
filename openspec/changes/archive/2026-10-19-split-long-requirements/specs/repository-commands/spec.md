## REMOVED Requirements

### Requirement: A repository command runs without a shell in the source tree
**Reason**: Split into three requirements that each state one behavior.
**Migration**: See "A repository command runs in the source tree without a shell", "A repository command gets an allowlisted environment" and "push, promote and pull never run a repository command".

## ADDED Requirements

### Requirement: A repository command runs in the source tree without a shell
Where `docs-kit.cue` names a command (an argv list), `build`, `check`, `revise` and `serve` SHALL run it with `argv[0]` looked up on `PATH` and no shell, in the source tree, in a process group of its own that a timeout or cancellation kills whole.

#### Scenario: Environment of a dump
- **WHEN** a cli edge build runs the command `["go", "run", "./hack/docskit-dump"]`
- **THEN** the program runs in the cli tree with `OPM_DOCS_PROJECT=cli` and `OPM_DOCS_VERSION=edge` set

### Requirement: A repository command gets an allowlisted environment
Where `docs-kit.cue` names a command (an argv list), `build`, `check`, `revise` and `serve` SHALL run it with an allowlisted environment (`PATH`, `HOME`, `TMPDIR`, `USER`, `LANG`, `LC_*`, the proxy variables, Go's `GO<CAPITALS>` variables, `CGO_*`, `CUE_*`, `OPM_DOCS*`) plus `OPM_DOCS=1`, `OPM_DOCS_PROJECT` and `OPM_DOCS_VERSION`, empty stdin and stderr passed through. No other variable of the build's environment, a token included, SHALL reach it.

#### Scenario: A token in the build's environment
- **WHEN** `GITHUB_TOKEN` is set where `opm-docs build` runs a command
- **THEN** the command's environment holds no `GITHUB_TOKEN`

### Requirement: push, promote and pull never run a repository command
`push`, `promote` and `pull` SHALL never run a repository command.

#### Scenario: Commands that never run a repository command
- **WHEN** `push`, `promote` or `pull` runs
- **THEN** it runs no repository command
