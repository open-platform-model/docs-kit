## Purpose

How docs bundle builds are tagged and ordered: immutable full tags, moving release, minor, major and edge tags, and docs revision numbers.

## ADDED Requirements

### Requirement: A release build gets one immutable full tag
A release or revision build SHALL be tagged `<version>.<revision>` (for example `4.4.5.0`, `4.4.5.1`, `1.0.0-beta.2.0`), and that tag SHALL never be moved. `opm-docs push` SHALL refuse, exiting 2, when the full tag already names a different digest, and SHALL succeed without writing when it names the same digest. Source: DESIGN decisions 6 and 7.

#### Scenario: Re-running a release publish is a no-op
- **WHEN** `4.4.5.0` already names digest D and a re-run pushes a bundle whose digest is D
- **THEN** `push` exits 0, prints D and the tag, and changes nothing in the registry

#### Scenario: A different build for a published version is refused
- **WHEN** `4.4.5.0` names digest D and a push of version 4.4.5 revision 0 produces digest E
- **THEN** `push` exits 2 with a message naming `4.4.5.0`, D, and that a documentation fix is a docs revision

### Requirement: Builds are ordered by version precedence, then revision
Release builds SHALL be ordered by SemVer 2.0.0 precedence of their `version` annotation, then numerically by their `revision` annotation. Edge builds SHALL take no part in this order. The identity of a build SHALL be read from its manifest annotations, never parsed out of a tag name.

#### Scenario: A revision sorts between patches
- **WHEN** builds 4.4.5 rev 0, 4.4.5 rev 1 and 4.4.6 rev 0 exist
- **THEN** the order is 4.4.5.0, 4.4.5.1, 4.4.6.0

#### Scenario: A prerelease of the next minor is newer
- **WHEN** builds 4.4.6 rev 0 and 4.5.0-rc.1 rev 0 exist
- **THEN** 4.5.0-rc.1.0 is the newer build

### Requirement: Moving tags follow the newest signed build of their line
`opm-docs promote --digest D` SHALL first verify D's signature, and SHALL then point each moving tag of D's line at D only when D is the newest build of that line: the release tag `<version>` (builds of that version), the minor tag `<MAJOR>.<MINOR>` and the major tag `<MAJOR>` (builds of that minor or major, prereleases included), or `edge` for an edge build. It SHALL never point a moving tag at any digest other than D.

#### Scenario: A revision of an older patch moves only its release tag
- **WHEN** 4.4.6.0 exists and `promote` runs for a new build 4.4.5.2
- **THEN** `4.4.5` points at 4.4.5.2, and `4.4` and `4` still point at 4.4.6.0

#### Scenario: A new patch moves three tags
- **WHEN** `promote` runs for a new build 4.4.6.0 and no newer 4.x build exists
- **THEN** `4.4.6`, `4.4` and `4` point at 4.4.6.0

#### Scenario: An unsigned digest is not promoted
- **WHEN** `promote` runs for a digest with no valid signature
- **THEN** it exits 2 naming the digest and moves no tag

### Requirement: Revision numbers count up from zero
A release's first build SHALL be revision 0. A docs revision SHALL take one more than the highest revision published for that version, and SHALL be refused when revision 0 of that version does not exist.

#### Scenario: Second docs fix of a release
- **WHEN** 4.4.5.0 and 4.4.5.1 exist and a docs revision of `opm-v4.4.5` is built
- **THEN** the build is revision 2 and is pushed as `4.4.5.2`

### Requirement: Edge builds have no full tag
An edge build SHALL carry version `edge` and revision 0, SHALL be pushed by digest without a full tag, and SHALL be reachable only through the `edge` tag. Source: DESIGN decision 5.

#### Scenario: Edge push writes no tag before promote
- **WHEN** `push` uploads an edge build
- **THEN** the registry holds the manifest by digest and no tag names it until `promote` moves `edge`
