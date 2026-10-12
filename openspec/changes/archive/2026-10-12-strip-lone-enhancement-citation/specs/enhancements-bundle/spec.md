## ADDED Requirements

### Requirement: The strip policy removes a lone enhancement citation
Under the `strip` citation policy, prose outside a code span SHALL lose each parenthesis whose whole content is `enhancement NNNN` (a capital first letter too) or `enhancements NNNN and NNNN` (ids joined by commas and `and`), together with the one space before it, so the sentence stays whole. A parenthesis that holds other words, a code span, a comment inside a spec block and a Markdown link destination MUST stay as written. The `link` policy MUST keep its output, except that code spans are found by CommonMark backtick runs under both policies: a double-backtick span that holds a backtick is one span, and an unclosed backtick is plain text.

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

## MODIFIED Requirements

### Requirement: The link policy links enhancement references
Under the `link` citation policy, prose outside a code span SHALL turn `enhancement NNNN` into a link to `/enhancements/NNNN/` and each id of `enhancements NNNN and NNNN` into one. `link` SHALL remove a decision citation (`NNNN:Dn` and its variants) without a link, as `strip` does, because an entry of the OpenSpec layout has no decisions page; `strip` SHALL leave `enhancement NNNN` as written, except in a parenthesis that holds nothing else. The `crd` and `go-api` sources SHALL show the same links.

#### Scenario: One enhancement
- **WHEN** a comment reads `(enhancement 0013)` under `link`
- **THEN** the page reads `(` + a link `enhancement 0013` to `/enhancements/0013/` + `)`

#### Scenario: Two enhancements
- **WHEN** a comment reads `(enhancements 0010 and 0026)` under `link`
- **THEN** `0010` and `0026` each link to their entry page

#### Scenario: A citation form
- **WHEN** a comment reads an enhancement id followed by a decision number, or `enhancement 0010 experiment 3`, under `link`
- **THEN** the decision citation and the experiment pointer are removed, with no link to `/enhancements/0010/`
