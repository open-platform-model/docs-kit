# Design: add-cobra-extractor

## Context

Builds on `generalize-build-assembly`: the registry, docs placement, repository commands (C14), `pins` (C15) and `publish.yml`'s `setup-go` input. The behavior to reproduce is the cli's `internal/cmdref` (`render.go` walk and pages, `text.go` Long/Example parsing, `sync.go` orphans) and `hack/cmdref/main.go` (`NewRootCmd`, `InitDefaultCompletionCmd`, home directory). Its output, `cli/docs/site/reference/cli/`, is wholly generated: `_index.md` (global flags) and `opm-<command>.md` per top-level command, anchors `#opm-module-apply`. The site serves it at `/docs/reference/cli/`.

## Goals / Non-Goals

**Goals:** `cobradump` and its release; the dump and pins formats; the `cobra` kind and its pages; parity with cmdref.

**Non-Goals:** reading a command tree any other way; non-cobra CLIs; man pages.

## Decisions

### D1. `cobradump` (the Go API, C19)

```go
// Package cobradump prints a cobra command tree, and a program's pins, as
// the JSON documents opm-docs reads. It depends on cobra and pflag only.
package cobradump // module github.com/open-platform-model/docs-kit/cobradump

// Options adjusts a dump.
type Options struct {
	// Home is replaced by "~" at the start of a flag default; "" leaves
	// defaults as they are. Write fills it from os.UserHomeDir when unset
	// and NoHome is false.
	Home   string
	NoHome bool
}

// Write prints root's tree as one docs.opmodel.dev/cobradump/v1 document.
// It calls root.InitDefaultCompletionCmd and resolves every command's
// inherited flags before walking, as cobra does on Execute.
func Write(root *cobra.Command, w io.Writer, opts Options) error

// WritePins prints one docs.opmodel.dev/pins/v1 document: project name to
// exact version, without "v".
func WritePins(w io.Writer, pins map[string]string) error
```

`cobradump` lives in `cobradump/` with its own `go.mod` (`go 1.26.0`, `github.com/spf13/cobra`, `github.com/spf13/pflag`), so a caller's module graph gains nothing else. The root module never imports it; the extractor reads only its JSON.

The cli's program, as its sibling change writes it (`hack/docskit-dump/main.go`; a `hack/` program, not a hidden command, so the shipped binary carries no docs code):

```go
func main() {
	if len(os.Args) > 1 && os.Args[1] == "pins" {
		// Illustrative names: the sibling change reads each pin where the
		// site's resolve-versions.sh reads it today (library from the cli's
		// go.mod via debug.ReadBuildInfo, core from the library's default
		// schema module, the operator from the cli's pinned operator version).
		check(cobradump.WritePins(os.Stdout, map[string]string{
			"library": libraryVersion(), "core": coreVersion(), "opm-operator": operatorVersion(),
		}))
		return
	}
	check(cobradump.Write(cmd.NewRootCmd(), os.Stdout, cobradump.Options{}))
}
```

### D2. The dump format (C19)

```json
{
  "schema": "docs.opmodel.dev/cobradump/v1",
  "name": "opm",
  "short": "...", "long": "<raw>", "useLine": "opm [command]",
  "globalFlags": [#Flag],
  "commands": [#Command]
}
```

```text
#Command: {path: "opm module apply", name: "apply", use: "apply [path]", useLine: "opm module apply [path] [flags]",
           short, long (raw), example (raw), aliases: [string], runnable: bool,
           flags: [#Flag]            // LocalNonPersistentFlags plus this command's own persistent flags
           inheritedFlags: [#Flag]   // InheritedFlags minus the root's persistent flags (those are globalFlags)
           commands: [#Command]}
#Flag:    {name, shorthand: "" | one letter, type: Value.Type(), default: DefValue (home as "~"), usage}
```

Included: every available command (`IsAvailableCommand()`), so hidden and deprecated commands and the help command are skipped, and the default completion command is present. A hidden flag or one with a non-empty `Deprecated` is skipped; `ShorthandDeprecated` drops only the shorthand. Commands are sorted by name, flags by name. Strings are raw (no trimming beyond cobra's own). Every list is present, empty or not. Two runs of one binary print the same bytes; `check` enforces it (C14).

### D3. Configuration (C6, C19)

```cue
#Cobra: {
	kind:        "cobra"
	command:     #Command                // C14 argv printing the dump: ["go", "run", "./hack/docskit-dump"]
	section:     =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$" // an owned directory: "reference/cli/"
	title:       string & !=""           // the section index
	description: string & !=""
	weight?:     int & >=1
	citations:   *"strip" | "link"
}
```

The cli's file, as its sibling change writes it:

```cue
bundles: cli: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/cli/"]}
	version: {from: "tag", prefix: "v"}
	pins: {command: ["go", "run", "./hack/docskit-dump", "pins"], projects: ["library", "core", "opm-operator"]}
	sources: [{
		kind:        "cobra"
		command:     ["go", "run", "./hack/docskit-dump"]
		section:     "reference/cli/"
		title:       "CLI Reference"
		description: "Every opm command and flag, generated from the CLI's cobra commands."
		weight:      2
	}]
}
```

and its `docs.yml` calls `publish.yml` with `setup-go: true`.

### D4. Data model and pages

The extractor validates the dump (schema id, shape) and writes it to `data/cobra.json` with schema `docs.opmodel.dev/data/cobra/v1`, adding `section`, `title`, `description` and `weight` from the config and a `page` per top-level command (`opm-<name>`). The renderer ports cmdref:

| Path | Page |
|---|---|
| `<section>_index.md` | front matter `title`, `description`, `weight`; the root's Long (parsed), its usage line, `## Global flags` table (Flag, Shorthand, Type, Default, Description) |
| `<section>opm-<top>.md` | front matter `title: "opm <top>"`, `description` = Short, `type: reference`; the global-flags pointer; per command, depth first, `## <path>` with Short, usage fence, aliases, parsed Long (prose, preformatted blocks, examples), `**Subcommands**` table, flags tables |

Anchors are the command path in kebab case (`#opm-module-apply`), links `/docs/<section>opm-<top>/#<anchor>`. Long and Example parsing (`text.go`: prose versus preformatted by indentation, examples split, code words joined, fences lengthened past backtick runs, shortcodes escaped) is docs-kit's, so a parsing fix reaches the cli at its next docs-kit bump with no cli release.

### D5. Release of `cobradump`

`release-please-config.json` gains a second package; the root package keeps its settings (C12 unchanged):

```json
"cobradump": {
  "release-type": "go",
  "component": "cobradump",
  "include-component-in-tag": true,
  "tag-separator": "/",
  "initial-version": "0.1.0",
  "draft": false,
  "changelog-path": "CHANGELOG.md"
}
```

so its tags are `cobradump/v0.1.0`, the form the Go module proxy needs for a nested module, and its release is published at once (the root package's draft-first flow exists only for goreleaser's assets). Commits touching `cobradump/` use the scope `cobradump` (added to the constitution's scopes). `release.yml` runs goreleaser only when the root package's release was created (release-please's `release_created` output for path `.`), never for `cobradump--release_created`. The tag rulesets (`tags-immutable`, `tags-create-app-only`) cover the new tags as they cover `v*`. The signer glob `refs/tags/v[0-9]*` does not match `cobradump/v...`, so a `publish.yml` ref at that tag never verifies.

`Taskfile.yml`: `test`, `vet` and `lint` also run in `cobradump/` (`go -C cobradump test ./...`); `ci.yml` does the same. A docs-kit test reads `cobradump/testdata/dump.golden.json` (written by `cobradump`'s own test from a fixture tree) as the extractor's input, so the producer and the consumer of the format are tested against one file.

### D6. Commands

No new command or flag. Messages: "the dump from `go run ./hack/docskit-dump` has schema `x`; this opm-docs reads `docs.opmodel.dev/cobradump/v1`" (exit 2).

## Research & Decisions

### Hook as a hidden command or a `hack/` program (decided in planning)

**Options considered**: 1. hidden `opm __docs` command - the shipped binary carries docs code and `cobradump`; 2. `hack/docskit-dump` program - same module (it imports `internal/cmd`), nothing in the binary.
**Decision**: option 2.

### Nested module versus a package in the root module (decided in planning)

**Context**: importing `github.com/open-platform-model/docs-kit/...` from the root module would add CUE, oras-go and sigstore-go to the cli's module graph.
**Decision**: nested module with its own release component, as DESIGN.md says ("no dependencies beyond cobra").

### Raw strings in the dump (decided in planning)

**Decision**: the dump carries `Long` and `Example` raw; parsing stays in docs-kit's renderer, so presentation fixes ship with docs-kit, not with the cli.

## Risks / Trade-offs

- `go run` in a release build needs the tag's Go toolchain; `setup-go` reads `src/go.mod` and `GOTOOLCHAIN=auto` covers a newer directive.
- A help-text fix is a Go string change, which a docs revision refuses (C3 "Docs revisions"); it waits for the next cli release.

## Durable decisions

- C19 (new): `cobradump` API (D1), dump format (D2), config (D3), pages and anchors (D4), parity record.
- C12: the `cobradump` component and its tag form (D5); the constitution's commit scopes gain `cobradump`.
- `AGENTS.md`: layout tree (`cobradump/`), releasing (two components).
