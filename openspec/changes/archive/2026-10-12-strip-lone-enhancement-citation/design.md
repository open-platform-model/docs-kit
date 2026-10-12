# Design

## Context

See proposal.md. `internal/doctext/policy.go` holds `Policy.Clean`. Under `link` it links references, then runs the package function `Clean`. Under `strip` it runs `Clean` only.

## Research & Decisions

### Where the rule lives

**Context**: Several callers use the package function `Clean` directly: the `crd` renderer on a summary that holds plain references for the renderer to link, the `go-api` synopsis, and `cuedefs` comments inside spec blocks.
**Explored**: `rg "doctext\.(Clean|CleanParagraphs|Strip|Link)"`. Strip-only callers: `cuecatalog` (the `CleanParagraphs` calls). Policy-aware callers: `crd`, `goapi` markdown printer, `cuedefs` `parseDoc`, `cobra` renderer.
**Options considered**:
1. Put the rule in the package function `Clean` - one place, but the `link` path and the `crd` renderer call it on prose that must keep plain references, so it breaks `link`.
2. Put the rule in `Policy.Clean` for the non-link branch - `link` stays as it is; the strip-only caller `cuecatalog` moves to `Strip.CleanParagraphs`.
**Decision**: Option 2.
**Rationale**: It keeps the `link` output byte for byte and gives every policy-aware extractor the rule through one method.

### What counts as a lone citation

**Decision**: The parenthesis content, after optional white space, is `enhancement NNNN` or `Enhancement NNNN` (four digits), or `enhancements NNNN` followed by more ids joined by commas and `and`, the forms the link policy already knows. Anything else in the parenthesis, including a lead-in word, keeps the parenthesis.
**Rationale**: The workspace rule names an enhancement by id beside a reason in words; the bare `(enhancement NNNN)` is the one that carries nothing a reader can use. A parenthesis with words is prose.

### Boundaries

- The rule runs outside code spans, split on backticks as `link` does. A code span keeps its text.
- A parenthesis directly after `]` is a Markdown link destination and stays.
- `CleanCode` (comments inside spec blocks) is unchanged: it never removed bare references, and the rule is for prose.
- Nested case `(see (enhancement 0013))` becomes `(see)`. Not handled: the form does not occur in the rule and the result is still valid text.

## Risks / Trade-offs

- A consumer page changes text on its next pin. No consumer file needs an edit.
- The `go-api` package synopsis stays plain text under both policies; under `strip` it is cleaned with `Strip.Clean`, under `link` with the package function `Clean`, so `link` output does not change.
- `outsideCode` now finds code spans by equal-length backtick runs, so a double-backtick span that holds a backtick is not split. This also applies to `link`, and only changes output for such a span.

## Durable decisions

- The `strip` policy removes a lone `(enhancement NNNN)`: lands in `docs/contracts.md` C6 ("Citation policy") and the `enhancements-bundle` spec at archive.
