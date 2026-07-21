# Aliases

A function manager for your favorite shell

## Features

- An alias manager that works WITH your shell
- Integrates with your favorite shell!

## Install

```bash
# With homebrew
homebrew install imdevan/aliases/aliases

# With AUR
yay -S aliases

# Manual
git clone https://github.com/imdevan/aliases.git
cd aliases
just build && sudo just install
```

See [install](https://devan.gg/aliases/install/) docs for more information.

## Creating an alias

```bash
# Create an alias that cds to current directory
~/Projects/favorite-project
aliases fp

# Add alias with name, value, description
aliases add sayhello "echo hello" "say hello description"

# Interactive list of aliases
al
```

## How `aliases` works

An **alias** in this case is an alias that is sourced into the shell on load time. 

Aliases live in `~/.aliases/aliases.zsh` by default (or `.sh` / `.fish` / `.nu` depending on shell). The location can be changed via config options. 

aliases is essentially a lightweight wrapper around that file. 

## Using different shell? 

aliases is set up for zsh first but works with any shell. Run `aliases config init` to create a custom config file.

See configuration options for more info.

## Commands

```bash
aliases                 # Without args: opens TUI list. With name: creates a cd alias for cwd
al                      # Interactive aliases list
aliases add             # Interactive form to add a new alias, or aliases add [name] [value] [description]
aliases delete          # Delete an alias
aliases edit            # Edit an alias or open aliases file in editor
aliases import          # Import aliases from a file or folder
aliases config          # View or edit configuration
aliases config init     # Generate default config file
aliases completion      # Generate shell completion scripts
```

## Configuration

aliases is designed to be highly customizable. 

```bash
aliases config # open config file location
```

Configuration file location: `~/.config/aliases/config.toml`  (`$XDG_CONFIG_HOME/aliases/config.toml`)

See [configuration](./CONFIGURATION.md) for installation options.

## Installation

See [install](./INSTALL.md) for installation options.
