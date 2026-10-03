## Why

A site version shows the docs of the cli release it follows and of the library, core and operator versions that release pins (DESIGN decisions 9 and 10). Today opmodel.dev works that out from git (`resolve-versions.sh`, about 900 lines: release lines, pin chains, branch heads). With phase 2 those docs arrive as bundles placed in `/docs/`, and `opm-docs pull` refuses every placement but a tab (C7: "Phase 2 adds `versions:`"). This change teaches `pull` site versions: which bundle anchors each one, how the anchor's pins choose the other bundles, where they unpack, how they are locked, and which collisions refuse a version.

Gate: `generalize-build-assembly` is merged (docs placement, `owns`, manifest `pins`). Parallel with the four extractor changes.

## What Changes

- **`bundles.cue` gains `docs` and `versions`** (C16): `docs` names every project that may be placed in `/docs/` and the one repository allowed to sign it; `versions` maps a site version (`v1.0`) to an anchor (project and tag), the projects pulled at the anchor's pins, and projects pulled by their own tag.
- **Resolution**: the anchor's tag (a minor, a major, an exact release or `edge`) resolves and verifies first; its `manifest.json` `pins` then name the exact release of each pinned project, resolved through that release's moving tag so its docs revisions follow (DESIGN decision 9). A missing pin or a pinned release without a bundle fails the pull naming what to publish.
- **Unpack layout** `<out>/_versions/<site-version>/<project>/`, replaced whole per site version, so a refused version keeps the previous one.
- **Collision and ownership checks** across one version's docs bundles: a page path in two bundles, or a page under a path another bundle `owns`, fails the version naming both projects.
- **Lock**: an optional `docs` list with role, tag, digest and the anchor's pins; `--frozen`, `--offline` and `--local <project>@<site-version>=<dir>` work for docs bundles as for tabs.
- **Contract C16** "Site versions", and C7's pull config, layout and lock updated.

Release class: MINOR. Additive: new config keys and lock key; tab pulls unchanged.

Scope: four sections, the last with the contracts and the archive.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `bundle-pull`: the pull config admits docs projects and site versions; requirements added for resolution, layout, checks, lock and local and frozen pulls.

## Impact

- Code: `schema/pull.cue`, `schema/lock.cue`, `internal/config` (`Pull.Docs`, `Pull.Versions`), `internal/pull` (versions, collisions, lock entries, `--local` parsing), `docs/contracts.md` C7 and C16.
- Consumers: **opmodel.dev** (sibling change `pull-reference-bundles`): `bundles.cue` with `docs` and `versions`, mounts `_versions/<v>/<project>/content` into that version's `/docs/`, takes "Last updated" and source links from the manifests, keeps git-sourced pages for paths no bundle owns, and switches v1.0 at the gate in `docs/orchestration.md`. **cli**: its bundle must carry `pins` (`publish-cli-bundle`). **core, library, opm-operator**: each pinned release needs a bundle (release publish or a dispatched backfill).
- Risk: a cli release whose pins name a release with no bundle makes every pull fail until that bundle is published; the site's frozen lock is the recovery path, as today.
