## ADDED Requirements

### Requirement: Revision mode can refresh units
`publish.yml` SHALL accept a boolean input `refresh-units` (default `false`). In `mode: revision`, with `refresh-units: true` and an empty `fix`, it SHALL run `opm-docs revise --project <project> --tag <tag> --refresh-units` in place of `--fix`, then push, sign and promote as for a fix. With `refresh-units` true and a non-empty `fix`, or with both empty, it SHALL fail before building, naming the inputs. Without `refresh-units`, the `fix` rules of the revision mode are unchanged.

#### Scenario: A module release republishes the controller docs
- **WHEN** opm-controller dispatches `mode: revision`, `tag: v1.0.0-beta.10`, `refresh-units: true` and `fix` empty
- **THEN** the next revision is published, signed and promoted with the refreshed `units`

#### Scenario: Both inputs
- **WHEN** `refresh-units` is true and `fix` is a commit
- **THEN** the job fails before building, saying the two are exclusive
