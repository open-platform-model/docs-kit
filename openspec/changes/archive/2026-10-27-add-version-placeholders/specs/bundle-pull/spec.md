## ADDED Requirements

### Requirement: Pull fills placeholders per site version
After a site version's docs bundles are unpacked, linted and checked for overlap, and before any tree is swapped in, `pull` SHALL replace each well-formed `{{opm:<name>.version}}` in the pages listed by its docs bundles' `manifest.json` by the value the `version-placeholders` capability defines. The replace SHALL be a plain text replace of the file, front matter and code included.

#### Scenario: A page in a code block and a link
- **WHEN** a pulled page holds `v{{opm:cli.version}}` in a fenced block and in a link destination
- **THEN** both become the `cli` bundle's version with a `v` before it, and the rest of the file is unchanged

### Requirement: The fill touches pages only and leaves the lock alone
The fill SHALL touch no file but the listed pages of the docs bundles of site versions. A `--frozen`, `--offline` or `--local` pull SHALL fill the same way. The lock SHALL NOT change: it records the verified digests.

#### Scenario: A frozen pull
- **WHEN** a `--frozen` pull takes the digests of a lock
- **THEN** the filled pages equal those of the pull that wrote the lock

#### Scenario: Data files stay as they are
- **WHEN** a bundle's `data/` file holds the text `{{opm:cli.version}}`
- **THEN** the pull leaves that file unchanged

### Requirement: A refused fill swaps nothing in
When the fill refuses, `pull` SHALL exit 2 and leave the previous trees and the previous lock as they were.

#### Scenario: A refused fill swaps nothing in
- **WHEN** a page names a value the site version cannot supply
- **THEN** the pull exits 2 and the previous `_versions/` trees and lock are unchanged

### Requirement: Pull refuses a leftover placeholder
After the fill, `pull` SHALL scan the pages of every staged tree for `{{opm:` and SHALL exit 2 naming the tree and page when one is left. See the `version-placeholders` capability.

#### Scenario: A bundle outside a site version
- **WHEN** a section bundle page holds `{{opm:cli.version}}`
- **THEN** the pull exits 2 naming the page
