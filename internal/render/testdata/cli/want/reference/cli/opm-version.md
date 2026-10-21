---
title: "opm version"
description: "Show version information."
type: reference
---

Every command on this page also takes the [global flags](/docs/reference/cli/#global-flags).

## opm version

Show version information.

```text
opm version [flags]
```

Show OPM CLI version information.

Displays:

- OPM CLI version, commit, and build date
- CUE SDK version (the one the binary is linked against)

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--output` | `-o` | string | `text` | Output format (text, json, yaml). |

**Examples**

```sh
# Print the version for a person to read
opm version

# Print it for a script
opm version -o json
```
