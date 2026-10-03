# Tasks: add-enhancements-bundle

Gate: `generalize-build-assembly` and `pull-docs-placement` are merged on docs-kit `main`. One PR, titled `feat: bundle the enhancements section`.

## 1. Spike: the enhancements repository through the transforms and the lint

- [x] 1.1 A throwaway program (not committed) applying design.md D3's transforms to the enhancements repository's `main` and running `opm-docs lint --bundle` over the result with a generated manifest; also run the enhancements repository's `scripts/check-links.sh` and compare the two link verdicts. Verify: the violation list (rule, count, files) and the link disagreements are written into design.md under "Spike findings".
- [x] 1.2 Decide from the findings, in design.md: which violations the enhancements sibling must fix in its sources, and whether any transform belongs in D3 (only a mechanical, meaning-preserving one). Verify: design.md D3 and "Spike findings" agree.
- [x] 1.3 `task check` green, then commit `docs(openspec): record the enhancements lint spike`.

## 2. The section placement

- [x] 2.1 `schema/config.cue`, `schema/manifest.cue`: `section` (D1), optional `version` for a section. `internal/build`: edge only, `--release` and `revise` refused; bundle-mode lint with an empty segment. Verify: tests for a section build, a refused release, a link to a missing section page.
- [x] 2.2 `task check` green, then commit `feat(build): add the section placement`.

## 3. The enhancements source

- [x] 3.1 `schema/config.cue`: `#Enhancements` (D2). `internal/extract/enhancements` and its renderer: reading, transforms (D3), data model (D4), pages (D2). Verify: a fixture repository under `internal/extract/enhancements/testdata/` (a live and an archived entry, INDEX, GRAPH, each link kind, an untagged fence, a comment, a dangling link, a shortcode); golden pages and `enhancements.golden.json`.
- [x] 3.2 `internal/dialect`: the `/enhancements/graph/` form and the `link-enhancements-graph` conformance fixture (D6). Verify: `go test ./internal/dialect/...`.
- [x] 3.3 `task check` green, then commit `feat(extract): add the enhancements source`.

## 4. Pull of sections

- [x] 4.1 `schema/pull.cue` `sections`, `schema/lock.cue` roots, `internal/config`, `internal/pull` (D5), the sweep keeps section projects. Verify: in-process registry tests for a section pull, no edge (fails), placement mismatch, `--local enhancements@edge=...`, a project in two roles (exit 1).
- [x] 4.2 `task check` green, then commit `feat(pull): pull the enhancements section`.

## 5. Contracts and archive

- [x] 5.1 `docs/contracts.md`: C21 (new) and C3, C7, C8, C11 (D1 to D6); `README.md`. Verify: schema text equals `schema/*.cue`.
- [x] 5.2 `openspec archive add-enhancements-bundle --yes`. Verify: `task openspec:check` green.
- [x] 5.3 `task check` green, then commit `docs(pull): document the enhancements section`.
