---
title: "0026: Requirements"
description: "Exactly what must hold, by capability."
type: explanation
weight: 3
---

## authored-platform

### Purpose

Define the authored platform: the data a platform team writes to say which catalog lineages a platform admits and within what range of releases. It admits and bounds. It never loads a catalog and never picks a version.

### ADDED Requirements

#### Requirement: A platform is pure data naming admitted lineages

An authored platform SHALL import nothing and carry no dependency list of its own. It SHALL name each admitted lineage by module path with its major, with an enable flag, a required floor, an optional ceiling and a prerelease opt-in that defaults to off.

##### Scenario: A platform that imports a catalog is rejected

- **WHEN** a platform value imports a catalog
- **THEN** platform validation rejects it

##### Scenario: A minimal entry is valid

- **WHEN** an entry names a path with its major and a floor only
- **THEN** the platform is valid, enabled, with no ceiling and prereleases off

#### Requirement: An entry needs a floor in its major

Platform validation SHALL reject an entry without a floor, and an entry whose floor or ceiling carries a major other than its path's.

##### Scenario: A missing floor

- **WHEN** an entry has no floor
- **THEN** platform validation rejects it

##### Scenario: A bound on another major

- **WHEN** an entry for a major 4 path has a ceiling in major 5
- **THEN** platform validation rejects it

#### Requirement: Prereleases are admitted only by opt-in

With prereleases off, a render SHALL refuse a pin that carries a prerelease suffix even when its ordering falls inside the range, and platform validation SHALL reject a bound that carries one. With it on, prerelease pins and bounds SHALL be admitted by ordering alone.

##### Scenario: A development tag inside the range is refused

- **WHEN** the floor is 4.2.0, prereleases are off, and a module pins 4.3.0-dev.2
- **THEN** the render is refused as not admitted

##### Scenario: A prerelease bound without opt-in is rejected

- **WHEN** an entry with prereleases off has a ceiling carrying a prerelease suffix
- **THEN** platform validation rejects it

##### Scenario: The opt-in admits a release candidate

- **WHEN** prereleases are on, the floor is 0.3.0, and a module pins 0.4.0-rc.1
- **THEN** the render holds 0.4.0-rc.1

#### Requirement: One platform value serves cluster and offline

For a render given a platform, one authored platform value, with the same fields, SHALL serve as the cluster's Platform resource spec and as the offline file the render starts from. A hand-written platform module SHALL NOT be an input.

##### Scenario: The same fields in both places

- **WHEN** a platform team moves a platform from an offline file to the cluster resource
- **THEN** the fields are the same, one nesting level apart

#### Requirement: Several majors of one catalog may be admitted

An authored platform SHALL be able to admit several majors of one catalog as separate admission entries. Admitting or disabling one major SHALL NOT change what a render of a module on another major holds.

##### Scenario: Disabling one major leaves the other alone

- **WHEN** a platform admits two majors and the older entry is disabled
- **THEN** a render of a module on the newer major holds the same versions as before

#### Requirement: The readiness build holds one major per catalog

The platform's own module-less build for readiness and the contract inventory SHALL hold each enabled static catalog at its floor, and SHALL NOT hold two majors of one catalog in one build. A platform admitting two majors of one catalog SHALL evaluate and report readiness, naming any slice that fails.

##### Scenario: Two majors evaluate

- **WHEN** a platform admits two majors of one catalog
- **THEN** readiness is reported per admitted major and the build does not fail to evaluate

##### Scenario: A broken slice is named

- **WHEN** one major's slice fails and another serves
- **THEN** readiness names the failing slice and does not hide it

## catalog-version-selection

### Purpose

Decide which version of each catalog a render holds, and refuse a render that the platform does not admit. The module's own committed pin picks the version; the platform bounds it. A refusal names the module, the path, the pin and the bound.

### ADDED Requirements

#### Requirement: A module's catalog pin is the version held

On every catalog path a module imports, a render SHALL hold the version the module's committed dependency list names, provided the platform admits it. Two modules on one platform pinning different admitted releases of one catalog SHALL each render at their own pin, and a platform change SHALL NOT move either.

##### Scenario: A module renders at its own pin

- **WHEN** the platform admits a catalog major from 4.0.0 with no ceiling, and a module pins 4.1.0
- **THEN** the render holds 4.1.0

##### Scenario: Two modules keep their own pins

- **WHEN** one module pins 4.1.0 and another pins 4.6.0 of the same admitted catalog
- **THEN** each render holds its own pin, and neither moves when the platform changes

#### Requirement: A render records the catalog versions it held

Every render SHALL record the version it held on each catalog path. The record SHALL be part of the render's identity, and a platform operator SHALL be able to read it on the rendered instance.

##### Scenario: A changed pin changes the identity

- **WHEN** a module is re-rendered after its pin on a catalog moves
- **THEN** the recorded versions differ and the render's identity differs

##### Scenario: The operator reads the record

- **WHEN** an operator inspects a rendered instance
- **THEN** it shows each catalog path with the version the render held

#### Requirement: An unadmitted catalog is refused

A render SHALL refuse a module that imports a catalog path with no enabled admission entry, naming the module and the path.

##### Scenario: No entry for the path

- **WHEN** a module imports a catalog major the platform has no entry for
- **THEN** the render is refused as not admitted, naming the module and the path

##### Scenario: A disabled entry admits nothing

- **WHEN** the platform keeps an entry for the path with admission disabled
- **THEN** the render is refused as not admitted

#### Requirement: A pin outside the platform's range is refused

A render SHALL refuse a pin below its entry's floor or above its ceiling, naming the module, the path, the pin, the bound and whose bound it is. The render SHALL NOT promote the pin to a version inside the range.

##### Scenario: A pin below the floor

- **WHEN** the floor is 4.2.0 and a module pins 4.1.0
- **THEN** the render is refused, naming the module, the path, 4.1.0 and the floor

##### Scenario: A pin above the ceiling

- **WHEN** the ceiling is 4.6.0 and a module pins 4.7.0
- **THEN** the render is refused, naming the ceiling as the platform's bound

#### Requirement: An open ceiling admits later releases

An entry with no ceiling SHALL admit every release of its path's major at or above the floor.

##### Scenario: A later release is admitted

- **WHEN** the floor is 4.0.0, there is no ceiling, and a module pins 4.9.0
- **THEN** the render holds 4.9.0

#### Requirement: The resolved platform is generated, never authored

For each render the kernel SHALL generate the resolved platform that holds the build's catalogs. A render SHALL take an authored platform and a module, and SHALL NOT take a resolved platform as input. The resolved platform SHALL keep the shape of today's render-time registry, which core renames from `#Platform` to `#ResolvedPlatform`, and `#Platform` SHALL name the authored platform.

##### Scenario: The kernel generates it

- **WHEN** a render is given a module and an authored platform
- **THEN** the build holds a resolved platform the kernel generated from the module's pins, the registrations and the platform

##### Scenario: A resolved platform is not an input

- **WHEN** a caller offers a resolved platform in place of an authored one
- **THEN** the render is refused

#### Requirement: The resolved platform holds only the build's catalogs

The resolved platform SHALL carry exactly one entry per catalog the build holds, at the version it holds. An admitted catalog that no module imports and no registration supplies SHALL be absent.

##### Scenario: An admitted but unimported catalog is absent

- **WHEN** the platform admits a catalog that the module does not import and no registration supplies
- **THEN** the resolved platform has no entry for it

#### Requirement: Identical resolutions share one generated platform

Generation SHALL be keyed by the resolved catalog set, the authored platform's generation and the accepted registrations, so that renders with the same resolution share one generated platform.

##### Scenario: Same pins share

- **WHEN** two modules resolve to the same catalog versions under the same platform and registrations
- **THEN** they use one generated platform

##### Scenario: Different pins do not

- **WHEN** a third module pins another admitted release
- **THEN** it gets its own generated platform

#### Requirement: A render without a platform holds its pins

A render given no platform SHALL hold every catalog it pins at the pinned version. It SHALL carry no registrations, SHALL meet none of the admission refusals, and SHALL NOT use an authored or synthesized platform.

##### Scenario: A platform-less render holds its pins

- **WHEN** a module is rendered with no platform
- **THEN** it holds each catalog at its pin and no admission refusal applies

##### Scenario: A platform-less render has no providers

- **WHEN** such a module demands a provider-fulfilled contract
- **THEN** the demand is unresolved, because no registration is present

#### Requirement: A provider catalog follows the same pin rule

A consumer module importing a provider catalog's path SHALL render at its own pin under the same admission rules as a static catalog, checked against the provider catalog's effective window. A pin outside that window SHALL be refused, naming the module, the path, the pin and the bound.

##### Scenario: A pin inside the window is held

- **WHEN** a provider's window is 1.4.0 to 1.8.0 and a consumer pins 1.7.0
- **THEN** the render holds 1.7.0

##### Scenario: A pin above the window is refused

- **WHEN** the same consumer pins 1.9.0 and no platform entry widens the window
- **THEN** the render is refused, naming the provider's ceiling

#### Requirement: A provider catalog defaults to the registration's version

When no consumer module pins a provider catalog's path, the render SHALL hold the version of the accepted registration.

##### Scenario: No consumer pin

- **WHEN** a module needs the provider's transformer and pins no version of the provider catalog
- **THEN** the render holds the registration's version

#### Requirement: A provider's requirements are checked against pins

For every catalog the build holds, a provider catalog's committed requirement on each shared platform path SHALL be at most the version the build holds there, within the same major, and a different major SHALL be refused. The refusal SHALL name the provider catalog, the consumer module, the path and both versions. Acceptance of the provider SHALL NOT exempt a render.

##### Scenario: A requirement above the pin is refused

- **WHEN** a provider catalog requires 4.5.0 of a shared path and the consumer's pin is 4.3.0
- **THEN** the render is refused, naming the provider catalog, the module, the path and both versions

##### Scenario: Acceptance does not exempt a render

- **WHEN** a provider was accepted at registration
- **THEN** a render whose pins do not hold the versions it requires is still refused

#### Requirement: A resolution holds one major per catalog

Each resolved platform SHALL hold at most one major of any catalog. A render of a module pinned to one major SHALL hold that major and no other major of the same catalog.

##### Scenario: A module holds only its major

- **WHEN** the platform admits two majors of a catalog and a module pins the older one
- **THEN** the resolved platform holds the older major only

#### Requirement: A provider joins only resolutions of its major

A resolution SHALL include a registered provider only when the provider was built against the major of its declaring catalog that the resolution holds. A provider built only against another major SHALL be absent from that resolution.

##### Scenario: A provider on another major is absent

- **WHEN** a provider was built against the older major and a module's resolution holds the newer major
- **THEN** the resolution does not include that provider

#### Requirement: A module importing two majors is refused

A render SHALL refuse a module whose own dependencies import two majors of one catalog, naming the module, the catalog path and both majors. The refusal SHALL hold with a platform and without one.

##### Scenario: Refused with a platform

- **WHEN** a module imports two majors of one catalog and the platform admits both
- **THEN** the render is refused, naming the module, the path and both majors

##### Scenario: Refused without a platform

- **WHEN** the same module is rendered with no platform
- **THEN** the render is refused in the same way

## registration-acceptance

### ADDED Requirements

#### Requirement: A provider may state a compatibility window

A provider module MAY state a floor and a ceiling on its registration, the releases of its catalog whose output its operator accepts. Each SHALL default to the registration's version, so a provider that states none behaves as before. The catalog, the version and the provider set SHALL stay derived and verified.

##### Scenario: No window stated

- **WHEN** a provider states no floor or ceiling
- **THEN** both equal the registration's version

##### Scenario: A window stated

- **WHEN** a provider states floor 1.4.0 and ceiling 1.8.0 around version 1.6.0
- **THEN** the effective window is 1.4.0 to 1.8.0

#### Requirement: The window reaches the cluster with defaults filled

The rendered registration SHALL carry the window with its defaults filled, so the claim is readable without the provider module in hand.

##### Scenario: A defaulted window is visible

- **WHEN** a provider states no window and its registration is applied
- **THEN** the applied registration shows floor and ceiling equal to its version

#### Requirement: A registration alone admits its provider catalog

An accepted registration SHALL admit its provider catalog. No platform admission entry for the catalog's path SHALL be required.

##### Scenario: No entry is needed

- **WHEN** a registration is accepted and the platform has no entry for its catalog
- **THEN** the catalog is admitted

#### Requirement: A platform entry replaces the provider's window whole

When an admission entry exists for a provider catalog's path, its floor and ceiling SHALL replace the registration's window whole. When none exists, the registration's window SHALL bind, and removing the entry SHALL make the author's window bind again.

##### Scenario: An entry replaces the window

- **WHEN** the platform has an entry with ceiling 1.9.0 and the author's ceiling is 1.8.0
- **THEN** the effective ceiling is 1.9.0

##### Scenario: Removing the entry restores the window

- **WHEN** that entry is deleted
- **THEN** the effective ceiling is 1.8.0 again

#### Requirement: The registration reports its effective window and source

Acceptance SHALL write the effective window and its source, platform or provider, to the registration's status.

##### Scenario: The source is reported

- **WHEN** a platform entry sets the window
- **THEN** the status reports that window with source platform

#### Requirement: A platform range wider than the window warns

A platform range wider than the author's window SHALL be accepted with a warning condition on the registration that names both bounds.

##### Scenario: A wider range warns

- **WHEN** the platform ceiling is 1.9.0 and the author's is 1.8.0
- **THEN** the registration is accepted and carries a warning naming 1.9.0 and 1.8.0

#### Requirement: A platform range excluding the version is refused

Acceptance SHALL refuse a registration whose version lies outside the platform's range for its catalog, because the default pick would be inadmissible and nobody chose another.

##### Scenario: The version is excluded

- **WHEN** the platform range is 1.7.0 to 1.9.0 and the registration's version is 1.6.0
- **THEN** the registration is refused, naming the range and the version

#### Requirement: A contract has one provider per resolution

Acceptance SHALL refuse a claim that provides a contract which another active claim or enabled entry serving the same major of the declaring catalog already provides, naming the holder and the contract. Providers serving different majors of the declaring catalog SHALL NOT count against each other. A claim's own catalog SHALL NOT hold a contract against that claim.

##### Scenario: Two providers of one major are refused

- **WHEN** a claim provides a contract another active claim serving the same major already provides
- **THEN** it is refused, naming the contract and the holding claim

##### Scenario: Providers of different majors are both accepted

- **WHEN** one claim serves the older major of the declaring catalog and another serves the newer, for the same contract
- **THEN** both are accepted

##### Scenario: An active claim is not refused against its own catalog

- **WHEN** an active claim is judged again against a platform that now carries its own catalog
- **THEN** it is not refused for providing its own contracts

### MODIFIED Requirements

#### Requirement: A build-incompatible provider is refused at acceptance

Acceptance SHALL compare the catalog's committed requirements with the platform's floors on each shared platform path, taking the floor of the admitted entry that shares the provider's major, and SHALL refuse a claim that requires a newer version within that major. It SHALL refuse a different major only when no admitted entry shares the provider's major. The refusal SHALL say the comparison is conservative and that lowering the requirement is the author's fix.

##### Scenario: A provider requiring a newer build is refused

- **WHEN** the catalog requires a shared path at a higher version than the floor of the admitted entry of its major
- **THEN** the claim is refused, naming the path, both versions and the one-line fix

##### Scenario: A provider on a different major is refused

- **WHEN** the catalog requires a shared path in a major no admitted entry shares
- **THEN** the claim is refused without comparing versions, naming every admitted major for that path

##### Scenario: A provider requiring an older build is accepted by this check

- **WHEN** every shared requirement is at or below the floor of the admitted entry of its major
- **THEN** this check does not refuse the claim

##### Scenario: A provider is compared against the entry of its own major

- **WHEN** the platform admits two majors of a catalog and the provider requires a version of the older one
- **THEN** the comparison uses the older major's floor, never the newer's

### REMOVED Requirements

#### Requirement: A contract has one provider

**Reason**: The count is taken per resolution. Two majors of one catalog are no longer two providers of one contract across the whole platform.

**Migration**: Read "A contract has one provider per resolution".

## single-build-render

### ADDED Requirements

#### Requirement: The render list promotes the module's catalog pins

The kernel SHALL write the render's dependency list by promotion from committed files: on a catalog path the module's own entry, and on every other path, core included, the module's tidied closure. No resolver, tidy or registry lookup SHALL run at render time to compute the list.

##### Scenario: The module wins a catalog path

- **WHEN** the platform admits a catalog and the module pins a version inside the range
- **THEN** the render holds the module's version

##### Scenario: No lookup runs at render

- **WHEN** a render builds its dependency list
- **THEN** it contacts no registry and runs no tidy

#### Requirement: The single-provider guard counts per resolution

The render SHALL read the provider count of the resolution being built and SHALL refuse a contract with provider fulfilment that two registered providers serving the same major of the declaring catalog provide, naming the contract and both entries. Providers serving different majors SHALL NOT count against each other.

##### Scenario: Two providers of one major are refused

- **WHEN** two enabled catalogs serving the same major each supply a transformer for one provider-fulfilled contract
- **THEN** the render fails, naming the contract and both entries

##### Scenario: Providers of different majors are not counted together

- **WHEN** one provider serves the older major, another serves the newer, and the resolution holds the newer
- **THEN** the render does not count the older provider and proceeds

##### Scenario: Catalog-fulfilled plurality is allowed

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the render proceeds

### REMOVED Requirements

#### Requirement: The render dependency list is derived by promotion

**Reason**: The platform no longer wins a shared catalog path. The module's committed pin is the version held.

**Migration**: Read "The render list promotes the module's catalog pins".

#### Requirement: The single-provider guard runs inside the build

**Reason**: The provider count is taken per resolution. Two majors of one catalog are no longer two providers of one contract across the whole platform.

**Migration**: Read "The single-provider guard counts per resolution".
