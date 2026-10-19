## 1. Batch docs-kit-B1 (28 requirements)

Specs: `bundle-pull` 9, `enhancements-bundle` 6, `crd-extractor` 4, `cue-definitions-extractor` 3,
`opm-docs-cli` 3, `publish-workflow` 3. Gate with `N=28`.

- [ ] 1.1 Per spec: read `openspec/specs/<spec>/spec.md` on `main`, the change's `design.md`
      and the repo rules; grep the repo (not `archive/`) for each old requirement header
- [ ] 1.2 `bundle-pull`: write the delta for its 9 requirements (read "Frozen, offline and local
      pulls" and "Local and frozen pulls of docs bundles" together; keep both rules)
- [ ] 1.3 `enhancements-bundle`: write the delta for its 6 requirements
- [ ] 1.4 `crd-extractor`: write the delta for its 4 requirements
- [ ] 1.5 `cue-definitions-extractor`: write the delta for its 3 requirements
- [ ] 1.6 `opm-docs-cli`: write the delta for its 3 requirements
- [ ] 1.7 `publish-workflow`: write the delta for its 3 requirements
- [ ] 1.8 Fill the scenario ledger and clause table in `design.md` for all 28 requirements
- [ ] 1.9 Checklist item 3 per rewritten requirement: no `DIFF:` and no `LOST:` line, exit 0
- [ ] 1.10 Gate (design.md "The gate") with `N=28`: `GATE PASS`, and
      `openspec validate <spec> --type spec --strict --json` shows no WARNING for a rewritten
      requirement of the six specs
- [ ] 1.11 `task check` green, then commit `docs(openspec): split the long requirements of batch B1`

## 2. Batch docs-kit-B2 (19 requirements)

Specs: `bundle-format` 2, `cobra-extractor` 2, `dialect-lint` 2, `docs-placement` 2,
`docs-revision` 2, `go-api-extractor` 2, `page-renderer` 2, `tool-release` 2,
`cue-catalog-extractor` 1, `repository-commands` 1, `version-history` 1. Gate with `N=47`.

- [ ] 2.1 Per spec: read the spec on `main` and grep the repo (not `archive/`) for each old
      requirement header
- [ ] 2.2 `bundle-format` (2), `cobra-extractor` (2), `dialect-lint` (2): write the deltas
- [ ] 2.3 `docs-placement` (2), `docs-revision` (2), `go-api-extractor` (2): write the deltas
- [ ] 2.4 `page-renderer` (2), `tool-release` (2): write the deltas
- [ ] 2.5 `cue-catalog-extractor` (1), `repository-commands` (1), `version-history` (1): write
      the deltas
- [ ] 2.6 Fill the scenario ledger and clause table in `design.md` for all 19 requirements
- [ ] 2.7 Checklist item 3 per rewritten requirement: no `DIFF:` and no `LOST:` line, exit 0
- [ ] 2.8 Gate with `N=47`: `GATE PASS`; `openspec validate --all --strict` now prints no
      WARNING from the main specs plus the change
- [ ] 2.9 `task check` green, then commit `docs(openspec): split the long requirements of batch B2`

## 3. Put `--strict` back and run the final gate (repo extra, LAST=1)

- [ ] 3.1 `Taskfile.yml`: `openspec:check` runs `openspec validate --all --strict --no-interactive`;
      its `desc` drops "(--strict is off until the long-requirement rewrite lands)"
- [ ] 3.2 `AGENTS.md` (commands table) and `openspec/config.yaml` (Validation Gates, item 4):
      state that `task openspec:check` runs with `--strict`; remove the "off until" text
- [ ] 3.3 `grep -rn "strict is off" .` outside `openspec/changes/archive/` prints nothing
- [ ] 3.4 Final gate with `LAST=1`: `GATE PASS`, `openspec validate --specs --strict` exit 0,
      no "Archive would refuse" line, no new ERROR item
- [ ] 3.5 `task check` green (it runs `openspec:check` with `--strict`), then commit
      `ci(openspec): validate with --strict again`
- [ ] 3.6 Archive the change on this branch (`openspec-archive-change`, "Sync now"), run
      `task check`, then commit `docs(openspec): archive split-long-requirements`
