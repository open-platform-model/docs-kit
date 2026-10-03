---
title: "demo completion"
description: "Generate the autocompletion script for the specified shell."
type: reference
---

Every command on this page also takes the [global flags](/docs/reference/cli/#global-flags).

## demo completion

Generate the autocompletion script for the specified shell.

```text
demo completion [command]
```

Generate the autocompletion script for demo for the specified shell. See each sub-command's help for details on how to use the generated script.

**Subcommands**

| Command | Summary |
| --- | --- |
| [demo completion bash](/docs/reference/cli/demo-completion/#demo-completion-bash) | Generate the autocompletion script for bash. |
| [demo completion fish](/docs/reference/cli/demo-completion/#demo-completion-fish) | Generate the autocompletion script for fish. |
| [demo completion powershell](/docs/reference/cli/demo-completion/#demo-completion-powershell) | Generate the autocompletion script for powershell. |
| [demo completion zsh](/docs/reference/cli/demo-completion/#demo-completion-zsh) | Generate the autocompletion script for zsh. |

## demo completion bash

Generate the autocompletion script for bash.

```text
demo completion bash
```

Generate the autocompletion script for the bash shell.

This script depends on the `bash-completion` package. If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

```text
source <(demo completion bash)
```

To load completions for every new session, execute once:

\#\#\#\# Linux:

```sh
demo completion bash > /etc/bash_completion.d/demo
```

\#\#\#\# macOS:

```sh
demo completion bash > $(brew --prefix)/etc/bash_completion.d/demo
```

You will need to start a new shell for this setup to take effect.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--no-descriptions` |  | bool |  | disable completion descriptions. |

## demo completion fish

Generate the autocompletion script for fish.

```text
demo completion fish [flags]
```

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

```sh
demo completion fish | source
```

To load completions for every new session, execute once:

```sh
demo completion fish > ~/.config/fish/completions/demo.fish
```

You will need to start a new shell for this setup to take effect.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--no-descriptions` |  | bool |  | disable completion descriptions. |

## demo completion powershell

Generate the autocompletion script for powershell.

```text
demo completion powershell [flags]
```

Generate the autocompletion script for powershell.

To load completions in your current shell session:

```sh
demo completion powershell | Out-String | Invoke-Expression
```

To load completions for every new session, add the output of the above command to your powershell profile.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--no-descriptions` |  | bool |  | disable completion descriptions. |

## demo completion zsh

Generate the autocompletion script for zsh.

```text
demo completion zsh [flags]
```

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need to enable it. You can execute the following once:

```text
echo "autoload -U compinit; compinit" >> ~/.zshrc
```

To load completions in your current shell session:

```text
source <(demo completion zsh)
```

To load completions for every new session, execute once:

\#\#\#\# Linux:

```sh
demo completion zsh > "${fpath[1]}/_demo"
```

\#\#\#\# macOS:

```sh
demo completion zsh > $(brew --prefix)/share/zsh/site-functions/_demo
```

You will need to start a new shell for this setup to take effect.

**Flags**

| Flag | Shorthand | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `--no-descriptions` |  | bool |  | disable completion descriptions. |
