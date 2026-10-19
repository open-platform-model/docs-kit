## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: A section bundle is edge-only and unversioned
**Reason**: Split into three requirements that each state one behavior.
**Migration**: See "A section bundle is built from main only", "A section bundle publishes at its root", "A section bundle holds exactly one enhancements source".

### Requirement: The enhancements source turns entries into dialect pages
**Reason**: Split into three requirements that each state one behavior.
**Migration**: See "The enhancements source reads old-layout entry directories", "The enhancements source writes the entries data file", "An old-layout entry becomes dialect pages".

### Requirement: Enhancement text is cleaned and its links resolved at build
**Reason**: Split into four requirements that each state one behavior.
**Migration**: See "Enhancement text is cleaned at build", "Relative links resolve against the repository path", "A link to a file name reads as the page title", "A dangling link fails the build".

### Requirement: A section page is checked as the site parses it
**Reason**: Split into five requirements that each state one behavior.
**Migration**: See "A section page body is prepared as Hugo prepares it", "A section page is parsed with goldmark as Hugo configures it", "Section lint reports raw HTML and heading attributes", "Section lint reports a destination with a forbidden scheme", "Every command applies the section check".

### Requirement: The link policy links enhancement references
**Reason**: Split into three requirements that each state one behavior.
**Migration**: See "The link policy turns enhancement ids into links", "The link policy removes decision citations", "The strip policy leaves other enhancement references as written".

## ADDED Requirements

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
