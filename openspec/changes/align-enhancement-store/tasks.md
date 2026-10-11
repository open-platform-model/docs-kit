## 1. Rules and policy

- [ ] 1.1 `openspec/config.yaml` and `AGENTS.md`: `references:`, the `Delivers` rule, the comment rule
- [ ] 1.2 `Link` keeps only "enhancement NNNN"; tests move with it; goldens regenerated
- [ ] 1.3 `task check` green, then commit `chore(render): link only enhancement references`

## 2. The rewrite

- [ ] 2.1 Run `enh cite-rewrite --write` and commit its result
- [ ] 2.2 Clear the worklist by hand: Go comments and strings, contracts, live specs, archived text
- [ ] 2.3 `enh cite-lint` and `enh cite-rewrite --dry-run` report nothing; `task check` green, then commit `docs: cite enhancements by id and reason`
