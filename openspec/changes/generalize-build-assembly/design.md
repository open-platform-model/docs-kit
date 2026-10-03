# Design: generalize-build-assembly

## Context

Contracts read: C1 (projects), C3 (`manifest.json`, `#Placement` already allowing `kind: "docs"`), C5 (the workflow), C6 (`docs-kit.cue`), C8 (URLs and links), C11 (dialect, docs and bundle modes), "Doc-comment rules". The cross-repo sequence for phases 1b, 2 and 3 is `docs/orchestration.md`; the four extractor changes (`add-cue-definitions-extractor`, `add-crd-extractor`, `add-cobra-extractor`, `add-go-api-extractor`) and `pull-docs-placement` start when this change is merged.

## Goals / Non-Goals

**Goals:** a registry any extractor plugs into; building and linting docs-placed bundles; completable pages; `include`; the citation policy; repository commands; pins in the manifest; the workflow job split; the project names of every later bundle.

**Non-Goals:** any new extractor (four sibling changes); pulling docs bundles (`pull-docs-placement`); authored `docs/site` trees in bundles beyond `include` (phase 3, `add-authored-docs`).

## Decisions

### D1. The registry

```go
package build // internal/build/source.go

// An Extractor turns one configured source into one data file.
type Extractor interface {
	Kind() string // the docs-kit.cue source kind, "cue-catalog"
	Extract(ctx context.Context, in Input) (Data, error)
}

// Input is what an extractor may read: the source tree, its own config
// entry (decoded by the extractor), the build identity and the command runner.
type Input struct {
	Source   string            // the --source tree
	Config   cue.Value         // this source's entry in docs-kit.cue
	Version  string            // "4.4.5" or "edge"
	Outside  bool              // the config came from outside the source tree (a backfill)
	Commands *command.Runner   // repository commands (D6)
	Doc      doctext.Policy    // the citation policy (D5)
}

// Data is one file under data/.
type Data struct {
	File   string // "catalog.json"
	Schema string // "docs.opmodel.dev/data/cue-catalog/v1"
	Bytes  []byte
	// Sources maps a page path the renderer will write to the repo file it
	// documents, for manifest.json pages[].source and lastmod.
	Sources map[string]string
}
```

```go
package render

// A Renderer turns one data file into pages. It reads the data file as
// written, never the extractor's value (Principle I).
type Renderer interface {
	Schema() string
	Render(data []byte, t Target) ([]Page, error)
}

type Page struct {
	Path        string // under content/
	Body        string
	Completable bool   // an authored page at Path may come first (D3)
	Heading     string // the first heading of the generated body, refused in an authored page that completes it
}

type Target struct {
	Kind    string // "tab" or "docs"
	Root    string // "/catalogs/opm/" or "/docs/"
	Segment string // a tab's "4.4" or "edge"; "" for docs
	Edge    bool
	Version string
	Repo    string
	Commit  string
}
```

`Target.URL(page)` is `<root><segment>/<page>/` for a tab (unchanged) and `/docs/<page>/` for docs. Registration is a static table in `internal/build`; `#Source` in `schema/config.cue` is the union of the registered kinds, and an extractor change adds its kind to both. One source per extractor kind per bundle (a second is refused naming both entries); the data file name is fixed per kind (`catalog.json` stays as C10 has it; later kinds write `<kind>.json`). Sources run in config order; renderers run in the same order after all extractors; the `write` collision check is unchanged.

### D2. Docs placement in build

`docs-kit.cue` and `manifest.json` share the placement (C3, C6):

```cue
#Placement: {
	kind: "tab" | "docs"
	if kind == "tab" {root: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"}
	if kind == "docs" {
		root: "/docs/"
		// Paths under content/ this bundle owns exclusively: a directory
		// ending "/" or a page ending ".md". pull refuses another bundle of
		// the same site version with a page in one (C16).
		owns: *[] | [...#Owned]
	}
}

#Owned: =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$|^([a-z0-9]+(-[a-z0-9]+)*/)*[a-z0-9]+(-[a-z0-9]+)*\\.md$"
```

Rules for a docs bundle (build exits 2 on each, naming the page and the config):

- every page a renderer writes (`generated: true`) lies under an owned path ("`reference/cli/opm-module.md` is generated but `cli` owns only `reference/definitions/`; add it to `placement.owns`");
- two owned paths of one bundle do not nest;
- a docs bundle carries no `cue-catalog` source (a catalog is a tab, C1).

`--release` on a docs bundle derives the version from the tag prefix as for a tab; the `metadata.version` check applies only to `cue-catalog`. A docs bundle's segment (`MAJOR.MINOR` or `edge`) means nothing for URLs: its pages publish under the site version that pulls it (C16).

**Bundle-mode lint of a docs bundle** (`lint --bundle`, run by `build` and `pull`): the docs-mode link rules (C11: `/catalogs/` links only through the bare root or a major) plus: a `/docs/` link whose path falls under an owned path names a page of the bundle (`/docs/reference/cli/opm-module/` is `reference/cli/opm-module.md` or `reference/cli/opm-module/_index.md`); fragments are not checked. A `/docs/` link outside the owned paths is not checked by the bundle: it points into another bundle or a site page, and the site's post-build link check covers it (decided in planning). The dialect version stays `1`: these are checks of bundle mode, not new page rules.

### D3. Completable pages

A renderer page with `Completable: true` and an authored page at the same path from a `markdown` source of the same bundle combine: the authored front matter and body come first, unchanged, then one blank line and the generated body. The authored body must not contain the generated page's `Heading` line (exit 2 naming the file, as the catalog landing does today). The combined page is `generated: false` with the authored file as `source` (C3). Without an authored page, the generated page stands alone with its own front matter. The catalog landing becomes the first completable page (its heading `## Catalog members`; its output unchanged). A completable page's generated body never carries front matter of its own when completed; the renderer returns both forms.

### D4. `markdown` `include`

```cue
#Markdown: {
	kind: "markdown"
	dir:  =~"^[^/.][^.]*$"
	// path.Match globs relative to dir, matched against each file's slash
	// path; "**" is not special. Only matching files are copied. Absent: all.
	include?: [string, ...string]
}
```

An `include` glob that matches nothing fails a normal build ("`include` `reference/operator-resource.md` matches no file under `docs/site`") and is ignored when the config came from outside the source tree (a backfill), like a missing `dir` (C5). A link-pinning rule (C8) applies only to tab bundles; a docs bundle's authored pages are copied as written.

### D5. The citation policy

Every extractor source takes `citations: *"strip" | "link"`. `strip` is today's rule ("Doc-comment rules", Citations). `link` rewrites each enhancement decision citation (`0010:D28`, `0010:D28:R2`, `0010:D28/D29`, a list of them) into a Markdown link per entry, `[0010:D28](/enhancements/0010/decisions/)`, the link text being the citation as written; every other citation form (OQ numbers, `SPEC.md § N`, experiments) is still removed. A citation inside a code span or a spec block's comment is stripped in both modes (a link cannot live there). The `/enhancements/<NNNN>/decisions/` form is already allowed by dialect 1. `markdown` sources take no `citations`: authored text is copied as written.

### D6. Repository commands (C14)

```cue
#Command: [string, ...string] // argv: argv[0] is looked up on PATH; no shell, no globbing
```

A build runs a command:

| Aspect | Rule |
|---|---|
| Directory | the source tree (`--source`; a revision's patched worktree) |
| Environment | the build's own, plus `OPM_DOCS=1`, `OPM_DOCS_PROJECT=<project>`, `OPM_DOCS_VERSION=<version or edge>` |
| stdin | empty |
| stdout | exactly one JSON document, at most 16 MiB; anything after it is an error |
| stderr | passed through to the build's stderr |
| Timeout | 10 minutes; then killed, exit 2 ("`go run ./hack/docskit-dump` ran over 10 minutes") |
| Failure | a non-zero exit is exit 2 naming the argv, the project and the status |
| Determinism | `check` runs every command twice and fails, naming the argv, when the two outputs differ ("a docs build must be deterministic; see docs-kit C14"). `build` runs it once. |

Each output document names its schema (`"schema": "<id>"`), and the consumer refuses an unknown one. A command runs repository code: it runs only in `build`, `check` and `revise`, never in `push`, `promote` or `pull`.

### D7. Pins (C15)

```cue
#Bundle: {
	...
	// The exact versions of other projects this build documents against
	// (DESIGN decision 10). The command prints a pins document.
	pins?: {
		command:  #Command
		projects: [#Project, ...#Project]
	}
}
```

The command prints

```json
{"schema": "docs.opmodel.dev/pins/v1", "pins": {"library": "1.0.0-beta.1", "core": "2.0.0-beta.1", "opm-operator": "1.0.0-beta.4"}}
```

and `build` checks the keys are exactly `projects` and each value matches C3's `#SemVer` (no `v`); then `manifest.json` gains

```cue
#Manifest: {
	...
	pins?: [#Project]: #SemVer
}
```

A backfilled release (config from outside the tree) whose tree cannot run the command fails: pins are a contract, never guessed. A build without `pins` writes none.

### D8. The workflow job split

`publish.yml` becomes two jobs, keeping its inputs, outputs and caller permissions (C5):

| Job | Runs | Permissions it declares | Repository code |
|---|---|---|---|
| `build` | checkout(s); install `opm-docs`; GHCR read login; `setup-go` when asked; `check`, `build` or `revise`; upload `out/<project>/` as a workflow artifact (not in `check` mode) | `contents: read`, `packages: read` | yes |
| `publish` | `needs: build`, not in `check` mode; install `opm-docs`; download the artifact; `push`; `cosign sign`; `promote` | none declared (inherits the caller's `packages: write`, `id-token: write`) | none: no checkout of the caller |

The concurrency groups of C5 move to `publish`; `build` has none. A new input:

```yaml
setup-go:
  description: Install Go (actions/setup-go, pinned by SHA) from the source tree's go.mod, for repository commands that run Go (docs-kit C14)
  type: boolean
  default: false
```

`go-version-file` is `src/go.mod` in `release` mode and `go.mod` otherwise. The artifact name is `docs-bundle-${{ inputs.project }}-${{ github.run_id }}`, retention 1 day.

### D9. Project names (C1)

C1's table becomes the registry of every planned project, so the parallel changes need not edit it:

| Project | Repository | Tag prefix | Placement | Phase |
|---|---|---|---|---|
| `catalog-opm` | catalog_opm | `opm-v` | tab `/catalogs/opm/` | 1 |
| `core` | core | `v` | docs, owns `reference/definitions/` | 2 |
| `opm-operator` | opm-operator | `v` | docs, owns `reference/operator-resources.md` | 2 |
| `cli` | cli | `v` | docs, owns `reference/cli/`, carries `pins` | 2 |
| `library` | library | `v` | docs, owns `reference/go-api/` | 2 |
| `catalog-opm-docs` | catalog_opm | `opm-v` | docs (its `docs/site/`) | 3 |
| `opm` | opm | `v` | docs (its `docs/site/`) | 3 |
| `enhancements` | enhancements | none (edge only) | section `/enhancements/` | 3 |

Naming rule (decided in planning): a repository's docs-placed bundle is named after the repository (`_` becomes `-`); when that name is already a tab project of the same repository, it takes the suffix `-docs`. Hence `catalog-opm-docs`.

### Commands, exit codes, messages

No new command or flag. `build` and `check` exit `1` for a config error (an unknown source kind, `owns` that nest, `pins.projects` empty), `2` for an execution error (a command failure, an output that does not validate, a generated page outside `owns`, a completable heading collision). Every message names the file and field or the page, and the fix.

## Research & Decisions

### Where repository code runs (decided in planning)

**Context**: the cobra extractor and pins run repository code (`go run`), in a job that today also holds `id-token: write`.
**Options considered**:
1. One job, as today - simplest; a compromised dependency of the caller could request an OIDC token and sign a bundle for that project.
2. Build and publish jobs, the bundle tree passed as an artifact - the signing job runs no repository code; costs one artifact round trip.
**Decision**: option 2.
**Rationale**: it keeps the signing identity away from code docs-kit does not control, at no contract cost (inputs and outputs unchanged), and `push` packs deterministically from the tree whatever job it runs in.

### Pins in the manifest or in `data/pins.json` (decided in planning)

**Options considered**: 1. `manifest.json` `pins` - validated with the manifest, read by `pull` before anything else; 2. `data/pins.json` - keeps the manifest small, but `data/` is the doc model renderers read, and pull would read a data file for trust decisions.
**Decision**: option 1.

### Cross-bundle `/docs/` links (decided in planning)

**Context**: a docs bundle cannot know the other bundles of the site version it lands in.
**Decision**: bundle-mode lint checks only links into the bundle's own `owns`; the site's existing post-build link check covers the rest. `pull` checks page collisions and ownership (C16), not links.

### Data file per kind (decided in planning)

**Decision**: `data/<kind>.json`, except `catalog.json`, which C10 already fixes. One source per kind per bundle until a second has a consumer (Principle VI).

## Risks / Trade-offs

- A command's nondeterminism is caught only by `check`, which runs it twice; a release build runs it once. The PR check runs on every change, so drift shows there first.
- `owns` granularity is a directory or a page; a finer claim (an anchor) is not expressible and not needed by any planned bundle.

## Durable decisions

- C1 project table and naming rule: `docs/contracts.md` C1.
- `placement.owns`, `pins` in the manifest: C3; config additions (`include`, `citations`, `pins`, the registry): C6; C8 gains the docs-bundle URL form `/docs/<page>/`.
- Repository commands: `docs/contracts.md` C14 (new). Docs placement build and lint rules, pins: C15 (new).
- The job split and `setup-go`: C5.
- `README.md`: the source-kind table and how to add an extractor (the registry); `AGENTS.md` layout tree gains `internal/command`.
