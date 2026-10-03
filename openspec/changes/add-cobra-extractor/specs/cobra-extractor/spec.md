## ADDED Requirements

### Requirement: cobradump prints the command tree a CLI shows
The `cobradump` module SHALL provide `Write(root, w, opts)` printing one `docs.opmodel.dev/cobradump/v1` document: the root's name, short and raw long text, usage line and global (root persistent) flags, and every available command recursively with its path, name, use, usage line, short text, raw long and example text, aliases, whether it runs, its own flags and its inherited flags other than the global ones. It SHALL add cobra's default completion command and resolve inherited flags before walking, skip hidden and deprecated commands, the help command, hidden flags and deprecated flags, drop a deprecated shorthand, write a flag default under the home directory with `~`, and sort commands and flags by name. It SHALL depend on cobra and pflag only.

#### Scenario: Completion is listed and help is not
- **WHEN** the cli's root is dumped
- **THEN** `commands` holds `completion` and no `help`

#### Scenario: Home in a default
- **WHEN** a flag defaults to `/home/u/.opm/config.cue` and the home directory is `/home/u`
- **THEN** its `default` is `~/.opm/config.cue`

#### Scenario: Two runs agree
- **WHEN** the same binary dumps twice
- **THEN** both outputs are byte-identical

### Requirement: cobradump prints pins
`WritePins(w, pins)` SHALL print one `docs.opmodel.dev/pins/v1` document with the given project-to-version map, keys sorted, and SHALL refuse a version starting with `v` or one that is not SemVer.

#### Scenario: A v-prefixed pin
- **WHEN** `WritePins` is given `{"core": "v2.0.0-beta.1"}`
- **THEN** it returns an error naming `core` and writes nothing

### Requirement: The cobra source renders the command reference
A `cobra` source SHALL run its command (C14), validate the dump, write `data/cobra.json` (`docs.opmodel.dev/data/cobra/v1`), and render `<section>_index.md` with the global flags and `<section>opm-<top>.md` for each top-level command holding that command and every command under it, depth first, each under a heading whose anchor is the command path in kebab case. The section SHALL be one of the bundle's owned paths. The pages SHALL match the cli's `internal/cmdref` output for the same command tree, except that no page carries a generator marker comment.

#### Scenario: A subcommand link
- **WHEN** `opm module` lists `apply`
- **THEN** its subcommand table links `/docs/reference/cli/opm-module/#opm-module-apply`

#### Scenario: An unknown dump schema
- **WHEN** the command prints a document with schema `docs.opmodel.dev/cobradump/v2`
- **THEN** `build` exits 2 naming the schema and the one this `opm-docs` reads

#### Scenario: Parity with cmdref
- **WHEN** the cli tree that first carries the hook is built with the planned config
- **THEN** every page equals cmdref's page for that tree with the marker comments removed
