## 1. Rules and policy

- [x] 1.1 `openspec/config.yaml` and `AGENTS.md`: `references:`, the `Delivers` rule, the comment rule, the archive guidance
- [x] 1.2 `Link` keeps only "enhancement NNNN"; tests move with it; goldens regenerated; the fixture catalog digest moves with the two enforcement rows
- [x] 1.3 `task check` green except the known failures (`TestRunSite`, `openspec:check` under the local 1.14.1), then commit `chore(render): link only enhancement references in the link citation policy`

## 2. The rewrite

- [x] 2.1 Run `enh cite-rewrite --write` (with `testdata/` excluded) and commit its result
- [x] 2.2 Clear the worklist by hand: Go comments and strings, contracts, live specs, archived text
- [x] 2.3 `enh cite-lint` and `enh cite-rewrite --dry-run` report nothing (with the old-form tests and `testdata/` excluded); then commit `docs: cite enhancements by id and reason`
