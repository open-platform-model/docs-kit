# docs-kit contracts

Everything here is read by another repository: the bundle format, the tag scheme, the reusable workflow, `docs-kit.cue`, the site's pull config and lock, the URL and link forms, the signing identity, the doc model, the page dialect and how the tool is distributed. A change to any of it follows constitution Principle II (`openspec/config.yaml`): prefer additive changes, bump the schema identifier for a breaking one, and name every consuming repository and what it must do.

The numbers (C1 onward) are stable; other repositories cite them as `docs-kit C5`. The approved design and its decisions are in [DESIGN.md](../DESIGN.md). These contracts were fixed by the OpenSpec change `build-opm-docs-phase-1` and extended by later changes (archived under `openspec/changes/archive/`), which hold the reasoning behind each. `docs/orchestration.md` lists which change adds which number.

## Doc-comment rules

How every extractor turns source comments into reader-facing text (`internal/doctext`, `internal/mdtext`). `cue-definitions` departs from the summary, escaping and spec-block wrapping rules where C17 says so, and `go-api` from the summary and escaping rules where C20 says so (a Go doc comment is Go doc syntax, not prose):

- **Maintainer comments.** Inside a definition, a comment group whose first line (after `//` and trimming) starts with `WHY` or with `//` (a `////` banner) is dropped. A `WHY` line inside a doc comment is dropped on its own (core's rule, adopted now).
- **Citations.** Removed from prose and from comments in spec blocks. The pattern accepts `0010:D28`, `0010 D28`, `OQ` numbers, `:R2` and `/R1/R2` requirement suffixes, `/D9` continuations, lists joined by `,`, `;` or `and`, an optional `(see|per|enhancement)` lead and the parenthesised form; then refgen's six clean-up rewrites (empty parens, comma before paren, space after paren, space before punctuation, double spaces, trim). Also removed: `See SPEC.md § N.` sentences, inline `SPEC.md § N`, `NNNN experiment N` and `enhancements/NNNN/experiments/...` (core's rules, adopted now). A changed comment paragraph in a spec block is re-wrapped to `80 - indent - 3` columns, minimum 40; an unchanged one keeps its breaks; a trailing comment that becomes empty is removed.
- **Citation policy.** Every extractor source takes `citations` (C6): `"strip"`, the default, applies the rule above; `"link"` turns each enhancement decision citation outside a code span (`0010:D28`, `0010:D28:R2`, `0010:D28/D29`, each entry of a list) into a Markdown link, `[0010:D28](/enhancements/0010/decisions/)`, its text as written, and still removes every other form (`OQ` numbers, `0010 D28`, a citation that runs on into an `OQ`, `SPEC.md § N`, experiments). A citation inside a code span or a spec block's comment is removed under both policies: a link cannot live there. A lead-in word or parenthesis around a linked citation stays as written. `cue-catalog` takes only `"strip"`: its doc model holds plain prose that its renderer escapes, so a link needs a doc model that carries one. The `crd` doc model holds plain citations and its renderer links them (C18). A `markdown` source takes no `citations`; authored text is copied as written.
- **Summary.** A member's doc comment opens with `metadata.description` plus `.` (whitespace collapsed), or the build refuses the member naming both texts. The summary is the page's front-matter `description` and is not repeated in the body. The remaining paragraphs are the notes.
- **Escaping.** Outside backtick code spans: `\ < > * _ [ ] |` are backslash-escaped and `{{` becomes `{\{`. The `crd` renderer also escapes `~` and `#` (C18). A table cell escapes every `|`, inside code spans too. A code span lengthens its fence past any backtick it holds. A YAML front-matter string escapes `\` and `"`. A rendered page that still holds `{{<` or `{{%` fails the build.
- **Doc-note links.** `docs/<name>.md` in prose becomes a link to that file on the source repository at the bundle's commit, only when the file exists and the text is outside a code span.
- **Marks.** A resource or trait that no transformer in its own catalog requires or optionally reads is marked **Provided by your platform** when its `fulfilment` default is `provider`, and **Not implemented** otherwise; a blueprint is never marked. Exact strings in "Page renderer".
- **Enforcement tags.** Only where derivable: the spec schema and each required match label are enforced by `cue`; a provider-fulfilled contract's single provider and a load-bearing trait's refused render are enforced by the `kernel`. Nothing else is tagged.

## C1. Where bundles live

```text
ghcr.io/open-platform-model/docs/<project>
```

Every project, planned ones included, so changes built in parallel need not edit this table:

| Project | Repository | Release tag prefix | Placement | Phase |
|---|---|---|---|---|
| `catalog-opm` | catalog_opm (`opmodel.dev/catalogs/opm@v4`) | `opm-v` | tab, `/catalogs/opm/` | 1 |
| `core` | core | `v` | docs, owns `reference/definitions/` | 2 |
| `opm-operator` | opm-operator | `v` | docs, owns `reference/operator-resources.md` | 2 |
| `cli` | cli | `v` | docs, owns `reference/cli/`, carries `pins` | 2 |
| `library` | library | `v` | docs, owns `reference/go-api/` | 2 |
| `catalog-opm-docs` | catalog_opm | `opm-v` | docs (its `docs/site/`) | 3 |
| `opm` | opm | `v` | docs (its `docs/site/`) | 3 |
| `enhancements` | enhancements | none (edge only) | section `/enhancements/` | 3 |

A project name matches `^[a-z0-9]+(-[a-z0-9]+)*$`. **Naming rule**: a repository's docs-placed bundle is named after the repository, `_` becoming `-`; when that name is already a tab project of the same repository, it takes the suffix `-docs` (hence `catalog-opm-docs` beside `catalog-opm`).

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

The layer is a gzip-compressed tar of the bundle tree (C3), built deterministically: entries sorted by path, directories before their contents, mode `0644` for files and `0755` for directories, uid and gid `0`, empty user and group names, modification time equal to `created`, no extended headers, gzip header with no name and no time. The same source, config and tool version give the same digest. A docs bundle's `pages[].edit` (C3) is one more input: it depends on the `HEAD` of `main` when the bundle is built, so the same release built after `main` renamed or deleted an authored page gives a different digest.

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
	if placement.kind == "tab" {pages: [...{edit?: error("a tab bundle's page has no edit")}]}
	if placement.kind == "section" {
		version: "edge"
		pages: [...{edit?: error("a section bundle's page has no edit")}]
	}
	data: [...#DataFile]
	// The exact versions of other projects this build documents against
	// (DESIGN decision 10), from the config's pins command.
	pins?: [#Project]: #SemVer
}

#Project: =~"^[a-z0-9]+(-[a-z0-9]+)*$"
#SHA:     =~"^[0-9a-f]{40}$"
#SemVer:  =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
#Version: #SemVer | "edge"

#Placement: {
	// "tab": its own section with its own versions, /catalogs/<name>/<MAJOR.MINOR>/.
	// "docs": merged into a site version's /docs/ tree (refused by a pull that predates it).
	// "section": one unversioned section outside every site version, built from
	// main only (refused by a pull that predates it).
	kind: "tab" | "docs" | "section"
	if kind == "tab" {root: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$"}
	if kind == "docs" {
		root: "/docs/"
		// Paths under content/ this bundle owns exclusively: a directory
		// ending "/" or a page ending ".md". Every generated page of the
		// bundle lies under one; two of them never nest.
		owns: *[] | [...#Owned]
	}
	// The one section root, until a second section has a consumer.
	if kind == "section" {root: "/enhancements/"}
}

#Owned: =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$|^([a-z0-9]+(-[a-z0-9]+)*/)*[a-z0-9]+(-[a-z0-9]+)*\\.md$"

#Page: {
	path:      =~"^([a-z0-9]+(-[a-z0-9]+)*/)*(_index|[a-z0-9]+(-[a-z0-9]+)*)\\.md$" // under content/
	source?:   string & !=""                                                        // repo-relative file the page came from at the commit built ("View source")
	lastmod?:  time.Time                                                            // that file's last commit date at the commit built, RFC 3339
	generated: bool                                                                 // generated reference, or an authored page
	// A docs bundle's authored page only: its source file's path on the
	// repository's main branch, when main still has that file ("Edit this
	// page"). Never on a generated page or a tab bundle's page (#Manifest),
	// so a pull that predates it still reads every tab bundle.
	edit?: #RepoPath
	if generated {edit?: error("a generated page has no edit")}
}

// A repository-relative file path: no leading "/" or ".", no ".." segment.
#RepoPath: string & =~"^[^/.]" & !~"(^|/)\\.\\.(/|$)"

#DataFile: {
	path:   =~"^[a-z0-9-]+\\.json$" // under data/
	schema: string & !=""           // the file's own schema id, e.g. "docs.opmodel.dev/data/cue-catalog/v1"
}
```

Tightened from DESIGN.md: closed structs, the SHA and SemVer patterns, `revision: 0` for edge, `patches`, typed `data` entries, and a placement root bound to its kind. The schemas rely on definition closedness rather than `close()` (implementation finding): a struct passed to `close()` is evaluated on its own, so its `if kind == "tab"` guard never sees the data's `kind`, and `close()` does not close a nested pattern map such as `bundles: [#Project]: #Bundle`, which then accepted any project name. A definition closes every struct inside it, so the shipped files drop `close()` with the same meaning. `created` (optional; implementation finding) records the source commit's time: the bundle-format spec requires every annotation to equal a `manifest.json` field and `push` takes only `--dir`, so the time `push` stamps on the tar entries and on `org.opencontainers.image.created` has to travel in the manifest. `build` always writes it and `push` refuses a manifest without it; `pull` accepts one without it, so a fixture tree written to the original shape still validates. `pages` lists every file under `content/`, and only those; `data` lists every file under `data/`, and only those. `lastmod` is set when the checkout has the file's history (the workflow checks out with full history) and omitted otherwise.

Added by the change `generalize-build-assembly`, both optional and additive:

- `placement.owns`, on a docs placement only: the paths under `content/` the bundle owns exclusively, each a directory ending `/` or a page ending `.md` (`#Owned`). Build rules: C15.
- `pins`: the exact versions of other projects the build documents against (DESIGN decision 10), written from the config's pins command (C6, C15). A build without `pins` in its config writes none.

Added by the change `add-authored-docs`, optional and additive:

- `pages[].edit`, in a docs-placed bundle only, on a page with `generated: false` (a completed page included, C15): the repository-relative path of the page's source file, written when that path is a regular file in the `HEAD` of the main tree (C15, "Authored docs"), and absent when `main` no longer has it there. A file renamed or deleted on `main` since the release is not followed: the page has no `edit` rather than a guessed one (DESIGN decision 19). A generated page, and every page of a tab bundle, never has it, and the schema refuses either ("pages.0.edit: a tab bundle's page has no edit"), so a site whose `opm-docs` predates the field still pulls every tab bundle; `edit` is a repository-relative path (`#RepoPath`: no leading `/` or `.`, no `..` segment); a docs bundle carrying it needs the site's bump first (C12).

Added by the change `add-enhancements-bundle`:

- `placement.kind: "section"`, root `/enhancements/` (C21): one unversioned bundle outside every site version. Its `version` is always `edge` and none of its pages carries `edit` ("pages.0.edit: a section bundle's page has no edit"). `#Placement` is closed, so an `opm-docs` that predates the change refuses a section bundle: the site bumps first (C12).

`#Manifest` is closed, so an `opm-docs` older than a field refuses a bundle that carries it; C12 orders the bumps so that never happens on the site.

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
2. **Full tags are immutable.** `push` refuses to write a full tag that already names a different digest ("`4.4.5.0` is already published as sha256:...; a documentation fix is a docs revision"). The same digest is a no-op, so a re-run of a failed workflow is safe. For a docs bundle that holds only while `main` is the same commit: a re-run of the failed run (the same `github.sha`) builds the same bytes, while a fresh `release` dispatch after `main` renamed or deleted an authored page builds a different `edit` and is refused here. A rebuilt docs revision keeps the `edit` values it recorded ("Docs revisions", step 3).
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
      setup-go:
        description: Install Go (actions/setup-go, pinned by SHA) from the source tree's go.mod, for repository commands that run Go (docs-kit C14)
        type: boolean
        default: false
    outputs:
      digest:
        description: The pushed manifest digest (empty in check mode)
        value: ${{ jobs.publish.outputs.digest }}
      tag:
        description: The full tag written (empty in check and edge modes)
        value: ${{ jobs.publish.outputs.tag }}
```

`setup-go` (added by `generalize-build-assembly`): when `true`, the build job installs Go with a SHA-pinned `actions/setup-go`, its cache off (a cache restored where the caller's code runs is an input another run's code could have written), before building, for a repository command (C14) that runs `go`. The Go version comes from `src/go.mod` (the release tree) in `release` mode and from the checkout of `main`'s `go.mod` otherwise, `revision` included: its patched release tree exists only inside `revise`, and Go builds an older module with a newer toolchain.

**Two jobs** (since `generalize-build-assembly`; inputs, outputs and the caller's grant unchanged). A repository command runs the caller's own code during its build, so the job that builds is not the job that signs:

| Job | Runs | Permissions it declares | Caller's code |
|---|---|---|---|
| `build` | the mode check; checkout(s) with `persist-credentials: false`; install `opm-docs`; `setup-go` when asked; `check`, `build` or `revise`; upload `out/<project>/` as the workflow artifact `docs-bundle-<project>-<run id>-<run attempt>` (retention 1 day; not in `check` mode) | `contents: read`, `packages: read` | yes |
| `publish` | `needs: build`, skipped in `check` mode; install the `opm-docs` release the build job installed (its tag passed as a job output); download the artifact into `out/<project>/` and recreate `content/` and `data/` (an artifact drops an empty directory); the identity check below; GHCR login; `push`; `cosign sign`; `promote` | none (inherits the caller's `packages: write`, `id-token: write`) | none: no checkout of the caller |

The build job can never request an OIDC token, so a compromised dependency of the caller cannot sign a bundle, and it holds no credential a repository command could read (C14): its checkouts persist none and it never logs in to a registry. `push` packs deterministically from the tree in whichever job runs it.

**Identity check.** The tree comes from a job that ran the caller's code, so before `push` the publish job refuses it unless `manifest.json` has `project` equal to `inputs.project`, `source.repo` equal to `github.repository`, and `source.ref` equal to `inputs.tag` (`release`, `revision`) or `version` equal to `edge` (`edge`). Each refusal names the value found and the one expected; nothing is pushed or signed.

No secrets are declared: the workflow uses `github.token`. The `publish` job declares no `permissions`, so it runs with the grant of the caller's job (a called workflow cannot raise it, and a job that asked for more than a check-only caller grants would fail that caller at start-up); the `build` job declares only the read grant every caller gives. The caller's job MUST grant:

| Mode | `contents` | `packages` | `id-token` |
|---|---|---|---|
| `check` | `read` | `read` | none |
| `edge`, `release`, `revision` | `read` | `write` | `write` |

**Registry login.** Only the publish job logs in to `ghcr.io`, with `github.token` (`docker login ghcr.io -u ${{ github.actor }} --password-stdin`); that login, under the caller's `packages: write`, is the credential `push` and `promote` use. The build job does not log in (since `generalize-build-assembly`; before it, it logged in for reads): a credential file there would be readable by a repository command. The extractor's CUE dependency resolution (`opmodel.dev/core@v2` and the catalog's other dependencies) and `revise`'s registry reads go to public GHCR packages anonymously. A caller whose CUE dependencies are private cannot build with this workflow.

What each mode does (all but `check` refuse unless `github.ref` is `refs/heads/main`):

| Mode | Caller runs it on | Steps |
|---|---|---|
| `check` | `pull_request` | checkout the PR head; `opm-docs check --project P` |
| `edge` | `push` to `main` | checkout `github.sha` with full history; `build --edge`; `push`; `cosign sign`; `promote` |
| `release` | the job that runs release-please, gated on that package's release (not on `release: published`); or `workflow_dispatch` for a release that has no bundle yet | checkout the commit of `main` the workflow runs on (`github.sha`) and, at `src/`, the tag with full history; `build --release <tag> --source src`; `push`; `cosign sign`; `promote` |

| `revision` | `workflow_dispatch`, inputs `tag` and `fix` | refuse a `fix` that is not 40 hex; checkout `github.sha` of `main` with full history (it brings `origin/main` and the tags); `revise --project P --tag <tag> --fix <fix>` ("Docs revisions" below); `push`; `cosign sign`; `promote` |

The `revision` mode came with docs-kit `v0.2.0` (change `add-docs-revisions`), without changing the other three. A caller on `v0.1.0` gets an error naming the mode if it asks for `revision`.

Signing: `sigstore/cosign-installer` pinned by SHA with a pinned cosign v3 release, then `cosign sign --yes --new-bundle-format=true ghcr.io/open-platform-model/docs/<project>@<digest>`. Never a tag. Every action is pinned by commit SHA with its version in a comment, like the sibling repositories' workflows.

**Concurrency**. GitHub keeps at most one running and one pending run per concurrency group, and a new pending run cancels the older pending one whatever `cancel-in-progress` says. One shared group per project would therefore let a burst of `main` pushes cancel a queued release publish. The groups are declared at the **workflow level** of `publish.yml` (top-level `concurrency:`), so one group covers both jobs of a run: in `revision` mode `revise` picks the next revision number in the build job, and a group on the publish job alone would let two revisions of one release pick the same number (since `generalize-build-assembly`; before it, the groups sat on the one job). The groups:

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
	// Where a release version comes from. A section bundle builds from main
	// only (edge), so its version may be left out and is ignored when given.
	version?: {
		from:   "tag"         // phase 1: the release version comes from the git tag
		prefix: string & !="" // "opm-v": tag "opm-v4.4.5" is version "4.4.5"
	}
	if placement.kind != "section" {version: _}
	sources: [#Source, ...#Source]
	// The exact versions of other projects this build documents against
	// (DESIGN decision 10). The command prints a pins document
	// ({"schema": "docs.opmodel.dev/pins/v1", "pins": {<project>: <version>}})
	// whose keys are exactly projects; build writes them to manifest.json pins.
	pins?: {
		command: #Command
		projects: [#Project, ...#Project]
	}
}

// A repository command (docs-kit C14): argv[0] is looked up on PATH; no
// shell, no globbing. It runs in the source tree and prints one JSON document.
#Command: [string & !="", ...string]

// What an extractor source's citations become: "strip" removes them;
// "link" links each enhancement decision citation to its decisions page.
#Citations: *"strip" | "link"

#Source: #CueCatalog | #Markdown | #Cobra | #CueDefinitions | #CRD | #GoAPI | #Enhancements

#CueCatalog: {
	kind:   "cue-catalog"
	module: =~"^\\./[^/]" // the CUE module root, repo-relative: "./opm"
	// The catalog doc model holds plain prose its renderer escapes, so a
	// catalog strips citations; "link" needs a doc model that carries links.
	citations: "strip"
}

#Markdown: {
	kind: "markdown"
	dir:  =~"^[^/.][^.]*$" // repo-relative directory, copied to content/ as it is
	// Patterns relative to dir, matched against each file's slash path: a
	// path.Match glob ("**" is not special), or a directory ending "/",
	// which matches every file under it. A file is copied when it matches
	// some include (or include is absent) and no exclude.
	include?: [string, ...string]
	exclude?: [string, ...string]
}

// A CUE package's exported definitions, parsed (never evaluated) and grouped
// into reference pages by the author's inclusion list (docs-kit C17).
#CueDefinitions: {
	kind:    "cue-definitions"
	package: =~"^\\./" & !~"(^|/)\\.\\.(/|$)" // the package directory, repo-relative: "./src"
	skip:    *[] | [...string] // path.Match globs on file base names: ["*_pins.cue"]
	// The directory the pages are written to, one of the bundle's owned
	// paths: "reference/definitions/".
	section:     =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$"
	title:       string & !="" // the section index's front matter
	description: string & !=""
	weight?:     int & >=1 // the section index's weight among its siblings
	// The section index's opening paragraph, Markdown; without it the
	// index names the package's module path.
	intro?: string & !=""
	pages: [#DefPage, ...#DefPage]
	// Exported definitions left out of the pages, each with the reason.
	exclude: [=~"^#"]: string & !=""
	citations?: #Citations
}

// One reference page: its file under section and its definitions in order.
#DefPage: {
	file:        =~"^[a-z0-9]+(-[a-z0-9]+)*$"
	title:       string & !=""
	description: string & !=""
	definitions: [=~"^#", ...=~"^#"]
}

// A cobra command tree, printed by the repository's own program through the
// cobradump module (docs-kit C19), rendered as a command reference.
#Cobra: {
	kind:        "cobra"
	command:     #Command                         // prints the dump: ["go", "run", "./hack/docskit-dump"]
	section:     =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$" // the directory the pages go in: "reference/cli/"
	title:       string & !=""                    // the section index's title
	description: string & !=""                    // the section index's description
	weight?:     int & >=1                        // the section index's weight
	citations?:  #Citations
}

// The crd source (docs-kit C18): controller-gen CRDs and their kubebuilder
// samples, rendered as one completable reference page.
#CRD: {
	kind:     "crd"
	dir:      =~"^\\./[^/]"  // controller-gen output: "./config/crd/bases"
	samples?: =~"^\\./[^/]"  // kubebuilder samples: "./config/samples"
	// A sample document containing any of these strings is never shown (a
	// dev or e2e fixture, not something a reader can apply).
	hideSamplesMatching: *[] | [...string & !=""]
	// Labels removed from a shown sample when they carry exactly this value
	// (kubebuilder's scaffold labels say how the repository applies it).
	stripLabels: [string & !=""]: string
	page:        =~"^([a-z0-9]+(-[a-z0-9]+)*/)*[a-z0-9]+(-[a-z0-9]+)*\\.md$" // under content/, an owned path
	title:       string & !="" // front matter when no authored page completes it
	description: string & !=""
	weight?:     int & >=1 // the page's weight when no authored page completes it
	order?: [string & !="", ...string & !=""] // these kinds first, in this order; the rest by name
	reconciledBy?: [string & !=""]: string & !="" // kind: the controller that reconciles it
	citations?: #Citations
}

// The go-api source (docs-kit C20): a Go module's exported packages, parsed
// (never built or type-checked) and rendered from their doc comments.
#GoAPI: {
	kind:   "go-api"
	module: #GoPath // the directory holding go.mod, repo-relative: "./"
	// Page names are package directories relative to root, which is
	// relative to module: "./opm" makes opm/helper/objectset the page
	// helper-objectset.
	root: #GoPath
	// Package patterns relative to module: "./opm/..."; "..." matches any
	// string, and a trailing "/..." also matches the directory itself.
	packages: [#GoPath, ...#GoPath]
	// The directory the pages go in, one of the bundle's owned paths:
	// "reference/go-api/".
	section:     =~"^([a-z0-9]+(-[a-z0-9]+)*/)+$"
	title:       string & !="" // the section index's front matter
	description: string & !=""
	weight?:     int & >=1 // the section index's weight among its siblings
	citations?:  #Citations
}

// A slash path that starts "./" and holds no ".." segment.
#GoPath: =~"^\\./" & !~"(^|/)\\.\\.(/|$)"

// The enhancements repository's entries, INDEX.md and GRAPH.md, as the
// /enhancements/ section (docs-kit C21); a section bundle only. Its text is
// authored, so it takes no citations policy: citations stay as written.
#Enhancements: {
	kind:        "enhancements"
	dir:         *"." | =~"^[^/.][^.]*$" // the entry root, repo-relative
	title:       *"Enhancements" | string & !="" // the section page's title
	description: string & !=""                   // the section page's description
}
```

`bundles` is keyed by project because one repository can publish several, as core and cli will in phase 2 and catalog_opm would with a second catalog. A `layout` choice for catalogs is left out until a second layout has a consumer. A path in content/ written by two sources fails the build, except an authored page at the path of a completable generated page (the catalog landing is one): it does not replace the generated page, the generated body is appended to it (C15).

**Source kinds.** `#Source` is the union of the kinds this `opm-docs` registers; a kind it does not admit exits `1` naming the kind and the kinds it builds. Each extractor kind writes one data file, `data/<kind>.json` with its own schema id (`cue-catalog` keeps `data/catalog.json`, C10), and the renderer registered for that schema turns it into pages, reading the data file as written. A bundle holds at most one source of each extractor kind (a second exits `1` naming both entries). Sources run in config order, then the renderers in the same order.

| Kind | Writes | Options |
|---|---|---|
| `cue-catalog` | `data/catalog.json` (C10); a tab bundle only | `module`; `citations` only `"strip"` |
| `markdown` | authored pages, copied | `dir`, `include`, `exclude` |
| `cue-definitions` | `data/cue-definitions.json` (C17); a docs bundle only | `package`, `skip`, `section`, `title`, `description`, `weight`, `intro`, `pages`, `exclude`; `citations` |
| `cobra` | `data/cobra.json` (C19) | `command`, `section`, `title`, `description`, `weight`, `citations` |
| `crd` | `data/crd.json` (C18) and one completable page; a docs bundle only | `dir`, `samples`, `hideSamplesMatching`, `stripLabels`, `page`, `title`, `description`, `weight`, `order`, `reconciledBy`, `citations` |
| `go-api` | `data/go-api.json` (C20); a docs bundle only | `module`, `root`, `packages`, `section`, `title`, `description`, `weight`, `citations` |
| `enhancements` | `data/enhancements.json` (C21) and the section's pages, which it builds itself; a section bundle only, and the only source there | `dir`, `title`, `description` (no `citations`: its text is authored) |

**Adding an extractor** (one OpenSpec change per kind): an `Extractor` in `internal/build` (`Kind`, `Extract(ctx, Input) (Data, error)`), registered in its `extractors` table; a `Renderer` in `internal/render` (`Schema`, `Render(data, Target) ([]Page, error)`), registered by its data schema; its kind added to `#Source` with `citations?: #Citations` and its own options; its data file documented as a contract of its own. A renderer page may set `Completable` with its `Heading`, further `Headings` and `Tail` (C15). An extractor's `Data.Inputs` lists, per page, every file a page was built from; a standalone page's `lastmod` is then the newest of them.

**`markdown`** copies one directory of authored pages, lints them and records their git dates. `include` and `exclude` are patterns relative to `dir`, matched against each file's slash path: a `path.Match` glob (`**` is not special) or a directory ending `/`, which matches every file under it. A file is copied when it matches some `include` (or `include` is absent) and no `exclude`. A pattern that matches no file fails the build (exit `2`, "markdown dir docs/site: exclude "reference/defintions/" matches no file"), except when the config came from outside the source tree (a backfill, C5), where a missing `dir` is likewise no error. A pattern that is not a valid glob exits `1`. Link pinning (C8) applies only in a tab bundle; a docs bundle's pages are copied as written.

**`citations`** (`#Citations`) is an extractor source's citation policy ("Doc-comment rules"). The `enhancements` source takes none: like `markdown`, it publishes authored text, whose decision citations are the design record itself.

**`version`** is required except in a section bundle (C21), which builds from `main` only: there it may be left out and is ignored when given.

**`pins`** names a repository command (C14) that prints `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"<project>": "<version>"}}` and the projects it must pin; build rules in C15.

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
	// Projects that may be placed in a site version's /docs/, each with the
	// only repository allowed to sign it.
	docs: [#Project]: {
		repo: =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$"
	}
	// Unversioned sections outside every site version, each pulled from its
	// edge tag only, with the only repository allowed to sign it.
	sections: [#Project]: {
		repo: =~"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$"
		root: "/enhancements/"
	}
	versions: [#SiteVersion]: {
		// The bundle that chooses the others: its manifest's pins.
		anchor: {project: #Project, tag: #DocsTag}
		// Pulled at the exact version the anchor's manifest.json pins.
		pinned: *[] | [...#Project]
		// Pulled by their own tag, not by a pin.
		tags: [#Project]: #DocsTag
	}
}

// A site version: "v1.0".
#SiteVersion: =~"^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"

// A minor tag ("1.0"), a major tag ("4"), an exact release ("2.0.0-beta.1")
// or "edge".
#DocsTag: =~"^((0|[1-9][0-9]*)(\\.(0|[1-9][0-9]*))?|edge)$" | #SemVer
```

The file opmodel.dev writes for its tabs (its site versions, `docs` and `versions`, are C16's):

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

Everything below is about tabs; a site version's docs bundles follow C16, and a section C21, which state where they differ.

- Default: list each tab's tags (C4), resolve, verify (C9), fetch the layer **by digest** after verification, unpack, lint, write the lock.
- `--frozen <lock>`: pull exactly the digests in that lock, no tag resolution; verification and lint still run. The recovery path, as `frozen.conf` is for site versions.
- `--offline`: no network; requires `--frozen` and every blob in the cache. Fails naming the first missing digest.
- `--local <project>@<segment>=<dir>`, repeatable: take that segment of that project from a local bundle tree (an `opm-docs build` output or a test fixture) instead of the registry, e.g. `--local catalog-opm@4.4=fixtures/catalog-opm/4.4 --local catalog-opm@4.5=... --local catalog-opm@edge=...`. The segment must equal the one the tree's `manifest.json` implies (`MAJOR.MINOR` of its version, or `edge`), and the project must be a tab in the config (a segment `v<MAJOR>.<MINOR>` names a site version instead, C16). A local entry is unsigned: `pull` skips signature verification and never fetches the Sigstore trusted root for it, but still validates the manifest, applies the unpack guards and lints it in bundle mode. When `--local` names a project, registry resolution is skipped for that whole project, so a pull whose every tab is local needs no network at all. Each local entry is marked `"local": true` in the lock. For an author's preview and the site's tests; a publishing CI build never passes it.
- Cache: blobs under `$XDG_CACHE_HOME/opm-docs/blobs/sha256/<hex>` (else `~/.cache/...`), reused by digest. The Sigstore trusted root is cached beside it.

Edge cases:

- **No `edge` tag** while the tab asks for edge (a project that has not pushed `main` since adopting docs-kit): the segment is skipped with a warning naming the project; the lock has no edge entry. Not an error.
- **No minor at or above `from`:** if the tab still gets an `edge` build, the pull succeeds with a warning; if it gets nothing at all, the pull fails naming the project, since the site cannot render an empty tab.
- **Placement cross-check:** every pulled or local manifest's `placement` must be `kind: "tab"` with `root` equal to the tab's `root` in the pull config, else that bundle fails the pull naming both roots.
- **`--frozen <lock>`:** the lock's `config` digest must equal the current `bundles.cue`'s, because the trust policy (each tab's owning `repo`, the signer) comes from the config; a mismatch fails naming both digests. Tags are not resolved; each entry's `repository` and `digest` are fetched as written, and a local entry in a frozen lock is refused (re-run with `--local`). So is an entry whose `repository` is not `<registry>/<project>` of the config, or whose segment the config does not show (below `from`, or `edge` with `edge: false`): a lock can only narrow what the config trusts (added in review, 2026-10-02).
- **`--offline` and the trusted root:** offline, `pull` uses the cached trusted root as it is and never refreshes it. When the cached TUF metadata has expired it warns and still verifies, because each signature is checked against the key validity window at its own timestamp, which an expired cache does not change. With no cached root at all, `--offline` fails.
- **Compressed size:** a layer descriptor larger than 32 MiB is refused before any byte is fetched; the 64 MiB uncompressed cap applies during unpacking. `push` refuses a bundle over the same limits (32 MiB packed, 10,000 entries, 64 MiB of files) before it writes anything, so no publish produces a bundle every pull refuses (added in review, 2026-10-02).

Unpack layout, owned entirely by `pull` (it removes any project or segment directory it did not write this run; `<out>/_versions/` holds the site versions, C16, and is swept by their rules). Each bundle unpacks and lints in `<out>/<project>/.incoming-<segment>/` and replaces its segment only after both pass, so a refused bundle leaves the previous segment in place (added in review, 2026-10-02). Since C13, no segment is swapped in until every bundle, every tab's history and the lock have passed: a refusal at any of them leaves the previous trees, `history.json` files and lock as they were:

```text
site/.bundles/
  lock.json
  catalog-opm/
    history.json
    4.4/      manifest.json  content/  data/
    4.5/      ...
    edge/     ...
  _versions/  (C16)
```

`<out>/<project>/history.json` is the tab's version history (C13), written after the sweep and before the lock for every tab with two or more segments holding `data/catalog.json`, and removed from a tab with fewer. The sweep never removes it.

The segment directory is the URL segment: `<MAJOR>.<MINOR>` of the build's version, or `edge`. Unpacking refuses an absolute path, `..`, a symlink, a hard link, a device or FIFO, a duplicate path, a top-level entry other than `manifest.json`, `content/` and `data/`, more than 10,000 entries, or more than 64 MiB uncompressed; and checks every file is listed in `manifest.json` and every listed file exists.

The lock, `site/.bundles/lock.json`, validated against `schema/lock.cue`:

```cue
package schema

#Lock: {
	schema: "docs.opmodel.dev/lock/v1"
	tool:   #SemVer                   // the opm-docs that wrote the lock
	config: =~"^sha256:[0-9a-f]{64}$" // SHA-256 of the bundles.cue bytes
	bundles: [...#Locked]
	history?: [...{project: #Project, digest: =~"^sha256:[0-9a-f]{64}$", path: =~"^[a-z0-9]+(-[a-z0-9]+)*/history\\.json$"}] // path relative to the lock's directory
	docs?: [...#DocsLocked] // the bundles of each site version; left out when the config has no versions
}

#Locked: #Pulled | #Local

#Pulled: {
	project:    #Project
	root:       #LockRoot
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
	root:     #LockRoot
	segment:  #Segment
	local:    true
	version:  #Version
	revision: int & >=0
	commit:   #SHA
	dialect:  int & >=1
	builtBy:  #SemVer
	dir:      string & !=""
}

// A tab's root, or the enhancements section's.
#LockRoot: =~"^/catalogs/[a-z0-9]+(-[a-z0-9]+)*/$|^/enhancements/$"

#Segment: =~"^((0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)|edge)$"

#DocsLocked: #DocsPulled | #DocsLocal

#DocsRole: "anchor" | "pinned" | "tag"

#DocsPulled: {
	site:       #SiteVersion
	project:    #Project
	role:       #DocsRole
	tag:        #DocsTag      // the tag resolved: "1.0", "2.0.0-beta.1", "4", "edge"
	repository: string & !="" // "ghcr.io/open-platform-model/docs/cli"
	digest:     =~"^sha256:[0-9a-f]{64}$"
	version:    #Version
	revision:   int & >=0
	commit:     #SHA
	dialect:    int & >=1
	builtBy:    #SemVer
	signer: {
		workflow:   string & !=""
		repository: string & !=""
		ref:        "refs/heads/main"
	}
	pins?: [#Project]: #SemVer // the anchor only: its manifest's pins as read
	dir: string & !="" // "_versions/v1.0/cli", relative to the lock's directory
}

#DocsLocal: {
	site:     #SiteVersion
	project:  #Project
	role:     #DocsRole
	local:    true
	version:  #Version
	revision: int & >=0
	commit:   #SHA
	dialect:  int & >=1
	builtBy:  #SemVer
	pins?: [#Project]: #SemVer
	dir: string & !=""
}
```

Serialization (so two pulls of one resolution compare byte for byte): JSON, two-space indent, a trailing newline, no timestamps; entries sorted by `project`, then `segment` (minors by numeric MAJOR then MINOR, ascending, `edge` last); keys in exactly the order the schema lists them (top level `schema`, `tool`, `config`, `bundles`, then `history` and `docs` (C16) when present; a pulled entry `project`, `root`, `segment`, `tag`, `repository`, `digest`, `version`, `revision`, `commit`, `dialect`, `builtBy`, `signer` with `workflow`, `repository`, `ref`, then `dir`; a local entry `project`, `root`, `segment`, `local`, `version`, `revision`, `commit`, `dialect`, `builtBy`, `dir`). A local entry omits `tag`, `repository`, `digest` and `signer`. Example:

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

Every entry carries `root`, the tab's placement root (or the section's, `/enhancements/`, C21), so a consumer maps a `dir` to its URLs without reading `bundles.cue`.

`history` (added with C13; optional, so the schema id stays `lock/v1` and every lock without it still validates) records each `history.json` this pull wrote: `project`, `digest` (`sha256:` and the file's SHA-256) and `path` relative to the lock's directory (`catalog-opm/history.json`; the schema allows only `<project>/history.json`, so the lock sits in `--out`, as the default `<out>/lock.json` does; a pull that writes a history with the lock elsewhere exits 1 before anything is swapped in), one entry per project sorted by project, keys in that order. The key is left out when no history file was written. The site checks the file it mounts against it. `--frozen` does not compare the frozen lock's history digest, since history is a function of the trees and the tool and the tool may have moved; it records the file it wrote. `#Lock` is closed, so an `opm-docs` older than 0.3.0 refuses a lock that carries `history` under `--frozen`: the site bumps its `opm-docs` and receives its first lock with `history` together.

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
| any page | the enhancements index or graph | `/enhancements/` or `/enhancements/graph/` |
| a section page | a page of the same section | `/enhancements/...` as above; must name a page of the bundle (C21) |
| a docs page (any repository's `docs/site/`) | a catalog | `/catalogs/<name>/<MAJOR>/` plus an optional page path; never a minor or `edge` |

An authored page in a bundle (a `markdown` source) links into its own catalog through the major alias, `/catalogs/<name>/<MAJOR>/<path>/`, the form that also reads correctly on GitHub and in docs mode. When `<MAJOR>` is the build's major (or the build is edge), the `markdown` source rewrites that link to the build's own segment, `/catalogs/<name>/<segment>/<path>/`, before bundle-mode lint checks that it names a page of the bundle. A link inside a fenced code block is an example and is left as written (added in review, 2026-10-02).

A section bundle's `content/<path>` publishes at `<root><page URL>`, with no segment and outside every site version (C21): `content/0025/decisions.md` is `/enhancements/0025/decisions/`, `content/graph.md` is `/enhancements/graph/`.

A docs bundle's `content/<path>` publishes at `/docs/<page URL>` under the site version that pulls it (C15): `content/reference/cli/opm-module.md` is `/docs/reference/cli/opm-module/` in every site version that holds it. Its segment (`MAJOR.MINOR` or `edge`) is not part of any URL. Its pages link other docs pages as `/docs/<section>/<page>/` and catalogs as a docs page does (the bare root or a major), and no link of a docs bundle is rewritten.

**Edit and source links** (DESIGN decision 19). What the site links from a bundle's page, from `manifest.json` (C3); `<repo>` is `source.repo`, `<commit>` is `source.commit`:

| Page | Edit this page | View source |
|---|---|---|
| a tab bundle's page (`/catalogs/...`) | none: a fix lands on `main` and reaches a released minor by a docs revision | `https://github.com/<repo>/blob/<commit>/<source>` when `source` is set |
| a docs bundle's authored page (completed pages included) | `https://github.com/<repo>/edit/main/<edit>` when `edit` is set, whichever version the page shows; none otherwise | as above |
| a docs bundle's generated page | none | as above, when `source` is set |
| a section page (`/enhancements/`) | none: an entry changes through its own review | as above |

"Last updated" is the page's `lastmod` (C3), the source file's last commit at the commit built, or in a docs revision the newest patch that touched it; a page without `lastmod` shows none.

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

`mark` is `"not-implemented"`, `"provided-by-platform"` or `null`. `demand` is `"required"` or `"optional"`. `spec.fields` is the structured spec, read from the evaluated value: every field under the spec key, depth first, regular, optional and required alike (`presence` is `"regular"`, `"optional"` or `"required"` from the selector's constraint type), with `path` dot-separated, `[]` for a list element and `[string]` for a pattern constraint; `type` the formatted constraint; `default` the formatted default or `null`; `ref` the definition name when the field's value is a definition outside this member's package, where the walk stops instead of expanding it. The walk stops at the module boundary and at a visited definition, and caps depth at 12. `spec.cue` keeps the authored text, so a field the walk cannot express still shows on the page. Order: siblings appear in the order CUE's `Fields` iterator yields them for the evaluated value (declaration order), each field followed by its descendants, so the list is stable for one source and tool version. `type` is the field's constraint expression printed with `cue/format` (`format.Simplify()`), collapsed to one line with single spaces; `default` likewise. Both are stable for one tool version, and the version history (C13) compares them only between bundles built by the same docs-kit minor. That makes a release rule: a change that can alter these strings, a `cuelang.org/go` bump above all (its formatter and simplifier write them), or any other dependency or code change that affects formatting, releases as a minor (`feat:`), never a patch, so a patch release never puts two different spellings of one constraint into one `full` comparison.

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
- Shortcodes, checked on every line, code fences included: only `{{< opm/<figure> >}}` with one of the nine figure names, no parameters and no closing tag; `{{</* ... */>}}` is a shown shortcode and passes.
- No `:::` line, no `import ... from` line, no component tag line, no image (`![`, `<img`), no raw `href=` or `src=`.
- Every code fence carries a language tag.
- An alert marker is exactly `> [!NOTE]` (or `TIP`, `IMPORTANT`, `WARNING`, `CAUTION`) alone on its line.
- Every link destination (inline and reference definitions) is `http:`, `https:`, `mailto:` or `#...`, or one of: `/docs/(<seg>/)*` with an optional fragment; `/enhancements/`, `/enhancements/graph/` (added by `add-enhancements-bundle`, with its fixture `link-enhancements-graph`), `/enhancements/<NNNN>/` or `/enhancements/<NNNN>/<document>/` with `<document>` one of `problem`, `design`, `decisions`, `graduation`, `risks`, `operational`, `questions`; `/catalogs/<name>/` or `/catalogs/<name>/<segment>/(<seg>/)*` with `<segment>` a major (`4`), a minor (`4.4`) or `edge`, with an optional fragment.
- Bundle mode (`lint --bundle <dir>`, run by `build` and `pull`) adds: a link into the bundle's own root uses the bundle's own segment and names a page in the bundle; and `manifest.json` lists exactly the pages present; and a link to another catalog uses its bare root or a major segment, as C8's table requires (implementation decision, accepted 2026-10-02). A docs bundle's bundle mode is C15's instead: the docs-mode `/catalogs/` rules, and `/docs/` links into its owned paths name a page of it. A section bundle's is C21's: the docs-mode `/catalogs/` rules, and a link into `/enhancements/` names a page of it.
- Docs mode (the default) adds: a `/catalogs/` link is the bare tab root (`/catalogs/opm/`) or uses a major segment (`/catalogs/opm/4/...`); a minor or `edge` segment is a violation. A malformed link is reported by its second segment, as the shell lint does: a minor (`/catalogs/opm/4.4/traits/backup`) or `edge` (`/catalogs/opm/edge`) gets `docs pages link catalogs through /catalogs/opm/4/` (or `.../<MAJOR>/`) even without its trailing slash; anything else, the slashless root `/catalogs/opm` included, gets the trailing-slash message. Bundle mode gives every malformed link the trailing-slash message.

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
- **The site bumps first** (since `generalize-build-assembly`). `#Manifest` is closed, so an `opm-docs` older than a manifest field refuses a bundle that carries it (`placement.owns` and `pins` came with `generalize-build-assembly`, `pages[].edit` with `add-authored-docs`; later changes add more). `#Pull` and `#Lock` are closed too: the site writes `docs` and `versions` (C16) into `bundles.cue` only once its pinned `opm-docs` reads them. **opmodel.dev's pinned `opm-docs` is never older than any producer's `.opm-docs-version`.** A docs-kit release reaches the site first (its `site/Dockerfile` bump); only then may a producer move its `.opm-docs-version` and `publish.yml@` ref to it. `docs/orchestration.md` orders every bump that way.
- **`cobradump`, the second component.** The nested Go module `github.com/open-platform-model/docs-kit/cobradump` (C19) releases on its own through release-please, from its own release PR (`separate-pull-requests`), tagged `cobradump/vX.Y.Z` (from `0.1.0`), the form the Go module proxy needs for a nested module. Its release is published at once and carries no assets: a CLI requires it by module version, and the proxy serves it from the tag. A commit under `cobradump/` releases only `cobradump` (the root package excludes that path) and takes the commit scope `cobradump`; goreleaser runs only for an `opm-docs` release, and `release.yml` marks a `cobradump` release not Latest (`gh release edit --latest=false`), so the Latest release is always `opm-docs`'. A `cobradump/` tag never matches the signer glob `refs/tags/v[0-9]*` (C9), so it can never act as a trusted `publish.yml` ref. The tag rulesets cover `cobradump/v*` as they cover `v*`.
- **Verification.** Download the archive and `checksums.txt` for the host's os and arch, check with `grep ' <archive>$' checksums.txt | sha256sum -c -` (refusing an archive with no line), then extract only `opm-docs`. A failed check stops the task; nothing falls back to building from source. The install target is a gitignored repo-local directory (`.bin/` or `site/.bin/`), never a global path.

## C13. Version history, `history.json`

`opm-docs pull` writes `<out>/<project>/history.json` for every tab project with two or more segments holding `data/catalog.json` (C10, schema `docs.opmodel.dev/data/cue-catalog/v1`), in every mode (registry, `--frozen`, `--offline`, `--local`; a `--local` segment counts like a pulled one), and removes it from a tab with fewer. A segment whose `manifest.json` lists no such data file takes no part. The history is recomputed on every run from the trees just unpacked, never cached across digests, since a docs revision changes a segment's data after the fact. Its SHA-256 is recorded in the lock's `history` (C7). Docs-placed bundles get no history: they have no segments of their own.

**Order.** Only segments that take part count: those carrying catalog data (`data/catalog.json`). They are ordered minors ascending (numeric MAJOR, then MINOR), `edge` last. The **previous** segment of a minor is the next lower minor that takes part; the previous of `edge` is the newest minor that takes part. The **floor** is the oldest minor that takes part.

**Comparison mode per pair.** A pair (previous, current) is compared in mode `full` when both bundles' `manifest.json` `tool` share MAJOR.MINOR, else in mode `paths`, because `type`, `default` and `ref` strings are stable only within one docs-kit minor (C10). In `paths` mode only field paths and `presence` are compared. Each pair and its mode are listed in `compared`, so the site can word a weaker comparison.

**Member changes.** Members match by `fqn` (it includes the apiVersion). For a member in both segments of a pair, its `spec.fields` are matched by `path`, and these changes are recorded, in this order per path (paths in the current segment's field order, then paths only the previous segment has, in its order):

| `op` | When | `from` / `to` | Mode |
|---|---|---|---|
| `removed` | path in previous only | previous `type` / `null` | both |
| `added` | path in current only | `null` / current `type` | both |
| `presence` | `presence` differs | the two presences, each `regular`, `optional` or `required` | both |
| `type` | `type` differs | the two types | `full` |
| `default` | `default` differs (either may be `null`) | the two defaults | `full` |
| `ref` | `ref` differs (either may be `null`) | the two refs | `full` |
| `spec` | none of the above for the member, and the spec blocks' CUE tokens differ with comments skipped | `null` / `null`, `path` `""` | `full` |

The site words a `presence` change by its target, and for a field becoming regular by where it came from:

| `from` → `to` | Wording |
|---|---|
| `optional` or `regular` → `required` | "made required" |
| `required` or `regular` → `optional` | "made optional" |
| `optional` → `regular` | "no longer optional" |
| `required` → `regular` | "no longer required" |

A field's `doc` is never compared: a doc-comment fix (a docs revision) must not read as a change. The token comparison is the one `revise` uses for `.cue` files ("Docs revisions", step 5): `cue/scanner` with comments skipped, an inserted comma equal to a written one. So a `matchN` or `if`-guard change the field walk cannot express still shows as one `spec` change. A change behind a `ref` (a shared schema) is not reported, since the walk stops at `ref`; the badge then under-claims, the safe direction. `level`, `appliesTo`, `servedBy`, `mark`, `optional` and `fulfilment` are not tracked.

A member present in the previous segment and absent from the current one is listed under `removed[<current>]` with the last segment that had it and its page there. A member that is removed and later returns keeps its original `first`; the segment it returns in records no change for it. A spec block that does not scan as CUE (the extractor never writes one) counts as changed. `lineage` maps each `<kind>/<name>` to, per segment, the apiVersions present there, newest first in the order C8 fixes for page paths.

**Input checks.** Before comparing, `pull` checks every member of each segment's doc model: an `fqn`, a `kind` of `resource`, `trait` or `blueprint`, a `page` matching `^(resources|traits|blueprints)/[a-z0-9]+(-[a-z0-9]+)*$` (C8), and every field's `presence` one of the three values. A bundle that fails is the bundle's fault: `pull` exits 2 naming the segment and its data file (`history for catalog-opm: catalog-opm 4.6 data/catalog.json: member <fqn>: page "../x" is not <kind>s/<name> (C8)`), and nothing is swapped in (C7).

`schema/history.cue`, embedded in the tool, validates the file before `pull` writes it. Since the input checks ran first, a file that fails is a tool bug, and `pull` exits 2 naming the project (`history for catalog-opm does not validate: <error>; report it against opm-docs`):

```cue
package schema

#History: {
	schema:  "docs.opmodel.dev/history/v1"
	project: #Project
	tool:    #SemVer // the opm-docs that computed it
	floor:   =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"
	segments: [#Segment, #Segment, ...#Segment] // ordered: minors ascending, edge last
	compared: [...{from: #Segment, to: #Segment, mode: "full" | "paths"}]
	members: [string]:                           #MemberHistory // keyed by FQN
	removed: [#Segment]: [#Removed, ...#Removed] // a segment key only when something was removed in it
	lineage: [string]: [#Segment]: [string, ...string] // "<kind>/<name>": segment: apiVersions, newest first (C8 order)
}

#MemberHistory: {
	kind:         "resource" | "trait" | "blueprint"
	name:         string
	apiVersion:   string
	first:        #Segment // the first segment that has it
	firstIsFloor: bool     // first == floor: the site writes "in <floor> or earlier", never "added in"
	in: [#Segment, ...#Segment]
	changes: [#Segment]: [#Change, ...#Change] // a segment key only when the member changed against its previous segment
}

#Change: {
	op:   "added" | "removed" | "presence" | "type" | "default" | "ref" | "spec"
	path: string // a spec.fields path; "" for op "spec"
	from: string | null
	to:   string | null
	if op == "presence" {
		from: #Presence
		to:   #Presence
	}
}

#Presence: "regular" | "optional" | "required"

#Removed: {
	fqn:        string
	kind:       "resource" | "trait" | "blueprint"
	name:       string
	apiVersion: string
	lastIn:     #Segment                                                   // the last segment that had it: the site links <root><lastIn>/<page>/
	page:       =~"^(resources|traits|blueprints)/[a-z0-9]+(-[a-z0-9]+)*$" // its page in lastIn, "traits/backup-v1alpha1"
}
```

`#Segment` is the lock's (C7); `#Project` and `#SemVer` are the manifest's (C3).

Serialization: JSON, two-space indent, a trailing newline, struct keys in the order above, map keys sorted (byte order), no timestamps. The same pulled trees and tool write the same bytes. Example for `backup`, first in 4.5 (the floor) and given a required field in 4.6:

```json
"opmodel.dev/catalogs/opm/traits/backup@v1alpha1": {
  "kind": "trait", "name": "backup", "apiVersion": "v1alpha1",
  "first": "4.5", "firstIsFloor": true, "in": ["4.5", "4.6", "edge"],
  "changes": {"4.6": [{"op": "presence", "path": "retention.daily", "from": "optional", "to": "required"}]}
}
```

What the site derives, so both sides agree:

| Badge or list | From |
|---|---|
| "Added in X" | `first` is a minor and `firstIsFloor` is false |
| "In <floor> or earlier" | `firstIsFloor` is true; never "added in" |
| "Unreleased" | `first` is `edge` |
| "Changed in X" | every key of the member's `changes` |
| "Removed in X" | `removed[X]`, on X's kind index, linking `<root><lastIn>/<page>/` |
| "Newer version" | `lineage` of the page's own segment |
| "Changes in X" list | the entries of `changes[X]`, one per field, at the page's end, never inline in the spec block (a code fence the bundle publishes as built) |

No side-by-side diff (DESIGN decision 12).

## C14. Repository commands

A config value of type `#Command` (C6) names a program of the repository: an argv list, `argv[0]` looked up on `PATH`, no shell and no globbing. The `pins` command (C15) and the extractors that need repository code use it.

| Aspect | Rule |
|---|---|
| Runs in | `build`, `check`, `revise` and `serve` (which builds through `build`) only; never `push`, `promote` or `pull` |
| Directory | the source tree (`--source`; a revision's patched worktree) |
| Environment | an allowlist of the build's own: `PATH`, `HOME`, `TMPDIR`, `USER`, `LANG`, `LC_*`, `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` (either case), Go's variables (`GO` then capitals and digits only: `GOPATH`, `GOFLAGS`, `GOPROXY`, `GOTOOLCHAIN`; never `GOOGLE_*`), `CGO_*`, `CUE_*`, `OPM_DOCS*`; plus `OPM_DOCS=1`, `OPM_DOCS_PROJECT=<project>`, `OPM_DOCS_VERSION=<version or edge>`. Nothing else passes: no `GITHUB_TOKEN`, no `ACTIONS_*`, no `DOCKER_CONFIG` |
| Credentials | none: besides the environment rule, a command never runs where a registry or git credential is on disk (in `publish.yml` the build job persists no checkout credential and never logs in, C5) |
| stdin | empty |
| stdout | exactly one JSON document, at most 16 MiB, whose `schema` field names a schema its consumer reads; anything after the document is an error |
| stderr | passed through to the build's stderr |
| Timeout | 10 minutes; then the command's whole process group is killed (it runs in a group of its own on Unix, so a program `go run` compiled dies with it); a canceled build (an interrupt) kills it the same way and names the cause |
| Failure | a non-zero exit, a timeout, an output over the cap, trailing output or an unknown schema exits `2` naming the project and the argv (``cli: command `go run ./hack/docskit-dump`: exited with status 1``) |
| Determinism | `check` runs every command twice and exits `2`, naming the argv, when the outputs differ ("two runs printed different output; a docs build must be deterministic"); `build` and `revise` run it once |

In `publish.yml` a command runs in the build job, which holds no signing credential (C5); the `setup-go` input installs Go for a command that needs it.

## C15. Docs placement

A bundle with `placement: {kind: "docs", root: "/docs/", owns: [...]}` merges into a site version's `/docs/` tree; its segment means nothing for URLs (C8), and `--release` derives its version from the tag prefix as for a tab. The rules `build` applies:

| Rule | Exit |
|---|---|
| two owned paths nest (`reference/` and `reference/cli/`), naming both | `1` |
| the bundle holds a `cue-catalog` source (a catalog is a tab, C1) | `1` |
| a page a renderer writes lies outside every owned path, a completed page included ("`docs-kit.cue: content/reference/commands/opm.md is generated, but cli owns only reference/cli/; add it to placement.owns`") | `2` |

**Bundle-mode lint of a docs bundle** (`lint --bundle`, run by `build`, `push` and `pull`): the docs-mode `/catalogs/` link rules (C11: the bare root or a major segment, "docs pages link catalogs through /catalogs/opm/4/"), plus: a `/docs/` link whose path lies under an owned path names a page of the bundle (`/docs/reference/cli/opm-module/` is `reference/cli/opm-module.md` or `reference/cli/opm-module/_index.md`); fragments are not checked. A `/docs/` link outside the owned paths points into another bundle or a site page and is not checked by the bundle: the site's post-build link check covers it. The dialect version stays `1`.

**Generated paths** (every bundle, tab or docs). Before anything is written, `build` refuses (exit `2`, naming the extractor kind and the path) a page path a renderer returns that does not match `#Page.path` (C3), a data file an extractor returns that does not match `#DataFile.path`, a page path rendered twice (by two renderers, or by one twice; two completable pages at one path name both extractor kinds), and a data file written by two extractors. Neither pattern admits `..`, an absolute path or upper case.

**Completable pages.** A renderer may mark a page completable, with the heading its generated body opens with. When a `markdown` source of the same bundle supplies a page at that path, the bundle's page is the authored front matter and body, unchanged, then one blank line, then the generated body without front matter, recorded as `generated: false` with the authored file as `source`. An authored body that already holds the heading, or another heading the renderer lists with it (the `crd` page lists every kind's), exits `2` naming the file. Without an authored page, the generated page stands alone with its own front matter, `generated: true`. The catalog landing is the first completable page (heading `## Catalog members`; output unchanged).

**Authored docs** (`add-authored-docs`, DESIGN decisions 19 to 21). A repository has one docs-placed bundle (C1's naming rule). Its authored `docs/site/` is a `markdown` source of that bundle, beside any extractor source: core, cli, library and opm-operator ship both from their adoption on (DESIGN decision 20); catalog_opm's `docs/site/` and opm's follow as their own docs bundles. The rules:

- The `markdown` source copies its `dir` (all of it, or what `include` and `exclude` select, C6) into `content/` as written: no link is rewritten, and docs-mode link rules apply in bundle-mode lint (above). Each page is `generated: false` with its `source` and `lastmod`. Figure shortcodes pass as in dialect `1` (C11).
- It may supply a root `_index.md` and section `_index.md` pages. Each section page under `/docs/` has one owner across a site version (opm owns `_index.md` and `start/_index.md`); that uniqueness is the site version's `pull` check (`pull-docs-placement`), not `build`'s.
- A page path written both by an extractor and by the `markdown` source fails the build (exit `2`, "content/reference/definitions/components.md is written by both markdown docs/site and cue-definitions"), unless the extractor's page is completable, when the authored page completes it (above). Committed generated pages are therefore excluded (`exclude: ["reference/definitions/"]`) until the site reads the bundle, then deleted.
- **The main tree.** `build` writes `pages[].edit` (C3) from the `HEAD` of the main tree, matching paths from the repository root: the current directory when it is a git work tree of the repository built (the same `owner/name`). That check relies on the workflow's layout: `publish.yml` runs every publishing mode in the caller's checkout of `main`, the release tree beside it at `src/` (C5), and in CI the name is `GITHUB_REPOSITORY`. Otherwise an edge build uses its source tree; a release build writes no `edit` and prints a note on stderr ("the current directory is not a checkout of <repo>, so the pages of <tag> get no edit path; run build from the repository's checkout of main"), never reading the tag's tree as `main`. `revise` passes its checkout of `main`, so a revision's pages link the files `main` has, a page the fix added included; a rebuilt revision keeps the `edit` it recorded ("Docs revisions", step 3).
- **Backfills** (C5, C6). A release built with a config from outside the source tree (a backfill: the tag has no `docs-kit.cue`, so `main`'s config meets the tag's sources). Then, and only then, a missing `markdown` `dir` yields no pages and an `include` or `exclude` pattern matching nothing is ignored; in every other build each fails with exit `2`. `edit` still comes from `main`, so a backfilled page whose file `main` has since moved has no Edit link.

The configurations, as the phase-2 and phase-3 changes write them:

```cue
// core, after its committed reference is deleted (until then the markdown
// source carries exclude: ["reference/definitions/"]).
bundles: core: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/definitions/"]}
	version: {from: "tag", prefix: "v"}
	sources: [
		{kind: "cue-definitions" /* options: the cue-definitions extractor's contract */},
		{kind: "markdown", dir: "docs/site"},
	]
}

// catalog_opm: the tab stays; its docs/site becomes a second, docs-placed project.
bundles: {
	"catalog-opm": {placement: {kind: "tab", root: "/catalogs/opm/"}, version: {from: "tag", prefix: "opm-v"}, sources: [/* unchanged */]}
	"catalog-opm-docs": {placement: {kind: "docs", root: "/docs/"}, version: {from: "tag", prefix: "opm-v"}, sources: [{kind: "markdown", dir: "docs/site"}]}
}

// opm: authored pages only, released by release-please with tags v<semver>
// (DESIGN decision 17), from 1.0.0-beta.1 (DESIGN decision 21).
bundles: opm: {
	placement: {kind: "docs", root: "/docs/"}
	version: {from: "tag", prefix: "v"}
	sources: [{kind: "markdown", dir: "docs/site"}]
}
```

`catalog-opm` and `catalog-opm-docs` publish from the same release tag (`opm-v4.6.0`): catalog_opm's release job calls `publish.yml` once per project. A site version names them separately: the tab through `tabs`, the docs through `versions."v1.0".tags."catalog-opm-docs"` (a major, `"4"`). `catalog-opm-docs` starts at the first opm release cut after catalog_opm deleted its committed reference pages (catalog_opm #127); every earlier tag still holds them under `docs/site/reference/`, and its `docs-kit.cue` names no `catalog-opm-docs` (C5 reads the tree's config first), so none is backfilled. opm's site-version tag is `tags.opm`, `"1.0"` for v1.0.

**Pins** (DESIGN decision 10). When a bundle's config has `pins: {command, projects}`, `build` runs the command (C14) and requires a `docs.opmodel.dev/pins/v1` document whose `pins` keys are exactly `projects` and whose values are SemVer versions without `v` (C3's `#SemVer`); it writes them to `manifest.json` `pins`. A missing or extra project, or a malformed version, exits `2` naming it and the command. A backfilled release (config from outside the tree) whose tree cannot run the command fails: pins are a contract, never guessed.

## C16. Site versions

A site version (`v1.0`) shows, in its `/docs/` tree, the docs bundles (C15) of the cli release it follows and of the exact versions that release pins (DESIGN decisions 9 and 10). `opm-docs pull` resolves them from `bundles.cue` (C7's `#Pull`, keys `docs` and `versions`); the site never works out a pin itself.

**Config.** `docs` names every project that may be placed in a site version's `/docs/`, each with the only repository allowed to sign it (C9's Source Repository URI). `versions` maps a site version to its `anchor` (a project and a tag), the `pinned` projects and the projects pulled by their own `tags`. A tag is a minor tag (`1.0`), a major tag (`4`), an exact release (`2.0.0-beta.1`) or `edge`. opmodel.dev's file once v1.0 reads bundles:

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

`docs` without `versions` is valid and pulls nothing for it. Phase 3 adds `opm` and `catalog-opm-docs` under `docs` and, for v1.0, `tags: {opm: "1.0", "catalog-opm-docs": "4"}`. Before any network call `pull` exits `1`, naming the version and the project, when a version names a project that is not a key of `docs`, names one project twice (in any two of `anchor`, `pinned`, `tags`), or a project is both a tab and a docs project.

**Resolution**, per site version, in version order:

1. **Anchor.** Resolve `anchor.tag` in `<registry>/<project>`, verify (C9 with `docs[project].repo`), fetch by digest, unpack, lint in bundle mode. The resolved build must be in the tag's line (C4): an edge build for `edge`, a build of that MAJOR.MINOR for a minor tag, of that MAJOR for a major tag, of exactly that version for a release tag.
2. **Pins.** Each `pinned` project resolves at the tag equal to the anchor's `manifest.json` `pins` entry for it: the release tag, which C4 moves to the newest revision of that release, so docs revisions follow with no config change. The resolved build's version must equal the pin. A pinned project the anchor does not pin exits `2` ("v1.0: cli 1.0.0-beta.6 (sha256:...) pins no version of core; ..."); a pin whose release tag does not exist exits `2` ("v1.0: cli 1.0.0-beta.6 pins core 2.0.0-beta.1, and ghcr.io/open-platform-model/docs/core has no bundle for it (or the package is not public); publish it: run core's docs workflow in release mode for its 2.0.0-beta.1 release"). A registry that denies the anonymous pull (401 or 403, GHCR's answer for a package that does not exist yet or is private) counts as a missing tag.
3. **Own tags.** Each `tags` project resolves like the anchor, at its own tag.

Every bundle of a site version must be `placement.kind: "docs"` and its own project, else exit `2` naming both. Edge has no special case: an anchor at `edge` pins exact releases like any other build. A missing anchor or `tags` tag exits `2` naming the repository and the tag.

**Layout and replacement.**

```text
<out>/
  lock.json
  catalog-opm/4.5/ ...                    tabs (C7)
  _versions/
    v1.0/
      cli/           manifest.json  content/  data/
      core/          ...
      library/       ...
      opm-operator/  ...
```

`_versions` cannot collide with a project (C1 names have no `_`). Every bundle of site version `v` unpacks and lints in `<out>/_versions/.incoming-<v>/<project>/`; the checks below run over that set; only once every site version, every tab, every history and the lock have passed is each `<v>` replaced whole (the previous tree moves aside to `.outgoing-<v>`, the new one takes its place, then the previous one is removed), so a refusal leaves the previous `<out>/_versions/<v>/` and the previous lock as they were. The sweep removes every entry of `_versions/` that this run did not write, and `_versions/` itself when the config has no `versions`.

**Checks across one site version** (exit `2`, naming the version, the path and both projects with their versions), in this order:

- two owned paths that nest, one equal to or under the other ("v1.0: cli 1.0.0 owns reference/cli/ and core 2.0.0 owns reference/; ...");
- a page under a path another bundle owns ("v1.0: core 2.0.0-beta.1 has reference/cli/x.md, under reference/cli/, which cli 1.0.0-beta.6 owns; ...");
- pages that serve one URL: one `content/` path in two bundles ("v1.0: reference/_index.md is in both cli 1.0.0-beta.6 and core 2.0.0-beta.1; ..."), or two paths Hugo serves at one URL ("v1.0: guides/cli.md in cli 1.0.0 and guides/cli/_index.md in core 2.0.0 serve one URL, /docs/guides/cli/; ...").

Every check compares by the URL Hugo serves: `.md` dropped, then a trailing `_index` or `index` (`cli.md`, `cli/_index.md` and `cli/index.md` are one page); an owned page `x.md` holds everything under `x/`, as an owned directory `x/` does.

Links across bundles are not checked here; the site's post-build link check covers them (C15).

**Lock.** `#Lock` (C7) carries the optional `docs` list, `#DocsLocked`, one entry per docs bundle: keys in the order the schema lists them (a pulled entry `site`, `project`, `role`, `tag`, `repository`, `digest`, `version`, `revision`, `commit`, `dialect`, `builtBy`, `signer` with `workflow`, `repository`, `ref`, `pins`, `dir`; a local entry `site`, `project`, `role`, `local`, `version`, `revision`, `commit`, `dialect`, `builtBy`, `pins`, `dir`), `pins` only on an anchor that has them (as its manifest holds them, keys sorted), `dir` relative to the lock's directory (`_versions/v1.0/cli`). Entries sort by site version (numeric MAJOR, then MINOR), then role (`anchor`, `pinned`, `tag`), then project. The key is left out when the config has no `versions`; the schema id stays `lock/v1`. Example entry:

```json
{
  "site": "v1.0",
  "project": "cli",
  "role": "anchor",
  "tag": "1.0",
  "repository": "ghcr.io/open-platform-model/docs/cli",
  "digest": "sha256:<64 hex>",
  "version": "1.0.0-beta.6",
  "revision": 0,
  "commit": "<40 hex>",
  "dialect": 1,
  "builtBy": "0.4.0",
  "signer": {
    "workflow": "https://github.com/open-platform-model/docs-kit/.github/workflows/publish.yml@refs/tags/v0.4.0",
    "repository": "https://github.com/open-platform-model/cli",
    "ref": "refs/heads/main"
  },
  "pins": {
    "core": "2.0.0-beta.1",
    "library": "1.0.0-beta.1",
    "opm-operator": "1.0.0-beta.4"
  },
  "dir": "_versions/v1.0/cli"
}
```

**`--local <project>@v<MAJOR>.<MINOR>=<dir>`** (a segment starting with `v` names a site version; `<MAJOR>.<MINOR>` and `edge` stay tab segments, C7): that project of that site version comes from the tree, with no signature and no Sigstore trusted root, and every other check (manifest, placement, guards, lint, the cross-bundle checks). Its version is not checked against a tag's line, so an author's edge build previews as the anchor. A local anchor's pins choose the pinned projects, each local or pulled; a local pinned tree whose version differs from its pin exits `1` naming both. A `--local` naming a site version the config lacks, a project that version does not pull, or one pair twice exits `1`. A pull whose every tab segment and every site-version project is local needs no network. Each local entry is `"local": true` in the lock.

**`--frozen <lock>`**: each configured project of each site version is fetched by the digest its `docs` entry names, verified and linted again; no tag is resolved. A lock that does not fit exits `1`: an entry whose site version, project or role is not the config's, whose `repository` is not `<registry>/<project>`, whose anchor or `tags` tag differs from the config's, a pinned entry whose version or tag differs from the locked anchor's `pins`, a local entry (re-run with `--local`), one site version and project locked twice, or a configured project the lock does not name and no `--local` supplies. These are checked before any network call. **`--offline`** with `--frozen` uses the cache only, as C7.

**Exit codes**, as C7: `1` for the config checks above, a bad `--local` and a frozen lock that does not fit; `2` for resolution, signature, placement, lint and cross-bundle failures.

**Consumers.** opmodel.dev (`pull-reference-bundles`) writes `docs` and `versions`, mounts `_versions/<v>/<project>/content` into that version's `/docs/` and reads the lock's `docs` entries and each manifest's page data. The cli's bundle carries `pins` (C15); core, library and opm-operator publish a bundle for every release the cli pins. `#Pull` and `#Lock` are closed, so an `opm-docs` older than this contract refuses a `bundles.cue` with `docs` or `versions` and a frozen lock with `docs`: the site bumps its `opm-docs` first (C12).

## C17. `cue-definitions`

A `cue-definitions` source documents a CUE package's exported definitions on reference pages grouped by the author (DESIGN.md phase 2; it replaces core's `tools/refgen`). Its config is `#CueDefinitions` (C6). core's file, the lists moved verbatim from `tools/refgen/groups.go`:

```cue
bundles: core: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/definitions/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:        "cue-definitions"
		package:     "./src"
		skip:        ["*_pins.cue"]
		section:     "reference/definitions/"
		title:       "Definitions"
		description: "Every OPM definition type, generated from the CUE schema in core."
		weight:      1
		intro:       "Every definition below belongs to `opmodel.dev/core@v2`, the CUE module every OPM artifact is typed against. ..."
		pages: [
			{file: "modules-and-instances", title: "Modules and instances", description: "...", definitions: ["#Module", "#ModuleInstance", "#InstanceIdentity"]},
			// ... the other seven pages of groups.go, in order
		]
		exclude: {"#BlueprintMap": "map shorthand" /* ... the rest of groups.go's excluded map */}
	}, {
		kind: "markdown", dir: "docs/site", exclude: ["reference/definitions/"]
	}]
}
```

**Config checks** (exit `1`, before anything is built): a `cue-definitions` source in a bundle whose placement is not `docs`; a `skip` pattern that is not a glob; a `package` that does not start `./` or holds a `..` segment.

**Extraction.** Every `.cue` file directly in `package` whose base name matches no `skip` glob (`path.Match`) is parsed with its comments; nothing is evaluated. A definition is an exported top-level field with an identifier label (`#Name`; not `_#Name`, not a quoted `"#Name"`); one declared in two files exits `2`. The module path the pages name is the `module` field of the nearest `cue.mod/module.cue` at or above `package`. Per placed definition:

| Field | From |
|---|---|
| `summary` | the doc comment's first sentence (a leading `#Name:` or `Name:` label removed, a bare leading `Name` given its `#`, the first letter capitalised; `e.g.`, `i.e.`, `etc.` and `vs.` end no sentence); a definition has no `metadata.description`, so the member summary rule does not apply |
| `notes` | the rest of the doc comment, one block per entry: a paragraph; `- item`, a list item; a line indented by two spaces, a preformatted line; `""`, a paragraph break |
| `example` | the values after `Example:` (one quoted string per line) and `Usage:` (`expr => result` becomes `expr // result`) |
| `cue` | the field formatted with four-space indents, after: hidden fields and comprehensions that only set hidden fields dropped; every comment but doc and same-line comments dropped, and so are `WHY` groups and `////` banners; a `T & {comprehensions}` value of 40 lines or more shown as `T` with "Derived in the source; the derivation is not shown here."; the definition's own doc comment removed |
| `shape` | `struct, closed`, `struct, open` (an `...`), `map` (pattern fields only), `disjunction`, `string constraint`, `unification` or `value`, plus `` `kind: "X"` `` when the struct pins a string `kind` |
| `embeds` | placed definitions embedded at its top level |
| `uses` | placed definitions it refers to: an identifier in expression position that resolves to the file's top level, or to nothing in the file (declared in another file of the package); a field label, a selector's field (`x.#B`) and a reference to a nested definition of the same name do not count. A reference to an excluded definition is followed through to what that one refers to. `usedBy` is its inverse; both sorted |
| `rules` | `by: "cue"` only: closed struct, required (`!`) fields (also inside plain structs and `if` bodies, with the condition), a field declared twice (pinned `true` beside a check, or two constraints), a string constraint's pattern (in `code`) and `strings.MinRunes`/`MaxRunes` bounds |

Prose (`summary`, `notes`) follows the source's `citations` policy ("Doc-comment rules"). A spec block's comments drop every citation under both policies; a changed doc paragraph there is re-wrapped at 76 columns and a same-line comment is never wrapped (refgen's rule, kept for parity; the 80-column rule of "Doc-comment rules" is the catalog's).

**Placement.** Every exported definition is on exactly one page or in `exclude`. With the config in the source tree, each of these exits `2`, all listed in one message: an unplaced definition ("`#Policy (src/policy.cue) is exported but neither placed in a page nor excluded in docs-kit.cue; add it to a page's definitions or to exclude with a reason`"), a definition on two pages, a placed name the package does not declare, an excluded name it does not declare, a name both placed and excluded, a placed definition whose doc comment yields no summary (missing, or holding only rationale or citations), two names on one page that differ only in case (their anchors collide). With the config from outside the tree (a backfill, C5), each is a `warning:` line on stderr: an unplaced definition is left out, an unknown name skipped, a second placement and a colliding name ignored, a page left empty dropped with its index row, and a definition without a summary kept with an empty one (no summary paragraph, an empty table cell).

**Data model**, `data/cue-definitions.json`:

```json
{
  "schema": "docs.opmodel.dev/data/cue-definitions/v1",
  "modulePath": "opmodel.dev/core@v2",
  "version": "2.0.0-beta.1",
  "section": "reference/definitions/",
  "title": "Definitions",
  "description": "Every OPM definition type, generated from the CUE schema in core.",
  "weight": 1,
  "pages": [
    {"file": "components", "title": "Components", "description": "...", "weight": 2, "definitions": ["#Component", "#ComponentNames"]}
  ],
  "definitions": [
    {
      "name": "#Component", "anchor": "component", "page": "components",
      "file": "src/component.cue",
      "summary": "<first sentence>", "notes": ["<block>"], "example": [],
      "shape": "struct, closed, `kind: \"Component\"`", "embeds": [],
      "cue": "<the formatted spec block>",
      "uses": ["#Blueprint"], "usedBy": ["#Module"],
      "rules": [{"rule": "<sentence>", "by": "cue"}, {"rule": "The string must match this regular expression:", "code": "^[a-z]+$", "by": "cue"}]
    }
  ],
  "excluded": [{"name": "#BlueprintMap", "reason": "map shorthand"}]
}
```

`weight` and `intro` are left out when unset; a rule's `code` when empty. Definitions are listed in page order, then in each page's configured order; `excluded` by name; every list is present, empty or not. A page's `weight` is its 1-based position among the pages written.

**Pages.** Every page lies under `section`, and `section` lies under one of the bundle's `owns` (C15); the renderer also refuses a target that is not a docs placement:

| Path | Page |
|---|---|
| `<section>_index.md` | front matter `title`, `description`, `weight` when set, no `type`; `intro` as configured (Markdown, written as it is), else "Every definition below belongs to `<module path>`."; `## Pages` (one line per page: link and description); `## All definitions` (a table: definition, page, summary). Completable (C15) with heading `## Pages`, so an authored index may replace the intro |
| `<section><file>.md` | front matter `title`, `description`, `type: reference`, `weight`; "Definitions on this page: ..."; per definition `## #Name`, the summary, `**At a glance**` (Source with file and module path, Shape, Embeds, Uses, Used by), `**Spec**` (a `cue` fence), `**Example**`, `**Notes**`, `**Enforcement**` ("CUE enforces each of these rules on a value unified with `#Name`:", a list, a `code` in a `text` fence under its rule) |

An anchor is the name lowercased without `#` (`#ComponentNames` is `componentnames`); a link to a definition is `/docs/<section><file>/#<anchor>`. In prose, outside code spans (a run of n backticks up to the next run of exactly n) and inline Markdown links (`[text](url)` or `[text](url "title")`, whose URL and title stay as written, so a fragment survives, and whose text has `<` and `{{` escaped), a `#Name` (or `#Name.field`) of a placed definition other than the entry's own becomes such a link, any other `#name` and `$name` a code span, and `<` and `{{` are escaped; nothing else is, since a doc comment's prose is Markdown as written. Every code fence and code span the renderer writes (spec, example, text blocks, a rule's `code`, the Source and Shape spans, rule paths and expressions) is one backtick longer than the longest backtick run in its content (a fence at least three). A link in a definition's prose whose URL names a scheme other than `http` or `https` exits `2` naming the definition and its file ("`#T (src/t.cue): the doc comment links "javascript:alert(1)"; a link is http, https or relative`"): the site renders links as written. No page carries a generator marker comment.

**Parity record.** At core `main` `c5a6076` (`v2.0.0-beta.1-15`; the tag itself predates refgen) the nine pages built with core's config equal refgen's committed pages byte for byte, marker comments removed (`TestCoreDefinitionsParity`, run with `OPM_CORE_CHECKOUT`). No planned difference. CI's `core definitions parity` job checks out core at that pinned commit and runs only that test, so parity stays checkable after core retires refgen. A backfill of `v2.0.0-beta.1` with that config builds with three warnings (definitions without a doc comment).

## C18. The `crd` source and `data/crd.json`

**Config.** `#CRD` in C6; a `crd` source belongs in a docs bundle (C15), and a tab bundle holding one exits `1`. `dir` holds controller-gen output; every `*.yaml` file in it is read, and every document in a file must be an `apiextensions.k8s.io/v1` `CustomResourceDefinition` with exactly one version that has a schema. `samples`, when set, holds kubebuilder samples. `order` puts kinds first in that order, the rest by name; `reconciledBy` maps a kind to the name of the controller that reconciles it (stated by the author, never inferred from code). `citations` is the source's citation policy ("Doc-comment rules"). Every CRD and sample file read must be a regular file (a symbolic link is refused). A `samples` directory that is set but missing exits `2`, except when the config came from outside the source tree (a backfill, C5), where it means no samples. opm-operator's file, as its change `publish-crd-bundle` writes it:

```cue
bundles: "opm-operator": {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/operator-resources.md"]}
	version: {from: "tag", prefix: "v"}
	sources: [
		{kind: "markdown", dir: "docs/site", exclude: ["reference/operator-resources.md"]},
		{
			kind:        "crd"
			dir:         "./config/crd/bases"
			samples:     "./config/samples"
			page:        "reference/operator-resources.md"
			title:       "Operator resources"
			description: "One generated entry per operator resource kind: ModuleInstance, ModulePackage, Platform and TransformerRegistration."
			weight:      7
			order: ["ModuleInstance", "ModulePackage", "Platform", "TransformerRegistration"]
			hideSamplesMatching: ["testing.opmodel.dev"]
			stripLabels: {"app.kubernetes.io/name": "opm-operator", "app.kubernetes.io/managed-by": "kustomize"}
			reconciledBy: {ModuleInstance: "moduleinstance", ModulePackage: "modulepackage", Platform: "platform", TransformerRegistration: "transformerregistration"}
			citations: "link"
		},
	]
}
```

**Samples.** For each kind, only the file kubebuilder names `<group>_<version>_<kind>.yaml` (the kind lower-cased) under `samples` is read, and in it the first document whose `apiVersion` is `<group>/<version>` and whose `kind` is the kind. Other files (`..._moduleinstance_jellyfin.yaml`) are never read. A missing file or document means no sample. A document whose text (comments included) contains a `hideSamplesMatching` string is not shown. A shown document is re-encoded without comments, keys sorted, numbers as written, each `stripLabels` label removed when its value matches exactly, and `metadata.labels` removed when that empties it.

**`data/crd.json`** (`docs.opmodel.dev/data/crd/v1`):

```json
{
  "schema": "docs.opmodel.dev/data/crd/v1",
  "citations": "link",
  "page": {"path": "reference/operator-resources.md", "title": "Operator resources", "description": "...", "weight": 7},
  "kinds": [
    {
      "kind": "ModuleInstance", "group": "opmodel.dev", "plural": "moduleinstances",
      "scope": "Namespaced", "shortNames": ["mi"], "categories": [], "subresources": ["status"],
      "versions": [{"name": "v1alpha1", "served": true, "storage": true}],
      "file": "config/crd/bases/opmodel.dev_moduleinstances.yaml",
      "summary": "ModuleInstance asks the operator to render one CUE module ...",
      "notes": ["<paragraph>"],
      "columns": [{"name": "Ready", "type": "string", "jsonPath": ".status.conditions[?(@.type=='Ready')].status", "priority": 0}],
      "spec": [{"path": "spec.module.path", "type": "string", "required": true, "description": "...", "default": null, "enum": []}],
      "status": [{"path": "status.conditions", "type": "[]Condition", "required": false, "description": "...", "default": null, "enum": []}],
      "rules": [
        {"field": "spec.module.path", "rule": "Required", "values": [], "message": "", "by": "api-server"},
        {"field": "", "rule": "CEL rule", "values": ["<expression>"], "message": "<message>", "by": "api-server"}
      ],
      "sample": {"file": "config/samples/opmodel.dev_v1alpha1_modulepackage.yaml", "yaml": "<text>"},
      "reconciledBy": "moduleinstance"
    }
  ]
}
```

- `page.weight`, `sample` and `reconciledBy` are `null` when absent; every list is present, empty when there is nothing.
- **`citations`** is the source's policy, `"strip"` or `"link"`.
- **Prose** (`summary`, `notes`, `description`, `message`) is plain source text with whitespace folded and the citation policy applied: under `"strip"` no citation is left; under `"link"` each enhancement decision citation stays as written (`0015:D3/D16`) and every other form is removed. Code spans stay in backticks. Nothing is escaped and nothing is a link: the renderer escapes the text and, under `"link"`, links each decision citation, so a link written in a description stays text. `summary` is the schema description's first sentence (the first `. ` not ending `e.g.` or `i.e.`); the rest of that paragraph and every later one are `notes`.
- **Fields** are listed depth-first, properties by name, written from `spec` or `status` with `[]` for an array item and `.<key>` for a map value. `type` is the schema type, or `[]<item type>`, `map[string]<value type>`, `integer or string`, `free-form object`, `string (date-time)`, `[]Condition` (the standard `metav1.Condition` list, whose own fields and rules are not listed), `any`. `default` is the default's JSON text.
- **Rules** follow the same walk: the object's own CEL rules and required fields, then for each value (`spec` and `status` included) `Required`, then `Must not be empty` / `At least N characters`, `At most N characters`, `Matches the pattern` (value: the pattern), `One of` (values: the enum), `At least`/`Greater than` and `At most`/`Less than` a bound, `At least`/`At most N items`, `At least`/`At most N entries`, `No duplicate items` (`uniqueItems` or a `set` list), `At most one item per` (values: the `map` list's keys), and `CEL rule` (value: the expression; `message` its message). `field` is `""` for the object itself. `by` is always `api-server`.

**Refusals** (exit `2`, naming the file). Decoding is strict: a key of the CRD or of a schema node that the extractor does not read is refused with the file, the kind and its path, the document path for a CRD key and the field path inside the schema (`config/crd/bases/x.yaml: Widget: spec.conversion: unknown field "conversion"`, `...: Widget: spec.port[].name: unknown field "dependencies"`), so no rule-bearing construct disappears silently. Accepted beside what the page shows, and ignored because they enforce no rule the page would miss: `metadata` `name`, `annotations`, `labels` and `creationTimestamp`; `names.listKind` and `names.singular`; a printer column's `format` and `description`; the CRD's `status` block; `preserveUnknownFields: false` (`true` is refused); a version's `deprecated` and `deprecationWarning`; a schema node's `x-kubernetes-map-type`, `title`, `example` and `externalDocs`; a CEL rule's `reason` and `fieldPath`. `dir` and `samples` must be directories, not symbolic links, that resolve inside the source tree (a linked parent may not lead out of it). Also refused: a kind name not matching `^[A-Z][A-Za-z0-9]*$`; a `scope` other than `Namespaced` or `Cluster`; a printer-column `type` other than `integer`, `number`, `string`, `boolean` or `date`; a top-level property other than `apiVersion`, `kind`, `metadata`, `spec` and `status`, or `apiVersion`, `kind` or `metadata` carrying anything beyond their type and a description; `multipleOf`, a `format` other than `date-time`, `int32` or `int64`, a CEL `messageExpression`, `x-kubernetes-embedded-resource` and a tuple `items` anywhere in a schema; `dir` leaves the source tree or holds no `.yaml` file or no CRD; a document that is not a CRD ("config/crd/bases/x.yaml: holds a v1 ConfigMap; a crd dir holds only apiextensions.k8s.io/v1 CustomResourceDefinitions"); YAML that does not parse ("config/samples/opmodel.dev_v1alpha1_platform.yaml: does not parse as YAML: ..."); two files defining one kind; a CRD with several versions; `allOf`, `anyOf`, `oneOf`, `not` or `nullable` anywhere in a schema (the reference cannot show them, so it refuses rather than drop a rule). controller-gen's `anyOf` for an int-or-string field is among them; its message names the field and says int-or-string is not supported yet ([docs-kit#23](https://github.com/open-platform-model/docs-kit/issues/23)); `order` or `reconciledBy` naming a kind no CRD defines, or `order` naming one twice.

**The page.** One page at `page`, completable (C15) with heading `## <first kind>`. Per kind, in model order: `## <Kind>`, the summary, `### At a glance` (`| Property | Value |`: Group, Version, Kind, Scope, Resource, then Short names, Categories, Subresources when present), the `kubectl get` columns table when the CRD declares columns, `### Spec` and `### Status` (`| Field | Type | Required | Description |`, with a `Default` column before Description when any row has a default), `### Example` (a `yaml` fence of backticks, three or one longer than the longest backtick or tilde run in the sample, so no sample line closes it) when a sample is shown, `### Notes`, `### Served by` ("The operator's `<controller>` controller watches every <Kind>.") when `reconciledBy` names one, and `### Enforcement` (`| Field | Rule | Enforced by |`, the rule's words, its values as code joined by `, `, then `; refused with: <message>`; "the object" for `""`; enforced by "API server"). A part with nothing to show is left out. The `###` headings repeat per kind, so the site gives the second and later ones anchor suffixes (`#spec-1`, `#spec-2`) that shift when the kind order changes, as crdref's page did; link to a kind's `##` heading, not to them. Prose is escaped by "Doc-comment rules" plus `~` and `#`, with `{{` escaped until none is left (`{{{` too); under `"link"` each decision citation becomes `[0015:D3](/enhancements/0015/decisions/)`, the only links the renderer writes; an unpaired backtick makes the whole text prose with the backtick escaped. An authored page that holds the heading of any kind, not only the first, exits `2` (C15). Standing alone, the page's front matter is `title`, `description`, `type: reference` and `weight` when set; `manifest.json` `pages[].source` is the first CRD's file, its `lastmod` the newest commit date of every CRD and sample file read, and it has no `edit`. Completed, all three are the authored page's (C15).

**Parity.** The page's generated body equals opm-operator `hack/crdref`'s block, byte for byte, for opm-operator `main` at `bf8d3c5` and for `v1.0.0-beta.4` (crdref run over that tag's tree), checked 2026-10-03. `TestOperatorCRDParity` runs in every test run on a copy of the operator's CRDs, samples and page at `bf8d3c5` (`internal/render/testdata/operator`), and on a live checkout with `OPM_OPERATOR_CHECKOUT=<checkout> go test ./internal/render -run TestOperatorCRDParity`. Deliberate differences, none visible for the operator: a CEL rule on `spec` or `status` itself is listed (crdref skipped it); `{{` is escaped; decoding is strict where crdref decoded strictly only the Kubernetes CRD type, and every construct crdref accepted but did not show (`multipleOf`, another `format`, `messageExpression`, `x-kubernetes-embedded-resource`, tuple `items`, `metadata` constraints, other top-level properties) is refused; the names, scope and column types above are refused; the example's fence grows past the sample's backtick and tilde runs; a decision link is built from the citation alone, never kept from the text.

**Consumers.** opm-operator (`publish-crd-bundle`): its `docs-kit.cue` as above; after the site reads the bundle it deletes crdref, the marker block and the `exclude`. opmodel.dev pulls `opm-operator` per site version (C16).

## C19. Command reference: `cobradump` and the `cobra` source

A program's cobra command tree exists only inside the program, so the CLI prints it, through the nested module `github.com/open-platform-model/docs-kit/cobradump` (released as its own component, C12), and the `cobra` source reads what it prints. The module depends on cobra and pflag only (and cobra's own dependencies); the root module never imports it and reads only its JSON.

**The Go API.**

```go
package cobradump // module github.com/open-platform-model/docs-kit/cobradump

// Options adjusts a dump. Home is written as "~" in a flag default (the rule
// below); "" leaves defaults as they are. Write fills it from
// os.UserHomeDir when it is unset and NoHome is false.
type Options struct {
	Home   string
	NoHome bool
}

// Write prints root's tree as one docs.opmodel.dev/cobradump/v1 document.
func Write(root *cobra.Command, w io.Writer, opts Options) error

// WritePins prints one docs.opmodel.dev/pins/v1 document (C15); a version
// that is not exact SemVer without "v" writes nothing and returns an error
// naming its project.
func WritePins(w io.Writer, pins map[string]string) error
```

A CLI calls it from a program of its own that the shipped binary does not contain; the cli's is `hack/docskit-dump`, run by its `docs-kit.cue` as `["go", "run", "./hack/docskit-dump"]` (the dump) and `["go", "run", "./hack/docskit-dump", "pins"]` (its pins).

**The dump, `docs.opmodel.dev/cobradump/v1`.** JSON, two-space indent, one trailing newline.

```text
{schema: "docs.opmodel.dev/cobradump/v1",
 name: root.Name(), path: root.CommandPath(), short, long (raw), example (raw), aliases: [string],
 useLine: root.UseLine(), runnable: bool,
 flags: [#Flag],           // the root's LocalNonPersistentFlags
 globalFlags: [#Flag],     // the root's persistent flags
 commands: [#Command]}
#Command: {path: "opm module apply", name: "apply", use: "apply [path]", useLine: "opm module apply [path] [flags]",
           short, long (raw), example (raw), aliases: [string], runnable: bool,
           flags: [#Flag]            // LocalFlags: its own, persistent or not
           inheritedFlags: [#Flag]   // InheritedFlags
           commands: [#Command]}
#Flag:    {name, shorthand: "" | one printable ASCII character (0x21-0x7E), type: Value.Type(), default: DefValue (home as "~"), usage}
```

`Write` first calls `root.InitDefaultCompletionCmd()` and resolves every command's inherited flags, as cobra does on `Execute`. It lists every available command (`IsAvailableCommand()`) except one named `help`, so hidden and deprecated commands, help topics and the help command are left out and the default completion command is in. A hidden or deprecated flag, the `help` flag, and in `flags` and `inheritedFlags` any flag named like a root persistent flag are left out; a deprecated shorthand is dropped. Commands and flags are sorted by name. `short`, `long`, `example` and `usage` are cobra's strings, unparsed, so a parsing fix ships with docs-kit and needs no CLI release. Every list is present, empty or not. Two runs of one binary print the same bytes; `check` enforces it (C14). A shorthand outside printable ASCII makes `Write` return an error, and `opm-docs` refuses one the same way.

**Home directory.** `Write` writes the home directory as `~` in a default only when the home is an absolute path other than the file-system root (a `HOME=/` rewrites nothing), and only where the match is followed by a path separator or ends the default: with home `/root`, `/root/x` becomes `~/x` and `/rootfs/x` stays as it is.

**Versioning.** `docs.opmodel.dev/cobradump/v1` is closed: `opm-docs` decodes it with unknown fields refused, and any field added, removed or changed is a new schema, `docs.opmodel.dev/cobradump/v2`. A `cobradump` release that prints v2 comes out only after an `opm-docs` that reads v2 has reached every producer (the site first, C12).

**The source.** A `cobra` source (C6) runs its `command` (C14) and requires one `docs.opmodel.dev/cobradump/v1` document: another schema exits `2` naming it and the one this `opm-docs` reads (``cli: command `go run ./hack/docskit-dump`: printed a document of schema "docs.opmodel.dev/cobradump/v2"; this build reads "docs.opmodel.dev/cobradump/v1"``); an unknown field, a command whose `path` is not its parent's path and its `name`, a command name that is not lower-case kebab-case, a command listed twice, a flag name with a leading `-`, a shorthand that is not one printable ASCII character or a top-level page name that is not lower-case kebab-case exits `2` naming the command. It writes `data/cobra.json`:

```text
{schema: "docs.opmodel.dev/data/cobra/v1",
 section, title, description, weight (omitted when unset), citations: "strip" | "link",   // from the config
 cli: <the dump without schema>, each top-level command adding page: "<section><root path with - for spaces>"}  // "reference/cli/opm-module"
```

**Pages.** Every page is generated (`generated: true`, no `source`) and must lie under an owned path (C15).

| Path | Page |
|---|---|
| `<section>_index.md` | front matter `title`, `description`, `weight` from the config (no `type`); the root's description; a fixed sentence saying what the section holds, naming `` `<cli> <command> --help` ``; the root's usage lines; `## Global flags` with the table Flag, Shorthand, Type, Default, Description |
| `<page>.md`, one per top-level command | front matter `title: "<path>"`, `description` its summary ending in a full stop, `type: reference`; a pointer to the global flags; then each command of the subtree, depth first in name order, under `## <path>`: summary, usage lines in a `text` fence, aliases, description, `**Flags**` (its own and inherited ones, by name), `**Examples**` in an `sh` fence, `**Subcommands**` (Command, Summary) |

The anchor of an entry is its path in lower-case kebab case (`#opm-module-apply`); a subcommand row links `<section URL><page>/#<anchor>` (`/docs/reference/cli/opm-module/#opm-module-apply` in a docs bundle). Usage lines are cobra's: a runnable command's use line, then `<path> [command]` when it has subcommands, else the path alone. A default that `--help` would not print (`""`, `false`, `[]`, `0`, `0s`, `<nil>`) is left empty, and a line break in a table cell becomes a space. The root's own `flags`, `example` and `aliases` are in the data but on no page, as cmdref shows none of them.

**Help-text rules.** `long` splits into paragraphs at blank lines. The first is prose; the base indentation is the shallowest first line of any later paragraph; a line indented beyond it is preformatted, a line at it is prose. A prose line `Examples:` (or `Example:`) starts examples, which run to the next prose line; `example` is appended after them, a blank line between. A preformatted block whose every line is `- item` or `N. item` becomes a list; any other is a fence, `sh` when it holds only commands of the CLI and comments, `text` otherwise. In prose, flags, paths, file names, placeholders, environment variables, CUE definitions and single-quoted spans become code spans (neighbours one space apart share one), and every other Markdown character is escaped. A fence is longer than any backtick run it holds, and Hugo shortcode delimiters are written in their shown form (`{{</* x */>}}`). `citations` applies to formatted prose only, never to a fence.

**Parity.** For the cli tree at commit `ebf0479b`, every page equals the page the cli's `internal/cmdref` committed under `docs/site/reference/cli/` once its two marker comments are removed (the cobra pages carry none). The dump and those pages are frozen in `internal/render/testdata/cli/`, and `TestCLICommandParity` checks them on every run. `TestCLICommandParityLive` repeats it against a live checkout named by `OPM_CLI_CHECKOUT` and skips without one, without `hack/docskit-dump` or without a committed reference.

**Backfill.** The hook must exist in the tagged tree, so a cli release tagged before it carries no cli bundle and cannot be backfilled; a docs revision of such a release is refused too (a help-text fix is a Go change, which "Docs revisions" refuses).

## C20. `go-api`

A `go-api` source documents a Go module's exported API from its doc comments, the library's first reference (DESIGN.md phase 2). Its config is `#GoAPI` (C6); a tab bundle holding one exits `1`. The library's file, as its change `publish-go-api-bundle` writes it:

```cue
bundles: library: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/go-api/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:        "go-api"
		module:      "./"
		root:        "./opm"
		packages: ["./opm/..."]
		section:     "reference/go-api/"
		title:       "Go API"
		description: "Every exported package of the OPM library, from its doc comments."
	}, {
		kind: "markdown", dir: "docs/site" // the library commits no generated pages
	}]
}
```

**Extraction.** Nothing is built or type-checked, so the build needs no Go toolchain and no module download. The import path prefix is `go.mod`'s `module` line. `packages` are patterns relative to `module`, as the go command reads them: `...` matches any string, and a trailing `/...` also matches the directory itself. A package is a directory a pattern matches that holds a `.go` file a `linux/amd64` build without cgo selects: `go/build`'s `MatchFile` (build constraints, `_GOOS` file names), then a file that imports `"C"` left out, and test files left out. The walk leaves out, with everything under them, a directory named `internal`, `testdata` or `vendor`, one whose name starts with `.` or `_`, and a nested module (a directory holding a `go.mod`). A `main` package (read from the package clause) is a command with no importable API: it is no package here, so a pattern that selects only commands matches nothing. The module's root package cannot be documented, since a page is named by its package's directory below `root`. Files are parsed with `go/parser` (comments kept) and read by `go/doc` (`doc.NewFromFiles`, exported symbols only), which groups a type's constants, variables, constructors and methods under it. An exported symbol without a doc comment keeps its declaration with an empty `doc`; no symbol is invented. `module`, `root`, `go.mod`, every Go file read and every directory a pattern selects must be regular files or directories: a symbolic link exits `2` naming it, and so does a `module` or `root` that resolves outside the source tree through a linked parent. The module path must pass `golang.org/x/mod/module.CheckPath` and every import path `CheckImportPath`: both reach pages as `pkg.go.dev` links.

**Refusals** (exit `2`): a pattern that matches no package ("`packages ./opm/nope/... matches no package under ./`"; a `warning:` line in a backfill, C5), the module's root package, a package at `root` itself (it has no page name) or outside `root`, an invalid module or import path, a page name that is not lower-case kebab-case or that two packages share, two package names in one directory, a file that does not parse, and a link in a doc comment whose URL is not `http` or `https` ("`package ./opm/kernel: the doc comment links "file:///etc/passwd"; a link is http or https`"; the comment parser makes links of absolute URLs only).

**Doc comments.** Rationale lines (`WHY`) are dropped, and so is every control character but tab and newline. The comment is parsed by `go/doc/comment`, its doc links resolved by the package's imports, then by the one selected package of that name, then by the standard library, and printed as Markdown by docs-kit's printer: a paragraph is one line, its citations handled by the source's policy ("Doc-comment rules"), then escaped outside code spans and links (``\ ` * _ [ ] < > | ! { }`` backslash-escaped, so no text opens raw HTML, a Hugo shortcode or a goldmark attribute block such as `{.class}` or `{#id}`, and a line that would open a block, starting `#`, `+`, `-`, `=`, `~` or `N.`/`N)`, escaped at its start); a backtick code span (a run of n backticks up to the next run of exactly n, not three or more at a line's start) is kept, its fence one longer than its longest backtick run; a link's text is escaped and its URL, a doc link's included, percent-encodes what would end it or open markup (space, `(`, `)`, `<`, `>`, `` ` ``, `{`, `}`, `\`, `"`); a heading is written at its page level (below), escapes `#` too and keeps no code span (its backticks are escaped), so nothing in it reads as heading attributes; a list keeps its items (`- ` or `N. `, a blank line between items when the source has one); a code block is a `text` fence one backtick longer than its longest backtick run (at least three). Go doc syntax reads ``` `` ``` and `''` as curly quotes, so a doc comment holds no code span of two backticks. A declaration is printed by `go/format` without its doc comment, keeping its fields' and specs' comments (a `WHY` group or `////` banner dropped, citations removed and the comment re-wrapped as "Doc-comment rules" say for a spec block) and gofmt's `// contains filtered or unexported fields`. The `synopsis` is `go/doc`'s first sentence of the package doc, plain text, every citation removed under both policies.

**Doc links.** A link to a package documented in the bundle links its page, `/docs/<section><page>/`; to a symbol documented in the bundle, `/docs/<section><page>/#<anchor>`, where a name of a constant or variable group takes its group's anchor and a field (`[T.Field]`) its type's; anything else `https://pkg.go.dev/<import path>`, plus `#<Name>` or `#<Recv>.<Name>`. A link into a bundled package that names nothing documented there links `pkg.go.dev` and prints a `warning:` line naming the comment.

**Data model**, `data/go-api.json`:

```json
{
  "schema": "docs.opmodel.dev/data/go-api/v1",
  "modulePath": "github.com/open-platform-model/library",
  "version": "1.0.0-beta.1",
  "section": "reference/go-api/", "title": "Go API", "description": "...", "weight": null,
  "packages": [
    {
      "importPath": "github.com/open-platform-model/library/opm/kernel",
      "name": "kernel", "page": "kernel", "synopsis": "<first sentence>",
      "doc": "<markdown>",
      "consts": [#Value], "vars": [#Value],
      "funcs":  [#Func],
      "types":  [{"name": "Kernel", "anchor": "kernel", "decl": "...", "doc": "...",
                  "consts": [#Value], "vars": [#Value], "funcs": [#Func], "methods": [#Func]}],
      "files": ["opm/kernel/doc.go", "opm/kernel/kernel.go"]
    }
  ]
}
```

with `#Value: {names: [string], anchor, decl, doc}` and `#Func: {name, recv: string | null, anchor, decl, doc}`; `recv` is a method's receiver as declared (`"*Kernel"`), `null` for a function or constructor. Packages sort by import path, symbols in `go/doc` order (by name); `files` are repo-relative, by name. `weight` is `null` when unset; every list is present, empty or not.

**Pages.** Every page lies under `section`, which lies under one of the bundle's `owns` (C15); the renderer also refuses a target that is not a docs placement. Every page is generated: no `edit`.

| Path | Page |
|---|---|
| `<section>_index.md` | front matter `title`, `description`, `weight` when set, no `type`; "Every package below belongs to the Go module `<module path>`."; `## Packages`, one line per package: a link titled by the import path relative to the module, then its escaped synopsis. Completable (C15) with heading `## Packages`; no `source` |
| `<section><page>.md` | front matter `title` the import path relative to the module (`opm/kernel`), `description` the synopsis (or "Package <name>." when empty), `type: reference`; the import line in a `go` fence; the package doc, its headings from `###`; then `## Constants`, `## Variables`, `## Functions` and `## Types`, each left out when empty. A constant or variable group (headed by its first name), a function and a type are `###` entries; a type's constants, variables, constructors (`#### New`) and methods (`#### Kernel.Render`) follow it as `####` entries. An entry is its heading, its declaration in a `go` fence one longer than its longest backtick run, then its doc, whose headings start one level below the entry's. `manifest.json` `pages[].source` is the package's first file by name, its `lastmod` the newest commit date of the package's files |

`<page>` is the package directory relative to `root` with `/` as `-` (`helper/objectset` is `helper-objectset`). An anchor is Hugo's default (`github`) anchor of the heading's text: lower case, a space or `-` as `-`, every other character but a letter, a digit or `_` dropped (`Kernel.Render` is `kernelrender`). The extractor assigns anchors walking the page's headings in the order the renderer writes them, the docs' own headings included, and makes a repeat unique as Hugo does, `-1`, then `-2`: a function `WidgetSpin` after a method `Widget.Spin` would collide, so the second heading on the page gets `widgetspin-1`. A rendered page that still opens a shortcode (`{{<` or `{{%` in a declaration, a code block or the synopsis in front matter) exits `2`.

**Parity record.** No generator existed, so there is nothing to equal. At library `v1.0.0-beta.1` (`02344e5`) the config above builds 9 package pages and the index, lint-clean; go.mod and the public Go files of that tag are frozen in `internal/render/testdata/library` with the pages they render, and `TestLibraryGoAPI` checks both on every run, with every doc link naming an anchor of its page; `TestGoAPIHeadingIDsMatchGoldmark` parses the golden pages with goldmark (heading attributes on, as Hugo's default) and checks that each heading gets the id the extractor assigned under Hugo's id rule, which it reimplements rather than runs; `OPM_LIBRARY_CHECKOUT=<checkout> go test ./internal/render -run TestLibraryGoAPILive` renders a live checkout, lints it and logs the exported symbols without a doc comment.

**Consumers.** library (`publish-go-api-bundle`): its `docs-kit.cue` as above, its doc comments repaired (the change's design.md lists what the trial build found at `v1.0.0-beta.1`). opmodel.dev pulls `library` per site version at the cli's pin (C16) and gains `/docs/reference/go-api/`.

## C21. Sections and the enhancements bundle

A **section** is one bundle outside every site version and outside the version switcher, built from `main` only (DESIGN decision 18). The one section is the enhancements repository's, at `/enhancements/`; the root is fixed in the schemas until a second section has a consumer. Added by the change `add-enhancements-bundle`.

**Placement and build.** `placement: {kind: "section", root: "/enhancements/"}` (C3). `build` builds a section as edge only (`version: "edge"`, revision `0`, `source.ref` the branch); `build --release --project <section>` and `revise` refuse it with exit `1` ("enhancements is a section bundle; it builds from main only (edge), so it has no release and no docs revision"); `build --release` without `--project` leaves the sections out and builds the repository's other bundles, and is refused only when the config holds sections alone. Its `docs-kit.cue` entry may leave out `version` (C6). A section bundle holds exactly one source, `enhancements`, and an `enhancements` source belongs in a section bundle only; either mistake exits `1` naming the source. Bundle-mode lint (C11): the docs-mode `/catalogs/` rules, a link into `/enhancements/` names a page of the bundle (`/enhancements/` is `_index.md`, `/enhancements/<path>/` is `<path>.md` or `<path>/_index.md`; fragments are not checked), and the markup check below. Its pages never carry `edit` (C8: an entry changes through its own review).

**The source.** `#Enhancements` (C6):

```cue
#Enhancements: {
	kind:        "enhancements"
	dir:         *"." | =~"^[^/.][^.]*$" // the entry root, repo-relative
	title:       *"Enhancements" | string & !="" // the section page's title
	description: string & !=""                   // the section page's description
}
```

The enhancements repository's file:

```cue
bundles: enhancements: {
	placement: {kind: "section", root: "/enhancements/"}
	sources: [{kind: "enhancements", description: "OPM's design record: every proposal, its decisions and its status."}]
}
```

It reads, under `dir`, every entry directory `NNNN/` and `archive/NNNN/` except `0000` (the template), plus `INDEX.md` and `GRAPH.md`. Each entry needs a `config.yaml` whose `id` equals its directory's name, a `README.md`, and exactly one `0<n>-*.md` for each `n` from 1 to 7 (no `08-*.md` or `09-*.md`). Every file read is a regular file (a symlink, file or directory, is refused), and an id both live and archived is refused. Each fault exits `2` naming the entry and the files. `config.yaml` is read as YAML; `title` and `summary` must be non-empty (whitespace collapsed), `created` and `updated` dates (`YYYY-MM-DD`), `slug`, `status`, `category` and each `affects` lower-case words, `depends_on` and `amends` four-digit ids, `supersedes`, `revives` and `superseded_by` four-digit ids or `legacy:NNN`.

**Pages**, each recorded `generated: true` with `source` (its file) and `lastmod` (that file's git date):

| Path | From | Front matter |
|---|---|---|
| `_index.md` | `INDEX.md` | `title` and `description` from the config |
| `graph.md` | `GRAPH.md` | `title: "Relationship graph"`, `description: "How the enhancements depend on and amend one another, by category."`, `type: explanation`, `weight: 1` |
| `<NNNN>/_index.md` | the entry's `README.md` | `title: "<NNNN>: <config title>"`, `description` the config `summary`, `weight` the id plus 1 |
| `<NNNN>/<slug>.md` | the entry's `0<n>-*.md` | `title: "<NNNN>: <document title>"`, `description` the document's, `type: explanation`, `weight` `n` |

An archived entry keeps its URL: `archive/0003/` publishes at `/enhancements/0003/`. The seven documents, by `n`: `problem` "Problem statement" ("What is wrong today, and for whom."), `design` "Design" ("How the proposal works."), `decisions` "Decisions" ("Each decision, numbered, with its rationale."), `graduation` "Graduation criteria" ("What has to be true before the design moves on."), `risks` "Risks and alternatives" ("What could go wrong, and what was weighed instead."), `operational` "Operational concerns" ("What it means for running and upgrading OPM."), `questions` "Open questions" ("What is still undecided."). The source builds these pages itself from the repository's text; no renderer reads `data/enhancements.json` for them (C6: the extractor's `Data.Pages`).

**Transforms**, applied to each file in order (each refusal exits `2` naming the file and line):

1. A line holding `{{<`, `{{%` or Hugo's context marker `{{__hugo_ctx`, code fences included, is refused: the site would act on it.
2. HTML comments are removed outside code fences and code spans, across lines; a line that held only comments is removed whole; an unclosed `<!--` is refused.
3. Leading blank lines are removed and, when the first line left is a `# ` heading, that line and the blank lines after it (the site prints the title).
4. Outside code fences (the dialect's fence rule), an opening fence without a language becomes `text`.
5. Every link destination outside code fences and code spans, inline or in a reference definition: a `#fragment`, a root-absolute path and an `http:`, `https:` or `mailto:` URL stay as written; another scheme is refused; a relative path is resolved against the file's directory (`%`-escapes decoded, a `?query` dropped, a fragment kept) and must name a path of the commit built (`git ls-tree -r -t`) without climbing out of the repository. The entry root and `INDEX.md` become `/enhancements/`, `GRAPH.md` `/enhancements/graph/`, an entry directory or its `README.md` `/enhancements/<NNNN>/`, a numbered document `/enhancements/<NNNN>/<slug>/`; any other path `https://github.com/<repo>/blob/<commit>/<path>` (`tree` for a directory).
6. A link whose text (one code span around it allowed) is its target file's name, when the target is a page, reads as the page's title, escaped as prose: `INDEX.md` "the index", `GRAPH.md` "the relationship graph", others the page's `title`.
7. Outside code spans and fences, a `<` that would open raw HTML (followed by a letter, `/`, `!` or `?`) is escaped `\<`; a CommonMark autolink stays (`<scheme:...>` with no blank or control character, `<name@host>` in CommonMark's email characters). The site renders raw HTML.

The transforms are best effort: they read Markdown line by line and do not hold against every construct CommonMark allows. **The guarantee is the markup check** (`internal/mdsafe`), which bundle-mode lint runs on every page of a section bundle, so `build`, `lint --bundle` and `pull` all apply it. It prepares and parses each page body as the site does. Hugo first un-indents every line that starts with its internal context marker `{{__hugo_ctx` (`hugocontext.DedentMarkers`), which can turn an indented code block into live markup; the check applies the same step before parsing, and refuses the marker itself wherever it appears, code included. It then parses with goldmark v1.8.6 (the version Hugo 0.167.0, opmodel.dev's pinned Hugo, builds with) with Hugo's default extensions, which opmodel.dev's `markup.goldmark` leaves in place (GFM tables, strikethrough, linkify, task lists, definition lists, footnotes, the typographer, heading attributes), and refuses, as `<file>:<line>: <message>` violations (exit `2`):

- raw HTML, inline or a block, comments and declarations included;
- a link, image, autolink or reference definition whose destination, decoded as goldmark's renderer decodes it (`util.URLEscape(dest, true)`: backslash escapes, then numeric, then named character references), repeated until nothing changes, and with the characters a browser drops removed, has a scheme other than `http`, `https` and `mailto`, or is protocol-relative (`//host`); a relative path passes;
- a heading attribute block (`## Title {...}`).

A goldmark bump follows opmodel.dev's Hugo bump: `mdsafe.HugoVersion` and `mdsafe.GoldmarkVersion` name the pair, a test fails when `go.mod` requires another goldmark, and a CI job renders every review probe with that Hugo and requires the check to refuse whatever it renders as live markup. Anything else the dialect refuses (component tags, images) fails the build's lint too; the enhancements repository fixes it in its sources. The lint is never relaxed for a section.

**`data/enhancements.json`** (`docs.opmodel.dev/data/enhancements/v1`), entries by id; `authors` and `history` are not copied; fields the site needs later are added, never changed:

```json
{
  "schema": "docs.opmodel.dev/data/enhancements/v1",
  "repo": "open-platform-model/enhancements",
  "entries": [
    {
      "id": "0025", "slug": "self-describing-modules", "title": "Self-Describing Modules",
      "summary": "...", "status": "draft", "category": "schema", "affects": ["core", "library"],
      "created": "2026-09-08", "updated": "2026-09-29", "archived": false,
      "dependsOn": ["0010", "0015", "0019"], "amends": [], "supersedes": [], "revives": [], "supersededBy": null,
      "page": "0025",
      "documents": [{"slug": "problem", "title": "Problem statement", "page": "0025/problem", "file": "0025/01-problem.md"}]
    }
  ]
}
```

Every list is present (empty when the config has none); `supersededBy` is a string or `null`; `page` and each document's `page` are paths under the section's root; `file` is repository-relative.

**Pull** (C7). `bundles.cue` names each section under `sections`, with the only repository allowed to sign it and its root:

```cue
sections: enhancements: {repo: "open-platform-model/enhancements", root: "/enhancements/"}
```

`pull` resolves a section's `edge` tag only, verifies it (C9 with the section's `repo`), requires placement `kind: "section"` with the configured root, unpacks and lints it into `<out>/<project>/edge/` as a tab segment, and locks it in `bundles` (`root: "/enhancements/"`, `segment` and `tag` `edge`; `#LockRoot` admits it). A section with no `edge` tag fails the pull ("enhancements has no edge build; the section would be empty, so publish main first"); so does a frozen lock with no entry for it. `--local enhancements@edge=<dir>` takes it from a local tree; any other segment exits `1`. No history is computed for a section. A project under more than one of `tabs`, `docs` and `sections`, and two sections claiming one root, exit `1` before any network call.

**Previews.** A repository with no GitHub origin, outside GitHub Actions, is named `local/<directory>` (`source.repo`), and every source link of its bundle names that made-up repository: `build` warns that the bundle is a preview, and `push` refuses any bundle whose `source.repo` starts with `local/`.

**Consumers.** The enhancements repository (enhancements#86, after the source fixes the spike of `add-enhancements-bundle` lists): `docs-kit.cue` as above, `.opm-docs-version`, and `docs.yml` calling `publish.yml` in `check` mode on pull requests and `edge` mode on push to `main`, never `release` or `revision`. opmodel.dev: bumps its `opm-docs` to a release carrying this contract first (C12), then adds `sections` to `bundles.cue`, builds the Enhancements section from `<out>/enhancements/edge/` (pages from `content/`, header data from `data/enhancements.json`), keeps an adapter only for what Hugo still needs (each page's `url` outside every version, since the dialect allows no `url` key), and copies the conformance fixtures `link-enhancements-graph` and the changed `link-enhancements` (C11).

## Site decisions

These are the opmodel.dev change's to build, recorded here so docs-kit's pull output and the sibling plan agree:

- **Sitemap:** only the newest minor of each major is listed. Older minors and `edge` are left out (and carry `noindex`).
- **Edge search:** `edge` gets its own Pagefind index, like every minor. It gets no alias stubs: `/catalogs/<name>/edge/` is its only address, and no alias ever resolves to it.
- **The preview interface** (`opm-docs serve --site <dir>`): in an opmodel.dev checkout `<dir>`, `task bundles:pull` and then `task serve`, both run with `OPM_BUNDLES_LOCAL` set to the space-separated pairs `<project>@<segment>=<dir>`: a tab (or section) bundle as `<project>@edge`, a docs bundle as `<project>@<site version>` (C16 `--local`). `serve` relies on those two task names and the variable; a site change to any of them names this contract.
- **Version history:** the data is a file written by `opm-docs pull`, at `<out>/<project>/history.json` (C13), computed from the `data/` of every segment it pulled; the site never computes history itself. What the site derives from it is listed in C13.

## Commands

Syntax `opm-docs <command> [args] [flags]`. Exit codes: `0` success, `1` usage error (unknown flag, missing argument, unreadable config), `2` execution error (lint violations, a refused build, a registry or signature failure). Every error names what failed and the fix.

| Command | Flags (type, default) | Does |
|---|---|---|
| `build` | `--config` (path, `docs-kit.cue`), `--project` (string, repeatable; default every project), `--out` (path, `out`), `--source` (path, `.`), one of `--edge` (default) or `--release <tag>` | Extract, render, lint; write `out/<project>/`. With `--release`, a `cue-catalog` source must declare the tag's version (prefix removed) as its `metadata.version`, else exit `2` naming both (a docs revision, built through `--release`, likewise). A dirty work tree is allowed for a local preview and recorded as `source.dirty: true`, which `push` refuses. Repository commands (C14) run once. A config error (an unknown source kind, owned paths that nest, a `pins.projects` that is empty) exits `1`, and so does `--release` for a section bundle (C21); a command failure, an output that does not validate, a generated page outside `owns` or a completable heading collision exits `2`. |
| `lint` | `--bundle` (bool, false), `--dialect` (int, 1) | Lint one or more page directories (or bundle directories) against the dialect. |
| `check` | `--config`, `--project` | `build` into a temporary directory, running every repository command twice (C14); exit 2 on any failure. The PR gate. |
| `push` | `--dir` (path, required), `--registry` (string, `ghcr.io/open-platform-model/docs`) | Validate, pack deterministically, push; write the full tag for a release build; print `{"digest": ..., "tag": ...}` as JSON on stdout. Refuses a bundle built from a dirty work tree or in a repository with no GitHub origin (`source.repo` `local/...`, C21). |
| `promote` | `--project` (required), `--digest` (required), `--registry` | Verify the signature of the digest (C9), then move the moving tags of its line to it (C4 rule 5). |
| `pull` | see C7; `--config` (path, `bundles.cue`), `--out` (path, `.bundles`), `--lock` (path, `<out>/lock.json`) | Resolve, verify, unpack, lint, write each tab's `history.json` (C13), each site version's docs bundles (C16) and each section's edge build (C21), lock. |
| `revise` | `--project`, `--tag`, `--fix` (required), `--out` (path, `out`), `--registry` (string, `ghcr.io/open-platform-model/docs`), `--config` (path; default `docs-kit.cue` in the release tree, else the current directory) | Build the next docs revision of a published release into `out/<project>/` ("Docs revisions" below). Pushes nothing. Exit `1` for a missing flag, an invalid config or a section project (C21), `2` when a step refuses. |
| `serve` | `--config`, `--project`, `--source` (path, `.`), `--port` (int, `1313`; refused with `--site`), `--site` (path, none; an opmodel.dev checkout with `Taskfile.yml` and `site/bundles.cue`), `--site-version` (site version `v<MAJOR>.<MINOR>`, none; with `--site`, required when a docs bundle is built) | Build the selected projects as edge bundles through `build` (dirty tree allowed; its repository commands and bundle-mode lint included) into a temporary directory removed on exit. Without `--site`: serve them with the host's `hugo` (0.146.0 or later) on a skeleton site embedded in `opm-docs`, on `127.0.0.1:<port>` only, a tab bundle at `<root>edge/`, a docs bundle at `/docs/`, any other placement at its root; poll each bundle's sources once a second and rebuild it on a change, copying a passing build over the served tree and keeping the last good one when a build fails; a page added or removed restarts `hugo`. An interrupt, SIGTERM or SIGHUP stops `hugo` and removes the temporary directory; on Linux `hugo` gets SIGTERM if `serve` dies, and a later `serve` removes a dead run's directory (one whose lock no process holds). A `--port` in use exits `1` before the build. With `--site`: build once, then run the site interface ("Site decisions"), without the caller's `OPM_BUNDLES_FROZEN` and `OPM_BUNDLES`. Exit `0` when stopped (during the first build too); `1` for a usage error, no or too old `hugo`, no `task` with `--site`, a docs bundle without `--site-version`; `2` when the first build fails or `hugo` or a site task fails. |
| `version` | none | Print `opm-docs <version>`. |

`build` also takes the hidden `--revision` (int, `0`) and `--patches` (commits, repeatable) that `revise` passes: they write `revision` and `source.patches` (C3), need `--release`, and record `source.dirty` when the work tree differs from its index (the staged fixes). Run by hand, without `revise`, a patched file keeps its `lastmod` at the release commit.

## Docs revisions

A published release's pages change only through a docs revision (DESIGN decision 6): its full tag is never overwritten, so the fix is built as the next revision of the same version. `opm-docs revise --project P --tag T --fix F` runs in the checkout of `main` (the workflow's `revision` mode) and takes these steps; each refusal exits `2` with the message shown:

1. **The fix.** `F` is a full 40-hex (SHA-1) hash of a commit with exactly one parent that is an ancestor of `origin/main` ("the fix must land on main first"; "has 2 parents ... not a merge"), and not already in `T` ("already in the release"). A SHA-256 repository is not supported.
2. **The release.** `T` carries the project's tag prefix (from the checkout's `docs-kit.cue`, or `--config`), and the registry holds revision `0` of its version `V` ("publish the release first: dispatch mode: release"); a later revision without revision `0` does not count. The next revision is `1 +` the highest published for `V` (C4 rule 3).
3. **The fixes so far.** The newest published revision `n`'s `manifest.json` (its layer unpacked with every check `pull` applies) must name `V`, `n` and `T`'s commit as `source.commit`. Its `source.patches` (empty for revision `0`) are trusted only after its signature verifies under the C9 policy, with the repository from `GITHUB_REPOSITORY` (as `promote`); an unsigned newest revision is refused ("finish the run that pushed it first"). Its `source.patches` plus `F` is the new list. `F` already in the list is refused ("already applied in `V.<n>`"), except a re-run of a run that failed after its push:
   - `F` is the last fix of revision `n`, and revision `n` is not promoted: some tag that `promote` would move to it (C4 rule 5) does not name it yet;
   - revision `n` was built by the running `opm-docs` version (its `tool`), else refused, since another version would not rebuild the pushed bytes;
   - when revision `n` is unsigned, revision `n-1` is signed and its fixes plus `F` are exactly revision `n`'s.

   Then revision `n` is built again with the same list, each page taking the `edit` revision `n` recorded (C3) rather than reading `main`, which may have moved since. The build is deterministic, so `push` finds the same digest and writes nothing, and signing and promote finish the run. Every earlier fix is checked again as in step 1.
4. **The patched tree.** `git worktree add --detach <tmp> T`, then `git cherry-pick --no-commit` of each fix in order. A conflict is refused naming the files ("land one fix on main that makes the whole change and revise with that"). After each pick the index is written as a tree, and the paths that pick changed in it (so a file main renamed after the release counts under its release-tree name) take the pick's committer date. A work tree that differs from its index afterwards is refused, and so is an `F` that leaves the tree unchanged ("changes nothing in the release tree"). The worktree is removed on every path.
5. **Documentation only.** `T`'s tree and the worktree's index (`git write-tree`) are compared with `git diff-tree -r -M`:

   | Path | Allowed |
   |---|---|
   | `*.md` | added, changed, renamed (from `.md`), removed; refused when any `.cue` file of `T` declares `@extern(embed)`, since CUE can then read a Markdown file as a value |
   | `*.cue` | changed only; both versions scanned with comments skipped give the same token sequence (a comma inserted at a line end equals a written one; interpolations resumed as the parser does) |
   | `*.go` | changed only; both versions scanned with comments skipped give the same token sequence (an inserted semicolon equals a written one); the directive comments (`//go:build`, `//go:embed`, `//line`, `//export`, any `//word:word`, `// +build`) equal in order and each at the same place (the number of code tokens before it), so a directive moved to other code is refused; never a file that imports `"C"` |
   | a symlink or submodule, at any path, before or after | refused |
   | anything else, or an added, renamed or removed `.cue` or `.go` file | refused |

   Layout is not compared: a comment added above a field moves the code below it, which a comparison of formatted output would refuse. A change to `metadata.description`, a default, a constraint or an attribute is a value change and refused. Every refusal is listed, one line per file, as `<path>: <why>; a change to code needs a patch release`.
6. **The build.** `build --release T --source <tmp> --revision <n> --patches <list>` into `--out`: `source.commit` is `T`'s commit, `source.ref` is `T`, `created` is `T`'s commit time, and a patched file's `lastmod` is the committer date of the newest fix that changed it (step 4).

`revise` pushes nothing: the workflow's `push`, `cosign sign` and `promote` follow, as for a release, so a tag never points at an unsigned build. Edge is never revised; it is rebuilt on every push to `main`. Revisions of one release share its concurrency group (C5): GitHub keeps one pending run per group and cancels an older pending one, so a revision dispatched while two others wait may be cancelled and must be dispatched again. A fix that changes code and documentation together is refused whole: split it, or ship a patch release.

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

For edge the second line reads "`<module path>` at `main` (commit `<12 hex>`), unreleased." The landing is a completable page (C15): when a `markdown` source supplies a root `_index.md`, its front matter and body come first, unchanged except for C8's alias pinning, then one blank line and the block; the authored body must not already hold a `## Catalog members` heading (the build refuses it, naming the file). The page is recorded in `manifest.json` with `generated: false` and `source` the authored file. Without an authored landing, the renderer writes front matter (`title` "<catalog name> catalog", `description` "Every member of `<module path>`, by kind.") and the block alone, `generated: true`.

No page carries a generator marker comment: a bundle's pages are wholly generated and never committed, so there is no authored text to keep apart.

**Parity with refgen (2026-10-02, tasks 3.6).** `TestCatalogOPMParity` on a catalog_opm checkout at `opm-v4.4.5` compared all 45 member pages with refgen's committed pages after normalizing the planned differences (no marker comments, links in the bundle's own segment, the contract link pointing at the landing, the newest apiVersion on the bare path). All 45 matched; no other difference was found.
