## Introduction

This document shows how to setup Recaller for Zsh.

# Set HISTFMT
```sh
export HISTTIMEFORMAT="%d/%m/%y %T "
```

You need to install fzf for Zsh auto-completion to work properly.

## Install `fzf`

### On Mac
```sh
brew install fzf
```

### On Linux

| Package Manager | Linux Distribution      | Command                            |
| --------------- | ----------------------- | ---------------------------------- |
| APK             | Alpine Linux            | `sudo apk add fzf`                 |
| APT             | Debian 9+/Ubuntu 19.10+ | `sudo apt install fzf`             |
| Conda           |                         | `conda install -c conda-forge fzf` |
| DNF             | Fedora                  | `sudo dnf install fzf`             |
| Nix             | NixOS, etc.             | `nix-env -iA nixpkgs.fzf`          |
| Pacman          | Arch Linux              | `sudo pacman -S fzf`               |
| pkg             | FreeBSD                 | `pkg install fzf`                  |
| pkgin           | NetBSD                  | `pkgin install fzf`                |
| pkg_add         | OpenBSD                 | `pkg_add fzf`                      |
| Portage         | Gentoo                  | `emerge --ask app-shells/fzf`      |
| Spack           |                         | `spack install fzf`                |
| XBPS            | Void Linux              | `sudo xbps-install -S fzf`         |
| Zypper          | openSUSE                | `sudo zypper install fzf`          |


## Setup Keyboard shortcut (Ctrl + h) for Run

```sh
# Launch recaller UI with Ctrl+h
recall-run-widget() {
    recaller run
    zle reset-prompt
}
zle -N recall-run-widget  # Register the widget

# Bind Alt+h to the widget
bindkey '^h' recall-run-widget
```

## Setup Keyboard shortcut (Ctrl + Option + s) for Suggest

```sh
# Recall suggestions widget
function recaller-select-suggestion-widget() {
  # Get the current command-line prefix.
  local prefix="$LBUFFER"
  # Get suggestions from your command.
  local suggestions suggestion
  suggestions=$(recaller history --match "$prefix")

  # If no suggestions, exit.
  if [[ -z "$suggestions" ]]; then
    return 0
  fi

  # Use fzf to let the user select one of the suggestions.
  suggestion=$(echo "$suggestions" | fzf --height 40% --reverse --prompt="Recalled History ⚡> ")

  # If a suggestion was chosen, update the command line.
  if [[ -n "$suggestion" ]]; then
    LBUFFER="$suggestion"
    CURSOR=${#LBUFFER}
  fi

  # Refresh the prompt.
  zle reset-prompt
}

# Register the widget with zle.
zle -N recaller-select-suggestion-widget

# Bind the widget to a key sequence (example: Ctrl+Option+s).
# Depending on your terminal, Ctrl+Option+s might send a unique sequence.
# The below binding is one possibility; use `cat -v` to determine your terminal's sequence.
bindkey '^[^S' recaller-select-suggestion-widget
```
