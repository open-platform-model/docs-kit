# Page-dialect conformance fixtures

These fixtures test `opm-docs lint` (this package) against page dialect 1
(docs/contracts.md, C11). `TestConformance` lints every case and fails unless
the lint reports exactly the lines in the case's `shell.out`, the expected
output. They bind no other linter: opmodel.dev's `site/scripts/lint-sources.sh`,
which the first cases came from, was retired with the site's git pipeline, and
the site now lints every page with `opm-docs lint`.

`shell.out` keeps the name it got when the first cases were captured from the
shell lint (below); it is the expected output of every case, whoever wrote it.

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
Its `shell.out` is written by hand.

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
is now allowed; its page is unchanged.

## Rule changes

- A dialect rule changes here only: with a new case, or a changed `shell.out`,
  that records it, in the same change as the rule.
- No other repository copies these fixtures; the site picks up a rule change
  when it moves its pinned `opm-docs` to the release that carries it.
