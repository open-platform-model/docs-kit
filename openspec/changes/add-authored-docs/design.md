# Design: add-authored-docs

## Context

Contracts read: C3 (`#Page`: `source`, `lastmod`, `generated`), C5 (release mode checks out `main` at the root and the tag at `src/`; revise works in a temporary worktree from the `main` checkout), C15 (docs placement, `owns`, completable pages, `include`), C16 (site versions). DESIGN decisions 17 (opm releases with release-please), 19 (Edit on `main`). The site today takes "Edit this page" from git (`layouts/_partials/opm/source.html`: in a line version the branch the docs came from, otherwise `main`) and "Last updated" from `gen-lastmod.sh`.

## Goals / Non-Goals

**Goals:** `pages[].edit`; the specification of authored docs in docs bundles; the phase-3 bundle shapes.

**Non-Goals:** the site's switch (opmodel.dev); `opm-docs serve` (`add-serve-command`); the enhancements bundle (`add-enhancements-bundle`); moving the site's own section pages (`/docs/authoring/_index.md` and the like stay site content).

## Decisions

### D1. `pages[].edit`

```cue
#Page: {
	path:      =~"..."            // unchanged
	source?:   string & !=""
	lastmod?:  time.Time
	generated: bool
	edit?:     string & !=""      // the source file's path on main, when main has it; authored pages only
}
```

The **main tree** is the current directory when `--source` names a different directory (release mode's `src/`, a revision's temporary worktree), else the source tree. `build` sets `edit` to `source` when that path is a regular file in the main tree's `HEAD` (`git cat-file -e HEAD:<path>`), and omits it otherwise. A rename on `main` is not followed: the site shows no Edit link for that page rather than guess (decided in planning). Generated pages (`generated: true`) never get `edit`; a completed page (authored front matter plus generated body) is authored and does.

What the site does with it (recorded in C8 so both sides agree):

| Page | Edit this page | View source |
|---|---|---|
| tab page (`/catalogs/`) | none: a fix lands on `main` and reaches a minor by a docs revision | `source` at `source.commit` |
| docs page, authored | `https://github.com/<source.repo>/edit/main/<edit>` when `edit` is set | `source` at `source.commit` |
| docs page, generated | none | `source` at `source.commit` when set |
| section page (`/enhancements/`) | none: an entry changes through its own review | `source` at `source.commit` |

### D2. Authored docs, one bundle per repository

A repository has one docs-placed bundle. Its generated reference and its authored `docs/site/` are sources of the same bundle, from the repository's adoption on: core, cli, library and opm-operator ship both at once (their phase-2 siblings), while their committed generated pages are excluded (`exclude`, C6) until the site reads the bundle and they are deleted; a committed generated page that is not excluded collides and fails the build. catalog_opm's docs and opm follow in phase 3. The configurations, as the siblings write them:

```cue
// core, after its committed reference is deleted (before that, the markdown
// source carries exclude: ["reference/definitions/"]).
bundles: core: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/definitions/"]}
	version: {from: "tag", prefix: "v"}
	sources: [
		{kind: "cue-definitions", /* as add-cue-definitions-extractor D1 */},
		{kind: "markdown", dir: "docs/site"},
	]
}

// catalog_opm: the tab stays; its docs/site becomes a second, docs-placed project.
bundles: {
	"catalog-opm":      {placement: {kind: "tab", root: "/catalogs/opm/"}, version: {from: "tag", prefix: "opm-v"}, sources: [/* unchanged */]}
	"catalog-opm-docs": {placement: {kind: "docs", root: "/docs/"}, version: {from: "tag", prefix: "opm-v"}, sources: [{kind: "markdown", dir: "docs/site"}]}
}

// opm: authored only, released by release-please with tags v<semver> (DESIGN decision 17).
bundles: opm: {
	placement: {kind: "docs", root: "/docs/"}
	version: {from: "tag", prefix: "v"}
	sources: [{kind: "markdown", dir: "docs/site"}]
}
```

`catalog-opm` and `catalog-opm-docs` publish from the same release tag (`opm-v4.6.0`): catalog_opm's `publish-docs` job calls `publish.yml` once per project. The opmodel.dev site version names them separately: the tab by its own config (`tabs`), the docs by `versions."v1.0".tags."catalog-opm-docs"` (a major, `"4"`, as the site's `catalog-line = opm-v4` does today). opm's version is the site version's `tags.opm`; its first release version is the opm sibling's choice (recommended `1.0.0-beta.1`, so `tags: {opm: "1.0"}` follows the v1.0 site version).

A `markdown` source in a docs bundle copies pages as written: no alias pinning (C8's rule pins only a tab bundle's own catalog links), and docs-mode link rules apply in bundle-mode lint (C15). Figure shortcodes pass as in dialect 1. `lastmod` is the file's last commit at the commit built (C3), as today's `gen-lastmod.sh` computes it from the archived tree.

### D3. Commands

No new command or flag. No new error beyond the existing collision message (C6).

## Research & Decisions

### Following renames for `edit` (decided in planning)

**Options considered**: 1. `git log --follow` from the release commit to `main` - can pick a wrong file after a split or a copy; 2. same path or nothing - never a wrong link.
**Decision**: option 2. A page whose file moved shows no Edit link until its next release.

### Where `edit` is decided (decided in planning)

**Context**: the site could link `edit/main/<source>` blindly.
**Decision**: the producer decides, because only it has `main` beside the release tree (C5); a blind link 404s after a rename and the site has no git to check against in phase 3.

## Risks / Trade-offs

- A section page under `/docs/` written by two repositories (two `start/_index.md`) fails the site version's pull (C16 D4); the owning repository is the one whose `_index.md` exists today (opm owns `start/_index.md`), and the phase-3 siblings keep it so.

## Durable decisions

- C3: `#Page.edit`. C8: the Edit and View-source table (D1). C15: authored docs in docs bundles and the D2 configurations.
- `README.md`: adopting phase 3 in a repository.
