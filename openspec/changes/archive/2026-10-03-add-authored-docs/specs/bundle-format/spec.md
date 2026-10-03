## ADDED Requirements

### Requirement: Pages may carry their edit path
`#Page` SHALL accept an optional `edit`, a non-empty repository-relative path, and `pull` SHALL accept a bundle with or without it.

#### Scenario: An older bundle
- **WHEN** a bundle built before `edit` existed is pulled
- **THEN** it validates and its pages have no `edit`
