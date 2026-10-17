# functions (fm)

A function manager for your favorite shell

## Features

- Manage shell functions (`global`, `lib`, `scripts`) and aliases seamlessly.
- Automatic double-sourcing guard generation for all created functions.
- Execute scripts on-demand with `fm run <script_name>`.
- Search functions and scripts instantly with `fm search <query>`.
- Interactive TUI browser powered by Bubble Tea.

## Commands

```bash
fm                                  # Open interactive function browser (TUI)
fm add [name] [type] [description]  # Add a new function (global, lib, scripts) or alias
fm edit [name]                      # Edit a function or script directly in your editor
fm open [name]                      # Open a function, script, or alias file in your editor
fm delete <name>                    # Delete a function or script
fm search [query]                   # Search functions and aliases by name/description
fm run <script_name> [args...]      # Search for script name and execute it with arguments
fm config                           # View or edit configuration file in your editor
fm config init                      # Generate default configuration file
```

## Architecture & Storage Layout

Default storage directory: `~/.functions`

```
.functions
├── global
│   └── <global_function>.zsh    # Loaded into global shell on startup
├── lib
│   └── <lib_function>.sh        # Library functions sourced by other functions
└── scripts
    ├── aliases.zsh              # Script aliases loaded on startup
    └── <script_function>.sh     # Executable scripts called on demand
```

## Function Types

- **`global`**: Functions loaded into global shell context on shell load.
- **`lib`**: Library functions imported by other functions via `source "$FUNCTIONS_ROOT/lib/<lib_name>.sh"`.
- **`scripts`**: Standalone executable scripts triggered via `fm run <script_name>` or automatically created script aliases in `scripts/aliases.zsh`.

## Configuration

Configuration location: `~/.config/functions/config.toml`

Run `fm config init` to create a default configuration file, or `fm config` to open it in your `$EDITOR`.
