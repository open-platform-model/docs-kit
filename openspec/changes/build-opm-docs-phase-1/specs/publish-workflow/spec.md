## Purpose

The reusable GitHub workflow that a repository calls to check, build, push, sign and promote its docs bundles.

## ADDED Requirements

### Requirement: One reusable workflow with three modes
docs-kit SHALL provide `.github/workflows/publish.yml`, callable with `workflow_call`, taking the inputs `project` (required), `mode` (`check`, `edge` or `release`, required), `tag` and `cue-registry`, and returning the outputs `digest` and `tag`, as `design.md` C5 fixes. It SHALL declare no secrets and use `github.token`. Every mode except `check` SHALL fail before building unless `github.ref` is `refs/heads/main`. A `mode` outside these three SHALL fail naming it. Source: DESIGN decisions 5 and 9.

#### Scenario: Pull-request check
- **WHEN** a caller runs the workflow with `mode: check` on a pull request
- **THEN** it runs `opm-docs check --project <project>`, pushes nothing and needs only `contents: read` and `packages: read`

#### Scenario: Publishing from a release branch is refused
- **WHEN** a caller runs `mode: edge` on `refs/heads/release/opm-v4.4`
- **THEN** the workflow fails before building, naming the ref and that bundles publish only from `main`

### Requirement: The tool version follows the workflow ref
`publish.yml` SHALL install `opm-docs` from the docs-kit release named by the literal `OPM_DOCS_VERSION` in the file, which release-please SHALL update in every release PR, and SHALL verify the archive's SHA-256 against that release's `checksums.txt` before installing it.

#### Scenario: Pinned ref, matching tool
- **WHEN** a caller uses `publish.yml@v0.2.0`
- **THEN** the job installs `opm-docs` 0.2.0 and the bundle's `dev.opmodel.docs.tool` annotation reads `0.2.0`

### Requirement: Publishing modes sign before tags move
In `edge` and `release` modes the workflow SHALL build, `push`, sign the pushed digest with a pinned cosign v3 keyless (`--new-bundle-format=true`, the digest, never a tag), and only then run `promote`. The caller job SHALL grant `packages: write` and `id-token: write`. Jobs for one repository and project SHALL be serialized with a concurrency group that never cancels a running publish.

#### Scenario: Edge publish
- **WHEN** a caller runs `mode: edge` on a push to `main`
- **THEN** the registry gains a signed manifest for the pushed commit and `edge` points at it, and no other tag moved

#### Scenario: Signing fails
- **WHEN** the cosign step fails after `push`
- **THEN** the job fails, `promote` does not run, and no moving tag points at the unsigned digest

### Requirement: Release mode publishes a release once
In `release` mode the workflow SHALL check out the tag at `src/` with full history beside `main`, build with `--release <tag> --source src`, and push the full tag `<version>.0`. A caller SHALL run it from the job that runs release-please, gated on that package's release, or by `workflow_dispatch` for a release that has no bundle yet.

#### Scenario: Backfill the current release
- **WHEN** catalog_opm dispatches `mode: release` with `tag: opm-v4.4.5`, a tag cut before catalog_opm had `docs-kit.cue`
- **THEN** the workflow builds from that tag with `main`'s `docs-kit.cue` and publishes `4.4.5.0`, then moves `4.4.5`, `4.4` and `4`
