## ADDED Requirements

### Requirement: The workflow has a revision mode
`publish.yml` SHALL accept `mode: revision` with the inputs `tag` (the release's git tag) and `fix` (a 40-hex commit on `main`). In that mode it SHALL check out `main` with full history, run `opm-docs revise --project <project> --tag <tag> --fix <fix>`, then `push`, sign the pushed digest with cosign keyless and run `promote`, under the same `refs/heads/main` guard, permissions and concurrency group as `release`. Source: DESIGN decision 6.

#### Scenario: A docs fix reaches a release
- **WHEN** catalog_opm dispatches `mode: revision`, `tag: opm-v4.4.5`, `fix: <sha of a comment-only fix on main>` and 4.4.5.0 is the newest revision
- **THEN** `4.4.5.1` is published and signed, and `4.4.5` points at it

#### Scenario: A code fix is refused
- **WHEN** the fix commit changes a CUE value
- **THEN** the job fails at `revise`, naming the file, and nothing is pushed
