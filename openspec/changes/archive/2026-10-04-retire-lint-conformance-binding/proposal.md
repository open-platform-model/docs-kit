## Why

opmodel.dev#47 (`retire-git-pipeline`, docs/orchestration.md step 7) deletes the site's shell lint, `site/scripts/lint-sources.sh`, its byte-identical contract copy and its fixture copy `site/tests/lint/`. Every page the site serves is then linted by `opm-docs lint` on pull, so the Go lint in `internal/dialect` is the only implementation of the page dialect. The rules that bound two linters to one fixture set (the `dialect-lint` requirement "The conformance fixture set binds both linters until phase 3", C11's "Agreement with the site's shell lint until phase 3", the conformance README's re-sync rule) now bind nothing and tell opmodel.dev to copy fixtures into a directory that no longer exists. This is step 8 of the orchestration plan, gated on G3.5.

## What Changes

- **`dialect-lint` spec**: dialect 1 is defined by the rule list in `docs/contracts.md` C11, not by a site script; the shell lint appears only as the rule set's origin. The binding requirement is removed. A new requirement keeps the conformance fixture set as ordinary tests of the Go lint, failing `task test` on any disagreement. Two requirements that tell opmodel.dev to copy fixtures or mirror the shell lint lose that clause.
- **`docs/contracts.md` C11**: the opening paragraph no longer says opmodel.dev keeps a shell lint at the same set; "Agreement with the site's shell lint until phase 3" becomes "Conformance fixtures", owned by docs-kit alone.
- **`internal/dialect`**: the conformance README and the package and test comments describe the fixtures as tests of this package, with the shell lint as their origin.
- **`docs/orchestration.md`**: step 8 and G3.5 are done, and phase 3 is complete.

SemVer class: none. Documentation, specs and comments only: no command, flag, message, exit code or lint rule changes, so the commits are `docs` and cut no docs-kit release.

Consumers: opmodel.dev has nothing left to do; after #47 it holds no lint and no fixtures to keep in step. No other repository reads C11's agreement clause.

Scope: two sections, then the archive step.

## Capabilities

### Modified Capabilities

- `dialect-lint`: dialect 1 is defined by C11; the conformance set tests the Go lint only.

## Impact

- Docs: `docs/contracts.md` (C11), `docs/orchestration.md`, `internal/dialect/testdata/conformance/README.md`.
- Code: comments in `internal/dialect/dialect.go` and `internal/dialect/dialect_test.go`; no behaviour.
- Risk: none at runtime. Merging before opmodel.dev#47 would leave the site's shell lint without a binding to the Go lint, so this change waits for that merge (G3.5).
