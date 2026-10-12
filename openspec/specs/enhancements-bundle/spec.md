# enhancements-bundle Specification

## Purpose
The section placement and the enhancements source: the enhancements repository built, from `main` only, into one unversioned bundle at `/enhancements/`, its entries turned into dialect pages with their relative links resolved and their header data written to `data/enhancements.json` (docs/contracts.md C21).

## Requirements

### Requirement: A section bundle is edge-only and unversioned
A bundle with `placement: {kind: "section", root: "/enhancements/"}` SHALL be built only as edge (`version: "edge"`, revision 0); `build --release` and `revise` SHALL refuse it with exit 1 naming the project and that a section publishes from `main` only. Its config MAY omit `version`. Its pages SHALL publish at `<root><page URL>` with no segment and SHALL carry no `edit`. A section bundle SHALL hold exactly one source, of kind `enhancements`, and an `enhancements` source SHALL be refused outside a section bundle, with exit 1 naming the source. Source: DESIGN decision 18.

#### Scenario: A release build of the enhancements bundle
- **WHEN** `opm-docs build --release v1.0.0 --project enhancements` runs
- **THEN** it exits 1 saying a section bundle builds from `main` only

#### Scenario: A markdown source in the section
- **WHEN** the enhancements bundle lists a `markdown` source beside its `enhancements` source
- **THEN** `build` exits 1 naming the markdown source

### Requirement: The enhancements source turns entries into dialect pages
An `enhancements` source SHALL read every old-layout entry directory `<legacy>/NNNN/` (`archive/NNNN/` by default), requiring a `config.yaml` whose `id` equals the directory's id, a `README.md` and exactly one file per numbered document `01-*.md` to `07-*.md`, and SHALL write `data/enhancements.json` (`docs.opmodel.dev/data/enhancements/v2`) with each entry's id, layout, title, summary, status, category, affects, created and updated dates, relations and whether it is archived, and the pages `_index.md` (from `INDEX.md`), `graph.md` (from `GRAPH.md`), `<NNNN>/_index.md` (from the README) and `<NNNN>/<slug>.md` for `problem`, `design`, `decisions`, `graduation`, `risks`, `operational` and `questions`, each with front matter `title`, `description` and, on a leaf page, `type: explanation`. A missing or extra numbered document, or an id mismatch, SHALL fail the build with exit 2 naming the entry.

#### Scenario: An archived entry keeps its URL
- **WHEN** entry 0003 lives at `archive/0003/`
- **THEN** its pages are `0003/_index.md` and `0003/<slug>.md`, published at `/enhancements/0003/...`

#### Scenario: Two decision files
- **WHEN** `archive/0025/` holds `03-decisions.md` and `03-decisions-old.md`
- **THEN** `build` exits 2 naming `0025` and both files

### Requirement: Enhancement text is cleaned and its links resolved at build
The source SHALL remove HTML comments outside code and the first `# ` heading of each file; SHALL tag an untagged opening code fence `text`; SHALL refuse a Hugo shortcode delimiter; SHALL escape a `<` outside code that would open raw HTML; SHALL refuse a link scheme other than `http`, `https` and `mailto`; and SHALL resolve every relative link against the file's repository path: an entry or its README to `/enhancements/<NNNN>/`, a numbered document to `/enhancements/<NNNN>/<slug>/`, `INDEX.md` to `/enhancements/`, `GRAPH.md` to `/enhancements/graph/`, any other path that exists at the commit built to `https://github.com/<repo>/blob/<commit>/<path>` (`tree` for a directory), keeping a fragment. A link whose text is the target's file name SHALL read as the target page's title (`INDEX.md` as "the index", `GRAPH.md` as "the relationship graph"). A link naming no path at that commit, or climbing out of the repository, SHALL fail the build with exit 2 naming the file and the link.

#### Scenario: A sibling document link
- **WHEN** `0010/02-design.md` links `03-decisions.md#d11`
- **THEN** the page links `/enhancements/0010/decisions/#d11`

#### Scenario: A schema file link
- **WHEN** `0013/02-design.md` links `schemas/target.cue`, which exists at the commit built
- **THEN** the page links `https://github.com/open-platform-model/enhancements/blob/<commit>/0013/schemas/target.cue`

#### Scenario: A dangling link
- **WHEN** a document links `missing.md`
- **THEN** `build` exits 2 naming the file and `missing.md`

#### Scenario: Raw HTML in prose
- **WHEN** a document's prose holds `<module-path>` outside a code span
- **THEN** the page holds `\<module-path>`, which the site shows as written

### Requirement: A section page is checked as the site parses it
Bundle-mode lint of a section bundle SHALL prepare every page body as Hugo 0.167.0 does (un-indenting each line that starts with Hugo's context marker `{{__hugo_ctx`), SHALL refuse that marker anywhere in the body, SHALL parse the body with goldmark as Hugo configures it for opmodel.dev (goldmark v1.8.6, the version Hugo 0.167.0 builds with), decoding each destination as goldmark's renderer does until it is stable, and SHALL report, as `<file>:<line>: <message>`, any raw HTML (inline or a block), any link, image, autolink or reference definition whose destination has a scheme other than `http`, `https` or `mailto` (character references resolved) or is protocol-relative, and any heading attribute block. This check, not the text transforms, is what keeps a section's authored text from putting markup or script URLs on the site; `build`, `lint --bundle` and `pull` all apply it.

#### Scenario: A script link in a quoted reference definition
- **WHEN** a document holds `> [r]: javascript:alert(1)` and `> [click][r]`
- **THEN** `build` exits 2 naming the page and the line of each

#### Scenario: Raw HTML after an escaped backslash
- **WHEN** a document's prose holds `\\<img src=x onerror=alert(1)>`
- **THEN** the page holds no raw HTML, or `build` exits 2 naming the page and the line

#### Scenario: A context marker un-indents an HTML line
- **WHEN** a document holds the indented line `    {{__hugo_ctx/}} <svg onload=alert(1)>`
- **THEN** `build` exits 2 naming the page and the line

#### Scenario: A script scheme hidden behind escapes
- **WHEN** a document holds `[a](javascript\:alert(1))` or `[a](javascript&#38;colon;alert(2))`
- **THEN** `build` exits 2 naming the page and the line

### Requirement: The source locates both layouts by three directories
The `enhancements` source SHALL take `changes` (default `openspec/changes`), `specs` (default `openspec/specs`) and `legacy` (default `archive`), each a repo-relative path under `dir`, and SHALL refuse a config where one of the three equals or nests another (exit 1). A directory that does not exist SHALL hold no entries.

#### Scenario: Defaults
- **WHEN** an `enhancements` source sets only `description`
- **THEN** it reads changes from `openspec/changes`, specs from `openspec/specs` and old entries from `archive`

#### Scenario: Overlapping directories
- **WHEN** `legacy` is `openspec/changes/archive`
- **THEN** loading the config exits 1 naming `legacy` and `changes`

### Requirement: The openspec reader reads changes as entries
The source SHALL read `<changes>/<id>-<slug>/` and `<changes>/archive/<date>-<id>-<slug>/` as entries, skip every `seed-` change, and require `config.yaml` (its `id` and `slug` equal to the folder's), a `README.md`, `proposal.md` and `design.md`; `tasks.md` and `specs/` MAY be absent. A change folder named any other way SHALL fail the build with exit 2 naming it.

#### Scenario: A live and an archived entry
- **WHEN** `openspec/changes/0026-x/` and `openspec/changes/archive/2026-10-20-0027-y/` both hold the required files
- **THEN** the bundle holds entries 0026 and 0027, the second with `archived: true`

#### Scenario: A seed change
- **WHEN** `openspec/changes/archive/2026-10-14-seed-single-build-render/` exists
- **THEN** it is no entry and no page

#### Scenario: A missing design
- **WHEN** an entry has no `design.md`
- **THEN** `build` exits 2 naming the entry and `design.md`

### Requirement: An entry belongs to one reader and is not a symlink
The source SHALL refuse an id found by both readers, an id found twice by the openspec reader, a symlink where it reads a directory or a file, and a directory `NNNN/` at the entry root (a legacy live folder), each with exit 2 naming the paths and the fix.

#### Scenario: One id in both layouts
- **WHEN** `archive/0003/` and `openspec/changes/0003-x/` both exist
- **THEN** `build` exits 2 naming `0003` and both paths

#### Scenario: A legacy live folder
- **WHEN** `0025/` exists at the entry root
- **THEN** `build` exits 2 saying a live legacy folder must be converted or moved to `archive/`

### Requirement: Data schema v2 describes both layouts
`data/enhancements.json` SHALL use the schema `docs.opmodel.dev/data/enhancements/v2`, entries by id, and every entry SHALL carry `layout`, `legacy` or `openspec`. A legacy entry SHALL keep every v1 field. An openspec entry SHALL carry `breaking` (`true`, `false` or `null`), `updated`, `after`, `supersededBy` as a list of ids, `accepted` (an object with `date` and `commit`, or `null`), `progress` (`done`, `total`), `capabilities`, `documents` and `aliases`.

#### Scenario: A legacy entry
- **WHEN** entry 0003 is read from `archive/0003/`
- **THEN** its record holds `layout: "legacy"`, its `dependsOn`, `amends`, `supersedes`, `revives` and string or null `supersededBy`, and its stored `updated`

#### Scenario: An openspec entry
- **WHEN** entry 0026 has `breaking: true`, `accepted: "2026-10-16 4be19d0"` and `tasks.md` of five open lines
- **THEN** its record holds `breaking: true`, `accepted: {date: "2026-10-16", commit: "4be19d0"}` and `progress: {done: 0, total: 5}`

### Requirement: An openspec entry's updated date comes from git
The `updated` of an openspec entry SHALL be the UTC date of the last commit, at or before the commit built, that touched the entry's folder. The source SHALL ignore any `updated` in `config.yaml`, use `created` when no commit touched the folder, and fail the build with exit 2 when the repository is a shallow clone, naming a checkout with `fetch-depth: 0`.

#### Scenario: Date of the last touching commit
- **WHEN** a commit on 2026-10-20 changed `design.md` of entry 0026 and no later commit touched its folder
- **THEN** its `updated` is `2026-10-20`

#### Scenario: Shallow clone
- **WHEN** the source tree is a shallow clone and an openspec entry exists
- **THEN** `build` exits 2 naming the shallow clone and the full-depth checkout

### Requirement: An openspec entry publishes five pages at most
An openspec entry SHALL publish `<id>/_index.md` (from its `README.md`), `<id>/proposal.md`, `<id>/design.md`, `<id>/requirements.md` when `specs/` holds a capability, and `<id>/tasks.md` when `tasks.md` exists. The requirements page SHALL hold one `## <capability>` block per capability, name order, with the delta text cleaned as other pages are and its headings one level lower. `evidence/` SHALL be no page; a link to it resolves to GitHub at the commit built.

#### Scenario: Requirements by capability
- **WHEN** entry 0026 has deltas for four capabilities
- **THEN** `0026/requirements.md` holds four `##` headings in name order, each followed by its delta with `## ADDED Requirements` shown as `###`

#### Scenario: No specs, no tasks
- **WHEN** an entry has neither `specs/` nor `tasks.md`
- **THEN** it publishes `_index.md`, `proposal.md` and `design.md` only

#### Scenario: Evidence link
- **WHEN** the README links `evidence/`
- **THEN** the page links `https://github.com/<repo>/tree/<commit>/<entry path>/evidence`

### Requirement: Each capability publishes a page
When `specs/` holds a capability, the source SHALL publish `capabilities/_index.md` and `capabilities/<cap>.md` from `<specs>/<cap>/spec.md`. A capability page SHALL take its title from the name and its description from the first sentence of its Purpose, and end with a table "Entries that changed it": one row for each archived change whose deltas touch the capability, with its archive date, linked only when it is a `delivered` entry.

#### Scenario: Entries that changed a capability
- **WHEN** a delivered entry 0027 and the change `seed-registration-acceptance` both touched `registration-acceptance`
- **THEN** its page lists both with their archive dates, 0027 as a link to `/enhancements/0027/`

#### Scenario: A rejected entry
- **WHEN** an archived entry with status `rejected` holds a delta for the capability
- **THEN** it has no row

### Requirement: Redirect data maps old page slugs
An openspec entry that holds `evidence/migration-map.md` (a migrated entry, one that had old pages) SHALL list `aliases` as `{from, to}` page paths: `problem` and `questions` to `proposal`, `risks` and `operational` to `design`, `decisions` to `requirements`, `graduation` to `tasks`. When the entry lacks the target page, `to` SHALL be the entry's own page. An openspec entry without a regular file `evidence/migration-map.md` (a new entry) SHALL have none, and so SHALL a legacy entry.

#### Scenario: Decisions go to requirements
- **WHEN** migrated entry 0026 has a requirements page
- **THEN** its aliases include `{from: "0026/decisions", to: "0026/requirements"}`

#### Scenario: No tasks page
- **WHEN** migrated entry 0005 has no `tasks.md`
- **THEN** `{from: "0005/graduation", to: "0005"}` is among its aliases

#### Scenario: A new entry
- **WHEN** an openspec entry has no `evidence/migration-map.md`, or it is a symbolic link
- **THEN** its `aliases` is an empty list

### Requirement: The page dialect admits the new enhancement links
The dialect SHALL accept `/enhancements/<NNNN>/proposal/`, `requirements/`, `tasks/`, `/enhancements/capabilities/` and `/enhancements/capabilities/<cap>/`, with the old seven slugs, and bundle-mode lint of a section SHALL require each to name a page of the bundle.

#### Scenario: A capability link
- **WHEN** a section page links `/enhancements/capabilities/single-build-render/` and the page exists
- **THEN** lint accepts it

#### Scenario: A link to a missing capability
- **WHEN** a section page links `/enhancements/capabilities/missing/`
- **THEN** bundle-mode lint reports that no page `capabilities/missing/` exists

### Requirement: The link policy links enhancement references
Under the `link` citation policy, prose outside a code span SHALL turn `enhancement NNNN` into a link to `/enhancements/NNNN/` and each id of `enhancements NNNN and NNNN` into one. `link` SHALL remove a decision citation (`NNNN:Dn` and its variants) without a link, as `strip` does, because an entry of the OpenSpec layout has no decisions page; `strip` SHALL leave `enhancement NNNN` as written. The `crd` and `go-api` sources SHALL show the same links.

#### Scenario: One enhancement
- **WHEN** a comment reads `(enhancement 0013)` under `link`
- **THEN** the page reads `(` + a link `enhancement 0013` to `/enhancements/0013/` + `)`

#### Scenario: Two enhancements
- **WHEN** a comment reads `(enhancements 0010 and 0026)` under `link`
- **THEN** `0010` and `0026` each link to their entry page

#### Scenario: A citation form
- **WHEN** a comment reads an enhancement id followed by a decision number, or `enhancement 0010 experiment 3`, under `link`
- **THEN** the decision citation and the experiment pointer are removed, with no link to `/enhancements/0010/`
