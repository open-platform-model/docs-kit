# Tasks: add-version-history

Gate: none in docs-kit (starts on `main` at `v0.2.1` or later). One PR, titled `feat: write the Catalogs tab's version history`. Cross-repo steps: `docs/orchestration.md`, "Phase 1b".

## 1. The history computation

- [ ] 1.1 `internal/cuetok`: move the CUE token scanner (`cueTokens`, `equalTokens`) out of `internal/gitsrc/doconly.go`; `gitsrc` calls it. Verify: `go test ./internal/gitsrc/...` unchanged and green.
- [ ] 1.2 `schema/history.cue` (`#History`, design.md D4), embedded beside the other schemas; `schema.ValidateJSON("#History", ...)` accepts the D4 example and refuses a `segments` list of one. Verify: `go test ./schema/...`.
- [ ] 1.3 `internal/history`: `Compute(project, tool string, segs []Segment) (*History, error)` per D1 to D4, `Encode` per D4's serialization. Verify: table tests over synthetic `cuecatalog.Model`s for every row of D3's table, `firstIsFloor`, an edge-only member, a removal with `lastIn`/`page`, lineage order, the `paths` mode under different tool minors, a doc-only change (no change), and two encodes byte-identical.
- [ ] 1.4 `task check` green, then commit `feat(pull): compute version history across a tab's segments`.

## 2. Pull writes the file and the lock records it

- [ ] 2.1 `internal/pull`: after the sweep, compute and write `<out>/<project>/history.json` for each tab project with two or more catalog segments, remove it otherwise (D5); validate before writing. Verify: `pull` tests with `--local` trees for 4.5, 4.6 and edge (fixtures under `internal/pull/testdata/history/`), one segment (file removed), and `--frozen --offline` writing the same bytes as the online run.
- [ ] 2.2 `schema/lock.cue` and `internal/pull/lock.go`: the optional `history` list (D5), key order and sorting. Verify: a lock without `history` still validates; the stable-lock test covers a lock with it.
- [ ] 2.3 `task check` green, then commit `feat(pull): write each tab's history.json and record its digest`.

## 3. Contracts and docs

- [ ] 3.1 `docs/contracts.md`: C13 "Version history" (D3 table, D4 schema, example and the site's derivations); C7 gains the lock's `history` key and drops "phase 1 never writes it"; "Site decisions" points at C13. `README.md`: `pull` writes `history.json`. Verify: every link resolves; the C13 schema text equals `schema/history.cue`.
- [ ] 3.2 `task check` green, then commit `docs(pull): document the version history contract`.

## 4. Archive (rides this PR)

- [ ] 4.1 `openspec archive add-version-history --yes`. Verify: `task openspec:check` green; every durable decision landed.
- [ ] 4.2 `task check` green, then commit `chore(openspec): archive add-version-history`.
