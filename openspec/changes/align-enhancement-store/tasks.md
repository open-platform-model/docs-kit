## 1. Rules and policy

- [x] 1.1 `openspec/config.yaml` and `AGENTS.md`: `references:`, the `Delivers` rule, the comment rule, the archive guidance
- [x] 1.2 `Link` keeps only "enhancement NNNN"; tests move with it; goldens regenerated; the fixture catalog digest moves with the two enforcement rows
- [x] 1.3 `task check` green except the known failures (`TestRunSite`, `openspec:check` under the local 1.14.1), then commit `chore(render): link only enhancement references in the link citation policy`

## 2. The rewrite

- [x] 2.1 Run `enh cite-rewrite --write` and commit its result
- [x] 2.2 Clear the worklist by hand: Go comments and strings, contracts, live specs, archived text
- [x] 2.3 `enh cite-lint` and `enh cite-rewrite --dry-run` report nothing with the excludes of proposal.md, then commit the hand pass as `docs: state the link citation policy in the contracts and live specs` and `docs(openspec): fix the rewrite of archived citation examples`

## 3. Review fixes

- [x] 3.1 Deltas for the four requirements the link policy changes (`skip_specs` removed); the strip fixtures keep one old-form citation; the exclude names exact paths; proposal and design corrected
