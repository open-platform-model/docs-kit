---
title: "CLI Reference"
description: "Every opm command and flag, generated from the CLI's cobra commands."
weight: 2
---

OPM CLI manages module lifecycle and configuration for the Open Platform Model.

Exit codes:

```text
0  success
1  general error, usage errors included (unknown command or flag,
   wrong number of arguments)
2  validation error or refusal
3  connectivity error (cluster or registry unreachable)
4  permission denied
5  not found
```

The help of a command states where it differs, for example `opm instance status`, which exits 2 when a resource is not ready.

Each page in this section covers one top-level command and every command under it: its usage, description, flags and examples, generated from the CLI's cobra commands. `opm <command> --help` prints the same facts for the CLI you have installed.

```text
opm [command]
```

## Global flags

Every command takes these flags.

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--config` |  | string |  | Path to config file (env: `OPM_CONFIG`). |
| `--registry` |  | string |  | CUE registry URL (env: `OPM_REGISTRY`). |
| `--timestamps` |  | bool | `true` | Show timestamps in log output. |
| `--verbose` | `-v` | bool |  | Enable verbose output. |
