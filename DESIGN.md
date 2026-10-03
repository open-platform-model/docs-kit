# docs-kit design

docs-kit builds each OPM repository's documentation into a versioned OCI artifact, a *docs bundle*, in that repository's own CI. The site, opmodel.dev, pulls the bundles it needs and builds from them. It reads no source repository and generates nothing itself.

The repository holds one Go program, `opm-docs`, and the shared workflow that runs it. This document is the design for building both, and for moving the site onto them.

Status: proposed, 2026-10-02. Nothing is built yet.

## Why

Today each of four repositories carries its own reference generator: cli (`hack/cmdref`), opm-operator (`hack/crdref`), catalog_opm (`tools/refgen`) and core (`tools/refgen`). That is about 5,800 lines of Go that solve the same problems four times: doc-comment cleanup, citation stripping, Markdown escaping, marker blocks and staleness checks. Each commits its generated pages to git, so every change to a doc comment needs a regenerate step, and catalog_opm's release workflow has to regenerate pages on the release branch.

The site, in turn, reads every source repository through git. It resolves release lines from tags, archives trees, computes git dates and lints every page with a shell copy of the page dialect. It cannot show more than one version of a single repository's pages side by side, which the Catalogs tab needs: one entry per catalog MAJOR.MINOR, switchable on its own.

docs-kit moves both jobs to the producer. Each repository builds, lints and publishes its docs as an artifact when it changes. The site only chooses versions and assembles them.

## Decisions

Made by the owner on 2026-10-02. Decisions 13 to 16 were approved by the owner later the same day, while phase 1 was planned (`openspec/changes/archive/2026-10-02-build-opm-docs-phase-1/`); where they differ from the text below, the text has been amended to match. Decisions 17 to 19 were made by the owner on 2026-10-03, while phases 1b, 2 and 3 were planned ([docs/orchestration.md](docs/orchestration.md)).

| # | Decision |
|---|---|
| 1 | The tool and its workflow live in `open-platform-model/docs-kit`. |
| 2 | Docs ship as OCI artifacts, built and pushed by each repository's CI. |
| 3 | `opm-docs` carries its own OCI client (oras-go) and does both the push in CI and the pull for the site. |
| 4 | Scope is all three phases: generated reference, then every repository's reference, then authored pages too. |
| 5 | `main` publishes an `edge` bundle, so the site can show unreleased docs. |
| 6 | A fix to released docs ships as a docs-only revision of that release, never by overwriting it. |
| 7 | The bundle is its own artifact under its own path, so its tags follow docs-kit's scheme, not CUE's module rules. |
| 8 | The Catalogs tab starts at the first opm release published after catalog_opm adopts docs-kit; earlier minors are not backfilled. Amended 2026-10-03: originally "starts at opm 4.4", but every earlier tag holds the catalog at `opm/` while `main` moved it to `src/`, so a backfill with `main`'s config cannot build. Until that release the tab shows only `edge`. |
| 9 | A site version follows release tags, never branch heads: it shows the newest release of its line plus that release's docs revisions. No bundle is published from a release branch. |
| 10 | A site version shows the docs of the versions its cli pins (library, core, opm-operator), as it does today. |
| 11 | Third-party catalogs in the Catalogs tab are a possible future extension, out of scope for now. |
| 12 | Version history in the Catalogs tab is badges and a per-member change list only; no side-by-side diff view. |
| 13 | Phase 1's done criterion is split: phase 1 ends when the site shows the first released opm minor and edge in the Catalogs tab from pulled, verified bundles; the docs-revision half (`opm-docs revise` and a revision reaching the site) is the follow-up change `add-docs-revisions`. |
| 14 | The site runs `opm-docs pull` inside its build image, with network on for that step only, instead of on the host. |
| 15 | `org.opencontainers.image.created` is the source commit's time, not the build time, so a rebuild of the same commit gives the same digest. |
| 16 | `push` and `promote` are separate commands: `push` writes only the immutable full tag, the workflow signs the digest, then `promote` verifies the signature and moves the moving tags. |
| 17 | The opm repository publishes releases with release-please, and its docs bundle ships on each release, as every other repository's does. |
| 18 | The enhancements repository is bundled too: one edge-only, unversioned bundle placed at `/enhancements/`. With it, the site build reads no git repository except its own. |
| 19 | "Edit this page" on a docs page links the page's source file on `main`, whichever version the page shows. |

Carried over from the site's existing rules: reference facts are generated in the repository that owns their source; a generated entry states only what its source proves; pages follow the page dialect in the workspace `STYLE.md` ("Site Pages").

## Overview

```text
 producer CI (each repository)
 ┌───────────────────────────────────────────────────────────────┐
 │ opm-docs build                                                │
 │   extract:  CUE · CRD YAML · cobra (via hook) · Go · Markdown  │
 │        ─► data/*.json (the doc model)                          │
 │   render:  data ─► content/**/*.md (page dialect)              │
 │   lint:    the page dialect, before anything is published      │
 │   pack:    manifest.json + data/ + content/ ─► one layer       │
 │ opm-docs push   ─► ghcr.io/open-platform-model/docs/<project>   │
 │ cosign sign (keyless, GitHub OIDC)                             │
 └───────────────────────────────┬───────────────────────────────┘
                                 ▼
 opmodel.dev build
   image:  opm-docs pull  (resolve tags ─► verify signature ─► pin digest ─► unpack)
           writes site/.bundles/<project>/<version>/ and a lock file
           (in the build image, network on for this step only; decision 14)
   Docker: Hugo mounts the unpacked bundles, offline, as it mounts source trees today
```

## The bundle

### Where it lives

One OCI repository per project, under a `docs/` path so the bundles never mix with CUE modules or images:

```text
ghcr.io/open-platform-model/docs/catalog-opm
ghcr.io/open-platform-model/docs/core
ghcr.io/open-platform-model/docs/cli
ghcr.io/open-platform-model/docs/opm-operator
ghcr.io/open-platform-model/docs/library
ghcr.io/open-platform-model/docs/opm          (phase 3: the authored-only repository)
```

The packages must be public. A container package pushed from Actions starts private, so making each one public is a one-time setup step, as it was for the CUE modules.

### Format

An OCI 1.1 image manifest with an artifact type and one layer:

| Part | Value |
|---|---|
| `artifactType` | `application/vnd.opmodel.docs.bundle.v1` |
| config | the empty descriptor, `application/vnd.oci.empty.v1+json` |
| layer | `application/vnd.opmodel.docs.bundle.layer.v1.tar+gzip`: the bundle tree below |

Annotations on the manifest, so a client can choose a bundle without downloading it:

| Annotation | Example |
|---|---|
| `org.opencontainers.image.version` | `4.4.5` (the release; `edge` for main) |
| `org.opencontainers.image.revision` | the source commit |
| `org.opencontainers.image.source` | `https://github.com/open-platform-model/catalog_opm` |
| `org.opencontainers.image.created` | the source commit's time, RFC 3339 (decision 15) |
| `dev.opmodel.docs.revision` | `0`, `1`, ... (decision 6) |
| `dev.opmodel.docs.project` | `catalog-opm` |
| `dev.opmodel.docs.dialect` | `1` (the page-dialect version the content passed) |
| `dev.opmodel.docs.tool` | the `opm-docs` version that built it |

### Layout

```text
manifest.json        what the bundle holds and where it goes (below)
content/             Hugo pages in the page dialect, paths relative to the placement root
data/                the doc model as JSON, one file per kind (members, commands, crds, defs)
```

`content/` is what the site mounts. `data/` is what the pages were rendered from. It lets the site add features that read across versions, such as "added in 4.4" or a member's changes between two minors, without re-parsing Markdown.

`manifest.json` is defined by a CUE schema in docs-kit (`schema/manifest.cue`) and validated on build and on pull:

```cue
#Manifest: {
	schema:   "docs.opmodel.dev/bundle/v1"
	project:  string                  // "catalog-opm"
	version:  string                  // "4.4.5" or "edge"
	revision: int & >=0               // docs revision (decision 6)
	source: {
		repo:   string                // "open-platform-model/catalog_opm"
		commit: string                // full SHA the bundle was built from
		ref:    string                // "opm-v4.4.5", or "main" for edge
	}
	tool:    string                   // opm-docs version
	dialect: int                      // page-dialect version
	placement: #Placement
	pages: [...#Page]
}

// Where the site mounts content/.
#Placement: {
	// "docs": merged into a site version's /docs/ tree, as docs/site/ is today.
	// "tab":  its own section with its own versions, e.g. /catalogs/opm/4.4/.
	kind: "docs" | "tab"
	root: string                      // "/docs/" or "/catalogs/opm/"
}

#Page: {
	path:     string                  // relative to content/
	source?:  string                  // repo file the page came from, for "Edit this page"
	lastmod?: string                  // the source file's last commit date, RFC 3339
	generated: bool                   // generated reference, or authored
}
```

The site uses `pages` for the information it takes from git today: "Last updated", "Edit this page" and "View source at".

## Tags and versions

| Tag | Example | Points at | Moves |
|---|---|---|---|
| release + revision | `4.4.5.0`, `4.4.5.1` | one immutable build | never |
| release | `4.4.5` | the newest revision of that release | on a docs revision |
| minor | `4.4` | the newest revision of the newest 4.4.x | on a patch or revision |
| major | `4` | the newest revision of the newest 4.x | on a minor, patch or revision |
| edge | `edge` | the latest build of `main` | on every push to main |

A prerelease such as `1.0.0-beta.5` keeps its full version, plus the revision: `1.0.0-beta.5.0`. Moving tags follow the same rule, so `1.0` points at the newest `1.0.x` docs, prereleases included, as the site's `cli-line` does today.

Four dot-separated numbers sort in order: `4.4.5.1` comes after `4.4.5.0` and before `4.4.6.0`. A SemVer suffix such as `-docs.1` would sort *before* `4.4.5`, and OCI tags cannot hold the `+` of SemVer build metadata. The site never parses meaning out of a tag: it reads the annotations.

The site resolves by a moving tag (`4.4`) and records the digest it got in its lock file. A build is reproducible from the lock even after the tag moves.

### Docs revisions

A fix to released docs, for example a wrong doc comment in opm 4.4.5:

1. The fix lands on `main` as usual. `edge` shows it at once.
2. A `workflow_dispatch` in the source repository takes the release tag (`opm-v4.4.5`) and the fix commit.
3. The workflow checks the fix is on `main`, applies it to the release tree and checks it changes documentation only (below).
4. It builds the bundle and pushes `4.4.5.<next>`, then moves `4.4.5`, `4.4` and `4` where they should point.

"Documentation only" is checked by the tool. A Markdown file may change freely. A `.cue` or `.go` file may change only in comments: the tool parses both versions and compares their syntax trees with comments removed. A fix that changes code is refused; it needs a patch release.

## The tool: `opm-docs`

### Commands

| Command | Does |
|---|---|
| `opm-docs build` | Reads `docs-kit.cue` in the repository, runs the extractors and the renderer, lints the result and writes the bundle tree to `out/`. |
| `opm-docs lint` | Lints a directory of pages against the page dialect. `build` runs it, and so does the site on pull. |
| `opm-docs check` | Builds into a temporary directory and fails if it does not lint. The PR gate. |
| `opm-docs push` | Packs `out/` and pushes it under its immutable full tag only (decision 16). |
| `opm-docs promote` | After the workflow has signed the pushed digest, verifies the signature and moves the moving tags (above) to it (decision 16). |
| `opm-docs pull` | For the site: resolves tags, verifies signatures, pins digests, unpacks, writes the lock. |
| `opm-docs revise` | The docs-revision steps 3 and 4 (change `add-docs-revisions`, decision 13). |
| `opm-docs serve` | Builds and serves one repository's bundle on a local Hugo, for an author previewing their pages. |

### Configuration

Each repository has one file, `docs-kit.cue`, validated against a schema in docs-kit. CUE fits: every OPM repository already uses it, and the schema documents itself.

```cue
project:   "catalog-opm"
placement: {kind: "tab", root: "/catalogs/opm/"}
version:   {from: "tag", prefix: "opm-v"}       // release version from the git tag

sources: [
	{kind: "cue-catalog", module: "./opm"},     // members, transformers, served-by
	{kind: "markdown",    dir: "docs/site"},     // authored pages (phase 3)
]
```

### Extractors

Each extractor turns one kind of source into the doc model in `data/`. The renderer never reads source.

| Kind | Reads | Replaces | Generic? |
|---|---|---|---|
| `cue-catalog` | a catalog module, evaluated with the CUE Go API: members, fulfilment, `optional`, served-by from the transformers, doc comments, and each spec as structured fields (path, type, default, required) beside its formatted CUE | catalog_opm `tools/refgen` | yes |
| `cue-definitions` | a CUE package's exported definitions, parsed with their doc comments | core `tools/refgen` | yes |
| `crd` | controller-gen CRD YAML, plus `config/samples` | opm-operator `hack/crdref` | yes |
| `cobra` | a JSON dump of a cobra command tree | cli `internal/cmdref` | needs a hook |
| `go-api` | a Go package's exported API with `go/doc` | nothing yet (library, phase 2) | yes |
| `markdown` | authored pages in the page dialect, copied with their git dates | the site's git archives (phase 3) | yes |

**The cobra hook.** A program's command tree exists only inside the program, so the tool cannot read it from source or from a built binary. docs-kit ships a small Go package, `docskit/cobradump`. The cli imports it from a hidden command or a `hack/` program that prints the tree as JSON, and the `cobra` extractor reads that JSON. This is the only docs-kit code a repository imports; it has no dependencies beyond cobra.

The shared parts of today's four generators become one package each: doc-comment cleanup (`WHY` blocks, citations, contributor-only notes), Markdown escaping, CUE formatting for spec blocks, and the derived marks (Not implemented, Provided by your platform). Rules the four agreed on stay as they are: an entry states only what the source proves, and enforcement tags appear only where derivable.

### Rendering

The renderer turns the doc model into pages using Go templates embedded in the tool, one per entry kind. All projects render the same way, so a presentation change happens once and reaches every bundle built with the new tool version.

Links between pages of one bundle are written for the bundle's placement and version, for example `/catalogs/opm/4.4/traits/backup/`. A link into another project uses that project's stable alias, such as `/catalogs/opm/4/` for "the newest 4.x catalog".

### The page dialect

The dialect lint moves from `opmodel.dev/site/scripts/lint-sources.sh` (shell and awk, with a byte-identical copy in an OpenSpec change) into the tool, in Go, versioned with it. A bundle records the dialect version it passed. The site runs the same lint again on pull, as a guard rather than as the first check.

## Publishing

A reusable workflow in docs-kit, `.github/workflows/publish.yml`, called by each repository and pinned to a docs-kit release:

| Trigger in the source repository | Publishes |
|---|---|
| pull request | nothing: runs `opm-docs check` |
| push to `main` | `edge` |
| a release is published | `<version>.0`, then moves the release, minor and major tags |
| `workflow_dispatch` (docs revision) | `<version>.<n>`, then moves the tags |

Each pushed manifest is signed with cosign, keyless, using the workflow's GitHub OIDC identity. `opm-docs pull` verifies that the signature's identity is the expected repository and workflow before it unpacks anything.

Credentials: push uses the workflow's `GITHUB_TOKEN` with `packages: write`. Pull is anonymous, since the packages are public.

## The site

### Pull and lock

The site gains one step, run inside its build image with network on for that step only (decision 14), before the offline build:

```text
opm-docs pull --config site/bundles.cue --out site/.bundles --lock site/.bundles/lock.json
```

`site/bundles.cue` replaces the source-repository rows of `site/versions.conf` phase by phase. It names, per site version and per tab, which project and which tag to pull. The lock records every digest, and the build stamp records the lock. A local preview of unpublished work uses `opm-docs build` in the source repository and points the site at its `out/` directory, so authors keep a local loop with no registry.

### The Catalogs tab (phase 1)

- **URL:** `/catalogs/opm/<MAJOR.MINOR>/`, outside every site version, like `/enhancements/`.
- **Versions:** every opm minor from the first release published after adoption on (decision 8). Each minor is the bundle tag `<MAJOR.MINOR>`. The list comes from the registry's tags, so a new minor appears with no site commit. `edge` appears as "main (unreleased)".
- **Order:** by MAJOR.MINOR, newest first.
- **Switcher:** a version menu on every catalog page. It keeps the reader on the same member when that member exists in the chosen version, otherwise it goes to the nearest parent. This is the logic of the site-version switcher, applied to the tab.
- **Aliases:** `/catalogs/opm/` and `/catalogs/opm/<MAJOR>/` redirect to the newest minor. Docs pages link through these aliases.
- **Landing:** the authored catalog contract page, which leaves Reference.
- **Search and indexing:** only the newest minor of each major is indexed. Older minors and `edge` are `noindex` and stay out of `llms.txt`, as draft enhancements do.

### Checks

The site's checks keep running on what it built: page set, links, stray files, supply chain, accessibility and screenshots. A bundle that fails the dialect lint or its signature check fails the site build and names the project, version and digest.

## Phases

### Phase 1: the tool and the Catalogs tab

- docs-kit: `build`, `lint`, `check`, `push`, `promote`, `pull`; the `cue-catalog` extractor; the renderer; the manifest schema; the publish workflow; signing and verification.
- catalog_opm: `docs-kit.cue`, the workflow call, `tools/refgen` and its committed pages removed, the release-workflow regeneration step removed.
- opmodel.dev: `opm-docs pull` in its build image, the Catalogs tab with its switcher, aliases and index rules; catalog pages leave Reference.
- Done when: the site shows the first released opm minor and edge in the Catalogs tab from pulled, verified bundles (decision 13). A docs revision of a released minor reaching the site without a catalog release is the done criterion of the follow-up change `add-docs-revisions`.

### Phase 1b: version history in the Catalogs tab

When a second minor (4.5) exists, `opm-docs pull` compares the `data/` of every pulled minor and writes one history file for the tab; the member templates read it.

| Badge | Derived from |
|---|---|
| Added in 4.5 | the member's FQN first appears in 4.5 |
| Changed in 4.6 | its structured spec differs from the previous minor: a field added or removed, a default changed, a field made required |
| Removed in 4.7 | present in 4.6, absent in 4.7; that minor links to the last one that had it |
| Newer version | the same name and kind at a later apiVersion (`backup@v1alpha1`, `backup@v1beta1`): the pages link each other |

Badges sit under the member's title and on each changed spec field ("New in 4.6", "Default changed in 4.6"), and the page ends with a "Changes in 4.6" list of what differs from the previous minor. There is no side-by-side diff view (decision 12).

The first published minor is the oldest with a bundle, so a member present in it reads "in <minor> or earlier", never "added in <minor>". Estimate: 2 to 3 days. The structured spec it needs is extracted from phase 1 on.

### Phase 2: every generated reference

- docs-kit: `cue-definitions`, `crd`, `cobra` (with `cobradump`) and `go-api` extractors.
- core, opm-operator, cli: their generators and committed pages removed; bundles placed in `/docs/` of the site version.
- library: an API reference for the first time.
- Done when: no repository commits generated pages, and the site's two Reference placeholders are gone.

### Phase 3: authored pages

- docs-kit: the `markdown` extractor with git dates; `opm-docs serve`.
- Every repository's `docs/site/` ships in its bundle; the opm repository gets one, released with release-please (decision 17); the enhancements repository gets an edge-only bundle (decision 18).
- opmodel.dev: site versions resolve from bundle tags instead of git; `materialise.sh`, the git-date step and the shell lint are removed.
- Done when: the site build reads no git repository except its own.

| Phase | Estimate |
|---|---|
| 1 | 1.5 to 2 weeks |
| 2 | 1 to 2 days per repository |
| 3 | about 1 week |

## Risks

- **The registry becomes a build dependency.** If GHCR is down, the site cannot pull. Mitigation: CI caches pulled bundles by digest, and the lock lets a build reuse the cache.
- **A bad tool release breaks every bundle it builds.** Bundles record the tool version, and the workflow pins a docs-kit release, so a repository upgrades on purpose and can roll back.
- **Generated pages leave git,** so a reader can no longer browse them on GitHub. The site and `opm-docs serve` are the places to read them.
- **GHCR's handling of OCI 1.1 artifacts is assumed, not tested.** GHCR stores Helm charts, Flux artifacts and our CUE modules, but this design also relies on `artifactType` and annotations surviving a push and a tag listing. Phase 1 starts with a spike that pushes, lists and pulls a bundle before anything else is built on it.
- **Phase 3 changes how site versions resolve.** Today a site version follows git tags and reads docs from release-branch heads. It will follow bundle tags only (decision 9): v1.0 shows the cli bundle `1.0` and the bundles of what that cli tag pins (decision 10). A docs fix reaches a site version through a docs revision, no longer through a branch push.

## Open questions

None. Answered on 2026-10-02 and moved to Decisions: release branches (decision 9), what a site version pins (decision 10), third-party catalogs (decision 11), version-history scope (decision 12).
