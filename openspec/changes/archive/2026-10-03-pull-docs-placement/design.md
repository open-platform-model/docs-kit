# Design: pull-docs-placement

## Context

Contracts read: C3 (manifest, `placement.owns` and `pins` from `generalize-build-assembly`), C4 (tags; a release tag names the newest revision of that release), C7 (pull config, layout, lock, `--frozen`, `--offline`, `--local`), C9 (signing identity), C15 (docs placement). DESIGN decisions 9 (a site version follows release tags and shows that release's docs revisions) and 10 (it shows the docs of what its cli pins). Today's site logic being replaced: `opmodel.dev/site/scripts/resolve-versions.sh` (library from the cli's `go.mod`, core from the library's default schema module, the operator from the cli's pinned operator version).

## Goals / Non-Goals

**Goals:** pull docs-placed bundles per site version; resolve pins; check one version's bundles against each other; lock and reproduce it.

**Non-Goals:** the site's mounts and templates (opmodel.dev); cross-bundle `/docs/` link checking (the site's post-build check, `generalize-build-assembly`); the enhancements section (`add-enhancements-bundle`); computing pins (the cli prints them, C15).

## Decisions

### Pull config (C7, C16)

```cue
#Pull: {
	registry: *"ghcr.io/open-platform-model/docs" | string
	signer: {...}                        // unchanged
	tabs: [#Project]: {...}              // unchanged
	// Projects that may be placed in a site version's /docs/, each with the
	// only repository allowed to sign it (C9's Source Repository URI).
	docs: [#Project]: {repo: =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$"}
	versions: [#SiteVersion]: {
		// The bundle that chooses the others (DESIGN decision 10).
		anchor: {project: #Project, tag: #DocsTag}
		// Pulled at the exact version the anchor's manifest.json pins.
		pinned: *[] | [...#Project]
		// Pulled by their own tag, not by a pin.
		tags: [#Project]: #DocsTag
	}
}

#SiteVersion: =~"^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"
// A minor tag ("1.0"), a major tag ("4"), an exact release ("2.0.0-beta.1") or "edge".
#DocsTag: =~"^((0|[1-9][0-9]*)(\\.(0|[1-9][0-9]*))?|edge)$" | #SemVer
```

Checked by `pull` before any network call (exit 1, naming the version and project): every project in `anchor`, `pinned` or `tags` is a key of `docs`; a project appears once per version; no project is both a tab and a docs project. opmodel.dev's file, as its sibling change writes it for phase 2:

```cue
docs: {
	cli:            {repo: "open-platform-model/cli"}
	core:           {repo: "open-platform-model/core"}
	library:        {repo: "open-platform-model/library"}
	"opm-operator": {repo: "open-platform-model/opm-operator"}
}
versions: "v1.0": {
	anchor: {project: "cli", tag: "1.0"}
	pinned: ["library", "core", "opm-operator"]
}
```

Phase 3 adds `opm` and `catalog-opm-docs` under `docs` and, for v1.0, `tags: {opm: "1.0", "catalog-opm-docs": "4"}`.

### Resolution

Per site version, in this order:

1. **Anchor.** Resolve `anchor.tag` in `<registry>/<project>`, verify (C9 with `docs[project].repo`), fetch, unpack to an incoming directory, lint in bundle mode. The resolved build must be in the tag's line: a minor tag names a build of that MAJOR.MINOR, a major tag of that MAJOR, an exact release that version, `edge` an edge build (else exit 2, as C4's consumer rule for tabs).
2. **Pins.** Read the anchor's `manifest.json` `pins`. Each `pinned` project must have a pin (else exit 2: "cli 1.0.0-beta.6 (sha256:...) pins no version of core; pinned projects need a pin in the anchor's manifest"). Its tag is the pin itself, the release tag, which C4 moves to the newest revision of that release, so docs revisions follow (DESIGN decision 9). The resolved build's version must equal the pin. A missing tag is exit 2: "cli 1.0.0-beta.6 pins core 2.0.0-beta.1, and ghcr.io/open-platform-model/docs/core has no bundle for it; publish it (core's docs workflow, dispatch mode: release, tag v2.0.0-beta.1)".
3. **Own tags.** Each `tags` entry resolves like the anchor.

Every bundle's placement must be `kind: "docs"` (else exit 2 naming both). Edge has no special case: an anchor at `edge` pins exact releases like any other build.

### Layout and replacement

```text
<out>/
  lock.json
  catalog-opm/4.5/ ...                    tabs, unchanged
  _versions/
    v1.0/
      cli/           manifest.json  content/  data/
      core/          ...
      library/       ...
      opm-operator/  ...
```

`_versions` cannot collide with a project (C1 names have no `_`). A site version is replaced whole: every bundle of it unpacks and lints into `<out>/_versions/.incoming-<v>/<project>/`, the checks across one version run over the incoming set, and only then does `.incoming-<v>` replace `<v>` (the previous tree moves aside to `.outgoing-<v>` first and is removed once the new one is in place); any failure leaves the previous `<v>` in place and fails the pull. The sweep removes any `_versions/<v>` not in the config and any project directory not written this run.

### Checks across one version

Over the docs bundles of one site version (exit 2, naming the version, the path and both projects with their versions), in this order, so the most specific message wins:

- two bundles owning overlapping paths (one equal to or under the other);
- a page under a path another bundle `owns` ("core 2.0.0-beta.1 has `reference/cli/x.md`, under `reference/cli/`, which cli owns");
- a `content/` path present in two bundles ("`reference/_index.md` is in both cli 1.0.0-beta.6 and core 2.0.0-beta.1"), or two paths that serve one URL (`cli.md` and `cli/_index.md`).

Every comparison is by the URL Hugo serves (decided in review): `.md` is dropped, then a trailing `_index` or `index`, and an owned page `x.md` holds `x/` as an owned directory does.

Links across bundles are not checked here; the site's post-build link check covers them (decided in `generalize-build-assembly`).

### Lock (C7, C16)

`#Lock` gains an optional key after `bundles` (and after `history` when that key exists):

```cue
docs?: [...#DocsLocked]

#DocsLocked: #DocsPulled | #DocsLocal

#DocsPulled: {
	site:       #SiteVersion
	project:    #Project
	role:       "anchor" | "pinned" | "tag"
	tag:        #DocsTag                  // the tag resolved: "1.0", "2.0.0-beta.1", "4", "edge"
	repository: string & !=""
	digest:     =~"^sha256:[0-9a-f]{64}$"
	version:    #Version
	revision:   int & >=0
	commit:     #SHA
	dialect:    int & >=1
	builtBy:    #SemVer
	signer: {workflow: string & !="", repository: string & !="", ref: "refs/heads/main"}
	pins?:      [#Project]: #SemVer       // the anchor only: its manifest's pins as read
	dir:        string & !=""             // "_versions/v1.0/cli"
}

#DocsLocal: {
	site: #SiteVersion, project: #Project, role: "anchor" | "pinned" | "tag", local: true
	version: #Version, revision: int & >=0, commit: #SHA, dialect: int & >=1, builtBy: #SemVer
	pins?: [#Project]: #SemVer
	dir: string & !=""
}
```

Key order as listed; entries sorted by `site` (numeric MAJOR, MINOR), then role (`anchor`, `pinned`, `tag`), then `project`. The key is omitted when the config has no `versions`. Schema id stays `lock/v1`.

### `--local`, `--frozen`, `--offline`

- `--local <project>@<site-version>=<dir>` (the segment `v<MAJOR>.<MINOR>` marks a docs project; `<MAJOR>.<MINOR>` and `edge` stay tab segments): that project of that version comes from the tree; no signature, same validation, guards, lint and checks across one version. A local tree's version is not checked against a tag's line (decided in implementation: an author's edge build previews as the anchor, and the pre-release pin check in `docs/orchestration.md` runs `--local cli@v1.0` on a tree built before the release). A local anchor's pins drive the pinned projects, each of which may be local or pulled. A local pinned tree's version must equal the pin (exit 1 naming both).
- `--frozen <lock>`: each `docs` entry is fetched by digest as written; its `site`, `project` and `role` must be in the config with that role, and its `repository` must be `<registry>/<project>` (exit 1 otherwise, as C7 for tabs); an anchor or `tags` entry's tag must equal the config's, and a pinned entry's version and tag must equal the locked anchor's `pins`; a configured project the lock does not name exits 1. The config digest check (C7) runs first, so a role check fires only on a lock edited by hand. A local docs entry in a frozen lock is refused (re-run with `--local`).
- `--offline` with `--frozen`: cache only, as C7.

### Commands

No new command or flag; `pull`'s existing flags gain docs semantics (see "`--local`, `--frozen`, `--offline`"). Exit codes as C7: `1` for config and usage errors (the pull-config checks, a bad `--local`, a frozen lock that does not fit), `2` for resolution, signature, lint and cross-bundle failures.

## Research & Decisions

### Generated config versus `versions` in `bundles.cue` (decided in planning)

**Context**: the lock's `config` digest is the SHA-256 of `bundles.cue`, and the trust policy lives there.
**Options considered**: 1. `versions` in `bundles.cue` - one reviewed file holds trust and selection; 2. `--pin project=version` flags generated by the site - trust split between a file and a script.
**Decision**: option 1.

### Pins resolve through the release tag (decided in planning)

**Options considered**: 1. the full tag `<pin>.0` - immutable but misses docs revisions; 2. the release tag `<pin>` - the newest revision of exactly that release.
**Decision**: option 2, which is DESIGN decision 9's "plus that release's docs revisions". The lock records the digest, so a build is still reproducible.

### Site version replaced whole (decided in planning)

**Context**: the checks across one version span bundles, so a per-bundle swap (as tabs do per segment) could leave a version mixing old and new bundles that collide.
**Decision**: swap per version.

## Risks / Trade-offs

- A newly published cli release whose pins lack bundles breaks the site's next pull; the publishing order in `docs/orchestration.md` (dependencies first, cli last) and the frozen lock cover it.
- Moving tags make two pulls on different days differ; the lock and the build stamp record what was pulled, as for tabs.

## Durable decisions

- C16 (new) "Site versions": every decision above but "Commands".
- C7: the config's `docs` and `versions`, the `_versions/` layout, the lock's `docs` key, `--local` with a site version; its "Phase 2 adds `versions:`" note removed.
- `README.md`: `pull` and site versions.
