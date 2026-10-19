## MODIFIED Requirements

### Requirement: A bundle's doc model is checked before it is compared
Before comparing, `pull` SHALL check every member of each segment's doc model for an `fqn`, a `kind` of `resource`, `trait` or `blueprint`, a `page` matching `^(resources|traits|blueprints)/[a-z0-9]+(-[a-z0-9]+)*$`, and a field `presence` of `regular`, `optional` or `required`, and SHALL fail a member that breaks one naming the segment and its data file, not as a tool bug.

#### Scenario: A bad page in a bundle
- **WHEN** the 4.6 bundle's doc model has a member whose page is `../x`
- **THEN** the pull fails naming `catalog-opm 4.6 data/catalog.json` and the member, and does not say to report it against opm-docs

## ADDED Requirements

### Requirement: The history schema constrains presence changes and removed pages
The three values and the pattern are those of "A bundle's doc model is checked before it is compared". The schema SHALL constrain a `presence` change's `from` and `to` to those three values and a removed member's `page` to that pattern.

#### Scenario: The schema constrains a presence change
- **WHEN** the `#History` schema validates a `presence` change and a removed member
- **THEN** the `from` and `to` of the change are `regular`, `optional` or `required`, and the `page` of the member matches `^(resources|traits|blueprints)/[a-z0-9]+(-[a-z0-9]+)*$`
