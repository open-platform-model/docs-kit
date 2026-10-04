# Tasks: release-key-environment

One PR. Every commit is `ci`: no docs-kit release.

## 1. Guard the release key and the release job

- [x] 1.1 `.github/workflows/release.yml`: workflow-level `permissions: {}`; `release-please` declares `environment: release` and `permissions: {}`; `goreleaser` keeps `contents: write` and sets `cache: false` on `actions/setup-go`. Verify: `grep -n "RELEASE_APP_PRIVATE_KEY\|environment:\|permissions\|cache" .github/workflows/*.yml` shows the key only in the job that declares `environment: release`, and `actionlint` passes.
- [x] 1.2 `.github/CODEOWNERS` naming the owners on `/.github/`, `/Taskfile*.yml`, `/release-please-config.json` and `/.release-please-manifest.json`; `.github/dependabot.yml` for weekly `github-actions` updates with the `ci` prefix. Verify: every CODEOWNERS path exists in the tree.
- [x] 1.3 `task check` and `actionlint` green, then commit `ci(workflow): read the release key only in the main-only release environment`.

## 2. Archive

- [x] 2.1 `openspec archive release-key-environment` folds the `tool-release` delta into the main spec; `task openspec:check` green; commit `chore(openspec): archive release-key-environment`.
