# cobra-extractor Specification

## Purpose
How a cobra CLI's command tree becomes a command reference: the cobradump module prints the tree and the pins as JSON from a program of the CLI, and the cobra source turns that dump into data/cobra.json and the reference pages (docs-kit C19).

## Requirements

### Requirement: cobradump prints the command tree a CLI shows
The `cobradump` module SHALL provide `Write(root, w, opts)` printing one `docs.opmodel.dev/cobradump/v1` document: the root's name, path, short text, raw long and example text, aliases, usage line, whether it runs, its local flags and its global (persistent) flags, and every available command recursively with its path, name, use, usage line, short text, raw long and example text, aliases, whether it runs, its own flags and its inherited flags other than the global ones. It SHALL add cobra's default completion command and resolve inherited flags before walking, skip hidden and deprecated commands, the help command, hidden flags and deprecated flags, drop a deprecated shorthand, refuse a shorthand that is not one printable ASCII character, write the home directory in a flag default as `~` only when the home directory is absolute and not the file-system root and the match is followed by a path separator or ends the default, and sort commands and flags by name. The v1 format is closed: any change to its fields is a new schema, `docs.opmodel.dev/cobradump/v2`. It SHALL depend on cobra and pflag only.

#### Scenario: Completion is listed and help is not
- **WHEN** the cli's root is dumped
- **THEN** `commands` holds `completion` and no `help`

#### Scenario: Home in a default
- **WHEN** a flag defaults to `/home/u/.opm/config.cue` and the home directory is `/home/u`
- **THEN** its `default` is `~/.opm/config.cue`

#### Scenario: A home that only prefixes a path
- **WHEN** the home directory is `/root` and a flag defaults to `/rootfs/x`
- **THEN** its `default` stays `/rootfs/x`

#### Scenario: The root as home
- **WHEN** the home directory is `/`
- **THEN** no default is rewritten

#### Scenario: Two runs agree
- **WHEN** the same binary dumps twice
- **THEN** both outputs are byte-identical

### Requirement: cobradump prints pins
`WritePins(w, pins)` SHALL print one `docs.opmodel.dev/pins/v1` document with the given project-to-version map, keys sorted, and SHALL refuse a version starting with `v` or one that is not SemVer.

#### Scenario: A v-prefixed pin
- **WHEN** `WritePins` is given `{"core": "v2.0.0-beta.1"}`
- **THEN** it returns an error naming `core` and writes nothing

### Requirement: The cobra source renders the command reference
A `cobra` source SHALL run its command (C14), validate the dump, write `data/cobra.json` (`docs.opmodel.dev/data/cobra/v1`), and render `<section>_index.md` with the global flags and `<section><root>-<top>.md` for each top-level command holding that command and every command under it, depth first, each under a heading whose anchor is the command path in kebab case. The section SHALL be one of the bundle's owned paths. Every command name SHALL be lower-case kebab-case. The pages SHALL match the cli's `internal/cmdref` output for the same command tree, except that no page carries a generator marker comment.

#### Scenario: A subcommand link
- **WHEN** `opm module` lists `apply`
- **THEN** its subcommand table links `/docs/reference/cli/opm-module/#opm-module-apply`

#### Scenario: An unknown dump schema
- **WHEN** the command prints a document with schema `docs.opmodel.dev/cobradump/v2`
- **THEN** `build` exits 2 naming the schema and the one this `opm-docs` reads

#### Scenario: Parity with cmdref
- **WHEN** the cli's dump at commit `ebf0479b`, frozen in the tests, is rendered with the planned config
- **THEN** every page equals the page cmdref committed at that commit with the marker comments removed
