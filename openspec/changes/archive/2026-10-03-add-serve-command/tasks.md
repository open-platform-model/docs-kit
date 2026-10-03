# Tasks: add-serve-command

Gate: `generalize-build-assembly` is merged on docs-kit `main`. One PR, titled `feat: add opm-docs serve`.

## 1. Serve on the embedded site

- [x] 1.1 `internal/serve`: the embedded skeleton site (design.md D2), the Hugo process with version check, mounts per placement, the polling watcher with last-good fallback (D3). Verify: unit tests for the mount config per placement and the watcher (fake clock, fake builder); a test that runs `hugo` when it is on `PATH`, else skips, and fetches one page over HTTP.
- [x] 1.2 `cmd/opm-docs/serve.go`: flags and exit codes (D1). Verify: `opm-docs serve --nope` exits 1; a missing `hugo` exits 1 naming it.
- [x] 1.3 `task check` green, then commit `feat(cmd): serve a repository's bundles on a local Hugo`.

## 2. The site mode

- [x] 2.1 `--site` (D4): build, then `task bundles:pull` and `task serve` in the site directory with `OPM_BUNDLES_LOCAL`; `--site-version` required for docs bundles. Verify: a test with a fake `task` on `PATH` recording its arguments and environment.
- [x] 2.2 `task check` green, then commit `feat(cmd): preview bundles in an opmodel.dev checkout`.

## 3. Docs and archive

- [x] 3.1 `docs/contracts.md` "Commands" (`serve`) and "Site decisions" (the site interface `--site` uses); `README.md` (preview). Verify: links resolve.
- [x] 3.2 `openspec archive add-serve-command --yes`. Verify: `task openspec:check` green.
- [x] 3.3 `task check` green, then commit `docs(cmd): document opm-docs serve`.
