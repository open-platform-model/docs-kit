# Design

## Context

See proposal.md. The citation policy lives in `internal/doctext/` (`doctext.go` strips old forms; `policy.go` implements `strip` and `link`). Five extractors and two renderers read the policy. `docs/contracts.md` C6, C8, C18 and "Doc-comment rules" describe it.

## Goals / Non-Goals

**Goals:**
- No decision number in any file the cite lint reads, except in the exact paths listed in proposal.md that exercise the old forms.
- `link` links "enhancement NNNN" only.

**Non-Goals:**
- Changing what `strip` removes. Old tags still carry the old forms, so every old form stays recognised.
- Rewriting the old-form inputs of the strip fixtures (crd, go-api, cuedefs, cue-catalog, the library snapshot). They hold old-form text on purpose, so the strip tests prove that old forms are removed. A separate sentence in three of them names an enhancement, for the link tests.

## Decisions

**Link keeps one branch.** `Policy.Clean` under `link` drops the `reDecision` branch. A decision citation outside a code span is then removed by `Clean`, the same as under `strip`. Alternative: leave the branch and link to `/enhancements/NNNN/decisions/`. Rejected: new-layout entries have no `decisions` page, so the link breaks the site build.

**The rewrite tool runs once, with the fixture folders excluded.** The tool is an aid, not an authority: it rewrites policy descriptions into nonsense ("the Markdown link `enhancement 0015 (...)`"), so the hand pass re-reads every site it touched. The exclude lists exact folders and files, not every `testdata/`, so a new fixture with a decision number fails the lint.

**Test files that test the old forms stay.** `internal/doctext/doctext_test.go` and `policy_test.go` keep old-form strings, so the lint skips them by path. Other tests that exercise `link` use the new form in their own strings.

## Risks / Trade-offs

- A reader of a page built under `strip` now sees "enhancement NNNN" where the comment had one; a decision citation never appeared there. This is the design of the store, not new here.
- `link` output for a decision citation changes from a link to nothing. Only `opm-operator` uses `link`, and its change drops the old forms from its sources.

## Durable decisions

- The `link` policy links "enhancement NNNN" only: lands in `docs/contracts.md` C6 and the `crd-extractor` and `opm-docs-cli` specs.
