# docs-kit

Builds each Open Platform Model repository's documentation into a versioned OCI artifact, and lets the opmodel.dev site pull and assemble those artifacts.

It holds one Go program, `opm-docs`, and the shared workflow that runs it. See [DESIGN.md](DESIGN.md) for the design.

Status: phase 1 is planned in [openspec/changes/build-opm-docs-phase-1](openspec/changes/build-opm-docs-phase-1/proposal.md). Only `opm-docs version` exists today.

Licensed under the Apache License, Version 2.0; see [LICENSE](LICENSE).
