## Why

The enhancements repository moves from its own folder layout (`NNNN/` with seven numbered documents) to OpenSpec changes: live and new entries become `openspec/changes/<id>-<slug>/`, and the ten archived entries stay at `archive/NNNN/`. The `enhancements` source reads only the old layout, so after the migration the site would show ten entries and none of the rest. The site also needs one data file that describes both layouts, the pages of the new layout, a page per capability, redirects from the old page URLs, and an `updated` date that a new entry no longer stores.

## What Changes

- **A second reader.** The `enhancements` source gains the `openspec` reader beside the legacy one. It reads `<changes>/<id>-<slug>/` and `<changes>/archive/<date>-<id>-<slug>/`, skips `seed-*`, and refuses a symlink and an id that both readers find. The legacy reader reads only `<legacy>/NNNN/`; a live legacy folder `NNNN/` at the entry root is refused with the fix named.
- **Source config.** `#Enhancements` gains `changes` (default `openspec/changes`), `specs` (default `openspec/specs`) and `legacy` (default `archive`), all relative to `dir`.
- **`data/enhancements.json` schema v2.** Every entry has `layout` (`legacy` or `openspec`). A legacy entry keeps its v1 fields, `updated` included. An openspec entry carries `breaking`, `updated`, `after`, `supersededBy` (a list), `accepted`, `progress`, `capabilities`, `documents` and `aliases`.
- **`updated` from git.** An openspec entry has no stored `updated`. The extractor reads the date of the last commit that touched the entry's folder and refuses a shallow clone, since the date would be wrong.
- **Pages.** An openspec entry publishes `_index`, `proposal`, `design`, `requirements` (the delta specs, one block per capability) and `tasks` (when the file exists). Each capability in `specs/` publishes `capabilities/<cap>.md`, and `capabilities/_index.md` lists them.
- **Redirects.** `aliases` lists, for each old page slug of a migrated entry (one that holds `evidence/migration-map.md`), the page that replaces it (`problem` and `questions` to `proposal`, `risks` and `operational` to `design`, `decisions` to `requirements`, `graduation` to `tasks`). A new entry gets none.
- **Links.** The page dialect admits `proposal`, `requirements`, `tasks` and `capabilities/<cap>/` under `/enhancements/`. The `link` citation policy links `enhancement NNNN` to `/enhancements/NNNN/`.
- **Contract.** `docs/contracts.md` C21 and the spec `enhancements-bundle` describe both readers and the v2 file.

Release class: a breaking contract change on 0.x, so a MINOR bump whose squash-merge message carries a `BREAKING CHANGE:` footer as the migration note: the schema identifier moves to `v2`, and the legacy reader refuses a live `NNNN/` folder. The v1 fields of a legacy entry do not change. Order: opmodel.dev accepts v1 and v2, then the enhancements repository moves its pinned docs-kit release, and only in the migration's last commit, after step M4 has deleted the live `NNNN/` folders.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `enhancements-bundle`: the enhancements source reads both layouts, writes data schema v2, publishes the new entry pages, capability pages and redirect data.

## Impact

- Code: `schema/config.cue`, `internal/config`, `internal/extract/enhancements`, `internal/build/enhancements.go`, `internal/gitsrc`, `internal/dialect/links.go`, `internal/doctext`, `internal/extract/crd` and `internal/render/crd.go` (the link policy), `docs/contracts.md` C6, C21 and the Doc-comment rules, `README.md`, `AGENTS.md`.
- Consumers, in this order. **opmodel.dev** first: its Enhancements adapter (`site/enhancements/_content.gotmpl`, `enh-status.html`, `gen-mounts.sh`) accepts schema v1 and v2, reads `layout`, `breaking`, `after`, `progress` and the list form of `supersededBy`, adds Hugo `aliases` from each entry's `aliases`, and renders the capability pages. **enhancements** last: it moves `.opm-docs-version` and its `docs.yml` ref together to the release carrying this change, with the migration. **opm-operator** drops `citations: "link"` once no `NNNN:Dn` citation is left (the last phase of the migration); the new policy links `enhancement NNNN` for any source that keeps `link`.
- Risk: the legacy reader no longer reads a live `NNNN/` folder, so a docs-kit built from this change refuses the pre-migration enhancements repository. The enhancements repository keeps its current pin until the migration's last commit.
