# Proposal

## Why

The comment rule of the workspace names an enhancement in a comment as `(enhancement 0013)`, beside a reason in words. On a public reference page a reader cannot resolve that parenthesis, and the words only add noise. Under the `strip` citation policy, the default of every extractor source, docs-kit already removes the references a reader cannot resolve. A parenthesis that holds only an enhancement reference is one of them. Now `strip` removes it. The owner asked for this as a separate change after the comment rule landed.

## What Changes

- `Strip.Clean` removes a parenthesis whose whole content is `enhancement NNNN`, or a list `enhancements NNNN and NNNN`, with the one space before it. The sentence stays whole: `rule (enhancement 0013).` becomes `rule.`.
- A parenthesis with other words (`(see enhancement 0013 for the design)`), a code span, a spec block comment and a Markdown link destination stay as written.
- The `cue-catalog` extractor calls the package function `CleanParagraphs` today; it now calls `Strip.CleanParagraphs`, so it gets the rule.
- The `link` policy does not change.
- `docs/contracts.md` "Citation policy" states the rule. The frozen library snapshot golden page loses one `(enhancement 0015)`.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `enhancements-bundle`: the `strip` policy no longer leaves a lone `(enhancement NNNN)` as written.

## Impact

- Affects `internal/doctext/policy.go`, `internal/extract/cuecatalog`, `docs/contracts.md` C6 text, one golden page.
- Contract touched: the rendered text of the citation policy (C6). Consumers that build pages under `strip`: catalog_opm (cue-catalog), core and library (cue-definitions, go-api), cli (cobra). opm-operator uses `link` and sees no change. Each consumer picks the change up with its next docs-kit pin; its rendered pages then lose the lone parenthesis. No consumer has to edit a file.
- Release class: `feat(doctext)`, a minor bump of `opm-docs` on the 0.x line. The change is not breaking: it removes text that no reader could use.

## Enhancement

None. No line of an enhancement `tasks.md` is delivered here; the owner decided this as a follow-up of the comment rule.
