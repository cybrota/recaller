# Bash Setup Guide

This document shows how to setup Recaller for Bash shell.

## ⚠️ Required: Enable History Timestamps

**IMPORTANT**: Recaller requires timestamped history entries to work properly. Without this configuration, Recaller will appear empty even if your `.bash_history` file exists.

Add the following to your `~/.bashrc` or `~/.bash_profile`:

```bash
# Enable timestamped history (required for Recaller)
export HISTTIMEFORMAT="%s "

# Optional: Increase history size for better search results
export HISTSIZE=10000
export HISTFILESIZE=20000

# Write history immediately after each command (recommended)
export PROMPT_COMMAND="history -a; $PROMPT_COMMAND"
```

### Apply the Configuration

After adding the above to your shell configuration file:

```bash
# Reload your shell configuration
source ~/.bashrc  # or ~/.bash_profile

# Write current session history to file
history -w

# Verify timestamps are enabled (should show numbers starting with #)
tail ~/.bash_history
```

## Verification

After configuration, your `~/.bash_history` should contain entries like:

```
#1692284400
ls -la
#1692284405
git status
#1692284410
recaller
```

If you see plain commands without `#` timestamps, the configuration is not applied correctly.

## Setup Keyboard Shortcut (Ctrl + h)

Add this to your `~/.bashrc`:

```bash
# Launch recaller with Ctrl+h
bind '"\C-h": "recaller\n"'
```

## Troubleshooting

### Problem: Recaller shows no commands
**Solution**: Ensure `HISTTIMEFORMAT` is set and run `history -w` to write current session history.

### Problem: History file exists but Recaller is empty
**Solution**: Your history lacks timestamps. Follow the "Enable History Timestamps" section above.

### Problem: Only recent commands appear
**Solution**: Increase `HISTSIZE` and `HISTFILESIZE` values in your configuration.

## For Ubuntu/Container Users

In minimal Ubuntu containers or fresh installations:

```bash
# Set environment variables
export HISTTIMEFORMAT="%s "
export HISTSIZE=10000

# Force write current history
history -w

# Add to ~/.bashrc for persistence
echo 'export HISTTIMEFORMAT="%s "' >> ~/.bashrc
echo 'export HISTSIZE=10000' >> ~/.bashrc
```

Now Recaller should work correctly with your bash history!