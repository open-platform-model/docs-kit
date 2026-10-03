## MODIFIED Requirements

### Requirement: The pull config names tabs, owners and the trusted signer
`opm-docs pull --config <file>` SHALL validate the file against the embedded `#Pull` schema: a `registry`, a `signer` (issuer, workflow URL, allowed ref globs), `tabs` keyed by project, each with the `repo` allowed to sign it, its URL `root`, the oldest minor `from` and whether `edge` is shown, `docs` keyed by project, each with the `repo` allowed to sign it, `versions` keyed by site version (`v<MAJOR>.<MINOR>`), each with an `anchor` (project and tag), the `pinned` projects and the projects pulled by their own `tags`, and `sections` keyed by project, each with the `repo` allowed to sign it and its `root`. Before any network call it SHALL refuse, with exit 1 naming the version and project, a project in a version that is not a key of `docs`, a project named twice in one version, and a project that appears under more than one of `tabs`, `docs` and `sections`. A tab bundle SHALL have placement `kind: "tab"`, a docs bundle `kind: "docs"` and a section bundle `kind: "section"` with the configured root.

#### Scenario: Missing owner repository
- **WHEN** a tab entry has no `repo`
- **THEN** `pull` exits 1 naming the project and the missing field, before any network call

#### Scenario: A version names an undeclared project
- **WHEN** `versions."v1.0".pinned` lists `core` and `docs` has no `core`
- **THEN** `pull` exits 1 naming `v1.0` and `core`, before any network call

#### Scenario: A project in two roles
- **WHEN** `enhancements` is both under `docs` and under `sections`
- **THEN** `pull` exits 1 naming `enhancements`, before any network call

## ADDED Requirements

### Requirement: Pull takes a section bundle from its edge tag
For each section, `pull` SHALL resolve only the `edge` tag, verify it (C9, with the section's `repo`), require placement `kind: "section"` with the configured root, unpack and lint it into `<out>/<project>/edge/` as a tab segment, and lock it in `bundles` with `root` the section's root and `segment` `edge`. A section with no `edge` tag SHALL fail the pull naming the project, since the section would be empty.

#### Scenario: The enhancements section
- **WHEN** `sections: enhancements: {repo: "open-platform-model/enhancements", root: "/enhancements/"}` and `docs/enhancements` has `edge`
- **THEN** `pull` unpacks `<out>/enhancements/edge/` and the lock has an entry with root `/enhancements/` and segment `edge`
