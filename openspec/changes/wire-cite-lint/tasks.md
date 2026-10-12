## 1. Wire the check

- [ ] 1.1 Add `task cite-lint` (tool at the pinned commit, the excludes of design.md) and run it from `task check`
- [ ] 1.2 Add `.github/workflows/cite-lint.yml`: a plain call of the reusable workflow, pinned to a full commit
- [ ] 1.3 List the task and the CI check in `AGENTS.md`
- [ ] 1.4 `task cite-lint` reports 0 findings; `task check` passes except the known failures; actionlint is clean
