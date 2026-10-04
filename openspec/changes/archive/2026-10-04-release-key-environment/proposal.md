## Why

`RELEASE_APP_PRIVATE_KEY` is an org secret with no environment, so a workflow run on any branch of this repository can read it on a push event, and with it the release App can create `v*` tags here. docs-kit's `v*` tags are the trust root opmodel.dev uses for signed docs bundles (`docs/contracts.md` C5, C9), so the key needs a tighter guard than any other secret here. The owner decided (release-cascade security pass, 2026-10-04) to move the key, unrotated, into a main-only `release` environment in every repository that reads it; the environment already exists here with a `main` branch policy. Three smaller gaps found by the same audit close in this change: `release.yml` grants write permissions at workflow level, the goreleaser job restores a Go build cache another run may have written, and no file names code owners for CI and release config.

## What Changes

- **`release.yml`**: the `release-please` job, the only reader of the key, runs in `environment: release` and declares `permissions: {}` (it acts only through the App token). The workflow-level grant becomes `permissions: {}`; the `goreleaser` job keeps its own `contents: write`. `actions/setup-go` in the `goreleaser` job sets `cache: false`.
- **`.github/CODEOWNERS`**: the owners review `.github/`, `Taskfile.yml`, `release-please-config.json` and `.release-please-manifest.json`.
- **`.github/dependabot.yml`**: weekly GitHub Actions updates, `ci`-prefixed.
- `ci.yml` already declares `contents: read`; `publish.yml` is reusable and takes its grants from the caller's job (C5), so neither changes.

SemVer class: none. Workflow and repository config only, committed as `ci`; no command, contract or release asset changes.

Consumers: none. `publish.yml`, the workflow interface other repositories call, does not change.

Scope: one section, then the archive step.

## Capabilities

### Modified Capabilities

- `tool-release`: the release workflow reads the release key only in the main-only `release` environment, grants each job only what it uses, and builds release binaries without an Actions cache.

## Impact

- Files: `.github/workflows/release.yml`, `.github/CODEOWNERS`, `.github/dependabot.yml`.
- Risk: until the owner stores the key in the `release` environment, the job reads the org secret as before, so the change is safe to merge first. A wrong `permissions: {}` on `release-please` would show on the next push to `main` as a failed release-please step and touch nothing else.
