# Tasks: check-markup-in-every-bundle

One PR closing docs-kit#27 and docs-kit#37; its `feat` commits release `v0.6.0`.

## 1. Propose the change

- [x] 1.1 Measure what authored pages and published bundles hold (design.md, "Research & Decisions"). Verify: the counts recorded there come from `mdsafe.Check` over the six repositories' `origin/main` and from each GHCR tag's layer, fetched anonymously.
- [x] 1.2 `openspec validate check-markup-in-every-bundle --strict` passes.
- [x] 1.3 `task check` green, then commit `docs(openspec): propose check-markup-in-every-bundle`.

## 2. Two rule sets in the markup check

- [x] 2.1 `internal/mdsafe`: `Authored` allows a raw HTML node made only of comments a browser closes where goldmark does; `Generated` keeps every rule (design.md D2). Verify: `TestCheckRefuses` and `TestCheckAllows` cases per mode, including `--!>`, `<!-->`, `<!--->`, text after `-->`, an unclosed comment and a well-formed block and inline comment.
- [x] 2.2 `internal/build`: a section page is checked as `Generated` (it was `Authored`, which no longer means the full check). Verify: the existing section tests pass unchanged.
- [x] 2.3 `internal/mdsafe` tests: the tokenizer-based detector replaces `reActive`; `TestProbesWithHugo` fails on a probe Hugo cannot build unless `hugoBuildFails` names it; every probe runs under both modes; comment probes added; `TestProbesWithSite` renders the probes through Hextra and the site's `_markup` hooks when `OPM_DOCS_SITE_DIR` is set (design.md D4). Verify: `OPM_DOCS_REQUIRE_HUGO=1 go test -run 'TestProbes' ./internal/mdsafe/` with Hugo 0.167.0, and with `OPM_DOCS_SITE_DIR` pointing at an opmodel.dev `site/` checkout.
- [x] 2.4 `task check` green, then commit `feat(lint): let authored pages keep HTML comments a browser closes where goldmark does`.

## 3. Check every page of every bundle

- [x] 3.1 `internal/build`: `Lint` runs the check on every listed page of every bundle, `Generated` for a generated page or a section page, `Authored` otherwise (design.md D1). Verify: `internal/build` tests: a tab bundle with raw HTML in a generated page fails `Lint`; a docs bundle whose authored page carries a writer's note passes; one whose authored page carries `<b>` fails.
- [x] 3.2 `internal/mdtext`: `Text` escapes `{` and `}` (design.md D3). Verify: `mdtext` tests; a `render` test feeds a cue-catalog note `## Install {.hx:fixed}` and the page passes `mdsafe.Check`; `crd` and `cobra` tests show a kind or command name with a brace is refused by the extractor; a `cue-definitions` doc comment with a heading attribute block fails `build` naming the page and line.
- [x] 3.3 `task check` green, then commit `feat(lint): check the markup of every bundle page`.

## 4. Contracts and published bundles

- [ ] 4.1 `docs/contracts.md`: C11 (bundle mode), C21 (the check and its two rule sets), "Doc-comment rules" (Escaping, the guarantee), C17 (a heading attribute block fails the build), "Commands" (`build`, `lint`, `pull`); `README.md` and `AGENTS.md` where they describe the check.
- [ ] 4.2 Verify the published bundles: `task build`, then `bin/opm-docs pull` anonymously with a config naming catalog-opm (from 4.5, edge), and core, opm-operator and cli at every published tag (edge and the backfill releases). Verify: the pull exits 0 and the lock lists every bundle.
- [ ] 4.3 `task check` green, then commit `docs: state the markup check as the guarantee for every bundle`.
