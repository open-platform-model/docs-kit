---
title: "Single build render"
description: "The render path as one CUE build per render: the kernel stages the instance and platform source trees into a generated render module, evaluates it once in a context that dies with the render, and reads matching verdicts and rendered output off the built value."
type: explanation
---

## Purpose
The render path as one CUE build per render: the kernel stages the instance and platform source trees into a generated render module, evaluates it once in a context that dies with the render, and reads matching verdicts and rendered output off the built value. Covers input requirements, the promoted dependency list and its refusal invariant, version-skew detection and policy, in-build matching, and decode.

## Requirements

### Requirement: The render dependency list is derived by promotion

The kernel SHALL write the render's dependency list by promotion from committed files: the platform's tidied list adopted whole, the instance module's list added for paths only the instance carries, and the platform's entry winning every shared path. No resolver, tidy or registry lookup SHALL run at render time to compute the list.

#### Scenario: The platform wins a shared path

- **WHEN** the platform and the instance module pin different versions of the same catalog path
- **THEN** the render holds the platform's version

#### Scenario: No lookup runs at render

- **WHEN** a render builds its dependency list
- **THEN** it contacts no registry and runs no tidy

### Requirement: The single-provider guard runs inside the build

The render SHALL read the platform's provider count from the contract inventory the platform value already carries and SHALL NOT compute one of its own. A contract with provider fulfilment that two or more enabled entries provide SHALL refuse the render, naming the contract and every entry. Two majors of one catalog SHALL count as two entries.

#### Scenario: A second provider is refused in the build

- **WHEN** two enabled catalogs each supply a transformer for one contract with provider fulfilment
- **THEN** the render fails, naming the contract and both entries

#### Scenario: Two majors of one provider catalog are two providers

- **WHEN** the platform enables two majors of one catalog and a transformer of each requires one provider-fulfilled contract
- **THEN** the render fails, naming the contract and both major-qualified entries

#### Scenario: Catalog-fulfilled plurality is allowed

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the render proceeds

## Entries that changed it

| Entry | Date |
| --- | --- |
| seed | 2026-10-14 |
