# enhancements-bundle Specification

## Purpose
The section placement and the enhancements source: the enhancements repository built, from `main` only, into one unversioned bundle at `/enhancements/`, its entries turned into dialect pages with their relative links resolved and their header data written to `data/enhancements.json` (docs/contracts.md C21).

## Requirements

### Requirement: The source locates both layouts by three directories
The `enhancements` source SHALL take `changes` (default `openspec/changes`), `specs` (default `openspec/specs`) and `legacy` (default `archive`), each a repo-relative path under `dir`, and SHALL refuse a config where one of the three equals or nests another (exit 1). A directory that does not exist SHALL hold no entries.

#### Scenario: Defaults
- **WHEN** an `enhancements` source sets only `description`
- **THEN** it reads changes from `openspec/changes`, specs from `openspec/specs` and old entries from `archive`

#### Scenario: Overlapping directories
- **WHEN** `legacy` is `openspec/changes/archive`
- **THEN** loading the config exits 1 naming `legacy` and `changes`

### Requirement: The openspec reader reads changes as entries
The source SHALL read `<changes>/<id>-<slug>/` and `<changes>/archive/<date>-<id>-<slug>/` as entries, skip every `seed-` and `correction-` change, and require `config.yaml` (its `id` and `slug` equal to the folder's), a `README.md`, `proposal.md` and `design.md`; `tasks.md` and `specs/` MAY be absent. A change folder named any other way SHALL fail the build with exit 2 naming it and the forms `<id>-<slug>`, `seed-<capability>` and `correction-<slug>`.

#### Scenario: A live and an archived entry
- **WHEN** `openspec/changes/0026-x/` and `openspec/changes/archive/2026-10-20-0027-y/` both hold the required files
- **THEN** the bundle holds entries 0026 and 0027, the second with `archived: true`

#### Scenario: A seed change
- **WHEN** `openspec/changes/archive/2026-10-14-seed-single-build-render/` exists
- **THEN** it is no entry and no page

#### Scenario: A refused name
- **WHEN** `openspec/changes/fix-things/` exists
- **THEN** `build` exits 2 with `change folder openspec/changes/fix-things is not <id>-<slug>, a seed-<capability> change or a correction-<slug> change; an enhancement is a change named by its id and slug`

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
When `specs/` holds a capability, the source SHALL publish `capabilities/_index.md` and `capabilities/<cap>.md` from `<specs>/<cap>/spec.md`. A capability page SHALL take its title from the name and its description from the first sentence of its Purpose, and end with a table "Entries that changed it": one row for each archived change whose deltas touch the capability, with its archive date, linked only when it is a `delivered` entry. A `seed-` change shows `seed` and a `correction-` change shows `correction`, each unlinked. Rows run by date, then entry id (none first), then `correction` before `seed`.

#### Scenario: Entries that changed a capability
- **WHEN** a delivered entry 0027 and the change `seed-registration-acceptance` both touched `registration-acceptance`
- **THEN** its page lists both with their archive dates, 0027 as a link to `/enhancements/0027/`

#### Scenario: Order of rows
- **WHEN** a seed and a correction are archived on 2026-10-21, the delivered entry 0027 on 2026-10-21 and a correction on 2026-10-23
- **THEN** the rows are `correction | 2026-10-21`, `seed | 2026-10-21`, `[0027: <title>](/enhancements/0027/) | 2026-10-21`, `correction | 2026-10-23`

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

### Requirement: The strip policy removes a lone enhancement citation
Under the `strip` citation policy, prose outside a code span SHALL lose each parenthesis whose whole content is `enhancement NNNN` (a capital first letter too) or `enhancements NNNN and NNNN` (ids joined by commas and `and`), together with the one space before it, so the sentence stays whole. A parenthesis that holds other words, a code span, a comment inside a spec block and a Markdown link destination MUST stay as written.

#### Scenario: End of a sentence
- **WHEN** prose reads `It refuses the render (enhancement 0013).` under `strip`
- **THEN** the page reads `It refuses the render.`

#### Scenario: Middle of a sentence
- **WHEN** prose reads `The rule (enhancement 0013) holds here.` under `strip`
- **THEN** the page reads `The rule holds here.`

#### Scenario: A list of ids
- **WHEN** prose reads `The rule (enhancements 0013 and 0019).` under `strip`
- **THEN** the page reads `The rule.`

#### Scenario: A parenthesis with other words
- **WHEN** prose reads `The design (see enhancement 0013 for it) is final.` under `strip`
- **THEN** the page keeps the text as written

#### Scenario: A code span
- **WHEN** prose reads `Write `(enhancement 0013)` here.` under `strip`
- **THEN** the code span stays as written

#### Scenario: Fewer than four digits
- **WHEN** prose reads `The rule (enhancement 13).` under `strip`
- **THEN** the page keeps the text as written

#### Scenario: The link policy
- **WHEN** prose reads `The rule (enhancement 0013).` under `link`
- **THEN** the parenthesis stays and `enhancement 0013` links to `/enhancements/0013/`

### Requirement: A section bundle is built from main only
A bundle with `placement: {kind: "section", root: "/enhancements/"}` SHALL be built only as edge (`version: "edge"`, revision 0); `build --release` and `revise` SHALL refuse it with exit 1 naming the project and that a section publishes from `main` only. Its config MAY omit `version`. Source: DESIGN decision 18.

#### Scenario: A release build of the enhancements bundle
- **WHEN** `opm-docs build --release v1.0.0 --project enhancements` runs
- **THEN** it exits 1 saying a section bundle builds from `main` only

### Requirement: A section bundle publishes at its root
The pages of a section bundle SHALL publish at `<root><page URL>` with no segment and SHALL carry no `edit`. Source: DESIGN decision 18.

#### Scenario: A section page URL
- **WHEN** the enhancements bundle holds the page `graph.md`
- **THEN** it publishes at `/enhancements/graph/`, with no segment and no `edit`

### Requirement: A section bundle holds exactly one enhancements source
A section bundle SHALL hold exactly one source, of kind `enhancements`, and an `enhancements` source SHALL be refused outside a section bundle, with exit 1 naming the source. Source: DESIGN decision 18.

#### Scenario: A markdown source in the section
- **WHEN** the enhancements bundle lists a `markdown` source beside its `enhancements` source
- **THEN** `build` exits 1 naming the markdown source

#### Scenario: An enhancements source outside a section
- **WHEN** a bundle with a placement other than `section` lists an `enhancements` source
- **THEN** `build` exits 1 naming the source

### Requirement: The enhancements source reads old-layout entry directories
An `enhancements` source SHALL read every old-layout entry directory `<legacy>/NNNN/` (`archive/NNNN/` by default), requiring a `config.yaml` whose `id` equals the directory's id, a `README.md` and exactly one file per numbered document `01-*.md` to `07-*.md`. A missing or extra numbered document, or an id mismatch, SHALL fail the build with exit 2 naming the entry.

#### Scenario: Two decision files
- **WHEN** `archive/0025/` holds `03-decisions.md` and `03-decisions-old.md`
- **THEN** `build` exits 2 naming `0025` and both files

#### Scenario: An id mismatch
- **WHEN** `archive/0025/config.yaml` holds an `id` other than `0025`
- **THEN** `build` exits 2 naming `0025`

### Requirement: The enhancements source writes the entries data file
An `enhancements` source SHALL write `data/enhancements.json` (`docs.opmodel.dev/data/enhancements/v2`) with each entry's id, layout, title, summary, status, category, affects, created and updated dates, relations and whether it is archived.

#### Scenario: The entries data file
- **WHEN** an `enhancements` source reads an entry
- **THEN** `data/enhancements.json` (`docs.opmodel.dev/data/enhancements/v2`) holds its id, layout, title, summary, status, category, affects, created and updated dates, relations and whether it is archived

### Requirement: An old-layout entry becomes dialect pages
An `enhancements` source SHALL write the pages `_index.md` (from `INDEX.md`), `graph.md` (from `GRAPH.md`), `<NNNN>/_index.md` (from the README) and `<NNNN>/<slug>.md` for `problem`, `design`, `decisions`, `graduation`, `risks`, `operational` and `questions`, each with front matter `title`, `description` and, on a leaf page, `type: explanation`.

#### Scenario: An archived entry keeps its URL
- **WHEN** entry 0003 lives at `archive/0003/`
- **THEN** its pages are `0003/_index.md` and `0003/<slug>.md`, published at `/enhancements/0003/...`

### Requirement: Enhancement text is cleaned at build
The source SHALL remove HTML comments outside code and the first `# ` heading of each file; SHALL tag an untagged opening code fence `text`; SHALL refuse a Hugo shortcode delimiter; SHALL escape a `<` outside code that would open raw HTML; and SHALL refuse a link scheme other than `http`, `https` and `mailto`.

#### Scenario: Raw HTML in prose
- **WHEN** a document's prose holds `<module-path>` outside a code span
- **THEN** the page holds `\<module-path>`, which the site shows as written

#### Scenario: A script link scheme
- **WHEN** a document links `javascript:alert(1)`
- **THEN** `build` refuses the link scheme

### Requirement: Relative links resolve against the repository path
The source SHALL resolve every relative link against the file's repository path: an entry or its README to `/enhancements/<NNNN>/`, a numbered document to `/enhancements/<NNNN>/<slug>/`, `INDEX.md` to `/enhancements/`, `GRAPH.md` to `/enhancements/graph/`, any other path that exists at the commit built to `https://github.com/<repo>/blob/<commit>/<path>` (`tree` for a directory), keeping a fragment.

#### Scenario: A sibling document link
- **WHEN** `0010/02-design.md` links `03-decisions.md#d11`
- **THEN** the page links `/enhancements/0010/decisions/#d11`

#### Scenario: A schema file link
- **WHEN** `0013/02-design.md` links `schemas/target.cue`, which exists at the commit built
- **THEN** the page links `https://github.com/open-platform-model/enhancements/blob/<commit>/0013/schemas/target.cue`

### Requirement: A link to a file name reads as the page title
A link whose text is the target's file name SHALL read as the target page's title (`INDEX.md` as "the index", `GRAPH.md` as "the relationship graph").

#### Scenario: A link named for the graph file
- **WHEN** a document holds a link whose text is `GRAPH.md` and whose target is `GRAPH.md`
- **THEN** the link text reads "the relationship graph"

### Requirement: A dangling link fails the build
A link naming no path at the commit built, or climbing out of the repository, SHALL fail the build with exit 2 naming the file and the link.

#### Scenario: A dangling link
- **WHEN** a document links `missing.md`
- **THEN** `build` exits 2 naming the file and `missing.md`

#### Scenario: A link out of the repository
- **WHEN** a document links a path that climbs out of the repository
- **THEN** `build` exits 2 naming the file and the link

### Requirement: A section page body is prepared as Hugo prepares it
Bundle-mode lint of a section bundle SHALL prepare every page body as Hugo 0.167.0 does (un-indenting each line that starts with Hugo's context marker `{{__hugo_ctx`), and SHALL refuse that marker anywhere in the body.

#### Scenario: A context marker un-indents an HTML line
- **WHEN** a document holds the indented line `    {{__hugo_ctx/}} <svg onload=alert(1)>`
- **THEN** `build` exits 2 naming the page and the line

### Requirement: A section page is parsed with goldmark as Hugo configures it
Bundle-mode lint of a section bundle SHALL parse the body with goldmark as Hugo configures it for opmodel.dev (goldmark v1.8.6, the version Hugo 0.167.0 builds with), decoding each destination as goldmark's renderer does until it is stable.

#### Scenario: A script scheme hidden behind escapes
- **WHEN** a document holds `[a](javascript\:alert(1))` or `[a](javascript&#38;colon;alert(2))`
- **THEN** `build` exits 2 naming the page and the line

### Requirement: Section lint reports raw HTML and heading attributes
Bundle-mode lint of a section bundle SHALL report, as `<file>:<line>: <message>`, any raw HTML (inline or a block) and any heading attribute block.

#### Scenario: Raw HTML after an escaped backslash
- **WHEN** a document's prose holds `\\<img src=x onerror=alert(1)>`
- **THEN** the page holds no raw HTML, or `build` exits 2 naming the page and the line

### Requirement: Section lint reports a destination with a forbidden scheme
Bundle-mode lint of a section bundle SHALL report, as `<file>:<line>: <message>`, any link, image, autolink or reference definition whose destination has a scheme other than `http`, `https` or `mailto` (character references resolved) or is protocol-relative.

#### Scenario: A script link in a quoted reference definition
- **WHEN** a document holds `> [r]: javascript:alert(1)` and `> [click][r]`
- **THEN** `build` exits 2 naming the page and the line of each

### Requirement: Every command applies the section check
`build`, `lint --bundle` and `pull` SHALL all apply the bundle-mode lint of a section bundle. That lint, not the text transforms, is what keeps a section's authored text from putting markup or script URLs on the site.

#### Scenario: The check runs in every command
- **WHEN** a section page holds raw HTML
- **THEN** `build`, `lint --bundle` and `pull` each report it as `<file>:<line>: <message>`

### Requirement: The link policy turns enhancement ids into links
Under the `link` citation policy, prose outside a code span SHALL turn `enhancement NNNN` into a link to `/enhancements/NNNN/` and each id of `enhancements NNNN and NNNN` into one. The `crd` and `go-api` sources SHALL show the same links.

#### Scenario: One enhancement
- **WHEN** a comment reads `(enhancement 0013)` under `link`
- **THEN** the page reads `(` + a link `enhancement 0013` to `/enhancements/0013/` + `)`

#### Scenario: Two enhancements
- **WHEN** a comment reads `(enhancements 0010 and 0026)` under `link`
- **THEN** `0010` and `0026` each link to their entry page

### Requirement: The link policy removes decision citations
`link` SHALL remove a decision citation (`NNNN:Dn` and its variants) without a link, as `strip` does, because an entry of the OpenSpec layout has no decisions page.

#### Scenario: A citation form
- **WHEN** a comment reads an enhancement id followed by a decision number, or `enhancement 0010 experiment 3`, under `link`
- **THEN** the decision citation and the experiment pointer are removed, with no link to `/enhancements/0010/`

### Requirement: The strip policy leaves other enhancement references as written
`strip` SHALL leave `enhancement NNNN` as written, except in a parenthesis that holds nothing else.

#### Scenario: A reference outside a parenthesis
- **WHEN** prose reads `See enhancement 0013 for it.` under `strip`
- **THEN** the page keeps the text as written

### Requirement: Code spans are found by CommonMark backtick runs under both policies
The `link` policy MUST keep its output unchanged by the `strip` rules, except that code spans are found by CommonMark backtick runs under both policies: a double-backtick span that holds a backtick is one span, and an unclosed backtick is plain text.

#### Scenario: A double-backtick span
- **WHEN** prose holds a double-backtick span that holds a backtick and `(enhancement 0013)`
- **THEN** the span is one code span under both policies

#### Scenario: An unclosed backtick
- **WHEN** prose holds an unclosed backtick
- **THEN** it is plain text under both policies

### Requirement: A correction change is no entry
The openspec reader SHALL skip a live `<changes>/correction-<slug>/` and an archived `<changes>/archive/<date>-correction-<slug>/` as it skips a seed: no entry, no page, no field in `data/enhancements.json`. The `<slug>` MUST be non-empty lower-case words (letters and digits) joined by single `-`, and need not name a capability.

#### Scenario: A live correction
- **WHEN** `openspec/changes/correction-registry-paths/` exists beside entries
- **THEN** `build` succeeds and the bundle holds the same entries and pages as without it

#### Scenario: An archived correction
- **WHEN** `openspec/changes/archive/2026-10-23-correction-registry-paths/` exists
- **THEN** it is no entry and no page

#### Scenario: A malformed slug
- **WHEN** a folder is named `correction-`, `correction`, `correction-Foo_Bar` or, archived, `2026-10-23-correction-`
- **THEN** `build` exits 2 naming the folder

#### Scenario: A numbered name
- **WHEN** a folder is named `0123-correction-x`
- **THEN** it is read as entry 0123 with slug `correction-x`

### Requirement: A capability page lists a correction as an unlinked row
A capability page SHALL list each archived correction whose `specs/` holds the capability as the row `correction` with its archive date, with no link. A correction that touches several capabilities SHALL give a row on each page.

#### Scenario: A correction row
- **WHEN** the archived change `2026-10-23-correction-registry-paths` holds deltas for `registry-paths` and `tag-rules`
- **THEN** both pages list `| correction | 2026-10-23 |`
