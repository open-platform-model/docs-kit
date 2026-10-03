## ADDED Requirements

### Requirement: Bundle mode checks every page's markup as the site parses it
Bundle-mode lint SHALL run the markup check (the one a section page gets, `enhancements-bundle`) on every page `manifest.json` lists, in every bundle kind, so `build`, `lint --bundle` and `pull` apply it, `pull` to bundles already published. A page with `generated: true`, and every page of a section bundle, SHALL get the full check. A page with `generated: false` in a tab or docs bundle SHALL get the full check except that a raw HTML node (inline or a block) made only of HTML comments separated by whitespace SHALL pass when each comment opens `<!--`, is not closed abruptly (`<!-->`, `<!--->`) and ends, inside the node, at its first `-->` or `--!>`. Every violation SHALL print as `<file>:<line>: <message>` and the command SHALL exit 2. Docs mode SHALL not run the check, so the conformance fixture set is unchanged.

#### Scenario: Raw HTML in a generated page
- **WHEN** a bundle's `manifest.json` lists `reference/crd.md` with `generated: true` and that page holds `<details open ontoggle=alert(1)>` outside code
- **THEN** `opm-docs lint --bundle` reports the page and the line and exits 2

#### Scenario: A comment in a generated page
- **WHEN** a `generated: true` page holds `<!-- note -->`
- **THEN** the lint reports the page and the line and exits 2

#### Scenario: A writer's note in an authored page
- **WHEN** a `generated: false` page of a docs bundle holds a line `<!-- Check against: core/src/x.cue -->` and an inline `| a | b <!-- c --> |`
- **THEN** the lint reports nothing for either line

#### Scenario: A comment a browser closes early
- **WHEN** a `generated: false` page holds `<!-- a --!> <svg onload=alert(1)> -->` or `<!--> <svg onload=alert(1)> -->`
- **THEN** the lint reports the page and the line and exits 2

#### Scenario: Markup after a comment
- **WHEN** a `generated: false` page holds `<!-- note --> <b>x</b>` on one line
- **THEN** the lint reports the page and the line and exits 2

#### Scenario: A heading attribute block in an authored page
- **WHEN** a `generated: false` page holds `## Install {.hx:fixed}`
- **THEN** the lint reports the page and the line and exits 2

#### Scenario: A published bundle with writer's notes still pulls
- **WHEN** `pull` fetches a docs bundle whose authored pages carry only well-formed HTML comments and whose generated pages carry no raw HTML
- **THEN** it unpacks the bundle and records it in the lock

#### Scenario: A docs-mode tree is not checked
- **WHEN** `opm-docs lint docs/site` runs over a tree whose page holds `<!-- note -->`
- **THEN** the lint reports nothing for that line
