# Zsh Setup Guide

This document shows how to setup Recaller for Zsh shell.

## ✅ History Configuration (Optional)

**Good news**: Zsh works with Recaller out of the box! Zsh automatically includes timestamps in `~/.zsh_history` when using the extended history format.

**Recommended settings** for optimal experience:

Add the following to your `~/.zshrc`:

```zsh
# Enable extended history format (recommended for Recaller)
setopt EXTENDED_HISTORY

# Optional: Increase history size for better search results
export HISTSIZE=10000
export SAVEHIST=20000

# Write history immediately after each command (recommended)
setopt INC_APPEND_HISTORY
```

### Apply the Configuration

After adding the above to your `~/.zshrc`:

```zsh
# Reload your shell configuration
source ~/.zshrc

# Verify extended history is enabled (should show timestamps)
head ~/.zsh_history
```

## Verification

After configuration, your `~/.zsh_history` should contain entries like:

```
: 1692284400:0;ls -la
: 1692284405:0;git status
: 1692284410:0;recaller
```

This format includes timestamps (the number after the first colon) that Recaller uses for intelligent ranking.

## Setup Keyboard Shortcut (Ctrl + h)

Add this to your `~/.zshrc`:

```zsh
# Launch recaller with Ctrl+h
recall-run-widget() {
    recaller
    zle reset-prompt
}
zle -N recall-run-widget
bindkey '^h' recall-run-widget
```

## Advanced: Inline Suggestions (Optional)

For advanced users who want inline history suggestions, install `fzf` and add this widget:

### Install fzf

```zsh
# macOS
brew install fzf

# Ubuntu/Debian
sudo apt install fzf

# Arch Linux  
sudo pacman -S fzf
```

### Setup Suggestion Widget

Add to your `~/.zshrc`:

```zsh
# Inline history suggestions with Ctrl+Alt+s
recaller-suggest-widget() {
  local prefix="$LBUFFER"
  local suggestions suggestion
  suggestions=$(recaller history --match "$prefix")
  
  if [[ -z "$suggestions" ]]; then
    return 0
  fi
  
  suggestion=$(echo "$suggestions" | fzf --height 40% --reverse --prompt="History > ")
  
  if [[ -n "$suggestion" ]]; then
    LBUFFER="$suggestion"
    CURSOR=${#LBUFFER}
  fi
  
  zle reset-prompt
}

zle -N recaller-suggest-widget
bindkey '^[^S' recaller-suggest-widget
```

## Troubleshooting

### Problem: Recaller shows no commands
**Solution**: Ensure you have run some commands in zsh first to populate `~/.zsh_history`.

### Problem: Commands appear but without proper ranking
**Solution**: Enable `EXTENDED_HISTORY` in your `~/.zshrc` for timestamp support.

### Problem: Only recent commands appear
**Solution**: Increase `HISTSIZE` and `SAVEHIST` values in your configuration.

### Problem: fzf widget not working
**Solution**: Install fzf using your package manager and restart your shell.

## For New Zsh Users

If you're new to zsh or have a minimal setup:

```zsh
# Essential zsh configuration for Recaller
echo 'setopt EXTENDED_HISTORY' >> ~/.zshrc
echo 'setopt INC_APPEND_HISTORY' >> ~/.zshrc
echo 'export HISTSIZE=10000' >> ~/.zshrc
echo 'export SAVEHIST=20000' >> ~/.zshrc

# Reload configuration
source ~/.zshrc
```

Now Recaller should work optimally with your zsh history!