## Purpose

Decide which version of each catalog a render holds, and refuse a render that the platform does not admit. The module's own committed pin picks the version; the platform bounds it. A refusal names the module, the path, the pin and the bound.

## ADDED Requirements

### Requirement: A module's catalog pin is the version held

On every catalog path a module imports, a render SHALL hold the version the module's committed dependency list names, provided the platform admits it. Two modules on one platform pinning different admitted releases of one catalog SHALL each render at their own pin, and a platform change SHALL NOT move either.

#### Scenario: A module renders at its own pin

- **WHEN** the platform admits a catalog major from 4.0.0 with no ceiling, and a module pins 4.1.0
- **THEN** the render holds 4.1.0

#### Scenario: Two modules keep their own pins

- **WHEN** one module pins 4.1.0 and another pins 4.6.0 of the same admitted catalog
- **THEN** each render holds its own pin, and neither moves when the platform changes

### Requirement: A render records the catalog versions it held

Every render SHALL record the version it held on each catalog path. The record SHALL be part of the render's identity, and a platform operator SHALL be able to read it on the rendered instance.

#### Scenario: A changed pin changes the identity

- **WHEN** a module is re-rendered after its pin on a catalog moves
- **THEN** the recorded versions differ and the render's identity differs

#### Scenario: The operator reads the record

- **WHEN** an operator inspects a rendered instance
- **THEN** it shows each catalog path with the version the render held

### Requirement: An unadmitted catalog is refused

A render SHALL refuse a module that imports a catalog path with no enabled admission entry, naming the module and the path.

#### Scenario: No entry for the path

- **WHEN** a module imports a catalog major the platform has no entry for
- **THEN** the render is refused as not admitted, naming the module and the path

#### Scenario: A disabled entry admits nothing

- **WHEN** the platform keeps an entry for the path with admission disabled
- **THEN** the render is refused as not admitted

### Requirement: A pin outside the platform's range is refused

A render SHALL refuse a pin below its entry's floor or above its ceiling, naming the module, the path, the pin, the bound and whose bound it is. The render SHALL NOT promote the pin to a version inside the range.

#### Scenario: A pin below the floor

- **WHEN** the floor is 4.2.0 and a module pins 4.1.0
- **THEN** the render is refused, naming the module, the path, 4.1.0 and the floor

#### Scenario: A pin above the ceiling

- **WHEN** the ceiling is 4.6.0 and a module pins 4.7.0
- **THEN** the render is refused, naming the ceiling as the platform's bound

### Requirement: An open ceiling admits later releases

An entry with no ceiling SHALL admit every release of its path's major at or above the floor.

#### Scenario: A later release is admitted

- **WHEN** the floor is 4.0.0, there is no ceiling, and a module pins 4.9.0
- **THEN** the render holds 4.9.0

### Requirement: The resolved platform is generated, never authored

For each render the kernel SHALL generate the resolved platform that holds the build's catalogs. A render SHALL take an authored platform and a module, and SHALL NOT take a resolved platform as input. The resolved platform SHALL keep the shape of today's render-time registry, which core renames from `#Platform` to `#ResolvedPlatform`, and `#Platform` SHALL name the authored platform.

#### Scenario: The kernel generates it

- **WHEN** a render is given a module and an authored platform
- **THEN** the build holds a resolved platform the kernel generated from the module's pins, the registrations and the platform

#### Scenario: A resolved platform is not an input

- **WHEN** a caller offers a resolved platform in place of an authored one
- **THEN** the render is refused

### Requirement: The resolved platform holds only the build's catalogs

The resolved platform SHALL carry exactly one entry per catalog the build holds, at the version it holds. An admitted catalog that no module imports and no registration supplies SHALL be absent.

#### Scenario: An admitted but unimported catalog is absent

- **WHEN** the platform admits a catalog that the module does not import and no registration supplies
- **THEN** the resolved platform has no entry for it

### Requirement: Identical resolutions share one generated platform

Generation SHALL be keyed by the resolved catalog set, the authored platform's generation and the accepted registrations, so that renders with the same resolution share one generated platform.

#### Scenario: Same pins share

- **WHEN** two modules resolve to the same catalog versions under the same platform and registrations
- **THEN** they use one generated platform

#### Scenario: Different pins do not

- **WHEN** a third module pins another admitted release
- **THEN** it gets its own generated platform

### Requirement: A render without a platform holds its pins

A render given no platform SHALL hold every catalog it pins at the pinned version. It SHALL carry no registrations, SHALL meet none of the admission refusals, and SHALL NOT use an authored or synthesized platform.

#### Scenario: A platform-less render holds its pins

- **WHEN** a module is rendered with no platform
- **THEN** it holds each catalog at its pin and no admission refusal applies

#### Scenario: A platform-less render has no providers

- **WHEN** such a module demands a provider-fulfilled contract
- **THEN** the demand is unresolved, because no registration is present

### Requirement: A provider catalog follows the same pin rule

A consumer module importing a provider catalog's path SHALL render at its own pin under the same admission rules as a static catalog, checked against the provider catalog's effective window. A pin outside that window SHALL be refused, naming the module, the path, the pin and the bound.

#### Scenario: A pin inside the window is held

- **WHEN** a provider's window is 1.4.0 to 1.8.0 and a consumer pins 1.7.0
- **THEN** the render holds 1.7.0

#### Scenario: A pin above the window is refused

- **WHEN** the same consumer pins 1.9.0 and no platform entry widens the window
- **THEN** the render is refused, naming the provider's ceiling

### Requirement: A provider catalog defaults to the registration's version

When no consumer module pins a provider catalog's path, the render SHALL hold the version of the accepted registration.

#### Scenario: No consumer pin

- **WHEN** a module needs the provider's transformer and pins no version of the provider catalog
- **THEN** the render holds the registration's version

### Requirement: A provider's requirements are checked against pins

For every catalog the build holds, a provider catalog's committed requirement on each shared platform path SHALL be at most the version the build holds there, within the same major, and a different major SHALL be refused. The refusal SHALL name the provider catalog, the consumer module, the path and both versions. Acceptance of the provider SHALL NOT exempt a render.

#### Scenario: A requirement above the pin is refused

- **WHEN** a provider catalog requires 4.5.0 of a shared path and the consumer's pin is 4.3.0
- **THEN** the render is refused, naming the provider catalog, the module, the path and both versions

#### Scenario: Acceptance does not exempt a render

- **WHEN** a provider was accepted at registration
- **THEN** a render whose pins do not hold the versions it requires is still refused

### Requirement: A resolution holds one major per catalog

Each resolved platform SHALL hold at most one major of any catalog. A render of a module pinned to one major SHALL hold that major and no other major of the same catalog.

#### Scenario: A module holds only its major

- **WHEN** the platform admits two majors of a catalog and a module pins the older one
- **THEN** the resolved platform holds the older major only

### Requirement: A provider joins only resolutions of its major

A resolution SHALL include a registered provider only when the provider was built against the major of its declaring catalog that the resolution holds. A provider built only against another major SHALL be absent from that resolution.

#### Scenario: A provider on another major is absent

- **WHEN** a provider was built against the older major and a module's resolution holds the newer major
- **THEN** the resolution does not include that provider

### Requirement: A module importing two majors is refused

A render SHALL refuse a module whose own dependencies import two majors of one catalog, naming the module, the catalog path and both majors. The refusal SHALL hold with a platform and without one.

#### Scenario: Refused with a platform

- **WHEN** a module imports two majors of one catalog and the platform admits both
- **THEN** the render is refused, naming the module, the path and both majors

#### Scenario: Refused without a platform

- **WHEN** the same module is rendered with no platform
- **THEN** the render is refused in the same way
