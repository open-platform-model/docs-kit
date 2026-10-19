## Before / After

**The three modules from the problem statement.** The platform admits `catalogs/opm@v4` with floor 4.0.0 and no ceiling.

| Module pins `opm` at | Before, platform at 4.3.0 | After |
| --- | --- | --- |
| 4.1.0 | renders at 4.3.0, silently | renders at 4.1.0 |
| 4.3.0 | renders at 4.3.0 | renders at 4.3.0 |
| 4.6.0 | refused, unknown key | renders at 4.6.0 |
| 3.9.0 | refused, different major | refused, `catalogs/opm@v3` not admitted |

Platform team later sets `floor: 4.2.0`. Before: a bump re-rendered all three. After: the 4.1.0 module is refused by name until its owner bumps; the other two do not move.

**The provider.** k8up module 2.1.0 pins `catalogs/k8up@v1` at 1.6.0 and declares floor 1.4.0, ceiling 1.8.0. No admission entry for the path.

| Consumer pins `k8up` at | Before | After |
| --- | --- | --- |
| nothing | 1.6.0, registration's version | 1.6.0, same |
| 1.7.0 | platform's version wins, pin inert | 1.7.0, inside the author's window |
| 1.9.0 | platform's version wins, pin inert | refused, above the installed provider's ceiling |

Platform team adds an admission entry for the path with ceiling 1.9.0. The 1.9.0 consumer renders, the registration's status reads `source: platform`, and a warning condition records that the platform exceeded the author's claim. Delete the entry and the author's window binds again.

