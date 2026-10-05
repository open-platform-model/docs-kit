## MODIFIED Requirements

### Requirement: A site version resolves from its anchor and the anchor's pins
For each site version, `pull` SHALL resolve, verify (C9, with the docs project's `repo`), unpack and lint the anchor first, requiring the resolved build to be in the anchor tag's line; SHALL then resolve each pinned project at the release tag equal to the anchor manifest's pin for it, requiring the resolved build's version to equal the pin; and SHALL resolve each `tags` project at its tag. A pinned project with no pin in the anchor's manifest, or a pin whose release tag does not exist, SHALL fail the pull with exit 2 naming the anchor's version, the project, the pinned version and the release to publish. Source: DESIGN decisions 9 and 10.

#### Scenario: Pins choose the docs
- **WHEN** `v1.0` anchors on `cli` tag `1.0`, which resolves to 1.0.0-beta.6 pinning core `2.0.0-beta.1`
- **THEN** `pull` resolves `docs/core` tag `2.0.0-beta.1` and unpacks it under `_versions/v1.0/core/`

#### Scenario: A docs revision of a pinned release
- **WHEN** core publishes `2.0.0-beta.1.1` and `2.0.0-beta.1` moves to it
- **THEN** the next pull of `v1.0` takes revision 1 of core with no config change

#### Scenario: A pinned release without a bundle
- **WHEN** the anchor pins `opm-controller` `1.0.0-beta.5` and `docs/opm-controller` has no tag `1.0.0-beta.5`
- **THEN** `pull` exits 2 naming the cli version, `opm-controller`, `1.0.0-beta.5` and the release to publish
