## MODIFIED Requirements

### Requirement: A bundle is one OCI 1.1 artifact with one layer
`opm-docs push` SHALL push each bundle as an OCI 1.1 image manifest with `artifactType` `application/vnd.opmodel.docs.bundle.v1`, the empty config descriptor (`application/vnd.oci.empty.v1+json`) and exactly one layer of media type `application/vnd.opmodel.docs.bundle.layer.v1.tar+gzip`, to the repository `ghcr.io/open-platform-model/docs/<project>` unless `--registry` names another registry prefix. Source: DESIGN decisions 2 and 7.

#### Scenario: Pushed manifest carries the bundle type
- **WHEN** `opm-docs push --dir out/catalog-opm` succeeds against a registry
- **THEN** the manifest at the printed digest has `artifactType` `application/vnd.opmodel.docs.bundle.v1`, an empty config and one layer of the bundle layer media type, in repository `docs/catalog-opm`

### Requirement: Manifest annotations describe the build
The pushed manifest SHALL carry the annotations `org.opencontainers.image.version` (the release version or `edge`), `org.opencontainers.image.revision` (the 40-hex source commit), `org.opencontainers.image.source` (`https://github.com/<owner>/<repo>` of the repository that built it), `org.opencontainers.image.created` (the source commit's committer time in RFC 3339 UTC), each equal to the matching `manifest.json` field.

#### Scenario: Annotations match manifest.json
- **WHEN** a release build of catalog-opm 4.4.5, revision 0, is pushed
- **THEN** its annotations read version `4.4.5`, revision `0`, project `catalog-opm`, source `https://github.com/open-platform-model/catalog_opm`, and the commit and created time of the commit tagged `opm-v4.4.5`

## ADDED Requirements

### Requirement: Outcome B changes the manifest of a bundle
The manifest is the one that `opm-docs push` pushes under "A bundle is one OCI 1.1 artifact with one layer". If the section-1 spike records outcome B, the manifest SHALL instead carry no `artifactType` and a config of media type `application/vnd.opmodel.docs.bundle.config.v1+json` holding `manifest.json`. Source: DESIGN decisions 2 and 7.

#### Scenario: Outcome B manifest
- **WHEN** `opm-docs push` pushes a bundle and the section-1 spike records outcome B
- **THEN** the manifest carries no `artifactType` and a config of media type `application/vnd.opmodel.docs.bundle.config.v1+json` holding `manifest.json`

### Requirement: The pushed manifest carries the dev.opmodel.docs annotations
The pushed manifest SHALL carry the annotations `dev.opmodel.docs.project`, `dev.opmodel.docs.revision`, `dev.opmodel.docs.dialect` and `dev.opmodel.docs.tool`, each equal to the matching `manifest.json` field.

#### Scenario: Docs annotations match manifest.json
- **WHEN** a release build of catalog-opm 4.4.5, revision 0, is pushed
- **THEN** `dev.opmodel.docs.project`, `dev.opmodel.docs.revision`, `dev.opmodel.docs.dialect` and `dev.opmodel.docs.tool` each equal the matching `manifest.json` field
