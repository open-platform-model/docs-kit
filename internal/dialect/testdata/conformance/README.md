# Page-dialect conformance fixtures

These fixtures bind two linters to one dialect until the site's shell lint is
retired: `opm-docs lint` (this package) and opmodel.dev's
`site/scripts/lint-sources.sh`. Both must report, for every case, exactly the
lines in its `shell.out` (docs/contracts.md, C11).

## Source

Copied from opmodel.dev `site/tests/lint/` at commit
`3a36678cfcf5242a82db120b7a74d59ee52f6783`: every case directory, with its
`expect` and `setup.sh`. `site/tests/dialect/` is not part of the set; it
tests the site's build checks, not the lint.

`shell.out` was captured once by running that commit's `lint-sources.sh` over
each case's `<repo>/docs/site` roots (in the site runner's repository order,
after `setup.sh`), with the case directory stripped from each path and the
summary line removed. `expect` is the site runner's own prefix form and is
kept as copied.

`link-catalogs/` is docs-kit's: the `/catalogs/` link forms of dialect 1.
Its `shell.out` is written by hand; the shell lint adopts it when the site
re-syncs.

`link-slashless-minor/`, `link-slashless-edge/` and `link-slashless-root/` are
docs-kit's too: a `/catalogs/` link without its trailing slash. A minor or
`edge` second segment reports the segment message, a bare root the
trailing-slash message. Their `shell.out` was captured by running opmodel.dev's
`lint-sources.sh` at commit `b6a306492d129557504708d67ada7333af1f7aab` over each
case's `core/docs/site`, as above.

`link-enhancements-graph/` is docs-kit's: the `/enhancements/graph/` link form
(change `add-enhancements-bundle`), with an optional fragment; its `shell.out`
is written by hand. The same change dropped the `/enhancements/graph/` line from
the copied `link-enhancements/` case's `shell.out` and `expect`, since the form
is now allowed; its page is unchanged. opmodel.dev takes both, and teaches
`lint-sources.sh` the form, in the PR that bumps its pinned `opm-docs` to the
release that carries them.

## Re-sync rule

- docs-kit is the source of every new fixture: a rule change lands here
  first, with its fixture and its `shell.out`, in a docs-kit release.
- opmodel.dev copies the changed fixtures into `site/tests/lint/` and updates
  `lint-sources.sh` in the same PR that bumps its pinned `opm-docs` to that
  release.
- Neither side changes a rule without the other's fixture.
