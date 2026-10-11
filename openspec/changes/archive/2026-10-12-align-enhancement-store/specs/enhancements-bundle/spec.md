## MODIFIED Requirements

### Requirement: The link policy links enhancement references
Under the `link` citation policy, prose outside a code span SHALL turn `enhancement NNNN` into a link to `/enhancements/NNNN/` and each id of `enhancements NNNN and NNNN` into one. `link` SHALL remove a decision citation (`NNNN:Dn` and its variants) without a link, as `strip` does, because an entry of the OpenSpec layout has no decisions page; `strip` SHALL leave `enhancement NNNN` as written. The `crd` and `go-api` sources SHALL show the same links.

#### Scenario: One enhancement
- **WHEN** a comment reads `(enhancement 0013)` under `link`
- **THEN** the page reads `(` + a link `enhancement 0013` to `/enhancements/0013/` + `)`

#### Scenario: Two enhancements
- **WHEN** a comment reads `(enhancements 0010 and 0026)` under `link`
- **THEN** `0010` and `0026` each link to their entry page

#### Scenario: A citation form
- **WHEN** a comment reads an enhancement id followed by a decision number, or `enhancement 0010 experiment 3`, under `link`
- **THEN** the decision citation and the experiment pointer are removed, with no link to `/enhancements/0010/`
