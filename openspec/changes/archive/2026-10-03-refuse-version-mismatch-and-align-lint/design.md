## Context

`build` derives a release bundle's version from the tag alone (`--release opm-v4.4.5` gives `4.4.5`); the `cue-catalog` extractor separately reads the catalog's `metadata.version` into `data/catalog.json`, which the landing renders. Nothing compares them. Separately, `internal/dialect/links.go` reports any `/catalogs/` link that fails the link-form regex with the trailing-slash message before looking at its segment, while opmodel.dev's `lint-sources.sh` looks at the segment first.

## Goals / Non-Goals

**Goals:** refuse a release or revision build whose tag version and catalog version differ; make both linters print the same line for every slashless catalog link the conformance set covers.

**Non-Goals:** checking the version of a markdown-only bundle (it declares none); changing bundle-mode messages; any change to the link forms that pass.

## Decisions

### D1. The version check

After the `cue-catalog` source is extracted, when the build is not edge, `build` compares the tag's version (the `--release` tag with the project's prefix removed) with the extracted `metadata.version` as strings. They MUST be equal; otherwise `build` exits 2 with:

```text
--release opm-v4.6.0 names version 4.6.0, but the catalog ./src declares metadata.version "4.5.0"; build the tag of the catalog's version, or advance the catalog's version on the release commit
```

`./src` is the source's `module` from `docs-kit.cue`. String equality, not SemVer equality: the tag version is already required to be SemVer and a catalog that declares `4.5` or `v4.5.0` for a `4.5.0` tag is itself the mismatch worth reporting. It is an execution error (exit 2), not a usage error: the invocation is well-formed; the source contradicts it. `revise` runs `build --release`, so a revision of a mislabelled release is refused the same way.

### D2. The slashless catalog link message

In docs mode, a `/catalogs/` link that fails the link form is classified as the shell lint does: strip `/catalogs/`, split on `/`; when the second segment is exactly `<int>.<int>` it reports `docs pages link catalogs through /catalogs/<first>/<major>/`; when it is exactly `edge`, `docs pages link catalogs through /catalogs/<first>/<MAJOR>/`; otherwise the trailing-slash message. So:

| Link | Message |
|---|---|
| `/catalogs/opm/4.4/traits/backup` | `catalog link "...": docs pages link catalogs through /catalogs/opm/4/` |
| `/catalogs/opm/edge` | `catalog link "...": docs pages link catalogs through /catalogs/opm/<MAJOR>/` |
| `/catalogs/opm` | `catalog link "...": write /catalogs/<name>/ or /catalogs/<name>/<MAJOR>/<path>/ with a trailing slash` |

The minor or `edge` segment is the more important fault: adding a slash would still leave a violation. Bundle mode keeps the trailing-slash message for any malformed link; the shell lint has no bundle mode to agree with.

## Research & Decisions

### Which message wins

**Context**: issue docs-kit#10 asks for one agreed message.
**Explored**: opmodel.dev `site/scripts/lint-sources.sh` lines 43 to 50.
**Options considered**:
1. Change the shell lint to the trailing-slash message - needs a site-side rule change first and hides the segment fault.
2. Match the shell lint in Go - no site change; the fixtures pin it.
**Decision**: option 2.
**Rationale**: the shell lint's message names the fault that remains after the slash is added, and the site needs only a fixture copy.

## Risks / Trade-offs

- A tag pushed by hand on a commit before the identity advance now fails the release job. That is the point; the message names the fix.

## Durable decisions

- The version check: `docs/contracts.md` "Commands" (`build` row) and the `opm-docs-cli` main spec.
- The slashless message rule: `docs/contracts.md` C11 and the `dialect-lint` main spec; the fixtures are the shared record.
