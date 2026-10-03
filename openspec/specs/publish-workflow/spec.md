# publish-workflow Specification

## Purpose
The reusable GitHub workflow that a repository calls to check, build, push, sign and promote its docs bundles.

## Requirements

### Requirement: One reusable workflow with three modes
docs-kit SHALL provide `.github/workflows/publish.yml`, callable with `workflow_call`, taking the inputs `project` (required), `mode` (`check`, `edge` or `release`, required), `tag` and `cue-registry`, and returning the outputs `digest` and `tag`, as `docs/contracts.md` C5 fixes. It SHALL declare no secrets and use `github.token`. Every mode except `check` SHALL fail before building unless `github.ref` is `refs/heads/main`. A `mode` outside these three SHALL fail naming it. Source: DESIGN decisions 5 and 9.

#### Scenario: Pull-request check
- **WHEN** a caller runs the workflow with `mode: check` on a pull request
- **THEN** it runs `opm-docs check --project <project>`, pushes nothing and needs only `contents: read` and `packages: read`

#### Scenario: Publishing from a release branch is refused
- **WHEN** a caller runs `mode: edge` on `refs/heads/release/opm-v4.4`
- **THEN** the workflow fails before building, naming the ref and that bundles publish only from `main`

### Requirement: The tool version is the caller's pin
`publish.yml` SHALL carry no version literal. It SHALL read the caller's repo-root `.opm-docs-version` from the checked-out caller tree (one line, a docs-kit release tag such as `v0.1.0`), SHALL fail before building, naming the file, when it is missing or malformed, and SHALL install that release's `opm-docs` only after verifying the archive's SHA-256 against that release's `checksums.txt`. A caller SHALL move `.opm-docs-version` and its `publish.yml@` ref together. Owner decision, 2026-10-02.

#### Scenario: Pinned release, matching tool
- **WHEN** a caller's `.opm-docs-version` reads `v0.2.0`
- **THEN** the job installs `opm-docs` 0.2.0 and the bundle's `dev.opmodel.docs.tool` annotation reads `0.2.0`

#### Scenario: Missing pin
- **WHEN** the caller's tree has no `.opm-docs-version`
- **THEN** the job fails before building, naming `.opm-docs-version` and the form it must take

### Requirement: Publishing modes sign before tags move
In `edge` and `release` modes the workflow SHALL build, `push`, sign the pushed digest with a pinned cosign v3 keyless (`--new-bundle-format=true`, the digest, never a tag), and only then run `promote`. The caller job SHALL grant `packages: write` and `id-token: write`. An `edge` job SHALL run in the concurrency group `docs-edge-${{ inputs.project }}` with `cancel-in-progress: true`, and a `release` job in `docs-release-${{ inputs.project }}-${{ inputs.tag }}` with `cancel-in-progress: false` (keyed on inputs, which are known when the group is evaluated), so that no edge run can cancel a pending or running release publish (GitHub keeps one pending run per group and cancels older pending ones).

#### Scenario: Edge publish
- **WHEN** a caller runs `mode: edge` on a push to `main`
- **THEN** the registry gains a signed manifest for the pushed commit and `edge` points at it, and no other tag moved

#### Scenario: A burst of main pushes does not drop a release
- **WHEN** a release publish of 4.4.6 is pending and three pushes to `main` start edge publishes of the same project
- **THEN** the edge runs share their own group, and the 4.4.6 release publish runs to completion

#### Scenario: Signing fails
- **WHEN** the cosign step fails after `push`
- **THEN** the job fails, `promote` does not run, and no moving tag points at the unsigned digest

### Requirement: Release mode publishes a release once
In `release` mode the workflow SHALL check out the tag at `src/` with full history beside `main`, build with `--release <tag> --source src`, and push the full tag `<version>.0`. A caller SHALL run it from the job that runs release-please, gated on that package's release, or by `workflow_dispatch` for a release that has no bundle yet.

#### Scenario: Backfill the current release
- **WHEN** catalog_opm dispatches `mode: release` with `tag: opm-v4.4.5`, a tag cut before catalog_opm had `docs-kit.cue`
- **THEN** the workflow builds from that tag with `main`'s `docs-kit.cue` and publishes `4.4.5.0`, then moves `4.4.5`, `4.4` and `4`

### Requirement: Only the publish job logs in to GHCR
The build job SHALL NOT log in to any registry, and its checkouts SHALL NOT persist credentials, so no credential is on disk where a repository command could read it; the extractors resolve CUE dependencies and `revise` reads bundles anonymously from public GHCR packages. The publish job SHALL log in to `ghcr.io` with `github.token` before `push` and `promote`. The `check` mode SHALL need no permission beyond `contents: read` and `packages: read`.

#### Scenario: Check resolves core from GHCR
- **WHEN** `mode: check` runs on a pull request with `packages: read`
- **THEN** the build resolves `opmodel.dev/core@v2` from GHCR anonymously, and no step needs a secret

### Requirement: Sources of a backfill resolve against the release tree
In `release` mode, when the release tree has no `docs-kit.cue`, the workflow SHALL build with `main`'s `docs-kit.cue` while every source in it resolves against the release tree at `src/`. A `markdown` dir missing from that release tree SHALL yield no pages without failing; the bundle then carries the generated landing only.

#### Scenario: Backfill without the authored landing
- **WHEN** `opm-v4.4.5` has neither `docs-kit.cue` nor `docs/catalogs/opm/`, and `main` has both
- **THEN** `4.4.5.0` holds the member pages extracted from the tag's `opm/` module and a generated landing, and the job succeeds

### Requirement: The workflow has a revision mode
`publish.yml` SHALL accept `mode: revision` with the inputs `tag` (the release's git tag) and `fix` (a 40-hex commit on `main`). In that mode it SHALL check out `main` with full history, run `opm-docs revise --project <project> --tag <tag> --fix <fix>`, then `push`, sign the pushed digest with cosign keyless and run `promote`, under the same `refs/heads/main` guard, permissions and concurrency group as `release`. Source: DESIGN decision 6.

#### Scenario: A docs fix reaches a release
- **WHEN** catalog_opm dispatches `mode: revision`, `tag: opm-v4.4.5`, `fix: <sha of a comment-only fix on main>` and 4.4.5.0 is the newest revision
- **THEN** `4.4.5.1` is published and signed, and `4.4.5` points at it

#### Scenario: A code fix is refused
- **WHEN** the fix commit changes a CUE value
- **THEN** the job fails at `revise`, naming the file, and nothing is pushed

### Requirement: Build and publish run in separate jobs
`publish.yml` SHALL build (`check`, `build` or `revise`) in a job that declares `contents: read` and `packages: read` and no `id-token`, and SHALL push, sign and promote in a second job that needs the first, checks out nothing of the caller and takes the bundle tree from a workflow artifact. Its inputs, outputs and the caller's required permissions SHALL be unchanged except the additive `setup-go` input, and the concurrency groups SHALL be declared at the workflow level, covering both jobs of a run, so two revisions of one release never build at once.

#### Scenario: Two revisions dispatched together
- **WHEN** two `revision` runs of `opm-v4.5.1` are dispatched a second apart
- **THEN** the second run's `build` job starts only after the first run has finished, and the two get different revision numbers

#### Scenario: Repository code never meets the signing token
- **WHEN** a cli release publish runs its dump command
- **THEN** that command runs in the build job, which cannot request an OIDC token, and the signing job runs no step from the cli tree

### Requirement: The publish job signs only the bundle the run asked for
Before `push`, the publish job SHALL refuse the downloaded tree unless its `manifest.json` names `inputs.project` as `project` and `github.repository` as `source.repo`, and either `inputs.tag` as `source.ref` (`release`, `revision`) or `edge` as `version` (`edge`). The artifact SHALL be named `docs-bundle-<project>-<run id>-<run attempt>`, so a re-run never takes an earlier attempt's tree.

#### Scenario: A build job writes another project's bundle
- **WHEN** the build job of a `core` edge run leaves a tree whose manifest names project `cli`
- **THEN** the publish job fails before `push`, naming `cli` and `core`, and nothing is pushed or signed

### Requirement: The workflow installs Go on request
`publish.yml` SHALL accept a boolean input `setup-go` (default `false`); when true, the build job SHALL install Go with a SHA-pinned `actions/setup-go` from `src/go.mod` in `release` mode and from the checkout of `main`'s `go.mod` otherwise (`revision` included), before building, with its cache off.

#### Scenario: cli check
- **WHEN** the cli calls `mode: check` with `setup-go: true`
- **THEN** Go from the cli's `go.mod` is on `PATH` when `opm-docs check` runs the dump command
