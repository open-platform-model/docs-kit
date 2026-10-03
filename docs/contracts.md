# docs-kit contracts

Everything here is read by another repository: the bundle format, the tag scheme, the reusable workflow, `docs-kit.cue`, the site's pull config and lock, the URL and link forms, the signing identity, the doc model, the page dialect and how the tool is distributed. A change to any of it follows constitution Principle II (`openspec/config.yaml`): prefer additive changes, bump the schema identifier for a breaking one, and name every consuming repository and what it must do.

The numbers (C1 to C12) are stable; other repositories cite them as `docs-kit C5`. The approved design and its decisions are in [DESIGN.md](../DESIGN.md). These contracts were fixed by the OpenSpec change `build-opm-docs-phase-1` (archived under `openspec/changes/archive/`), which holds the reasoning behind each.

## Doc-comment rules

How every extractor turns source comments into reader-facing text (`internal/doctext`, `internal/mdtext`):

- **Maintainer comments.** Inside a definition, a comment group whose first line (after `//` and trimming) starts with `WHY` or with `//` (a `////` banner) is dropped. A `WHY` line inside a doc comment is dropped on its own (core's rule, adopted now).
- **Citations.** Removed from prose and from comments in spec blocks. The pattern accepts `0010:D28`, `0010 D28`, `OQ` numbers, `:R2` and `/R1/R2` requirement suffixes, `/D9` continuations, lists joined by `,`, `;` or `and`, an optional `(see|per|enhancement)` lead and the parenthesised form; then refgen's six clean-up rewrites (empty parens, comma before paren, space after paren, space before punctuation, double spaces, trim). Also removed: `See SPEC.md § N.` sentences, inline `SPEC.md § N`, `NNNN experiment N` and `enhancements/NNNN/experiments/...` (core's rules, adopted now). A changed comment paragraph in a spec block is re-wrapped to `80 - indent - 3` columns, minimum 40; an unchanged one keeps its breaks; a trailing comment that becomes empty is removed.
- **Summary.** A member's doc comment opens with `metadata.description` plus `.` (whitespace collapsed), or the build refuses the member naming both texts. The summary is the page's front-matter `description` and is not repeated in the body. The remaining paragraphs are the notes.
- **Escaping.** Outside backtick code spans: `\ < > * _ [ ] |` are backslash-escaped and `{{` becomes `{\{`. A table cell escapes every `|`, inside code spans too. A code span lengthens its fence past any backtick it holds. A YAML front-matter string escapes `\` and `"`. A rendered page that still holds `{{<` or `{{%` fails the build.
- **Doc-note links.** `docs/<name>.md` in prose becomes a link to that file on the source repository at the bundle's commit, only when the file exists and the text is outside a code span.
- **Marks.** A resource or trait that no transformer in its own catalog requires or optionally reads is marked **Provided by your platform** when its `fulfilment` default is `provider`, and **Not implemented** otherwise; a blueprint is never marked. Exact strings in "Page renderer".
- **Enforcement tags.** Only where derivable: the spec schema and each required match label are enforced by `cue`; a provider-fulfilled contract's single provider and a load-bearing trait's refused render are enforced by the `kernel`. Nothing else is tagged.

## C1. Where bundles live

```text
ghcr.io/open-platform-model/docs/<project>
```

| Project | Source | Release tag prefix | Placement |
|---|---|---|---|
| `catalog-opm` | catalog_opm `opm/` (`opmodel.dev/catalogs/opm@v4`) | `opm-v` | tab, `/catalogs/opm/` |

A project name matches `^[a-z0-9]+(-[a-z0-9]+)*$`. Phases 2 and 3 add `core`, `cli`, `opm-operator`, `library` and `opm` under the same path.

## C2. Media types and annotations

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
| `org.opencontainers.image.created` | the source commit's committer time, RFC 3339 UTC |
| `dev.opmodel.docs.project` | the project, `catalog-opm` |
| `dev.opmodel.docs.revision` | the docs revision, decimal (`0` for edge) |
| `dev.opmodel.docs.dialect` | the page-dialect version the content passed, decimal |
| `dev.opmodel.docs.tool` | the `opm-docs` version that built it, without `v` |

`org.opencontainers.image.source` names the calling repository because GHCR links a new package to the repository named there, and the workflow's `GITHUB_TOKEN` can create a package only when that link is in its first push.

The layer is a gzip-compressed tar of the bundle tree (C3), built deterministically: entries sorted by path, directories before their contents, mode `0644` for files and `0755` for directories, uid and gid `0`, empty user and group names, modification time equal to `created`, no extended headers, gzip header with no name and no time. The same source, config and tool version give the same digest.

Signatures are cosign v3 Sigstore bundles (`--new-bundle-format=true`): a referrer manifest with `artifactType` `application/vnd.dev.sigstore.bundle.v0.3+json` whose subject is the bundle's digest. GHCR has no referrers API, so the referrer is stored under the fallback tag `sha256-<hex>`; every consumer ignores tags of that form when it lists versions.

**Verified on GHCR (2026-10-02; runs [37053604905](https://github.com/open-platform-model/docs-kit/actions/runs/37053604905), [37053754999](https://github.com/open-platform-model/docs-kit/actions/runs/37053754999), [37054087676](https://github.com/open-platform-model/docs-kit/actions/runs/37054087676)).** The form above holds.

- GHCR kept the manifest byte for byte: `artifactType`, the empty config (oras writes it with `"data":"e30="`), the one layer and all eight annotations, fetched by every tag and by digest.
- An untagged manifest pushed by digest was accepted and fetchable by digest at once, in an existing package and as the first write to a new one (`docs/spike-edge-first`, the shape of a project's first edge build).
- One transient: in the first run, the first manifest of the brand-new `docs/spike`, pushed by digest, returned 404 when fetched by digest a moment later, so tagging it failed; the same push and fetch succeeded in both later runs and in the second new package. Rule adopted: `push` writes a full tag with the manifest PUT itself (no fetch), and every fetch of a manifest this run just pushed (by `push` or `promote`) retries a 404 with backoff for up to 30 seconds before failing.
- The package was created by the workflow's `GITHUB_TOKEN` with `packages: write` and no other grant, linked to `open-platform-model/docs-kit` through `org.opencontainers.image.source`, and created **public**: it took the visibility of the public repository it is linked to. DESIGN.md's "a container package pushed from Actions starts private" did not hold here; catalog_opm's go-live still checks the visibility, but making it public may be a no-op.
- After signing, the tag list held `0.0.1.0`, `0.0.1`, `0.0`, `0` and one `sha256-<hex>` tag per signed digest (both the tagged and the untagged manifest).
- oras-go's `registry.Referrers` found each signature through the fallback tag schema: a referrer manifest with `artifactType` `application/vnd.dev.sigstore.bundle.v0.3+json`, one layer holding the Sigstore bundle, annotations `dev.sigstore.bundle.content: dsse-envelope` and `dev.sigstore.bundle.predicateType: https://sigstore.dev/cosign/sign/v1`. The bundle is a DSSE envelope over an in-toto statement whose subject is the manifest digest; sigstore-go's artifact-digest policy matches that subject.
- sigstore-go v1.3.0 verified both digests under the C9 policy (one SCT, one transparency-log entry, observer timestamps from Rekor and the timestamp authority) with SAN `.../spike-sign.yml@refs/heads/spike/ghcr`, Source Repository URI `https://github.com/open-platform-model/docs-kit` and ref `refs/heads/spike/ghcr`, all from the spike's flags. The certificate's Build Config URI named the caller (`spike.yml`), confirming the split. It refused the wrong Source Repository URI and the caller as SAN. `cosign verify` with the equivalent flags passed. No R3 fallback is needed.
- Anonymous `spike pull --tag 0.0` resolved, verified and fetched the layer by digest with no credential, in CI and from a workstation.
- cosign v3.1.3 marks `--new-bundle-format` deprecated (the new format is its default and will be the only one). `publish.yml` keeps the flag while it is accepted, since cosign is pinned.

## C3. Bundle tree and `manifest.json`

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

#Manifest: {
	schema:   "docs.opmodel.dev/bundle/v1"
	project:  #Project
	version:  #Version
	revision: int & >=0
	if version == "edge" {revision: 0}
	source: {
		repo:   =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$" // "open-platform-model/catalog_opm"
		commit: #SHA                                  // the commit built (a revision: the release commit)
		ref:    string & !=""                         // the release tag "opm-v4.4.5", or the branch built ("main" for edge)
		dirty?: true                                  // built from a work tree with uncommitted changes; push refuses it
		// A docs revision only: the fix commits applied to the release tree, oldest first,
		// every earlier revision's fixes included. revise writes it; a pull of 0.1.0, which
		// never wrote it, already accepts it.
		patches?: [#SHA, ...#SHA]
	}
	// The source commit's committer time, RFC 3339 UTC: the tar entries' time and the
	// org.opencontainers.image.created annotation. build always writes it and push
	// requires it; pull accepts a bundle without it.
	created?:  time.Time
	tool:      #SemVer // the opm-docs version, without "v"
	dialect:   int & >=1
	placement: #Placement
	pages: list.MinItems(1) & [...#Page]
	data: [...#DataFile]
}

#Project: =~"^[a-z0-9]+(-[a-z0-9]+)*$"
#SHA:     =~"^[0-9a-f]{40}$"
#SemVer:  =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
#Version: #SemVer | "edge"

#Placement: {
	// "tab": its own section with its own versions, /catalogs/<name>/<MAJOR.MINOR>/.
	// "docs": merged into a site version's /docs/ tree (phase 2; refused by phase-1 pull).
	kind: "tab" | "docs"
	if kind == "tab" {root: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"}
	if kind == "docs" {root: "/docs/"}
}

#Page: {
	path:      =~"^([a-z0-9]+(-[a-z0-9]+)*/)*(_index|[a-z0-9]+(-[a-z0-9]+)*)\\.md$" // under content/
	source?:   string & !=""                                                        // repo-relative file the page came from ("Edit this page", "View source")
	lastmod?:  time.Time                                                            // that file's last commit date at the commit built, RFC 3339
	generated: bool                                                                 // generated reference, or an authored page
}

#DataFile: {
	path:   =~"^[a-z0-9-]+\\.json$" // under data/
	schema: string & !=""           // the file's own schema id, e.g. "docs.opmodel.dev/data/cue-catalog/v1"
}
```

Tightened from DESIGN.md: closed structs, the SHA and SemVer patterns, `revision: 0` for edge, `patches`, typed `data` entries, and a placement root bound to its kind. The schemas rely on definition closedness rather than `close()` (implementation finding): a struct passed to `close()` is evaluated on its own, so its `if kind == "tab"` guard never sees the data's `kind`, and `close()` does not close a nested pattern map such as `bundles: [#Project]: #Bundle`, which then accepted any project name. A definition closes every struct inside it, so the shipped files drop `close()` with the same meaning. `created` (optional; implementation finding) records the source commit's time: the bundle-format spec requires every annotation to equal a `manifest.json` field and `push` takes only `--dir`, so the time `push` stamps on the tar entries and on `org.opencontainers.image.created` has to travel in the manifest. `build` always writes it and `push` refuses a manifest without it; `pull` accepts one without it, so a fixture tree written to the original shape still validates. `pages` lists every file under `content/`, and only those; `data` lists every file under `data/`, and only those. `lastmod` is set when the checkout has the file's history (the workflow checks out with full history) and omitted otherwise.

## C4. Tags

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
3. **Revision numbers.** A release's first build is revision `0`. A docs revision ("Docs revisions" below) takes `1 +` the highest revision published for that version, and is refused when revision `0` does not exist.
4. **Edge** builds carry `version: "edge"`, `revision: 0` and no full tag; they are pushed by digest and reached only through `edge`.
5. **Moving a tag.** `promote --digest D` moves each moving tag of D's line to D only when D is the newest build of that tag's line, and only after D's signature verifies. Just before each move it resolves the tag's current build again and skips the move when that build is newer than D, because releases of different versions may publish at once (C5, Concurrency). It never points a tag at any other digest. Promote enumerates builds from every tag of the repository that equals `<version>.<revision>` of its own manifest's annotations; other tags (moving tags, `edge`, `sha256-*`) are not builds.
   A moving tag never takes over a full tag: when the tag already names a build whose full tag is that very name (the release tag `1.0.0-beta.5` of version `1.0.0-beta.5` and the full tag of `1.0.0-beta` revision `5` are the same string), `promote` refuses and moves nothing. `edge` never moves back: when it already names an edge build whose `org.opencontainers.image.created` is later than D's, `promote` leaves it (a late run of an older commit). Both rules added in review, 2026-10-02.
6. **No release-branch bundles.** Every publishing mode runs on `refs/heads/main` (DESIGN decision 9); `pull` checks the same ref in the signature (C9).

A consumer selects a tab's versions by tag name: every tag matching `^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$` (a minor tag) at or above the tab's `from`, plus `edge` when asked. It then trusts only the resolved manifest's annotations and `manifest.json`, and refuses a minor tag whose build's version is not in that minor.

## C5. The reusable workflow

File `.github/workflows/publish.yml` in this repository. A caller pins a docs-kit **tag**:

```yaml
uses: open-platform-model/docs-kit/.github/workflows/publish.yml@v0.1.0
```

**Callers reference `publish.yml` by release tag**: `@vX.Y.Z`, never a branch and never a commit SHA. The signing certificate names the workflow at the ref the caller wrote, and `pull` trusts only the SAN `https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml@refs/tags/v[0-9]*` (C9).

This is a deliberate exception to the org's convention of pinning every action and reusable workflow by full commit SHA. It is safe only because docs-kit's tags are immutable: the `tags-immutable` and `tags-create-app-only` org rulesets cover docs-kit and its releases are immutable, so `v0.1.0` can never be moved to different code. Those settings are a hard gate before docs-kit's first release; without them a moved tag would let other code sign as a trusted publisher. Rejected alternative: SHA pinning with an allowlist of docs-kit release SHAs in `signer.refs`, which would need a site commit for every docs-kit release.

**The tool version is the caller's pin** (amended in review, owner decision 2026-10-02). `publish.yml` carries no version of its own. It reads the caller's repo-root `.opm-docs-version` from the checked-out caller tree (the commit the workflow runs on; in release mode, `main`'s, not the release tree's): one line, a docs-kit release tag such as `v0.1.0`, matching `^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$`. A missing file, a second line or another form fails the job before anything is built, naming the file and the form. It downloads that release's `opm-docs_<version>_linux_amd64.tar.gz` and `checksums.txt` and checks the SHA-256 before installing, exactly as C12 requires of every consumer. A caller moves `.opm-docs-version` and its `publish.yml@vX.Y.Z` ref together, in one PR; the signing identity names the ref, the bundle's `dev.opmodel.docs.tool` annotation names the tool.

```yaml
on:
  workflow_call:
    inputs:
      project:
        description: The project, a key under `bundles` in the caller's docs-kit.cue
        type: string
        required: true
      mode:
        description: "check | edge | release | revision"
        type: string
        required: true
      tag:
        description: release and revision modes, the release's git tag (opm-v4.4.5)
        type: string
        default: ""
      fix:
        description: revision mode, the 40-hex commit on main whose documentation change to apply
        type: string
        default: ""
      cue-registry:
        description: CUE_REGISTRY for the extractors
        type: string
        default: "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
    outputs:
      digest:
        description: The pushed manifest digest (empty in check mode)
        value: ${{ jobs.docs.outputs.digest }}
      tag:
        description: The full tag written (empty in check and edge modes)
        value: ${{ jobs.docs.outputs.tag }}
```

No secrets are declared: the workflow uses `github.token`. `publish.yml` declares no `permissions` of its own, so every job in it runs with the grant of the caller's job (a called workflow cannot raise it, and a job that asked for more than a check-only caller grants would fail that caller at start-up). The caller's job MUST grant:

| Mode | `contents` | `packages` | `id-token` |
|---|---|---|---|
| `check` | `read` | `read` | none |
| `edge`, `release`, `revision` | `read` | `write` | `write` |

**Registry read login.** Before `build` or `check` in every mode, the job logs in to `ghcr.io` with `github.token` (`docker login ghcr.io -u ${{ github.actor }} --password-stdin`), so the extractor's CUE dependency resolution (`opmodel.dev/core@v2` and the catalog's other dependencies on GHCR) is authenticated and not rate-limited. `check` needs only `packages: read` for this; the publishing modes' `packages: write` includes it. The same login is the credential `push` and `promote` use later.

What each mode does (all but `check` refuse unless `github.ref` is `refs/heads/main`):

| Mode | Caller runs it on | Steps |
|---|---|---|
| `check` | `pull_request` | checkout the PR head; `opm-docs check --project P` |
| `edge` | `push` to `main` | checkout `github.sha` with full history; `build --edge`; `push`; `cosign sign`; `promote` |
| `release` | the job that runs release-please, gated on that package's release (not on `release: published`); or `workflow_dispatch` for a release that has no bundle yet | checkout the commit of `main` the workflow runs on (`github.sha`) and, at `src/`, the tag with full history; `build --release <tag> --source src`; `push`; `cosign sign`; `promote` |

| `revision` | `workflow_dispatch`, inputs `tag` and `fix` | refuse a `fix` that is not 40 hex; checkout `github.sha` of `main` with full history (it brings `origin/main` and the tags); `revise --project P --tag <tag> --fix <fix>` ("Docs revisions" below); `push`; `cosign sign`; `promote` |

The `revision` mode came with docs-kit `v0.2.0` (change `add-docs-revisions`), without changing the other three. A caller on `v0.1.0` gets an error naming the mode if it asks for `revision`.

Signing: `sigstore/cosign-installer` pinned by SHA with a pinned cosign v3 release, then `cosign sign --yes --new-bundle-format=true ghcr.io/open-platform-model/docs/<project>@<digest>`. Never a tag. Every action is pinned by commit SHA with its version in a comment, like the sibling repositories' workflows.

**Concurrency**. GitHub keeps at most one running and one pending run per concurrency group, and a new pending run cancels the older pending one whatever `cancel-in-progress` says. One shared group per project would therefore let a burst of `main` pushes cancel a queued release publish. The job uses separate groups:

| Mode | Group | `cancel-in-progress` | Why |
|---|---|---|---|
| `check` | `docs-check-${{ inputs.project }}-${{ github.run_id }}` (alone) | `false` | writes nothing |
| `edge` | `docs-edge-${{ inputs.project }}` | `true` | only the newest `main` matters; a cancelled run leaves at most an unpromoted digest, which the next run supersedes |
| `release`, `revision` | `docs-release-${{ inputs.project }}-${{ inputs.tag }}` | `false` | a re-run of one release waits for the running one; no other run can cancel it; revisions of one release run one at a time, so two never take the same revision number |

Groups are keyed on the workflow's inputs because the concurrency expression is evaluated before any step runs: the release tag is known then, the version derived from it is not. The `revision` mode shares the release's `docs-release-${{ inputs.project }}-${{ inputs.tag }}` group, so revisions of one release are serialized (their revision numbers depend on it) and a revision never cancels its release. Releases of different versions may run at once. `promote` stays correct under that: before moving a tag it resolves the tag's current build and skips the move when that build is already newer than D (C4 rule 5), so a slower, older publish never pulls `4` or `4.4` back.

**Config and sources for a release cut before the repository adopted docs-kit**: `build --source src` reads `src/docs-kit.cue` when the release tree has one, and the checked-out `main`'s `docs-kit.cue` otherwise. Only the config comes from `main`: every source in it (the `cue-catalog` `module` and the `markdown` `dir`) resolves against the release tree at `src/`, so the bundle documents the release, never `main`. When the config came from `main` and a `markdown` dir does not exist in the release tree, that source yields no pages and is not an error; the bundle then has only the generated landing (C8). In every other build a missing `markdown` dir is an error, so a typo in `docs-kit.cue` fails the PR check. This is how catalog_opm publishes `4.4.5.0` for a release cut before its first `docs-kit.cue`, which DESIGN decision 8 needs.

## C6. `docs-kit.cue`

One file at the repository root, validated against `schema/config.cue`. A `package` clause is optional and ignored: the file is loaded as a single CUE file and its top-level fields are validated, whatever package it names or whether it names one.

```cue
package schema

#Config: {
	bundles: [#Project]: #Bundle
}

#Bundle: {
	placement: #Placement
	version: {
		from:   "tag"         // phase 1: the release version comes from the git tag
		prefix: string & !="" // "opm-v": tag "opm-v4.4.5" is version "4.4.5"
	}
	sources: [#Source, ...#Source]
}

#Source: #CueCatalog | #Markdown

#CueCatalog: {
	kind:   "cue-catalog"
	module: =~"^\\./[^/]" // the CUE module root, repo-relative: "./opm"
}

#Markdown: {
	kind: "markdown"
	dir:  =~"^[^/.][^.]*$" // repo-relative directory, copied to content/ as it is
}
```

`bundles` is keyed by project because one repository can publish several, as core and cli will in phase 2 and catalog_opm would with a second catalog. Phase 1 has one entry. A `layout` choice for catalogs is left out until a second layout has a consumer. The `markdown` kind in phase 1 copies one directory of authored pages, lints them and records their git dates; phase 3 extends it, it does not replace it. A path in content/ written by two sources fails the build, except a root `_index.md` from a `markdown` source: it does not replace the generated landing, the renderer appends its generated block to it (C8).

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

## C7. Pull config, unpack layout and lock

The site's `site/bundles.cue`, validated against `schema/pull.cue`:

```cue
package schema

#Pull: {
	registry: *"ghcr.io/open-platform-model/docs" | string
	signer: {
		issuer:                                       *"https://token.actions.githubusercontent.com" | string
		workflow:                                     *"https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml" | string
		refs: *["refs/tags/v[0-9]*"] | [string, ...string] // glob over the workflow ref in the certificate
	}
	tabs: [#Project]: {
		repo: =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$" // the only repository allowed to sign this project
		root: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"
		from: =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$" // the oldest minor shown
		edge: *true | bool
	}
	// Phase 2 adds `versions:` for bundles placed in a site version's /docs/.
}
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
- `--local <project>@<segment>=<dir>`, repeatable: take that segment of that project from a local bundle tree (an `opm-docs build` output or a test fixture) instead of the registry, e.g. `--local catalog-opm@4.4=fixtures/catalog-opm/4.4 --local catalog-opm@4.5=... --local catalog-opm@edge=...`. The segment must equal the one the tree's `manifest.json` implies (`MAJOR.MINOR` of its version, or `edge`), and the project must be a tab in the config. A local entry is unsigned: `pull` skips signature verification and never fetches the Sigstore trusted root for it, but still validates the manifest, applies the unpack guards and lints it in bundle mode. When `--local` names a project, registry resolution is skipped for that whole project, so a pull whose every tab is local needs no network at all. Each local entry is marked `"local": true` in the lock. For an author's preview and the site's tests; a publishing CI build never passes it.
- Cache: blobs under `$XDG_CACHE_HOME/opm-docs/blobs/sha256/<hex>` (else `~/.cache/...`), reused by digest. The Sigstore trusted root is cached beside it.

Edge cases:

- **No `edge` tag** while the tab asks for edge (a project that has not pushed `main` since adopting docs-kit): the segment is skipped with a warning naming the project; the lock has no edge entry. Not an error.
- **No minor at or above `from`:** if the tab still gets an `edge` build, the pull succeeds with a warning; if it gets nothing at all, the pull fails naming the project, since the site cannot render an empty tab.
- **Placement cross-check:** every pulled or local manifest's `placement` must be `kind: "tab"` with `root` equal to the tab's `root` in the pull config, else that bundle fails the pull naming both roots.
- **`--frozen <lock>`:** the lock's `config` digest must equal the current `bundles.cue`'s, because the trust policy (each tab's owning `repo`, the signer) comes from the config; a mismatch fails naming both digests. Tags are not resolved; each entry's `repository` and `digest` are fetched as written, and a local entry in a frozen lock is refused (re-run with `--local`). So is an entry whose `repository` is not `<registry>/<project>` of the config, or whose segment the config does not show (below `from`, or `edge` with `edge: false`): a lock can only narrow what the config trusts (added in review, 2026-10-02).
- **`--offline` and the trusted root:** offline, `pull` uses the cached trusted root as it is and never refreshes it. When the cached TUF metadata has expired it warns and still verifies, because each signature is checked against the key validity window at its own timestamp, which an expired cache does not change. With no cached root at all, `--offline` fails.
- **Compressed size:** a layer descriptor larger than 32 MiB is refused before any byte is fetched; the 64 MiB uncompressed cap applies during unpacking. `push` refuses a bundle over the same limits (32 MiB packed, 10,000 entries, 64 MiB of files) before it writes anything, so no publish produces a bundle every pull refuses (added in review, 2026-10-02).

Unpack layout, owned entirely by `pull` (it removes any project or segment directory it did not write this run). Each bundle unpacks and lints in `<out>/<project>/.incoming-<segment>/` and replaces its segment only after both pass, so a refused bundle leaves the previous segment in place (added in review, 2026-10-02):

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

The lock, `site/.bundles/lock.json`, validated against `schema/lock.cue`:

```cue
package schema

#Lock: {
	schema: "docs.opmodel.dev/lock/v1"
	tool:   #SemVer                   // the opm-docs that wrote the lock
	config: =~"^sha256:[0-9a-f]{64}$" // SHA-256 of the bundles.cue bytes
	bundles: [...#Locked]
}

#Locked: #Pulled | #Local

#Pulled: {
	project:    #Project
	root:       =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"
	segment:    #Segment
	tag:        #Segment      // the tag resolved: the segment itself
	repository: string & !="" // "ghcr.io/open-platform-model/docs/catalog-opm"
	digest:     =~"^sha256:[0-9a-f]{64}$"
	version:    #Version
	revision:   int & >=0
	commit:     #SHA
	dialect:    int & >=1
	builtBy:    #SemVer // the bundle's dev.opmodel.docs.tool
	signer: {
		workflow:   string & !="" // the certificate SAN
		repository: string & !="" // Source Repository URI
		ref:        "refs/heads/main"
	}
	dir: string & !="" // "<project>/<segment>", relative to the lock's directory
}

#Local: {
	project:  #Project
	root:     =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"
	segment:  #Segment
	local:    true
	version:  #Version
	revision: int & >=0
	commit:   #SHA
	dialect:  int & >=1
	builtBy:  #SemVer
	dir:      string & !=""
}

#Segment: =~"^((0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)|edge)$"
```

Serialization (so two pulls of one resolution compare byte for byte): JSON, two-space indent, a trailing newline, no timestamps; entries sorted by `project`, then `segment` (minors by numeric MAJOR then MINOR, ascending, `edge` last); keys in exactly the order the schema lists them (top level `schema`, `tool`, `config`, `bundles`; a pulled entry `project`, `root`, `segment`, `tag`, `repository`, `digest`, `version`, `revision`, `commit`, `dialect`, `builtBy`, `signer` with `workflow`, `repository`, `ref`, then `dir`; a local entry `project`, `root`, `segment`, `local`, `version`, `revision`, `commit`, `dialect`, `builtBy`, `dir`). A local entry omits `tag`, `repository`, `digest` and `signer`. Example:

```json
{
  "schema": "docs.opmodel.dev/lock/v1",
  "tool": "0.1.0",
  "config": "sha256:<64 hex>",
  "bundles": [
    {
      "project": "catalog-opm",
      "root": "/catalogs/opm/",
      "segment": "4.4",
      "tag": "4.4",
      "repository": "ghcr.io/open-platform-model/docs/catalog-opm",
      "digest": "sha256:<64 hex>",
      "version": "4.4.5",
      "revision": 0,
      "commit": "<40 hex>",
      "dialect": 1,
      "builtBy": "0.1.0",
      "signer": {
        "workflow": "https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml@refs/tags/v0.1.0",
        "repository": "https://github.com/open-platform-model/catalog_opm",
        "ref": "refs/heads/main"
      },
      "dir": "catalog-opm/4.4"
    },
    {
      "project": "catalog-opm",
      "root": "/catalogs/opm/",
      "segment": "edge",
      "local": true,
      "version": "edge",
      "revision": 0,
      "commit": "<40 hex>",
      "dialect": 1,
      "builtBy": "0.1.0",
      "dir": "catalog-opm/edge"
    }
  ]
}
```

Every entry carries `root`, the tab's placement root, so a consumer maps a `dir` to its URLs without reading `bundles.cue`.

## C8. URLs and links

A tab bundle's `content/<path>` publishes at `<root><segment>/<page URL>`, where `x/_index.md` is `x/` and `x/y.md` is `x/y/`. For `catalog-opm` 4.4: `content/traits/backup.md` is `/catalogs/opm/4.4/traits/backup/`; edge is `/catalogs/opm/edge/traits/backup/`.

Page paths the renderer writes:

| Path | Page |
|---|---|
| `_index.md` | landing: the authored root `_index.md` when a `markdown` source has one (catalog_opm supplies the contract page), followed by the generated "Catalog members" block; otherwise a generated page holding only that block |
| `blueprints/_index.md`, `resources/_index.md`, `traits/_index.md` | kind index, weights 1, 2, 3 |
| `<kind>/<name>.md` | the member page of the newest `apiVersion` of that name and kind |
| `<kind>/<name>-<apiVersion>.md` | every older `apiVersion` of the same name and kind in the same build |

The newest `apiVersion` is the most stable level, then the highest number (`v1` > `v1beta2` > `v1beta1` > `v1alpha1`). So the bare path names the newest contract of a name in every minor, and the switcher keeps a reader on it.

Links the renderer writes, and the forms the dialect lint (C11) allows:

| From | To | Form |
|---|---|---|
| a tab page | a page of the same bundle | `<root><segment>/<path>/`, e.g. `/catalogs/opm/4.4/resources/volumes/`; must resolve inside the bundle |
| a tab page | another catalog (none in phase 1) | its major alias, `/catalogs/<name>/<MAJOR>/...` |
| a tab page | the docs | `/docs/<section>/<page>/`; the site resolves it in its default version |
| a tab page | an enhancement | `/enhancements/<NNNN>/` or `/enhancements/<NNNN>/<document>/` |
| a docs page (any repository's `docs/site/`) | a catalog | `/catalogs/<name>/<MAJOR>/` plus an optional page path; never a minor or `edge` |

An authored page in a bundle (a `markdown` source) links into its own catalog through the major alias, `/catalogs/<name>/<MAJOR>/<path>/`, the form that also reads correctly on GitHub and in docs mode. When `<MAJOR>` is the build's major (or the build is edge), the `markdown` source rewrites that link to the build's own segment, `/catalogs/<name>/<segment>/<path>/`, before bundle-mode lint checks that it names a page of the bundle. A link inside a fenced code block is an example and is left as written (added in review, 2026-10-02).

Aliases the site serves: `/catalogs/<name>/` and `/catalogs/<name>/<MAJOR>/` go to the newest minor (of that major), and `/catalogs/<name>/<MAJOR>/<path>/` to the same path in that minor, so docs pages can deep-link through the alias. For `catalog-opm` the contract page is the landing, so `/catalogs/opm/4/` replaces `/docs/reference/catalog-contract/`.

## C9. Signing identity

Every publishing mode signs the pushed digest with cosign keyless from the reusable workflow's GitHub OIDC token. The certificate's subject is the **reusable** workflow; the caller appears only in Fulcio's extensions. `pull` and `promote` accept a bundle only when its Sigstore bundle verifies with sigstore-go against the public-good trusted root (signed certificate timestamp, transparency-log entry and an observer timestamp, one each), with an artifact digest equal to the manifest digest, and:

| Certificate field | Must equal |
|---|---|
| issuer | `https://token.actions.githubusercontent.com` |
| SAN (Build Signer URI) | `https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml@<ref>` with `<ref>` matching a `signer.refs` glob, default `refs/tags/v[0-9]*` |
| Source Repository URI | `https://github.com/<tabs[project].repo>` |
| Source Repository Ref | `refs/heads/main` |

The default ref glob is `refs/tags/v[0-9]*`, so only a release tag of docs-kit qualifies; a tag such as `vendor-x` never matches.

The phase-1 spike (its workflows are in git history at 4aaf0f9) proved this split on GHCR: a reusable `spike-sign.yml` signs on behalf of a calling `spike.yml`, and verification passes only with the reusable workflow as SAN and docs-kit as Source Repository URI. The Source Repository URI check is the tenant check: without it any repository calling `publish.yml` could sign a bundle for another project. `promote` applies the same policy with the repository taken from `GITHUB_REPOSITORY`. Humans can check the same thing:

```text
cosign verify ghcr.io/open-platform-model/docs/catalog-opm@sha256:<hex> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp '^https://github\.com/open-platform-model/docs-kit/\.github/workflows/publish\.yml@refs/tags/v[0-9]' \
  --certificate-github-workflow-repository open-platform-model/catalog_opm \
  --certificate-github-workflow-ref refs/heads/main
```

## C10. The doc model, `data/catalog.json`

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

`mark` is `"not-implemented"`, `"provided-by-platform"` or `null`. `demand` is `"required"` or `"optional"`. `spec.fields` is the structured spec, read from the evaluated value: every field under the spec key, depth first, regular, optional and required alike (`presence` is `"regular"`, `"optional"` or `"required"` from the selector's constraint type), with `path` dot-separated, `[]` for a list element and `[string]` for a pattern constraint; `type` the formatted constraint; `default` the formatted default or `null`; `ref` the definition name when the field's value is a definition outside this member's package, where the walk stops instead of expanding it. The walk stops at the module boundary and at a visited definition, and caps depth at 12. `spec.cue` keeps the authored text, so a field the walk cannot express still shows on the page. Order: siblings appear in the order CUE's `Fields` iterator yields them for the evaluated value (declaration order), each field followed by its descendants, so the list is stable for one source and tool version. `type` is the field's constraint expression printed with `cue/format` (`format.Simplify()`), collapsed to one line with single spaces; `default` likewise. Both are stable for one tool version, and phase 1b compares them only between bundles built by the same docs-kit minor.

**Additions made in implementation** (additive, so no schema-id change): a top-level `docNotes` lists the repository notes (`docs/<name>.md`) that member notes mention and that exist at the commit built, because the renderer reads no source and the doc-note rule links a note only when the file exists; each `external` entry carries `definition` (as the spec block writes it, `k8s.#EnvVar`) beside `package` and `vendored`, and a `linked` entry's `definition` is likewise the written form (`res.#ContainerSchema`), so the "Defined elsewhere" list renders from the model alone. Members are listed in catalog-map order (`#resources`, `#traits`, `#blueprints`, each by FQN key); `category`, `wrapper`, `fulfilment`, `optional` and `mark` are `null` where they do not apply, and every list is present, empty or not. A field's `doc` is its doc comment, cleaned like notes, paragraphs separated by a blank line.

**What the walk expresses (verified 2026-10-02 with the spike program `hack/spike/specdump`, since removed and kept in git history at 4aaf0f9, on catalog_opm `opm-v4.4.5`, members `container`, `volumes`, `backup`).** Two runs printed identical output (121 lines); `backup` lists `schedule` and `retention` with presence `required`. 

- **Struct-level validators and if-guards.** A struct carrying `matchN` (`#BackupRetentionSchema`'s "at least one tier", `#VolumeSchema`'s "exactly one source") or an `if` comprehension over a non-concrete field (`container.image.reference`) has an open kind after evaluation (`_` or `_|_`), and CUE's `Fields` iterator refuses it. The walk then takes the field names, in declaration order, from the value's syntax and looks each field up by selector, so every declared field still appears with its type and presence. The validator itself and the guarded value are not expressed in `spec.fields`; `spec.cue` shows them. A member of a "one of" set appears as an optional field.
- **Disjunction defaults.** `type` keeps the alternatives with the default marker (`*"any" | "snapshot" | "filesystem"`); `default` is the marked value (`"any"`). `default` is set only when the value has a default and is not itself concrete, so a list type such as `[string, ...string]` reports no spurious default.
- **Pattern constraints.** `[string]: T` is walked as `[string]`. A pattern with another label constraint (`[=~"^a"]: T`) is not expressed; opm 4.4.5 has none.
- **Lists.** A list field has `type` `"list"` and its element type is walked under `[]`; open (`[...T]`) and non-empty (`[T, ...T]`) lists are not told apart, `spec.cue` shows the minimum length.
- **Types.** A struct has `type` `"struct"`; a scalar's `type` is its evaluated constraint (`cue.Final()`) with the import declarations `Syntax` adds for builtins dropped, so `strings.MaxRunes(128)` stays qualified. Final simplification can drop a base kind next to a validator (`string & strings.MaxRunes(128)` prints as `strings.MaxRunes(128)`); phase 1b compares these strings only between bundles of one docs-kit minor, so the loss is stable.
- **`ref`.** A field one of whose conjuncts references a definition in another package, the module's own shared packages included (`schemas.#CronSchema`, `schemas.#NameType`), gets `ref` `"<import path>.<definition>"` (`opmodel.dev/catalogs/opm/schemas.#CronSchema`) and is not expanded; its `type` is the resolved constraint. A definition of the member's own package (`#BackupRetentionSchema`, `#VolumeSchema`) is expanded in place.

## C11. The page dialect, version 1

`opm-docs lint` ports `opmodel.dev/site/scripts/lint-sources.sh` rule for rule and adds the `/catalogs/` link forms (C8). Dialect 1 is exactly that set; opmodel.dev keeps its shell lint at the same set (below). The rules:

- A tree holds only regular `.md` files: no symlink, no `.mdx`, no other file; names are lower-case kebab-case; no `index.md` (a section page is `_index.md`).
- Front matter opens on line 1 with `---` and closes with `---`; keys are only `title`, `description`, `type`, `weight`; `title` and `description` are required and non-empty; `type` is required on a leaf page and is one of `tutorial`, `how-to`, `explanation`, `reference`; an `_index.md` declares no `type`; `weight` is a positive integer; `sidebar:` is named as Starlight front matter.
- Shortcodes, checked on every line, code fences included: only `{{< opm/<figure> >}}` with one of the seven figure names, no parameters and no closing tag; `{{</* ... */>}}` is a shown shortcode and passes.
- No `:::` line, no `import ... from` line, no component tag line, no image (`![`, `<img`), no raw `href=` or `src=`.
- Every code fence carries a language tag.
- An alert marker is exactly `> [!NOTE]` (or `TIP`, `IMPORTANT`, `WARNING`, `CAUTION`) alone on its line.
- Every link destination (inline and reference definitions) is `http:`, `https:`, `mailto:` or `#...`, or one of: `/docs/(<seg>/)*` with an optional fragment; `/enhancements/`, `/enhancements/<NNNN>/` or `/enhancements/<NNNN>/<document>/` with `<document>` one of `problem`, `design`, `decisions`, `graduation`, `risks`, `operational`, `questions`; `/catalogs/<name>/` or `/catalogs/<name>/<segment>/(<seg>/)*` with `<segment>` a major (`4`), a minor (`4.4`) or `edge`, with an optional fragment.
- Bundle mode (`lint --bundle <dir>`, run by `build` and `pull`) adds: a link into the bundle's own root uses the bundle's own segment and names a page in the bundle; and `manifest.json` lists exactly the pages present; and a link to another catalog uses its bare root or a major segment, as C8's table requires (implementation decision, accepted 2026-10-02).
- Docs mode (the default) adds: a `/catalogs/` link is the bare tab root (`/catalogs/opm/`) or uses a major segment (`/catalogs/opm/4/...`); a minor or `edge` segment is a violation.

**Agreement with the site's shell lint until phase 3**. Until phase 3 retires `opmodel.dev/site/scripts/lint-sources.sh`, the site lints `docs/site/` trees with the shell script and every bundle with `opm-docs lint`, so the two must agree. They are kept in agreement by one conformance fixture set, `internal/dialect/testdata/conformance/`, each fixture paired with its expected `<file>:<line>` output, which both linters must pass:

- **Initial set:** a copy of exactly the fixture directories under opmodel.dev `site/tests/lint/`, with the opmodel.dev commit recorded in its README. `site/tests/dialect/` is not part of it (those fixtures test the site's build checks, not the lint).
- **Source of new fixtures:** docs-kit. The `/catalogs/` link fixtures, and any fixture for a later rule, are written in docs-kit first, since the Go lint is the reference.
- **Re-sync rule:** a rule change lands in docs-kit first, with its fixture, in a docs-kit release; opmodel.dev copies the changed fixtures into `site/tests/lint/` and updates the shell lint in the same PR that bumps its pinned `opm-docs` to that release. Neither side changes a rule without the other's fixture.

Every violation prints as `<file>:<line>: <message>`, the shell lint's format.

## C12. Tool distribution

Callers and the site never `go run` or `go install` `opm-docs`: every consumer runs a released binary, so the bytes that built a bundle are the bytes a release names.

- **Assets.** Every docs-kit release `vX.Y.Z` carries `opm-docs_X.Y.Z_<os>_<arch>.tar.gz` for `linux_amd64`, `linux_arm64`, `darwin_arm64` and `darwin_amd64` (each holding the `opm-docs` binary and `LICENSE`, the Apache-2.0 text at docs-kit's root; the org adopted Apache-2.0 on 2026-10-02), and `checksums.txt` (SHA-256, `sha256sum` format, one line per archive). Built by goreleaser in a draft-first release workflow, the pattern cli already uses. URL: `https://github.com/open-platform-model/docs-kit/releases/download/vX.Y.Z/<asset>`.
- **Pinned in a build image**. A consumer may instead pin the `linux_amd64` archive by its SHA-256 in its own build image (opmodel.dev does this in `site/Dockerfile`, as it pins Hugo and Pagefind) and run `opm-docs` inside that image, with network on for the `pull` step only. The SHA-256 it pins is the archive's line in that release's `checksums.txt`. docs-kit therefore keeps shipping the `linux_amd64` archive and `checksums.txt` in every release, under the names above.
- **Pin file.** A caller of `publish.yml`, and any consumer that runs the tool on the host (catalog_opm for its local `docs:bundle` tasks; opmodel.dev only if it does not use the image pattern), pins it in a repo-root file `.opm-docs-version`: one line, the release tag (`v0.1.0`), matching `^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$`. `publish.yml` reads the same file (C5), so every caller of `publish.yml` has one; its tag and the `publish.yml@` ref name the same release and move in one PR.
- **Verification.** Download the archive and `checksums.txt` for the host's os and arch, check with `grep ' <archive>$' checksums.txt | sha256sum -c -` (refusing an archive with no line), then extract only `opm-docs`. A failed check stops the task; nothing falls back to building from source. The install target is a gitignored repo-local directory (`.bin/` or `site/.bin/`), never a global path.

## Site decisions

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
| `pull` | see C7; `--config` (path, `bundles.cue`), `--out` (path, `.bundles`), `--lock` (path, `<out>/lock.json`) | Resolve, verify, unpack, lint, lock. |
| `revise` | `--project`, `--tag`, `--fix` (required), `--out` (path, `out`), `--registry` (string, `ghcr.io/open-platform-model/docs`), `--config` (path; default `docs-kit.cue` in the release tree, else the current directory) | Build the next docs revision of a published release into `out/<project>/` ("Docs revisions" below). Pushes nothing. Exit `1` for a missing flag or an invalid config, `2` when a step refuses. |
| `version` | none | Print `opm-docs <version>`. |

`build` also takes the hidden `--revision` (int, `0`) and `--patches` (commits, repeatable) that `revise` passes: they write `revision` and `source.patches` (C3), need `--release`, and accept a work tree that is exactly the release commit plus the staged fixes.

## Docs revisions

A published release's pages change only through a docs revision (DESIGN decision 6): its full tag is never overwritten, so the fix is built as the next revision of the same version. `opm-docs revise --project P --tag T --fix F` runs in the checkout of `main` (the workflow's `revision` mode) and takes these steps; each refusal exits `2` with the message shown:

1. **The fix.** `F` is a full 40-hex hash of a commit with exactly one parent that is an ancestor of `origin/main` ("the fix must land on main first"; "has 2 parents ... not a merge"), and not already in `T` ("already in the release").
2. **The release.** `T` carries the project's tag prefix (from the checkout's `docs-kit.cue`, or `--config`), and the registry holds revision `0` of its version `V` ("publish the release first: dispatch mode: release"). The next revision is `1 +` the highest published for `V` (C4 rule 3).
3. **The fixes so far.** The newest published revision's `manifest.json` (its layer unpacked with every check `pull` applies) must name `V`, that revision and `T`'s commit as `source.commit`. Its `source.patches` (empty for revision `0`) plus `F` is the new list. `F` already in the list is refused ("already applied in `V.<n>`"), except a re-run: when `F` is the last fix of revision `n` and the release tag `V` does not name revision `n` (pushed, never promoted), revision `n` is built again with the same list. The build is deterministic, so `push` finds the same digest and writes nothing, and signing and promote finish the run. Every earlier fix is checked again as in step 1.
4. **The patched tree.** `git worktree add --detach <tmp> T`, then `git cherry-pick --no-commit` of each fix in order. A conflict is refused naming the files ("land one fix on main that makes the whole change and revise with that"). The worktree is removed on every path.
5. **Documentation only.** `T`'s tree and the worktree's index (`git write-tree`) are compared with `git diff-tree -r -M`:

   | Path | Allowed |
   |---|---|
   | `*.md` | added, changed, renamed (from `.md`), removed |
   | `*.cue` | changed only; both versions scanned with comments skipped give the same token sequence (a comma inserted at a line end equals a written one; interpolations resumed as the parser does) |
   | `*.go` | changed only; both versions scanned with comments skipped give the same token sequence (an inserted semicolon equals a written one); the directive comments (`//go:build`, `//go:embed`, `//line`, `//export`, any `//word:word`, `// +build`) equal in order; never a file that imports `"C"` |
   | a symlink or submodule, at any path | refused |
   | anything else, or an added, renamed or removed `.cue` or `.go` file | refused |

   Layout is not compared: a comment added above a field moves the code below it, which a comparison of formatted output would refuse. A change to `metadata.description`, a default, a constraint or an attribute is a value change and refused. Every refusal is listed, one line per file, as `<path>: <why>; a change to code needs a patch release`.
6. **The build.** `build --release T --source <tmp> --revision <n> --patches <list>` into `--out`: `source.commit` is `T`'s commit, `source.ref` is `T`, `created` is `T`'s commit time, and a patched file's `lastmod` is the committer date of the newest fix that touched it.

`revise` pushes nothing: the workflow's `push`, `cosign sign` and `promote` follow, as for a release, so a tag never points at an unsigned build. Edge is never revised; it is rebuilt on every push to `main`. A fix that changes code and documentation together is refused whole: split it, or ship a patch release.

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

with `%s` "a component that attaches it still renders, and the render warns that the trait is not handled and ignores its values" for an advisory trait, else "rendering a component that declares it fails" (resource) or "... that attaches it fails" (trait). The kind index lists the marked members in the two sentences refgen writes today. The landing always ends with a generated block:

```markdown
## Catalog members

`opmodel.dev/catalogs/opm@v4` version `4.4.5`.

- [Blueprints](/catalogs/opm/4.4/blueprints/): 5
- [Resources](/catalogs/opm/4.4/resources/): 12
- [Traits](/catalogs/opm/4.4/traits/): 28
```

For edge the second line reads "`<module path>` at `main` (commit `<12 hex>`), unreleased." When a `markdown` source supplies a root `_index.md`, its front matter and body come first, unchanged except for C8's alias pinning, then one blank line and the block; the authored body must not already hold a `## Catalog members` heading (the build refuses it, naming the file). The page is recorded in `manifest.json` with `generated: false` and `source` the authored file. Without an authored landing, the renderer writes front matter (`title` "<catalog name> catalog", `description` "Every member of `<module path>`, by kind.") and the block alone, `generated: true`.

No page carries a generator marker comment: a bundle's pages are wholly generated and never committed, so there is no authored text to keep apart.

**Parity with refgen (2026-10-02, tasks 3.6).** `TestCatalogOPMParity` on a catalog_opm checkout at `opm-v4.4.5` compared all 45 member pages with refgen's committed pages after normalizing the planned differences (no marker comments, links in the bundle's own segment, the contract link pointing at the landing, the newest apiVersion on the bare path). All 45 matched; no other difference was found.
