# Design: build-opm-docs-phase-1

## Context

`DESIGN.md` is the approved design and its twelve decisions bind this change. Phase 1 builds `opm-docs` far enough to publish the opm catalog's reference as a signed OCI bundle from catalog_opm's CI, and to let opmodel.dev pull it into a Catalogs tab. Two sibling changes carry the other repositories' halves (`orchestration.md`). This document fixes every contract those siblings depend on, so they can be planned and built against it without reading docs-kit's code.

What exists today, and what phase 1 learns from:

| Generator | Lines | What phase 1 keeps |
|---|---|---|
| catalog_opm `tools/refgen` | 2,042 | Port target. Member enumeration from the catalog maps, served-by and blueprint matching, the two marks, the enforcement rows, the spec block (authored `spec:` plus in-module definitions it references), doc-comment cleanup, citation stripping, Markdown escaping. |
| core `tools/refgen` | 1,790 | Learn from: `SPEC.md §` and experiment-reference stripping, the colon-only citation form, the WHY line inside a doc comment, `{{<` refusal. Ported in phase 2. |
| cli `internal/cmdref` | ~1,180 | Learn from: shortcode escaping `{{</* */>}}`, line-start escaping. Phase 2 (`cobra` with `cobradump`). |
| opm-operator `hack/crdref` | 950 | Learn from: citations linked rather than stripped, schema-derived rules. Phase 2 (`crd`). |

The shared rules phase 1 ports, stated once so later extractors reuse them (package names in "Packages" below):

- **Maintainer comments.** Inside a definition, a comment group whose first line (after `//` and trimming) starts with `WHY` or with `//` (a `////` banner) is dropped. A `WHY` line inside a doc comment is dropped on its own (core's rule, adopted now).
- **Citations.** Removed from prose and from comments in spec blocks. The pattern accepts `0010:D28`, `0010 D28`, `OQ` numbers, `:R2` and `/R1/R2` requirement suffixes, `/D9` continuations, lists joined by `,`, `;` or `and`, an optional `(see|per|enhancement)` lead and the parenthesised form; then refgen's six clean-up rewrites (empty parens, comma before paren, space after paren, space before punctuation, double spaces, trim). Also removed: `See SPEC.md § N.` sentences, inline `SPEC.md § N`, `NNNN experiment N` and `enhancements/NNNN/experiments/...` (core's rules, adopted now). A changed comment paragraph in a spec block is re-wrapped to `80 - indent - 3` columns, minimum 40; an unchanged one keeps its breaks; a trailing comment that becomes empty is removed.
- **Summary.** A member's doc comment opens with `metadata.description` plus `.` (whitespace collapsed), or the build refuses the member naming both texts. The summary is the page's front-matter `description` and is not repeated in the body. The remaining paragraphs are the notes.
- **Escaping.** Outside backtick code spans: `\ < > * _ [ ] |` are backslash-escaped and `{{` becomes `{\{`. A table cell escapes every `|`, inside code spans too. A code span lengthens its fence past any backtick it holds. A YAML front-matter string escapes `\` and `"`. A rendered page that still holds `{{<` or `{{%` fails the build.
- **Doc-note links.** `docs/<name>.md` in prose becomes a link to that file on the source repository at the bundle's commit, only when the file exists and the text is outside a code span.
- **Marks.** A resource or trait that no transformer in its own catalog requires or optionally reads is marked **Provided by your platform** when its `fulfilment` default is `provider`, and **Not implemented** otherwise; a blueprint is never marked. Exact strings in "Page renderer".
- **Enforcement tags.** Only where derivable: the spec schema and each required match label are enforced by `cue`; a provider-fulfilled contract's single provider and a load-bearing trait's refused render are enforced by the `kernel`. Nothing else is tagged.

## Goals / Non-Goals

**Goals:**

- `opm-docs build`, `lint`, `check`, `push`, `promote` and `pull`, with the `cue-catalog` extractor, a minimal `markdown` source, the renderer and the dialect lint.
- The bundle format, the tag scheme, `docs-kit.cue`, the pull config and the lock, each with a CUE schema embedded in the tool.
- The reusable workflow `publish.yml`, cosign keyless signing in it, and signature verification in `pull` and `promote`.
- docs-kit's own CI and release, so a caller can pin `publish.yml@v0.1.0`.

**Non-Goals:**

- The `cue-definitions`, `crd`, `cobra` and `go-api` extractors, `cobradump`, `opm-docs serve`, git-date-aware authored pages for whole `docs/site/` trees (phases 2 and 3).
- Version history badges (phase 1b). Phase 1 only extracts the structured spec they need.
- Docs revisions: `opm-docs revise`, the documentation-only check and the workflow's `revision` mode move to the follow-up change `add-docs-revisions` (supervisor decision, 2026-10-02). Phase 1 keeps every contract a revision needs (the `revision` field and annotation, `source.patches`, the tag rules), so that change adds behavior without changing a contract.
- The raw Kubernetes catalog (`k8s/`, `opmodel.dev/catalogs/k8s@v1`). It is being removed from catalog_opm in a separate session; no bundle, tab or layout is planned for it.
- Any edit in catalog_opm or opmodel.dev. Their changes are specified in `orchestration.md`.

## Contracts

Everything in this section is read by another repository. A change to any of it follows constitution Principle II.

### C1. Where bundles live

```text
ghcr.io/open-platform-model/docs/<project>
```

| Project | Source | Release tag prefix | Placement |
|---|---|---|---|
| `catalog-opm` | catalog_opm `opm/` (`opmodel.dev/catalogs/opm@v4`) | `opm-v` | tab, `/catalogs/opm/` |

A project name matches `^[a-z0-9]+(-[a-z0-9]+)*$`. Phases 2 and 3 add `core`, `cli`, `opm-operator`, `library` and `opm` under the same path.

### C2. Media types and annotations

An OCI 1.1 image manifest, packed with `oras.PackManifestVersion1_1`:

| Part | Value |
|---|---|
| `artifactType` | `application/vnd.opmodel.docs.bundle.v1` |
| config | the empty descriptor, `application/vnd.oci.empty.v1+json` (`{}`) |
| layers | exactly one, `application/vnd.opmodel.docs.bundle.layer.v1.tar+gzip` |

Manifest annotations:

| Annotation | Value |
|---|---|
| `org.opencontainers.image.version` | the release version (`4.4.5`, `1.0.0-beta.2`) or `edge` |
| `org.opencontainers.image.revision` | the 40-hex source commit (`source.commit`) |
| `org.opencontainers.image.source` | `https://github.com/<owner>/<repo>` of the **calling** repository |
| `org.opencontainers.image.created` | the source commit's committer time, RFC 3339 UTC (decided in planning: DESIGN.md says build time, which would change the digest of every rebuild) |
| `dev.opmodel.docs.project` | the project, `catalog-opm` |
| `dev.opmodel.docs.revision` | the docs revision, decimal (`0` for edge) |
| `dev.opmodel.docs.dialect` | the page-dialect version the content passed, decimal |
| `dev.opmodel.docs.tool` | the `opm-docs` version that built it, without `v` |

`org.opencontainers.image.source` names the calling repository because GHCR links a new package to the repository named there, and the workflow's `GITHUB_TOKEN` can create a package only when that link is in its first push.

The layer is a gzip-compressed tar of the bundle tree (C3), built deterministically: entries sorted by path, directories before their contents, mode `0644` for files and `0755` for directories, uid and gid `0`, empty user and group names, modification time equal to `created`, no extended headers, gzip header with no name and no time. The same source, config and tool version give the same digest.

**Spike fallback (only if section 1 finds GHCR rejects the form above).** Outcome B: no `artifactType`, config media type `application/vnd.opmodel.docs.bundle.config.v1+json` whose content is `manifest.json`, the same layer and annotations. Both producer and consumer are `opm-docs`, so outcome B changes no sibling contract; the spike records which outcome holds in this section.

Signatures are cosign v3 Sigstore bundles (`--new-bundle-format=true`): a referrer manifest with `artifactType` `application/vnd.dev.sigstore.bundle.v0.3+json` whose subject is the bundle's digest. GHCR has no referrers API, so the referrer is stored under the fallback tag `sha256-<hex>`; every consumer ignores tags of that form when it lists versions.

### C3. Bundle tree and `manifest.json`

```text
manifest.json
content/      pages in the page dialect, paths relative to the bundle's URL root (C8)
data/         the doc model, JSON, one file per extractor kind (C10)
```

Nothing else is allowed at the top level. `schema/manifest.cue`, embedded in the tool, validates `manifest.json` on build, push and pull:

```cue
package schema

import (
	"list"
	"time"
)

#Manifest: close({
	schema:   "docs.opmodel.dev/bundle/v1"
	project:  #Project
	version:  #Version
	revision: int & >=0
	if version == "edge" {revision: 0}
	source: close({
		repo:   =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$" // "open-platform-model/catalog_opm"
		commit: #SHA                                  // the commit built (a revision: the release commit)
		ref:    string & !=""                         // the release tag "opm-v4.4.5", or the branch built ("main" for edge)
		dirty?: true                                  // built from a work tree with uncommitted changes; push refuses it
		// A docs revision only: the fix commits applied to the release tree, oldest first,
		// every earlier revision's fixes included (decided in planning). Phase 1 never writes
		// it, but its pull accepts it, so a site on 0.1.0 can read add-docs-revisions' bundles.
		patches?: [#SHA, ...#SHA]
	})
	tool:      #SemVer // the opm-docs version, without "v"
	dialect:   int & >=1
	placement: #Placement
	pages: list.MinItems(1) & [...#Page]
	data: [...#DataFile]
})

#Project:  =~"^[a-z0-9]+(-[a-z0-9]+)*$"
#SHA:      =~"^[0-9a-f]{40}$"
#SemVer:   =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
#Version:  #SemVer | "edge"

#Placement: close({
	// "tab": its own section with its own versions, /catalogs/<name>/<MAJOR.MINOR>/.
	// "docs": merged into a site version's /docs/ tree (phase 2; refused by phase-1 pull).
	kind: "tab" | "docs"
	if kind == "tab" {root: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"}
	if kind == "docs" {root: "/docs/"}
})

#Page: close({
	path:      =~"^([a-z0-9]+(-[a-z0-9]+)*/)*(_index|[a-z0-9]+(-[a-z0-9]+)*)\\.md$" // under content/
	source?:   string & !=""  // repo-relative file the page came from ("Edit this page", "View source")
	lastmod?:  time.Time      // that file's last commit date at the commit built, RFC 3339
	generated: bool           // generated reference, or an authored page
})

#DataFile: close({
	path:   =~"^[a-z0-9-]+\\.json$" // under data/
	schema: string & !=""           // the file's own schema id, e.g. "docs.opmodel.dev/data/cue-catalog/v1"
})
```

Tightened from DESIGN.md: closed structs, the SHA and SemVer patterns, `revision: 0` for edge, `patches`, typed `data` entries, and a placement root bound to its kind. `pages` lists every file under `content/`, and only those; `data` lists every file under `data/`, and only those. `lastmod` is set when the checkout has the file's history (the workflow checks out with full history) and omitted otherwise.

### C4. Tags

A **build** is one pushed manifest. Its identity is read from its annotations (`version`, `revision`), never parsed out of a tag name.

| Tag | Example | Points at | Moves |
|---|---|---|---|
| full | `4.4.5.0`, `4.4.5.1`, `1.0.0-beta.2.0` | one build: `<version>.<revision>` | never; written once |
| release | `4.4.5`, `1.0.0-beta.2` | the newest revision of that version | on a docs revision |
| minor | `4.4`, `1.0` | the newest build whose version has that MAJOR.MINOR, prereleases included | on a patch, a prerelease or a revision |
| major | `4`, `1` | the newest build with that MAJOR, prereleases included | on a minor, patch, prerelease or revision |
| edge | `edge` | the newest build of `main` | on every push to `main` |

Rules:

1. **Order.** Builds of released versions order by SemVer 2.0.0 precedence of `version`, then numerically by `revision`. `4.4.5.1` follows `4.4.5.0` and precedes `4.4.6.0`; `4.5.0-rc.1.0` follows `4.4.6.0`. Edge builds take no part in the order.
2. **Full tags are immutable.** `push` refuses to write a full tag that already names a different digest ("`4.4.5.0` is already published as sha256:...; a documentation fix is a docs revision"). The same digest is a no-op, so a re-run of a failed workflow is safe.
3. **Revision numbers.** A release's first build is revision `0`. A docs revision (change `add-docs-revisions`) takes `1 +` the highest revision published for that version, and is refused when revision `0` does not exist. Phase 1 publishes only revision `0`, but `promote` and `pull` already handle any revision.
4. **Edge** builds carry `version: "edge"`, `revision: 0` and no full tag; they are pushed by digest and reached only through `edge`.
5. **Moving a tag.** `promote --digest D` moves each moving tag of D's line to D only when D is the newest build of that tag's line, and only after D's signature verifies. Just before each move it resolves the tag's current build again and skips the move when that build is newer than D, because releases of different versions may publish at once (C5, Concurrency). It never points a tag at any other digest. Promote enumerates builds from every tag of the repository that equals `<version>.<revision>` of its own manifest's annotations; other tags (moving tags, `edge`, `sha256-*`) are not builds.
6. **No release-branch bundles.** Every publishing mode runs on `refs/heads/main` (DESIGN decision 9); `pull` checks the same ref in the signature (C9).

A consumer selects a tab's versions by tag name: every tag matching `^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$` (a minor tag) at or above the tab's `from`, plus `edge` when asked. It then trusts only the resolved manifest's annotations and `manifest.json`, and refuses a minor tag whose build's version is not in that minor.

### C5. The reusable workflow

File `.github/workflows/publish.yml` in this repository. A caller pins a docs-kit **tag**:

```yaml
uses: open-platform-model/docs-kit/.github/workflows/publish.yml@v0.1.0
```

Pinning a tag, not a SHA, is the plan (decided in planning): the signing certificate names the workflow at the ref the caller wrote, and `pull` accepts only `refs/tags/v*` (C9). docs-kit's tags are immutable once the owner adds docs-kit to the tag rulesets (`orchestration.md`, owner setup).

> **Known conflict, unresolved; left to review.** The org convention pins every action and reusable workflow by full commit SHA with the version in a comment (`@de0fac2e... # v6.0.2`), and SHA-pinning tools (Dependabot, actionlint rules, reviewers) expect it. This contract asks callers to pin `publish.yml` by tag instead. The alternative keeps the convention: callers pin `publish.yml@<full SHA> # v0.1.0`, the certificate SAN then ends in `@<SHA>`, and `pull`'s `signer.refs` becomes an allowlist of docs-kit release commit SHAs (each release adds its SHA to `site/bundles.cue`), instead of the `refs/tags/v*` glob. The cost is one site commit per docs-kit release; the gain is the org-wide pinning rule and no dependence on tag immutability. Nothing else in this design changes with the choice: C9's other checks, the tool-version literal and the modes are the same.

**The tool version follows the ref.** `publish.yml` installs `opm-docs` from the docs-kit release named by a literal in the file, `OPM_DOCS_VERSION: "0.1.0" # x-release-please-version`, which release-please rewrites in every release PR. So `publish.yml@v0.3.0` always runs `opm-docs` 0.3.0, and a caller upgrades both with one ref bump. It downloads `opm-docs_<version>_linux_amd64.tar.gz` and `checksums.txt` from that release and checks the SHA-256 before installing, exactly as C12 requires of every consumer.

```yaml
on:
  workflow_call:
    inputs:
      project:
        description: The project, a key under `bundles` in the caller's docs-kit.cue
        type: string
        required: true
      mode:
        description: "check | edge | release"
        type: string
        required: true
      tag:
        description: release mode, the release's git tag (opm-v4.4.5)
        type: string
        default: ""
      cue-registry:
        description: CUE_REGISTRY for the extractors
        type: string
        default: "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
    outputs:
      digest:
        description: The pushed manifest digest (empty in check mode)
      tag:
        description: The full tag written (empty in check and edge modes)
```

No secrets are declared: the workflow uses `github.token`. The caller's job MUST grant:

| Mode | `contents` | `packages` | `id-token` |
|---|---|---|---|
| `check` | `read` | `read` | none |
| `edge`, `release` | `read` | `write` | `write` |

**Registry read login.** Before `build` or `check` in every mode, the job logs in to `ghcr.io` with `github.token` (`docker login ghcr.io -u ${{ github.actor }} --password-stdin`), so the extractor's CUE dependency resolution (`opmodel.dev/core@v2` and the catalog's other dependencies on GHCR) is authenticated and not rate-limited. `check` needs only `packages: read` for this; the publishing modes' `packages: write` includes it. The same login is the credential `push` and `promote` use later.

What each mode does (all but `check` refuse unless `github.ref` is `refs/heads/main`):

| Mode | Caller runs it on | Steps |
|---|---|---|
| `check` | `pull_request` | checkout the PR head; `opm-docs check --project P` |
| `edge` | `push` to `main` | checkout `github.sha` with full history; `build --edge`; `push`; `cosign sign`; `promote` |
| `release` | the job that runs release-please, gated on that package's release (not on `release: published`); or `workflow_dispatch` for a release that has no bundle yet | checkout `main` and, at `src/`, the tag with full history; `build --release <tag> --source src`; `push`; `cosign sign`; `promote` |

`add-docs-revisions` adds a `revision` mode (`workflow_dispatch`, inputs `tag` and `fix`, the same permissions as `release`) without changing these three. A caller on `v0.1.0` gets an error naming the mode if it asks for `revision`.

Signing: `sigstore/cosign-installer` pinned by SHA with a pinned cosign v3 release, then `cosign sign --yes --new-bundle-format=true ghcr.io/open-platform-model/docs/<project>@<digest>`. Never a tag. Every action is pinned by commit SHA with its version in a comment, like the sibling repositories' workflows.

**Concurrency** (supervisor decision, 2026-10-02). GitHub keeps at most one running and one pending run per concurrency group, and a new pending run cancels the older pending one whatever `cancel-in-progress` says. One shared group per project would therefore let a burst of `main` pushes cancel a queued release publish. The job uses separate groups:

| Mode | Group | `cancel-in-progress` | Why |
|---|---|---|---|
| `check` | none | n/a | writes nothing |
| `edge` | `docs-edge-<project>` | `true` | only the newest `main` matters; a cancelled run leaves at most an unpromoted digest, which the next run supersedes |
| `release` | `docs-release-<project>-<version>` | `false` | a re-run of one release waits for the running one; no other run can cancel it |

`add-docs-revisions` puts the `revision` mode in the same `docs-release-<project>-<version>` group, so revisions of one release are serialized (their revision numbers depend on it) and a revision never cancels its release. Releases of different versions may run at once. `promote` stays correct under that: before moving a tag it resolves the tag's current build and skips the move when that build is already newer than D (C4 rule 5), so a slower, older publish never pulls `4` or `4.4` back.

**Config and sources for a release cut before the repository adopted docs-kit** (decided in planning, sources clarified by supervisor decision 2026-10-02): `build --source src` reads `src/docs-kit.cue` when the release tree has one, and the checked-out `main`'s `docs-kit.cue` otherwise. Only the config comes from `main`: every source in it (the `cue-catalog` `module` and the `markdown` `dir`) resolves against the release tree at `src/`, so the bundle documents the release, never `main`. When the config came from `main` and a `markdown` dir does not exist in the release tree, that source yields no pages and is not an error; the bundle then has only the generated landing (C8). In every other build a missing `markdown` dir is an error, so a typo in `docs-kit.cue` fails the PR check. This is how catalog_opm publishes `4.4.5.0` for a release cut before its first `docs-kit.cue`, which DESIGN decision 8 needs.

### C6. `docs-kit.cue`

One file at the repository root, validated against `schema/config.cue`. A `package` clause is optional and ignored: the file is loaded as a single CUE file and its top-level fields are validated, whatever package it names or whether it names one.

```cue
package schema

#Config: close({
	bundles: [#Project]: #Bundle
})

#Bundle: close({
	placement: #Placement
	version: close({
		from:   "tag"          // phase 1: the release version comes from the git tag
		prefix: string & !=""  // "opm-v": tag "opm-v4.4.5" is version "4.4.5"
	})
	sources: [#Source, ...#Source]
})

#Source: #CueCatalog | #Markdown

#CueCatalog: close({
	kind:   "cue-catalog"
	module: =~"^\\./[^/]"     // the CUE module root, repo-relative: "./opm"
})

#Markdown: close({
	kind: "markdown"
	dir:  =~"^[^/.][^.]*$"    // repo-relative directory, copied to content/ as it is
})
```

`bundles` is keyed by project because one repository can publish several, as core and cli will in phase 2 and catalog_opm would with a second catalog (decided in planning; DESIGN.md shows a single top-level `project`). Phase 1 has one entry. A `layout` choice for catalogs is left out until a second layout has a consumer. The `markdown` kind in phase 1 copies one directory of authored pages, lints them and records their git dates; phase 3 extends it, it does not replace it. A path in content/ written by two sources fails the build, except a root `_index.md` from a `markdown` source: it does not replace the generated landing, the renderer appends its generated block to it (C8).

catalog_opm's file, as the sibling change writes it:

```cue
bundles: {
	"catalog-opm": {
		placement: {kind: "tab", root: "/catalogs/opm/"}
		version: {from: "tag", prefix: "opm-v"}
		sources: [
			{kind: "cue-catalog", module: "./opm"},
			{kind: "markdown", dir: "docs/catalogs/opm"},
		]
	}
}
```

### C7. Pull config, unpack layout and lock

The site's `site/bundles.cue`, validated against `schema/pull.cue`:

```cue
package schema

#Pull: close({
	registry: *"ghcr.io/open-platform-model/docs" | string
	signer: close({
		issuer:   *"https://token.actions.githubusercontent.com" | string
		workflow: *"https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml" | string
		refs: *["refs/tags/v*"] | [string, ...string] // glob over the workflow ref in the certificate
	})
	tabs: [#Project]: close({
		repo: =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$" // the only repository allowed to sign this project
		root: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"
		from: =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$" // the oldest minor shown
		edge: *true | bool
	})
	// Phase 2 adds `versions:` for bundles placed in a site version's /docs/.
})
```

The file opmodel.dev writes:

```cue
tabs: {
	"catalog-opm": {repo: "open-platform-model/catalog_opm", root: "/catalogs/opm/", from: "4.4"}
}
```

Command:

```text
opm-docs pull --config site/bundles.cue --out site/.bundles --lock site/.bundles/lock.json
              [--frozen <lock>] [--offline] [--local <project>@<segment>=<dir>]...
```

- Default: list each tab's tags (C4), resolve, verify (C9), fetch the layer **by digest** after verification, unpack, lint, write the lock.
- `--frozen <lock>`: pull exactly the digests in that lock, no tag resolution; verification and lint still run. The recovery path, as `frozen.conf` is for site versions.
- `--offline`: no network; requires `--frozen` and every blob in the cache. Fails naming the first missing digest.
- `--local <project>@<segment>=<dir>`, repeatable (supervisor decision, 2026-10-02): take that segment of that project from a local bundle tree (an `opm-docs build` output or a test fixture) instead of the registry, e.g. `--local catalog-opm@4.4=fixtures/catalog-opm/4.4 --local catalog-opm@4.5=... --local catalog-opm@edge=...`. The segment must equal the one the tree's `manifest.json` implies (`MAJOR.MINOR` of its version, or `edge`), and the project must be a tab in the config. A local entry is unsigned: `pull` skips signature verification and never fetches the Sigstore trusted root for it, but still validates the manifest, applies the unpack guards and lints it in bundle mode. When `--local` names a project, registry resolution is skipped for that whole project, so a pull whose every tab is local needs no network at all. Each local entry is marked `"local": true` in the lock. For an author's preview and the site's tests; a publishing CI build never passes it.
- Cache: blobs under `$XDG_CACHE_HOME/opm-docs/blobs/sha256/<hex>` (else `~/.cache/...`), reused by digest. The Sigstore trusted root is cached beside it.

Unpack layout, owned entirely by `pull` (it removes any project or segment directory it did not write this run):

```text
site/.bundles/
  lock.json
  catalog-opm/
    4.4/      manifest.json  content/  data/
    4.5/      ...
    edge/     ...
```

`<out>/<project>/history.json` is reserved for phase 1b's version history (see "Site decisions" below); phase 1 never writes it.

The segment directory is the URL segment: `<MAJOR>.<MINOR>` of the build's version, or `edge`. Unpacking refuses an absolute path, `..`, a symlink, a hard link, a device or FIFO, a duplicate path, a top-level entry other than `manifest.json`, `content/` and `data/`, more than 10,000 entries, or more than 64 MiB uncompressed; and checks every file is listed in `manifest.json` and every listed file exists.

Lock, `schema/lock.cue` (`docs.opmodel.dev/lock/v1`), written with sorted keys and no timestamps, so the same resolution writes the same bytes:

```json
{
  "schema": "docs.opmodel.dev/lock/v1",
  "tool": "0.1.0",
  "config": "sha256:<hex of the bundles.cue bytes>",
  "bundles": [
    {
      "project": "catalog-opm",
      "root": "/catalogs/opm/",
      "segment": "4.4",
      "tag": "4.4",
      "repository": "ghcr.io/open-platform-model/docs/catalog-opm",
      "digest": "sha256:<hex>",
      "version": "4.4.5",
      "revision": 1,
      "commit": "<40 hex>",
      "dialect": 1,
      "builtBy": "0.1.0",
      "signer": {
        "workflow": "https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml@refs/tags/v0.1.0",
        "repository": "https://github.com/open-platform-model/catalog_opm",
        "ref": "refs/heads/main"
      },
      "dir": "catalog-opm/4.4"
    }
  ]
}
```

Entries sort by project, then by version newest first, `edge` last. Every entry carries `root`, the tab's placement root (supervisor decision, 2026-10-02), so a consumer maps a `dir` to its URLs without reading `bundles.cue`. A `local` entry has `"local": true`, `root`, `segment`, `version`, `revision`, `commit` and `dir` from its manifest, and no `digest`, `repository`, `tag` or `signer`.

### C8. URLs and links

A tab bundle's `content/<path>` publishes at `<root><segment>/<page URL>`, where `x/_index.md` is `x/` and `x/y.md` is `x/y/`. For `catalog-opm` 4.4: `content/traits/backup.md` is `/catalogs/opm/4.4/traits/backup/`; edge is `/catalogs/opm/edge/traits/backup/`.

Page paths the renderer writes:

| Path | Page |
|---|---|
| `_index.md` | landing: the authored root `_index.md` when a `markdown` source has one (catalog_opm supplies the contract page), followed by the generated "Catalog members" block; otherwise a generated page holding only that block |
| `blueprints/_index.md`, `resources/_index.md`, `traits/_index.md` | kind index, weights 1, 2, 3 |
| `<kind>/<name>.md` | the member page of the newest `apiVersion` of that name and kind |
| `<kind>/<name>-<apiVersion>.md` | every older `apiVersion` of the same name and kind in the same build |

The newest `apiVersion` is the most stable level, then the highest number (`v1` > `v1beta2` > `v1beta1` > `v1alpha1`). So the bare path names the newest contract of a name in every minor, and the switcher keeps a reader on it (decided in planning; refgen suffixes both pages instead).

Links the renderer writes, and the forms the dialect lint (C11) allows:

| From | To | Form |
|---|---|---|
| a tab page | a page of the same bundle | `<root><segment>/<path>/`, e.g. `/catalogs/opm/4.4/resources/volumes/`; must resolve inside the bundle |
| a tab page | another catalog (none in phase 1) | its major alias, `/catalogs/<name>/<MAJOR>/...` |
| a tab page | the docs | `/docs/<section>/<page>/`; the site resolves it in its default version |
| a tab page | an enhancement | `/enhancements/<NNNN>/` or `/enhancements/<NNNN>/<document>/` |
| a docs page (any repository's `docs/site/`) | a catalog | `/catalogs/<name>/<MAJOR>/` plus an optional page path; never a minor or `edge` |

An authored page in a bundle (a `markdown` source) links into its own catalog through the major alias, `/catalogs/<name>/<MAJOR>/<path>/`, the form that also reads correctly on GitHub and in docs mode. When `<MAJOR>` is the build's major (or the build is edge), the `markdown` source rewrites that link to the build's own segment, `/catalogs/<name>/<segment>/<path>/`, before bundle-mode lint checks that it names a page of the bundle (decided in planning).

Aliases the site serves: `/catalogs/<name>/` and `/catalogs/<name>/<MAJOR>/` go to the newest minor (of that major), and `/catalogs/<name>/<MAJOR>/<path>/` to the same path in that minor, so docs pages can deep-link through the alias. For `catalog-opm` the contract page is the landing, so `/catalogs/opm/4/` replaces `/docs/reference/catalog-contract/`.

### C9. Signing identity

Every publishing mode signs the pushed digest with cosign keyless from the reusable workflow's GitHub OIDC token. The certificate's subject is the **reusable** workflow; the caller appears only in Fulcio's extensions. `pull` and `promote` accept a bundle only when its Sigstore bundle verifies with sigstore-go against the public-good trusted root (signed certificate timestamp, transparency-log entry and an observer timestamp, one each), with an artifact digest equal to the manifest digest, and:

| Certificate field | Must equal |
|---|---|
| issuer | `https://token.actions.githubusercontent.com` |
| SAN (Build Signer URI) | `https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml@<ref>` with `<ref>` matching a `signer.refs` glob, default `refs/tags/v*` |
| Source Repository URI | `https://github.com/<tabs[project].repo>` |
| Source Repository Ref | `refs/heads/main` |

The Source Repository URI check is the tenant check: without it any repository calling `publish.yml` could sign a bundle for another project. `promote` applies the same policy with the repository taken from `GITHUB_REPOSITORY`. Humans can check the same thing:

```text
cosign verify ghcr.io/open-platform-model/docs/catalog-opm@sha256:<hex> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp '^https://github\.com/open-platform-model/docs-kit/\.github/workflows/publish\.yml@refs/tags/v' \
  --certificate-github-workflow-repository open-platform-model/catalog_opm \
  --certificate-github-workflow-ref refs/heads/main
```

### C10. The doc model, `data/catalog.json`

Written by `cue-catalog` (schema id `docs.opmodel.dev/data/cue-catalog/v1`). Phase 1b's history reads it across minors, so it is a contract, and the renderer reads nothing else.

```json
{
  "schema": "docs.opmodel.dev/data/cue-catalog/v1",
  "modulePath": "opmodel.dev/catalogs/opm@v4",
  "version": "4.4.5",
  "members": [
    {
      "kind": "trait",
      "name": "backup",
      "apiVersion": "v1alpha1",
      "level": "alpha",
      "fqn": "opmodel.dev/catalogs/opm/traits/backup@v1alpha1",
      "modulePath": "opmodel.dev/catalogs/opm/traits/v1alpha1",
      "title": "Backup",
      "definition": "#BackupTrait",
      "wrapper": "#Backup",
      "file": "opm/traits/v1alpha1/backup.cue",
      "page": "traits/backup",
      "description": "Scheduled backup policy for a component's persistent state",
      "notes": ["<cleaned paragraph>"],
      "category": "storage",
      "fulfilment": "provider",
      "optional": false,
      "appliesTo": ["opmodel.dev/catalogs/opm/resources/volumes@v1beta1"],
      "composedResources": [],
      "composedTraits": [],
      "matchLabels": [{"key": "...", "value": "...", "concrete": true, "required": true}],
      "servedBy": [{"transformer": "...", "fqn": "...", "demand": "required"}],
      "mark": "provided-by-platform",
      "enforcement": [{"rule": "<sentence>", "by": "cue"}],
      "spec": {
        "key": "backup",
        "cue": "<the formatted spec block, citations stripped>",
        "fields": [
          {
            "path": "schedule",
            "type": "string",
            "presence": "required",
            "default": null,
            "doc": "<cleaned doc comment>",
            "ref": null
          }
        ],
        "linked": [{"definition": "#X", "page": "resources/volumes"}],
        "external": [{"package": "opmodel.dev/core/v2", "vendored": false}]
      }
    }
  ],
  "transformers": [
    {
      "name": "...", "fqn": "...", "description": "...",
      "requiredLabels": [], "requiredResources": [], "optionalResources": [],
      "requiredTraits": [], "optionalTraits": []
    }
  ]
}
```

`mark` is `"not-implemented"`, `"provided-by-platform"` or `null`. `demand` is `"required"` or `"optional"`. `spec.fields` is the structured spec, read from the evaluated value: every field under the spec key, depth first, regular, optional and required alike (`presence` is `"regular"`, `"optional"` or `"required"` from the selector's constraint type), with `path` dot-separated, `[]` for a list element and `[string]` for a pattern constraint; `type` the formatted constraint; `default` the formatted default or `null`; `ref` the definition name when the field's value is a definition outside this member's package, where the walk stops instead of expanding it. The walk stops at the module boundary and at a visited definition, and caps depth at 12. `spec.cue` keeps the authored text, so a field the walk cannot express still shows on the page.

### C11. The page dialect, version 1

`opm-docs lint` ports `opmodel.dev/site/scripts/lint-sources.sh` rule for rule and adds the `/catalogs/` link forms (C8). Dialect 1 is exactly that set; the opmodel.dev change updates its shell lint to the same set (`orchestration.md`). The rules:

- A tree holds only regular `.md` files: no symlink, no `.mdx`, no other file; names are lower-case kebab-case; no `index.md` (a section page is `_index.md`).
- Front matter opens on line 1 with `---` and closes with `---`; keys are only `title`, `description`, `type`, `weight`; `title` and `description` are required and non-empty; `type` is required on a leaf page and is one of `tutorial`, `how-to`, `explanation`, `reference`; an `_index.md` declares no `type`; `weight` is a positive integer; `sidebar:` is named as Starlight front matter.
- Shortcodes, checked on every line, code fences included: only `{{< opm/<figure> >}}` with one of the seven figure names, no parameters and no closing tag; `{{</* ... */>}}` is a shown shortcode and passes.
- No `:::` line, no `import ... from` line, no component tag line, no image (`![`, `<img`), no raw `href=` or `src=`.
- Every code fence carries a language tag.
- An alert marker is exactly `> [!NOTE]` (or `TIP`, `IMPORTANT`, `WARNING`, `CAUTION`) alone on its line.
- Every link destination (inline and reference definitions) is `http:`, `https:`, `mailto:` or `#...`, or one of: `/docs/(<seg>/)*` with an optional fragment; `/enhancements/`, `/enhancements/<NNNN>/` or `/enhancements/<NNNN>/<document>/` with `<document>` one of `problem`, `design`, `decisions`, `graduation`, `risks`, `operational`, `questions`; `/catalogs/<name>/` or `/catalogs/<name>/<segment>/(<seg>/)*` with `<segment>` a major (`4`), a minor (`4.4`) or `edge`, with an optional fragment.
- Bundle mode (`lint --bundle <dir>`, run by `build` and `pull`) adds: a link into the bundle's own root uses the bundle's own segment and names a page in the bundle; and `manifest.json` lists exactly the pages present.
- Docs mode (the default) adds: a `/catalogs/` link is the bare tab root (`/catalogs/opm/`) or uses a major segment (`/catalogs/opm/4/...`); a minor or `edge` segment is a violation.

**Agreement with the site's shell lint until phase 3** (supervisor decision, 2026-10-02). Until phase 3 retires `opmodel.dev/site/scripts/lint-sources.sh`, the site lints `docs/site/` trees with the shell script and every bundle with `opm-docs lint`, so the two must agree. They are kept in agreement by one conformance fixture set: docs-kit's `internal/dialect/testdata/conformance/` is a copy of opmodel.dev `site/tests/lint/` (with its source commit recorded), each fixture paired with the expected `<file>:<line>` output, and both linters must pass every fixture. A rule change lands in both repositories with the fixture that proves it, in docs-kit first (the Go lint is the reference) and in the same week in opmodel.dev; neither side changes a rule without the other.

Every violation prints as `<file>:<line>: <message>`, the shell lint's format.

### C12. Tool distribution

Callers and the site never `go run` or `go install` `opm-docs` (supervisor decision, 2026-10-02): every consumer runs a released binary, so the bytes that built a bundle are the bytes a release names.

- **Assets.** Every docs-kit release `vX.Y.Z` carries `opm-docs_X.Y.Z_<os>_<arch>.tar.gz` for `linux_amd64`, `linux_arm64`, `darwin_arm64` and `darwin_amd64` (each holding the `opm-docs` binary and `LICENSE`), and `checksums.txt` (SHA-256, `sha256sum` format, one line per archive). Built by goreleaser in a draft-first release workflow, the pattern cli already uses (decided in planning: the owner's draft-first runbook, the immutable-release setting and reviewers' knowledge carry over unchanged; a hand-rolled Task build would re-implement archives and checksums for no gain). URL: `https://github.com/open-platform-model/docs-kit/releases/download/vX.Y.Z/<asset>`.
- **Pinned in a build image** (supervisor decision, 2026-10-02). A consumer may instead pin the `linux_amd64` archive by its SHA-256 in its own build image (opmodel.dev does this in `site/Dockerfile`, as it pins Hugo and Pagefind) and run `opm-docs` inside that image, with network on for the `pull` step only. The SHA-256 it pins is the archive's line in that release's `checksums.txt`. docs-kit therefore keeps shipping the `linux_amd64` archive and `checksums.txt` in every release, under the names above.
- **Pin file.** A consumer that runs the tool on the host, outside `publish.yml` (catalog_opm for its local `docs:bundle` tasks; opmodel.dev only if it does not use the image pattern) pins it in a repo-root file `.opm-docs-version`: one line, the release tag (`v0.1.0`), matching `^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$`. `publish.yml` pins itself through its `OPM_DOCS_VERSION` literal (C5). In a repository that has both, the `.opm-docs-version` tag and the `publish.yml@` ref name the same release and move in one PR.
- **Verification.** Download the archive and `checksums.txt` for the host's os and arch, check with `grep ' <archive>$' checksums.txt | sha256sum -c -` (refusing an archive with no line), then extract only `opm-docs`. A failed check stops the task; nothing falls back to building from source. The install target is a gitignored repo-local directory (`.bin/` or `site/.bin/`), never a global path.

### Site decisions recorded here (supervisor, 2026-10-02; the owner may override)

These are the opmodel.dev change's to build, recorded here so docs-kit's pull output and the sibling plan agree:

- **Sitemap:** only the newest minor of each major is listed. Older minors and `edge` are left out (and carry `noindex`).
- **Edge search:** `edge` gets its own Pagefind index, like every minor. It gets no alias stubs: `/catalogs/<name>/edge/` is its only address, and no alias ever resolves to it.
- **Phase-1b history:** the version-history data is a file written by `opm-docs pull`, at `<out>/<project>/history.json`, computed from the `data/` of every minor it pulled. Phase 1 reserves that path (pull's stale-directory sweep leaves it alone) and does not write it; the site never computes history itself.

## Commands

Syntax `opm-docs <command> [args] [flags]`. Exit codes: `0` success, `1` usage error (unknown flag, missing argument, unreadable config), `2` execution error (lint violations, a refused build, a registry or signature failure). Every error names what failed and the fix.

| Command | Flags (type, default) | Does |
|---|---|---|
| `build` | `--config` (path, `docs-kit.cue`), `--project` (string, repeatable; default every project), `--out` (path, `out`), `--source` (path, `.`), one of `--edge` (default) or `--release <tag>` | Extract, render, lint; write `out/<project>/`. A dirty work tree is allowed for a local preview and recorded as `source.dirty: true`, which `push` refuses. |
| `lint` | `--bundle` (bool, false), `--dialect` (int, 1) | Lint one or more page directories (or bundle directories) against the dialect. |
| `check` | `--config`, `--project` | `build` into a temporary directory; exit 2 on any failure. The PR gate. |
| `push` | `--dir` (path, required), `--registry` (string, `ghcr.io/open-platform-model/docs`) | Validate, pack deterministically, push; write the full tag for a release build; print `{"digest": ..., "tag": ...}` as JSON on stdout. |
| `promote` | `--project` (required), `--digest` (required), `--registry` | Verify the signature of the digest (C9), then move the moving tags of its line to it (C4 rule 5). |
| `pull` | see C7 | Resolve, verify, unpack, lint, lock. |
| `version` | none | Print `opm-docs <version>`. |

`add-docs-revisions` adds `opm-docs revise` (and a `--revision` flag on `build`); its documentation-only check and patch handling are specified there.

## Packages

```text
cmd/opm-docs/            cobra root and one file per command; flag parsing only
schema/                  manifest.cue, config.cue, pull.cue, lock.cue; embedded with go:embed
internal/version/        build identity
internal/config/         load and validate docs-kit.cue and bundles.cue
internal/bundle/         the tree model, manifest, deterministic pack and guarded unpack
internal/tags/           version parsing, ordering, full and moving tags, revision numbers (pure)
internal/doctext/        maintainer-comment dropping, citation stripping, summary split, wrapping
internal/mdtext/         escaping, code spans, table cells, YAML strings
internal/extract/cuecatalog/   the cue-catalog extractor: doc model, structured spec, served-by, marks
internal/extract/markdown/     the markdown source
internal/render/         embedded templates: landing, kind index, member page
internal/dialect/        the page-dialect lint
internal/gitsrc/         git dates and commit times (add-docs-revisions adds worktrees and the documentation-only check)
internal/oci/            oras-go: push, list, resolve, fetch by digest, tag
internal/verify/         sigstore-go: find the Sigstore bundle, apply the C9 policy
internal/pull/           tab resolution, cache, unpack layout, lock
```

Dependencies: `cuelang.org/go` v0.17.1 (the version catalog_opm and cli use), `oras.land/oras-go/v2` at v2.6.2 or later (the hard-link extraction fix), `github.com/sigstore/sigstore-go` and `github.com/spf13/cobra`. `internal/tags` implements SemVer 2.0.0 precedence itself rather than use `golang.org/x/mod/semver`, which requires a leading `v`. Tests run offline: an in-process registry (`go-containerregistry`'s `pkg/registry`, or `zot` if the spike shows referrer fallback needs it) and sigstore-go's virtual Sigstore for signatures.

## Page renderer

Templates are Go `text/template` files embedded in the binary, one per page kind. A member page follows refgen's order: front matter (`title`, `description` = `metadata.description`, `type: reference`), `## At a glance` (an optional mark alert, then a `| Field | Value |` table: FQN, API version with level and a link to the contract landing, Module path, Definition with its file, Component wrapper, Catalog with module path and version, Category, Fulfilment, Optional posture, Applies to, Composed resources and traits, Match label), `## Spec` (`cue` fence, then "Defined elsewhere" links), `## Notes`, `## Served by`, `## Enforcement` (links `/docs/concepts/what-enforces-a-rule/`).

For an edge build, the Catalog row reads "`<module path>` at `main` (commit `<12 hex>`), unreleased" instead of a version.

The mark strings, kept from refgen byte for byte:

```markdown
> [!IMPORTANT]
> **Provided by your platform**
>
> This catalog defines the contract and ships no transformer for it. Your platform needs exactly one catalog that implements it. Without one, %s; with two, the kernel refuses every render on that platform.
```

```markdown
> [!WARNING]
> **Not implemented**
>
> This catalog defines the %s and ships no transformer that handles it. On a platform where no other catalog handles it, %s.
```

with `%s` "a component that attaches it still renders, and the render warns that the trait is not handled and ignores its values" for an advisory trait, else "rendering a component that declares it fails" (resource) or "... that attaches it fails" (trait). The kind index lists the marked members in the two sentences refgen writes today. The landing always ends with a generated block (supervisor decision, 2026-10-02):

```markdown
## Catalog members

`opmodel.dev/catalogs/opm@v4` version `4.4.5`.

- [Blueprints](/catalogs/opm/4.4/blueprints/): 5
- [Resources](/catalogs/opm/4.4/resources/): 12
- [Traits](/catalogs/opm/4.4/traits/): 28
```

For edge the second line reads "`<module path>` at `main` (commit `<12 hex>`), unreleased." When a `markdown` source supplies a root `_index.md`, its front matter and body come first, unchanged except for C8's alias pinning, then one blank line and the block; the authored body must not already hold a `## Catalog members` heading (the build refuses it, naming the file). The page is recorded in `manifest.json` with `generated: false` and `source` the authored file. Without an authored landing, the renderer writes front matter (`title` "<catalog name> catalog", `description` "Every member of `<module path>`, by kind.") and the block alone, `generated: true`.

No page carries a generator marker comment: a bundle's pages are wholly generated and never committed, so there is no authored text to keep apart (decided in planning).

## Risks / Trade-offs

- [GHCR drops `artifactType` or the empty config, or mangles annotations] → Section 1's spike gates everything else; outcome B (C2) keeps every sibling contract.
- [sigstore-go cannot find or verify a cosign v3 bundle stored under the `sha256-` fallback tag] → the spike verifies the round trip with the real tools before `internal/verify` is written. If it fails, `pull` shells out to a pinned `cosign verify` with the C9 flags (a second binary on the site host, recorded as a regression of DESIGN decision 3).
- [The Sigstore trusted root needs network] → `pull` runs on the host step that already has network; the root is cached and `--offline` refuses with a message rather than skipping verification.
- [A deterministic layer is not byte-stable across Go versions (gzip)] → the digest only has to be stable for one tool build; a test pins the digest of a fixture tree for the tool's own Go version.
- [Callers pin `publish.yml` by tag, against the org's SHA-pinning convention] → recorded as a known conflict in C5 with its alternative (a SHA allowlist in `signer.refs`); the choice is left to review.
- [Edge builds pushed by digest accumulate untagged versions on GHCR] → harmless to readers; pruning is a later concern, never automatic deletion of a full tag.

## Research & Decisions

### R1. Where the tool version comes from

**Context**: A reusable workflow has no context variable for its own ref, so `publish.yml` cannot ask which docs-kit release it is.
**Explored**: an extra `version` input; checking out docs-kit at `github.workflow_sha` (that is the caller's SHA); a literal in the file.
**Options considered**:
1. A `version` input the caller passes - two pins that can disagree.
2. A literal maintained by release-please's generic updater - one pin, the ref.
**Decision**: option 2, decided in planning.
**Rationale**: the caller pins one ref and gets the matching tool; a mismatch is impossible.

### R2. The raw Kubernetes catalog

**Context**: refgen writes both catalogs' reference today; phase 1 retires refgen.
**Decision**: no k8s bundle, tab or table layout (supervisor decision, 2026-10-02): the owner is removing the k8s catalog from catalog_opm in a separate session. catalog_opm retires refgen only after that removal has landed on its `main` (`orchestration.md`), so no k8s page is left without a generator.
**Rationale**: planning around a catalog that is being removed would add a layout and a tab with no lasting consumer.

### R3. Signing in the workflow, verifying in the tool

**Context**: DESIGN decision 3 gives `opm-docs` its own OCI client; signing could also be in-process.
**Options considered**:
1. Sign and verify with sigstore-go in the tool - one binary, but our own Fulcio and Rekor client code.
2. Sign with the pinned cosign CLI in the workflow, verify with sigstore-go in the tool - the standard signer, an in-process verifier, `cosign verify` usable by hand.
**Decision**: option 2, decided in planning.
**Rationale**: signing runs only in CI, where cosign is one pinned action; verification runs on every site build and belongs in the tool.

### R4. Moving tags only after the signature

**Context**: a moving tag pointed at an unsigned digest fails every site build until signing finishes.
**Decision**: `push` writes only the full tag (or nothing, for edge); `cosign sign`; then `promote` verifies and moves. Decided in planning: DESIGN.md's `push` moved the tags itself.
**Rationale**: no reader ever resolves a moving tag to an unsigned build.

### R5. Lock file not committed

**Context**: the site resolves tab versions from moving tags on every build, as line versions do.
**Decision**: the lock is a build output under `site/.bundles/`, recorded in the build stamp; a recovery build passes `--frozen` with a saved lock. Decided in planning.
**Rationale**: a committed lock would need a commit per catalog release, which DESIGN.md's "a new minor appears with no site commit" rules out.

## Durable decisions

| Decision | Lands in |
|---|---|
| C1 to C12, the contracts | `docs/contracts.md` (new), linked from `AGENTS.md` and `README.md` |
| How callers pin `publish.yml` (a tag, or a SHA allowlist if review chooses it; C5) | `docs/contracts.md` C5 and `README.md` "Using the workflow" |
| Commands, flags, exit codes | `README.md` "Commands" |
| Consumers install a released, checksum-verified `opm-docs` pinned in `.opm-docs-version`, never `go run` (C12) | `docs/contracts.md` C12 and `README.md` "Installing" |
| How to release docs-kit, and that `publish.yml`'s version literal is release-please's | `AGENTS.md` |
