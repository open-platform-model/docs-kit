## Context

`openspec validate --all --strict` under 1.14.1 reports 47 requirements over 500 characters in
17 main specs of this repo (counted 2026-10-19 on `origin/main` 28a5f5d, with
`2>&1 | grep -c WARNING`; the per-spec counts equal those of the swarm's inventory). The length
is the requirement body: the lines between `### Requirement:` and the first line that starts with
`#`, without blank lines, fenced lines and `**Key**:` lines, joined by `\n`. Scenarios do not
count. The check runs on main specs and on ADDED blocks, and not on MODIFIED blocks. A MODIFIED
body that stays over 500 therefore passes `validate <change> --strict`, archives, and then fails
`validate --specs --strict`. This repo has no active change, so no rewrite collides with one.

The change is text only. It has no trust boundary, credential, personal data or code path, so it
has no security section beyond this line: the only risk is a lost rule, and the ledger and the
review checklist guard that.

## Research & Decisions

### Which form per requirement
**Context**: Each of the 47 requirements needs a rewrite that keeps every rule and every
scenario, and the validator does not compare scenario text.
**Explored**: The four forms of the approved guide (a: MODIFIED with a shorter body and more
scenarios; b: MODIFIED plus ADDED; c2: REMOVED plus ADDED under new names; d: RENAMED plus
MODIFIED) and their unsafe pairs, run in scratch copies by the swarm.
**Options considered**:
1. Form a everywhere - keeps names and scenarios safe, but 40 of the 47 requirements are labelled
   "several behaviors" and a body of several SHALL clauses stays untestable one by one.
2. Form c2 everywhere - gives one behavior per requirement, but it is the only form that may lose
   a scenario with no error, and it breaks any reference to the old header.
3. Choose per requirement from what the excess is (the guide's table).
**Decision**: Option 3.
**Rationale**: The table of the guide fits this repo:

| The excess is | Form | Typical docs-kit case |
| --- | --- | --- |
| Examples, lists, formats, rationale around one behavior | a | "format" only: `bundle-format`, "One version's docs bundles never overlap", "Authored docs pages ship in docs bundles" |
| Several behaviors, and the old scenarios fit the parent that stays | b | A parent rule plus one or two split-off behaviors |
| Several behaviors, and the old scenarios belong to different behaviors | c2 | Long "several behaviors" requirements such as the `opm-docs-cli` commands and `bundle-pull` pull config |
| A name that must change, with no rule change | d, alone | Not expected; no rename is planned |

The kind labels of the inventory are a heuristic: the implementer reads the text before it
chooses. A default or an exact value may move from a body to a scenario THEN line (keeping its
SHALL). A clause that adds or changes a condition stays in a body. A separate behavior goes to
form b or c2.

### The batch order
**Context**: Batches are whole specs; B1 has 28 requirements, B2 has 19.
**Options considered**: (1) B1 then B2 (first-fit decreasing, as the inventory built them);
(2) B2 first because it is the smaller. 
**Decision**: B1 then B2, one tasks.md section each, then the `--strict` section.
**Rationale**: The `--strict` flag can return only after both batches pass, so the extra is last
and carries the final gate. Each section ends green and closes with one commit
(Principle VII). The order inside B1 starts with `bundle-pull`, the largest spec (9), whose
two pull requirements overlap in topic, so their rules are read together first.

### The extra: `--strict` back in `openspec:check`
**Decision**: One section after both batches (success criterion 4 of the swarm). It edits the
command and its `desc` in `Taskfile.yml`, the commands-table row in `AGENTS.md`, and the
Validation Gates item 4 in `openspec/config.yaml`. CI runs `task check` (`.github/workflows/ci.yml`)
and the CLI pin is already 1.14.1 there, so CI needs no edit.
**Reversibility**: high (a one-line revert).

## The rewrite rules of this repo

From `openspec/config.yaml` rules for specs and the guide:

- Every body keeps RFC 2119 keywords (SHALL or MUST). A clause moved to a scenario THEN line
  keeps its SHALL there.
- A citation (`DESIGN decision N`, `docs-kit C5`, an enhancement id beside a reason in words)
  stays in the body of the block that carries the rule it supports, never in a header, and never
  alone as an enhancement id.
- Requirements describe observable behavior: files, bytes, tags, exit codes, messages. A new
  scenario uses only words and values of the old body; the plain negation of an old condition is
  allowed. A scenario that adds a case not in the old text is a question for the human.
- The loopholes are forbidden in a changed block: no `**Key**:` line (only `**Reason**:` and
  `**Migration**:` in REMOVED), no heading of level 5 or deeper, no fenced block in a body.
- A new requirement name is unique in its spec. Before a split or a c2 the implementer greps the
  repo, outside `openspec/changes/archive/`, for the old header text (including `DESIGN.md`,
  `docs/`, `README.md`, tests and testdata) and fixes or reports every stale reference.
- Every ADDED body is 500 characters or less, has SHALL or MUST, and has at least one scenario
  with a body. A MODIFIED body is also kept to 500 or less, although validate does not check it.
- A touched spec must have a real Purpose. None of the 17 specs has the placeholder today (the
  47 warnings are all length warnings); the implementer re-checks it in the gate.
- Commit types: `docs` for spec text, `ci` or `chore` for the gate. Scope `openspec`. None
  releases `opm-docs`.

## The scenario ledger

For each rewritten requirement the implementing unit adds a table here, in the section of its
batch, before its commit:

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |

- Forms a, b, d: every old name stays under the same requirement, byte for byte.
- Form c2: every old name reappears in exactly one new requirement, with its WHEN and THEN
  unchanged. OpenSpec does not check this, so the reviewer checks it by name.
- A second table lists each clause of the old body that moves (old clause, new place), including
  citations, with the SHALL strength unchanged.

Ledger B1 (section 1). Old scenarios in a new requirement of the same delta are kept byte for byte; the new scenarios added by the rewrite are listed in the delta only. Every SHALL, value, default, exit code and citation of an old body is in the clause rows.

**`bundle-pull`: The pull config names tabs, owners and the trusted signer** (form c2; old body 1012 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Missing owner repository | A tab entry names its signing repository, root and oldest minor, Missing owner repository | no |
| A version names an undeclared project | Pull refuses an inconsistent config before any network call, A version names an undeclared project | no |
| A project in two roles | Pull refuses an inconsistent config before any network call, A project in two roles | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| validate against `#Pull`; registry, signer; top-level keys | The pull config is validated against the embedded schema | 224 |
| tabs fields; tab placement kind | A tab entry names its signing repository, root and oldest minor | 192 |
| docs fields; versions fields; docs placement kind | A docs entry names its signing repository and its site versions | 297 |
| sections fields; section placement kind | A section entry names its signing repository and root | 175 |
| the three refusals, exit 1, before any network call | Pull refuses an inconsistent config before any network call | 259 |

**`bundle-pull`: Nothing is unpacked before its signature verifies** (form c2; old body 681 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A bundle signed by another repository | The signer identity must match the config, A bundle signed by another repository | no |
| Unsigned edge | A bundle is fetched and unpacked only after its signature verifies, Unsigned edge | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| verify; only then fetch by digest, unpack, lint; any failure fails the pull | A bundle is fetched and unpacked only after its signature verifies | 248 |
| sigstore-go, trusted root, timestamps, log entry, digest equal | The Sigstore bundle is verified against the public-good trusted root | 261 |
| issuer, SAN, Source Repository URI, Source Repository Ref | The signer identity must match the config | 255 |

**`bundle-pull`: The unpack layout and lock are fixed** (form b; old body 689 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stable lock | The unpack layout and lock are fixed, Stable lock | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| unpack layout; sweep; lock fields, key order, root; byte-identical lock | The unpack layout and lock are fixed | 445 |
| local entry fields | A local lock entry carries only local data | 101 |
| JSON form, two-space indent, trailing newline, sort order, no timestamps | The lock is serialized in one form | 177 |

**`bundle-pull`: Frozen, offline and local pulls** (form c2; old body 650 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An all-local pull needs no network | A local pull takes a segment from a local tree, An all-local pull needs no network | no |
| Segment mismatch | A local pull takes a segment from a local tree, Segment mismatch | no |
| Offline rebuild from a lock | An offline pull uses only the cache, Offline rebuild from a lock | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| --frozen | A frozen pull fetches exactly the locked digests | 107 |
| --offline | An offline pull uses only the cache | 116 |
| --local; segment mismatch; skip registry resolution; local: true | A local pull takes a segment from a local tree | 320 |
| skip signature verification and the trusted root; still validate, guard, lint | A local entry skips signature verification | 130 |

**`bundle-pull`: Pull handles missing and mismatched versions** (form c2; old body 619 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A project before its first edge push | A tab without an edge tag is skipped and a tab without a build fails, A project before its first edge push | no |
| Placement mismatch | A tab bundle must declare the tab placement, Placement mismatch | no |
| Frozen lock from another config | A frozen pull refuses a lock from another config, Frozen lock from another config | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| skip edge with warning; fail when no minor and no edge | A tab without an edge tag is skipped and a tab without a build fails | 181 |
| placement refusal | A tab bundle must declare the tab placement | 115 |
| config digest refusal | A frozen pull refuses a lock from another config | 99 |
| offline trusted root, TUF expiry | An offline pull verifies with the cached trusted root | 165 |
| 32 MiB limit | A layer over 32 MiB is refused | 84 |

**`bundle-pull`: A site version resolves from its anchor and the anchor's pins** (form c2; old body 646 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Pins choose the docs | Pinned projects resolve at the anchor's pins, Pins choose the docs | no |
| A docs revision of a pinned release | Pinned projects resolve at the anchor's pins, A docs revision of a pinned release | no |
| A pinned release without a bundle | A missing pin or pinned release fails the pull, A pinned release without a bundle | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| anchor first: resolve, verify, unpack, lint, in the tag's line; citation | The anchor of a site version is resolved first | 219 |
| pinned projects at the pin; tags projects at their tag; citation | Pinned projects resolve at the anchor's pins | 262 |
| missing pin or release exit 2; citation | A missing pin or pinned release fails the pull | 253 |

**`bundle-pull`: One version's docs bundles never overlap** (form b; old body 518 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| One page in two bundles | One version's docs bundles never overlap, One page in two bundles | no |
| Two pages that serve one URL | One version's docs bundles never overlap, Two pages that serve one URL | no |
| A page under another bundle's owned directory | One version's docs bundles never overlap, A page under another bundle's owned directory | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| the three overlap failures, exit 2, named parts | One version's docs bundles never overlap | 296 |
| path comparison by Hugo URL; owned page holds its directory | Docs bundle paths compare by the URL Hugo serves | 234 |

**`bundle-pull`: The lock records each site version's bundles** (form b; old body 584 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stable docs lock | The lock records each site version's bundles, Stable docs lock | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| docs list position; entry fields and key order; local entry | The lock records each site version's bundles | 390 |
| sort order | The docs list of the lock is sorted | 104 |
| key omitted; schema id stays | The docs list is omitted without versions | 123 |

**`bundle-pull`: Local and frozen pulls of docs bundles** (form c2; old body 958 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Author preview of the cli reference | A local docs bundle replaces a project of a site version, Author preview of the cli reference | no |
| Frozen lock with a changed role | A frozen pull fetches each docs entry by its digest, Frozen lock with a changed role | no |
| Frozen offline rebuild | A frozen pull fetches each docs entry by its digest, Frozen offline rebuild | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| --local for docs; anchor pins; pinned version equals pin | A local docs bundle replaces a project of a site version | 349 |
| --local naming an unknown version or project | A local docs bundle must name a pulled project | 105 |
| frozen docs entry by digest; site version, project, role; repository; local | A frozen pull fetches each docs entry by its digest | 247 |
| tag and pin differences | A frozen pull refuses a docs entry whose tag or pin changed | 203 |
| configured project absent from the frozen lock | A frozen pull needs every configured docs project | 115 |

**`enhancements-bundle`: A section bundle is edge-only and unversioned** (form c2; old body 577 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A release build of the enhancements bundle | A section bundle is built from main only, A release build of the enhancements bundle | no |
| A markdown source in the section | A section bundle holds exactly one enhancements source, A markdown source in the section | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| edge only; release and revise refused; version omitted; citation | A section bundle is built from main only | 313 |
| page URL, no segment, no edit | A section bundle publishes at its root | 108 |
| exactly one enhancements source; refused outside a section; citation | A section bundle holds exactly one enhancements source | 202 |

**`enhancements-bundle`: The enhancements source turns entries into dialect pages** (form c2; old body 904 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An archived entry keeps its URL | An old-layout entry becomes dialect pages, An archived entry keeps its URL | no |
| Two decision files | The enhancements source reads old-layout entry directories, Two decision files | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| read entry directories; required files; failure exit 2 | The enhancements source reads old-layout entry directories | 368 |
| data/enhancements.json and its fields | The enhancements source writes the entries data file | 241 |
| the pages and their front matter | An old-layout entry becomes dialect pages | 347 |

**`enhancements-bundle`: Enhancement text is cleaned and its links resolved at build** (form c2; old body 989 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A sibling document link | Relative links resolve against the repository path, A sibling document link | no |
| A schema file link | Relative links resolve against the repository path, A schema file link | no |
| A dangling link | A dangling link fails the build, A dangling link | no |
| Raw HTML in prose | Enhancement text is cleaned at build, Raw HTML in prose | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| cleaning rules: comments, first heading, fence tag, shortcode, `<`, link scheme | Enhancement text is cleaned at build | 311 |
| resolution targets and fragment | Relative links resolve against the repository path | 401 |
| link text from file name | A link to a file name reads as the page title | 150 |
| dangling or climbing link, exit 2 | A dangling link fails the build | 140 |

**`enhancements-bundle`: A section page is checked as the site parses it** (form c2; old body 886 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A script link in a quoted reference definition | Section lint reports a destination with a forbidden scheme, A script link in a quoted reference definition | no |
| Raw HTML after an escaped backslash | Section lint reports raw HTML and heading attributes, Raw HTML after an escaped backslash | no |
| A context marker un-indents an HTML line | A section page body is prepared as Hugo prepares it, A context marker un-indents an HTML line | no |
| A script scheme hidden behind escapes | A section page is parsed with goldmark as Hugo configures it, A script scheme hidden behind escapes | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| prepare the body; refuse the marker | A section page body is prepared as Hugo prepares it | 218 |
| goldmark parse; decoding until stable | A section page is parsed with goldmark as Hugo configures it | 240 |
| report raw HTML and heading attribute block | Section lint reports raw HTML and heading attributes | 147 |
| report forbidden schemes and protocol-relative destinations | Section lint reports a destination with a forbidden scheme | 258 |
| build, lint --bundle and pull apply it; the check keeps markup and script URLs off the site | Every command applies the section check | 217 |

**`enhancements-bundle`: The link policy links enhancement references** (form c2; old body 503 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| One enhancement | The link policy turns enhancement ids into links, One enhancement | no |
| Two enhancements | The link policy turns enhancement ids into links, Two enhancements | no |
| A citation form | The link policy removes decision citations, A citation form | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| enhancement NNNN and the list form; crd and go-api show the same links | The link policy turns enhancement ids into links | 238 |
| decision citation removed without a link | The link policy removes decision citations | 164 |
| strip leaves the reference except in a lone parenthesis | The strip policy leaves other enhancement references as written | 99 |

**`enhancements-bundle`: The strip policy removes a lone enhancement citation** (form b; old body 648 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| End of a sentence | The strip policy removes a lone enhancement citation, End of a sentence | no |
| Middle of a sentence | The strip policy removes a lone enhancement citation, Middle of a sentence | no |
| A list of ids | The strip policy removes a lone enhancement citation, A list of ids | no |
| A parenthesis with other words | The strip policy removes a lone enhancement citation, A parenthesis with other words | no |
| A code span | The strip policy removes a lone enhancement citation, A code span | no |
| Fewer than four digits | The strip policy removes a lone enhancement citation, Fewer than four digits | no |
| The link policy | The strip policy removes a lone enhancement citation, The link policy | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| strip removal and what must stay | The strip policy removes a lone enhancement citation | 428 |
| link policy keeps output; code spans by CommonMark backtick runs | Code spans are found by CommonMark backtick runs under both policies | 250 |

**`crd-extractor`: The crd source reads controller-gen CRDs and samples** (form c2; old body 920 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A required spec field | The crd model holds the field tables and rules of each kind, A required spec field | no |
| A second sample file of one kind | The crd source reads the sample of each kind, A second sample file of one kind | no |
| A fixture sample is hidden | The crd source reads the sample of each kind, A fixture sample is hidden | no |
| Scaffold labels stripped | The crd source reads the sample of each kind, Scaffold labels stripped | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| read CRDs; data/crd.json and schema; per-kind facts; reconciledBy; kind order | The crd source writes a model of each kind | 410 |
| field tables; validation rules | The crd model holds the field tables and rules of each kind | 230 |
| sample selection, hide, strip labels; sample text in the model | The crd source reads the sample of each kind | 374 |

**`crd-extractor`: The crd page is one completable page** (form c2; old body 882 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Authored intro kept | The crd page is completable, Authored intro kept | no |
| Citation linked | The crd page applies the citation policy to descriptions, Citation linked | no |
| Decision citation removed | The crd page applies the citation policy to descriptions, Decision citation removed | no |
| Parity with crdref | The served-by sentence names no product, Parity with crdref | no |
| The served-by sentence names no product | The served-by sentence names no product, The served-by sentence names no product | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| page layout; one page at `page`; entry parts and their order | The crd page holds one entry per kind | 419 |
| citation policy applied to field descriptions on the page (rule stated by the two scenarios and by the model requirement) | The crd page applies the citation policy to descriptions | 198 |
| served-by sentence; entries match crdref apart from it | The served-by sentence names no product | 222 |
| completable page; authored front matter and intro; configured title, description, weight | The crd page is completable | 211 |

**`crd-extractor`: CRD text never reaches the page as markup** (form c2; old body 663 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A sample cannot close its fence | The crd renderer fences a sample with a longer fence, A sample cannot close its fence | no |
| A description cannot forge a link | The crd renderer escapes prose and links only enhancement references, A description cannot forge a link | no |
| An unread field is refused | The crd source refuses what its page cannot show, An unread field is refused | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| untrusted strings; the refusals | The crd source refuses what its page cannot show | 456 |
| escape prose; links only from the link policy | The crd renderer escapes prose and links only enhancement references | 127 |
| fence length | The crd renderer fences a sample with a longer fence | 91 |

**`crd-extractor`: The crd source renders a section of kind pages** (form c2; old body 935 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Operator section | The crd section writes an index and one page per kind, Operator section | no |
| Authored index completed | The crd section index lists the kinds, Authored index completed | no |
| The index links each kind page | The crd section index lists the kinds, The index links each kind page | no |
| Both page and section | A crd source takes exactly one of page and section, Both page and section | no |
| Page layout unchanged | A crd source takes exactly one of page and section, Page layout unchanged | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| exactly one of page and section; page unchanged | A crd source takes exactly one of page and section | 210 |
| section files; equal lower-cased names | The crd section writes an index and one page per kind | 192 |
| index content; completable | The crd section index lists the kinds | 196 |
| kind page title, weight, type, entry | A kind page holds its entry one level up | 217 |
| layout and page path in the model | The crd model records the layout | 116 |

**`cue-definitions-extractor`: The cue-definitions source reads a CUE package's exported definitions** (form b; old body 541 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Uses and used-by | The cue-definitions source reads a CUE package's exported definitions, Uses and used-by | no |
| Pins files skipped | The cue-definitions source reads a CUE package's exported definitions, Pins files skipped | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| parse, skip, take definitions and their parts | The cue-definitions source reads a CUE package's exported definitions | 441 |
| data file and schema | The cue-definitions source writes its model file | 152 |

**`cue-definitions-extractor`: Every exported definition is placed or excluded** (form c2; old body 947 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A new definition | A config in the source tree must place or exclude every definition, A new definition | no |
| Backfill of an older tag | A config from outside the source tree gives warnings, Backfill of an older tag | no |
| A definition without a doc comment in a backfill | A config from outside the source tree gives warnings, A definition without a doc comment in a backfill | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| exactly one page or exclude with a reason | Each exported definition is placed once or excluded with a reason | 99 |
| in-tree failures: unplaced, two pages, undeclared placed, undeclared excluded, both | A config in the source tree must place or exclude every definition | 440 |
| in-tree failures: no summary, anchor collision | A config in the source tree must give each definition a summary and an anchor | 409 |
| backfill warnings and their handling | A config from outside the source tree gives warnings | 334 |

**`cue-definitions-extractor`: Definitions render as a section index and one page per group** (form c2; old body 656 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Link to another group | The renderer writes a section index and one page per group, Link to another group | no |
| Parity with refgen | The definition pages match core's refgen output, Parity with refgen | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| index and group pages; headings, anchors, links | The renderer writes a section index and one page per group | 385 |
| owned path; refused outside docs placement | The definitions section lies under an owned path | 144 |
| refgen parity; no marker comment | The definition pages match core's refgen output | 125 |

**`opm-docs-cli`: docs-kit.cue configures the bundles a repository builds** (form c2; old body 859 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Package clause ignored | docs-kit.cue is validated before any extraction, Package clause ignored | no |
| A misspelled key is refused | docs-kit.cue is validated before any extraction, A misspelled key is refused | no |
| An unknown source kind | docs-kit.cue holds bundles with sources, An unknown source kind | no |
| Include selects one page | A markdown source selects files by include and exclude, Include selects one page | no |
| Exclude a committed generated directory | A markdown source selects files by include and exclude, Exclude a committed generated directory | no |
| Citations linked | Extractor sources accept a citation policy, Citations linked | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| read and validate; package clause | docs-kit.cue is validated before any extraction | 200 |
| bundles and their fields; at most one source per extractor kind | docs-kit.cue holds bundles with sources | 322 |
| citations option | Extractor sources accept a citation policy | 86 |
| include and exclude | A markdown source selects files by include and exclude | 254 |

**`opm-docs-cli`: build writes one bundle tree per project** (form c2; old body 1146 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Release build from a tag | build --release derives the version from the tag, Release build from a tag | no |
| Release older than docs-kit adoption | build --source resolves sources against one directory, Release older than docs-kit adoption | no |
| A missing markdown dir in a normal build | A missing markdown dir depends on where the config came from, A missing markdown dir in a normal build | no |
| Wrong prefix | build --release derives the version from the tag, Wrong prefix | no |
| Tag and catalog version differ | A release build agrees with the catalog version, Tag and catalog version differ | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| out tree per project; edge bundle without --release | build writes one bundle tree per selected project | 186 |
| version from tag prefix; refusals | build --release derives the version from the tag | 178 |
| tag and catalog version equal; revise too | A release build agrees with the catalog version | 260 |
| --source | build --source resolves sources against one directory | 288 |
| missing markdown dir | A missing markdown dir depends on where the config came from | 148 |
| source.dirty; push refuses | A build from a dirty work tree is recorded and not pushed | 123 |

**`opm-docs-cli`: serve previews a repository's bundles** (form c2; old body 1027 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Preview an authored page | serve rebuilds a bundle when its sources change, Preview an authored page | no |
| A failed rebuild keeps the last good page | serve rebuilds a bundle when its sources change, A failed rebuild keeps the last good page | no |
| No hugo | serve serves the bundles on its embedded Hugo site, No hugo | no |
| Preview in the real site | serve --site previews in the real site, Preview in the real site | no |
| Docs bundle without a site version | serve --site previews in the real site, Docs bundle without a site version | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| edge bundles in a temp dir; build's extractors and lint | serve builds edge bundles into a temporary directory | 270 |
| embedded site; hugo; 127.0.0.1; mounts | serve serves the bundles on its embedded Hugo site | 343 |
| watch and rebuild; failure keeps last good bundle | serve rebuilds a bundle when its sources change | 248 |
| --site | serve --site previews in the real site | 260 |

**`publish-workflow`: One reusable workflow with three modes** (form c2; old body 512 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Pull-request check | publish.yml is a reusable workflow with fixed inputs and outputs, Pull-request check | no |
| Publishing from a release branch is refused | Publishing modes run only from main, Publishing from a release branch is refused | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| inputs, outputs, no secrets, github.token; citation | publish.yml is a reusable workflow with fixed inputs and outputs | 366 |
| unknown mode fails; citation | A mode outside the three fails the workflow | 102 |
| main-only guard; citation | Publishing modes run only from main | 128 |

**`publish-workflow`: Publishing modes sign before tags move** (form c2; old body 702 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Edge publish | Publishing modes sign the pushed digest before tags move, Edge publish | no |
| A burst of main pushes does not drop a release | Edge and release publishes use separate concurrency groups, A burst of main pushes does not drop a release | no |
| Signing fails | Publishing modes sign the pushed digest before tags move, Signing fails | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| build, push, sign, promote order; caller permissions | Publishing modes sign the pushed digest before tags move | 270 |
| concurrency groups | Edge and release publishes use separate concurrency groups | 431 |

**`publish-workflow`: Build and publish run in separate jobs** (form c2; old body 557 characters)

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Two revisions dispatched together | Concurrency groups cover both jobs of a run, Two revisions dispatched together | no |
| Repository code never meets the signing token | The build job holds no signing permission, Repository code never meets the signing token | no |

| Old clause | New place (requirement) | Body length |
| --- | --- | --- |
| build job permissions | The build job holds no signing permission | 136 |
| second job; no checkout; artifact | The publish job takes the bundle from a workflow artifact | 165 |
| interface unchanged except setup-go | The job split leaves the workflow interface unchanged | 149 |
| workflow-level concurrency groups | Concurrency groups cover both jobs of a run | 145 |


Ledger B2: filled in during section 2.

## The gate

Run in a scratch copy of `openspec/`, never in the worktree. Set before the run:

- `WT` = `/var/home/emil/dev/open-platform-model/agent-swarm/openspec-requirement-split/worktrees/docs-kit-split-long-requirements`
- `CH` = `split-long-requirements`
- `S` = a new folder under the session scratchpad (for example `.../scratchpad/T4.3-gate`)
- `N` = 28 after section 1, 47 after section 2; `LAST=1` only in section 3
- CLI: `openspec` 1.14.1 (the pin of `Taskfile.yml`), `jq` installed.

The block is the one of guide section 6 unchanged (`errs()`, base copy from `git archive main`,
`validate $CH --strict`, `archive $CH --yes`, `validate --all --strict`, the "Archive would
refuse" check, the new-ERROR set check, then the `N` check or the `--specs --strict` check). It
passes only on `GATE PASS` with exit 0. Because this repo has no other open change, in section 3
`validate --all --strict` also exits 0. Count by hand with
`openspec validate --all --strict --no-interactive 2>&1 | grep -c WARNING`; it prints 47 on
`main` and must print 0 after section 2 in the scratch copy.

The review checklist of the guide (scope, meaning kept, scenario text byte equal, no invented
rule, ledger, no loophole, lengths, forms, one behavior each, repo rules, references, Purpose,
gate rerun, commits) applies to each section.

## Durable decisions

- `--strict` is on in `task openspec:check`: lands in `Taskfile.yml`, `AGENTS.md` and
  `openspec/config.yaml` (section 3).
- The ledger and the choice of form per requirement: stay with the change.
