## ADDED Requirements

### Requirement: The enhancements graph is a link target
Dialect 1 SHALL accept the link destination `/enhancements/graph/` with an optional fragment, and the conformance fixture set SHALL gain a fixture for it, which opmodel.dev copies with its shell lint change in the PR that bumps its pinned `opm-docs`.

#### Scenario: A docs page links the graph
- **WHEN** a page links `/enhancements/graph/`
- **THEN** the lint reports nothing for that line
