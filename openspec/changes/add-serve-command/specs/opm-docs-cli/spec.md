## MODIFIED Requirements

### Requirement: The command set
`opm-docs` SHALL provide the commands `build`, `lint`, `check`, `push`, `promote`, `pull`, `revise`, `serve` and `version`, with the flags `docs/contracts.md` lists under "Commands". It SHALL exit 0 on success, 1 on a usage error (unknown command or flag, missing argument, unreadable or invalid config) and 2 on an execution error (lint violations, a refused build, a registry or signature failure). Every error message SHALL name what failed and the fix.

#### Scenario: Version
- **WHEN** `opm-docs version` runs
- **THEN** it prints `opm-docs <version>` and exits 0

#### Scenario: Unknown flag
- **WHEN** `opm-docs build --nope` runs
- **THEN** it exits 1 naming `--nope`

## ADDED Requirements

### Requirement: serve previews a repository's bundles
`opm-docs serve` SHALL build the selected projects as edge bundles (a dirty work tree allowed) into a temporary directory it removes on exit, each with `build`'s own extractors, repository commands and bundle-mode dialect lint, so a page `build` refuses is never served, and SHALL serve them: by default on its embedded minimal Hugo site with the host's `hugo`, listening on `127.0.0.1` only (refusing, with exit 1, a missing `hugo` or one older than 0.146.0, naming the version found), mounting each tab bundle at `<root>edge/`, each docs bundle at `/docs/` and any other placement at its root, and starting a rebuild of a bundle within two seconds of a change to a file under its sources; with `--site <dir>`, by running `task bundles:pull` and then `task serve` in that directory with `OPM_BUNDLES_LOCAL` naming each built tree (a tab or section as `<project>@edge`, a docs bundle as `<project>@<--version>`), without watching. A build failure while watching SHALL be printed and SHALL keep the last good bundle served.

#### Scenario: Preview an authored page
- **WHEN** an author in the cli repository runs `opm-docs serve` and edits `docs/site/start/install.md`
- **THEN** the page at `/docs/start/install/` on the printed local URL shows the edit after the rebuild

#### Scenario: A failed rebuild keeps the last good page
- **WHEN** the author saves a page that breaks the page dialect while `opm-docs serve` runs
- **THEN** the violations print as `build` prints them, and the page keeps showing the last build that passed

#### Scenario: No hugo
- **WHEN** `opm-docs serve` runs without `hugo` on `PATH`
- **THEN** it exits 1 naming `hugo` and the version it needs

#### Scenario: Preview in the real site
- **WHEN** `opm-docs serve --site ../opmodel.dev --version v1.0` runs in the cli repository
- **THEN** it runs the site's `task bundles:pull` and `task serve` with `OPM_BUNDLES_LOCAL="cli@v1.0=<built tree>"`

#### Scenario: Docs bundle without a site version
- **WHEN** `opm-docs serve --site ../opmodel.dev` runs for a docs bundle without `--version`
- **THEN** it exits 1 naming `--version`
