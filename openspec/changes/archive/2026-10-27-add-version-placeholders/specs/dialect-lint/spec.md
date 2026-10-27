## ADDED Requirements

### Requirement: The lint refuses an unknown version placeholder
The lint SHALL scan every line of a page, front matter and code fences included, for the text `{{opm:`. Each occurrence SHALL be a placeholder `{{opm:<name>.version}}` whose name is on the list of the `version-placeholders` capability; any other occurrence SHALL be a violation. The message SHALL name the placeholder, say what is wrong (unknown name, unknown field, or not of the form `{{opm:<name>.version}}`) and, for an unknown name, list the known names. The dialect version SHALL stay `1`.

#### Scenario: A misspelt name
- **WHEN** a page line 12 holds `{{opm:opm-controler.version}}`
- **THEN** the lint prints `<file>:12: placeholder "{{opm:opm-controler.version}}": unknown name "opm-controler"; known names: cli, core, library, opm-controller, opm-controller-module, catalog-opm` and exits 2

#### Scenario: An unknown field
- **WHEN** a page holds `{{opm:cli.tag}}`
- **THEN** the lint reports an unknown field `tag` and says to write `{{opm:cli.version}}`

#### Scenario: A placeholder that is not closed
- **WHEN** a page holds `{{opm:cli.version` with no closing braces
- **THEN** the lint reports that the placeholder is not closed or not of the form `{{opm:<name>.version}}`

#### Scenario: Known names in prose, in a code fence and in a link
- **WHEN** a page holds `{{opm:cli.version}}` in a sentence, inside a fenced `bash` block, and in the link `[notes](https://github.com/open-platform-model/cli/releases/tag/v{{opm:cli.version}})`
- **THEN** the lint reports nothing for those lines

#### Scenario: A shown shortcode is unaffected
- **WHEN** a page holds `{{</* opm/figure */>}}`
- **THEN** the lint reports nothing, as before
