## ADDED Requirements

### Requirement: The render list promotes the module's catalog pins

The kernel SHALL write the render's dependency list by promotion from committed files: on a catalog path the module's own entry, and on every other path, core included, the module's tidied closure. No resolver, tidy or registry lookup SHALL run at render time to compute the list.

#### Scenario: The module wins a catalog path

- **WHEN** the platform admits a catalog and the module pins a version inside the range
- **THEN** the render holds the module's version

#### Scenario: No lookup runs at render

- **WHEN** a render builds its dependency list
- **THEN** it contacts no registry and runs no tidy

### Requirement: The single-provider guard counts per resolution

The render SHALL read the provider count of the resolution being built and SHALL refuse a contract with provider fulfilment that two registered providers serving the same major of the declaring catalog provide, naming the contract and both entries. Providers serving different majors SHALL NOT count against each other.

#### Scenario: Two providers of one major are refused

- **WHEN** two enabled catalogs serving the same major each supply a transformer for one provider-fulfilled contract
- **THEN** the render fails, naming the contract and both entries

#### Scenario: Providers of different majors are not counted together

- **WHEN** one provider serves the older major, another serves the newer, and the resolution holds the newer
- **THEN** the render does not count the older provider and proceeds

#### Scenario: Catalog-fulfilled plurality is allowed

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the render proceeds

## REMOVED Requirements

### Requirement: The render dependency list is derived by promotion

**Reason**: The platform no longer wins a shared catalog path. The module's committed pin is the version held.

**Migration**: Read "The render list promotes the module's catalog pins".

### Requirement: The single-provider guard runs inside the build

**Reason**: The provider count is taken per resolution. Two majors of one catalog are no longer two providers of one contract across the whole platform.

**Migration**: Read "The single-provider guard counts per resolution".
