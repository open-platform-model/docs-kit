# docs-kit repository guide

docs-kit builds each Open Platform Model repository's documentation into a versioned, signed OCI
artifact (a docs bundle) in that repository's CI, and lets the opmodel.dev site pull and assemble
those bundles. It holds one Go program, `opm-docs`, and the reusable workflow that runs it.
`DESIGN.md` is the approved design; `openspec/config.yaml` is the constitution (principles, gates,
artifact rules).

## Branch model

| Branch | What lands |
| --- | --- |
| `main` | All work, by PR. |

## Repository rules

- The Decisions table of `DESIGN.md` binds every change. Cite a decision as `DESIGN decision 9`,
  never a bare `D9`. This form is repo-local.
- `docs/contracts.md` fixes every contract another repository reads. Other repositories cite it by
  number (`docs-kit C5`); a change to it follows the constitution's Principle II.
- The bundle format, tag scheme, workflow interface, `docs-kit.cue`, pull config and lock are
  contracts other repositories read. A change to one names every consuming repository.
- Feature work ships as an OpenSpec change (`spec-driven` schema, specs included), cut into
  mergeable sections that each end green and close with their own commit.
  The OpenSpec workflow skills (`openspec-propose`, `openspec-apply-change` and the rest) are in
  `.claude/skills/`.
- Never push to a registry from a laptop. Registry writes happen in GitHub Actions, through
  `publish.yml` in a caller's CI.
- Callers pin `publish.yml` by docs-kit release tag. A moved docs-kit tag would let other code sign
  as the trusted publisher. A full bundle tag (`4.4.5.0`) is never overwritten.
- No enhancement or design reference in anything the tool renders or prints. The code that strips
  or reads the old citation forms keeps them in its tests and testdata, because a docs revision
  builds old release tags.

## Layout

```text
cmd/opm-docs/       cobra root, one file per command; flag parsing only
schema/             CUE schemas (manifest, config, pull, lock, history), embedded
cobradump/          nested Go module (own go.mod), the CLI's hook; released as its own component
internal/           one package per concern: config, bundle, build, extract/ (one per source),
                    render, dialect, mdsafe, oci, verify, publish, pull, history, revise, serve
docs/               contracts other repositories read; the cross-repo orchestration plan
.github/workflows/  the reusable publish workflow, CI, cite-lint, release
openspec/           OpenSpec workspace: config.yaml (constitution), specs/, changes/
```

## Commands

| Command | What it does |
| --- | --- |
| `task build` | Builds `bin/opm-docs` with the version stamped in |
| `task fmt`, `task fmt:check` | Formats; fails on an unformatted file |
| `task vet`, `task lint`, `task test` | Vet, golangci-lint, offline unit tests. They cover `cobradump/` too |
| `task openspec:check` | Validates every spec and active change; `--strict` is off until the long-requirement rewrite lands. `task openspec:install` installs the CLI |
| `task cite-lint` | Checks Go and CUE files against the comment rule. Run it by hand: it needs a newer Go than `task check` assumes. CI runs the same check; the excludes are in both and change together |
| `task check` | `fmt:check`, `vet`, `lint`, `openspec:check`, `test`: the gate before every commit |

CI also runs `actionlint` over the workflows, and a job runs the serve test against a pinned Hugo
(it skips locally without `hugo` on `PATH`).

## Technology standards

Kept in this file because `openspec/config.yaml` points here for them.

- `github.com/yuin/goldmark` is pinned to the version that the Hugo pinned by opmodel.dev builds
  with (`internal/mdsafe`, `docs/contracts.md` C21). Bump it with the site's Hugo, never alone.

## Style

Go style, error handling, comments and references follow `openspec/config.yaml` (Code Style): accept
interfaces, return structs; wrap errors with context; table-driven tests. Read the package and its
tests before editing. Keep `README.md`'s command list and the layout above current.

@CONTEXT.md
