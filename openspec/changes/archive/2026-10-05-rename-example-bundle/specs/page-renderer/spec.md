## REMOVED Requirements

### Requirement: Completable pages take an authored page first
**Reason**: its scenario names the controller by its old product name; OpenSpec cannot retitle a scenario in place.
**Migration**: replaced, with the same behaviour, by "An authored page completes a generated page", whose scenario is "The controller's resource page".

## ADDED Requirements

### Requirement: An authored page completes a generated page
When a renderer marks a page completable and a `markdown` source of the same bundle supplies a page at the same path, the bundle's page SHALL be the authored front matter and body, one blank line, then the generated body without its own front matter, recorded as `generated: false` with the authored file as `source`. An authored body holding the generated body's first heading SHALL fail the build with exit 2 naming the file. Without an authored page, the generated page SHALL stand alone with generated front matter.

#### Scenario: The controller's resource page
- **WHEN** the `crd` source renders `reference/controller-resources.md` and a `markdown` source over `docs/site` supplies `reference/controller-resources.md`
- **THEN** the page holds the authored front matter and intro followed by the generated resource entries, and `manifest.json` records it with `source` `docs/site/reference/controller-resources.md`
