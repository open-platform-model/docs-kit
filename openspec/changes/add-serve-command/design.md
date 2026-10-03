# Design: add-serve-command

## Context

DESIGN.md's command table lists `opm-docs serve`: "Builds and serves one repository's bundle on a local Hugo, for an author previewing their pages." Contracts read: C6 (`docs-kit.cue`), C7 (`--local`), C15 and C16 (docs placement, site versions), C21 (sections, once `add-enhancements-bundle` ships; until then `serve` knows tab and docs placements). The site's preview interface today: opmodel.dev `task bundles:pull` honors `OPM_BUNDLES_LOCAL="<project>@<segment>=<dir> ..."` and `task serve` serves the result (opmodel.dev `Taskfile.yml`, `site/scripts/run-in-image.sh`).

## Goals / Non-Goals

**Goals:** a one-command preview with live rebuild, and a path into the real site.

**Non-Goals:** reproducing the site's theme inside docs-kit; publishing anything; serving bundles pulled from a registry (that is the site's own preview).

## Decisions

### D1. Syntax

```text
opm-docs serve [--config docs-kit.cue] [--project P]... [--source .] [--port 1313] [--site <dir> [--version vM.m]]
```

| Flag | Type, default | Meaning |
|---|---|---|
| `--config` | path, as `build` | the config |
| `--project` | string, repeatable; default every project | projects to build |
| `--source` | path, `.` | the source tree |
| `--port` | int, `1313` | the embedded site's port (ignored with `--site`) |
| `--site` | path, none | an opmodel.dev checkout: serve through it (D4) |
| `--version` | site version, none | the site version a docs bundle previews in; required with `--site` when a docs bundle is built |

Exit codes: `0` when stopped by an interrupt, `1` usage (unknown flag, missing `--version`, no or too old `hugo`, no `task` with `--site`), `2` when the first build fails (later failures while watching are printed and the last good bundle stays).

### D2. The embedded site

`internal/serve/site/` (embedded with `go:embed`): `hugo.toml` (no theme, `markup.goldmark.renderer.unsafe = false`, `disableKinds` for taxonomies), `layouts/` with a single base template rendering the title, description, body and a sidebar of the section's pages, a `render-blockquote-alert.html` for the five alert types, and `layouts/_shortcodes/opm/<figure>.html` for the seven figure names as a labelled placeholder box. It is a preview of content, not of the site's design. `serve` writes it into a temporary directory and adds one Hugo mount per bundle: a tab bundle's `content/` at `content/<root without leading slash>edge/`, a docs bundle's at `content/docs/`, a section bundle's at its root. `hugo server --source <tmp> --port <port> --bind 127.0.0.1` runs from `PATH`; `serve` refuses a `hugo` older than 0.146.0 (the template layout the skeleton uses) and prints the URL.

### D3. Watching

`serve` polls every file under each source's paths (the `markdown` `dir`, an extractor's module, package or `dir`) once a second, comparing size and modification time; on a change it rebuilds that project into a staging directory and, when the build passes, swaps it into the mounted directory, so Hugo's own watcher reloads. A failed rebuild prints its errors and leaves the mounted bundle as it was. A repository command (C14) reruns with its source; its own inputs are the source tree's files, so a change under the source tree triggers it.

### D4. Site mode

With `--site <dir>`: build once into a temporary directory, then run `task bundles:pull` and `task serve` with working directory `<dir>` and `OPM_BUNDLES_LOCAL` set to the space-separated pairs `<project>@edge=<tree>` (tab and section bundles) and `<project>@<--version>=<tree>` (docs bundles; `pull-docs-placement` D6), and wait for `task serve`. No watching: the site's own preview is not designed for a changing local tree. This relies on opmodel.dev keeping those two task names and the variable; the interface is recorded in `docs/contracts.md` "Site decisions" so a site change names it.

## Departures in implementation (2026-10-03)

docs-kit `main` moved after this design was written (authored docs and edit paths, four extractors, site versions in `pull`, the C14 runner). Re-read against it, the implementation departs as follows; the decisions above stand otherwise.

- **D1.** `--version` without `--site` and a `--port` outside 1 to 65535 exit `1`. `hugo server` exiting on its own (a port in use) exits `2` and says to pass `--port`. The address is fixed: there is no `--bind` flag, the skeleton listens on `127.0.0.1` only.
- **D2, the config.** The skeleton's config is `hugo.json`, written with `encoding/json`, instead of `hugo.toml`: mount paths are absolute temporary paths, and JSON needs no hand-written quoting. It also passes `--baseURL http://127.0.0.1:<port>/ --appendPort=false`.
- **D2, templates.** Hugo 0.146's layout: `layouts/baseof.html`, `single.html`, `list.html`, and one `layouts/_markup/render-blockquote.html` that renders the five alert types (and a plain blockquote), instead of a separate `render-blockquote-alert.html`.
- **D2, figures.** Dialect 1 now allows nine figure names, not seven. The placeholder shortcodes are not embedded files: `serve` writes one per name from the dialect's own list (`dialect.Figures`), so a figure added to the dialect is previewed without a change here.
- **D2, sections.** C21 (`section` placement) is not on `main` yet. Mounting is "a tab at `<root>edge/`, a docs bundle at `/docs/`, any other placement at its root", so a section bundle mounts at `/enhancements/` when `add-enhancements-bundle` lands, with no change to `serve`.
- **D3, the swap.** A passing rebuild is copied file by file over the served tree (a changed file rewritten, a removed one deleted) instead of swapping the directory: Hugo watches the mounted directory, and a replaced mount root loses its watch.
- **D3, what is watched.** Per source kind: a `markdown` `dir`, a `cue-catalog` `module`, a `cue-definitions` `package`, a `crd` `dir` and `samples`. A `cobra` source, a bundle with `pins` (both run a repository command), and a kind `serve` does not know (the extractors in flight) watch the whole source tree. `docs-kit.cue` is always watched; directories starting with `.` (`.git`, `.cue-cache`) never are. After a rebuild the roots are read again, so a source added to the config is watched.
- **The lint.** Every preview build is `build.Run`, unchanged: the same extractors, repository commands (C14, run once per build as `build` runs them) and bundle-mode dialect lint. A page `build` refuses is never served; while watching, its violations print as `build` prints them and the last good build stays. C14's "Runs in" row and the `repository-commands` spec gain `serve`.
- **Shared code.** Two additions, each in a file of its own so parallel changes merge cleanly: `build.Selected` (the config and projects `Run` would build, with `Run`'s usage errors, so `serve` refuses a bad flag before it builds or looks for `hugo`) and `dialect.Figures`.
- **Dependencies.** None added: polling (D3's decision), `os/exec` for `hugo` and `task` (no shell, fixed argv), `encoding/json` for the config, `go:embed` for the skeleton.

## Research & Decisions

### Embedded skeleton versus the site's theme (decided in planning)

**Options considered**:
1. Only `--site`: real look, but every author needs an opmodel.dev checkout and its build image, and no live reload.
2. Only an embedded skeleton: instant and self-contained, but not the real look.
3. Both: the skeleton by default for the edit loop, `--site` for a final check.
**Decision**: option 3.
**Rationale**: the edit loop needs speed and no second checkout; the theme check needs the site, which already has a preview path.

### Polling instead of an fs-notify dependency (decided in planning)

**Decision**: polling once a second. Source trees are small, and it adds no dependency or platform-specific code.

## Risks / Trade-offs

- The skeleton can accept a page the site renders differently (a figure placeholder, the sidebar order); the lint is the same, so nothing that fails the site passes `serve` silently.

## Durable decisions

- "Commands": `serve` (D1). "Site decisions": the site-mode interface (D4).
- `README.md`: "Preview your pages".
