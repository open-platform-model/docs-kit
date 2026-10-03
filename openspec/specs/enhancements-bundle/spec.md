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
An `enhancements` source SHALL read every entry directory `NNNN/` and `archive/NNNN/` except `0000`, requiring a `config.yaml` whose `id` equals the directory's id, a `README.md` and exactly one file per numbered document `01-*.md` to `07-*.md`, and SHALL write `data/enhancements.json` (`docs.opmodel.dev/data/enhancements/v1`) with each entry's id, title, summary, status, category, affects, created and updated dates, relations and whether it is archived, and the pages `_index.md` (from `INDEX.md`), `graph.md` (from `GRAPH.md`), `<NNNN>/_index.md` (from the README) and `<NNNN>/<slug>.md` for `problem`, `design`, `decisions`, `graduation`, `risks`, `operational` and `questions`, each with front matter `title`, `description` and, on a leaf page, `type: explanation`. A missing or extra numbered document, or an id mismatch, SHALL fail the build with exit 2 naming the entry.

#### Scenario: An archived entry keeps its URL
- **WHEN** entry 0003 lives at `archive/0003/`
- **THEN** its pages are `0003/_index.md` and `0003/<slug>.md`, published at `/enhancements/0003/...`

#### Scenario: Two decision files
- **WHEN** `0025/` holds `03-decisions.md` and `03-decisions-old.md`
- **THEN** `build` exits 2 naming `0025` and both files

### Requirement: Enhancement text is cleaned and its links resolved at build
The source SHALL remove HTML comments outside code and the first `# ` heading of each file; SHALL tag an untagged opening code fence `text`; SHALL refuse a Hugo shortcode delimiter; SHALL escape a `<` outside code that would open raw HTML; SHALL refuse a link scheme other than `http`, `https` and `mailto`; and SHALL resolve every relative link against the file's repository path: an entry or its README to `/enhancements/<NNNN>/`, a numbered document to `/enhancements/<NNNN>/<slug>/`, `INDEX.md` to `/enhancements/`, `GRAPH.md` to `/enhancements/graph/`, any other path that exists at the commit built to `https://github.com/<repo>/blob/<commit>/<path>` (`tree` for a directory), keeping a fragment. A link whose text is the target's file name SHALL read as the target page's title (`INDEX.md` as "the index", `GRAPH.md` as "the relationship graph"). A link naming no path at that commit, or climbing out of the repository, SHALL fail the build with exit 2 naming the file and the link.

#### Scenario: A sibling document link
- **WHEN** `0025/02-design.md` links `03-decisions.md#d11`
- **THEN** the page links `/enhancements/0025/decisions/#d11`

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
