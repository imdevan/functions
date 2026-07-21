# Goal
 
`functions` a function manager for your favorite shell

recreate /home/devy/dotfiles/functions/new_function.sh:3

as a go cli

# Context

alias already exsists
see ../aliases for current

# Commands

fm
fm add
fm edit
fm delete
fm search
fm run script_name search for script name, then execute it
fm config
fm config init

# Architecture


## config

Default config location ~/.config/functions/config.toml already implemented.

New config opions: index_folders: a list of glob patterns for the user to defined directories in addition 
  to the default fm location which the user would like to index. e.g. ~/.bookmarks, or ~/dotfiles which 
  would would allow for fm in those folders to be found when the user uses the `alias search` command


## storage

Default fm storage location ~/.functions/ (or appropriate shell). The place where aliases are added via `functions add` 

### .functions folder layout

```bash
.functions
├── global
│   └── global_name.zsh
├── lib
│   └── lib_name.sh
└── scripts
    ├── aliases.zsh
    └── script_name.sh

```
# Open Questions


## Types of functions

each function is generated based on an embeded tmpl file 

all files should be protected form loading multiple times via 

```bash
if [[ -n "${_FLOAD_LIB_SCRIPT_NAME:-}" ]]; then
  return
fi
typeset -g _FLOAD_LIB_SCRIPT_NAME=1
```
middle word matches function type 

### lib
functions used in other functions
- should be imported into those functions via 
`source "$FUNCTIONS_ROOT/lib/script_name.zsh"`
- not loaded into global shell 

### global
Load into global config on shell load

### scripts
Only needed occasionally, okay with calling file first

could be run by ~/.functions/scripts/script_name.sh ...
functions run script_name

functions edit script_name to_function ? 


## scripts/aliases

scripts are asynchronous functions that are not loaded at shell load time. 

however .functions/scripts/aliases is which allows for scripts to be quickly triggered from the global context without having to source them on shell load. 





