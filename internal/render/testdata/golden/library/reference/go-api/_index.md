---
title: "Go API"
description: "Every exported package of the OPM library, from its doc comments."
---

Every package below belongs to the Go module `github.com/open-platform-model/library`.

## Packages

- [opm/catalog](/docs/reference/go-api/catalog/): Package catalog defines the Catalog type, mirroring the #Catalog definition in the OPM core schema: the contracts a catalog defines (#resources, #traits, #blueprints) beside the transformers that implement them (#transformers).
- [opm/errors](/docs/reference/go-api/errors/): Package errors provides the structured verdict rows and error types of OPM.
- [opm/helper](/docs/reference/go-api/helper/): Package helper is the opt-in convenience boundary of the OPM library.
- [opm/helper/objectset](/docs/reference/go-api/helper-objectset/): Package objectset finds rendered objects that share one Kubernetes apply identity, so a runtime can refuse the render instead of letting the last write silently overwrite the first.
- [opm/helper/platformmodule](/docs/reference/go-api/helper-platformmodule/): Package platformmodule generates a platform CUE module from catalog coordinates.
- [opm/kernel](/docs/reference/go-api/kernel/): Package kernel exposes the OPM runtime as a single struct, Kernel.
- [opm/module](/docs/reference/go-api/module/): Package module defines the Module type, mirroring the #Module definition in the OPM core schema.
- [opm/platform](/docs/reference/go-api/platform/): Package platform defines the Platform and PlatformMetadata types, mirroring the #Platform definition of the OPM core schema.
- [opm/schema](/docs/reference/go-api/schema/): Package schema is the kernel's single source of truth for OPM schema-side knowledge: CUE paths, metadata types, and the OCI-backed schema loader.
