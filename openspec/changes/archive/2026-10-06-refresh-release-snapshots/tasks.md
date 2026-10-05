# Tasks: refresh-release-snapshots

One PR, shipped with `rename-example-bundle`. Every commit is `test` or `docs`: no docs-kit release.

## 1. Compare the parity tests with released bundles

- [x] 1.1 `hack/refresh-snapshots.sh` and the `snapshots:refresh` task: pull and verify the bundle, export the inputs from the release commit, refuse a docs revision whose fix touches an input, run the cli dump twice, write `want/` and `SOURCE` (design.md D2 to D4, D6). Verify: `task snapshots:refresh KIND=controller PROJECT=opm-operator VERSION=1.0.0-beta.8 CHECKOUT=<opm-operator clone>` and `task snapshots:refresh KIND=cli PROJECT=cli VERSION=1.0.0-beta.10 CHECKOUT=<cli clone>` each print `refreshed`, and a second run leaves `git status` unchanged.
- [x] 1.2 Seed `internal/render/testdata/controller` and `internal/render/testdata/cli` with those two runs; the crdref page and the cmdref pages go. Verify: `ls internal/render/testdata/{controller,cli}` shows `SOURCE`, `docs-kit.cue` and `want/`.
- [x] 1.3 `TestControllerCRDParity` and `TestCLICommandParity` read the snapshot's `docs-kit.cue` and compare with `want/` (`release_snapshot_test.go`); `compareWithRelease` applies named rewrites (the served-by one today) and fails on one that matches nothing (D5); `TestControllerCRDSection` reads the same config; `TestCLICommandParityLive`, `OPM_CONTROLLER_CHECKOUT`, the marker constants and both `TODO(cutover)` comments go. Verify: `go test ./internal/render/` passes, and editing one line of a `want/` page makes the matching test fail.
- [x] 1.4 `task check` green, then commit `test(render): compare the parity tests with released docs bundles`.

## 2. Record the release snapshots in the docs

- [x] 2.1 `docs/contracts.md` C18 and C19 parity paragraphs state what the tests compare with, the rewrite, and the refresh command, keeping the crdref and cmdref measurements as a dated record; `README.md` names the tests and the task; the `AGENTS.md` layout lists `hack/`. Verify: `git grep -n 'OPM_CONTROLLER_CHECKOUT\|OPM_CLI_CHECKOUT\|ParityLive' -- docs README.md AGENTS.md internal` prints only `docs/orchestration.md` (a dated plan).
- [x] 2.2 `task check` green, then commit `docs: record that the parity tests compare with released bundles`.
