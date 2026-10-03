## MODIFIED Requirements

### Requirement: Text is escaped for Hugo and Markdown
Rendered prose SHALL escape `\ < > * _ [ ] | { }` outside code spans, so no text opens raw HTML, a Hugo shortcode or a heading attribute block; table cells SHALL escape every `|`; a rendered page that still contains `{{<` or `{{%` SHALL fail the build naming the page.

#### Scenario: A shortcode-looking doc comment
- **WHEN** a member's note contains `{{< foo >}}` outside a code span
- **THEN** the page holds `\{\{\< foo \>\}\}` and passes the dialect lint

#### Scenario: A note that reads as a heading with attributes
- **WHEN** a member's note paragraph is `## Install {.hx:fixed style="position:fixed"}`
- **THEN** the page holds `## Install \{.hx:fixed style="position:fixed"\}` and passes the markup check
