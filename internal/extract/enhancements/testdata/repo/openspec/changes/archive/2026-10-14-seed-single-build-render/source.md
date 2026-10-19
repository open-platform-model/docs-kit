# Source note: seed-single-build-render (example shortcut, see below)

**Example shortcut, not allowed by the design.** In the real migration a capability is seeded whole, once: `task seed:new CAP` refuses a capability that already exists in `specs/` (T2.3 §3), and a seed is ADDED only (V33). This seed holds 2 of the 22 requirements of the capability, only the ones that enhancement 0026 modifies, because a full seed of `single-build-render` is out of scope for the worked example. It stands for a full seed. The capability's `## Purpose` is the real one. Capability: `single-build-render`. Written 2026-10-13 (invented date).

## Sources

Read at `library` commit `a87fdda` (HEAD of 2026-10-09), file `openspec/specs/single-build-render/spec.md`.

| Requirement here | Source requirement | Lines |
| --- | --- | --- |
| The render dependency list is derived by promotion | "The render module's dependency list is derived by promotion", first paragraph only. The replacement-handling paragraphs are left out as mechanism. | 22-24 |
| The single-provider guard runs inside the build | "The single-provider guard runs inside the build". The parity-test scenario is left out as test structure. | 189-219 |

## Not in this seed, because of the shortcut

"Version skew is detected from the two committed resolutions" (lines 96 to 118). Enhancement 0026 makes the platform's version-skew comparison meaningless but does not say what happens to the requirement. A full seed would have put the requirement in `specs/`, where the collision review could show it. See the walkthrough, finding F9 (an example artifact).

sampled: 2026-10-14 b7d20c1
