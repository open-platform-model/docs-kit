---
title: "CLI Reference"
description: "Every demo command and flag."
weight: 2
---

Demo manages widgets and the gadgets they hold.

Each page in this section covers one top-level command and every command under it: its usage, description, flags and examples, generated from the CLI's cobra commands. `demo <command> --help` prints the same facts for the CLI you have installed.

```text
demo [command]
```

## Global flags

Every command takes these flags.

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--config` |  | string | `~/.demo/config.cue` | Path to config file (env: `DEMO_CONFIG`). |
| `--verbose` | `-v` | bool |  | Enable verbose output. |
