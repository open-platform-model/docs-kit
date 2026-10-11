## ADDED Requirements

### Requirement: A provider may state a compatibility window

A provider module MAY state a floor and a ceiling on its registration, the releases of its catalog whose output its operator accepts. Each SHALL default to the registration's version, so a provider that states none behaves as before. The catalog, the version and the provider set SHALL stay derived and verified.

#### Scenario: No window stated

- **WHEN** a provider states no floor or ceiling
- **THEN** both equal the registration's version

#### Scenario: A window stated

- **WHEN** a provider states floor 1.4.0 and ceiling 1.8.0 around version 1.6.0
- **THEN** the effective window is 1.4.0 to 1.8.0

### Requirement: The window reaches the cluster with defaults filled

The rendered registration SHALL carry the window with its defaults filled, so the claim is readable without the provider module in hand.

#### Scenario: A defaulted window is visible

- **WHEN** a provider states no window and its registration is applied
- **THEN** the applied registration shows floor and ceiling equal to its version

### Requirement: A registration alone admits its provider catalog

An accepted registration SHALL admit its provider catalog. No platform admission entry for the catalog's path SHALL be required.

#### Scenario: No entry is needed

- **WHEN** a registration is accepted and the platform has no entry for its catalog
- **THEN** the catalog is admitted

### Requirement: A platform entry replaces the provider's window whole

When an admission entry exists for a provider catalog's path, its floor and ceiling SHALL replace the registration's window whole. When none exists, the registration's window SHALL bind, and removing the entry SHALL make the author's window bind again.

#### Scenario: An entry replaces the window

- **WHEN** the platform has an entry with ceiling 1.9.0 and the author's ceiling is 1.8.0
- **THEN** the effective ceiling is 1.9.0

#### Scenario: Removing the entry restores the window

- **WHEN** that entry is deleted
- **THEN** the effective ceiling is 1.8.0 again

### Requirement: The registration reports its effective window and source

Acceptance SHALL write the effective window and its source, platform or provider, to the registration's status.

#### Scenario: The source is reported

- **WHEN** a platform entry sets the window
- **THEN** the status reports that window with source platform

### Requirement: A platform range wider than the window warns

A platform range wider than the author's window SHALL be accepted with a warning condition on the registration that names both bounds.

#### Scenario: A wider range warns

- **WHEN** the platform ceiling is 1.9.0 and the author's is 1.8.0
- **THEN** the registration is accepted and carries a warning naming 1.9.0 and 1.8.0

### Requirement: A platform range excluding the version is refused

Acceptance SHALL refuse a registration whose version lies outside the platform's range for its catalog, because the default pick would be inadmissible and nobody chose another.

#### Scenario: The version is excluded

- **WHEN** the platform range is 1.7.0 to 1.9.0 and the registration's version is 1.6.0
- **THEN** the registration is refused, naming the range and the version

### Requirement: A contract has one provider per resolution

Acceptance SHALL refuse a claim that provides a contract which another active claim or enabled entry serving the same major of the declaring catalog already provides, naming the holder and the contract. Providers serving different majors of the declaring catalog SHALL NOT count against each other. A claim's own catalog SHALL NOT hold a contract against that claim.

#### Scenario: Two providers of one major are refused

- **WHEN** a claim provides a contract another active claim serving the same major already provides
- **THEN** it is refused, naming the contract and the holding claim

#### Scenario: Providers of different majors are both accepted

- **WHEN** one claim serves the older major of the declaring catalog and another serves the newer, for the same contract
- **THEN** both are accepted

#### Scenario: An active claim is not refused against its own catalog

- **WHEN** an active claim is judged again against a platform that now carries its own catalog
- **THEN** it is not refused for providing its own contracts

## MODIFIED Requirements

### Requirement: A build-incompatible provider is refused at acceptance

Acceptance SHALL compare the catalog's committed requirements with the platform's floors on each shared platform path, taking the floor of the admitted entry that shares the provider's major, and SHALL refuse a claim that requires a newer version within that major. It SHALL refuse a different major only when no admitted entry shares the provider's major. The refusal SHALL say the comparison is conservative and that lowering the requirement is the author's fix.

#### Scenario: A provider requiring a newer build is refused

- **WHEN** the catalog requires a shared path at a higher version than the floor of the admitted entry of its major
- **THEN** the claim is refused, naming the path, both versions and the one-line fix

#### Scenario: A provider on a different major is refused

- **WHEN** the catalog requires a shared path in a major no admitted entry shares
- **THEN** the claim is refused without comparing versions, naming every admitted major for that path

#### Scenario: A provider requiring an older build is accepted by this check

- **WHEN** every shared requirement is at or below the floor of the admitted entry of its major
- **THEN** this check does not refuse the claim

#### Scenario: A provider is compared against the entry of its own major

- **WHEN** the platform admits two majors of a catalog and the provider requires a version of the older one
- **THEN** the comparison uses the older major's floor, never the newer's

## REMOVED Requirements

### Requirement: A contract has one provider

**Reason**: The count is taken per resolution. Two majors of one catalog are no longer two providers of one contract across the whole platform.

**Migration**: Read "A contract has one provider per resolution".
