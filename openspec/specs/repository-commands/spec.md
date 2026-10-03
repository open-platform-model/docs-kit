# repository-commands Specification

## Purpose
How a build runs a program of the repository and reads its output: argv without a shell, the source tree, a fixed environment, one JSON document with a schema id, a timeout and the double run of `check`. The contract is `docs/contracts.md` C14.

## Requirements

### Requirement: A repository command runs without a shell in the source tree
Where `docs-kit.cue` names a command (an argv list), `build`, `check`, `revise` and `serve` SHALL run it with `argv[0]` looked up on `PATH` and no shell, in the source tree, in a process group of its own that a timeout or cancellation kills whole, with an allowlisted environment (`PATH`, `HOME`, `TMPDIR`, `USER`, `LANG`, `LC_*`, the proxy variables, Go's `GO<CAPITALS>` variables, `CGO_*`, `CUE_*`, `OPM_DOCS*`) plus `OPM_DOCS=1`, `OPM_DOCS_PROJECT` and `OPM_DOCS_VERSION`, empty stdin and stderr passed through. No other variable of the build's environment, a token included, SHALL reach it. `push`, `promote` and `pull` SHALL never run a repository command.

#### Scenario: Environment of a dump
- **WHEN** a cli edge build runs the command `["go", "run", "./hack/docskit-dump"]`
- **THEN** the program runs in the cli tree with `OPM_DOCS_PROJECT=cli` and `OPM_DOCS_VERSION=edge` set

#### Scenario: A token in the build's environment
- **WHEN** `GITHUB_TOKEN` is set where `opm-docs build` runs a command
- **THEN** the command's environment holds no `GITHUB_TOKEN`

### Requirement: A command's output is one JSON document with a schema id
The command's stdout SHALL be exactly one JSON document of at most 16 MiB carrying a `schema` field its consumer knows. Extra output after the document, an unknown schema, a non-zero exit or a run longer than 10 minutes SHALL fail the build with exit 2, naming the argv and the project.

#### Scenario: The command fails
- **WHEN** the dump program exits 1
- **THEN** `build` exits 2 naming `go run ./hack/docskit-dump`, the project and exit status 1

#### Scenario: Trailing output
- **WHEN** the program prints a JSON document followed by a log line on stdout
- **THEN** `build` exits 2 saying stdout must hold exactly one JSON document

### Requirement: check runs every command twice
`opm-docs check` SHALL run each repository command twice and SHALL exit 2, naming the argv, when the two outputs differ.

#### Scenario: A map printed in random order
- **WHEN** a dump program prints flags in map iteration order
- **THEN** `check` fails saying a docs build must be deterministic
