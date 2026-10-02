# bundle-format Specification

## Purpose
What a docs bundle is on the wire and on disk: the OCI artifact, its annotations, the bundle tree and `manifest.json`, deterministic packing and guarded unpacking.

## Requirements

### Requirement: A bundle is one OCI 1.1 artifact with one layer
`opm-docs push` SHALL push each bundle as an OCI 1.1 image manifest with `artifactType` `application/vnd.opmodel.docs.bundle.v1`, the empty config descriptor (`application/vnd.oci.empty.v1+json`) and exactly one layer of media type `application/vnd.opmodel.docs.bundle.layer.v1.tar+gzip`, to the repository `ghcr.io/open-platform-model/docs/<project>` unless `--registry` names another registry prefix. If the section-1 spike records outcome B, the manifest SHALL instead carry no `artifactType` and a config of media type `application/vnd.opmodel.docs.bundle.config.v1+json` holding `manifest.json`. Source: DESIGN decisions 2 and 7.

#### Scenario: Pushed manifest carries the bundle type
- **WHEN** `opm-docs push --dir out/catalog-opm` succeeds against a registry
- **THEN** the manifest at the printed digest has `artifactType` `application/vnd.opmodel.docs.bundle.v1`, an empty config and one layer of the bundle layer media type, in repository `docs/catalog-opm`

### Requirement: Manifest annotations describe the build
The pushed manifest SHALL carry the annotations `org.opencontainers.image.version` (the release version or `edge`), `org.opencontainers.image.revision` (the 40-hex source commit), `org.opencontainers.image.source` (`https://github.com/<owner>/<repo>` of the repository that built it), `org.opencontainers.image.created` (the source commit's committer time in RFC 3339 UTC), `dev.opmodel.docs.project`, `dev.opmodel.docs.revision`, `dev.opmodel.docs.dialect` and `dev.opmodel.docs.tool`, each equal to the matching `manifest.json` field.

#### Scenario: Annotations match manifest.json
- **WHEN** a release build of catalog-opm 4.4.5, revision 0, is pushed
- **THEN** its annotations read version `4.4.5`, revision `0`, project `catalog-opm`, source `https://github.com/open-platform-model/catalog_opm`, and the commit and created time of the commit tagged `opm-v4.4.5`

### Requirement: The bundle tree holds manifest.json, content and data only
A bundle tree SHALL hold exactly `manifest.json`, `content/` and `data/` at its top level. `manifest.json` SHALL validate against the embedded `#Manifest` schema (`docs.opmodel.dev/bundle/v1`) on build, push and pull. Its `pages` SHALL list every file under `content/` and nothing else, and its `data` SHALL list every file under `data/` and nothing else.

#### Scenario: An unlisted page is refused
- **WHEN** a bundle directory holds `content/traits/extra.md` that `manifest.json` does not list
- **THEN** `opm-docs push` exits 2 naming `content/traits/extra.md` as unlisted and pushes nothing

#### Scenario: An edge manifest with a revision is refused
- **WHEN** `manifest.json` has `version: "edge"` and `revision: 1`
- **THEN** schema validation fails and the command exits 2 naming the field

### Requirement: Packing is deterministic
The layer SHALL be a gzip-compressed tar with entries sorted by path, directories before their contents, file mode 0644, directory mode 0755, uid and gid 0, empty owner names, every modification time equal to `org.opencontainers.image.created`, no extended headers, and a gzip header with no name and no time. Two builds of the same source commit, config and tool version SHALL produce the same manifest digest.

#### Scenario: Rebuild gives the same digest
- **WHEN** the same commit is built and packed twice with the same `opm-docs` binary
- **THEN** both packs have the same layer digest and the same manifest digest

### Requirement: Unpacking refuses unsafe archives
Unpacking a bundle layer SHALL refuse, writing nothing, an entry with an absolute path or a `..` element, a symlink, a hard link, a device, a FIFO, a duplicate path, a top-level entry other than `manifest.json`, `content/` and `data/`, more than 10,000 entries, or more than 64 MiB of uncompressed content.

#### Scenario: A traversal entry is refused
- **WHEN** a layer contains the entry `content/../../etc/passwd`
- **THEN** unpacking fails naming the entry, and the target directory is not created
