## Context

The workspace design (owner-approved) is: pages write `{{opm:<project>.version}}`; the opmodel.dev
workflow fills them when it collects the docs bundles, before Hugo, from the versions it collected
for each site version; lint refuses unknown names in each repo; the site build fails on a leftover;
the controller module version, which no docs bundle carries, is recorded in the controller bundle
from the release manifest. This change is the docs-kit part.

Facts read in this tree:

- The dialect lint matches shortcodes only as `{{<` or `{{%` (`internal/dialect/page.go`,
  `reShortcode`). `{{opm:x}}` is not matched, so today it passes unexamined. Spike result: the case `placeholder-known` (the placeholder in prose, in a fenced `bash` block, in an `https:` link destination and in the `description`) lints clean, and `TestCheckAllowsPlaceholder` shows the markup check refuses none of the four forms, in both modes.
- `opm-docs pull` stages each site version in `internal/pull/versions.go`: `stageVersion` resolves the
  anchor, the pinned projects and the tag projects, runs `checkOverlap`, and leaves the unpacked,
  linted trees under `<out>/_versions/.incoming-<v>/<project>/`. `swapVersions` then replaces each
  version whole. The site's `task bundles:pull` is this command; `gen-docs-bundles.sh` runs after it
  and reads the lock and the manifests.
- `manifest.json` is closed (`schema/manifest.cue`), `pins` is the precedent for a map of versions.
- `revise` builds the release tag's tree plus fixes that change only `*.md`, and comments in `*.cue`
  and `*.go` (`docs/contracts.md`, "Docs revisions", step 5). A `.json` file is refused.

## Goals / Non-Goals

**Goals:** one placeholder form; names checked in the author's repository and at pull; the fill per
site version; a module version a page can name; every file and test named.

**Non-Goals:** a fill in `opm-docs build` or `serve` (a local preview shows the raw placeholder; owner
decision); a value other than `version`; a placeholder in a generated page; an escape for a literal
`{{opm:`; changing the opmodel.dev workflow itself (it only bumps `opm-docs`); changing the
opm-controller pages, markers or tests.

## Research & Decisions

### Where the fill runs

**Context**: the design says "the opmodel.dev workflow, at the step that collects the bundles, before
Hugo runs". The brief asks whether that step is `opm-docs pull` or a site script.
**Explored**: `site/Taskfile.yml` `bundles:pull` runs `opm-docs pull`, which verifies, unpacks and lints
every bundle; `gen-docs-bundles.sh` only turns the lock and manifests into JSON for Hugo. Only `pull`
knows the full bundle set of a site version at the moment before Hugo reads the trees.
**Options considered**:
1. In `opm-docs pull`, after `checkOverlap`, before the swap. One implementation in docs-kit, tested
   offline with the in-process registry; the site needs only a tool bump. The tree on disk differs
   from the verified layer (by design, and only inside `_versions/`); the lock still records the
   verified digests.
2. A site script after `pull`. Duplicates the name table and the version lookup in a shell script
   that docs-kit cannot test; two places to change when a name is added.
**Decision**: option 1. The fill and the leftover check are in this change. Task T7.3 (opmodel.dev)
only bumps `opm-docs` and adopts it in the site; it adds no fill code.
**Rationale**: the approved decisions (fill at collection time, from the versions collected, both
checks) all hold; the name list "lives in docs-kit" and now so does the only code that uses it.

### Name to value

**Decided in planning.** A name is one of two kinds, fixed in `internal/placeholder`:

| Name | Kind | Value |
| --- | --- | --- |
| `cli` | bundle | `version` of the site version's `cli` docs bundle |
| `core` | bundle | `version` of its `core` docs bundle |
| `library` | bundle | `version` of its `library` docs bundle |
| `opm-controller` | bundle | `version` of its `opm-controller` docs bundle |
| `catalog-opm` | bundle | `version` of its `catalog-opm-docs` docs bundle (the docs project of the catalog repository, C1 naming rule) |
| `opm-controller-module` | unit | `units["opm-controller-module"]` of the manifest of one docs bundle of the site version |

A bundle value is a bundle's own version, as pull collected it (the anchor's tag, a pin, or a tag
project). A unit value is an extra version a bundle recorded. If two docs bundles of one site version
record the same unit, the pull fails naming both. A unit name is configurable in `docs-kit.cue` only
for the names of kind unit.
Failures at the fill, each naming the site version, the page and the placeholder: the site version
pulls no bundle for the name (or no bundle records the unit); or the value is `edge` (a bare number
is promised, and `v` plus `edge` is wrong).

### Where the extra version is recorded

**Context**: "the bundle's data, beside the app version".
**Options considered**:
1. A new optional `units` map in `manifest.json`, next to `pins`. Pull already parses and validates the
   manifest; `gen-docs-bundles.sh` already reads manifests; no new file kind.
2. A new `data/units.json` with its own schema id. A second parse path in pull, and `data/` is
   listed per bundle in `manifest.data`.
**Decision**: option 1 (deviation in wording: "bundle data" is read as `manifest.json`; the
information is the same and sits where `pins` sits).
**Rationale**: smallest addition; closed-schema consequence is known and ordered by C12.

```cue
// schema/manifest.cue, in #Manifest, after pins
// Versions of released units of the repository that have no docs bundle of
// their own, read from a release manifest file (docs-kit.cue units).
units?: [#Unit]: #SemVer

#Unit: "opm-controller-module"
```

```cue
// schema/config.cue, in #Bundle, after pins
// A version read from a JSON file of the source tree: file[key] is an exact
// version without "v". file defaults to release-please's manifest.
units?: [#Unit]: {
	file: *".release-please-manifest.json" | =~"^[^/.]"
	key:  string & !=""
}
```

opm-controller then writes, in its `docs-kit.cue`:
`units: "opm-controller-module": key: "modules/opm_controller"` (the file in that repository holds
`"modules/opm_controller": "0.1.1"`).

`build` reads `file` from the source tree it builds (`--source`, or the checkout), requires a
string that matches `#SemVer`, and exits 2 naming the file and key when the file, the key or the
form is missing (message names the fix). The `file` path holds no `..` segment.

### The placeholder grammar and the lint rule

**Decided in planning.** Grammar: `{{opm:` NAME `.version}}`, where NAME is on the list; no spaces,
one field. Anything that begins `{{opm:` and does not match is a violation. The lint scans every
line, front matter and code fences included (the fill replaces text in all of them), and reports
each occurrence once per line position:

```text
docs/site/start/install.md:12: placeholder "{{opm:opm-controler.version}}": unknown name "opm-controler"; known names: cli, core, library, opm-controller, opm-controller-module, catalog-opm
docs/site/start/install.md:14: placeholder "{{opm:cli.tag}}": unknown field "tag"; write {{opm:cli.version}}
docs/site/start/install.md:15: placeholder "{{opm:cli": not closed or not of the form {{opm:<name>.version}}
```

A known placeholder produces no violation. The dialect version stays `1`; the rule joins C11 and
a conformance fixture records it. Docs-mode and bundle-mode lint apply it alike. A shown shortcode
(`{{</* */>}}`) is unaffected.

### Fill and leftover check in pull

`stageVersion` ends with `checkOverlap`. After it, `fillVersion(sv, set)` builds the value table
for that site version from `set` (bundle values from each `docsBundle.m.Version` by project, unit
values from each `m.Units`) and rewrites every page listed in each bundle's `manifest.json` under
`<incoming>/<project>/content/` that holds `{{opm:`, by a plain text replace of each well-formed
placeholder with its value, writing the file back with its mode kept. Only listed pages are touched;
`manifest.json` and `data/` are not. Because the lint ran when the bundle was unpacked, a page cannot
hold a malformed placeholder here; the fill still re-parses and fails on one, since a `--local` tree
is linted too but the check is cheap.

Then, still before any swap, `checkLeftover` scans every page of every staged tree (tab segments,
sections, site versions) for `{{opm:` and fails the pull with `<tree>/<page>: placeholder left after
the fill: "{{opm:core.version}}"; only docs bundles of a site version are filled`. So a placeholder
in a tab or section bundle fails the pull; a generated section never holds one (extractors escape
`{{`).

Order in `Run`: `stageAll`, `stageVersions` (which now fills), `histories`, `commit`. A failure
swaps nothing in, as for every other refusal. A `--frozen` or `--offline` pull fills the same way,
from the digests it takes; a `--local` docs tree is filled from its own manifest.

The lock does not change: it records the digest of the verified layer, not of the filled tree.
Because the fill makes the unpacked tree differ from the layer, the docs of `pull` say so (C7).

### Republishing on a module release

**Context**: the design says a module release republishes the controller bundle as a docs revision,
so the site sees the new module version. Today that cannot happen. A revision is the release tag's tree
plus doc-only fixes; the release manifest is a `.json` file, which the doc-only table refuses
(C "Docs revisions", step 5); and `revise` refuses a fix that leaves the tree unchanged. The module
release commit is not in the tag's tree. The controller module is released between controller
releases, so the controller bundle of the newest controller release would keep a stale module
version until the next controller release.
**Options considered**:
1. `revise --refresh-units`: no fix; keeps the newest revision's patch list; reads `units` from the
   checkout of `main` (the checkout `revise` already runs in; precedent: pages take their `edit`
   from `main`); publishes revision n+1. `publish.yml` revision mode takes `refresh-units: true`
   in place of `fix`.
2. Let a fix commit change the release manifest. Widens the doc-only table to a data file and still
   needs a commit that changes only that file.
3. Do not republish; the site reads the module version elsewhere (for example from the
   `opm_controller-v*` tag list). Contradicts the approved decision that the bundle records it.
**Decision**: option 1, proposed here and listed as a deviation: the approved design assumes
`revise` can do this, and it cannot. Owner decision needed at the proposal gate; if the owner
rejects it, section 5 is dropped and the module version is as fresh as the last controller release.
**Rationale**: option 1 keeps the tag scheme, signing and promote untouched and adds one flag and
one workflow input; the determinism rule holds (same inputs, same bytes) because `units` and the
patch list are inputs recorded in the manifest.

Behaviour: `opm-docs revise --project P --tag T --refresh-units` (exclusive with `--fix`) refuses
unless the newest published revision n of T's version is signed (as step 3) and the project's
`docs-kit.cue` on `main` declares `units`; it builds revision n+1 from T's tree with n's
`source.patches` applied and `units` read from `main`; it refuses with "units already current in
V.n" when the values equal n's. Exit codes as `revise`.

### Pull config, CLI and exit codes

No new flag on `lint`, `build` or `pull`. `build` exit 2 on a `units` failure; `lint` exit 2 on a
placeholder violation (as every dialect violation); `pull` exit 2 on a fill failure or a leftover.
`revise` gains `--refresh-units` (bool, false): usage error, exit 1, when given with `--fix`, or
without `--tag`/`--project`.

## Risks / Trade-offs

- A literal `{{opm:` in prose (a page that documents the placeholders) is refused by lint and has
  no escape. No page needs one now; add an escape when one does (YAGNI).
- A fill after verification means the unpacked tree is not byte-equal to the layer. Mitigation: the
  lock keeps the layer digest; the fill runs only on listed pages of site-version docs bundles.
- Older `opm-docs` binaries refuse a bundle with `units`. Mitigation: C12 ordering, in the proposal.
- The name table binds docs project names (`catalog-opm-docs`); renaming a docs project is a
  docs-kit change. Accepted: the name list is a docs-kit change by decision.

## Durable decisions

- The placeholder form, the name table and the name-to-value rule: stay in
  `docs/contracts.md` C11 and C16, and in the `version-placeholders` spec.
- `units` in the manifest and in `docs-kit.cue`: `docs/contracts.md` C3 and C6.
- `refresh-units` in `publish.yml`: `docs/contracts.md` C5 and the revision paragraph.
- Adoption order (site bumps first): `docs/orchestration.md`.
- Authoring rule "use `{{opm:<name>.version}}`, never a typed number": `docs/contracts.md` C11 only;
  repo authoring guides are the consumers' own text.
