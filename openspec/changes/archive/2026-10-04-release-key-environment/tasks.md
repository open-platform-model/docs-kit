# Tasks: release-key-environment

One PR. Every commit is `ci`: no docs-kit release.

## 1. Guard the release key and the release job

- [x] 1.1 `.github/workflows/release.yml`: workflow-level `permissions: {}`; `release-please` declares `environment: release` and `permissions: {}`; `goreleaser` keeps `contents: write` and sets `cache: false` on `actions/setup-go`. Verify: `grep -n "RELEASE_APP_PRIVATE_KEY\|environment:\|permissions\|cache" .github/workflows/*.yml` shows the key only in the job that declares `environment: release`, and `actionlint` passes.
- [x] 1.2 `.github/CODEOWNERS` naming the owners on `/.github/`, `/Taskfile*.yml`, `/release-please-config.json` and `/.release-please-manifest.json`; `.github/dependabot.yml` for weekly `github-actions` updates with the `ci` prefix. Verify: every CODEOWNERS path exists in the tree.
- [x] 1.3 `task check` and `actionlint` green, then commit `ci(workflow): read the release key only in the main-only release environment`.

## 2. Review follow-ups (PR 50)

- [x] 2.1 The App token step requests only `permission-contents`, `permission-pull-requests` and `permission-issues` write; the `goreleaser` checkout sets `persist-credentials: false`; CODEOWNERS adds `/.goreleaser.yml` and states review is enforced only by the ruleset; dependabot ignores `open-platform-model/.github*`. The delta and main spec name the token scope and the checkout. Verify: `actionlint`, `task check`, `task openspec:check` green.

## 3. Archive

- [x] 3.1 `openspec archive release-key-environment` folds the `tool-release` delta into the main spec; `task openspec:check` green; commit `chore(openspec): archive release-key-environment`.
