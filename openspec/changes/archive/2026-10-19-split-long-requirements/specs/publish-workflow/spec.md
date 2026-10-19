## REMOVED Requirements

### Requirement: One reusable workflow with three modes
**Reason**: Split into three requirements that each state one behavior.
**Migration**: See "publish.yml is a reusable workflow with fixed inputs and outputs", "A mode outside the three fails the workflow", "Publishing modes run only from main".

### Requirement: Publishing modes sign before tags move
**Reason**: Split into two requirements that each state one behavior.
**Migration**: See "Publishing modes sign the pushed digest before tags move", "Edge and release publishes use separate concurrency groups".

### Requirement: Build and publish run in separate jobs
**Reason**: Split into four requirements that each state one behavior.
**Migration**: See "The build job holds no signing permission", "The publish job takes the bundle from a workflow artifact", "The job split leaves the workflow interface unchanged", "Concurrency groups cover both jobs of a run".

## ADDED Requirements

### Requirement: publish.yml is a reusable workflow with fixed inputs and outputs
docs-kit SHALL provide `.github/workflows/publish.yml`, callable with `workflow_call`, taking the inputs `project` (required), `mode` (`check`, `edge` or `release`, required), `tag` and `cue-registry`, and returning the outputs `digest` and `tag`, as `docs/contracts.md` C5 fixes. It SHALL declare no secrets and use `github.token`. Source: DESIGN decisions 5 and 9.

#### Scenario: Pull-request check
- **WHEN** a caller runs the workflow with `mode: check` on a pull request
- **THEN** it runs `opm-docs check --project <project>`, pushes nothing and needs only `contents: read` and `packages: read`

### Requirement: A mode outside the three fails the workflow
A `mode` outside `check`, `edge` and `release` SHALL fail naming it. Source: DESIGN decisions 5 and 9.

#### Scenario: An unknown mode
- **WHEN** a caller runs the workflow with a `mode` outside `check`, `edge` and `release`
- **THEN** the workflow fails naming the mode

### Requirement: Publishing modes run only from main
Every mode except `check` SHALL fail before building unless `github.ref` is `refs/heads/main`. Source: DESIGN decisions 5 and 9.

#### Scenario: Publishing from a release branch is refused
- **WHEN** a caller runs `mode: edge` on `refs/heads/release/opm-v4.4`
- **THEN** the workflow fails before building, naming the ref and that bundles publish only from `main`

### Requirement: Publishing modes sign the pushed digest before tags move
In `edge` and `release` modes the workflow SHALL build, `push`, sign the pushed digest with a pinned cosign v3 keyless (`--new-bundle-format=true`, the digest, never a tag), and only then run `promote`. The caller job SHALL grant `packages: write` and `id-token: write`.

#### Scenario: Edge publish
- **WHEN** a caller runs `mode: edge` on a push to `main`
- **THEN** the registry gains a signed manifest for the pushed commit and `edge` points at it, and no other tag moved

#### Scenario: Signing fails
- **WHEN** the cosign step fails after `push`
- **THEN** the job fails, `promote` does not run, and no moving tag points at the unsigned digest

### Requirement: Edge and release publishes use separate concurrency groups
An `edge` job SHALL run in the concurrency group `docs-edge-${{ inputs.project }}` with `cancel-in-progress: true`, and a `release` job in `docs-release-${{ inputs.project }}-${{ inputs.tag }}` with `cancel-in-progress: false` (keyed on inputs, which are known when the group is evaluated), so that no edge run can cancel a pending or running release publish (GitHub keeps one pending run per group and cancels older pending ones).

#### Scenario: A burst of main pushes does not drop a release
- **WHEN** a release publish of 4.4.6 is pending and three pushes to `main` start edge publishes of the same project
- **THEN** the edge runs share their own group, and the 4.4.6 release publish runs to completion

### Requirement: The build job holds no signing permission
`publish.yml` SHALL build (`check`, `build` or `revise`) in a job that declares `contents: read` and `packages: read` and no `id-token`.

#### Scenario: Repository code never meets the signing token
- **WHEN** a cli release publish runs its dump command
- **THEN** that command runs in the build job, which cannot request an OIDC token, and the signing job runs no step from the cli tree

### Requirement: The publish job takes the bundle from a workflow artifact
`publish.yml` SHALL push, sign and promote in a second job that needs the first, checks out nothing of the caller and takes the bundle tree from a workflow artifact.

#### Scenario: The publish job checks out nothing
- **WHEN** a publish run reaches the second job
- **THEN** it checks out nothing of the caller and takes the bundle tree from a workflow artifact

### Requirement: The job split leaves the workflow interface unchanged
The inputs, outputs and the caller's required permissions of `publish.yml` SHALL be unchanged by the job split, except the additive `setup-go` input.

#### Scenario: A caller after the split
- **WHEN** a caller that used the workflow before the job split calls it again
- **THEN** its inputs, outputs and required permissions are unchanged, except the additive `setup-go` input

### Requirement: Concurrency groups cover both jobs of a run
The concurrency groups SHALL be declared at the workflow level, covering both jobs of a run, so two revisions of one release never build at once.

#### Scenario: Two revisions dispatched together
- **WHEN** two `revision` runs of `opm-v4.5.1` are dispatched a second apart
- **THEN** the second run's `build` job starts only after the first run has finished, and the two get different revision numbers
